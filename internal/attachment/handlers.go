package attachment

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/company/smartticket/internal/authz"
	"github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/models"
	"github.com/gin-gonic/gin"
)

// actorFromContext builds the authorization Actor from the values the auth
// middleware places in the gin context. Mirrors ticket.actorFromContext.
func actorFromContext(c *gin.Context) authz.Actor {
	a := authz.Actor{
		UserID: c.GetUint("user_id"),
		Role:   c.GetString("user_role"),
	}
	if v, ok := c.Get("user_customer_id"); ok {
		if cid, ok := v.(uint); ok {
			a.CustomerID = &cid
		}
	}
	return a
}

// attachmentView is the public JSON representation of an attachment. It
// deliberately omits FilePath so the on-disk location never leaks to clients.
type attachmentView struct {
	ID           uint      `json:"id"`
	TicketID     uint      `json:"ticket_id"`
	OriginalName string    `json:"original_name"`
	FileSize     int64     `json:"file_size"`
	ContentType  string    `json:"content_type"`
	CreatedAt    time.Time `json:"created_at"`
}

func toView(a *models.Attachment) attachmentView {
	return attachmentView{
		ID:           a.ID,
		TicketID:     a.TicketID,
		OriginalName: a.OriginalName,
		FileSize:     a.FileSize,
		ContentType:  a.ContentType,
		CreatedAt:    a.CreatedAt,
	}
}

// Handlers provides attachment HTTP handlers.
type Handlers struct {
	service *Service
}

// NewHandlers creates new attachment handlers.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// Upload handles a multipart file upload for a ticket.
// @Summary Upload a ticket attachment
// @Description Uploads a file attachment to a ticket (multipart form field "file").
// @Tags tickets
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param id path int true "Ticket ID"
// @Param file formData file true "File to upload"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 404 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/tickets/{id}/attachments [post]
func (h *Handlers) Upload(c *gin.Context) {
	ticketID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		errors.ErrorHandler(c, errors.NewInvalidInputError("ticket_id", c.Param("id")))
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		errors.ErrorHandler(c, errors.NewValidationError("missing form file field \"file\""))
		return
	}
	defer func() { _ = file.Close() }()

	contentType := header.Header.Get("Content-Type")

	att, err := h.service.Upload(
		actorFromContext(c),
		uint(ticketID),
		c.GetUint("user_id"),
		header.Filename,
		contentType,
		file,
		header.Size,
	)
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": toView(att)})
}

// List returns the attachments for a ticket.
// @Summary List ticket attachments
// @Description Lists attachments on a ticket. Customer-isolated.
// @Tags tickets
// @Produce json
// @Security BearerAuth
// @Param id path int true "Ticket ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/tickets/{id}/attachments [get]
func (h *Handlers) List(c *gin.Context) {
	ticketID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		errors.ErrorHandler(c, errors.NewInvalidInputError("ticket_id", c.Param("id")))
		return
	}

	atts, err := h.service.List(actorFromContext(c), uint(ticketID))
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	views := make([]attachmentView, 0, len(atts))
	for i := range atts {
		views = append(views, toView(&atts[i]))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": views})
}

// Download streams an attachment's file to the client.
// @Summary Download an attachment
// @Description Downloads an attachment's file. Customer-isolated.
// @Tags tickets
// @Produce application/octet-stream
// @Security BearerAuth
// @Param id path int true "Attachment ID"
// @Success 200 {file} binary
// @Failure 404 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/attachments/{id}/download [get]
func (h *Handlers) Download(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		errors.ErrorHandler(c, errors.NewInvalidInputError("attachment_id", c.Param("id")))
		return
	}

	att, err := h.service.Get(actorFromContext(c), uint(id))
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	if _, statErr := os.Stat(att.FilePath); statErr != nil {
		errors.ErrorHandler(c, errors.NewNotFoundError("attachment"))
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", att.OriginalName))
	if att.ContentType != "" {
		c.Header("Content-Type", att.ContentType)
	}
	c.File(att.FilePath)
}
