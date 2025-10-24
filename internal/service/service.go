package service

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

// Service provides service management functionality.
type Service struct {
	db *gorm.DB
}

// NewService creates a new service management instance.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// Request/Response structures for service management.
type CreateServiceRequest struct {
	ProductID       uint   `json:"product_id" binding:"required"`
	Name            string `json:"name" binding:"required,min=1,max=255"`
	Code            string `json:"code" binding:"required,min=1,max=100"`
	Description     string `json:"description"`
	Type            string `json:"type" binding:"required"`
	Status          string `json:"status"`
	Availability    string `json:"availability"`
	SupportChannels string `json:"support_channels"`
	EscalationRules string `json:"escalation_rules"`
	Configuration   string `json:"configuration"`
	Tags            string `json:"tags"`
}

type UpdateServiceRequest struct {
	ProductID       *uint  `json:"product_id"`
	Name            string `json:"name" binding:"omitempty,min=1,max=255"`
	Code            string `json:"code" binding:"omitempty,min=1,max=100"`
	Description     string `json:"description"`
	Type            string `json:"type"`
	Status          string `json:"status"`
	Availability    string `json:"availability"`
	SupportChannels string `json:"support_channels"`
	EscalationRules string `json:"escalation_rules"`
	Configuration   string `json:"configuration"`
	Tags            string `json:"tags"`
}

type ServiceResponse struct {
	ID              uint                   `json:"id"`
	ProductID       uint                   `json:"product_id"`
	Name            string                 `json:"name"`
	Code            string                 `json:"code"`
	Description     string                 `json:"description"`
	Type            string                 `json:"type"`
	Status          string                 `json:"status"`
	Availability    string                 `json:"availability"`
	SupportChannels []string               `json:"support_channels"`
	EscalationRules map[string]interface{} `json:"escalation_rules"`
	Configuration   map[string]interface{} `json:"configuration"`
	Tags            []string               `json:"tags"`
	TenantID        uint                   `json:"tenant_id"`
	IsDeleted       bool                   `json:"is_deleted"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	Product         *ProductResponse       `json:"product,omitempty"`
}

type ProductResponse struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	Code         string `json:"code"`
	Description  string `json:"description"`
	Category     string `json:"category"`
	Version      string `json:"version"`
	Status       string `json:"status"`
	IsManaged    bool   `json:"is_managed"`
	SupportLevel string `json:"support_level"`
	TenantID     uint   `json:"tenant_id"`
}

type ListServicesRequest struct {
	Page      int    `form:"page,default=1" binding:"min=1"`
	PageSize  int    `form:"page_size,default=20" binding:"min=1,max=100"`
	Search    string `form:"search"`
	ProductID uint   `form:"product_id"`
	Type      string `form:"type"`
	Status    string `form:"status"`
	SortBy    string `form:"sort_by,default=created_at"`
	SortOrder string `form:"sort_order,default=desc"`
}

// CreateService creates a new service.
func (s *Service) CreateService(tenantID uint, req *CreateServiceRequest) (*ServiceResponse, error) {
	// Normalize input
	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.ToUpper(strings.TrimSpace(req.Code))

	// Validate required fields
	if req.Name == "" {
		return nil, errors.NewInvalidInputError("name", "Service name cannot be empty")
	}
	if req.Code == "" {
		return nil, errors.NewInvalidInputError("code", "Service code cannot be empty")
	}

	// Validate product exists and belongs to tenant
	var product models.Product
	if err := s.db.Where("id = ? AND tenant_id = ?", req.ProductID, tenantID).First(&product).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewNotFoundError("Product")
		}
		return nil, fmt.Errorf("failed to validate product: %w", err)
	}

	// Check if service code already exists for this tenant
	var existingService models.Service
	err := s.db.Where("tenant_id = ? AND code = ?", tenantID, req.Code).First(&existingService).Error
	if err == nil {
		return nil, errors.NewConflictError("Service code already exists")
	}
	if !stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check service code uniqueness: %w", err)
	}

	// Validate status
	if req.Status != "" && req.Status != "active" && req.Status != "inactive" && req.Status != "maintenance" {
		return nil, errors.NewInvalidInputError("status", "Invalid service status")
	}

	// Validate type
	validTypes := map[string]bool{
		"infrastructure": true,
		"application":    true,
		"support":        true,
		"consulting":     true,
	}
	if !validTypes[req.Type] {
		return nil, errors.NewInvalidInputError("type", "Invalid service type")
	}

	// Validate availability
	if req.Availability != "" && req.Availability != "24x7" && req.Availability != "business_hours" && req.Availability != "custom" {
		return nil, errors.NewInvalidInputError("availability", "Invalid availability setting")
	}

	// Parse configuration JSON
	var configMap map[string]interface{}
	if req.Configuration != "" {
		if err := json.Unmarshal([]byte(req.Configuration), &configMap); err != nil {
			return nil, errors.NewInvalidInputError("configuration", "Configuration must be valid JSON")
		}
	}

	// Create service
	service := &models.Service{
		BaseModel: models.BaseModel{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			CreatedBy: nil, // TODO: Set to current user
			UpdatedBy: nil, // TODO: Set to current user
		},
		ProductID:       req.ProductID,
		Name:            req.Name,
		Code:            req.Code,
		Description:     req.Description,
		Type:            req.Type,
		Status:          "active", // Default status
		Availability:    req.Availability,
		SupportChannels: req.SupportChannels,
		EscalationRules: req.EscalationRules,
		Configuration:   req.Configuration,
		Tags:            req.Tags,
	}

	if req.Status != "" {
		service.Status = req.Status
	}

	if req.Availability == "" {
		service.Availability = "24x7"
	}

	if err := s.db.Create(service).Error; err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	return s.serviceToResponse(service, true), nil
}

// GetService gets a service by ID.
func (s *Service) GetService(tenantID uint, serviceID uint) (*ServiceResponse, error) {
	var service models.Service
	if err := s.db.Where("id = ? AND tenant_id = ?", serviceID, tenantID).
		Preload("Product").
		First(&service).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewNotFoundError("Service")
		}
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	return s.serviceToResponse(&service, true), nil
}

// ListServices lists services with pagination and filtering.
func (s *Service) ListServices(tenantID uint, req *ListServicesRequest) ([]ServiceResponse, int64, error) {
	// Build query
	query := s.db.Where("tenant_id = ?", tenantID)

	// Apply filters
	if req.Search != "" {
		searchTerm := "%" + req.Search + "%"
		query = query.Where("name LIKE ? OR code LIKE ? OR description LIKE ?", searchTerm, searchTerm, searchTerm)
	}

	if req.ProductID != 0 {
		query = query.Where("product_id = ?", req.ProductID)
	}

	if req.Type != "" {
		query = query.Where("type = ?", req.Type)
	}

	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	// Count total records
	var total int64
	if err := query.Model(&models.Service{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count services: %w", err)
	}

	// Validate pagination
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	offset := (req.Page - 1) * req.PageSize

	// Validate sort field
	validSortFields := map[string]bool{
		"name":       true,
		"code":       true,
		"type":       true,
		"status":     true,
		"created_at": true,
		"updated_at": true,
	}
	if !validSortFields[req.SortBy] {
		req.SortBy = "created_at"
	}

	// Validate sort order
	if req.SortOrder != "asc" && req.SortOrder != "desc" {
		req.SortOrder = "desc"
	}

	// Get paginated results
	var services []models.Service
	if err := query.Offset(offset).
		Limit(req.PageSize).
		Order(fmt.Sprintf("%s %s", req.SortBy, req.SortOrder)).
		Find(&services).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list services: %w", err)
	}

	// Convert to response
	var responses []ServiceResponse
	for _, service := range services {
		responses = append(responses, *s.serviceToResponse(&service, false))
	}

	return responses, total, nil
}

// UpdateService updates an existing service.
func (s *Service) UpdateService(tenantID uint, serviceID uint, req *UpdateServiceRequest) (*ServiceResponse, error) {
	var service models.Service
	if err := s.db.Where("id = ? AND tenant_id = ? AND is_deleted = ?", serviceID, tenantID, false).First(&service).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewNotFoundError("Service")
		}
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	// Update fields if provided
	if req.Name != "" {
		service.Name = strings.TrimSpace(req.Name)
	}

	if req.Code != "" {
		req.Code = strings.ToUpper(strings.TrimSpace(req.Code))
		// Check if code conflicts with another service
		var existingService models.Service
		err := s.db.Where("tenant_id = ? AND code = ? AND id != ? AND is_deleted = ?", tenantID, req.Code, serviceID, false).First(&existingService).Error
		if err == nil {
			return nil, errors.NewConflictError("Service code already exists")
		}
		if !stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("failed to check service code uniqueness: %w", err)
		}
		service.Code = req.Code
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
		service.ProductID = *req.ProductID
	}

	if req.Description != "" {
		service.Description = req.Description
	}

	if req.Type != "" {
		validTypes := map[string]bool{
			"infrastructure": true,
			"application":    true,
			"support":        true,
			"consulting":     true,
		}
		if !validTypes[req.Type] {
			return nil, errors.NewInvalidInputError("type", "Invalid service type")
		}
		service.Type = req.Type
	}

	if req.Status != "" {
		if req.Status != "active" && req.Status != "inactive" && req.Status != "maintenance" {
			return nil, errors.NewInvalidInputError("status", "Invalid service status")
		}
		service.Status = req.Status
	}

	if req.Availability != "" {
		if req.Availability != "24x7" && req.Availability != "business_hours" && req.Availability != "custom" {
			return nil, errors.NewInvalidInputError("availability", "Invalid availability setting")
		}
		service.Availability = req.Availability
	}

	if req.SupportChannels != "" {
		service.SupportChannels = req.SupportChannels
	}

	if req.EscalationRules != "" {
		service.EscalationRules = req.EscalationRules
	}

	if req.Configuration != "" {
		// Validate JSON format
		var configMap map[string]interface{}
		if err := json.Unmarshal([]byte(req.Configuration), &configMap); err != nil {
			return nil, errors.NewInvalidInputError("configuration", "Configuration must be valid JSON")
		}
		service.Configuration = req.Configuration
	}

	if req.Tags != "" {
		service.Tags = req.Tags
	}

	service.UpdatedAt = time.Now()
	service.UpdatedBy = nil // TODO: Set to current user

	if err := s.db.Save(&service).Error; err != nil {
		return nil, fmt.Errorf("failed to update service: %w", err)
	}

	// Reload with associations
	if err := s.db.Preload("Product").First(&service, service.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to reload service: %w", err)
	}

	return s.serviceToResponse(&service, true), nil
}

// DeleteService soft deletes a service.
func (s *Service) DeleteService(tenantID uint, serviceID uint) error {
	var service models.Service
	if err := s.db.Where("id = ? AND tenant_id = ? AND is_deleted = ?", serviceID, tenantID, false).First(&service).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.NewNotFoundError("Service")
		}
		return fmt.Errorf("failed to get service: %w", err)
	}

	// Check if service has associated tickets
	var ticketCount int64
	if err := s.db.Model(&models.Ticket{}).Where("service_id = ? AND is_deleted = ?", serviceID, false).Count(&ticketCount).Error; err != nil {
		return fmt.Errorf("failed to check associated tickets: %w", err)
	}

	if ticketCount > 0 {
		return errors.NewBusinessRuleError("Cannot delete service", "Please delete or transfer associated tickets first")
	}

	// Soft delete
	if err := s.db.Delete(&service).Error; err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	return nil
}

// ActivateService activates a service.
func (s *Service) ActivateService(tenantID uint, serviceID uint) error {
	return s.updateServiceStatus(tenantID, serviceID, "active")
}

// DeactivateService deactivates a service.
func (s *Service) DeactivateService(tenantID uint, serviceID uint) error {
	return s.updateServiceStatus(tenantID, serviceID, "inactive")
}

// Helper functions.
func (s *Service) updateServiceStatus(tenantID uint, serviceID uint, status string) error {
	var service models.Service
	if err := s.db.Where("id = ? AND tenant_id = ? AND is_deleted = ?", serviceID, tenantID, false).First(&service).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.NewNotFoundError("Service")
		}
		return fmt.Errorf("failed to get service: %w", err)
	}

	service.Status = status
	service.UpdatedAt = time.Now()
	service.UpdatedBy = nil // TODO: Set to current user

	if err := s.db.Save(&service).Error; err != nil {
		return fmt.Errorf("failed to update service status: %w", err)
	}

	return nil
}

func (s *Service) serviceToResponse(service *models.Service, includeProduct bool) *ServiceResponse {
	response := &ServiceResponse{
		ID:           service.ID,
		ProductID:    service.ProductID,
		Name:         service.Name,
		Code:         service.Code,
		Description:  service.Description,
		Type:         service.Type,
		Status:       service.Status,
		Availability: service.Availability,
		IsDeleted:    service.DeletedAt.Valid,
		CreatedAt:    service.CreatedAt,
		UpdatedAt:    service.UpdatedAt,
		Tags:         []string{},
	}

	// Parse support channels JSON
	if service.SupportChannels != "" {
		var channelsArray []string
		if err := json.Unmarshal([]byte(service.SupportChannels), &channelsArray); err == nil {
			response.SupportChannels = channelsArray
		}
	}

	// Parse escalation rules JSON
	if service.EscalationRules != "" {
		var rulesMap map[string]interface{}
		if err := json.Unmarshal([]byte(service.EscalationRules), &rulesMap); err == nil {
			response.EscalationRules = rulesMap
		}
	}

	// Parse configuration JSON
	if service.Configuration != "" {
		var configMap map[string]interface{}
		if err := json.Unmarshal([]byte(service.Configuration), &configMap); err == nil {
			response.Configuration = configMap
		}
	}

	// Parse tags JSON
	if service.Tags != "" {
		var tagsArray []string
		if err := json.Unmarshal([]byte(service.Tags), &tagsArray); err == nil {
			response.Tags = tagsArray
		}
	}

	// Include product if requested and available
	if includeProduct && service.Product.ID != 0 {
		response.Product = &ProductResponse{
			ID:           service.Product.ID,
			Name:         service.Product.Name,
			Code:         service.Product.Code,
			Description:  service.Product.Description,
			Category:     service.Product.Category,
			Version:      service.Product.Version,
			Status:       service.Product.Status,
			IsManaged:    service.Product.IsManaged,
			SupportLevel: service.Product.SupportLevel,
		}
	}

	return response
}
