package subscription

import (
	"errors"
	"fmt"
	"strings"
	"time"

	apperrors "github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/models"
	"gorm.io/gorm"
)

// Service provides support-subscription / licensing management functionality.
type Service struct {
	db *gorm.DB
}

// NewService creates a new subscription service instance.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// Request/Response structures for subscription management.

// CreateSubscriptionRequest is the payload for creating a subscription.
type CreateSubscriptionRequest struct {
	CustomerID    uint      `json:"customer_id" binding:"required"`
	ProductID     uint      `json:"product_id" binding:"required"`
	SLATemplateID *uint     `json:"sla_template_id"`
	Plan          string    `json:"plan"`
	BillingUnit   string    `json:"billing_unit"`
	NodeCount     int       `json:"node_count"`
	BillingPeriod string    `json:"billing_period"`
	StartsAt      time.Time `json:"starts_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	Status        string    `json:"status"`
	UnitPrice     float64   `json:"unit_price"`
	Currency      string    `json:"currency"`
	Notes         string    `json:"notes"`
}

// UpdateSubscriptionRequest patches provided subscription fields.
type UpdateSubscriptionRequest struct {
	SLATemplateID *uint      `json:"sla_template_id,omitempty"`
	Plan          *string    `json:"plan,omitempty"`
	BillingUnit   *string    `json:"billing_unit,omitempty"`
	NodeCount     *int       `json:"node_count,omitempty"`
	BillingPeriod *string    `json:"billing_period,omitempty"`
	StartsAt      *time.Time `json:"starts_at,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	Status        *string    `json:"status,omitempty"`
	UnitPrice     *float64   `json:"unit_price,omitempty"`
	Currency      *string    `json:"currency,omitempty"`
	Notes         *string    `json:"notes,omitempty"`
}

// ListSubscriptionsRequest filters/paginates the subscription list.
type ListSubscriptionsRequest struct {
	Page       int    `form:"page,default=1" binding:"min=1"`
	PageSize   int    `form:"page_size,default=20" binding:"min=1,max=100"`
	CustomerID *uint  `form:"customer_id"`
	Status     string `form:"status"`
}

// SubscriptionResponse is a flat DTO for a subscription including related names.
type SubscriptionResponse struct {
	ID              uint      `json:"id"`
	CustomerID      uint      `json:"customer_id"`
	CustomerName    string    `json:"customer_name"`
	ProductID       uint      `json:"product_id"`
	ProductName     string    `json:"product_name"`
	SLATemplateID   *uint     `json:"sla_template_id"`
	SLATemplateName string    `json:"sla_template_name"`
	Plan            string    `json:"plan"`
	BillingUnit     string    `json:"billing_unit"`
	NodeCount       int       `json:"node_count"`
	TotalUnits      int       `json:"total_units"`
	BillingPeriod   string    `json:"billing_period"`
	StartsAt        time.Time `json:"starts_at"`
	ExpiresAt       time.Time `json:"expires_at"`
	Status          string    `json:"status"`
	UnitPrice       float64   `json:"unit_price"`
	Currency        string    `json:"currency"`
	Notes           string    `json:"notes"`
	IsExpired       bool      `json:"is_expired"`
	IsDeleted       bool      `json:"is_deleted"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

const (
	billingUnitPerNode    = "per_node"
	billingUnitPerCluster = "per_cluster"
	billingPeriodAnnual   = "annual"
	billingPeriodMonthly  = "monthly"
	statusActive          = "active"
	statusExpired         = "expired"
	statusCancelled       = "cancelled"
)

func isValidBillingUnit(v string) bool {
	return v == billingUnitPerNode || v == billingUnitPerCluster
}

func isValidBillingPeriod(v string) bool {
	return v == billingPeriodAnnual || v == billingPeriodMonthly
}

func isValidStatus(v string) bool {
	return v == statusActive || v == statusExpired || v == statusCancelled
}

// Create creates a new subscription.
func (s *Service) Create(req *CreateSubscriptionRequest) (*SubscriptionResponse, error) {
	if req.CustomerID == 0 {
		return nil, apperrors.NewInvalidInputError("customer_id", "customer is required")
	}
	if req.ProductID == 0 {
		return nil, apperrors.NewInvalidInputError("product_id", "product is required")
	}

	// Validate customer exists.
	var customer models.Customer
	if err := s.db.First(&customer, req.CustomerID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewValidationError("customer not found")
		}
		return nil, fmt.Errorf("查找客户失败: %w", err)
	}

	// Validate product exists.
	var product models.Product
	if err := s.db.First(&product, req.ProductID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewValidationError("product not found")
		}
		return nil, fmt.Errorf("查找产品失败: %w", err)
	}

	// Validate SLA template if provided.
	if req.SLATemplateID != nil {
		var slaTemplate models.SLATemplate
		if err := s.db.First(&slaTemplate, *req.SLATemplateID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperrors.NewValidationError("sla template not found")
			}
			return nil, fmt.Errorf("查找SLA模板失败: %w", err)
		}
	}

	// Apply defaults.
	billingUnit := strings.TrimSpace(req.BillingUnit)
	if billingUnit == "" {
		billingUnit = billingUnitPerNode
	}
	if !isValidBillingUnit(billingUnit) {
		return nil, apperrors.NewInvalidInputError("billing_unit", "must be per_node or per_cluster")
	}

	billingPeriod := strings.TrimSpace(req.BillingPeriod)
	if billingPeriod == "" {
		billingPeriod = billingPeriodAnnual
	}
	if !isValidBillingPeriod(billingPeriod) {
		return nil, apperrors.NewInvalidInputError("billing_period", "must be annual or monthly")
	}

	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = statusActive
	}
	if !isValidStatus(status) {
		return nil, apperrors.NewInvalidInputError("status", "must be active, expired or cancelled")
	}

	nodeCount := req.NodeCount
	if nodeCount == 0 {
		nodeCount = 1
	}
	if nodeCount < 1 {
		return nil, apperrors.NewInvalidInputError("node_count", "must be >= 1")
	}

	currency := strings.TrimSpace(req.Currency)
	if currency == "" {
		currency = "USD"
	}

	subscription := &models.Subscription{
		BaseModel: models.BaseModel{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		CustomerID:    req.CustomerID,
		ProductID:     req.ProductID,
		SLATemplateID: req.SLATemplateID,
		Plan:          strings.TrimSpace(req.Plan),
		BillingUnit:   billingUnit,
		NodeCount:     nodeCount,
		BillingPeriod: billingPeriod,
		StartsAt:      req.StartsAt,
		ExpiresAt:     req.ExpiresAt,
		Status:        status,
		UnitPrice:     req.UnitPrice,
		Currency:      currency,
		Notes:         req.Notes,
	}

	if err := s.db.Create(subscription).Error; err != nil {
		return nil, fmt.Errorf("创建订阅失败: %w", err)
	}

	return s.Get(subscription.ID)
}

// List lists subscriptions with pagination and optional filters.
func (s *Service) List(req *ListSubscriptionsRequest) ([]SubscriptionResponse, int64, error) {
	query := s.db.Model(&models.Subscription{})

	if req.CustomerID != nil {
		query = query.Where("customer_id = ?", *req.CustomerID)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计订阅数量失败: %w", err)
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}
	offset := (req.Page - 1) * req.PageSize

	var subscriptions []models.Subscription
	if err := query.
		Preload("Customer").
		Preload("Product").
		Preload("SLATemplate").
		Offset(offset).
		Limit(req.PageSize).
		Order("created_at desc").
		Find(&subscriptions).Error; err != nil {
		return nil, 0, fmt.Errorf("获取订阅列表失败: %w", err)
	}

	responses := make([]SubscriptionResponse, 0, len(subscriptions))
	for i := range subscriptions {
		responses = append(responses, *toResponse(&subscriptions[i]))
	}

	return responses, total, nil
}

// Get returns a subscription by ID with related entities preloaded.
func (s *Service) Get(id uint) (*SubscriptionResponse, error) {
	var subscription models.Subscription
	if err := s.db.
		Preload("Customer").
		Preload("Product").
		Preload("SLATemplate").
		First(&subscription, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFoundError("subscription")
		}
		return nil, fmt.Errorf("获取订阅失败: %w", err)
	}
	return toResponse(&subscription), nil
}

// Update patches an existing subscription with the provided fields.
func (s *Service) Update(id uint, req *UpdateSubscriptionRequest) (*SubscriptionResponse, error) {
	var subscription models.Subscription
	if err := s.db.First(&subscription, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFoundError("subscription")
		}
		return nil, fmt.Errorf("获取订阅失败: %w", err)
	}

	if req.SLATemplateID != nil {
		var slaTemplate models.SLATemplate
		if err := s.db.First(&slaTemplate, *req.SLATemplateID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperrors.NewValidationError("sla template not found")
			}
			return nil, fmt.Errorf("查找SLA模板失败: %w", err)
		}
		subscription.SLATemplateID = req.SLATemplateID
	}

	if req.Plan != nil {
		subscription.Plan = strings.TrimSpace(*req.Plan)
	}
	if req.BillingUnit != nil {
		v := strings.TrimSpace(*req.BillingUnit)
		if !isValidBillingUnit(v) {
			return nil, apperrors.NewInvalidInputError("billing_unit", "must be per_node or per_cluster")
		}
		subscription.BillingUnit = v
	}
	if req.NodeCount != nil {
		if *req.NodeCount < 1 {
			return nil, apperrors.NewInvalidInputError("node_count", "must be >= 1")
		}
		subscription.NodeCount = *req.NodeCount
	}
	if req.BillingPeriod != nil {
		v := strings.TrimSpace(*req.BillingPeriod)
		if !isValidBillingPeriod(v) {
			return nil, apperrors.NewInvalidInputError("billing_period", "must be annual or monthly")
		}
		subscription.BillingPeriod = v
	}
	if req.StartsAt != nil {
		subscription.StartsAt = *req.StartsAt
	}
	if req.ExpiresAt != nil {
		subscription.ExpiresAt = *req.ExpiresAt
	}
	if req.Status != nil {
		v := strings.TrimSpace(*req.Status)
		if !isValidStatus(v) {
			return nil, apperrors.NewInvalidInputError("status", "must be active, expired or cancelled")
		}
		subscription.Status = v
	}
	if req.UnitPrice != nil {
		subscription.UnitPrice = *req.UnitPrice
	}
	if req.Currency != nil {
		v := strings.TrimSpace(*req.Currency)
		if v != "" {
			subscription.Currency = v
		}
	}
	if req.Notes != nil {
		subscription.Notes = *req.Notes
	}

	subscription.UpdatedAt = time.Now()

	if err := s.db.Save(&subscription).Error; err != nil {
		return nil, fmt.Errorf("更新订阅失败: %w", err)
	}

	return s.Get(subscription.ID)
}

// Delete soft deletes a subscription.
func (s *Service) Delete(id uint) error {
	var subscription models.Subscription
	if err := s.db.First(&subscription, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.NewNotFoundError("subscription")
		}
		return fmt.Errorf("获取订阅失败: %w", err)
	}

	if err := s.db.Delete(&subscription).Error; err != nil {
		return fmt.Errorf("删除订阅失败: %w", err)
	}
	return nil
}

func toResponse(sub *models.Subscription) *SubscriptionResponse {
	resp := &SubscriptionResponse{
		ID:            sub.ID,
		CustomerID:    sub.CustomerID,
		ProductID:     sub.ProductID,
		SLATemplateID: sub.SLATemplateID,
		Plan:          sub.Plan,
		BillingUnit:   sub.BillingUnit,
		NodeCount:     sub.NodeCount,
		BillingPeriod: sub.BillingPeriod,
		StartsAt:      sub.StartsAt,
		ExpiresAt:     sub.ExpiresAt,
		Status:        sub.Status,
		UnitPrice:     sub.UnitPrice,
		Currency:      sub.Currency,
		Notes:         sub.Notes,
		IsDeleted:     sub.DeletedAt.Valid,
		CreatedAt:     sub.CreatedAt,
		UpdatedAt:     sub.UpdatedAt,
	}

	// total_units derives from node_count for per_node billing.
	resp.TotalUnits = sub.NodeCount

	if !sub.ExpiresAt.IsZero() {
		resp.IsExpired = sub.ExpiresAt.Before(time.Now())
	}

	if sub.Customer != nil {
		resp.CustomerName = sub.Customer.Name
	}
	if sub.Product != nil {
		resp.ProductName = sub.Product.Name
	}
	if sub.SLATemplate != nil {
		resp.SLATemplateName = sub.SLATemplate.Name
	}

	return resp
}
