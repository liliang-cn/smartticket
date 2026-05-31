package branding

import (
	"fmt"
	"net/http"

	"github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/models"
	"github.com/gin-gonic/gin"
)

// brandingView is the public JSON representation of the branding config. It
// omits the on-disk logo path and instead exposes a cache-busting logo URL plus
// a has_logo flag for the client.
type brandingView struct {
	AppName       string `json:"app_name"`
	AppSubtitle   string `json:"app_subtitle"`
	WorkspaceName string `json:"workspace_name"`
	PrimaryColor  string `json:"primary_color"`
	LoginTagline  string `json:"login_tagline"`
	LoginSubtext  string `json:"login_subtext"`
	HasLogo       bool   `json:"has_logo"`
	LogoURL       string `json:"logo_url"`
	UpdatedAt     int64  `json:"updated_at"`
}

func toView(b *models.Branding) brandingView {
	v := brandingView{
		AppName:       b.AppName,
		AppSubtitle:   b.AppSubtitle,
		WorkspaceName: b.WorkspaceName,
		PrimaryColor:  b.PrimaryColor,
		LoginTagline:  b.LoginTagline,
		LoginSubtext:  b.LoginSubtext,
		HasLogo:       b.LogoPath != "",
		UpdatedAt:     b.UpdatedAt.Unix(),
	}
	if v.HasLogo {
		// The ?v= query busts the browser cache whenever the logo changes.
		v.LogoURL = fmt.Sprintf("/api/v1/settings/branding/logo?v=%d", b.UpdatedAt.Unix())
	}
	return v
}

// Handlers provides branding HTTP handlers.
type Handlers struct {
	service *Service
}

// NewHandlers creates branding handlers.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// Get returns the branding configuration. Public — the login page and app shell
// render it before authentication.
// @Summary Get branding configuration
// @Tags settings
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/settings/branding [get]
func (h *Handlers) Get(c *gin.Context) {
	b, err := h.service.Get()
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": toView(b)})
}

// Update patches the branding configuration. Admin-only.
// @Summary Update branding configuration
// @Tags settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body branding.UpdateRequest true "Branding fields"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/settings/branding [put]
func (h *Handlers) Update(c *gin.Context) {
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ErrorHandler(c, errors.NewInvalidInputError("request_body", err.Error()))
		return
	}
	b, err := h.service.Update(&req)
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}
	c.Set("security_event", "branding_updated")
	c.JSON(http.StatusOK, gin.H{"success": true, "data": toView(b)})
}

// UploadLogo stores an uploaded logo image (multipart field "file"). Admin-only.
// @Summary Upload branding logo
// @Tags settings
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "Logo image"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/settings/branding/logo [post]
func (h *Handlers) UploadLogo(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		errors.ErrorHandler(c, errors.NewValidationError("missing form file field \"file\""))
		return
	}
	defer func() { _ = file.Close() }()

	b, err := h.service.SaveLogo(header.Filename, header.Header.Get("Content-Type"), file, header.Size)
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}
	c.Set("security_event", "branding_logo_updated")
	c.JSON(http.StatusOK, gin.H{"success": true, "data": toView(b)})
}

// DeleteLogo removes the uploaded logo. Admin-only.
// @Summary Delete branding logo
// @Tags settings
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/settings/branding/logo [delete]
func (h *Handlers) DeleteLogo(c *gin.Context) {
	b, err := h.service.DeleteLogo()
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}
	c.Set("security_event", "branding_logo_removed")
	c.JSON(http.StatusOK, gin.H{"success": true, "data": toView(b)})
}

// ServeLogo streams the uploaded logo image. Public.
// @Summary Get branding logo image
// @Tags settings
// @Produce image/png
// @Success 200 {file} binary
// @Router /api/v1/settings/branding/logo [get]
func (h *Handlers) ServeLogo(c *gin.Context) {
	path, contentType, err := h.service.LogoFile()
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}
	c.Header("Cache-Control", "public, max-age=300")
	c.Header("Content-Type", contentType)
	c.File(path)
}
