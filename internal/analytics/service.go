package analytics

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strings"
	"time"

	"github.com/company/smartticket/internal/models"
	"gorm.io/gorm"
)

// EventRequest is the public analytics payload sent by the landing page.
type EventRequest struct {
	EventType string `json:"event_type"`
	Path      string `json:"path"`
	Title     string `json:"title"`
	Referrer  string `json:"referrer"`
	Source    string `json:"source"`
	Locale    string `json:"locale"`
	Target    string `json:"target"`
}

// Summary is the admin-facing analytics rollup.
type Summary struct {
	Days           int       `json:"days"`
	TotalEvents    int64     `json:"total_events"`
	Pageviews      int64     `json:"pageviews"`
	Clicks         int64     `json:"clicks"`
	UniqueVisitors int64     `json:"unique_visitors"`
	TopReferrers   []Bucket  `json:"top_referrers"`
	TopSources     []Bucket  `json:"top_sources"`
	TopPaths       []Bucket  `json:"top_paths"`
	TopTargets     []Bucket  `json:"top_targets"`
	RecentEvents   []EventVM `json:"recent_events"`
}

// Bucket is a label/count pair used for top-N analytics.
type Bucket struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// EventVM is a compact analytics event view.
type EventVM struct {
	EventType  string `json:"event_type"`
	Path       string `json:"path"`
	Referrer   string `json:"referrer"`
	Source     string `json:"source"`
	Target     string `json:"target"`
	DeviceType string `json:"device_type"`
	CreatedAt  int64  `json:"created_at"`
}

// Service records and summarizes website analytics.
type Service struct {
	db     *gorm.DB
	secret string
}

func NewService(db *gorm.DB, secret string) *Service {
	return &Service{db: db, secret: secret}
}

func (s *Service) Record(ctx context.Context, req EventRequest, ip, ua string) error {
	eventType := normalizeEventType(req.EventType)
	path := clean(req.Path, 500)
	referrer := clean(req.Referrer, 500)
	source := clean(req.Source, 100)
	if source == "" {
		source = sourceFrom(path, referrer)
	}

	event := models.AnalyticsEvent{
		EventType:   eventType,
		Path:        path,
		Title:       clean(req.Title, 255),
		Referrer:    referrer,
		Source:      source,
		Locale:      clean(req.Locale, 20),
		Target:      clean(req.Target, 255),
		UserAgent:   clean(ua, 500),
		DeviceType:  deviceType(ua),
		VisitorHash: s.visitorHash(ip, ua, time.Now().UTC()),
	}
	return s.db.WithContext(ctx).Create(&event).Error
}

func (s *Service) Summary(ctx context.Context, days int) (*Summary, error) {
	if days <= 0 || days > 365 {
		days = 30
	}
	since := time.Now().AddDate(0, 0, -days)
	base := func() *gorm.DB {
		return s.db.WithContext(ctx).Model(&models.AnalyticsEvent{}).Where("created_at >= ?", since)
	}

	var out Summary
	out.Days = days
	if err := base().Count(&out.TotalEvents).Error; err != nil {
		return nil, err
	}
	if err := base().Where("event_type = ?", "pageview").Count(&out.Pageviews).Error; err != nil {
		return nil, err
	}
	if err := base().Where("event_type = ?", "click").Count(&out.Clicks).Error; err != nil {
		return nil, err
	}
	if err := base().Where("visitor_hash <> ''").Distinct("visitor_hash").Count(&out.UniqueVisitors).Error; err != nil {
		return nil, err
	}

	var err error
	if out.TopReferrers, err = s.top(ctx, since, "referrer", "direct", 8); err != nil {
		return nil, err
	}
	if out.TopSources, err = s.top(ctx, since, "source", "direct", 8); err != nil {
		return nil, err
	}
	if out.TopPaths, err = s.top(ctx, since, "path", "unknown", 8); err != nil {
		return nil, err
	}
	if out.TopTargets, err = s.topClicks(ctx, since, 8); err != nil {
		return nil, err
	}

	var events []models.AnalyticsEvent
	if err := s.db.WithContext(ctx).Where("created_at >= ?", since).
		Order("created_at DESC").Limit(20).Find(&events).Error; err != nil {
		return nil, err
	}
	out.RecentEvents = make([]EventVM, 0, len(events))
	for _, e := range events {
		out.RecentEvents = append(out.RecentEvents, EventVM{
			EventType:  e.EventType,
			Path:       e.Path,
			Referrer:   e.Referrer,
			Source:     e.Source,
			Target:     e.Target,
			DeviceType: e.DeviceType,
			CreatedAt:  e.CreatedAt.Unix(),
		})
	}
	return &out, nil
}

func (s *Service) top(ctx context.Context, since time.Time, field, fallback string, limit int) ([]Bucket, error) {
	var rows []Bucket
	expr := "COALESCE(NULLIF(" + field + ", ''), '" + fallback + "') as name, COUNT(*) as count"
	err := s.db.WithContext(ctx).Model(&models.AnalyticsEvent{}).
		Select(expr).Where("created_at >= ?", since).
		Group("name").Order("count DESC").Limit(limit).Scan(&rows).Error
	return rows, err
}

func (s *Service) topClicks(ctx context.Context, since time.Time, limit int) ([]Bucket, error) {
	var rows []Bucket
	err := s.db.WithContext(ctx).Model(&models.AnalyticsEvent{}).
		Select("COALESCE(NULLIF(target, ''), 'unknown') as name, COUNT(*) as count").
		Where("created_at >= ? AND event_type = ?", since, "click").
		Group("name").Order("count DESC").Limit(limit).Scan(&rows).Error
	return rows, err
}

func (s *Service) visitorHash(ip, ua string, at time.Time) string {
	sum := sha256.Sum256([]byte(at.Format("2006-01-02") + "|" + s.secret + "|" + ip + "|" + ua))
	return hex.EncodeToString(sum[:])
}

func normalizeEventType(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "click":
		return "click"
	default:
		return "pageview"
	}
}

func clean(v string, max int) string {
	v = strings.TrimSpace(v)
	if len(v) > max {
		return v[:max]
	}
	return v
}

func sourceFrom(pathValue, referrer string) string {
	if u, err := url.Parse(pathValue); err == nil {
		if source := u.Query().Get("utm_source"); source != "" {
			return clean(source, 100)
		}
		if source := u.Query().Get("ref"); source != "" {
			return clean(source, 100)
		}
	}
	if referrer == "" {
		return "direct"
	}
	host := referrer
	if u, err := url.Parse(referrer); err == nil && u.Host != "" {
		host = u.Host
	}
	host = strings.TrimPrefix(strings.ToLower(host), "www.")
	switch {
	case strings.Contains(host, "producthunt.com"):
		return "producthunt"
	case strings.Contains(host, "news.ycombinator.com"):
		return "hackernews"
	case strings.Contains(host, "reddit.com"):
		return "reddit"
	case strings.Contains(host, "github.com"):
		return "github"
	case strings.Contains(host, "google."):
		return "google"
	default:
		return clean(host, 100)
	}
}

func deviceType(ua string) string {
	ua = strings.ToLower(ua)
	switch {
	case strings.Contains(ua, "mobile") || strings.Contains(ua, "iphone") || strings.Contains(ua, "android"):
		return "mobile"
	case strings.Contains(ua, "ipad") || strings.Contains(ua, "tablet"):
		return "tablet"
	default:
		return "desktop"
	}
}
