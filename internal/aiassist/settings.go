package aiassist

import (
	"gorm.io/gorm"

	"github.com/company/smartticket/internal/models"
)

const settingsID = 1

var defaultSettings = models.AISettings{
	Enabled:        true,
	SuggestReplies: true,
	KnowledgeAI:    true,
	AutoClassify:   false,
}

// SettingsStore manages the AI feature-toggle singleton.
type SettingsStore struct{ db *gorm.DB }

// NewSettingsStore builds the store.
func NewSettingsStore(db *gorm.DB) *SettingsStore { return &SettingsStore{db: db} }

// Get returns the settings, creating the default singleton on first access.
func (s *SettingsStore) Get() (*models.AISettings, error) {
	var a models.AISettings
	err := s.db.First(&a, settingsID).Error
	if err == gorm.ErrRecordNotFound {
		a = defaultSettings
		a.ID = settingsID
		if cerr := s.db.Create(&a).Error; cerr != nil {
			return nil, cerr
		}
		return &a, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// UpdateSettings carries the editable AI flags; nil pointers leave a value as-is.
type UpdateSettings struct {
	Enabled           *bool   `json:"enabled"`
	SuggestReplies    *bool   `json:"suggest_replies"`
	KnowledgeAI       *bool   `json:"knowledge_ai"`
	AutoClassify      *bool   `json:"auto_classify"`
	ReplyInstructions *string `json:"reply_instructions"`
}

// Update applies the provided fields to the singleton.
func (s *SettingsStore) Update(in UpdateSettings) (*models.AISettings, error) {
	a, err := s.Get()
	if err != nil {
		return nil, err
	}
	if in.Enabled != nil {
		a.Enabled = *in.Enabled
	}
	if in.SuggestReplies != nil {
		a.SuggestReplies = *in.SuggestReplies
	}
	if in.KnowledgeAI != nil {
		a.KnowledgeAI = *in.KnowledgeAI
	}
	if in.AutoClassify != nil {
		a.AutoClassify = *in.AutoClassify
	}
	if in.ReplyInstructions != nil {
		a.ReplyInstructions = *in.ReplyInstructions
	}
	if err := s.db.Save(a).Error; err != nil {
		return nil, err
	}
	return a, nil
}
