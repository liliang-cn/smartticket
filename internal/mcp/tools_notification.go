package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/company/smartticket/internal/models"
)

// notificationView is the schema-safe MCP view of an in-app notification.
type notificationView struct {
	ID        uint      `json:"id" jsonschema:"notification ID"`
	Type      string    `json:"type" jsonschema:"event type, e.g. ticket_reply, ticket_assigned"`
	Title     string    `json:"title" jsonschema:"short title"`
	Body      string    `json:"body,omitempty" jsonschema:"notification body"`
	RefType   string    `json:"ref_type,omitempty" jsonschema:"referenced entity type, e.g. ticket"`
	RefID     uint      `json:"ref_id,omitempty" jsonschema:"referenced entity ID"`
	IsRead    bool      `json:"is_read" jsonschema:"whether the notification has been read"`
	CreatedAt time.Time `json:"created_at" jsonschema:"when the notification was created"`
}

func notificationViewFrom(n *models.Notification) notificationView {
	return notificationView{
		ID: n.ID, Type: n.Type, Title: n.Title, Body: n.Body,
		RefType: n.RefType, RefID: n.RefID, IsRead: n.IsRead, CreatedAt: n.CreatedAt,
	}
}

// registerNotificationTools registers the notification-domain MCP tools. Every
// tool operates on the calling session's OWN notifications; there is no
// cross-user access, so no specific permission code is required beyond an
// authenticated session (which the closures verify).
func registerNotificationTools(s *mcp.Server, b Backend) {
	registerTool(s, "notification_list",
		"List the authenticated user's in-app notifications, newest first.",
		"",
		func(ctx context.Context, in notificationListInput) (notificationListOutput, string, error) {
			return notificationList(ctx, b, in)
		})

	registerTool(s, "notification_unread_count",
		"Return the number of unread notifications for the authenticated user.",
		"",
		func(ctx context.Context, _ struct{}) (notificationCountOutput, string, error) {
			return notificationUnreadCount(ctx, b)
		})

	registerTool(s, "notification_mark_read",
		"Mark one of the authenticated user's notifications as read, by its ID.",
		"",
		func(ctx context.Context, in notificationIDInput) (notificationMarkOutput, string, error) {
			return notificationMarkRead(ctx, b, in)
		})

	registerTool(s, "notification_mark_all_read",
		"Mark all of the authenticated user's notifications as read.",
		"",
		func(ctx context.Context, _ struct{}) (notificationMarkOutput, string, error) {
			return notificationMarkAllRead(ctx, b)
		})
}

// ---- schemas ----

type notificationListInput struct {
	UnreadOnly bool `json:"unread_only,omitempty" jsonschema:"only return unread notifications"`
	Page       int  `json:"page,omitempty" jsonschema:"page number, 1-based (default 1)"`
	PageSize   int  `json:"page_size,omitempty" jsonschema:"items per page, 1-100 (default 20)"`
}

type notificationListOutput struct {
	Notifications []notificationView `json:"notifications,omitempty" jsonschema:"the page of notifications"`
	Total         int64              `json:"total" jsonschema:"total matching notifications"`
	Page          int                `json:"page" jsonschema:"page number returned"`
	PageSize      int                `json:"page_size" jsonschema:"page size used"`
}

type notificationCountOutput struct {
	Unread int64 `json:"unread" jsonschema:"number of unread notifications"`
}

type notificationIDInput struct {
	ID uint `json:"id" jsonschema:"notification ID to mark as read"`
}

type notificationMarkOutput struct {
	OK bool `json:"ok" jsonschema:"true when the operation succeeded"`
}

// ---- closures ----

func notificationList(ctx context.Context, b Backend, in notificationListInput) (notificationListOutput, string, error) {
	session, ok := SessionFromContext(ctx)
	if !ok || session == nil {
		return notificationListOutput{}, "", ErrUnauthenticated
	}
	page, pageSize := in.Page, in.PageSize
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 20
	}
	items, total, err := b.ListNotifications(session.UserID, in.UnreadOnly, page, pageSize)
	if err != nil {
		return notificationListOutput{}, "", err
	}
	views := make([]notificationView, 0, len(items))
	for i := range items {
		views = append(views, notificationViewFrom(&items[i]))
	}
	out := notificationListOutput{Notifications: views, Total: total, Page: page, PageSize: pageSize}
	return out, fmt.Sprintf("Returned %d of %d notification(s).", len(items), total), nil
}

func notificationUnreadCount(ctx context.Context, b Backend) (notificationCountOutput, string, error) {
	session, ok := SessionFromContext(ctx)
	if !ok || session == nil {
		return notificationCountOutput{}, "", ErrUnauthenticated
	}
	n, err := b.UnreadNotificationCount(session.UserID)
	if err != nil {
		return notificationCountOutput{}, "", err
	}
	return notificationCountOutput{Unread: n}, fmt.Sprintf("%d unread notification(s).", n), nil
}

func notificationMarkRead(ctx context.Context, b Backend, in notificationIDInput) (notificationMarkOutput, string, error) {
	session, ok := SessionFromContext(ctx)
	if !ok || session == nil {
		return notificationMarkOutput{}, "", ErrUnauthenticated
	}
	if err := b.MarkNotificationRead(session.UserID, in.ID); err != nil {
		return notificationMarkOutput{}, "", err
	}
	return notificationMarkOutput{OK: true}, fmt.Sprintf("Marked notification #%d as read.", in.ID), nil
}

func notificationMarkAllRead(ctx context.Context, b Backend) (notificationMarkOutput, string, error) {
	session, ok := SessionFromContext(ctx)
	if !ok || session == nil {
		return notificationMarkOutput{}, "", ErrUnauthenticated
	}
	if err := b.MarkAllNotificationsRead(session.UserID); err != nil {
		return notificationMarkOutput{}, "", err
	}
	return notificationMarkOutput{OK: true}, "Marked all notifications as read.", nil
}
