package knowledge

import (
	"net/http"
	"strconv"

	"github.com/company/smartticket/internal/errors"
	"github.com/gin-gonic/gin"
)

// Handlers handles knowledge-related HTTP requests.
type Handlers struct {
	service *Service
}

// NewHandlers creates new knowledge handlers.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// CreateKnowledgeArticle creates a new knowledge article.
// @Summary Create a new knowledge article
// @Description Creates a new knowledge base article with the provided content
// @Tags knowledge
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param request body knowledge.CreateKnowledgeArticleRequest true "Knowledge article creation data"
// @Success 201 {object} knowledge.KnowledgeArticleResponse
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/knowledge/articles [post]
func (h *Handlers) CreateKnowledgeArticle(c *gin.Context) {
	// Parse request
	var req CreateKnowledgeArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := errors.NewInvalidInputError("request_body", err.Error())
		errors.ErrorHandler(c, appErr)
		return
	}

	// Get user info from context
	userID := c.GetUint("user_id")

	// Log knowledge article creation attempt
	c.Set("security_event", "knowledge_article_creation_attempt")
	c.Set("target_resource", req.Title)

	// Create knowledge article
	article, err := h.service.CreateKnowledgeArticleCtx(c.Request.Context(), userID, &req)
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	// Log successful creation
	c.Set("security_event", "knowledge_article_created")
	c.Set("target_resource", article.Title)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    article,
	})
}

// GetKnowledgeArticle retrieves a knowledge article by ID.
// @Summary Get a knowledge article by ID
// @Description Retrieves a specific knowledge base article by its unique identifier
// @Tags knowledge
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param id path int true "Knowledge Article ID"
// @Success 200 {object} knowledge.KnowledgeArticleResponse
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 404 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/knowledge/articles/{id} [get]
func (h *Handlers) GetKnowledgeArticle(c *gin.Context) {
	// Parse article ID
	articleID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErr := errors.NewInvalidInputError("article_id", c.Param("id"))
		errors.ErrorHandler(c, appErr)
		return
	}

	// Get knowledge article
	article, err := h.service.GetKnowledgeArticle(uint(articleID))
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    article,
	})
}

// ListKnowledgeArticles retrieves knowledge articles with pagination and filtering.
// @Summary List knowledge articles
// @Description Retrieves a paginated list of knowledge base articles with optional filtering
// @Tags knowledge
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param page query int false "Page number" default(1) minimum(1)
// @Param page_size query int false "Number of articles per page" default(20) minimum(1) maximum(100)
// @Param status query string false "Filter by article status" Enums(draft,published,archived)
// @Param category query string false "Filter by category"
// @Param product_id query int false "Filter by product ID"
// @Param service_id query int false "Filter by service ID"
// @Param search query string false "Search articles by title, content, or summary"
// @Success 200 {object} github_com_company_smartticket_internal_server.Response
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/knowledge/articles [get]
func (h *Handlers) ListKnowledgeArticles(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Parse filters
	filters := make(map[string]interface{})
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	if category := c.Query("category"); category != "" {
		filters["category"] = category
	}
	if productID := c.Query("product_id"); productID != "" {
		if pid, err := strconv.ParseUint(productID, 10, 32); err == nil {
			filters["product_id"] = uint(pid)
		}
	}
	if serviceID := c.Query("service_id"); serviceID != "" {
		if sid, err := strconv.ParseUint(serviceID, 10, 32); err == nil {
			filters["service_id"] = uint(sid)
		}
	}
	if search := c.Query("search"); search != "" {
		filters["search"] = search
	}

	// Get knowledge articles
	result, err := h.service.ListKnowledgeArticles(page, pageSize, filters)
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result.Data,
		"meta": gin.H{
			"total":       result.Total,
			"page":        result.Page,
			"page_size":   result.PageSize,
			"total_pages": result.TotalPages,
		},
	})
}

// UpdateKnowledgeArticle updates an existing knowledge article.
// @Summary Update a knowledge article
// @Description Updates an existing knowledge base article with new information
// @Tags knowledge
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param id path int true "Knowledge Article ID"
// @Param request body knowledge.UpdateKnowledgeArticleRequest true "Knowledge article update data"
// @Success 200 {object} knowledge.KnowledgeArticleResponse
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 404 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/knowledge/articles/{id} [put]
func (h *Handlers) UpdateKnowledgeArticle(c *gin.Context) {
	// Parse article ID
	articleID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErr := errors.NewInvalidInputError("article_id", c.Param("id"))
		errors.ErrorHandler(c, appErr)
		return
	}

	// Get user info from context
	userID := c.GetUint("user_id")

	// Parse request
	var req UpdateKnowledgeArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := errors.NewInvalidInputError("request_body", err.Error())
		errors.ErrorHandler(c, appErr)
		return
	}

	// Log knowledge article update attempt
	c.Set("security_event", "knowledge_article_update_attempt")
	c.Set("target_resource", strconv.FormatUint(articleID, 10))

	// Update knowledge article
	article, err := h.service.UpdateKnowledgeArticleCtx(c.Request.Context(), uint(articleID), userID, &req)
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	// Log successful update
	c.Set("security_event", "knowledge_article_updated")
	c.Set("target_resource", article.Title)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    article,
	})
}

// DeleteKnowledgeArticle soft deletes a knowledge article.
// @Summary Delete a knowledge article
// @Description Soft deletes a knowledge base article (marks as deleted but preserves data)
// @Tags knowledge
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Param id path int true "Knowledge Article ID"
// @Success 200 {object} github_com_company_smartticket_internal_server.Response
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 404 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/knowledge/articles/{id} [delete]
func (h *Handlers) DeleteKnowledgeArticle(c *gin.Context) {
	// Parse article ID
	articleID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErr := errors.NewInvalidInputError("article_id", c.Param("id"))
		errors.ErrorHandler(c, appErr)
		return
	}

	// Get user info from context
	userID := c.GetUint("user_id")

	// Log knowledge article deletion attempt
	c.Set("security_event", "knowledge_article_deletion_attempt")
	c.Set("target_resource", strconv.FormatUint(articleID, 10))

	// Delete knowledge article
	err = h.service.DeleteKnowledgeArticleCtx(c.Request.Context(), uint(articleID), userID)
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	// Log successful deletion
	c.Set("security_event", "knowledge_article_deleted")
	c.Set("target_resource", strconv.FormatUint(articleID, 10))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Knowledge article deleted successfully",
	})
}

// GetKnowledgeArticleStats retrieves knowledge article statistics.
// @Summary Get knowledge article statistics
// @Description Retrieves statistical information about knowledge base articles
// @Tags knowledge
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} knowledge.KnowledgeArticleStatsResponse
// @Failure 401 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 403 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 500 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/knowledge/articles/stats [get]
func (h *Handlers) GetKnowledgeArticleStats(c *gin.Context) {
	// Get knowledge article statistics
	stats, err := h.service.GetKnowledgeArticleStats()
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// SearchRequest is the body for POST /knowledge/search.
type SearchRequest struct {
	Query string `json:"query"`
	TopK  int    `json:"top_k"`
}

// SearchKnowledge runs a semantic search over indexed knowledge articles.
// @Summary Semantic search over knowledge base
// @Tags knowledge
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body knowledge.SearchRequest true "Search query"
// @Success 200 {object} github_com_company_smartticket_internal_server.Response
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 503 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/knowledge/search [post]
func (h *Handlers) SearchKnowledge(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ErrorHandler(c, errors.NewInvalidInputError("request_body", err.Error()))
		return
	}
	if trimString(req.Query) == "" {
		errors.ErrorHandler(c, errors.NewValidationError("query must not be empty"))
		return
	}

	hits, err := h.service.Search(c.Request.Context(), req.Query, req.TopK)
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"hits": hits},
	})
}

// AskRequest is the body for POST /knowledge/ask.
type AskRequest struct {
	Question string `json:"question"`
	TopK     int    `json:"top_k"`
}

// AskKnowledge answers a question using RAG over the knowledge base.
// @Summary Ask the knowledge base assistant (RAG)
// @Tags knowledge
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body knowledge.AskRequest true "Question"
// @Success 200 {object} github_com_company_smartticket_internal_server.Response
// @Failure 400 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Failure 503 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/knowledge/ask [post]
func (h *Handlers) AskKnowledge(c *gin.Context) {
	var req AskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ErrorHandler(c, errors.NewInvalidInputError("request_body", err.Error()))
		return
	}
	if trimString(req.Question) == "" {
		errors.ErrorHandler(c, errors.NewValidationError("question must not be empty"))
		return
	}

	res, err := h.service.Ask(c.Request.Context(), req.Question, req.TopK)
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"answer":    res.Answer,
			"citations": res.Citations,
		},
	})
}

// ReindexKnowledge re-indexes all non-deleted articles into the semantic store.
// @Summary Re-index the knowledge base (admin)
// @Tags knowledge
// @Produce json
// @Security BearerAuth
// @Success 200 {object} github_com_company_smartticket_internal_server.Response
// @Failure 503 {object} github_com_company_smartticket_internal_errors.ErrorResponse
// @Router /api/v1/knowledge/reindex [post]
func (h *Handlers) ReindexKnowledge(c *gin.Context) {
	indexed, failed, err := h.service.Reindex(c.Request.Context())
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"indexed": indexed, "failed": failed},
	})
}
