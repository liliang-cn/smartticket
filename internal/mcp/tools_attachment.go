package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/company/smartticket/internal/models"
)

// attachmentView is the schema-safe MCP view of a ticket attachment. It exposes
// metadata only; the binary content is not transferred over MCP and the on-disk
// path is never leaked.
type attachmentView struct {
	ID           uint      `json:"id" jsonschema:"attachment ID"`
	TicketID     uint      `json:"ticket_id" jsonschema:"owning ticket ID"`
	OriginalName string    `json:"original_name" jsonschema:"original uploaded file name"`
	FileSize     int64     `json:"file_size" jsonschema:"file size in bytes"`
	ContentType  string    `json:"content_type,omitempty" jsonschema:"MIME content type"`
	Hash         string    `json:"hash,omitempty" jsonschema:"SHA-256 hash of the file content"`
	CreatedAt    time.Time `json:"created_at" jsonschema:"when the attachment was uploaded"`
}

func attachmentViewFrom(a *models.Attachment) attachmentView {
	return attachmentView{
		ID: a.ID, TicketID: a.TicketID, OriginalName: a.OriginalName,
		FileSize: a.FileSize, ContentType: a.ContentType, Hash: a.Hash, CreatedAt: a.CreatedAt,
	}
}

// registerAttachmentTools registers the attachment-domain MCP tools (metadata
// only; uploading/downloading binary content is intentionally not exposed over
// MCP). Access is customer-isolated via the session actor and gated by
// ticket:read.
func registerAttachmentTools(s *mcp.Server, b Backend) {
	registerTool(s, "attachment_list",
		"List the file attachments on a ticket (metadata only). Customer-isolated.",
		"ticket:read",
		func(ctx context.Context, in attachmentListInput) (attachmentListOutput, string, error) {
			return attachmentList(ctx, b, in)
		})

	registerTool(s, "attachment_get",
		"Get a single attachment's metadata by its numeric ID. Customer-isolated.",
		"ticket:read",
		func(ctx context.Context, in attachmentGetInput) (attachmentView, string, error) {
			return attachmentGet(ctx, b, in)
		})
}

// ---- schemas ----

type attachmentListInput struct {
	TicketID uint `json:"ticket_id" jsonschema:"ID of the ticket whose attachments to list"`
}

type attachmentListOutput struct {
	Attachments []attachmentView `json:"attachments,omitempty" jsonschema:"the ticket's attachments"`
	Total       int              `json:"total" jsonschema:"number of attachments"`
}

type attachmentGetInput struct {
	ID uint `json:"id" jsonschema:"numeric ID of the attachment"`
}

// ---- closures ----

func attachmentList(ctx context.Context, b Backend, in attachmentListInput) (attachmentListOutput, string, error) {
	items, err := b.ListAttachments(sessionActor(ctx), in.TicketID)
	if err != nil {
		return attachmentListOutput{}, "", err
	}
	views := make([]attachmentView, 0, len(items))
	for i := range items {
		views = append(views, attachmentViewFrom(&items[i]))
	}
	return attachmentListOutput{Attachments: views, Total: len(views)},
		fmt.Sprintf("Ticket #%d has %d attachment(s).", in.TicketID, len(views)), nil
}

func attachmentGet(ctx context.Context, b Backend, in attachmentGetInput) (attachmentView, string, error) {
	a, err := b.GetAttachment(sessionActor(ctx), in.ID)
	if err != nil {
		return attachmentView{}, "", err
	}
	return attachmentViewFrom(a), fmt.Sprintf("Attachment %q (#%d, %d bytes).", a.OriginalName, a.ID, a.FileSize), nil
}
