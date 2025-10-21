package product

import (
	"errors"
	"net/http"
	"strconv"

	apperrors "github.com/company/smartticket/internal/errors"
	"github.com/gin-gonic/gin"
)

// Handlers provides HTTP handlers for product management.
type Handlers struct {
	service *Service
}

// NewHandlers creates a new product handlers instance.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// parseProductID extracts and validates product ID from request parameters.
func (h *Handlers) parseProductID(c *gin.Context) (uint, error) {
	productID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErr := apperrors.NewInvalidInputError("id", "无效的产品ID")
		apperrors.ErrorHandler(c, appErr)
		return 0, err
	}
	return uint(productID), nil
}

// getTenantID extracts tenant ID from context with error handling.
func (h *Handlers) getTenantID(c *gin.Context) (uint, error) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		appErr := apperrors.NewUnauthorizedError("租户信息缺失")
		apperrors.ErrorHandler(c, appErr)
		return 0, errors.New("tenant info missing")
	}
	return tenantID.(uint), nil
}

// logProductEvent logs product-related security events.
func (h *Handlers) logProductEvent(c *gin.Context, event, target string) {
	c.Set("security_event", event)
	c.Set("target_resource", target)
}

// CreateProduct handles product creation.
func (h *Handlers) CreateProduct(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := apperrors.NewInvalidInputError("request_body", err.Error())
		apperrors.ErrorHandler(c, appErr)
		return
	}

	// Get user info from context
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		appErr := apperrors.NewUnauthorizedError("租户信息缺失")
		apperrors.ErrorHandler(c, appErr)
		return
	}

	// Log product creation attempt
	c.Set("security_event", "product_creation_attempt")
	c.Set("target_resource", req.Name)

	product, err := h.service.CreateProduct(tenantID.(uint), &req)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	// Log successful product creation
	c.Set("security_event", "product_created")
	c.Set("resource_id", product.ID)
	c.Set("resource_name", product.Name)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    product,
		"message": "产品创建成功",
	})
}

// ListProducts handles product listing.
func (h *Handlers) ListProducts(c *gin.Context) {
	var req ListProductsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		appErr := apperrors.NewInvalidInputError("query_params", err.Error())
		apperrors.ErrorHandler(c, appErr)
		return
	}

	// Get user info from context
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		appErr := apperrors.NewUnauthorizedError("租户信息缺失")
		apperrors.ErrorHandler(c, appErr)
		return
	}

	products, total, err := h.service.ListProducts(tenantID.(uint), &req)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	// Calculate pagination
	totalPages := (total + int64(req.PageSize) - 1) / int64(req.PageSize)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    products,
		"meta": gin.H{
			"page":        req.Page,
			"page_size":   req.PageSize,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// GetProduct handles getting a single product.
func (h *Handlers) GetProduct(c *gin.Context) {
	productID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErr := apperrors.NewInvalidInputError("id", "无效的产品ID")
		apperrors.ErrorHandler(c, appErr)
		return
	}

	// Get user info from context
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		appErr := apperrors.NewUnauthorizedError("租户信息缺失")
		apperrors.ErrorHandler(c, appErr)
		return
	}

	product, err := h.service.GetProduct(tenantID.(uint), uint(productID))
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    product,
	})
}

// UpdateProduct handles product update.
func (h *Handlers) UpdateProduct(c *gin.Context) {
	productID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErr := apperrors.NewInvalidInputError("id", "无效的产品ID")
		apperrors.ErrorHandler(c, appErr)
		return
	}

	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := apperrors.NewInvalidInputError("request_body", err.Error())
		apperrors.ErrorHandler(c, appErr)
		return
	}

	// Get user info from context
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		appErr := apperrors.NewUnauthorizedError("租户信息缺失")
		apperrors.ErrorHandler(c, appErr)
		return
	}

	// Log product update attempt
	c.Set("security_event", "product_update_attempt")
	c.Set("target_resource", strconv.FormatUint(productID, 10))

	product, err := h.service.UpdateProduct(tenantID.(uint), uint(productID), &req)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	// Log successful product update
	c.Set("security_event", "product_updated")
	c.Set("resource_id", product.ID)
	c.Set("resource_name", product.Name)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    product,
		"message": "产品更新成功",
	})
}

// DeleteProduct handles product deletion.
func (h *Handlers) DeleteProduct(c *gin.Context) {
	productID, err := h.parseProductID(c)
	if err != nil {
		return
	}

	tenantID, err := h.getTenantID(c)
	if err != nil {
		return
	}

	// Log product deletion attempt
	h.logProductEvent(c, "product_deletion_attempt", strconv.FormatUint(uint64(productID), 10))

	err = h.service.DeleteProduct(tenantID, productID)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	// Log successful product deletion
	c.Set("security_event", "product_deleted")
	c.Set("resource_id", productID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "产品删除成功",
	})
}

// ActivateProduct handles product activation.
func (h *Handlers) ActivateProduct(c *gin.Context) {
	productID, err := h.parseProductID(c)
	if err != nil {
		return
	}

	tenantID, err := h.getTenantID(c)
	if err != nil {
		return
	}

	// Log product activation attempt
	h.logProductEvent(c, "product_activation_attempt", strconv.FormatUint(uint64(productID), 10))

	err = h.service.ActivateProduct(tenantID, productID)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	// Log successful product activation
	c.Set("security_event", "product_activated")
	c.Set("resource_id", productID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "产品激活成功",
	})
}

// DeactivateProduct handles product deactivation.
func (h *Handlers) DeactivateProduct(c *gin.Context) {
	productID, err := h.parseProductID(c)
	if err != nil {
		return
	}

	tenantID, err := h.getTenantID(c)
	if err != nil {
		return
	}

	// Log product deactivation attempt
	h.logProductEvent(c, "product_deactivation_attempt", strconv.FormatUint(uint64(productID), 10))

	err = h.service.DeactivateProduct(tenantID, productID)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	// Log successful product deactivation
	c.Set("security_event", "product_deactivated")
	c.Set("resource_id", productID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "产品停用成功",
	})
}
