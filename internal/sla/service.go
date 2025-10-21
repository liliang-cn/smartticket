package sla

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	stderrors "errors"

	"github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/models"
	"gorm.io/gorm"
)

// Service provides SLA management functionality.
type Service struct {
	db *gorm.DB
}

// NewService creates a new SLA service instance.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// Request/Response structures for SLA management.
type CreateSLATemplateRequest struct {
	Name            string `json:"name" binding:"required,min=1,max=255"`
	Description     string `json:"description"`
	IsDefault       bool   `json:"is_default"`
	IsActive        bool   `json:"is_active"`
	PriorityLevels  string `json:"priority_levels"`
	SeverityLevels  string `json:"severity_levels"`
	ResponseTimes   string `json:"response_times"`
	ResolutionTimes string `json:"resolution_times"`
	BusinessHours   string `json:"business_hours"`
	Holidays        string `json:"holidays"`
	Configuration   string `json:"configuration"`
}

type SLATemplateResponse struct {
	ID              uint                   `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	IsDefault       bool                   `json:"is_default"`
	IsActive        bool                   `json:"is_active"`
	PriorityLevels  []string               `json:"priority_levels"`
	SeverityLevels  []string               `json:"severity_levels"`
	ResponseTimes   map[string]interface{} `json:"response_times"`
	ResolutionTimes map[string]interface{} `json:"resolution_times"`
	BusinessHours   map[string]interface{} `json:"business_hours"`
	Holidays        []string               `json:"holidays"`
	Configuration   map[string]interface{} `json:"configuration"`
	TenantID        uint                   `json:"tenant_id"`
	IsDeleted       bool                   `json:"is_deleted"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

type ListSLATemplatesRequest struct {
	Page      int    `form:"page,default=1" binding:"min=1"`
	PageSize  int    `form:"page_size,default=20" binding:"min=1,max=100"`
	Search    string `form:"search"`
	IsActive  *bool  `form:"is_active"`
	SortBy    string `form:"sort_by,default=created_at"`
	SortOrder string `form:"sort_order,default=desc"`
}

// CreateSLATemplate creates a new SLA template.
func (s *Service) CreateSLATemplate(tenantID uint, req *CreateSLATemplateRequest) (*SLATemplateResponse, error) {
	req.Name = strings.TrimSpace(req.Name)

	if req.Name == "" {
		return nil, errors.NewInvalidInputError("name", "SLA template name cannot be empty")
	}

	var existingTemplate models.SLATemplate
	err := s.db.Where("tenant_id = ? AND name = ? AND is_deleted = ?", tenantID, req.Name, false).First(&existingTemplate).Error
	if err == nil {
		return nil, errors.NewConflictError("SLA template name already exists")
	}
	if !stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check SLA template name uniqueness: %w", err)
	}

	template := &models.SLATemplate{
		BaseModel: models.BaseModel{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			CreatedBy: nil,
			UpdatedBy: nil,
		},
		TenantID:        tenantID,
		Name:            req.Name,
		Description:     req.Description,
		IsDefault:       req.IsDefault,
		IsActive:        req.IsActive,
		PriorityLevels:  req.PriorityLevels,
		SeverityLevels:  req.SeverityLevels,
		ResponseTimes:   req.ResponseTimes,
		ResolutionTimes: req.ResolutionTimes,
		BusinessHours:   req.BusinessHours,
		Holidays:        req.Holidays,
		Configuration:   req.Configuration,
	}

	if err := s.db.Create(template).Error; err != nil {
		return nil, fmt.Errorf("failed to create SLA template: %w", err)
	}

	return s.slaTemplateToResponse(template), nil
}

// ListSLATemplates lists SLA templates with pagination and filtering.
func (s *Service) ListSLATemplates(tenantID uint, req *ListSLATemplatesRequest) ([]SLATemplateResponse, int64, error) {
	query := s.db.Where("tenant_id = ? AND is_deleted = ?", tenantID, false)

	if req.Search != "" {
		searchTerm := "%" + req.Search + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", searchTerm, searchTerm)
	}

	if req.IsActive != nil {
		query = query.Where("is_active = ?", *req.IsActive)
	}

	var total int64
	if err := query.Model(&models.SLATemplate{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count SLA templates: %w", err)
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	offset := (req.Page - 1) * req.PageSize

	if req.SortOrder != "asc" && req.SortOrder != "desc" {
		req.SortOrder = "desc"
	}

	var templates []models.SLATemplate
	if err := query.Offset(offset).
		Limit(req.PageSize).
		Order(fmt.Sprintf("%s %s", req.SortBy, req.SortOrder)).
		Find(&templates).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list SLA templates: %w", err)
	}

	var responses []SLATemplateResponse
	for _, template := range templates {
		responses = append(responses, *s.slaTemplateToResponse(&template))
	}

	return responses, total, nil
}

// Helper function.
func (s *Service) slaTemplateToResponse(template *models.SLATemplate) *SLATemplateResponse {
	response := &SLATemplateResponse{
		ID:             template.ID,
		Name:           template.Name,
		Description:    template.Description,
		IsDefault:      template.IsDefault,
		IsActive:       template.IsActive,
		TenantID:       template.TenantID,
		IsDeleted:      template.DeletedAt.Valid,
		CreatedAt:      template.CreatedAt,
		UpdatedAt:      template.UpdatedAt,
		PriorityLevels: []string{},
		SeverityLevels: []string{},
		Holidays:       []string{},
	}

	if template.PriorityLevels != "" {
		var levelsArray []string
		if err := json.Unmarshal([]byte(template.PriorityLevels), &levelsArray); err == nil {
			response.PriorityLevels = levelsArray
		}
	}

	if template.SeverityLevels != "" {
		var levelsArray []string
		if err := json.Unmarshal([]byte(template.SeverityLevels), &levelsArray); err == nil {
			response.SeverityLevels = levelsArray
		}
	}

	if template.ResponseTimes != "" {
		var timesMap map[string]interface{}
		if err := json.Unmarshal([]byte(template.ResponseTimes), &timesMap); err == nil {
			response.ResponseTimes = timesMap
		}
	}

	if template.ResolutionTimes != "" {
		var timesMap map[string]interface{}
		if err := json.Unmarshal([]byte(template.ResolutionTimes), &timesMap); err == nil {
			response.ResolutionTimes = timesMap
		}
	}

	if template.BusinessHours != "" {
		var hoursMap map[string]interface{}
		if err := json.Unmarshal([]byte(template.BusinessHours), &hoursMap); err == nil {
			response.BusinessHours = hoursMap
		}
	}

	if template.Holidays != "" {
		var holidaysArray []string
		if err := json.Unmarshal([]byte(template.Holidays), &holidaysArray); err == nil {
			response.Holidays = holidaysArray
		}
	}

	if template.Configuration != "" {
		var configMap map[string]interface{}
		if err := json.Unmarshal([]byte(template.Configuration), &configMap); err == nil {
			response.Configuration = configMap
		}
	}

	return response
}

// GetSLATemplate gets a single SLA template by ID.
func (s *Service) GetSLATemplate(tenantID uint, templateID uint) (*SLATemplateResponse, error) {
	var template models.SLATemplate
	if err := s.db.Where("id = ? AND tenant_id = ? AND is_deleted = ?", templateID, tenantID, false).First(&template).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewNotFoundError("SLA template")
		}
		return nil, fmt.Errorf("failed to get SLA template: %w", err)
	}

	return s.slaTemplateToResponse(&template), nil
}

// UpdateSLATemplateRequest represents an SLA template update request.
type UpdateSLATemplateRequest struct {
	Name            *string `json:"name" binding:"omitempty,min=1,max=255"`
	Description     *string `json:"description"`
	IsDefault       *bool   `json:"is_default"`
	IsActive        *bool   `json:"is_active"`
	PriorityLevels  *string `json:"priority_levels"`
	SeverityLevels  *string `json:"severity_levels"`
	ResponseTimes   *string `json:"response_times"`
	ResolutionTimes *string `json:"resolution_times"`
	BusinessHours   *string `json:"business_hours"`
	Holidays        *string `json:"holidays"`
	Configuration   *string `json:"configuration"`
}

// UpdateSLATemplate updates an existing SLA template.
func (s *Service) UpdateSLATemplate(tenantID uint, templateID uint, req *UpdateSLATemplateRequest) (*SLATemplateResponse, error) {
	var template models.SLATemplate
	if err := s.db.Where("id = ? AND tenant_id = ? AND is_deleted = ?", templateID, tenantID, false).First(&template).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewNotFoundError("SLA template")
		}
		return nil, fmt.Errorf("failed to get SLA template: %w", err)
	}

	// Update fields if provided
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, errors.NewInvalidInputError("name", "SLA template name cannot be empty")
		}

		// Check if name conflicts with another template
		var existingTemplate models.SLATemplate
		err := s.db.Where("tenant_id = ? AND name = ? AND id != ? AND is_deleted = ?", tenantID, name, templateID, false).First(&existingTemplate).Error
		if err == nil {
			return nil, errors.NewConflictError("SLA template name already exists")
		}
		if !stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("failed to check SLA template name uniqueness: %w", err)
		}

		template.Name = name
	}

	if req.Description != nil {
		template.Description = *req.Description
	}

	if req.IsDefault != nil {
		// If setting as default, unset other default templates
		if *req.IsDefault && !template.IsDefault {
			if err := s.db.Model(&models.SLATemplate{}).Where("tenant_id = ? AND is_default = ? AND is_deleted = ?", tenantID, true, false).Update("is_default", false).Error; err != nil {
				return nil, fmt.Errorf("failed to unset existing default templates: %w", err)
			}
		}
		template.IsDefault = *req.IsDefault
	}

	if req.IsActive != nil {
		template.IsActive = *req.IsActive
	}

	if req.PriorityLevels != nil {
		template.PriorityLevels = *req.PriorityLevels
	}

	if req.SeverityLevels != nil {
		template.SeverityLevels = *req.SeverityLevels
	}

	if req.ResponseTimes != nil {
		template.ResponseTimes = *req.ResponseTimes
	}

	if req.ResolutionTimes != nil {
		template.ResolutionTimes = *req.ResolutionTimes
	}

	if req.BusinessHours != nil {
		template.BusinessHours = *req.BusinessHours
	}

	if req.Holidays != nil {
		template.Holidays = *req.Holidays
	}

	if req.Configuration != nil {
		template.Configuration = *req.Configuration
	}

	template.UpdatedAt = time.Now()

	if err := s.db.Save(&template).Error; err != nil {
		return nil, fmt.Errorf("failed to update SLA template: %w", err)
	}

	return s.slaTemplateToResponse(&template), nil
}

// DeleteSLATemplate soft deletes an SLA template.
func (s *Service) DeleteSLATemplate(tenantID uint, templateID uint) error {
	var template models.SLATemplate
	if err := s.db.Where("id = ? AND tenant_id = ? AND is_deleted = ?", templateID, tenantID, false).First(&template).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.NewNotFoundError("SLA template")
		}
		return fmt.Errorf("failed to get SLA template: %w", err)
	}

	// Check if template is being used by SLA rules
	var ruleCount int64
	if err := s.db.Model(&models.SLARule{}).Where("sla_template_id = ? AND is_deleted = ?", templateID, false).Count(&ruleCount).Error; err != nil {
		return fmt.Errorf("failed to check associated SLA rules: %w", err)
	}

	if ruleCount > 0 {
		return errors.NewBusinessRuleError("Cannot delete SLA template", "Please delete or modify associated SLA rules first")
	}

	// Soft delete
	if err := s.db.Delete(&template).Error; err != nil {
		return fmt.Errorf("failed to delete SLA template: %w", err)
	}

	return nil
}

// SetDefaultSLATemplate sets an SLA template as the default.
func (s *Service) SetDefaultSLATemplate(tenantID uint, templateID uint) error {
	var template models.SLATemplate
	if err := s.db.Where("id = ? AND tenant_id = ? AND is_deleted = ?", templateID, tenantID, false).First(&template).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.NewNotFoundError("SLA template")
		}
		return fmt.Errorf("failed to get SLA template: %w", err)
	}

	// Unset current default template
	if err := s.db.Model(&models.SLATemplate{}).Where("tenant_id = ? AND is_default = ? AND is_deleted = ?", tenantID, true, false).Update("is_default", false).Error; err != nil {
		return fmt.Errorf("failed to unset existing default template: %w", err)
	}

	// Set new default template
	template.IsDefault = true
	template.UpdatedAt = time.Now()

	if err := s.db.Save(&template).Error; err != nil {
		return fmt.Errorf("failed to set default SLA template: %w", err)
	}

	return nil
}

// ActivateSLATemplate activates an SLA template.
func (s *Service) ActivateSLATemplate(tenantID uint, templateID uint) error {
	return s.updateTemplateStatus(tenantID, templateID, true)
}

// DeactivateSLATemplate deactivates an SLA template.
func (s *Service) DeactivateSLATemplate(tenantID uint, templateID uint) error {
	return s.updateTemplateStatus(tenantID, templateID, false)
}

// Helper function to update template status.
func (s *Service) updateTemplateStatus(tenantID uint, templateID uint, isActive bool) error {
	var template models.SLATemplate
	if err := s.db.Where("id = ? AND tenant_id = ? AND is_deleted = ?", templateID, tenantID, false).First(&template).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.NewNotFoundError("SLA template")
		}
		return fmt.Errorf("failed to get SLA template: %w", err)
	}

	template.IsActive = isActive
	template.UpdatedAt = time.Now()

	if err := s.db.Save(&template).Error; err != nil {
		return fmt.Errorf("failed to update SLA template status: %w", err)
	}

	return nil
}

// SLA Rule Management

type CreateSLARuleRequest struct {
	TemplateID     uint   `json:"template_id" binding:"required"`
	Priority       string `json:"priority" binding:"required,oneof=low medium high critical"`
	Severity       string `json:"severity" binding:"required,oneof=trivial minor major critical"`
	ResponseTime   int    `json:"response_time" binding:"required,min=1"`
	ResolutionTime int    `json:"resolution_time" binding:"required,min=1"`
	BusinessOnly   bool   `json:"business_only"`
	ProductID      *uint  `json:"product_id"`
	ServiceID      *uint  `json:"service_id"`
	Conditions     string `json:"conditions"`
}

type UpdateSLARuleRequest struct {
	TemplateID     *uint   `json:"template_id"`
	Priority       *string `json:"priority" binding:"omitempty,oneof=low medium high critical"`
	Severity       *string `json:"severity" binding:"omitempty,oneof=trivial minor major critical"`
	ResponseTime   *int    `json:"response_time" binding:"omitempty,min=1"`
	ResolutionTime *int    `json:"resolution_time" binding:"omitempty,min=1"`
	BusinessOnly   *bool   `json:"business_only"`
	ProductID      *uint   `json:"product_id"`
	ServiceID      *uint   `json:"service_id"`
	Conditions     *string `json:"conditions"`
}

type SLARuleResponse struct {
	ID             uint      `json:"id"`
	TemplateID     uint      `json:"template_id"`
	Priority       string    `json:"priority"`
	Severity       string    `json:"severity"`
	ResponseTime   int       `json:"response_time"`
	ResolutionTime int       `json:"resolution_time"`
	BusinessOnly   bool      `json:"business_only"`
	ProductID      *uint     `json:"product_id"`
	ServiceID      *uint     `json:"service_id"`
	Conditions     string    `json:"conditions"`
	IsActive       bool      `json:"is_active"`
	TenantID       uint      `json:"tenant_id"`
	IsDeleted      bool      `json:"is_deleted"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ListSLARulesRequest struct {
	Page      int    `form:"page,default=1" binding:"min=1"`
	PageSize  int    `form:"page_size,default=20" binding:"min=1,max=100"`
	Search    string `form:"search"`
	IsActive  *bool  `form:"is_active"`
	Priority  string `form:"priority"`
	Severity  string `form:"severity"`
	ProductID *uint  `form:"product_id"`
	ServiceID *uint  `form:"service_id"`
	SortBy    string `form:"sort_by,default=created_at"`
	SortOrder string `form:"sort_order,default=desc"`
}

// CreateSLARule creates a new SLA rule.
func (s *Service) CreateSLARule(tenantID uint, req *CreateSLARuleRequest) (*SLARuleResponse, error) {
	// Validate template exists and belongs to tenant
	var template models.SLATemplate
	if err := s.db.Where("id = ? AND tenant_id = ? AND is_deleted = ?", req.TemplateID, tenantID, false).First(&template).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewNotFoundError("SLA template")
		}
		return nil, fmt.Errorf("failed to validate SLA template: %w", err)
	}

	// Validate product and service if provided
	if req.ProductID != nil {
		var product models.Product
		if err := s.db.Where("id = ? AND tenant_id = ? AND is_deleted = ?", *req.ProductID, tenantID, false).First(&product).Error; err != nil {
			if stderrors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.NewNotFoundError("Product")
			}
			return nil, fmt.Errorf("failed to validate product: %w", err)
		}
	}

	if req.ServiceID != nil {
		var service models.Service
		if err := s.db.Where("id = ? AND tenant_id = ? AND is_deleted = ?", *req.ServiceID, tenantID, false).First(&service).Error; err != nil {
			if stderrors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.NewNotFoundError("Service")
			}
			return nil, fmt.Errorf("failed to validate service: %w", err)
		}
	}

	// Check if rule already exists for this combination
	var existingRule models.SLARule
	err := s.db.Where("tenant_id = ? AND priority = ? AND severity = ? AND product_id = ? AND service_id = ?",
		tenantID, req.Priority, req.Severity, req.ProductID, req.ServiceID).First(&existingRule).Error
	if err == nil {
		return nil, errors.NewConflictError("SLA rule already exists for this combination")
	}
	if !stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check SLA rule uniqueness: %w", err)
	}

	rule := &models.SLARule{
		BaseModel: models.BaseModel{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			CreatedBy: nil,
			UpdatedBy: nil,
		},
		TenantID:       tenantID,
		SLATemplateID:  req.TemplateID,
		Priority:       req.Priority,
		Severity:       req.Severity,
		ResponseTime:   req.ResponseTime,
		ResolutionTime: req.ResolutionTime,
		BusinessOnly:   req.BusinessOnly,
		ProductID:      req.ProductID,
		ServiceID:      req.ServiceID,
		Conditions:     req.Conditions,
		IsActive:       true,
	}

	if err := s.db.Create(rule).Error; err != nil {
		return nil, fmt.Errorf("failed to create SLA rule: %w", err)
	}

	return s.slaRuleToResponse(rule), nil
}

// ListSLARules lists SLA rules with pagination and filtering.
func (s *Service) ListSLARules(tenantID uint, req *ListSLARulesRequest) ([]SLARuleResponse, int64, error) {
	query := s.db.Where("tenant_id = ? AND is_deleted = ?", tenantID, false)

	if req.Search != "" {
		searchTerm := "%" + req.Search + "%"
		query = query.Where("conditions LIKE ?", searchTerm)
	}

	if req.IsActive != nil {
		query = query.Where("is_active = ?", *req.IsActive)
	}

	if req.Priority != "" {
		query = query.Where("priority = ?", req.Priority)
	}

	if req.Severity != "" {
		query = query.Where("severity = ?", req.Severity)
	}

	if req.ProductID != nil {
		query = query.Where("product_id = ?", *req.ProductID)
	}

	if req.ServiceID != nil {
		query = query.Where("service_id = ?", *req.ServiceID)
	}

	var total int64
	if err := query.Model(&models.SLARule{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count SLA rules: %w", err)
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	offset := (req.Page - 1) * req.PageSize

	if req.SortOrder != "asc" && req.SortOrder != "desc" {
		req.SortOrder = "desc"
	}

	var rules []models.SLARule
	if err := query.Offset(offset).
		Limit(req.PageSize).
		Order(fmt.Sprintf("%s %s", req.SortBy, req.SortOrder)).
		Find(&rules).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list SLA rules: %w", err)
	}

	var responses []SLARuleResponse
	for _, rule := range rules {
		responses = append(responses, *s.slaRuleToResponse(&rule))
	}

	return responses, total, nil
}

// GetSLARule gets a single SLA rule by ID.
func (s *Service) GetSLARule(tenantID uint, ruleID uint) (*SLARuleResponse, error) {
	var rule models.SLARule
	if err := s.db.Where("id = ? AND tenant_id = ? AND is_deleted = ?", ruleID, tenantID, false).First(&rule).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewNotFoundError("SLA rule")
		}
		return nil, fmt.Errorf("failed to get SLA rule: %w", err)
	}

	return s.slaRuleToResponse(&rule), nil
}

// UpdateSLARule updates an existing SLA rule.
func (s *Service) UpdateSLARule(tenantID uint, ruleID uint, req *UpdateSLARuleRequest) (*SLARuleResponse, error) {
	var rule models.SLARule
	if err := s.db.Where("id = ? AND tenant_id = ? AND is_deleted = ?", ruleID, tenantID, false).First(&rule).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewNotFoundError("SLA rule")
		}
		return nil, fmt.Errorf("failed to get SLA rule: %w", err)
	}

	// Update fields if provided
	if req.TemplateID != nil {
		// Validate template exists and belongs to tenant
		var template models.SLATemplate
		if err := s.db.Where("id = ? AND tenant_id = ? AND is_deleted = ?", *req.TemplateID, tenantID, false).First(&template).Error; err != nil {
			if stderrors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.NewNotFoundError("SLA template")
			}
			return nil, fmt.Errorf("failed to validate SLA template: %w", err)
		}
		rule.SLATemplateID = *req.TemplateID
	}

	if req.Priority != nil {
		rule.Priority = *req.Priority
	}

	if req.Severity != nil {
		rule.Severity = *req.Severity
	}

	if req.ResponseTime != nil {
		rule.ResponseTime = *req.ResponseTime
	}

	if req.ResolutionTime != nil {
		rule.ResolutionTime = *req.ResolutionTime
	}

	if req.BusinessOnly != nil {
		rule.BusinessOnly = *req.BusinessOnly
	}

	if req.ProductID != nil {
		// Validate product exists and belongs to tenant
		var product models.Product
		if err := s.db.Where("id = ? AND tenant_id = ? AND is_deleted = ?", *req.ProductID, tenantID, false).First(&product).Error; err != nil {
			if stderrors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.NewNotFoundError("Product")
			}
			return nil, fmt.Errorf("failed to validate product: %w", err)
		}
		rule.ProductID = req.ProductID
	}

	if req.ServiceID != nil {
		// Validate service exists and belongs to tenant
		var service models.Service
		if err := s.db.Where("id = ? AND tenant_id = ? AND is_deleted = ?", *req.ServiceID, tenantID, false).First(&service).Error; err != nil {
			if stderrors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.NewNotFoundError("Service")
			}
			return nil, fmt.Errorf("failed to validate service: %w", err)
		}
		rule.ServiceID = req.ServiceID
	}

	if req.Conditions != nil {
		rule.Conditions = *req.Conditions
	}

	rule.UpdatedAt = time.Now()

	if err := s.db.Save(&rule).Error; err != nil {
		return nil, fmt.Errorf("failed to update SLA rule: %w", err)
	}

	return s.slaRuleToResponse(&rule), nil
}

// DeleteSLARule soft deletes an SLA rule.
func (s *Service) DeleteSLARule(tenantID uint, ruleID uint) error {
	var rule models.SLARule
	if err := s.db.Where("id = ? AND tenant_id = ? AND is_deleted = ?", ruleID, tenantID, false).First(&rule).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.NewNotFoundError("SLA rule")
		}
		return fmt.Errorf("failed to get SLA rule: %w", err)
	}

	// Soft delete
	if err := s.db.Delete(&rule).Error; err != nil {
		return fmt.Errorf("failed to delete SLA rule: %w", err)
	}

	return nil
}

// ActivateSLARule activates an SLA rule.
func (s *Service) ActivateSLARule(tenantID uint, ruleID uint) error {
	return s.updateRuleStatus(tenantID, ruleID, true)
}

// DeactivateSLARule deactivates an SLA rule.
func (s *Service) DeactivateSLARule(tenantID uint, ruleID uint) error {
	return s.updateRuleStatus(tenantID, ruleID, false)
}

// Helper function to update rule status.
func (s *Service) updateRuleStatus(tenantID uint, ruleID uint, isActive bool) error {
	var rule models.SLARule
	if err := s.db.Where("id = ? AND tenant_id = ? AND is_deleted = ?", ruleID, tenantID, false).First(&rule).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.NewNotFoundError("SLA rule")
		}
		return fmt.Errorf("failed to get SLA rule: %w", err)
	}

	rule.IsActive = isActive
	rule.UpdatedAt = time.Now()

	if err := s.db.Save(&rule).Error; err != nil {
		return fmt.Errorf("failed to update SLA rule status: %w", err)
	}

	return nil
}

// Helper function to convert SLA rule model to response.
func (s *Service) slaRuleToResponse(rule *models.SLARule) *SLARuleResponse {
	return &SLARuleResponse{
		ID:             rule.ID,
		TemplateID:     rule.SLATemplateID,
		Priority:       rule.Priority,
		Severity:       rule.Severity,
		ResponseTime:   rule.ResponseTime,
		ResolutionTime: rule.ResolutionTime,
		BusinessOnly:   rule.BusinessOnly,
		ProductID:      rule.ProductID,
		ServiceID:      rule.ServiceID,
		Conditions:     rule.Conditions,
		IsActive:       rule.IsActive,
		TenantID:       rule.TenantID,
		IsDeleted:      rule.DeletedAt.Valid,
		CreatedAt:      rule.CreatedAt,
		UpdatedAt:      rule.UpdatedAt,
	}
}
