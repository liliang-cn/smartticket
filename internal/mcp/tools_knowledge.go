package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/company/smartticket/internal/knowledge"
)

// ----------------------------------------------------------------------------
// Knowledge-domain MCP tools.
//
// Each tool declares its own MCP-specific Input struct (json + jsonschema tags),
// translates it into the knowledge service DTOs, and delegates to the Backend.
//
// Output types use the cycle-safe knowledgeArticle / knowledgeArticleList views
// instead of knowledge.KnowledgeArticleResponse: the service response embeds
// *models.Product, *models.Service and []models.Attachment, which transitively
// reference cyclic GORM models (Product↔Service↔Ticket↔User) that the SDK's
// JSON-schema reflector rejects with "cycle detected". The flat views surface
// the associations as numeric IDs / counts instead. All identifiers carry a
// "knowledge" prefix to avoid collisions with sibling domains in this package.
// See server.go for the conventions and auth_whoami reference implementation.
// ----------------------------------------------------------------------------

// knowledgeArticle is the cycle-safe MCP output view of a knowledge article. It
// mirrors the scalar fields of knowledge.KnowledgeArticleResponse but flattens
// the embedded *models.Product, *models.Service and []models.Attachment (which
// transitively reference cyclic GORM models) to numeric IDs and a count.
type knowledgeArticle struct {
	ID              uint      `json:"id" jsonschema:"the article's numeric ID"`
	Title           string    `json:"title" jsonschema:"article title"`
	Content         string    `json:"content" jsonschema:"article body content"`
	Summary         string    `json:"summary,omitempty" jsonschema:"short summary"`
	Category        string    `json:"category" jsonschema:"category: technical, troubleshooting, guide, faq, tutorial, or other"`
	Tags            string    `json:"tags,omitempty" jsonschema:"tags as a JSON array string"`
	Status          string    `json:"status" jsonschema:"status: draft, published, or archived"`
	ViewCount       int64     `json:"view_count" jsonschema:"number of times the article has been viewed"`
	Version         int       `json:"version" jsonschema:"the article's version number"`
	ProductID       *uint     `json:"product_id,omitempty" jsonschema:"numeric ID of the associated product, if any"`
	ServiceID       *uint     `json:"service_id,omitempty" jsonschema:"numeric ID of the associated service, if any"`
	AttachmentCount int       `json:"attachment_count" jsonschema:"number of attachments on the article"`
	IsDeleted       bool      `json:"is_deleted" jsonschema:"whether the article is soft-deleted"`
	CreatedAt       time.Time `json:"created_at" jsonschema:"when the article was created"`
	UpdatedAt       time.Time `json:"updated_at" jsonschema:"when the article was last updated"`
	CreatedBy       string    `json:"created_by,omitempty" jsonschema:"identifier of the creating user"`
	UpdatedBy       string    `json:"updated_by,omitempty" jsonschema:"identifier of the last updating user"`
}

// knowledgeArticleFrom converts a service-layer knowledge.KnowledgeArticleResponse
// into the cycle-safe MCP view, flattening the embedded associations.
func knowledgeArticleFrom(r *knowledge.KnowledgeArticleResponse) knowledgeArticle {
	return knowledgeArticle{
		ID:              r.ID,
		Title:           r.Title,
		Content:         r.Content,
		Summary:         r.Summary,
		Category:        r.Category,
		Tags:            r.Tags,
		Status:          r.Status,
		ViewCount:       r.ViewCount,
		Version:         r.Version,
		ProductID:       r.ProductID,
		ServiceID:       r.ServiceID,
		AttachmentCount: len(r.Attachments),
		IsDeleted:       r.IsDeleted,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
		CreatedBy:       r.CreatedBy,
		UpdatedBy:       r.UpdatedBy,
	}
}

// knowledgeArticleList is the cycle-safe MCP output of knowledge_list. It mirrors
// knowledge.KnowledgeArticleListResponse but carries the flat knowledgeArticle view.
type knowledgeArticleList struct {
	Data       []knowledgeArticle `json:"data,omitempty" jsonschema:"the page of knowledge articles"`
	Total      int64              `json:"total" jsonschema:"total number of matching articles"`
	Page       int                `json:"page" jsonschema:"the 1-based page number returned"`
	PageSize   int                `json:"page_size" jsonschema:"the page size used"`
	TotalPages int                `json:"total_pages" jsonschema:"total number of pages available"`
}

// knowledgeCreateInput is the MCP input schema for knowledge_create.
type knowledgeCreateInput struct {
	Title     string `json:"title" jsonschema:"article title (3-255 characters)"`
	Content   string `json:"content" jsonschema:"article body content (minimum 10 characters)"`
	Summary   string `json:"summary,omitempty" jsonschema:"optional short summary (up to 1000 characters)"`
	Category  string `json:"category,omitempty" jsonschema:"category: technical, troubleshooting, guide, faq, tutorial, or other (defaults to technical)"`
	Tags      string `json:"tags,omitempty" jsonschema:"optional JSON array of tags"`
	Status    string `json:"status,omitempty" jsonschema:"status: draft, published, or archived (defaults to draft)"`
	ProductID *uint  `json:"product_id,omitempty" jsonschema:"optional associated product ID"`
	ServiceID *uint  `json:"service_id,omitempty" jsonschema:"optional associated service ID"`
}

// knowledgeGetInput is the MCP input schema for knowledge_get.
type knowledgeGetInput struct {
	ID uint `json:"id" jsonschema:"the numeric ID of the knowledge article to retrieve"`
}

// knowledgeListInput is the MCP input schema for knowledge_list.
type knowledgeListInput struct {
	Page      int    `json:"page,omitempty" jsonschema:"1-based page number (defaults to 1)"`
	PageSize  int    `json:"page_size,omitempty" jsonschema:"number of items per page (defaults to 20)"`
	Status    string `json:"status,omitempty" jsonschema:"filter by status: draft, published, or archived"`
	Category  string `json:"category,omitempty" jsonschema:"filter by category"`
	ProductID *uint  `json:"product_id,omitempty" jsonschema:"filter by associated product ID"`
	ServiceID *uint  `json:"service_id,omitempty" jsonschema:"filter by associated service ID"`
	Search    string `json:"search,omitempty" jsonschema:"free-text search across title, content, and summary"`
}

// knowledgeUpdateInput is the MCP input schema for knowledge_update. Empty
// string / nil fields are left unchanged by the service.
type knowledgeUpdateInput struct {
	ID        uint   `json:"id" jsonschema:"the numeric ID of the knowledge article to update"`
	Title     string `json:"title,omitempty" jsonschema:"new title (3-255 characters)"`
	Content   string `json:"content,omitempty" jsonschema:"new body content (minimum 10 characters)"`
	Summary   string `json:"summary,omitempty" jsonschema:"new summary (up to 1000 characters)"`
	Category  string `json:"category,omitempty" jsonschema:"new category: technical, troubleshooting, guide, faq, tutorial, or other"`
	Tags      string `json:"tags,omitempty" jsonschema:"new JSON array of tags"`
	Status    string `json:"status,omitempty" jsonschema:"new status: draft, published, or archived"`
	ProductID *uint  `json:"product_id,omitempty" jsonschema:"new associated product ID"`
	ServiceID *uint  `json:"service_id,omitempty" jsonschema:"new associated service ID"`
}

// knowledgeDeleteInput is the MCP input schema for knowledge_delete.
type knowledgeDeleteInput struct {
	ID uint `json:"id" jsonschema:"the numeric ID of the knowledge article to delete"`
}

// knowledgeStatsInput is the MCP input schema for knowledge_stats. It takes no
// arguments.
type knowledgeStatsInput struct{}

// knowledgeRecentArticle is the schema-safe MCP view of a recently-updated
// article in the stats payload.
type knowledgeRecentArticle struct {
	ID        uint      `json:"id" jsonschema:"the article's numeric ID"`
	Title     string    `json:"title" jsonschema:"the article title"`
	Status    string    `json:"status" jsonschema:"the article status"`
	UpdatedAt time.Time `json:"updated_at" jsonschema:"when the article was last updated"`
}

// knowledgeStatsOutput is the schema-safe MCP view of knowledge article
// statistics. The service-layer knowledge.KnowledgeArticleStatsResponse cannot be
// reused directly: its CategoryBreakdown (map) and RecentActivity ([]RecentArticle)
// fields lack `omitempty`, so a nil value marshals to JSON null and the go-sdk
// rejects it against the inferred object/array output schema. Here both carry
// `omitempty`.
type knowledgeStatsOutput struct {
	TotalArticles     int64                    `json:"total_articles" jsonschema:"total number of articles"`
	PublishedArticles int64                    `json:"published_articles" jsonschema:"number of published articles"`
	DraftArticles     int64                    `json:"draft_articles" jsonschema:"number of draft articles"`
	ArchivedArticles  int64                    `json:"archived_articles" jsonschema:"number of archived articles"`
	CategoryBreakdown map[string]int64         `json:"category_breakdown,omitempty" jsonschema:"article counts keyed by category"`
	TotalViews        int64                    `json:"total_views" jsonschema:"total article views"`
	RecentActivity    []knowledgeRecentArticle `json:"recent_activity,omitempty" jsonschema:"recently updated articles"`
}

// knowledgeStatsOutputFrom converts the service-layer stats response into the
// schema-safe MCP view.
func knowledgeStatsOutputFrom(r *knowledge.KnowledgeArticleStatsResponse) knowledgeStatsOutput {
	if r == nil {
		return knowledgeStatsOutput{}
	}
	out := knowledgeStatsOutput{
		TotalArticles:     r.TotalArticles,
		PublishedArticles: r.PublishedArticles,
		DraftArticles:     r.DraftArticles,
		ArchivedArticles:  r.ArchivedArticles,
		CategoryBreakdown: r.CategoryBreakdown,
		TotalViews:        r.TotalViews,
	}
	if len(r.RecentActivity) > 0 {
		out.RecentActivity = make([]knowledgeRecentArticle, len(r.RecentActivity))
		for i := range r.RecentActivity {
			out.RecentActivity[i] = knowledgeRecentArticle{
				ID:        r.RecentActivity[i].ID,
				Title:     r.RecentActivity[i].Title,
				Status:    r.RecentActivity[i].Status,
				UpdatedAt: r.RecentActivity[i].UpdatedAt,
			}
		}
	}
	return out
}

// knowledgeDeleteOutput is the structured output of knowledge_delete.
type knowledgeDeleteOutput struct {
	ID      uint `json:"id" jsonschema:"the ID of the deleted knowledge article"`
	Deleted bool `json:"deleted" jsonschema:"whether the article was deleted"`
}

// registerKnowledgeTools registers the knowledge-domain MCP tools.
// See server.go for the tool implementation conventions and auth_whoami template.
func registerKnowledgeTools(s *mcp.Server, b Backend) {
	registerTool(s,
		"knowledge_create",
		"Create a new knowledge base article.",
		"knowledge:write",
		func(ctx context.Context, in knowledgeCreateInput) (knowledgeArticle, string, error) {
			return knowledgeCreate(ctx, b, in)
		},
	)

	registerTool(s,
		"knowledge_get",
		"Retrieve a single knowledge base article by ID (increments its view count).",
		"knowledge:read",
		func(ctx context.Context, in knowledgeGetInput) (knowledgeArticle, string, error) {
			return knowledgeGet(ctx, b, in)
		},
	)

	registerTool(s,
		"knowledge_list",
		"List knowledge base articles with pagination and optional filters.",
		"knowledge:read",
		func(ctx context.Context, in knowledgeListInput) (knowledgeArticleList, string, error) {
			return knowledgeList(ctx, b, in)
		},
	)

	registerTool(s,
		"knowledge_update",
		"Update an existing knowledge base article; omitted fields are left unchanged.",
		"knowledge:write",
		func(ctx context.Context, in knowledgeUpdateInput) (knowledgeArticle, string, error) {
			return knowledgeUpdate(ctx, b, in)
		},
	)

	registerTool(s,
		"knowledge_delete",
		"Soft-delete a knowledge base article by ID.",
		"knowledge:write",
		func(ctx context.Context, in knowledgeDeleteInput) (knowledgeDeleteOutput, string, error) {
			return knowledgeDelete(ctx, b, in)
		},
	)

	registerTool(s,
		"knowledge_stats",
		"Return aggregate statistics about knowledge base articles.",
		"knowledge:read",
		func(ctx context.Context, in knowledgeStatsInput) (knowledgeStatsOutput, string, error) {
			return knowledgeStats(ctx, b, in)
		},
	)
}

// knowledgeCreate handles knowledge_create.
func knowledgeCreate(ctx context.Context, b Backend, in knowledgeCreateInput) (knowledgeArticle, string, error) {
	session, ok := SessionFromContext(ctx)
	if !ok || session == nil {
		return knowledgeArticle{}, "", ErrUnauthenticated
	}

	req := &knowledge.CreateKnowledgeArticleRequest{
		Title:     in.Title,
		Content:   in.Content,
		Summary:   in.Summary,
		Category:  in.Category,
		Tags:      in.Tags,
		Status:    in.Status,
		ProductID: in.ProductID,
		ServiceID: in.ServiceID,
	}

	resp, err := b.CreateKnowledgeArticle(session.UserID, req)
	if err != nil {
		return knowledgeArticle{}, "", err
	}

	summary := fmt.Sprintf("Created knowledge article #%d %q (status: %s).", resp.ID, resp.Title, resp.Status)
	return knowledgeArticleFrom(resp), summary, nil
}

// knowledgeGet handles knowledge_get.
func knowledgeGet(_ context.Context, b Backend, in knowledgeGetInput) (knowledgeArticle, string, error) {
	resp, err := b.GetKnowledgeArticle(in.ID)
	if err != nil {
		return knowledgeArticle{}, "", err
	}

	summary := fmt.Sprintf("Knowledge article #%d %q (status: %s, %d view(s)).", resp.ID, resp.Title, resp.Status, resp.ViewCount)
	return knowledgeArticleFrom(resp), summary, nil
}

// knowledgeList handles knowledge_list.
func knowledgeList(_ context.Context, b Backend, in knowledgeListInput) (knowledgeArticleList, string, error) {
	page := in.Page
	if page < 1 {
		page = 1
	}
	pageSize := in.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	filters := make(map[string]interface{})
	if in.Status != "" {
		filters["status"] = in.Status
	}
	if in.Category != "" {
		filters["category"] = in.Category
	}
	if in.ProductID != nil {
		filters["product_id"] = *in.ProductID
	}
	if in.ServiceID != nil {
		filters["service_id"] = *in.ServiceID
	}
	if in.Search != "" {
		filters["search"] = in.Search
	}

	resp, err := b.ListKnowledgeArticles(page, pageSize, filters)
	if err != nil {
		return knowledgeArticleList{}, "", err
	}

	articles := make([]knowledgeArticle, len(resp.Data))
	for i := range resp.Data {
		articles[i] = knowledgeArticleFrom(&resp.Data[i])
	}
	out := knowledgeArticleList{
		Data:       articles,
		Total:      resp.Total,
		Page:       resp.Page,
		PageSize:   resp.PageSize,
		TotalPages: resp.TotalPages,
	}

	summary := fmt.Sprintf("Found %d knowledge article(s); showing page %d of %d.", resp.Total, resp.Page, resp.TotalPages)
	return out, summary, nil
}

// knowledgeUpdate handles knowledge_update.
func knowledgeUpdate(ctx context.Context, b Backend, in knowledgeUpdateInput) (knowledgeArticle, string, error) {
	session, ok := SessionFromContext(ctx)
	if !ok || session == nil {
		return knowledgeArticle{}, "", ErrUnauthenticated
	}

	req := &knowledge.UpdateKnowledgeArticleRequest{
		Title:     in.Title,
		Content:   in.Content,
		Summary:   in.Summary,
		Category:  in.Category,
		Tags:      in.Tags,
		Status:    in.Status,
		ProductID: in.ProductID,
		ServiceID: in.ServiceID,
	}

	resp, err := b.UpdateKnowledgeArticle(in.ID, session.UserID, req)
	if err != nil {
		return knowledgeArticle{}, "", err
	}

	summary := fmt.Sprintf("Updated knowledge article #%d %q (status: %s, version %d).", resp.ID, resp.Title, resp.Status, resp.Version)
	return knowledgeArticleFrom(resp), summary, nil
}

// knowledgeDelete handles knowledge_delete.
func knowledgeDelete(ctx context.Context, b Backend, in knowledgeDeleteInput) (knowledgeDeleteOutput, string, error) {
	session, ok := SessionFromContext(ctx)
	if !ok || session == nil {
		return knowledgeDeleteOutput{}, "", ErrUnauthenticated
	}

	if err := b.DeleteKnowledgeArticle(in.ID, session.UserID); err != nil {
		return knowledgeDeleteOutput{}, "", err
	}

	out := knowledgeDeleteOutput{ID: in.ID, Deleted: true}
	summary := fmt.Sprintf("Deleted knowledge article #%d.", in.ID)
	return out, summary, nil
}

// knowledgeStats handles knowledge_stats.
func knowledgeStats(_ context.Context, b Backend, _ knowledgeStatsInput) (knowledgeStatsOutput, string, error) {
	resp, err := b.GetKnowledgeArticleStats()
	if err != nil {
		return knowledgeStatsOutput{}, "", err
	}

	summary := fmt.Sprintf("Knowledge base: %d article(s) total (%d published, %d draft, %d archived), %d total view(s).",
		resp.TotalArticles, resp.PublishedArticles, resp.DraftArticles, resp.ArchivedArticles, resp.TotalViews)
	return knowledgeStatsOutputFrom(resp), summary, nil
}
