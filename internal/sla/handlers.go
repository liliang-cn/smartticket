package sla

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/company/smartticket/internal/auth"
	apperrors "github.com/company/smartticket/internal/errors"
	"github.com/gin-gonic/gin"
)

// Handlers provides HTTP handlers for SLA management
type Handlers struct {
	service    *Service
	calculator *Calculator
}

// NewHandlers creates a new SLA handlers instance
func NewHandlers(service *Service, calculator *Calculator) *Handlers {
	return &Handlers{
		service:    service,
		calculator: calculator,
	}
}

// parseTemplateID extracts and validates SLA template ID from request parameters
func (h *Handlers) parseTemplateID(c *gin.Context) (uint, error) {
	templateID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErr := apperrors.NewInvalidInputError("id", "Invalid SLA template ID")
		apperrors.ErrorHandler(c, appErr)
		return 0, err
	}
	return uint(templateID), nil
}

// parseRuleID extracts and validates SLA rule ID from request parameters
func (h *Handlers) parseRuleID(c *gin.Context) (uint, error) {
	ruleID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErr := apperrors.NewInvalidInputError("id", "Invalid SLA rule ID")
		apperrors.ErrorHandler(c, appErr)
		return 0, err
	}
	return uint(ruleID), nil
}

// getUserInfo extracts user info from context with error handling
func (h *Handlers) getUserInfo(c *gin.Context) (*auth.UserInfo, error) {
	userInfo, exists := c.Get("user")
	if !exists {
		appErr := apperrors.NewUnauthorizedError("User not authenticated")
		apperrors.ErrorHandler(c, appErr)
		return nil, errors.New("user not authenticated")
	}
	return userInfo.(*auth.UserInfo), nil
}

// parseAndValidateUser extracts user info from context with unified error handling
func (h *Handlers) parseAndValidateUser(c *gin.Context) (*auth.UserInfo, error) {
	userInfo, exists := c.Get("user")
	if !exists {
		appErr := apperrors.NewUnauthorizedError("User not authenticated")
		apperrors.ErrorHandler(c, appErr)
		return nil, errors.New("user not authenticated")
	}
	return userInfo.(*auth.UserInfo), nil
}

// sendPaginatedResponse sends a standardized paginated response
func (h *Handlers) sendPaginatedResponse(c *gin.Context, data interface{}, total int64, page, pageSize int) {
	totalPages := (total + int64(pageSize) - 1) / int64(pageSize)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
		"meta": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// sendSuccessResponse sends a standardized success response
func (h *Handlers) sendSuccessResponse(c *gin.Context, data interface{}, message string) {
	response := gin.H{
		"success": true,
		"data":    data,
	}
	if message != "" {
		response["message"] = message
	}
	c.JSON(http.StatusOK, response)
}

// sendCreatedResponse sends a standardized created response
func (h *Handlers) sendCreatedResponse(c *gin.Context, data interface{}, message string) {
	response := gin.H{
		"success": true,
		"data":    data,
	}
	if message != "" {
		response["message"] = message
	}
	c.JSON(http.StatusCreated, response)
}

// logSecurityEvent logs a security event with resource information
func (h *Handlers) logSecurityEvent(c *gin.Context, event, resource string, resourceID interface{}) {
	c.Set("security_event", event)
	c.Set("target_resource", resource)
	if resourceID != nil {
		c.Set("resource_id", resourceID)
	}
}

// logSecurityEventWithName logs a security event with resource information and name
func (h *Handlers) logSecurityEventWithName(c *gin.Context, event, resourceName string) {
	c.Set("security_event", event)
	c.Set("resource_name", resourceName)
}

// SLA Template Handlers

// CreateSLATemplate handles SLA template creation
func (h *Handlers) CreateSLATemplate(c *gin.Context) {
	var req CreateSLATemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := apperrors.NewInvalidInputError("request_body", err.Error())
		apperrors.ErrorHandler(c, appErr)
		return
	}

	user, err := h.parseAndValidateUser(c)
	if err != nil {
		return
	}

	h.logSecurityEventWithName(c, "sla_template_creation_attempt", req.Name)

	template, err := h.service.CreateSLATemplate(user.TenantID, &req)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	// Log successful SLA template creation
	h.logSecurityEvent(c, "sla_template_created", "sla_template", template.ID)
	c.Set("resource_name", template.Name)

	h.sendCreatedResponse(c, template, "SLA template created successfully")
}

// ListSLATemplates handles SLA template listing
func (h *Handlers) ListSLATemplates(c *gin.Context) {
	var req ListSLATemplatesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		appErr := apperrors.NewInvalidInputError("query_params", err.Error())
		apperrors.ErrorHandler(c, appErr)
		return
	}

	user, err := h.parseAndValidateUser(c)
	if err != nil {
		return
	}

	templates, total, err := h.service.ListSLATemplates(user.TenantID, &req)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	h.sendPaginatedResponse(c, templates, total, req.Page, req.PageSize)
}

// GetSLATemplate handles getting a single SLA template
func (h *Handlers) GetSLATemplate(c *gin.Context) {
	templateID, err := h.parseTemplateID(c)
	if err != nil {
		return
	}

	user, err := h.parseAndValidateUser(c)
	if err != nil {
		return
	}

	template, err := h.service.GetSLATemplate(user.TenantID, templateID)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	h.sendSuccessResponse(c, template, "")
}

// UpdateSLATemplate handles SLA template update
func (h *Handlers) UpdateSLATemplate(c *gin.Context) {
	templateID, err := h.parseTemplateID(c)
	if err != nil {
		return
	}

	var req UpdateSLATemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := apperrors.NewInvalidInputError("request_body", err.Error())
		apperrors.ErrorHandler(c, appErr)
		return
	}

	user, err := h.parseAndValidateUser(c)
	if err != nil {
		return
	}

	h.logSecurityEvent(c, "sla_template_update_attempt", "sla_template", templateID)

	template, err := h.service.UpdateSLATemplate(user.TenantID, templateID, &req)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	// Log successful SLA template update
	h.logSecurityEvent(c, "sla_template_updated", "sla_template", template.ID)
	c.Set("resource_name", template.Name)

	h.sendSuccessResponse(c, template, "SLA template updated successfully")
}

// DeleteSLATemplate handles SLA template deletion
func (h *Handlers) DeleteSLATemplate(c *gin.Context) {
	templateID, err := h.parseTemplateID(c)
	if err != nil {
		return
	}

	user, err := h.parseAndValidateUser(c)
	if err != nil {
		return
	}

	h.logSecurityEvent(c, "sla_template_deletion_attempt", "sla_template", templateID)

	err = h.service.DeleteSLATemplate(user.TenantID, templateID)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	// Log successful SLA template deletion
	h.logSecurityEvent(c, "sla_template_deleted", "sla_template", templateID)

	h.sendSuccessResponse(c, nil, "SLA template deleted successfully")
}

// SLA Rule Handlers

// CreateSLARule handles SLA rule creation
func (h *Handlers) CreateSLARule(c *gin.Context) {
	var req CreateSLARuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := apperrors.NewInvalidInputError("request_body", err.Error())
		apperrors.ErrorHandler(c, appErr)
		return
	}

	user, err := h.parseAndValidateUser(c)
	if err != nil {
		return
	}

	h.logSecurityEvent(c, "sla_rule_creation_attempt", "sla_rule", nil)

	rule, err := h.service.CreateSLARule(user.TenantID, &req)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	// Log successful SLA rule creation
	h.logSecurityEvent(c, "sla_rule_created", "sla_rule", rule.ID)

	h.sendCreatedResponse(c, rule, "SLA rule created successfully")
}

// ListSLARules handles SLA rule listing
func (h *Handlers) ListSLARules(c *gin.Context) {
	var req ListSLARulesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		appErr := apperrors.NewInvalidInputError("query_params", err.Error())
		apperrors.ErrorHandler(c, appErr)
		return
	}

	user, err := h.parseAndValidateUser(c)
	if err != nil {
		return
	}

	rules, total, err := h.service.ListSLARules(user.TenantID, &req)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	h.sendPaginatedResponse(c, rules, total, req.Page, req.PageSize)
}

// GetSLARule handles getting a single SLA rule
func (h *Handlers) GetSLARule(c *gin.Context) {
	ruleID, err := h.parseRuleID(c)
	if err != nil {
		return
	}

	user, err := h.parseAndValidateUser(c)
	if err != nil {
		return
	}

	rule, err := h.service.GetSLARule(user.TenantID, ruleID)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	h.sendSuccessResponse(c, rule, "")
}

// UpdateSLARule handles SLA rule update
func (h *Handlers) UpdateSLARule(c *gin.Context) {
	ruleID, err := h.parseRuleID(c)
	if err != nil {
		return
	}

	var req UpdateSLARuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := apperrors.NewInvalidInputError("request_body", err.Error())
		apperrors.ErrorHandler(c, appErr)
		return
	}

	user, err := h.parseAndValidateUser(c)
	if err != nil {
		return
	}

	h.logSecurityEvent(c, "sla_rule_update_attempt", "sla_rule", ruleID)

	rule, err := h.service.UpdateSLARule(user.TenantID, ruleID, &req)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	// Log successful SLA rule update
	h.logSecurityEvent(c, "sla_rule_updated", "sla_rule", rule.ID)

	h.sendSuccessResponse(c, rule, "SLA rule updated successfully")
}

// DeleteSLARule handles SLA rule deletion
func (h *Handlers) DeleteSLARule(c *gin.Context) {
	ruleID, err := h.parseRuleID(c)
	if err != nil {
		return
	}

	user, err := h.parseAndValidateUser(c)
	if err != nil {
		return
	}

	h.logSecurityEvent(c, "sla_rule_deletion_attempt", "sla_rule", ruleID)

	err = h.service.DeleteSLARule(user.TenantID, ruleID)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	// Log successful SLA rule deletion
	h.logSecurityEvent(c, "sla_rule_deleted", "sla_rule", ruleID)

	h.sendSuccessResponse(c, nil, "SLA rule deleted successfully")
}

// ActivateSLARule handles SLA rule activation
func (h *Handlers) ActivateSLARule(c *gin.Context) {
	ruleID, err := h.parseRuleID(c)
	if err != nil {
		return
	}

	user, err := h.parseAndValidateUser(c)
	if err != nil {
		return
	}

	h.logSecurityEvent(c, "sla_rule_activation_attempt", "sla_rule", ruleID)

	err = h.service.ActivateSLARule(user.TenantID, ruleID)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	// Log successful SLA rule activation
	h.logSecurityEvent(c, "sla_rule_activated", "sla_rule", ruleID)

	h.sendSuccessResponse(c, nil, "SLA rule activated successfully")
}

// DeactivateSLARule handles SLA rule deactivation
func (h *Handlers) DeactivateSLARule(c *gin.Context) {
	ruleID, err := h.parseRuleID(c)
	if err != nil {
		return
	}

	user, err := h.parseAndValidateUser(c)
	if err != nil {
		return
	}

	h.logSecurityEvent(c, "sla_rule_deactivation_attempt", "sla_rule", ruleID)

	err = h.service.DeactivateSLARule(user.TenantID, ruleID)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	// Log successful SLA rule deactivation
	h.logSecurityEvent(c, "sla_rule_deactivated", "sla_rule", ruleID)

	h.sendSuccessResponse(c, nil, "SLA rule deactivated successfully")
}
