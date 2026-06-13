package aiassist

import (
	"gorm.io/gorm"

	"github.com/company/smartticket/internal/models"
)

const settingsID = 1

var defaultSettings = models.AISettings{
	Enabled:                 true,
	SuggestReplies:          true,
	KnowledgeAI:             true,
	AutoClassify:            false,
	AutoReplyEnabled:        false,
	AutoReplyConfidence:     0.75,
	AutoResolveEnabled:      false,
	MaxAutoRepliesPerTicket: 2,
	AutoSummarizeOnResolve:  false,
	TriageEnabled:           true,
	SentinelEnabled:         true,
	SentinelThrottleSec:     60,
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
	Enabled                 *bool    `json:"enabled"`
	SuggestReplies          *bool    `json:"suggest_replies"`
	KnowledgeAI             *bool    `json:"knowledge_ai"`
	AutoClassify            *bool    `json:"auto_classify"`
	ReplyInstructions       *string  `json:"reply_instructions"`
	AutoReplyEnabled        *bool    `json:"auto_reply_enabled"`
	AutoReplyConfidence     *float64 `json:"auto_reply_confidence"`
	AutoResolveEnabled      *bool    `json:"auto_resolve_enabled"`
	MaxAutoRepliesPerTicket *int     `json:"max_auto_replies_per_ticket"`
	AutoSummarizeOnResolve  *bool    `json:"auto_summarize_on_resolve"`
	TriageEnabled           *bool    `json:"triage_enabled"`
	SentinelEnabled         *bool    `json:"sentinel_enabled"`
	SentinelThrottleSec     *int     `json:"sentinel_throttle_sec"`
}

// Update applies the provided fields to the singleton.
// AutoReplyConfidence is clamped to [0, 1]; MaxAutoRepliesPerTicket must be >= 1.
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
	if in.AutoReplyEnabled != nil {
		a.AutoReplyEnabled = *in.AutoReplyEnabled
	}
	if in.AutoReplyConfidence != nil {
		c := *in.AutoReplyConfidence
		if c < 0 {
			c = 0
		} else if c > 1 {
			c = 1
		}
		a.AutoReplyConfidence = c
	}
	if in.AutoResolveEnabled != nil {
		a.AutoResolveEnabled = *in.AutoResolveEnabled
	}
	if in.MaxAutoRepliesPerTicket != nil {
		m := *in.MaxAutoRepliesPerTicket
		if m < 1 {
			m = 1
		}
		a.MaxAutoRepliesPerTicket = m
	}
	if in.AutoSummarizeOnResolve != nil {
		a.AutoSummarizeOnResolve = *in.AutoSummarizeOnResolve
	}
	if in.TriageEnabled != nil {
		a.TriageEnabled = *in.TriageEnabled
	}
	if in.SentinelEnabled != nil {
		a.SentinelEnabled = *in.SentinelEnabled
	}
	if in.SentinelThrottleSec != nil {
		sec := *in.SentinelThrottleSec
		if sec < 0 {
			sec = 0
		}
		a.SentinelThrottleSec = sec
	}
	if err := s.db.Save(a).Error; err != nil {
		return nil, err
	}
	return a, nil
}
