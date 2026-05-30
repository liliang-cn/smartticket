package customer

import (
	"net/http"
	"strconv"

	apperrors "github.com/company/smartticket/internal/errors"
	"github.com/gin-gonic/gin"
)

// Handlers provides HTTP handlers for customer management.
type Handlers struct {
	service *Service
}

// NewHandlers creates a new customer handlers instance.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// parseCustomerID extracts and validates the customer ID from request parameters.
func (h *Handlers) parseCustomerID(c *gin.Context) (uint, error) {
	customerID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErr := apperrors.NewInvalidInputError("id", "无效的客户ID")
		apperrors.ErrorHandler(c, appErr)
		return 0, err
	}
	return uint(customerID), nil
}

// CreateCustomer handles customer creation.
// @Summary Create a new customer
// @Description Creates a new customer organization
// @Tags customers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param request body customer.CreateCustomerRequest true "Customer creation data"
// @Success 201 {object} customer.CustomerResponse
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 409 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/customers [post]
func (h *Handlers) CreateCustomer(c *gin.Context) {
	var req CreateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := apperrors.NewInvalidInputError("request_body", err.Error())
		apperrors.ErrorHandler(c, appErr)
		return
	}

	c.Set("security_event", "customer_creation_attempt")
	c.Set("target_resource", req.Name)

	customer, err := h.service.CreateCustomer(&req)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	c.Set("security_event", "customer_created")
	c.Set("resource_id", customer.ID)
	c.Set("resource_name", customer.Name)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    customer,
		"message": "客户创建成功",
	})
}

// ListCustomers handles customer listing.
// @Summary List customers
// @Description Retrieves a paginated list of customers with optional filtering
// @Tags customers
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param page query int false "Page number" default(1) minimum(1)
// @Param page_size query int false "Number of customers per page" default(20) minimum(1) maximum(100)
// @Param search query string false "Search customers by name, code, domain or description"
// @Param is_active query bool false "Filter by active status"
// @Success 200 {object} github_com_company_smartticket_internal_server.Response
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/customers [get]
func (h *Handlers) ListCustomers(c *gin.Context) {
	var req ListCustomersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		appErr := apperrors.NewInvalidInputError("query_params", err.Error())
		apperrors.ErrorHandler(c, appErr)
		return
	}

	customers, total, err := h.service.ListCustomers(&req)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	totalPages := (total + int64(req.PageSize) - 1) / int64(req.PageSize)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    customers,
		"meta": gin.H{
			"page":        req.Page,
			"page_size":   req.PageSize,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// GetCustomer handles getting a single customer.
// @Summary Get a customer by ID
// @Description Retrieves a specific customer by its unique identifier
// @Tags customers
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param id path int true "Customer ID"
// @Success 200 {object} customer.CustomerResponse
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 404 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/customers/{id} [get]
func (h *Handlers) GetCustomer(c *gin.Context) {
	customerID, err := h.parseCustomerID(c)
	if err != nil {
		return
	}

	customer, err := h.service.GetCustomer(customerID)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    customer,
	})
}

// UpdateCustomer handles customer update.
// @Summary Update a customer
// @Description Updates an existing customer with new information
// @Tags customers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param id path int true "Customer ID"
// @Param request body customer.UpdateCustomerRequest true "Customer update data"
// @Success 200 {object} customer.CustomerResponse
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 404 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 409 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/customers/{id} [put]
func (h *Handlers) UpdateCustomer(c *gin.Context) {
	customerID, err := h.parseCustomerID(c)
	if err != nil {
		return
	}

	var req UpdateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := apperrors.NewInvalidInputError("request_body", err.Error())
		apperrors.ErrorHandler(c, appErr)
		return
	}

	c.Set("security_event", "customer_update_attempt")
	c.Set("target_resource", strconv.FormatUint(uint64(customerID), 10))

	customer, err := h.service.UpdateCustomer(customerID, &req)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	c.Set("security_event", "customer_updated")
	c.Set("resource_id", customer.ID)
	c.Set("resource_name", customer.Name)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    customer,
		"message": "客户更新成功",
	})
}

// DeleteCustomer handles customer deletion.
// @Summary Delete a customer
// @Description Soft deletes a customer organization
// @Tags customers
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param id path int true "Customer ID"
// @Success 200 {object} github_com_company_smartticket_internal_server.Response
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 404 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/customers/{id} [delete]
func (h *Handlers) DeleteCustomer(c *gin.Context) {
	customerID, err := h.parseCustomerID(c)
	if err != nil {
		return
	}

	c.Set("security_event", "customer_deletion_attempt")
	c.Set("target_resource", strconv.FormatUint(uint64(customerID), 10))

	if err := h.service.DeleteCustomer(customerID); err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	c.Set("security_event", "customer_deleted")
	c.Set("resource_id", customerID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "客户删除成功",
	})
}

// ListCustomerUsers handles listing a customer's contact users.
// @Summary List a customer's users
// @Description Retrieves the contact users belonging to a customer
// @Tags customers
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param id path int true "Customer ID"
// @Success 200 {object} github_com_company_smartticket_internal_server.Response
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 404 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/customers/{id}/users [get]
func (h *Handlers) ListCustomerUsers(c *gin.Context) {
	customerID, err := h.parseCustomerID(c)
	if err != nil {
		return
	}

	users, err := h.service.ListCustomerUsers(customerID)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    users,
	})
}
