package knowledge

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	apperrors "github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/knowledgebase"
	"github.com/company/smartticket/internal/llm"
	"github.com/company/smartticket/internal/logger"
	"github.com/company/smartticket/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// aiUnavailableMsg is returned by AI endpoints when semantic search is not configured.
const aiUnavailableMsg = "AI search is not configured (set up an embedding provider)"

// aiSearchFailedMsg is returned when AI is configured but a search call fails at runtime.
const aiSearchFailedMsg = "AI search is temporarily unavailable, please try again"

// askFallback is returned by Ask when no relevant context is found.
const askFallback = "I don't have information on that in the knowledge base."

// askSystemPrompt instructs the assistant to answer strictly from supplied context.
const askSystemPrompt = "You are SmartTicket's support assistant. Answer the user's question using ONLY the provided context from the knowledge base. Cite the article titles you used. If the context does not contain the answer, say you don't have that information."

// Service provides knowledge base operations.
type Service struct {
	db    *gorm.DB
	store *knowledgebase.Store
	llm   *llm.Service
}

// NewService creates a new knowledge service. store and llmSvc are optional:
// when nil (or not AI-ready) the semantic-search/ask endpoints return 503 and
// the auto-indexing hooks are skipped silently.
func NewService(db *gorm.DB, store *knowledgebase.Store, llmSvc *llm.Service) *Service {
	return &Service{db: db, store: store, llm: llmSvc}
}

// aiReady reports whether semantic search/indexing is available.
func (s *Service) aiReady() bool {
	return s.store != nil && s.store.Healthy() && s.store.DB().HasEmbedder()
}

// aiUnavailable returns a 503 AppError carrying msg verbatim.
func aiUnavailable(msg string) error {
	return &apperrors.AppError{
		Code:       apperrors.ErrCodeServiceUnavailable,
		Message:    msg,
		HTTPStatus: http.StatusServiceUnavailable,
		Severity:   apperrors.SeverityHigh,
		Timestamp:  time.Now(),
	}
}

// indexArticle best-effort indexes an article into the knowledge store. It runs
// DETACHED in a goroutine with its own context so embedding a large article can
// neither block the originating HTTP request nor be cancelled when the client
// disconnects (the request context must not be used here). Errors are logged.
func (s *Service) indexArticle(_ context.Context, id uint, title, content, sourceURL string) {
	if !s.aiReady() {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if err := s.store.SaveArticle(ctx, id, title, content, sourceURL); err != nil {
			logger.Warn("knowledge auto-index failed", zap.Uint("article_id", id), zap.Error(err))
		}
	}()
}

// unindexArticle best-effort removes an article from the knowledge store,
// detached from the request context.
func (s *Service) unindexArticle(_ context.Context, id uint) {
	if !s.aiReady() {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if err := s.store.DeleteArticle(ctx, id); err != nil {
			logger.Warn("knowledge auto-unindex failed", zap.Uint("article_id", id), zap.Error(err))
		}
	}()
}

// SearchHit is re-exported for handler/response use.
type SearchHit = knowledgebase.SearchHit

// Search runs a semantic search over indexed articles. Returns a 503 AppError
// when AI search is not configured.
func (s *Service) Search(ctx context.Context, query string, topK int) ([]SearchHit, error) {
	if !s.aiReady() {
		return nil, aiUnavailable(aiUnavailableMsg)
	}
	if topK <= 0 {
		topK = 5
	}
	res, err := s.store.Search(ctx, query, topK)
	if err != nil {
		logger.Warn("knowledge search failed", zap.Error(err))
		return nil, aiUnavailable(aiSearchFailedMsg)
	}
	return res.Hits, nil
}

// AskResult bundles a generated answer with the citations it was grounded in.
type AskResult struct {
	Answer    string      `json:"answer"`
	Citations []SearchHit `json:"citations"`
}

// Ask answers a question using retrieved knowledge context (RAG). Returns a 503
// AppError when AI search or chat is not configured.
func (s *Service) Ask(ctx context.Context, question string, topK int) (*AskResult, error) {
	if !s.aiReady() {
		return nil, aiUnavailable(aiUnavailableMsg)
	}
	if s.llm == nil {
		return nil, aiUnavailable(aiUnavailableMsg)
	}
	if topK <= 0 {
		topK = 5
	}
	res, err := s.store.Search(ctx, question, topK)
	if err != nil {
		logger.Warn("knowledge ask search failed", zap.Error(err))
		return nil, aiUnavailable(aiSearchFailedMsg)
	}
	if len(res.Hits) == 0 {
		return &AskResult{Answer: askFallback, Citations: []SearchHit{}}, nil
	}
	msgs := []llm.ChatMessage{
		{Role: "system", Content: askSystemPrompt},
		{Role: "user", Content: "Context:\n" + res.Context + "\n\nQuestion: " + question},
	}
	answer, err := s.llm.Chat(ctx, msgs)
	if err != nil {
		return nil, aiUnavailable("AI assistant is not configured (set up a chat provider)")
	}
	return &AskResult{Answer: answer, Citations: res.Hits}, nil
}

// reindexAll synchronously re-embeds every non-deleted article and returns the
// success/failure counts. Used directly by tests; the public Reindex runs it
// detached in the background.
func (s *Service) reindexAll(ctx context.Context) (indexed, failed int) {
	var articles []models.KnowledgeArticle
	if err := s.db.Find(&articles).Error; err != nil {
		logger.Warn("reindex: failed to load articles", zap.Error(err))
		return 0, 0
	}
	for i := range articles {
		a := &articles[i]
		if serr := s.store.SaveArticle(ctx, a.ID, a.Title, a.Content, ""); serr != nil {
			logger.Warn("reindex article failed", zap.Uint("article_id", a.ID), zap.Error(serr))
			failed++
			continue
		}
		indexed++
	}
	return indexed, failed
}

// Reindex schedules a background re-index of ALL non-deleted articles and
// returns how many were scheduled. Embedding runs detached (with its own
// long-lived context) so a large corpus neither blocks the request nor is
// cancelled by the reverse proxy / client disconnect.
func (s *Service) Reindex(_ context.Context) (scheduled int, err error) {
	if !s.aiReady() {
		return 0, aiUnavailable(aiUnavailableMsg)
	}
	var count int64
	if err := s.db.Model(&models.KnowledgeArticle{}).Count(&count).Error; err != nil {
		return 0, apperrors.NewInternalError("failed to count articles for reindex: %w", err)
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
		defer cancel()
		ind, fail := s.reindexAll(ctx)
		logger.Info("knowledge reindex complete", zap.Int("indexed", ind), zap.Int("failed", fail))
	}()
	return int(count), nil
}

// CreateKnowledgeArticleRequest represents the request to create a knowledge article.
type CreateKnowledgeArticleRequest struct {
	Title     string `json:"title" binding:"required,min=3,max=255"`
	Content   string `json:"content" binding:"required,min=10"`
	Summary   string `json:"summary" binding:"max=1000"`
	Category  string `json:"category" binding:"required,oneof=technical troubleshooting guide faq tutorial other"`
	Tags      string `json:"tags"` // JSON array
	Status    string `json:"status" binding:"oneof=draft published archived"`
	ProductID *uint  `json:"product_id"`
	ServiceID *uint  `json:"service_id"`
}

// UpdateKnowledgeArticleRequest represents the request to update a knowledge article.
type UpdateKnowledgeArticleRequest struct {
	Title     string `json:"title" binding:"omitempty,min=3,max=255"`
	Content   string `json:"content" binding:"omitempty,min=10"`
	Summary   string `json:"summary" binding:"omitempty,max=1000"`
	Category  string `json:"category" binding:"omitempty,oneof=technical troubleshooting guide faq tutorial other"`
	Tags      string `json:"tags"`
	Status    string `json:"status" binding:"omitempty,oneof=draft published archived"`
	ProductID *uint  `json:"product_id"`
	ServiceID *uint  `json:"service_id"`
}

// KnowledgeArticleResponse represents the response for a knowledge article.
type KnowledgeArticleResponse struct {
	ID          uint                `json:"id"`
	Title       string              `json:"title"`
	Content     string              `json:"content"`
	Summary     string              `json:"summary"`
	Category    string              `json:"category"`
	Tags        string              `json:"tags"`
	Status      string              `json:"status"`
	ViewCount   int64               `json:"view_count"`
	Version     int                 `json:"version"`
	ProductID   *uint               `json:"product_id"`
	ServiceID   *uint               `json:"service_id"`
	Product     *models.Product     `json:"product,omitempty"`
	Service     *models.Service     `json:"service,omitempty"`
	Attachments []models.Attachment `json:"attachments"`
	IsDeleted   bool                `json:"is_deleted"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	CreatedBy   string              `json:"created_by"`
	UpdatedBy   string              `json:"updated_by"`
}

// KnowledgeArticleListResponse represents a paginated knowledge article list.
type KnowledgeArticleListResponse struct {
	Data       []KnowledgeArticleResponse `json:"data"`
	Total      int64                      `json:"total"`
	Page       int                        `json:"page"`
	PageSize   int                        `json:"page_size"`
	TotalPages int                        `json:"total_pages"`
}

// KnowledgeArticleStatsResponse represents knowledge article statistics.
type KnowledgeArticleStatsResponse struct {
	TotalArticles     int64            `json:"total_articles"`
	PublishedArticles int64            `json:"published_articles"`
	DraftArticles     int64            `json:"draft_articles"`
	ArchivedArticles  int64            `json:"archived_articles"`
	CategoryBreakdown map[string]int64 `json:"category_breakdown"`
	TotalViews        int64            `json:"total_views"`
	RecentActivity    []RecentArticle  `json:"recent_activity"`
}

// RecentArticle represents a recently updated article.
type RecentArticle struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateKnowledgeArticle creates a new knowledge article.
func (s *Service) CreateKnowledgeArticle(userID uint, req *CreateKnowledgeArticleRequest) (*KnowledgeArticleResponse, error) {
	return s.createKnowledgeArticle(context.Background(), userID, req)
}

// CreateKnowledgeArticleCtx is the context-aware variant used by handlers so the
// best-effort indexing hook can honor request cancellation.
func (s *Service) CreateKnowledgeArticleCtx(ctx context.Context, userID uint, req *CreateKnowledgeArticleRequest) (*KnowledgeArticleResponse, error) {
	return s.createKnowledgeArticle(ctx, userID, req)
}

func (s *Service) createKnowledgeArticle(ctx context.Context, userID uint, req *CreateKnowledgeArticleRequest) (*KnowledgeArticleResponse, error) {
	// Normalize and validate input
	req.Title = trimString(req.Title)
	req.Summary = trimString(req.Summary)
	req.Category = trimString(req.Category)
	req.Status = trimString(req.Status)

	if req.Status == "" {
		req.Status = "draft"
	}
	if req.Category == "" {
		req.Category = "technical"
	}

	// Get user from ID
	var user models.User
	if err := s.db.Where("id = ? AND is_active = ?", userID, true).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewUnauthorizedError("用户未找到")
		}
		return nil, apperrors.NewInternalError("failed to find user: %w", err)
	}

	// Generate slug from title
	slug := generateSlug(req.Title)

	// Create knowledge article
	article := &models.KnowledgeArticle{

		Title:     req.Title,
		Slug:      slug,
		Content:   req.Content,
		Summary:   req.Summary,
		AuthorID:  user.ID,
		Status:    req.Status,
		Views:     0,
		Version:   1,
		ProductID: req.ProductID,
		ServiceID: req.ServiceID,
		Category:  req.Category,
		Tags:      req.Tags,
	}

	if err := s.db.Create(article).Error; err != nil {
		return nil, apperrors.NewInternalError("failed to create knowledge article: %w", err)
	}

	// Set CreatedBy and UpdatedBy after creation
	if err := s.db.Model(article).Updates(map[string]interface{}{
		"created_by": user.Email,
		"updated_by": user.Email,
	}).Error; err != nil {
		// Log error but don't fail - the article is already created
	}

	// Best-effort semantic index (after commit, non-fatal).
	s.indexArticle(ctx, article.ID, article.Title, article.Content, "")

	return s.getKnowledgeArticleResponse(article)
}

// GetKnowledgeArticle retrieves a knowledge article by ID and increments view count.
func (s *Service) GetKnowledgeArticle(id uint) (*KnowledgeArticleResponse, error) {
	var article models.KnowledgeArticle
	if err := s.db.Where("id = ?", id).
		Preload("Product").
		Preload("Service").
		Preload("Attachments").
		First(&article).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFoundError("knowledge article not found")
		}
		return nil, apperrors.NewInternalError("failed to retrieve knowledge article: %w", err)
	}

	// Increment view count
	if err := s.db.Model(&article).UpdateColumn("views", gorm.Expr("views + ?", 1)).Error; err != nil {
		// Log error but don't fail the request
		// This is a non-critical operation
	}

	article.Views++
	return s.getKnowledgeArticleResponse(&article)
}

// ListKnowledgeArticles retrieves knowledge articles with pagination and filtering.
func (s *Service) ListKnowledgeArticles(page, pageSize int, filters map[string]interface{}) (*KnowledgeArticleListResponse, error) {
	offset := (page - 1) * pageSize

	// Build query
	query := s.db.Model(&models.KnowledgeArticle{}).
		Preload("Product").
		Preload("Service").
		Preload("Attachments")

	// Apply filters
	if status, ok := filters["status"]; ok {
		query = query.Where("status = ?", status)
	}
	if category, ok := filters["category"]; ok {
		query = query.Where("category = ?", category)
	}
	if productID, ok := filters["product_id"]; ok {
		query = query.Where("product_id = ?", productID)
	}
	if serviceID, ok := filters["service_id"]; ok {
		query = query.Where("service_id = ?", serviceID)
	}
	if search, ok := filters["search"]; ok {
		searchStr := fmt.Sprintf("%%%s%%", search)
		query = query.Where("title LIKE ? OR content LIKE ? OR summary LIKE ?", searchStr, searchStr, searchStr)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, apperrors.NewInternalError("failed to count knowledge articles: %w", err)
	}

	// Get articles
	var articles []models.KnowledgeArticle
	if err := query.Order("updated_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&articles).Error; err != nil {
		return nil, apperrors.NewInternalError("failed to retrieve knowledge articles: %w", err)
	}

	// Convert to response
	responses := make([]KnowledgeArticleResponse, len(articles))
	for i, article := range articles {
		response, err := s.getKnowledgeArticleResponse(&article)
		if err != nil {
			return nil, err
		}
		responses[i] = *response
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &KnowledgeArticleListResponse{
		Data:       responses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// UpdateKnowledgeArticle updates an existing knowledge article.
func (s *Service) UpdateKnowledgeArticle(id uint, userID uint, req *UpdateKnowledgeArticleRequest) (*KnowledgeArticleResponse, error) {
	return s.updateKnowledgeArticle(context.Background(), id, userID, req)
}

// UpdateKnowledgeArticleCtx is the context-aware variant used by handlers.
func (s *Service) UpdateKnowledgeArticleCtx(ctx context.Context, id uint, userID uint, req *UpdateKnowledgeArticleRequest) (*KnowledgeArticleResponse, error) {
	return s.updateKnowledgeArticle(ctx, id, userID, req)
}

func (s *Service) updateKnowledgeArticle(ctx context.Context, id uint, userID uint, req *UpdateKnowledgeArticleRequest) (*KnowledgeArticleResponse, error) {
	var article models.KnowledgeArticle
	if err := s.db.Where("id = ?", id).
		First(&article).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFoundError("knowledge article not found")
		}
		return nil, apperrors.NewInternalError("failed to retrieve knowledge article: %w", err)
	}

	// Get user email from database
	var user models.User
	if err := s.db.Where("id = ? AND is_active = ?", userID, true).First(&user).Error; err != nil {
		return nil, apperrors.NewInternalError("failed to find user: %w", err)
	}

	// Update fields if provided
	updateData := make(map[string]interface{})
	updateData["updated_by"] = user.Email
	updateData["version"] = article.Version + 1

	if req.Title != "" {
		updateData["title"] = trimString(req.Title)
		updateData["slug"] = generateSlug(req.Title)
	}
	if req.Content != "" {
		updateData["content"] = req.Content
	}
	if req.Summary != "" {
		updateData["summary"] = trimString(req.Summary)
	}
	if req.Category != "" {
		updateData["category"] = trimString(req.Category)
	}
	if req.Tags != "" {
		updateData["tags"] = req.Tags
	}
	if req.Status != "" {
		updateData["status"] = trimString(req.Status)
	}
	if req.ProductID != nil {
		updateData["product_id"] = req.ProductID
	}
	if req.ServiceID != nil {
		updateData["service_id"] = req.ServiceID
	}

	// Update article
	if err := s.db.Model(&article).Updates(updateData).Error; err != nil {
		return nil, apperrors.NewInternalError("failed to update knowledge article: %w", err)
	}

	// Reload article with relationships
	if err := s.db.Where("id = ?", id).
		Preload("Product").
		Preload("Service").
		Preload("Attachments").
		First(&article).Error; err != nil {
		return nil, apperrors.NewInternalError("failed to reload knowledge article: %w", err)
	}

	// Best-effort semantic re-index (after commit, non-fatal).
	s.indexArticle(ctx, article.ID, article.Title, article.Content, "")

	return s.getKnowledgeArticleResponse(&article)
}

// DeleteKnowledgeArticle soft deletes a knowledge article.
func (s *Service) DeleteKnowledgeArticle(id uint, userID uint) error {
	return s.deleteKnowledgeArticle(context.Background(), id, userID)
}

// DeleteKnowledgeArticleCtx is the context-aware variant used by handlers.
func (s *Service) DeleteKnowledgeArticleCtx(ctx context.Context, id uint, userID uint) error {
	return s.deleteKnowledgeArticle(ctx, id, userID)
}

func (s *Service) deleteKnowledgeArticle(ctx context.Context, id uint, userID uint) error {
	// Get user email from database
	var user models.User
	if err := s.db.Where("id = ? AND is_active = ?", userID, true).First(&user).Error; err != nil {
		return apperrors.NewInternalError("failed to find user: %w", err)
	}

	result := s.db.Model(&models.KnowledgeArticle{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"deleted_at": gorm.Expr("CURRENT_TIMESTAMP"),
			"updated_by": user.Email,
		})

	if result.Error != nil {
		return apperrors.NewInternalError("failed to delete knowledge article: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return apperrors.NewNotFoundError("knowledge article not found")
	}

	// Best-effort removal from the semantic index (non-fatal).
	s.unindexArticle(ctx, id)

	return nil
}

// GetKnowledgeArticleStats retrieves knowledge article statistics.
func (s *Service) GetKnowledgeArticleStats() (*KnowledgeArticleStatsResponse, error) {
	stats := &KnowledgeArticleStatsResponse{}

	// Get total counts
	if err := s.db.Model(&models.KnowledgeArticle{}).
		Count(&stats.TotalArticles).Error; err != nil {
		return nil, apperrors.NewInternalError("failed to count total articles: %w", err)
	}

	// Get status breakdown
	var statusCounts []struct {
		Status string
		Count  int64
	}
	if err := s.db.Model(&models.KnowledgeArticle{}).
		Select("status, count(*) as count").
		Group("status").
		Scan(&statusCounts).Error; err != nil {
		return nil, apperrors.NewInternalError("failed to get status breakdown: %w", err)
	}

	stats.PublishedArticles = 0
	stats.DraftArticles = 0
	stats.ArchivedArticles = 0
	stats.CategoryBreakdown = make(map[string]int64)

	for _, sc := range statusCounts {
		switch sc.Status {
		case "published":
			stats.PublishedArticles = sc.Count
		case "draft":
			stats.DraftArticles = sc.Count
		case "archived":
			stats.ArchivedArticles = sc.Count
		}
	}

	// Get category breakdown
	var categoryCounts []struct {
		Category string
		Count    int64
	}
	if err := s.db.Model(&models.KnowledgeArticle{}).
		Select("category, count(*) as count").
		Group("category").
		Scan(&categoryCounts).Error; err != nil {
		return nil, apperrors.NewInternalError("failed to get category breakdown: %w", err)
	}

	for _, cc := range categoryCounts {
		stats.CategoryBreakdown[cc.Category] = cc.Count
	}

	// Get total views
	if err := s.db.Model(&models.KnowledgeArticle{}).
		Select("COALESCE(SUM(views), 0)").
		Scan(&stats.TotalViews).Error; err != nil {
		return nil, apperrors.NewInternalError("failed to get total views: %w", err)
	}

	// Get recent activity (last 10 updated articles)
	var recentArticles []models.KnowledgeArticle
	if err := s.db.Model(&models.KnowledgeArticle{}).
		Order("updated_at DESC").
		Limit(10).
		Select("id, title, status, updated_at").
		Find(&recentArticles).Error; err != nil {
		return nil, apperrors.NewInternalError("failed to get recent activity: %w", err)
	}

	stats.RecentActivity = make([]RecentArticle, len(recentArticles))
	for i, article := range recentArticles {
		stats.RecentActivity[i] = RecentArticle{
			ID:        article.ID,
			Title:     article.Title,
			Status:    article.Status,
			UpdatedAt: article.UpdatedAt,
		}
	}

	return stats, nil
}

// getKnowledgeArticleResponse converts a model to response format.
func (s *Service) getKnowledgeArticleResponse(article *models.KnowledgeArticle) (*KnowledgeArticleResponse, error) {
	response := &KnowledgeArticleResponse{
		ID:          article.ID,
		Title:       article.Title,
		Content:     article.Content,
		Summary:     article.Summary,
		Category:    article.Category,
		Tags:        article.Tags,
		Status:      article.Status,
		ViewCount:   int64(article.Views),
		Version:     article.Version,
		ProductID:   article.ProductID,
		ServiceID:   article.ServiceID,
		Product:     article.Product,
		Service:     article.Service,
		Attachments: article.Attachments,
		IsDeleted:   article.DeletedAt.Valid,
		CreatedAt:   article.CreatedAt,
		UpdatedAt:   article.UpdatedAt,
	}

	if article.CreatedBy != nil {
		response.CreatedBy = *article.CreatedBy
	}
	if article.UpdatedBy != nil {
		response.UpdatedBy = *article.UpdatedBy
	}

	return response, nil
}

// Helper functions.
func trimString(s string) string {
	return strings.TrimSpace(s)
}

// generateSlug creates a URL-friendly slug from title.
func generateSlug(title string) string {
	// Simple slug generation - can be enhanced
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")

	// Remove special characters except hyphens
	result := ""
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result += string(r)
		}
	}

	// Remove multiple consecutive hyphens
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}

	// Remove leading/trailing hyphens
	result = strings.Trim(result, "-")

	if result == "" {
		return "article"
	}

	return result
}

// ValidateProductService validates that product and service exist and are active.
func (s *Service) ValidateProductService(productID, serviceID *uint) error {
	if productID != nil {
		var product models.Product
		if err := s.db.Where("id = ? AND status = ?", *productID, "active").
			First(&product).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperrors.NewValidationError("product not found or inactive")
			}
			return apperrors.NewInternalError("failed to validate product: %w", err)
		}
	}

	if serviceID != nil {
		var service models.Service
		if err := s.db.Where("id = ? AND status = ?", *serviceID, "active").
			First(&service).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperrors.NewValidationError("service not found or inactive")
			}
			return apperrors.NewInternalError("failed to validate service: %w", err)
		}
	}

	return nil
}
