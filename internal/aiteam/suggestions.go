package aiteam

import (
	"time"

	"github.com/company/smartticket/internal/models"
	"gorm.io/gorm"
)

// SuggestionStore persists AISuggestion rows (one per ticket+agent).
type SuggestionStore struct{ db *gorm.DB }

func NewSuggestionStore(db *gorm.DB) *SuggestionStore { return &SuggestionStore{db: db} }

// Upsert creates or updates the single suggestion for (ticketID, agentName).
func (s *SuggestionStore) Upsert(ticketID uint, agentName, status string, confidence float64, payload string) (*models.AISuggestion, error) {
	var sug models.AISuggestion
	err := s.db.Where("ticket_id = ? AND agent_name = ?", ticketID, agentName).First(&sug).Error
	if err == gorm.ErrRecordNotFound {
		sug = models.AISuggestion{TicketID: ticketID, AgentName: agentName, Status: status, Confidence: confidence, Payload: payload}
		if cerr := s.db.Create(&sug).Error; cerr != nil {
			return nil, cerr
		}
		return &sug, nil
	}
	if err != nil {
		return nil, err
	}
	sug.Status = status
	sug.Confidence = confidence
	sug.Payload = payload
	sug.AdoptedBy = nil
	sug.ResolvedAt = nil
	if uerr := s.db.Save(&sug).Error; uerr != nil {
		return nil, uerr
	}
	return &sug, nil
}

func (s *SuggestionStore) List(ticketID uint) ([]models.AISuggestion, error) {
	var out []models.AISuggestion
	err := s.db.Where("ticket_id = ?", ticketID).Order("agent_name").Find(&out).Error
	return out, err
}

func (s *SuggestionStore) Get(id uint) (*models.AISuggestion, error) {
	var sug models.AISuggestion
	if err := s.db.First(&sug, id).Error; err != nil {
		return nil, err
	}
	return &sug, nil
}

func (s *SuggestionStore) Adopt(id, userID uint) error {
	now := time.Now().Unix()
	return s.db.Model(&models.AISuggestion{}).Where("id = ?", id).
		Updates(map[string]any{"status": "adopted", "adopted_by": userID, "resolved_at": now}).Error
}

func (s *SuggestionStore) Dismiss(id uint) error {
	now := time.Now().Unix()
	return s.db.Model(&models.AISuggestion{}).Where("id = ?", id).
		Updates(map[string]any{"status": "dismissed", "resolved_at": now}).Error
}
