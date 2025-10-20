package product

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	apperrors "github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/models"
	"gorm.io/gorm"
)

// Service provides product management functionality
type Service struct {
	db *gorm.DB
}

// NewService creates a new product service instance
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// Request/Response structures for product management
type CreateProductRequest struct {
	Name          string `json:"name" binding:"required,min=1,max=255"`
	Code          string `json:"code" binding:"required,min=1,max=100"`
	Description   string `json:"description"`
	Category      string `json:"category"`
	Version       string `json:"version"`
	Status        string `json:"status"`
	IsManaged     bool   `json:"is_managed"`
	SupportLevel  string `json:"support_level"`
	Documentation string `json:"documentation"`
	Configuration string `json:"configuration"`
	Tags          string `json:"tags"`
}

type UpdateProductRequest struct {
	Name          string `json:"name" binding:"omitempty,min=1,max=255"`
	Code          string `json:"code" binding:"omitempty,min=1,max=100"`
	Description   string `json:"description"`
	Category      string `json:"category"`
	Version       string `json:"version"`
	Status        string `json:"status"`
	IsManaged     *bool  `json:"is_managed"`
	SupportLevel  string `json:"support_level"`
	Documentation string `json:"documentation"`
	Configuration string `json:"configuration"`
	Tags          string `json:"tags"`
}

type ProductResponse struct {
	ID            uint                   `json:"id"`
	Name          string                 `json:"name"`
	Code          string                 `json:"code"`
	Description   string                 `json:"description"`
	Category      string                 `json:"category"`
	Version       string                 `json:"version"`
	Status        string                 `json:"status"`
	IsManaged     bool                   `json:"is_managed"`
	SupportLevel  string                 `json:"support_level"`
	Documentation string                 `json:"documentation"`
	Configuration map[string]interface{} `json:"configuration"`
	Tags          []string               `json:"tags"`
	TenantID      uint                   `json:"tenant_id"`
	IsDeleted     bool                   `json:"is_deleted"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	Services      []ServiceResponse      `json:"services,omitempty"`
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
}

type ListProductsRequest struct {
	Page      int    `form:"page,default=1" binding:"min=1"`
	PageSize  int    `form:"page_size,default=20" binding:"min=1,max=100"`
	Search    string `form:"search"`
	Category  string `form:"category"`
	Status    string `form:"status"`
	IsManaged *bool  `form:"is_managed"`
	SortBy    string `form:"sort_by,default=created_at"`
	SortOrder string `form:"sort_order,default=desc"`
}

// CreateProduct creates a new product
func (s *Service) CreateProduct(tenantID uint, req *CreateProductRequest) (*ProductResponse, error) {
	// Normalize input
	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.ToUpper(strings.TrimSpace(req.Code))

	// Validate required fields
	if req.Name == "" {
		return nil, apperrors.NewInvalidInputError("name", "产品名称不能为空")
	}
	if req.Code == "" {
		return nil, apperrors.NewInvalidInputError("code", "产品代码不能为空")
	}

	// Check if product code already exists for this tenant
	var existingProduct models.Product
	err := s.db.Where("tenant_id = ? AND code = ?", tenantID, req.Code).First(&existingProduct).Error
	if err == nil {
		return nil, apperrors.NewConflictError("产品代码已存在")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("检查产品代码唯一性时出错: %w", err)
	}

	// Validate status
	if req.Status != "" && req.Status != "active" && req.Status != "inactive" && req.Status != "deprecated" {
		return nil, apperrors.NewInvalidInputError("status", "无效的产品状态")
	}

	// Validate support level
	if req.SupportLevel != "" && req.SupportLevel != "basic" && req.SupportLevel != "premium" && req.SupportLevel != "enterprise" {
		return nil, apperrors.NewInvalidInputError("support_level", "无效的支持级别")
	}

	// Parse configuration JSON
	var configMap map[string]interface{}
	if req.Configuration != "" {
		if err := json.Unmarshal([]byte(req.Configuration), &configMap); err != nil {
			return nil, apperrors.NewInvalidInputError("configuration", "配置格式无效，必须是有效的JSON")
		}
	}

	// Create product
	product := &models.Product{
		BaseModel: models.BaseModel{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			CreatedBy: nil, // TODO: Set to current user
			UpdatedBy: nil, // TODO: Set to current user
		},
		TenantID:      tenantID,
		Name:          req.Name,
		Code:          req.Code,
		Description:   req.Description,
		Category:      req.Category,
		Version:       req.Version,
		Status:        "active", // Default status
		IsManaged:     req.IsManaged,
		SupportLevel:  req.SupportLevel,
		Documentation: req.Documentation,
		Configuration: req.Configuration,
		Tags:          req.Tags,
	}

	if req.Status != "" {
		product.Status = req.Status
	}

	if err := s.db.Create(product).Error; err != nil {
		return nil, fmt.Errorf("创建产品失败: %w", err)
	}

	return s.productToResponse(product), nil
}

// GetProduct gets a product by ID
func (s *Service) GetProduct(tenantID uint, productID uint) (*ProductResponse, error) {
	var product models.Product
	if err := s.db.Where("id = ? AND tenant_id = ?", productID, tenantID).
		Preload("Services").
		First(&product).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFoundError("产品")
		}
		return nil, fmt.Errorf("获取产品失败: %w", err)
	}

	return s.productToResponse(&product), nil
}

// ListProducts lists products with pagination and filtering
func (s *Service) ListProducts(tenantID uint, req *ListProductsRequest) ([]ProductResponse, int64, error) {
	// Build query
	query := s.db.Where("tenant_id = ?", tenantID)

	// Apply filters
	if req.Search != "" {
		searchTerm := "%" + req.Search + "%"
		query = query.Where("name LIKE ? OR code LIKE ? OR description LIKE ?", searchTerm, searchTerm, searchTerm)
	}

	if req.Category != "" {
		query = query.Where("category = ?", req.Category)
	}

	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	if req.IsManaged != nil {
		query = query.Where("is_managed = ?", *req.IsManaged)
	}

	// Count total records
	var total int64
	if err := query.Model(&models.Product{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计产品数量失败: %w", err)
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
		"category":   true,
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
	var products []models.Product
	if err := query.Offset(offset).
		Limit(req.PageSize).
		Order(fmt.Sprintf("%s %s", req.SortBy, req.SortOrder)).
		Find(&products).Error; err != nil {
		return nil, 0, fmt.Errorf("获取产品列表失败: %w", err)
	}

	// Convert to response
	var responses []ProductResponse
	for _, product := range products {
		responses = append(responses, *s.productToResponse(&product))
	}

	return responses, total, nil
}

// UpdateProduct updates an existing product
func (s *Service) UpdateProduct(tenantID uint, productID uint, req *UpdateProductRequest) (*ProductResponse, error) {
	var product models.Product
	if err := s.db.Where("id = ? AND tenant_id = ?", productID, tenantID).First(&product).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFoundError("产品")
		}
		return nil, fmt.Errorf("获取产品失败: %w", err)
	}

	// Update fields if provided
	if req.Name != "" {
		product.Name = strings.TrimSpace(req.Name)
	}

	if req.Code != "" {
		req.Code = strings.ToUpper(strings.TrimSpace(req.Code))
		// Check if code conflicts with another product
		var existingProduct models.Product
		err := s.db.Where("tenant_id = ? AND code = ? AND id != ?", tenantID, req.Code, productID).First(&existingProduct).Error
		if err == nil {
			return nil, apperrors.NewConflictError("产品代码已存在")
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("检查产品代码唯一性时出错: %w", err)
		}
		product.Code = req.Code
	}

	if req.Description != "" {
		product.Description = req.Description
	}

	if req.Category != "" {
		product.Category = req.Category
	}

	if req.Version != "" {
		product.Version = req.Version
	}

	if req.Status != "" {
		if req.Status != "active" && req.Status != "inactive" && req.Status != "deprecated" {
			return nil, apperrors.NewInvalidInputError("status", "无效的产品状态")
		}
		product.Status = req.Status
	}

	if req.IsManaged != nil {
		product.IsManaged = *req.IsManaged
	}

	if req.SupportLevel != "" {
		if req.SupportLevel != "basic" && req.SupportLevel != "premium" && req.SupportLevel != "enterprise" {
			return nil, apperrors.NewInvalidInputError("support_level", "无效的支持级别")
		}
		product.SupportLevel = req.SupportLevel
	}

	if req.Documentation != "" {
		product.Documentation = req.Documentation
	}

	if req.Configuration != "" {
		// Validate JSON format
		var configMap map[string]interface{}
		if err := json.Unmarshal([]byte(req.Configuration), &configMap); err != nil {
			return nil, apperrors.NewInvalidInputError("configuration", "配置格式无效，必须是有效的JSON")
		}
		product.Configuration = req.Configuration
	}

	if req.Tags != "" {
		product.Tags = req.Tags
	}

	product.UpdatedAt = time.Now()
	product.UpdatedBy = nil // TODO: Set to current user

	if err := s.db.Save(&product).Error; err != nil {
		return nil, fmt.Errorf("更新产品失败: %w", err)
	}

	// Reload with associations
	if err := s.db.Preload("Services").First(&product, product.ID).Error; err != nil {
		return nil, fmt.Errorf("重新加载产品失败: %w", err)
	}

	return s.productToResponse(&product), nil
}

// DeleteProduct soft deletes a product
func (s *Service) DeleteProduct(tenantID uint, productID uint) error {
	var product models.Product
	if err := s.db.Where("id = ? AND tenant_id = ?", productID, tenantID).First(&product).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.NewNotFoundError("产品")
		}
		return fmt.Errorf("获取产品失败: %w", err)
	}

	// Check if product has associated services
	var serviceCount int64
	if err := s.db.Model(&models.Service{}).Where("product_id = ?", productID).Count(&serviceCount).Error; err != nil {
		return fmt.Errorf("检查关联服务失败: %w", err)
	}

	if serviceCount > 0 {
		return apperrors.NewBusinessRuleError("无法删除产品", "请先删除或转移其关联的服务")
	}

	// Check if product has associated tickets
	var ticketCount int64
	if err := s.db.Model(&models.Ticket{}).Where("product_id = ?", productID).Count(&ticketCount).Error; err != nil {
		return fmt.Errorf("检查关联工单失败: %w", err)
	}

	if ticketCount > 0 {
		return apperrors.NewBusinessRuleError("Cannot delete product", "Please delete or transfer associated tickets first")
	}

	// Soft delete
	if err := s.db.Delete(&product).Error; err != nil {
		return fmt.Errorf("删除产品失败: %w", err)
	}

	return nil
}

// ActivateProduct activates a product
func (s *Service) ActivateProduct(tenantID uint, productID uint) error {
	return s.updateProductStatus(tenantID, productID, "active")
}

// DeactivateProduct deactivates a product
func (s *Service) DeactivateProduct(tenantID uint, productID uint) error {
	return s.updateProductStatus(tenantID, productID, "inactive")
}

// Helper functions
func (s *Service) updateProductStatus(tenantID uint, productID uint, status string) error {
	var product models.Product
	if err := s.db.Where("id = ? AND tenant_id = ?", productID, tenantID).First(&product).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.NewNotFoundError("产品")
		}
		return fmt.Errorf("获取产品失败: %w", err)
	}

	product.Status = status
	product.UpdatedAt = time.Now()
	product.UpdatedBy = nil // TODO: Set to current user

	if err := s.db.Save(&product).Error; err != nil {
		return fmt.Errorf("更新产品状态失败: %w", err)
	}

	return nil
}

func (s *Service) productToResponse(product *models.Product) *ProductResponse {
	response := &ProductResponse{
		ID:            product.ID,
		Name:          product.Name,
		Code:          product.Code,
		Description:   product.Description,
		Category:      product.Category,
		Version:       product.Version,
		Status:        product.Status,
		IsManaged:     product.IsManaged,
		SupportLevel:  product.SupportLevel,
		Documentation: product.Documentation,
		TenantID:      product.TenantID,
		IsDeleted:     product.DeletedAt.Valid,
		CreatedAt:     product.CreatedAt,
		UpdatedAt:     product.UpdatedAt,
		Tags:          []string{},
	}

	// Parse configuration JSON
	if product.Configuration != "" {
		var configMap map[string]interface{}
		if err := json.Unmarshal([]byte(product.Configuration), &configMap); err == nil {
			response.Configuration = configMap
		}
	}

	// Parse tags JSON
	if product.Tags != "" {
		var tagsArray []string
		if err := json.Unmarshal([]byte(product.Tags), &tagsArray); err == nil {
			response.Tags = tagsArray
		}
	}

	// Convert services
	for _, service := range product.Services {
		response.Services = append(response.Services, *s.serviceToResponse(&service))
	}

	return response
}

func (s *Service) serviceToResponse(service *models.Service) *ServiceResponse {
	response := &ServiceResponse{
		ID:           service.ID,
		ProductID:    service.ProductID,
		Name:         service.Name,
		Code:         service.Code,
		Description:  service.Description,
		Type:         service.Type,
		Status:       service.Status,
		Availability: service.Availability,
		TenantID:     service.TenantID,
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

	return response
}
