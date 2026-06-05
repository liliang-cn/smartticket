// Package survey implements CSAT (Customer Satisfaction) surveys that are
// automatically triggered when a ticket is resolved. Each ticket gets at most
// one survey (CreateForTicket is idempotent). The survey is accessed via a
// random token so no authentication is required to submit a rating.
package survey

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/company/smartticket/internal/models"
	"gorm.io/gorm"
)

// Service provides CSAT survey business logic.
type Service struct {
	db *gorm.DB
}

// NewService creates a survey service backed by the given database.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// CreateForTicket creates a new survey for ticketID, or returns the existing
// unanswered one (idempotent). A brand-new survey is assigned a random 64-hex
// token and SentAt is set to the current time.
func (s *Service) CreateForTicket(ticketID uint) (*models.SatisfactionSurvey, error) {
	// Return existing unanswered survey if present.
	var existing models.SatisfactionSurvey
	err := s.db.Where("ticket_id = ? AND responded_at IS NULL", ticketID).First(&existing).Error
	if err == nil {
		return &existing, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("survey: lookup failed: %w", err)
	}

	token, err := randomToken()
	if err != nil {
		return nil, fmt.Errorf("survey: token generation failed: %w", err)
	}

	now := time.Now()
	survey := models.SatisfactionSurvey{
		TicketID: ticketID,
		Token:    token,
		SentAt:   &now,
	}
	if err := s.db.Create(&survey).Error; err != nil {
		return nil, fmt.Errorf("survey: create failed: %w", err)
	}
	return &survey, nil
}

// GetByToken looks up a survey by its public access token.
// Returns gorm.ErrRecordNotFound when the token is unknown.
func (s *Service) GetByToken(token string) (*models.SatisfactionSurvey, error) {
	var survey models.SatisfactionSurvey
	if err := s.db.Where("token = ?", token).First(&survey).Error; err != nil {
		return nil, err
	}
	return &survey, nil
}

// Submit records the customer's rating and optional comment.
// It validates that rating is 1..5, and rejects duplicate submissions.
func (s *Service) Submit(token string, rating int, comment string) error {
	if rating < 1 || rating > 5 {
		return fmt.Errorf("survey: rating must be between 1 and 5")
	}

	survey, err := s.GetByToken(token)
	if err != nil {
		return err
	}
	if survey.RespondedAt != nil {
		return fmt.Errorf("survey: already responded")
	}

	now := time.Now()
	survey.Rating = rating
	survey.Comment = comment
	survey.RespondedAt = &now
	if err := s.db.Save(survey).Error; err != nil {
		return fmt.Errorf("survey: save failed: %w", err)
	}
	return nil
}

// Stats returns aggregate statistics across all surveys in the system.
type Stats struct {
	SentCount      int     `json:"sent_count"`
	ResponseCount  int     `json:"response_count"`
	ResponseRate   float64 `json:"response_rate"` // 0..1
	AverageRating  float64 `json:"average_rating"` // 0 when no responses
}

// GetStats computes aggregate CSAT statistics.
func (s *Service) GetStats() (Stats, error) {
	var sentCount int64
	if err := s.db.Model(&models.SatisfactionSurvey{}).Count(&sentCount).Error; err != nil {
		return Stats{}, fmt.Errorf("survey: count sent: %w", err)
	}

	type agg struct {
		Count int
		Sum   float64
	}
	var a agg
	if err := s.db.Model(&models.SatisfactionSurvey{}).
		Where("responded_at IS NOT NULL").
		Select("COUNT(*) as count, SUM(rating) as sum").
		Scan(&a).Error; err != nil {
		return Stats{}, fmt.Errorf("survey: aggregate: %w", err)
	}

	st := Stats{
		SentCount:     int(sentCount),
		ResponseCount: a.Count,
	}
	if sentCount > 0 {
		st.ResponseRate = float64(a.Count) / float64(sentCount)
	}
	if a.Count > 0 {
		st.AverageRating = a.Sum / float64(a.Count)
	}
	return st, nil
}

// randomToken generates a 32-byte (64 hex char) random token.
func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
