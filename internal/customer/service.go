// Package customer provides CRUD management for customer organizations
// (the operator's client companies). Customer-role users belong to one customer
// and their tickets are scoped to it; this package handles the team-only
// management of those organizations and their contact users.
package customer

import (
	"errors"
	"fmt"
	"strings"
	"time"

	apperrors "github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/models"
	"gorm.io/gorm"
)

// Service provides customer organization management functionality.
type Service struct {
	db *gorm.DB
}

// NewService creates a new customer service instance.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// CreateCustomerRequest represents a customer creation request.
type CreateCustomerRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=255"`
	Code        string `json:"code" binding:"omitempty,max=100"`
	Domain      string `json:"domain" binding:"omitempty,max=255"`
	Description string `json:"description"`
	IsActive    *bool  `json:"is_active"`
}

// UpdateCustomerRequest represents a customer update request.
type UpdateCustomerRequest struct {
	Name        string `json:"name" binding:"omitempty,min=1,max=255"`
	Code        string `json:"code" binding:"omitempty,max=100"`
	Domain      string `json:"domain" binding:"omitempty,max=255"`
	Description string `json:"description"`
	IsActive    *bool  `json:"is_active"`
}

// ListCustomersRequest represents a customer listing request with filters.
type ListCustomersRequest struct {
	Page      int    `form:"page,default=1" binding:"min=1"`
	PageSize  int    `form:"page_size,default=20" binding:"min=1,max=100"`
	Search    string `form:"search"`
	IsActive  *bool  `form:"is_active"`
	SortBy    string `form:"sort_by,default=created_at"`
	SortOrder string `form:"sort_order,default=desc"`
}

// CustomerResponse represents a customer in API responses.
type CustomerResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Domain      string    `json:"domain"`
	IsActive    bool      `json:"is_active"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CustomerUserResponse represents a customer's contact user without any
// sensitive fields (no password hash).
type CustomerUserResponse struct {
	ID        uint      `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateCustomer creates a new customer organization.
func (s *Service) CreateCustomer(req *CreateCustomerRequest) (*CustomerResponse, error) {
	// Normalize input
	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.ToUpper(strings.TrimSpace(req.Code))
	req.Domain = strings.ToLower(strings.TrimSpace(req.Domain))

	// Validate required fields
	if req.Name == "" {
		return nil, apperrors.NewInvalidInputError("name", "客户名称不能为空")
	}

	// Check if customer code already exists (when provided)
	if req.Code != "" {
		var existing models.Customer
		err := s.db.Where("code = ?", req.Code).First(&existing).Error
		if err == nil {
			return nil, apperrors.NewConflictError("客户代码已存在")
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("检查客户代码唯一性时出错: %w", err)
		}
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	customer := &models.Customer{
		BaseModel: models.BaseModel{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Name:        req.Name,
		Code:        optionalString(req.Code),
		Domain:      req.Domain,
		IsActive:    isActive,
		Description: req.Description,
	}

	// Select IsActive explicitly so an intentional false is persisted rather
	// than overridden by the column's default:true (GORM treats the bool zero
	// value as "unset").
	if err := s.db.Omit("Users", "Tickets").Create(customer).Error; err != nil {
		return nil, fmt.Errorf("创建客户失败: %w", err)
	}
	if !isActive {
		if err := s.db.Model(customer).Update("is_active", false).Error; err != nil {
			return nil, fmt.Errorf("创建客户失败: %w", err)
		}
		customer.IsActive = false
	}

	return s.customerToResponse(customer), nil
}

// GetCustomer gets a customer by ID.
func (s *Service) GetCustomer(customerID uint) (*CustomerResponse, error) {
	var customer models.Customer
	if err := s.db.Where("id = ?", customerID).First(&customer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFoundError("客户")
		}
		return nil, fmt.Errorf("获取客户失败: %w", err)
	}

	return s.customerToResponse(&customer), nil
}

// ListCustomers lists customers with pagination and filtering.
func (s *Service) ListCustomers(req *ListCustomersRequest) ([]CustomerResponse, int64, error) {
	query := s.db.Model(&models.Customer{})

	// Apply filters
	if req.Search != "" {
		searchTerm := "%" + req.Search + "%"
		query = query.Where("name LIKE ? OR code LIKE ? OR domain LIKE ? OR description LIKE ?",
			searchTerm, searchTerm, searchTerm, searchTerm)
	}

	if req.IsActive != nil {
		query = query.Where("is_active = ?", *req.IsActive)
	}

	// Count total records
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计客户数量失败: %w", err)
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

	var customers []models.Customer
	if err := query.Offset(offset).
		Limit(req.PageSize).
		Order(fmt.Sprintf("%s %s", req.SortBy, req.SortOrder)).
		Find(&customers).Error; err != nil {
		return nil, 0, fmt.Errorf("获取客户列表失败: %w", err)
	}

	responses := make([]CustomerResponse, 0, len(customers))
	for i := range customers {
		responses = append(responses, *s.customerToResponse(&customers[i]))
	}

	return responses, total, nil
}

// UpdateCustomer updates an existing customer.
func (s *Service) UpdateCustomer(customerID uint, req *UpdateCustomerRequest) (*CustomerResponse, error) {
	var customer models.Customer
	if err := s.db.Where("id = ?", customerID).First(&customer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFoundError("客户")
		}
		return nil, fmt.Errorf("获取客户失败: %w", err)
	}

	if req.Name != "" {
		customer.Name = strings.TrimSpace(req.Name)
	}

	if req.Code != "" {
		req.Code = strings.ToUpper(strings.TrimSpace(req.Code))
		// Check if code conflicts with another customer
		var existing models.Customer
		err := s.db.Where("code = ? AND id != ?", req.Code, customerID).First(&existing).Error
		if err == nil {
			return nil, apperrors.NewConflictError("客户代码已存在")
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("检查客户代码唯一性时出错: %w", err)
		}
		customer.Code = optionalString(req.Code)
	}

	if req.Domain != "" {
		customer.Domain = strings.ToLower(strings.TrimSpace(req.Domain))
	}

	if req.Description != "" {
		customer.Description = req.Description
	}

	if req.IsActive != nil {
		customer.IsActive = *req.IsActive
	}

	customer.UpdatedAt = time.Now()

	if err := s.db.Save(&customer).Error; err != nil {
		return nil, fmt.Errorf("更新客户失败: %w", err)
	}

	return s.customerToResponse(&customer), nil
}

// DeleteCustomer soft deletes a customer.
func (s *Service) DeleteCustomer(customerID uint) error {
	var customer models.Customer
	if err := s.db.Where("id = ?", customerID).First(&customer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.NewNotFoundError("客户")
		}
		return fmt.Errorf("获取客户失败: %w", err)
	}

	if err := s.db.Delete(&customer).Error; err != nil {
		return fmt.Errorf("删除客户失败: %w", err)
	}

	return nil
}

// ListCustomerUsers returns the contact users belonging to a customer, without
// any sensitive fields (no password hash).
func (s *Service) ListCustomerUsers(customerID uint) ([]CustomerUserResponse, error) {
	// Verify the customer exists first to give a clear NotFound.
	var customer models.Customer
	if err := s.db.Where("id = ?", customerID).First(&customer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFoundError("客户")
		}
		return nil, fmt.Errorf("获取客户失败: %w", err)
	}

	var users []models.User
	if err := s.db.Where("customer_id = ?", customerID).
		Order("created_at desc").
		Find(&users).Error; err != nil {
		return nil, fmt.Errorf("获取客户用户失败: %w", err)
	}

	responses := make([]CustomerUserResponse, 0, len(users))
	for i := range users {
		u := &users[i]
		responses = append(responses, CustomerUserResponse{
			ID:        u.ID,
			Email:     u.Email,
			Username:  u.Username,
			FirstName: u.FirstName,
			LastName:  u.LastName,
			Role:      u.Role,
			IsActive:  u.IsActive,
			CreatedAt: u.CreatedAt,
			UpdatedAt: u.UpdatedAt,
		})
	}

	return responses, nil
}

// optionalString returns nil for an empty string, otherwise a pointer to it, so
// that an absent optional unique field stores NULL rather than "".
func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// derefString returns the pointed-to string, or "" when nil.
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (s *Service) customerToResponse(customer *models.Customer) *CustomerResponse {
	return &CustomerResponse{
		ID:          customer.ID,
		Name:        customer.Name,
		Code:        derefString(customer.Code),
		Domain:      customer.Domain,
		IsActive:    customer.IsActive,
		Description: customer.Description,
		CreatedAt:   customer.CreatedAt,
		UpdatedAt:   customer.UpdatedAt,
	}
}
