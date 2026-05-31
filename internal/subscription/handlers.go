package subscription

import (
	"net/http"
	"strconv"

	apperrors "github.com/company/smartticket/internal/errors"
	"github.com/gin-gonic/gin"
)

// Handlers provides HTTP handlers for subscription management.
type Handlers struct {
	service *Service
}

// NewHandlers creates a new subscription handlers instance.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

func (h *Handlers) parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apperrors.ErrorHandler(c, apperrors.NewInvalidInputError("id", "无效的订阅ID"))
		return 0, false
	}
	return uint(id), true
}

// CreateSubscription handles subscription creation.
func (h *Handlers) CreateSubscription(c *gin.Context) {
	var req CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperrors.ErrorHandler(c, apperrors.NewInvalidInputError("request_body", err.Error()))
		return
	}

	c.Set("security_event", "subscription_creation_attempt")

	subscription, err := h.service.Create(&req)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	c.Set("security_event", "subscription_created")
	c.Set("resource_id", subscription.ID)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    subscription,
		"message": "订阅创建成功",
	})
}

// ListSubscriptions handles subscription listing.
func (h *Handlers) ListSubscriptions(c *gin.Context) {
	var req ListSubscriptionsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		apperrors.ErrorHandler(c, apperrors.NewInvalidInputError("query_params", err.Error()))
		return
	}

	subscriptions, total, err := h.service.List(&req)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	totalPages := (total + int64(req.PageSize) - 1) / int64(req.PageSize)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    subscriptions,
		"meta": gin.H{
			"page":        req.Page,
			"page_size":   req.PageSize,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// GetSubscription handles getting a single subscription.
func (h *Handlers) GetSubscription(c *gin.Context) {
	id, ok := h.parseID(c)
	if !ok {
		return
	}

	subscription, err := h.service.Get(id)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    subscription,
	})
}

// UpdateSubscription handles subscription updates.
func (h *Handlers) UpdateSubscription(c *gin.Context) {
	id, ok := h.parseID(c)
	if !ok {
		return
	}

	var req UpdateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperrors.ErrorHandler(c, apperrors.NewInvalidInputError("request_body", err.Error()))
		return
	}

	c.Set("security_event", "subscription_update_attempt")
	c.Set("target_resource", strconv.FormatUint(uint64(id), 10))

	subscription, err := h.service.Update(id, &req)
	if err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	c.Set("security_event", "subscription_updated")
	c.Set("resource_id", subscription.ID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    subscription,
		"message": "订阅更新成功",
	})
}

// DeleteSubscription handles subscription deletion.
func (h *Handlers) DeleteSubscription(c *gin.Context) {
	id, ok := h.parseID(c)
	if !ok {
		return
	}

	c.Set("security_event", "subscription_deletion_attempt")
	c.Set("target_resource", strconv.FormatUint(uint64(id), 10))

	if err := h.service.Delete(id); err != nil {
		apperrors.ErrorHandler(c, err)
		return
	}

	c.Set("security_event", "subscription_deleted")
	c.Set("resource_id", id)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "订阅删除成功",
	})
}
