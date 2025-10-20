package knowledge

import (
	"errors"
	"fmt"
	"strings"
	"time"

	apperrors "github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/models"
	"gorm.io/gorm"
)

// Service provides knowledge base operations
type Service struct {
	db *gorm.DB
}

// NewService creates a new knowledge service
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// CreateKnowledgeArticleRequest represents the request to create a knowledge article
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

// UpdateKnowledgeArticleRequest represents the request to update a knowledge article
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

// KnowledgeArticleResponse represents the response for a knowledge article
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

// KnowledgeArticleListResponse represents a paginated knowledge article list
type KnowledgeArticleListResponse struct {
	Data       []KnowledgeArticleResponse `json:"data"`
	Total      int64                      `json:"total"`
	Page       int                        `json:"page"`
	PageSize   int                        `json:"page_size"`
	TotalPages int                        `json:"total_pages"`
}

// KnowledgeArticleStatsResponse represents knowledge article statistics
type KnowledgeArticleStatsResponse struct {
	TotalArticles     int64            `json:"total_articles"`
	PublishedArticles int64            `json:"published_articles"`
	DraftArticles     int64            `json:"draft_articles"`
	ArchivedArticles  int64            `json:"archived_articles"`
	CategoryBreakdown map[string]int64 `json:"category_breakdown"`
	TotalViews        int64            `json:"total_views"`
	RecentActivity    []RecentArticle  `json:"recent_activity"`
}

// RecentArticle represents a recently updated article
type RecentArticle struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateKnowledgeArticle creates a new knowledge article
func (s *Service) CreateKnowledgeArticle(tenantID uint, userID uint, req *CreateKnowledgeArticleRequest) (*KnowledgeArticleResponse, error) {
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
	if err := s.db.Where("id = ? AND tenant_id = ? AND is_active = ?", userID, tenantID, true).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewUnauthorizedError("用户未找到")
		}
		return nil, apperrors.NewInternalError("failed to find user: %w", err)
	}

	// Generate slug from title
	slug := generateSlug(req.Title)

	// Create knowledge article
	article := &models.KnowledgeArticle{
		TenantID:  tenantID,
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

	return s.getKnowledgeArticleResponse(article)
}

// GetKnowledgeArticle retrieves a knowledge article by ID and increments view count
func (s *Service) GetKnowledgeArticle(tenantID uint, id uint) (*KnowledgeArticleResponse, error) {
	var article models.KnowledgeArticle
	if err := s.db.Where("id = ? AND tenant_id = ?", id, tenantID).
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

// ListKnowledgeArticles retrieves knowledge articles with pagination and filtering
func (s *Service) ListKnowledgeArticles(tenantID uint, page, pageSize int, filters map[string]interface{}) (*KnowledgeArticleListResponse, error) {
	offset := (page - 1) * pageSize

	// Build query
	query := s.db.Model(&models.KnowledgeArticle{}).
		Where("tenant_id = ?", tenantID).
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

// UpdateKnowledgeArticle updates an existing knowledge article
func (s *Service) UpdateKnowledgeArticle(tenantID uint, id uint, userID uint, req *UpdateKnowledgeArticleRequest) (*KnowledgeArticleResponse, error) {
	var article models.KnowledgeArticle
	if err := s.db.Where("id = ? AND tenant_id = ?", id, tenantID).
		First(&article).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFoundError("knowledge article not found")
		}
		return nil, apperrors.NewInternalError("failed to retrieve knowledge article: %w", err)
	}

	// Get user email from database
	var user models.User
	if err := s.db.Where("id = ? AND tenant_id = ? AND is_active = ?", userID, tenantID, true).First(&user).Error; err != nil {
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

	return s.getKnowledgeArticleResponse(&article)
}

// DeleteKnowledgeArticle soft deletes a knowledge article
func (s *Service) DeleteKnowledgeArticle(tenantID uint, id uint, userID uint) error {
	// Get user email from database
	var user models.User
	if err := s.db.Where("id = ? AND tenant_id = ? AND is_active = ?", userID, tenantID, true).First(&user).Error; err != nil {
		return apperrors.NewInternalError("failed to find user: %w", err)
	}

	result := s.db.Model(&models.KnowledgeArticle{}).
		Where("id = ? AND tenant_id = ?", id, tenantID).
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

	return nil
}

// GetKnowledgeArticleStats retrieves knowledge article statistics
func (s *Service) GetKnowledgeArticleStats(tenantID uint) (*KnowledgeArticleStatsResponse, error) {
	stats := &KnowledgeArticleStatsResponse{}

	// Get total counts
	if err := s.db.Model(&models.KnowledgeArticle{}).
		Where("tenant_id = ?", tenantID).
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
		Where("tenant_id = ?", tenantID).
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
		Where("tenant_id = ?", tenantID).
		Group("category").
		Scan(&categoryCounts).Error; err != nil {
		return nil, apperrors.NewInternalError("failed to get category breakdown: %w", err)
	}

	for _, cc := range categoryCounts {
		stats.CategoryBreakdown[cc.Category] = cc.Count
	}

	// Get total views
	if err := s.db.Model(&models.KnowledgeArticle{}).
		Where("tenant_id = ?", tenantID).
		Select("COALESCE(SUM(views), 0)").
		Scan(&stats.TotalViews).Error; err != nil {
		return nil, apperrors.NewInternalError("failed to get total views: %w", err)
	}

	// Get recent activity (last 10 updated articles)
	var recentArticles []models.KnowledgeArticle
	if err := s.db.Where("tenant_id = ?", tenantID).
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

// getKnowledgeArticleResponse converts a model to response format
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

// Helper functions
func trimString(s string) string {
	return strings.TrimSpace(s)
}

// generateSlug creates a URL-friendly slug from title
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

// ValidateProductService validates that product and service belong to the same tenant
func (s *Service) ValidateProductService(tenantID uint, productID, serviceID *uint) error {
	if productID != nil {
		var product models.Product
		if err := s.db.Where("id = ? AND tenant_id = ? AND status = ?", *productID, tenantID, "active").
			First(&product).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperrors.NewValidationError("product not found or inactive")
			}
			return apperrors.NewInternalError("failed to validate product: %w", err)
		}
	}

	if serviceID != nil {
		var service models.Service
		if err := s.db.Where("id = ? AND tenant_id = ? AND status = ?", *serviceID, tenantID, "active").
			First(&service).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperrors.NewValidationError("service not found or inactive")
			}
			return apperrors.NewInternalError("failed to validate service: %w", err)
		}
	}

	return nil
}
