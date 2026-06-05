package macro

import (
	"encoding/json"
	stderrors "errors"

	"github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/models"
	"gorm.io/gorm"
)

// Service provides macro CRUD and apply business logic.
type Service struct {
	db *gorm.DB
}

// NewService creates a macro service backed by the provided GORM database.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// CreateRequest carries the fields for creating a new macro.
type CreateRequest struct {
	Title    string `json:"title" binding:"required,min=1,max=200"`
	Category string `json:"category" binding:"omitempty,max=100"`
	Body     string `json:"body" binding:"required,min=1"`
	Actions  string `json:"actions"` // optional JSON
	Shared   *bool  `json:"shared"`  // nil → defaults to true
}

// UpdateRequest carries the editable macro fields.
type UpdateRequest struct {
	Title    *string `json:"title"`
	Category *string `json:"category"`
	Body     *string `json:"body"`
	Actions  *string `json:"actions"`
	Shared   *bool   `json:"shared"`
}

// Action represents a single side-effect from the Actions JSON field.
type Action struct {
	Type   string            `json:"type"`
	Params map[string]string `json:"params,omitempty"`
}

// isVisible reports whether the macro is accessible to userID — shared macros
// are visible to everyone; private macros only to their owner.
func isVisible(m *models.Macro, userID uint) bool {
	return m.Shared || m.OwnerID == userID
}

// Create inserts a new macro. OwnerID is always set from userID regardless of
// what the caller provides.
func (s *Service) Create(userID uint, req CreateRequest) (*models.Macro, error) {
	shared := true
	if req.Shared != nil {
		shared = *req.Shared
	}
	m := &models.Macro{
		Title:    req.Title,
		Category: req.Category,
		Body:     req.Body,
		Actions:  req.Actions,
		Shared:   shared,
		OwnerID:  userID,
	}
	if err := s.db.Create(m).Error; err != nil {
		return nil, errors.NewDatabaseError("create macro", err)
	}
	return m, nil
}

// List returns all macros visible to userID:
//   - all Shared=true macros, regardless of owner
//   - Shared=false macros where OwnerID = userID
func (s *Service) List(userID uint) ([]models.Macro, error) {
	var macros []models.Macro
	err := s.db.Where("shared = ? OR (shared = ? AND owner_id = ?)", true, false, userID).
		Order("title").Find(&macros).Error
	if err != nil {
		return nil, errors.NewDatabaseError("list macros", err)
	}
	return macros, nil
}

// Get returns a single macro if it is visible to userID.
func (s *Service) Get(userID, macroID uint) (*models.Macro, error) {
	var m models.Macro
	if err := s.db.First(&m, macroID).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewNotFoundError("macro")
		}
		return nil, errors.NewDatabaseError("get macro", err)
	}
	if !isVisible(&m, userID) {
		return nil, errors.NewNotFoundError("macro")
	}
	return &m, nil
}

// Update patches the macro fields. Private macro: only its owner may update.
// Shared macro: any authenticated user may update.
func (s *Service) Update(userID, macroID uint, req UpdateRequest) (*models.Macro, error) {
	m, err := s.Get(userID, macroID)
	if err != nil {
		return nil, err
	}
	// Extra authz: private macros are only editable by their owner.
	if !m.Shared && m.OwnerID != userID {
		return nil, errors.NewForbiddenError("only the owner can update a private macro")
	}

	if req.Title != nil {
		m.Title = *req.Title
	}
	if req.Category != nil {
		m.Category = *req.Category
	}
	if req.Body != nil {
		m.Body = *req.Body
	}
	if req.Actions != nil {
		m.Actions = *req.Actions
	}
	if req.Shared != nil {
		m.Shared = *req.Shared
	}

	if err := s.db.Save(m).Error; err != nil {
		return nil, errors.NewDatabaseError("update macro", err)
	}
	return m, nil
}

// Delete removes a macro. Private macro: only its owner may delete.
// Shared macro: any authenticated user may delete.
func (s *Service) Delete(userID, macroID uint) error {
	m, err := s.Get(userID, macroID)
	if err != nil {
		return err
	}
	if !m.Shared && m.OwnerID != userID {
		return errors.NewForbiddenError("only the owner can delete a private macro")
	}
	if err := s.db.Delete(m).Error; err != nil {
		return errors.NewDatabaseError("delete macro", err)
	}
	return nil
}

// Apply loads the macro (checking visibility), renders its Body with rctx,
// increments UsageCount, and returns the rendered text plus parsed Actions.
func (s *Service) Apply(macroID, userID uint, rctx RenderContext) (string, []Action, error) {
	m, err := s.Get(userID, macroID)
	if err != nil {
		return "", nil, err
	}

	rendered := Render(m.Body, rctx)

	// Parse actions JSON if present.
	var actions []Action
	if m.Actions != "" {
		if jerr := json.Unmarshal([]byte(m.Actions), &actions); jerr != nil {
			// Non-fatal: return empty actions and continue.
			actions = nil
		}
	}

	// Increment usage count.
	if err := s.db.Model(m).UpdateColumn("usage_count", gorm.Expr("usage_count + 1")).Error; err != nil {
		// Non-fatal: log but don't fail the apply.
		_ = err
	}

	return rendered, actions, nil
}
