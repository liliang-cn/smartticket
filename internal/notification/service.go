// Package notification provides in-app (站内) notifications for recipient users.
// Notifications are per-user, listable with an unread count, and respect
// customer isolation and internal-note privacy at the call sites that emit them.
package notification

import (
	"context"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/company/smartticket/internal/logger"
	"github.com/company/smartticket/internal/models"
)

// Service provides in-app notification business logic.
type Service struct {
	db *gorm.DB
}

// NewService creates a new notification service.
func NewService(db *gorm.DB) *Service { return &Service{db: db} }

// Notify creates one notification per recipient userID (deduped, skips zero IDs).
// It is best-effort: errors are logged, never returned to the caller's request
// path, so a notification failure can never break the originating operation.
func (s *Service) Notify(ctx context.Context, userIDs []uint, ntype, title, body, refType string, refID uint) {
	seen := make(map[uint]struct{}, len(userIDs))
	rows := make([]models.Notification, 0, len(userIDs))
	for _, uid := range userIDs {
		if uid == 0 {
			continue
		}
		if _, ok := seen[uid]; ok {
			continue
		}
		seen[uid] = struct{}{}
		rows = append(rows, models.Notification{
			UserID:  uid,
			Type:    ntype,
			Title:   title,
			Body:    body,
			RefType: refType,
			RefID:   refID,
		})
	}
	if len(rows) == 0 {
		return
	}
	if err := s.db.WithContext(ctx).Create(&rows).Error; err != nil {
		logger.Error("failed to create in-app notifications",
			zap.String("type", ntype), zap.Uint("ref_id", refID), zap.Error(err))
	}
}

// List returns a user's notifications newest-first, paginated, with the total
// count matching the same filter. When unreadOnly is true only unread rows are
// returned (and counted).
func (s *Service) List(userID uint, unreadOnly bool, page, pageSize int) ([]models.Notification, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	q := s.db.Model(&models.Notification{}).Where("user_id = ?", userID)
	if unreadOnly {
		q = q.Where("is_read = ?", false)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []models.Notification
	if err := q.Order("created_at DESC, id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&out).Error; err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// UnreadCount returns the number of unread notifications for a user.
func (s *Service) UnreadCount(userID uint) (int64, error) {
	var count int64
	err := s.db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count).Error
	return count, err
}

// MarkRead marks a single notification read, scoped to its owner. It is a no-op
// (no error) if the row does not exist or belongs to another user.
func (s *Service) MarkRead(userID, id uint) error {
	return s.db.Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_read", true).Error
}

// MarkAllRead marks every unread notification owned by the user as read.
func (s *Service) MarkAllRead(userID uint) error {
	return s.db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Update("is_read", true).Error
}
