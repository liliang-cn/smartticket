package product

import (
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

// logProductEvent logs product-related security events.
func (h *Handlers) logProductEvent(c *gin.Context, event, target string) {
	c.Set("security_event", event)
	c.Set("target_resource", target)
}

// CreateProduct handles product creation.
// @Summary Create a new product
// @Description Creates a new product with the provided information
// @Tags products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param request body product.CreateProductRequest true "Product creation data"
// @Success 201 {object} product.ProductResponse
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/products [post]
func (h *Handlers) CreateProduct(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := apperrors.NewInvalidInputError("request_body", err.Error())
		apperrors.ErrorHandler(c, appErr)
		return
	}

	// Log product creation attempt
	c.Set("security_event", "product_creation_attempt")
	c.Set("target_resource", req.Name)

	product, err := h.service.CreateProduct(&req)
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
// @Summary List products
// @Description Retrieves a paginated list of products with optional filtering
// @Tags products
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param page query int false "Page number" default(1) minimum(1)
// @Param page_size query int false "Number of products per page" default(20) minimum(1) maximum(100)
// @Param search query string false "Search products by name or description"
// @Param category query string false "Filter by category"
// @Param status query string false "Filter by status" Enums(active,inactive,deprecated)
// @Success 200 {object} github_com_company_smartticket_internal_server.Response
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/products [get]
func (h *Handlers) ListProducts(c *gin.Context) {
	var req ListProductsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		appErr := apperrors.NewInvalidInputError("query_params", err.Error())
		apperrors.ErrorHandler(c, appErr)
		return
	}

	products, total, err := h.service.ListProducts(&req)
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
// @Summary Get a product by ID
// @Description Retrieves a specific product by its unique identifier
// @Tags products
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param id path int true "Product ID"
// @Success 200 {object} product.ProductResponse
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 404 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/products/{id} [get]
func (h *Handlers) GetProduct(c *gin.Context) {
	productID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErr := apperrors.NewInvalidInputError("id", "无效的产品ID")
		apperrors.ErrorHandler(c, appErr)
		return
	}

	product, err := h.service.GetProduct(uint(productID))
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
// @Summary Update a product
// @Description Updates an existing product with new information
// @Tags products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param id path int true "Product ID"
// @Param request body product.UpdateProductRequest true "Product update data"
// @Success 200 {object} product.ProductResponse
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 404 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/products/{id} [put]
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

	// Log product update attempt
	c.Set("security_event", "product_update_attempt")
	c.Set("target_resource", strconv.FormatUint(productID, 10))

	product, err := h.service.UpdateProduct(uint(productID), &req)
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
// @Summary Delete a product
// @Description Soft deletes a product (marks as deleted but preserves data)
// @Tags products
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param id path int true "Product ID"
// @Success 200 {object} github_com_company_smartticket_internal_server.Response
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 404 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/products/{id} [delete]
func (h *Handlers) DeleteProduct(c *gin.Context) {
	productID, err := h.parseProductID(c)
	if err != nil {
		return
	}

	// Log product deletion attempt
	h.logProductEvent(c, "product_deletion_attempt", strconv.FormatUint(uint64(productID), 10))

	err = h.service.DeleteProduct(productID)
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
// @Summary Activate a product
// @Description Activates an existing product, making it available for use
// @Tags products
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param id path int true "Product ID"
// @Success 200 {object} github_com_company_smartticket_internal_server.Response
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 404 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/products/{id}/activate [post]
func (h *Handlers) ActivateProduct(c *gin.Context) {
	productID, err := h.parseProductID(c)
	if err != nil {
		return
	}

	// Log product activation attempt
	h.logProductEvent(c, "product_activation_attempt", strconv.FormatUint(uint64(productID), 10))

	err = h.service.ActivateProduct(productID)
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
// @Summary Deactivate a product
// @Description Deactivates an existing product, making it unavailable for use
// @Tags products
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param id path int true "Product ID"
// @Success 200 {object} github_com_company_smartticket_internal_server.Response
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 404 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/products/{id}/deactivate [post]
func (h *Handlers) DeactivateProduct(c *gin.Context) {
	productID, err := h.parseProductID(c)
	if err != nil {
		return
	}

	// Log product deactivation attempt
	h.logProductEvent(c, "product_deactivation_attempt", strconv.FormatUint(uint64(productID), 10))

	err = h.service.DeactivateProduct(productID)
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
