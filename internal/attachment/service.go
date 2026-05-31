// Package attachment provides ticket file attachment storage: upload, list and
// download. Files are stored on local disk under <dataPath>/attachments/
// ticket-<ticketID>/ and metadata is persisted in the attachments table. All
// operations are customer-isolated: a customer-role actor may only touch
// attachments on tickets belonging to their own customer organization.
package attachment

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/company/smartticket/internal/authz"
	"github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Service provides attachment storage business logic.
type Service struct {
	db         *gorm.DB
	dataPath   string
	maxSize    int64
	allowedExt map[string]bool
}

// NewService creates a new attachment service. An empty dataPath defaults to
// "./data". allowedExt is a list of lowercase extensions including the leading
// dot (e.g. ".png"); an empty list allows any extension.
func NewService(db *gorm.DB, dataPath string, maxSize int64, allowedExt []string) *Service {
	if dataPath == "" {
		dataPath = "./data"
	}
	ext := make(map[string]bool, len(allowedExt))
	for _, e := range allowedExt {
		ext[strings.ToLower(e)] = true
	}
	return &Service{db: db, dataPath: dataPath, maxSize: maxSize, allowedExt: ext}
}

// loadTicketScoped loads a ticket by id, enforcing customer isolation: a
// customer actor may only load tickets belonging to their own customer.
func (s *Service) loadTicketScoped(actor authz.Actor, ticketID uint) (*models.Ticket, error) {
	var ticket models.Ticket
	q := s.db.Where("id = ?", ticketID)
	if actor.IsCustomer() {
		if actor.CustomerID == nil {
			return nil, errors.NewNotFoundError("ticket")
		}
		q = q.Where("customer_id = ?", *actor.CustomerID)
	}
	if err := q.First(&ticket).Error; err != nil {
		return nil, errors.NewNotFoundError("ticket")
	}
	return &ticket, nil
}

// Upload validates and stores an uploaded file for a ticket, returning the
// persisted attachment record. The reader is streamed to disk while a SHA-256
// hash is computed; the declared size is not trusted (a hard limit is enforced
// during the copy).
func (s *Service) Upload(actor authz.Actor, ticketID, userID uint, originalName, contentType string, r io.Reader, size int64) (*models.Attachment, error) {
	if _, err := s.loadTicketScoped(actor, ticketID); err != nil {
		return nil, err
	}

	if size <= 0 {
		return nil, errors.NewValidationError("file is empty")
	}
	if s.maxSize > 0 && size > s.maxSize {
		return nil, errors.NewValidationError("file exceeds the maximum size")
	}

	ext := strings.ToLower(filepath.Ext(originalName))
	if len(s.allowedExt) > 0 && !s.allowedExt[ext] {
		return nil, errors.NewValidationError("file type not allowed")
	}

	dir := filepath.Join(s.dataPath, "attachments", fmt.Sprintf("ticket-%d", ticketID))
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, errors.NewInternalError("failed to create storage directory", err)
	}

	stored := uuid.NewString() + ext
	fullPath := filepath.Join(dir, stored)

	f, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return nil, errors.NewInternalError("failed to create file", err)
	}

	hasher := sha256.New()
	// Enforce the size limit during the copy rather than trusting `size`.
	var reader io.Reader = r
	if s.maxSize > 0 {
		reader = io.LimitReader(r, s.maxSize+1)
	}
	written, copyErr := io.Copy(io.MultiWriter(f, hasher), reader)
	closeErr := f.Close()

	cleanup := func() { _ = os.Remove(fullPath) }

	if copyErr != nil {
		cleanup()
		return nil, errors.NewInternalError("failed to write file", copyErr)
	}
	if closeErr != nil {
		cleanup()
		return nil, errors.NewInternalError("failed to write file", closeErr)
	}
	if written == 0 {
		cleanup()
		return nil, errors.NewValidationError("file is empty")
	}
	if s.maxSize > 0 && written > s.maxSize {
		cleanup()
		return nil, errors.NewValidationError("file exceeds the maximum size")
	}

	att := &models.Attachment{
		TicketID:     ticketID,
		FileName:     stored,
		OriginalName: originalName,
		FilePath:     fullPath,
		FileSize:     written,
		ContentType:  contentType,
		Hash:         hex.EncodeToString(hasher.Sum(nil)),
	}
	if err := s.db.Create(att).Error; err != nil {
		cleanup()
		return nil, errors.NewDatabaseError("create attachment", err)
	}
	return att, nil
}

// List returns the attachments for a ticket ordered by creation time. Access is
// customer-isolated.
func (s *Service) List(actor authz.Actor, ticketID uint) ([]models.Attachment, error) {
	if _, err := s.loadTicketScoped(actor, ticketID); err != nil {
		return nil, err
	}
	var atts []models.Attachment
	if err := s.db.Where("ticket_id = ?", ticketID).Order("created_at asc").Find(&atts).Error; err != nil {
		return nil, errors.NewDatabaseError("list attachments", err)
	}
	return atts, nil
}

// Get loads a single attachment by id, enforcing that the actor can access the
// attachment's ticket. Used for download.
func (s *Service) Get(actor authz.Actor, attachmentID uint) (*models.Attachment, error) {
	var att models.Attachment
	if err := s.db.Where("id = ?", attachmentID).First(&att).Error; err != nil {
		return nil, errors.NewNotFoundError("attachment")
	}
	if _, err := s.loadTicketScoped(actor, att.TicketID); err != nil {
		// Hide existence from unauthorized customers.
		return nil, errors.NewNotFoundError("attachment")
	}
	return &att, nil
}
