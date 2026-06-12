package webhook

import (
	"encoding/json"

	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/utils"
	"gorm.io/gorm"
)

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

type CreateInput struct {
	Name   string
	URL    string
	Events []string
}

func (s *Service) Create(in CreateInput, createdBy uint) (*models.Webhook, error) {
	events, _ := json.Marshal(in.Events)
	wh := &models.Webhook{
		Name:      in.Name,
		URL:       in.URL,
		Secret:    utils.GenerateAPIKey("whsec", 24),
		Events:    string(events),
		Active:    true,
		CreatorID: createdBy,
	}
	if err := s.db.Create(wh).Error; err != nil {
		return nil, err
	}
	return wh, nil
}

func (s *Service) List() ([]models.Webhook, error) {
	var whs []models.Webhook
	err := s.db.Order("created_at DESC").Find(&whs).Error
	return whs, err
}

func (s *Service) SetActive(id uint, active bool) error {
	return s.db.Model(&models.Webhook{}).Where("id = ?", id).Update("active", active).Error
}

func (s *Service) Delete(id uint) error {
	return s.db.Delete(&models.Webhook{}, id).Error
}

func (s *Service) Deliveries(webhookID uint, limit int) ([]models.WebhookDelivery, error) {
	var ds []models.WebhookDelivery
	err := s.db.Where("webhook_id = ?", webhookID).Order("created_at DESC").Limit(limit).Find(&ds).Error
	return ds, err
}

// Enqueue writes a pending delivery for each active webhook subscribed to eventType.
func (s *Service) Enqueue(eventType, payload string) error {
	var whs []models.Webhook
	if err := s.db.Where("active = ?", true).Find(&whs).Error; err != nil {
		return err
	}
	for _, wh := range whs {
		var events []string
		_ = json.Unmarshal([]byte(wh.Events), &events)
		if !contains(events, eventType) {
			continue
		}
		d := models.WebhookDelivery{WebhookID: wh.ID, EventType: eventType, Payload: payload, Status: "pending"}
		if err := s.db.Create(&d).Error; err != nil {
			return err
		}
	}
	return nil
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
