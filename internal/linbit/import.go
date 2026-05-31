// Package linbit imports the LINBIT UG9 English documentation from GitHub as
// knowledge articles and provisions a "LINBIT" customer. It is wired into the
// `importlinbit` CLI command.
package linbit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/company/smartticket/internal/customer"
	"github.com/company/smartticket/internal/knowledge"
	"github.com/company/smartticket/internal/logger"
	"github.com/company/smartticket/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// contentsURL is the GitHub contents API listing the UG9 English docs.
const contentsURL = "https://api.github.com/repos/LINBIT/linbit-documentation/contents/UG9/en"

// userAgent is sent with every request; GitHub rejects requests without a UA.
const userAgent = "smartticket-importlinbit/1.0"

// ghContent is a single entry in a GitHub contents API listing.
type ghContent struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Size        int    `json:"size"`
	DownloadURL string `json:"download_url"`
}

// Result summarizes an import run.
type Result struct {
	FilesFound      int
	ArticlesCreated int
	ArticlesSkipped int
	CustomerCreated bool
	EmbeddingActive bool
}

// Importer holds the dependencies needed to run an import.
type Importer struct {
	DB        *gorm.DB
	Knowledge *knowledge.Service
	Customer  *customer.Service
	// AIReady reports whether an embedding provider is configured so articles
	// will be auto-indexed on creation.
	AIReady bool

	// NamePrefix, when set, limits the import to files whose name starts with it
	// (e.g. "drbd-" to import only the DRBD docs).
	NamePrefix string
	// MaxBytes, when > 0, skips files larger than this many bytes (e.g. to skip
	// the very large LINSTOR manuals).
	MaxBytes int

	httpClient *http.Client
}

// NewImporter builds an Importer with a sensible default HTTP client.
func NewImporter(db *gorm.DB, kb *knowledge.Service, cust *customer.Service, aiReady bool) *Importer {
	return &Importer{
		DB:         db,
		Knowledge:  kb,
		Customer:   cust,
		AIReady:    aiReady,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// Run fetches the LINBIT UG9 docs, imports them as knowledge articles, and
// provisions the LINBIT customer. Per-file errors are logged and skipped.
func (im *Importer) Run(ctx context.Context, authorID uint) (*Result, error) {
	res := &Result{EmbeddingActive: im.AIReady}

	files, err := im.listFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list LINBIT docs: %w", err)
	}

	for _, f := range files {
		if f.Type != "file" || !strings.HasSuffix(f.Name, ".adoc") {
			continue
		}
		if im.NamePrefix != "" && !strings.HasPrefix(f.Name, im.NamePrefix) {
			continue
		}
		if im.MaxBytes > 0 && f.Size > im.MaxBytes {
			logger.Info("importlinbit: skipping large file", zap.String("name", f.Name), zap.Int("size", f.Size))
			continue
		}
		res.FilesFound++

		raw, err := im.fetchRaw(ctx, f.DownloadURL)
		if err != nil {
			logger.Warn("importlinbit: failed to fetch file", zap.String("name", f.Name), zap.Error(err))
			continue
		}

		content := cleanAsciiDoc(raw)
		if len(content) < 10 {
			logger.Warn("importlinbit: cleaned content too short, skipping", zap.String("name", f.Name))
			continue
		}

		title := deriveTitle(raw, f.Name)
		if len(title) < 3 {
			logger.Warn("importlinbit: title too short, skipping", zap.String("name", f.Name))
			continue
		}

		// Idempotency: skip if an article with this title already exists.
		var existing models.KnowledgeArticle
		err = im.DB.Where("title = ?", title).First(&existing).Error
		if err == nil {
			res.ArticlesSkipped++
			continue
		}
		if err != gorm.ErrRecordNotFound {
			logger.Warn("importlinbit: lookup failed, skipping", zap.String("title", title), zap.Error(err))
			continue
		}

		req := &knowledge.CreateKnowledgeArticleRequest{
			Title:    title,
			Content:  content,
			Category: "technical",
			Status:   "published",
		}
		// CreateKnowledgeArticleCtx auto-indexes into CortexDB when AI is ready.
		if _, err := im.Knowledge.CreateKnowledgeArticleCtx(ctx, authorID, req); err != nil {
			logger.Warn("importlinbit: failed to create article", zap.String("title", title), zap.Error(err))
			continue
		}
		res.ArticlesCreated++
	}

	created, err := im.ensureCustomer()
	if err != nil {
		return res, fmt.Errorf("failed to ensure LINBIT customer: %w", err)
	}
	res.CustomerCreated = created

	return res, nil
}

// listFiles fetches and parses the GitHub contents listing.
func (im *Importer) listFiles(ctx context.Context) ([]ghContent, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, contentsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := im.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("github contents API returned %d: %s", resp.StatusCode, string(body))
	}

	var items []ghContent
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("failed to parse contents listing: %w", err)
	}
	return items, nil
}

// fetchRaw downloads the raw text of a file from its download URL.
func (im *Importer) fetchRaw(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := im.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// ensureCustomer idempotently creates the LINBIT customer. Returns true if it
// was newly created.
func (im *Importer) ensureCustomer() (bool, error) {
	var existing models.Customer
	err := im.DB.Where("code = ?", "LINBIT").First(&existing).Error
	if err == nil {
		return false, nil
	}
	if err != gorm.ErrRecordNotFound {
		return false, err
	}

	active := true
	if _, cerr := im.Customer.CreateCustomer(&customer.CreateCustomerRequest{
		Name:     "LINBIT",
		Code:     "LINBIT",
		IsActive: &active,
	}); cerr != nil {
		return false, cerr
	}
	return true, nil
}

// --- AsciiDoc cleaning ---

var (
	reAttribute  = regexp.MustCompile(`^:.*:`)
	reAttrUnset  = regexp.MustCompile(`^:!.*`)
	reHeading    = regexp.MustCompile(`^=+\s+`)
	reHashHead   = regexp.MustCompile(`^#+\s+`)
	reSourceLine = regexp.MustCompile(`^\[source.*\]`)
	reBlankRun   = regexp.MustCompile(`\n{3,}`)
)

// blockDelimiters are lines that, when they constitute the entire (trimmed)
// line, are AsciiDoc block delimiters to drop.
var blockDelimiters = map[string]bool{
	"----": true,
	"====": true,
	"....": true,
	"****": true,
	"|===": true,
	"--":   true,
	"+++":  true,
}

// cleanAsciiDoc converts AsciiDoc source into clean-enough plain text for RAG.
// It is intentionally line-oriented and lossy; the goal is readable prose.
func cleanAsciiDoc(src string) string {
	lines := strings.Split(src, "\n")
	out := make([]string, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Drop attribute / conditional / include / image lines.
		if reAttribute.MatchString(trimmed) ||
			reAttrUnset.MatchString(trimmed) ||
			strings.HasPrefix(trimmed, "ifdef::") ||
			strings.HasPrefix(trimmed, "ifndef::") ||
			strings.HasPrefix(trimmed, "endif::") ||
			strings.HasPrefix(trimmed, "include::") ||
			strings.HasPrefix(trimmed, "image::") {
			continue
		}

		// Drop block delimiter lines and [source,...] markers.
		if blockDelimiters[trimmed] || reSourceLine.MatchString(trimmed) {
			continue
		}

		// Strip heading markers but keep the heading text.
		if reHeading.MatchString(line) {
			line = reHeading.ReplaceAllString(line, "")
		} else if reHashHead.MatchString(line) {
			line = reHashHead.ReplaceAllString(line, "")
		}

		// Strip stray inline "+++" passthrough tokens.
		line = strings.ReplaceAll(line, "+++", "")

		out = append(out, line)
	}

	cleaned := strings.Join(out, "\n")
	cleaned = reBlankRun.ReplaceAllString(cleaned, "\n\n")
	return strings.TrimSpace(cleaned)
}

// deriveTitle extracts a title from the first AsciiDoc heading, falling back to
// the filename. The result is clamped to 255 characters.
func deriveTitle(src, filename string) string {
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "= ") || strings.HasPrefix(trimmed, "== ") {
			title := strings.TrimSpace(reHeading.ReplaceAllString(trimmed, ""))
			if title != "" {
				return clampTitle(title)
			}
		}
	}

	base := strings.TrimSuffix(filename, ".adoc")
	base = strings.ReplaceAll(base, "-", " ")
	base = strings.ReplaceAll(base, "_", " ")
	base = strings.TrimSpace(base)
	return clampTitle(titleCase(base))
}

// titleCase upper-cases the first letter of each space-separated word.
func titleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if w == "" {
			continue
		}
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

// clampTitle ensures the title does not exceed the 255-char model limit.
func clampTitle(s string) string {
	if len(s) > 255 {
		return s[:255]
	}
	return s
}
