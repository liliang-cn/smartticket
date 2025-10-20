package tenant

import (
	"fmt"
	"strings"
	"time"

	stderrors "errors"
	"github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/models"
	"gorm.io/gorm"
)

// Service provides tenant management business logic
type Service struct {
	db *gorm.DB
}

// NewService creates a new tenant service
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// CreateTenantRequest represents a tenant creation request
type CreateTenantRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=255"`
	Slug     string `json:"slug" binding:"required,min=2,max=100"`
	Domain   string `json:"domain" binding:"omitempty,max=255"`
	Plan     string `json:"plan" binding:"omitempty,oneof=basic premium enterprise"`
	MaxUsers int    `json:"max_users" binding:"omitempty,min=1,max=10000"`
	Settings string `json:"settings" binding:"omitempty"` // JSON string
}

// UpdateTenantRequest represents a tenant update request
type UpdateTenantRequest struct {
	Name     string `json:"name" binding:"omitempty,min=2,max=255"`
	Slug     string `json:"slug" binding:"omitempty,min=2,max=100"`
	Domain   string `json:"domain" binding:"omitempty,max=255"`
	Plan     string `json:"plan" binding:"omitempty,oneof=basic premium enterprise"`
	MaxUsers int    `json:"max_users" binding:"omitempty,min=1,max=10000"`
	Settings string `json:"settings" binding:"omitempty"` // JSON string
	IsActive *bool  `json:"is_active"`
}

// TenantResponse represents a tenant response
type TenantResponse struct {
	ID        uint       `json:"id"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	Domain    string     `json:"domain"`
	Plan      string     `json:"plan"`
	MaxUsers  int        `json:"max_users"`
	IsActive  bool       `json:"is_active"`
	ExpiredAt *time.Time `json:"expired_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	UserCount int        `json:"user_count"`
}

// TenantListResponse represents a paginated tenant list
type TenantListResponse struct {
	Data       []TenantResponse `json:"data"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}

// CreateTenant creates a new tenant
func (s *Service) CreateTenant(req *CreateTenantRequest) (*TenantResponse, error) {
	// Normalize input
	req.Name = strings.TrimSpace(req.Name)
	req.Slug = strings.ToLower(strings.TrimSpace(req.Slug))
	req.Domain = strings.ToLower(strings.TrimSpace(req.Domain))

	// Validate slug format
	if !s.isValidSlug(req.Slug) {
		return nil, errors.NewInvalidInputError("slug", req.Slug)
	}

	// Set defaults
	if req.Plan == "" {
		req.Plan = "basic"
	}
	if req.MaxUsers == 0 {
		req.MaxUsers = 100
	}

	// Check if slug already exists
	var existingTenant models.Tenant
	if err := s.db.Where("slug = ?", req.Slug).First(&existingTenant).Error; err == nil {
		return nil, errors.NewConflictError("Tenant with this slug already exists")
	}

	// Create tenant
	tenant := &models.Tenant{
		Name:     req.Name,
		Slug:     req.Slug,
		Domain:   req.Domain,
		Plan:     req.Plan,
		MaxUsers: req.MaxUsers,
		Settings: req.Settings,
		IsActive: true,
	}

	if err := s.db.Create(tenant).Error; err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	// Count users (should be 0 for new tenant)
	var userCount int64
	s.db.Model(&models.User{}).Where("tenant_id = ?", tenant.ID).Count(&userCount)

	return s.tenantToResponse(tenant, int(userCount)), nil
}

// GetTenant gets a tenant by ID
func (s *Service) GetTenant(tenantID uint) (*TenantResponse, error) {
	var tenant models.Tenant
	if err := s.db.First(&tenant, tenantID).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewNotFoundError("tenant")
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Count users
	var userCount int64
	s.db.Model(&models.User{}).Where("tenant_id = ?", tenant.ID).Count(&userCount)

	return s.tenantToResponse(&tenant, int(userCount)), nil
}

// GetTenantBySlug gets a tenant by slug
func (s *Service) GetTenantBySlug(slug string) (*TenantResponse, error) {
	var tenant models.Tenant
	if err := s.db.Where("slug = ?", slug).First(&tenant).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewNotFoundError("Tenant not found")
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Count users
	var userCount int64
	s.db.Model(&models.User{}).Where("tenant_id = ?", tenant.ID).Count(&userCount)

	return s.tenantToResponse(&tenant, int(userCount)), nil
}

// ListTenants lists all tenants with pagination
func (s *Service) ListTenants(page, pageSize int, search string) (*TenantListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	var tenants []models.Tenant
	var total int64

	query := s.db.Model(&models.Tenant{})

	// Apply search filter
	if search != "" {
		search = strings.TrimSpace(search)
		query = query.Where("name ILIKE ? OR slug ILIKE ? OR domain ILIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// Count total records
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count tenants: %w", err)
	}

	// Get paginated results
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&tenants).Error; err != nil {
		return nil, fmt.Errorf("failed to list tenants: %w", err)
	}

	// Convert to response and count users for each tenant
	responses := make([]TenantResponse, len(tenants))
	for i, tenant := range tenants {
		var userCount int64
		s.db.Model(&models.User{}).Where("tenant_id = ?", tenant.ID).Count(&userCount)
		responses[i] = *s.tenantToResponse(&tenant, int(userCount))
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &TenantListResponse{
		Data:       responses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// UpdateTenant updates a tenant
func (s *Service) UpdateTenant(tenantID uint, req *UpdateTenantRequest) (*TenantResponse, error) {
	var tenant models.Tenant
	if err := s.db.First(&tenant, tenantID).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewNotFoundError("Tenant not found")
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Normalize input
	if req.Slug != "" {
		req.Slug = strings.ToLower(strings.TrimSpace(req.Slug))
	}
	if req.Domain != "" {
		req.Domain = strings.ToLower(strings.TrimSpace(req.Domain))
	}

	// Validate slug format if provided
	if req.Slug != "" && !s.isValidSlug(req.Slug) {
		return nil, errors.NewBusinessRuleError("slug_validation", "Invalid slug format. Use only letters, numbers, hyphens, and underscores")
	}

	// Check if slug is being changed and if it already exists
	if req.Slug != "" && req.Slug != tenant.Slug {
		var existingTenant models.Tenant
		if err := s.db.Where("slug = ? AND id != ?", req.Slug, tenantID).First(&existingTenant).Error; err == nil {
			return nil, errors.NewConflictError("Tenant with this slug already exists")
		}
		tenant.Slug = req.Slug
	}

	// Update fields
	if req.Name != "" {
		tenant.Name = req.Name
	}
	if req.Domain != "" {
		tenant.Domain = req.Domain
	}
	if req.Plan != "" {
		tenant.Plan = req.Plan
	}
	if req.MaxUsers > 0 {
		tenant.MaxUsers = req.MaxUsers
	}
	if req.Settings != "" {
		tenant.Settings = req.Settings
	}
	if req.IsActive != nil {
		tenant.IsActive = *req.IsActive
	}

	if err := s.db.Save(&tenant).Error; err != nil {
		return nil, fmt.Errorf("failed to update tenant: %w", err)
	}

	// Count users
	var userCount int64
	s.db.Model(&models.User{}).Where("tenant_id = ?", tenant.ID).Count(&userCount)

	return s.tenantToResponse(&tenant, int(userCount)), nil
}

// DeleteTenant soft deletes a tenant
func (s *Service) DeleteTenant(tenantID uint) error {
	var tenant models.Tenant
	if err := s.db.First(&tenant, tenantID).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.NewNotFoundError("Tenant not found")
		}
		return fmt.Errorf("failed to get tenant: %w", err)
	}

	// Check if tenant has users
	var userCount int64
	s.db.Model(&models.User{}).Where("tenant_id = ? AND is_active = ?", tenantID, true).Count(&userCount)
	if userCount > 0 {
		return errors.NewBusinessRuleError("tenant_deletion", "Cannot delete tenant with active users")
	}

	if err := s.db.Delete(&tenant).Error; err != nil {
		return fmt.Errorf("failed to delete tenant: %w", err)
	}

	return nil
}

// ActivateTenant activates a tenant
func (s *Service) ActivateTenant(tenantID uint) error {
	return s.updateTenantStatus(tenantID, true)
}

// DeactivateTenant deactivates a tenant
func (s *Service) DeactivateTenant(tenantID uint) error {
	var userCount int64
	s.db.Model(&models.User{}).Where("tenant_id = ? AND is_active = ?", tenantID, true).Count(&userCount)
	if userCount > 0 {
		return errors.NewBusinessRuleError("tenant_deactivation", "Cannot deactivate tenant with active users")
	}

	return s.updateTenantStatus(tenantID, false)
}

// GetTenantStats gets tenant statistics
func (s *Service) GetTenantStats(tenantID uint) (map[string]interface{}, error) {
	var tenant models.Tenant
	if err := s.db.First(&tenant, tenantID).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewNotFoundError("Tenant not found")
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Get user statistics
	var totalUsers, activeUsers int64
	s.db.Model(&models.User{}).Where("tenant_id = ?", tenantID).Count(&totalUsers)
	s.db.Model(&models.User{}).Where("tenant_id = ? AND is_active = ?", tenantID, true).Count(&activeUsers)

	// Get ticket statistics (will be 0 for now since tickets aren't implemented yet)
	var totalTickets, openTickets int64
	s.db.Model(&models.Ticket{}).Where("tenant_id = ?", tenantID).Count(&totalTickets)
	s.db.Model(&models.Ticket{}).Where("tenant_id = ? AND status IN ?", tenantID, []string{"open", "in_progress"}).Count(&openTickets)

	stats := map[string]interface{}{
		"total_users":     totalUsers,
		"active_users":    activeUsers,
		"inactive_users":  totalUsers - activeUsers,
		"user_quota_used": int(float64(activeUsers) / float64(tenant.MaxUsers) * 100),
		"max_users":       tenant.MaxUsers,
		"total_tickets":   totalTickets,
		"open_tickets":    openTickets,
		"plan":            tenant.Plan,
		"is_active":       tenant.IsActive,
		"created_at":      tenant.CreatedAt,
	}

	return stats, nil
}

// Helper methods

func (s *Service) updateTenantStatus(tenantID uint, isActive bool) error {
	var tenant models.Tenant
	if err := s.db.First(&tenant, tenantID).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.NewNotFoundError("Tenant not found")
		}
		return fmt.Errorf("failed to get tenant: %w", err)
	}

	tenant.IsActive = isActive
	if err := s.db.Save(&tenant).Error; err != nil {
		return fmt.Errorf("failed to update tenant status: %w", err)
	}

	return nil
}

func (s *Service) isValidSlug(slug string) bool {
	if len(slug) < 2 || len(slug) > 100 {
		return false
	}

	for _, char := range slug {
		if !((char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_') {
			return false
		}
	}
	return true
}

func (s *Service) tenantToResponse(tenant *models.Tenant, userCount int) *TenantResponse {
	return &TenantResponse{
		ID:        tenant.ID,
		Name:      tenant.Name,
		Slug:      tenant.Slug,
		Domain:    tenant.Domain,
		Plan:      tenant.Plan,
		MaxUsers:  tenant.MaxUsers,
		IsActive:  tenant.IsActive,
		ExpiredAt: tenant.ExpiredAt,
		CreatedAt: tenant.CreatedAt,
		UpdatedAt: tenant.UpdatedAt,
		UserCount: userCount,
	}
}
