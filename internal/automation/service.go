package automation

import (
	"encoding/json"
	"fmt"
	"time"

	stderrors "errors"

	"github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/models"
	"gorm.io/gorm"
)

// Service provides CRUD operations for AutomationRule and implements RuleStore
// and TicketRepo.
type Service struct {
	db *gorm.DB
}

// NewService creates a new automation Service.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// RulesForEvent implements RuleStore: returns enabled rules ordered by Position.
func (s *Service) RulesForEvent(event string) ([]Rule, error) {
	var rows []models.AutomationRule
	if err := s.db.Where("event = ? AND enabled = ?", event, true).
		Order("position asc").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("RulesForEvent: %w", err)
	}

	out := make([]Rule, 0, len(rows))
	for _, r := range rows {
		rule, err := rowToRule(r)
		if err != nil {
			// Skip malformed rules rather than halting the engine.
			continue
		}
		out = append(out, rule)
	}
	return out, nil
}

// OverdueTicketIDs implements TicketRepo: returns IDs of open/in_progress tickets
// past their due_date.
func (s *Service) OverdueTicketIDs(now time.Time) ([]uint, error) {
	var ids []uint
	err := s.db.Model(&models.Ticket{}).
		Select("id").
		Where("is_deleted = ? AND status IN (?) AND due_date IS NOT NULL AND due_date < ?",
			false, []string{"open", "in_progress"}, now).
		Pluck("id", &ids).Error
	return ids, err
}

// SilentCustomerTicketIDs implements TicketRepo: returns IDs of in-progress tickets
// where the last public message is older than windowSeconds.
func (s *Service) SilentCustomerTicketIDs(now time.Time, windowSeconds int64) ([]uint, error) {
	cutoff := now.Add(-time.Duration(windowSeconds) * time.Second)
	var ids []uint
	// A ticket is "silent" if it has at least one message and no message newer than cutoff.
	err := s.db.Raw(`
		SELECT t.id FROM tickets t
		WHERE t.is_deleted = 0
		  AND t.status IN ('open','in_progress')
		  AND EXISTS (
			SELECT 1 FROM messages m
			WHERE m.ticket_id = t.id
			  AND m.is_internal = 0
			  AND m.is_from_ai = 0
		  )
		  AND (
			SELECT MAX(m2.created_at) FROM messages m2
			WHERE m2.ticket_id = t.id
			  AND m2.is_internal = 0
			  AND m2.is_from_ai = 0
		  ) < ?
	`, cutoff).Pluck("id", &ids).Error
	return ids, err
}

// --- CRUD ---

// CreateRuleRequest is the JSON body for POST /automations.
type CreateRuleRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=200"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	Event       string `json:"event" binding:"required"`
	Match       string `json:"match" binding:"omitempty,oneof=all any"`
	Conditions  string `json:"conditions"` // raw JSON array string (validated below)
	Actions     string `json:"actions"`    // raw JSON array string (validated below)
	Position    int    `json:"position"`
}

// UpdateRuleRequest is the JSON body for PUT /automations/:id.
type UpdateRuleRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=1,max=200"`
	Description *string `json:"description"`
	Enabled     *bool   `json:"enabled"`
	Event       *string `json:"event"`
	Match       *string `json:"match" binding:"omitempty,oneof=all any"`
	Conditions  *string `json:"conditions"`
	Actions     *string `json:"actions"`
	Position    *int    `json:"position"`
}

// ReorderRequest carries an ordered list of rule IDs for POST /automations/reorder.
type ReorderRequest struct {
	IDs []uint `json:"ids" binding:"required"`
}

// RuleResponse is the API-level representation of an AutomationRule.
type RuleResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled"`
	Event       string    `json:"event"`
	Match       string    `json:"match"`
	Conditions  string    `json:"conditions"`
	Actions     string    `json:"actions"`
	Position    int       `json:"position"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateRule persists a new automation rule.
func (s *Service) CreateRule(req *CreateRuleRequest) (*RuleResponse, error) {
	if req.Match == "" {
		req.Match = "all"
	}
	// Validate Conditions JSON (must be array or empty).
	if req.Conditions != "" {
		if err := validateJSONArray(req.Conditions); err != nil {
			return nil, errors.NewInvalidInputError("conditions", "conditions must be a JSON array")
		}
	}
	// Validate Actions JSON.
	if req.Actions != "" {
		if err := validateJSONArray(req.Actions); err != nil {
			return nil, errors.NewInvalidInputError("actions", "actions must be a JSON array")
		}
	}

	rule := &models.AutomationRule{
		Name:        req.Name,
		Description: req.Description,
		Enabled:     req.Enabled,
		Event:       req.Event,
		Match:       req.Match,
		Conditions:  req.Conditions,
		Actions:     req.Actions,
		Position:    req.Position,
	}
	if err := s.db.Create(rule).Error; err != nil {
		return nil, fmt.Errorf("create rule: %w", err)
	}
	return ruleToResponse(rule), nil
}

// GetRule returns a single rule by ID.
func (s *Service) GetRule(id uint) (*RuleResponse, error) {
	var rule models.AutomationRule
	if err := s.db.First(&rule, id).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewNotFoundError("automation rule")
		}
		return nil, fmt.Errorf("get rule: %w", err)
	}
	return ruleToResponse(&rule), nil
}

// ListRules returns all rules ordered by position.
func (s *Service) ListRules() ([]RuleResponse, error) {
	var rules []models.AutomationRule
	if err := s.db.Order("position asc").Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("list rules: %w", err)
	}
	out := make([]RuleResponse, len(rules))
	for i, r := range rules {
		out[i] = *ruleToResponse(&r)
	}
	return out, nil
}

// UpdateRule applies partial updates to a rule.
func (s *Service) UpdateRule(id uint, req *UpdateRuleRequest) (*RuleResponse, error) {
	var rule models.AutomationRule
	if err := s.db.First(&rule, id).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewNotFoundError("automation rule")
		}
		return nil, fmt.Errorf("get rule for update: %w", err)
	}

	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.Description != nil {
		rule.Description = *req.Description
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if req.Event != nil {
		rule.Event = *req.Event
	}
	if req.Match != nil {
		rule.Match = *req.Match
	}
	if req.Conditions != nil {
		if *req.Conditions != "" {
			if err := validateJSONArray(*req.Conditions); err != nil {
				return nil, errors.NewInvalidInputError("conditions", "conditions must be a JSON array")
			}
		}
		rule.Conditions = *req.Conditions
	}
	if req.Actions != nil {
		if *req.Actions != "" {
			if err := validateJSONArray(*req.Actions); err != nil {
				return nil, errors.NewInvalidInputError("actions", "actions must be a JSON array")
			}
		}
		rule.Actions = *req.Actions
	}
	if req.Position != nil {
		rule.Position = *req.Position
	}
	rule.UpdatedAt = time.Now()

	if err := s.db.Save(&rule).Error; err != nil {
		return nil, fmt.Errorf("update rule: %w", err)
	}
	return ruleToResponse(&rule), nil
}

// DeleteRule hard-deletes an automation rule.
func (s *Service) DeleteRule(id uint) error {
	result := s.db.Delete(&models.AutomationRule{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete rule: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("automation rule")
	}
	return nil
}

// ReorderRules assigns Position values 0, 1, 2… to the rules in the supplied
// order. Rules not mentioned in ids keep their existing positions.
func (s *Service) ReorderRules(ids []uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for i, id := range ids {
			if err := tx.Model(&models.AutomationRule{}).
				Where("id = ?", id).
				Update("position", i).Error; err != nil {
				return fmt.Errorf("reorder: set position for rule %d: %w", id, err)
			}
		}
		return nil
	})
}

// --- helpers ---

func rowToRule(r models.AutomationRule) (Rule, error) {
	var conds []Condition
	if r.Conditions != "" {
		if err := json.Unmarshal([]byte(r.Conditions), &conds); err != nil {
			return Rule{}, fmt.Errorf("parse conditions: %w", err)
		}
	}
	var actions []Action
	if r.Actions != "" {
		if err := json.Unmarshal([]byte(r.Actions), &actions); err != nil {
			return Rule{}, fmt.Errorf("parse actions: %w", err)
		}
	}
	return Rule{
		ID:         r.ID,
		Match:      r.Match,
		Conditions: conds,
		Actions:    actions,
	}, nil
}

func ruleToResponse(r *models.AutomationRule) *RuleResponse {
	return &RuleResponse{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		Enabled:     r.Enabled,
		Event:       r.Event,
		Match:       r.Match,
		Conditions:  r.Conditions,
		Actions:     r.Actions,
		Position:    r.Position,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func validateJSONArray(s string) error {
	var v []any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return err
	}
	return nil
}
