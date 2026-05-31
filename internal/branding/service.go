// Package branding manages the org-wide white-label configuration for this
// single-tenant deployment: product/workspace names, the accent color and an
// optional uploaded logo. The configuration is a singleton row (ID 1) that is
// created lazily with sensible defaults on first read.
package branding

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/models"
	"gorm.io/gorm"
)

// singletonID is the fixed primary key of the one-and-only branding row.
const singletonID = 1

// Defaults mirror the values the UI shipped with before branding became
// configurable, so an un-customized deployment looks unchanged.
var defaults = models.Branding{
	AppName:       "SmartTicket",
	AppSubtitle:   "console",
	WorkspaceName: "LINBIT workspace",
	PrimaryColor:  "#f59e0b",
	LoginTagline:  "Every ticket, SLA and customer — under one calm, fast surface.",
	LoginSubtext:  "Self-hosted. Single-tenant. Your data, your rules.",
}

// allowedLogoExt is the set of image extensions accepted for the logo.
var allowedLogoExt = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".svg": true, ".webp": true, ".gif": true,
}

// maxLogoSize caps the uploaded logo at 2 MiB — a logo never needs more.
const maxLogoSize = 2 << 20

// Service provides branding business logic.
type Service struct {
	db       *gorm.DB
	dataPath string
}

// NewService creates a branding service. An empty dataPath defaults to "./data".
func NewService(db *gorm.DB, dataPath string) *Service {
	if dataPath == "" {
		dataPath = "./data"
	}
	return &Service{db: db, dataPath: dataPath}
}

// Get returns the branding singleton, creating it with defaults on first call.
func (s *Service) Get() (*models.Branding, error) {
	var b models.Branding
	err := s.db.First(&b, singletonID).Error
	if err == gorm.ErrRecordNotFound {
		b = defaults
		b.ID = singletonID
		if cerr := s.db.Create(&b).Error; cerr != nil {
			return nil, errors.NewDatabaseError("create branding", cerr)
		}
		return &b, nil
	}
	if err != nil {
		return nil, errors.NewDatabaseError("load branding", err)
	}
	return &b, nil
}

// UpdateRequest carries the editable branding fields. All fields are optional;
// a nil pointer leaves the stored value unchanged.
type UpdateRequest struct {
	AppName       *string `json:"app_name"`
	AppSubtitle   *string `json:"app_subtitle"`
	WorkspaceName *string `json:"workspace_name"`
	PrimaryColor  *string `json:"primary_color"`
	LoginTagline  *string `json:"login_tagline"`
	LoginSubtext  *string `json:"login_subtext"`
}

// Update applies the provided fields to the branding singleton.
func (s *Service) Update(req *UpdateRequest) (*models.Branding, error) {
	b, err := s.Get()
	if err != nil {
		return nil, err
	}

	if req.AppName != nil {
		b.AppName = strings.TrimSpace(*req.AppName)
	}
	if req.AppSubtitle != nil {
		b.AppSubtitle = strings.TrimSpace(*req.AppSubtitle)
	}
	if req.WorkspaceName != nil {
		b.WorkspaceName = strings.TrimSpace(*req.WorkspaceName)
	}
	if req.PrimaryColor != nil {
		c := strings.TrimSpace(*req.PrimaryColor)
		if c != "" && !isHexColor(c) {
			return nil, errors.NewValidationError("primary_color must be a hex color like #f59e0b")
		}
		b.PrimaryColor = c
	}
	if req.LoginTagline != nil {
		b.LoginTagline = strings.TrimSpace(*req.LoginTagline)
	}
	if req.LoginSubtext != nil {
		b.LoginSubtext = strings.TrimSpace(*req.LoginSubtext)
	}

	if err := s.db.Save(b).Error; err != nil {
		return nil, errors.NewDatabaseError("update branding", err)
	}
	return b, nil
}

// SaveLogo validates and stores an uploaded logo image, replacing any previous
// one, and records its location on the branding singleton.
func (s *Service) SaveLogo(originalName, contentType string, r io.Reader, size int64) (*models.Branding, error) {
	if size <= 0 {
		return nil, errors.NewValidationError("file is empty")
	}
	if size > maxLogoSize {
		return nil, errors.NewValidationError("logo exceeds the 2 MB maximum")
	}
	ext := strings.ToLower(filepath.Ext(originalName))
	if !allowedLogoExt[ext] {
		return nil, errors.NewValidationError("logo must be a PNG, JPG, SVG, WEBP or GIF image")
	}

	b, err := s.Get()
	if err != nil {
		return nil, err
	}

	dir := filepath.Join(s.dataPath, "branding")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, errors.NewInternalError("failed to create storage directory", err)
	}
	fullPath := filepath.Join(dir, "logo"+ext)

	f, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return nil, errors.NewInternalError("failed to create file", err)
	}
	written, copyErr := io.Copy(f, io.LimitReader(r, maxLogoSize+1))
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(fullPath)
		return nil, errors.NewInternalError("failed to write file", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(fullPath)
		return nil, errors.NewInternalError("failed to write file", closeErr)
	}
	if written > maxLogoSize {
		_ = os.Remove(fullPath)
		return nil, errors.NewValidationError("logo exceeds the 2 MB maximum")
	}

	// Remove a previous logo with a different extension to avoid orphans.
	if b.LogoPath != "" && b.LogoPath != fullPath {
		_ = os.Remove(b.LogoPath)
	}

	b.LogoPath = fullPath
	b.LogoExt = ext
	if err := s.db.Save(b).Error; err != nil {
		_ = os.Remove(fullPath)
		return nil, errors.NewDatabaseError("update branding", err)
	}
	return b, nil
}

// DeleteLogo removes the uploaded logo (file + record), reverting to the glyph.
func (s *Service) DeleteLogo() (*models.Branding, error) {
	b, err := s.Get()
	if err != nil {
		return nil, err
	}
	if b.LogoPath != "" {
		_ = os.Remove(b.LogoPath)
	}
	b.LogoPath = ""
	b.LogoExt = ""
	if err := s.db.Save(b).Error; err != nil {
		return nil, errors.NewDatabaseError("update branding", err)
	}
	return b, nil
}

// LogoFile returns the on-disk path and content type of the uploaded logo, or
// an error if none is set.
func (s *Service) LogoFile() (path, contentType string, err error) {
	b, err := s.Get()
	if err != nil {
		return "", "", err
	}
	if b.LogoPath == "" {
		return "", "", errors.NewNotFoundError("logo")
	}
	if _, statErr := os.Stat(b.LogoPath); statErr != nil {
		return "", "", errors.NewNotFoundError("logo")
	}
	return b.LogoPath, contentTypeForExt(b.LogoExt), nil
}

// isHexColor reports whether s looks like #rgb / #rrggbb / #rrggbbaa.
func isHexColor(s string) bool {
	if len(s) < 4 || s[0] != '#' {
		return false
	}
	hex := s[1:]
	if l := len(hex); l != 3 && l != 6 && l != 8 {
		return false
	}
	for _, r := range hex {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

// contentTypeForExt maps a logo extension to its image MIME type.
func contentTypeForExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".svg":
		return "image/svg+xml"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		return "application/octet-stream"
	}
}
