use chrono::{DateTime, Utc};
use sqlx::Row;
use std::sync::Arc;
use tonic::{Request, Response, Status};
use uuid::Uuid;

use crate::auth_middleware::RequestExt;
use smartticket_shared_database::models::KnowledgeStatus;
use smartticket_shared_database::models::KnowledgeVisibility;
use crate::smartticket_v1::{
    knowledge_service_server::KnowledgeService, CreateArticleRequest, CreateArticleResponse,
    CreateCategoryRequest, CreateCategoryResponse, DeleteArticleRequest, DeleteArticleResponse,
    GetArticleRequest, GetArticleResponse, GetCategoriesRequest, GetCategoriesResponse,
    KnowledgeArticle, KnowledgeCategory, ListArticlesRequest, ListArticlesResponse,
    PaginationResponse, Response as ApiResponse, SearchArticlesRequest, SearchArticlesResponse,
    UpdateArticleRequest, UpdateArticleResponse,
};

/// Knowledge service implementation
#[derive(Debug, Clone)]
pub struct KnowledgeGrpcService {
    db_pool: Arc<sqlx::PgPool>,
}

impl KnowledgeGrpcService {
    pub fn new(db_pool: Arc<sqlx::PgPool>) -> Self {
        Self { db_pool }
    }

    /// Create response wrapper
    fn create_response(&self, success: bool, message: &str, request_id: &str) -> ApiResponse {
        ApiResponse {
            success,
            message: message.to_string(),
            data: None,
            errors: vec![],
            request_id: request_id.to_string(),
        }
    }

    /// Convert timestamp to protobuf timestamp
    fn timestamp_to_proto(&self, ts: DateTime<Utc>) -> ::prost_types::Timestamp {
        ::prost_types::Timestamp {
            seconds: ts.timestamp(),
            nanos: ts.timestamp_subsec_nanos() as i32,
        }
    }

    /// Get request ID from metadata
    fn get_request_id(&self, metadata: &Option<crate::smartticket_v1::RequestMetadata>) -> String {
        metadata
            .as_ref()
            .and_then(|m| {
                if m.request_id.is_empty() {
                    None
                } else {
                    Some(m.request_id.clone())
                }
            })
            .unwrap_or_else(|| Uuid::new_v4().to_string())
    }

    /// Convert database KnowledgeStatus to proto integer value
    fn knowledge_status_to_proto(&self, status: KnowledgeStatus) -> i32 {
        match status {
            KnowledgeStatus::Draft => 0,
            KnowledgeStatus::Review => 1,
            KnowledgeStatus::Published => 2,
            KnowledgeStatus::Archived => 3,
        }
    }

    /// Convert database KnowledgeVisibility to proto integer value
    fn knowledge_visibility_to_proto(&self, visibility: KnowledgeVisibility) -> i32 {
        match visibility {
            KnowledgeVisibility::Public => 0,
            KnowledgeVisibility::Internal => 1,
            KnowledgeVisibility::Restricted => 2,
        }
    }
}

#[tonic::async_trait]
impl KnowledgeService for KnowledgeGrpcService {
    /// Create a new knowledge article
    async fn create_article(
        &self,
        request: Request<CreateArticleRequest>,
    ) -> Result<Response<CreateArticleResponse>, Status> {
        let tenant_context = request.tenant_context()?;
        let req = request.into_inner();
        let request_id = self.get_request_id(&req.metadata);

        // Validate required fields
        if req.title.is_empty() {
            return Ok(Response::new(CreateArticleResponse {
                response: Some(self.create_response(false, "Title is required", &request_id)),
                article: None,
            }));
        }

        if req.content.is_empty() {
            return Ok(Response::new(CreateArticleResponse {
                response: Some(self.create_response(false, "Content is required", &request_id)),
                article: None,
            }));
        }

        // Generate new article
        let article_id = Uuid::new_v4();
        let now = Utc::now();

        // Parse category_id if provided
        let category_uuid: Option<Uuid> = if req.category_id.is_empty() {
            None
        } else {
            req.category_id.parse::<Uuid>().ok()
        };

        // Insert article into database
        let query = r#"
            INSERT INTO knowledge_articles (
                id, tenant_id, title, content, summary, category_id, author_id,
                status, visibility, language, tags, view_count, helpful_count,
                not_helpful_count, version, published_at, expires_at, created_at, updated_at
            ) VALUES (
                $1, $2, $3, $4, $5, $6, $7,
                $8, $9, $10, $11, $12, $13,
                $14, $15, $16, $17, $18, $19
            ) RETURNING *
        "#;

        let row = sqlx::query(query)
            .bind(article_id)
            .bind(tenant_context.tenant_id)
            .bind(&req.title)
            .bind(&req.content)
            .bind(&req.summary)
            .bind(category_uuid)
            .bind(tenant_context.user_id)
            .bind(KnowledgeStatus::Draft)
            .bind(match req.visibility {
                0 => KnowledgeVisibility::Public,
                1 => KnowledgeVisibility::Internal,
                2 => KnowledgeVisibility::Restricted,
                _ => KnowledgeVisibility::Internal,
            })
            .bind(&req.language)
            .bind(&req.tags)
            .bind(0i32) // view_count
            .bind(0i32) // helpful_count
            .bind(0i32) // not_helpful_count
            .bind(1i32) // version
            .bind(None::<DateTime<Utc>>) // published_at
            .bind(
                req.expires_at
                    .as_ref()
                    .map(|ts| DateTime::from_timestamp(ts.seconds, ts.nanos as u32))
                    .unwrap_or(None),
            ) // expires_at
            .bind(now) // created_at
            .bind(now) // updated_at
            .fetch_one(&*self.db_pool)
            .await
            .map_err(|e| Status::internal(format!("Failed to create article: {}", e)))?;

        // Create article response
        let article = KnowledgeArticle {
            id: row.get::<Uuid, _>("id").to_string(),
            tenant_id: row.get::<Uuid, _>("tenant_id").to_string(),
            title: row.get("title"),
            content: row.get("content"),
            summary: row.get("summary"),
            category_id: row.get::<Option<Uuid>, _>("category_id").map(|u| u.to_string()).unwrap_or_default(),
            author_id: row.get::<Uuid, _>("author_id").to_string(),
            status: self.knowledge_status_to_proto(row.get("status")),
            visibility: self.knowledge_visibility_to_proto(row.get("visibility")),
            language: row.get("language"),
            tags: row.get("tags"),
            view_count: row.get("view_count"),
            helpful_count: row.get("helpful_count"),
            not_helpful_count: row.get("not_helpful_count"),
            version: row.get("version"),
            published_at: row
                .get::<Option<DateTime<Utc>>, _>("published_at")
                .map(|ts| self.timestamp_to_proto(ts)),
            expires_at: row
                .get::<Option<DateTime<Utc>>, _>("expires_at")
                .map(|ts| self.timestamp_to_proto(ts)),
            created_at: Some(self.timestamp_to_proto(row.get("created_at"))),
            updated_at: Some(self.timestamp_to_proto(row.get("updated_at"))),
            author: None,   // TODO: Implement proper author parsing
            category: None, // TODO: Implement proper category parsing
        };

        Ok(Response::new(CreateArticleResponse {
            response: Some(self.create_response(true, "Article created successfully", &request_id)),
            article: Some(article),
        }))
    }

    /// Create article category
    async fn create_category(
        &self,
        request: Request<CreateCategoryRequest>,
    ) -> Result<Response<CreateCategoryResponse>, Status> {
        let tenant_context = request.tenant_context()?;
        let req = request.into_inner();
        let request_id = self.get_request_id(&req.metadata);

        // Validate required fields
        if req.name.is_empty() {
            return Ok(Response::new(CreateCategoryResponse {
                response: Some(self.create_response(
                    false,
                    "Category name is required",
                    &request_id,
                )),
                category: None,
            }));
        }

        // Create new category
        let category_id = Uuid::new_v4();
        let now = Utc::now();

        let insert_query = r#"
            INSERT INTO knowledge_categories (
                id, tenant_id, name, description, parent_id, icon, created_at, updated_at
            ) VALUES (
                $1, $2, $3, $4, $5, $6, $7, $8
            ) RETURNING *
        "#;

        let row = sqlx::query(insert_query)
            .bind(category_id)
            .bind(tenant_context.tenant_id)
            .bind(&req.name)
            .bind(&req.description)
            .bind(&req.parent_id)
            .bind(&req.icon)
            .bind(now)
            .bind(now)
            .fetch_one(&*self.db_pool)
            .await
            .map_err(|e| Status::internal(format!("Failed to create category: {}", e)))?;

        // Create category response
        let category = KnowledgeCategory {
            id: row.get::<Uuid, _>("id").to_string(),
            tenant_id: row.get::<Uuid, _>("tenant_id").to_string(),
            name: row.get("name"),
            description: row.get("description"),
            parent_id: row.get::<Option<Uuid>, _>("parent_id").map(|u| u.to_string()).unwrap_or_default(),
            icon: row.get("icon"),
            created_at: Some(self.timestamp_to_proto(row.get("created_at"))),
            updated_at: Some(self.timestamp_to_proto(row.get("updated_at"))),
            children: vec![], // TODO: Implement hierarchical loading
        };

        Ok(Response::new(CreateCategoryResponse {
            response: Some(self.create_response(
                true,
                "Category created successfully",
                &request_id,
            )),
            category: Some(category),
        }))
    }

    /// Get an article by ID
    async fn get_article(
        &self,
        request: Request<GetArticleRequest>,
    ) -> Result<Response<GetArticleResponse>, Status> {
        let tenant_context = request.tenant_context()?;
        let req = request.into_inner();
        let request_id = self.get_request_id(&req.metadata);

        // Get article from database
        let query = r#"
            SELECT * FROM knowledge_articles
            WHERE id = $1 AND tenant_id = $2 AND is_deleted = false
        "#;

        let row = sqlx::query(query)
            .bind(&req.article_id)
            .bind(tenant_context.tenant_id)
            .fetch_optional(&*self.db_pool)
            .await
            .map_err(|e| Status::internal(format!("Failed to retrieve article: {}", e)))?;

        if let Some(row) = row {
            // Create article response
            let article = KnowledgeArticle {
                id: row.get::<Uuid, _>("id").to_string(),
                tenant_id: row.get::<Uuid, _>("tenant_id").to_string(),
                title: row.get("title"),
                content: row.get("content"),
                summary: row.get("summary"),
                category_id: row.get::<Option<Uuid>, _>("category_id").map(|u| u.to_string()).unwrap_or_default(),
                author_id: row.get::<Uuid, _>("author_id").to_string(),
                status: self.knowledge_status_to_proto(row.get("status")),
                visibility: self.knowledge_visibility_to_proto(row.get("visibility")),
                language: row.get("language"),
                tags: row.get("tags"),
                view_count: row.get("view_count"),
                helpful_count: row.get("helpful_count"),
                not_helpful_count: row.get("not_helpful_count"),
                version: row.get("version"),
                published_at: row
                    .get::<Option<DateTime<Utc>>, _>("published_at")
                    .map(|ts| self.timestamp_to_proto(ts)),
                expires_at: row
                    .get::<Option<DateTime<Utc>>, _>("expires_at")
                    .map(|ts| self.timestamp_to_proto(ts)),
                created_at: Some(self.timestamp_to_proto(row.get("created_at"))),
                updated_at: Some(self.timestamp_to_proto(row.get("updated_at"))),
                author: None,   // TODO: Join with users table
                category: None, // TODO: Join with categories table
            };

            Ok(Response::new(GetArticleResponse {
                response: Some(self.create_response(
                    true,
                    "Article retrieved successfully",
                    &request_id,
                )),
                article: Some(article),
                related_articles: vec![],
            }))
        } else {
            Ok(Response::new(GetArticleResponse {
                response: Some(self.create_response(false, "Article not found", &request_id)),
                article: None,
                related_articles: vec![],
            }))
        }
    }

    /// Get article categories
    async fn get_categories(
        &self,
        request: Request<GetCategoriesRequest>,
    ) -> Result<Response<GetCategoriesResponse>, Status> {
        let tenant_context = request.tenant_context()?;
        let req = request.into_inner();
        let request_id = self.get_request_id(&req.metadata);

        // Build query for categories
        let query = r#"
            SELECT * FROM knowledge_categories
            WHERE tenant_id = $1 AND is_deleted = false
            ORDER BY name ASC
        "#;

        let rows = sqlx::query(query)
            .bind(tenant_context.tenant_id)
            .fetch_all(&*self.db_pool)
            .await
            .map_err(|e| Status::internal(format!("Failed to retrieve categories: {}", e)))?;

        // Convert rows to KnowledgeCategory objects
        let categories: Vec<KnowledgeCategory> = rows
            .into_iter()
            .map(|row| {
                KnowledgeCategory {
                    id: row.get::<Uuid, _>("id").to_string(),
                    tenant_id: row.get::<Uuid, _>("tenant_id").to_string(),
                    name: row.get("name"),
                    description: row.get("description"),
                    parent_id: row.get::<Option<Uuid>, _>("parent_id").map(|u| u.to_string()).unwrap_or_default(),
                    icon: row.get("icon"),
                    created_at: Some(self.timestamp_to_proto(row.get("created_at"))),
                    updated_at: Some(self.timestamp_to_proto(row.get("updated_at"))),
                    children: vec![], // TODO: Implement hierarchical loading
                }
            })
            .collect();

        Ok(Response::new(GetCategoriesResponse {
            response: Some(self.create_response(
                true,
                "Categories retrieved successfully",
                &request_id,
            )),
            categories,
        }))
    }

    /// List articles with filtering and pagination
    async fn list_articles(
        &self,
        request: Request<ListArticlesRequest>,
    ) -> Result<Response<ListArticlesResponse>, Status> {
        let tenant_context = request.tenant_context()?;
        let req = request.into_inner();
        let request_id = self.get_request_id(&req.metadata);

        // Simple query for now - this is a working implementation
        let query = r#"
            SELECT * FROM knowledge_articles
            WHERE tenant_id = $1 AND is_deleted = false
            ORDER BY created_at DESC
            LIMIT 50
        "#;

        let rows = sqlx::query(query)
            .bind(tenant_context.tenant_id)
            .fetch_all(&*self.db_pool)
            .await
            .map_err(|e| Status::internal(format!("Failed to retrieve articles: {}", e)))?;

        // Convert rows to KnowledgeArticle objects
        let articles: Vec<KnowledgeArticle> = rows
            .into_iter()
            .map(|row| {
                KnowledgeArticle {
                    id: row.get::<Uuid, _>("id").to_string(),
                    tenant_id: row.get::<Uuid, _>("tenant_id").to_string(),
                    title: row.get("title"),
                    content: row.get("content"),
                    summary: row.get("summary"),
                    category_id: row.get::<Option<Uuid>, _>("category_id").map(|u| u.to_string()).unwrap_or_default(),
                    author_id: row.get::<Uuid, _>("author_id").to_string(),
                    status: self.knowledge_status_to_proto(row.get("status")),
                    visibility: self.knowledge_visibility_to_proto(row.get("visibility")),
                    language: row.get("language"),
                    tags: row.get("tags"),
                    view_count: row.get("view_count"),
                    helpful_count: row.get("helpful_count"),
                    not_helpful_count: row.get("not_helpful_count"),
                    version: row.get("version"),
                    published_at: row
                        .get::<Option<DateTime<Utc>>, _>("published_at")
                        .map(|ts| self.timestamp_to_proto(ts)),
                    expires_at: row
                        .get::<Option<DateTime<Utc>>, _>("expires_at")
                        .map(|ts| self.timestamp_to_proto(ts)),
                    created_at: Some(self.timestamp_to_proto(row.get("created_at"))),
                    updated_at: Some(self.timestamp_to_proto(row.get("updated_at"))),
                    author: None,   // TODO: Join with users table
                    category: None, // TODO: Join with categories table
                }
            })
            .collect();

        let pagination_response = PaginationResponse {
            total_count: articles.len() as i32,
            page_size: 50,
            next_page_token: String::new(),
            prev_page_token: String::new(),
        };

        Ok(Response::new(ListArticlesResponse {
            response: Some(self.create_response(
                true,
                "Articles retrieved successfully",
                &request_id,
            )),
            articles,
            pagination: Some(pagination_response),
        }))
    }

    /// Search knowledge articles
    async fn search_articles(
        &self,
        request: Request<SearchArticlesRequest>,
    ) -> Result<Response<SearchArticlesResponse>, Status> {
        let tenant_context = request.tenant_context()?;
        let req = request.into_inner();
        let request_id = self.get_request_id(&req.metadata);

        // Simple search implementation
        let query = if req.query.is_empty() {
            r#"
                SELECT * FROM knowledge_articles
                WHERE tenant_id = $1 AND is_deleted = false
                ORDER BY updated_at DESC
                LIMIT 20
            "#
        } else {
            r#"
                SELECT * FROM knowledge_articles
                WHERE tenant_id = $1 AND is_deleted = false
                  AND (title ILIKE $2 OR content ILIKE $2 OR summary ILIKE $2)
                ORDER BY updated_at DESC
                LIMIT 20
            "#
        };

        let mut query_builder = sqlx::query(query).bind(tenant_context.tenant_id);
        if !req.query.is_empty() {
            query_builder = query_builder.bind(format!("%{}%", req.query));
        }

        let rows = query_builder
            .fetch_all(&*self.db_pool)
            .await
            .map_err(|e| Status::internal(format!("Failed to search articles: {}", e)))?;

        // Convert rows to KnowledgeArticle objects
        let articles: Vec<KnowledgeArticle> = rows
            .into_iter()
            .map(|row| {
                KnowledgeArticle {
                    id: row.get::<Uuid, _>("id").to_string(),
                    tenant_id: row.get::<Uuid, _>("tenant_id").to_string(),
                    title: row.get("title"),
                    content: row.get("content"),
                    summary: row.get("summary"),
                    category_id: row.get::<Option<Uuid>, _>("category_id").map(|u| u.to_string()).unwrap_or_default(),
                    author_id: row.get::<Uuid, _>("author_id").to_string(),
                    status: self.knowledge_status_to_proto(row.get("status")),
                    visibility: self.knowledge_visibility_to_proto(row.get("visibility")),
                    language: row.get("language"),
                    tags: row.get("tags"),
                    view_count: row.get("view_count"),
                    helpful_count: row.get("helpful_count"),
                    not_helpful_count: row.get("not_helpful_count"),
                    version: row.get("version"),
                    published_at: row
                        .get::<Option<DateTime<Utc>>, _>("published_at")
                        .map(|ts| self.timestamp_to_proto(ts)),
                    expires_at: row
                        .get::<Option<DateTime<Utc>>, _>("expires_at")
                        .map(|ts| self.timestamp_to_proto(ts)),
                    created_at: Some(self.timestamp_to_proto(row.get("created_at"))),
                    updated_at: Some(self.timestamp_to_proto(row.get("updated_at"))),
                    author: None,   // TODO: Join with users table
                    category: None, // TODO: Join with categories table
                }
            })
            .collect();

        let total_count = articles.len() as i32;
        let pagination_response = PaginationResponse {
            total_count,
            page_size: 20,
            next_page_token: String::new(),
            prev_page_token: String::new(),
        };

        Ok(Response::new(SearchArticlesResponse {
            response: Some(self.create_response(
                true,
                "Search completed successfully",
                &request_id,
            )),
            articles,
            pagination: Some(pagination_response),
            total_matches: total_count,
        }))
    }

    /// Update an article
    async fn update_article(
        &self,
        request: Request<UpdateArticleRequest>,
    ) -> Result<Response<UpdateArticleResponse>, Status> {
        let tenant_context = request.tenant_context()?;
        let req = request.into_inner();
        let request_id = self.get_request_id(&req.metadata);

        // Check if article exists
        let check_query = r#"
            SELECT id FROM knowledge_articles
            WHERE id = $1 AND tenant_id = $2 AND is_deleted = false
        "#;

        let check_row = sqlx::query(check_query)
            .bind(&req.article_id)
            .bind(tenant_context.tenant_id)
            .fetch_optional(&*self.db_pool)
            .await
            .map_err(|e| Status::internal(format!("Failed to check article: {}", e)))?;

        if check_row.is_none() {
            return Ok(Response::new(UpdateArticleResponse {
                response: Some(self.create_response(false, "Article not found", &request_id)),
                article: None,
            }));
        }

        // For now, just return the existing article
        // TODO: Implement proper update logic
        Ok(Response::new(UpdateArticleResponse {
            response: Some(self.create_response(true, "Article update not implemented yet", &request_id)),
            article: None,
        }))
    }

    /// Delete an article (soft delete)
    async fn delete_article(
        &self,
        request: Request<DeleteArticleRequest>,
    ) -> Result<Response<DeleteArticleResponse>, Status> {
        let tenant_context = request.tenant_context()?;
        let req = request.into_inner();
        let request_id = self.get_request_id(&req.metadata);

        // Soft delete by setting is_deleted flag
        let update_query = r#"
            UPDATE knowledge_articles
            SET is_deleted = true, updated_at = NOW()
            WHERE id = $1 AND tenant_id = $2
        "#;

        sqlx::query(update_query)
            .bind(&req.article_id)
            .bind(tenant_context.tenant_id)
            .execute(&*self.db_pool)
            .await
            .map_err(|e| Status::internal(format!("Failed to delete article: {}", e)))?;

        Ok(Response::new(DeleteArticleResponse {
            response: Some(self.create_response(
                true,
                "Article deleted successfully",
                &request_id,
            )),
        }))
    }

    // Placeholder implementations for remaining methods
    async fn publish_article(
        &self,
        _request: Request<crate::smartticket_v1::PublishArticleRequest>,
    ) -> Result<Response<crate::smartticket_v1::PublishArticleResponse>, Status> {
        Err(Status::unimplemented("PublishArticle not implemented yet"))
    }

    async fn archive_article(
        &self,
        _request: Request<crate::smartticket_v1::ArchiveArticleRequest>,
    ) -> Result<Response<crate::smartticket_v1::ArchiveArticleResponse>, Status> {
        Err(Status::unimplemented("ArchiveArticle not implemented yet"))
    }

    async fn get_article_suggestions(
        &self,
        _request: Request<crate::smartticket_v1::GetArticleSuggestionsRequest>,
    ) -> Result<Response<crate::smartticket_v1::GetArticleSuggestionsResponse>, Status> {
        Err(Status::unimplemented("GetArticleSuggestions not implemented yet"))
    }

    async fn rate_article(
        &self,
        _request: Request<crate::smartticket_v1::RateArticleRequest>,
    ) -> Result<Response<crate::smartticket_v1::RateArticleResponse>, Status> {
        Err(Status::unimplemented("RateArticle not implemented yet"))
    }
}