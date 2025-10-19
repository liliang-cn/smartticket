use chrono::Utc;
use sqlx::{PgPool, Row};
use tracing::{error, info, instrument};
use uuid::Uuid;

use crate::models::ticket::*;
use smartticket_shared_database::TenantContext;
use smartticket_shared_error::{Result, SmartTicketError};

/// Ticket service for managing tickets
pub struct TicketService {
    pool: PgPool,
}

impl TicketService {
    /// Create a new ticket service
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// Create a new ticket
    #[instrument(skip(self))]
    pub async fn create_ticket(
        &self,
        request: CreateTicketRequest,
        context: &TenantContext,
        created_by: String,
    ) -> Result<Ticket> {
        // Validate tenant context
        if context.tenant_id != request.tenant_id {
            return Err(SmartTicketError::Unauthorized(
                "Tenant ID mismatch".to_string(),
            ));
        }

        let ticket = Ticket::new(request, created_by.clone());

        // Insert ticket into database
        let query = r#"
            INSERT INTO tickets (
                id, tenant_id, ticket_number, title, description,
                status, priority, severity, category_id, contact_id,
                assigned_to_id, created_by_id, resolved_at, closed_at,
                due_at, resolution, tags, custom_fields, external_reference, is_deleted,
                created_at, updated_at, ticket_type
            ) VALUES (
                $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
                $11, $12, $13, $14, $15, $16, $17, $18, $19,
                $20, $21, $22, $23, $24
            ) RETURNING *
        "#;

        let row = sqlx::query(query)
            .bind(ticket.id)
            .bind(ticket.tenant_id)
            .bind(format!("TICKET-{}", ticket.id)) // Generate ticket number
            .bind(&ticket.title)
            .bind(&ticket.description)
            .bind(ticket.status) // Use enum directly
            .bind(ticket.priority) // Use enum directly
            .bind(ticket.severity) // Use severity from ticket
            .bind(ticket.category_id)
            .bind(ticket.customer_id) // Maps to contact_id in database
            .bind(ticket.assigned_agent_id) // Maps to assigned_to_id in database
            .bind(ticket.created_by.as_ref().and_then(|s| Uuid::parse_str(s).ok()).unwrap_or_else(Uuid::new_v4)) // Convert Option<String> to UUID for created_by_id
            .bind(ticket.resolved_at)
            .bind(ticket.closed_at)
            .bind(ticket.due_date)
            .bind(&ticket.resolution)
            .bind(&ticket.tags) // Convert Vec<String> to PostgreSQL array directly
            .bind(&ticket.custom_fields) // Bind custom_fields as JSONB
            .bind(&ticket.external_reference) // Bind external_reference as TEXT
            .bind(ticket.is_deleted)
            .bind(ticket.created_at)
            .bind(ticket.updated_at)
            .bind(ticket.ticket_type) // Use enum directly
            .fetch_one(&self.pool)
            .await
            .map_err(|e| {
                error!("Failed to create ticket: {}", e);
                SmartTicketError::Database(e)
            })?;

        // Construct ticket from database row
        let result = Ticket {
            id: row.get("id"),
            tenant_id: row.get("tenant_id"),
            customer_id: row.get("contact_id"), // Map back from contact_id
            assigned_agent_id: row.get("assigned_to_id"), // Map back from assigned_to_id
            team_id: None, // No team_id column in database
            title: row.get("title"),
            description: row.get("description"),
            status: row.get("status"),
            priority: row.get("priority"),
            severity: row.get("severity"), // Get severity from database
            ticket_type: row.get("ticket_type"),
            category_id: row.get("category_id"),
            tags: row.get("tags"),
            created_at: row.get("created_at"),
            updated_at: row.get("updated_at"),
            due_date: row.get("due_at"), // Map back from due_at
            resolved_at: row.get("resolved_at"),
            closed_at: row.get("closed_at"),
            resolution: row.get("resolution"),
            satisfaction_rating: None, // No satisfaction_rating column in database
            external_reference: row.get("external_reference"), // Get external_reference from database
            custom_fields: row.get("custom_fields"), // Get custom_fields from database as JSONB
            is_deleted: row.get("is_deleted"),
            created_by: row.get("created_by"), // Get from text column
            updated_by: row.get("updated_by"), // Get from text column
        };

        // Create activity log
        self.log_activity(
            result.id,
            created_by,
            "Created".to_string(),
            None,
            None,
            Some(serde_json::json!({
                "title": ticket.title,
                "priority": ticket.priority,
                "type": ticket.ticket_type
            })),
        )
        .await?;

        info!(
            "Created ticket {} for tenant {}",
            result.id, result.tenant_id
        );
        Ok(result)
    }

    /// Get a ticket by ID
    #[instrument(skip(self))]
    pub async fn get_ticket(&self, ticket_id: Uuid, context: &TenantContext) -> Result<Ticket> {
        let query = r#"
            SELECT
                id, tenant_id,
                contact_id as customer_id,
                assigned_to_id as assigned_agent_id,
                title, description, status, priority, severity,
                ticket_type, category_id, tags, external_reference,
                due_at as due_date, resolved_at, closed_at, resolution,
                is_deleted, created_at, updated_at,
                created_by, updated_by, created_by_id,
                ticket_number, custom_fields
            FROM tickets
            WHERE id = $1 AND tenant_id = $2 AND is_deleted = false
        "#;

        let ticket = sqlx::query_as::<_, Ticket>(query)
            .bind(ticket_id)
            .bind(context.tenant_id)
            .fetch_optional(&self.pool)
            .await
            .map_err(|e| {
                error!("Failed to get ticket {}: {}", ticket_id, e);
                SmartTicketError::Database(e)
            })?
            .ok_or_else(|| SmartTicketError::not_found("Ticket", &ticket_id.to_string()))?;

        Ok(ticket)
    }

    /// Update ticket details
    #[instrument(skip(self))]
    pub async fn update_ticket(
        &self,
        request: UpdateTicketRequest,
        context: &TenantContext,
        updated_by: String,
    ) -> Result<Ticket> {
        // Get existing ticket
        let mut ticket = self.get_ticket(request.id, context).await?;

        // Store old values for activity logging
        let old_title = ticket.title.clone();
        let old_description = ticket.description.clone();
        let old_priority = ticket.priority;
        let old_category_id = ticket.category_id;

        // Update fields if provided
        if let Some(title) = request.title {
            ticket.title = title;
        }
        if let Some(description) = request.description {
            ticket.description = description;
        }
        if let Some(priority) = request.priority {
            ticket.priority = priority;
        }
        if let Some(category_id) = request.category_id {
            ticket.category_id = Some(category_id);
        }
        if let Some(tags) = request.tags {
            ticket.tags = tags;
        }
        if let Some(due_date) = request.due_date {
            ticket.due_date = Some(due_date);
        }
        if let Some(external_reference) = request.external_reference {
            ticket.external_reference = Some(external_reference);
        }

        ticket.updated_at = Utc::now();
        ticket.updated_by = Some(updated_by.clone());

        // Update in database
        let query = r#"
            UPDATE tickets SET
                title = $2, description = $3, priority = $4, category_id = $5,
                tags = $6, due_date = $7, external_reference = $8,
                updated_at = $9, updated_by = $10
            WHERE id = $1 AND tenant_id = $11 AND is_deleted = false
            RETURNING *
        "#;

        let updated_ticket = sqlx::query_as::<_, Ticket>(query)
            .bind(ticket.id)
            .bind(&ticket.title)
            .bind(&ticket.description)
            .bind(ticket.priority)
            .bind(ticket.category_id)
            .bind(&ticket.tags)
            .bind(ticket.due_date)
            .bind(&ticket.external_reference)
            .bind(ticket.updated_at)
            .bind(&ticket.updated_by)
            .bind(context.tenant_id)
            .fetch_one(&self.pool)
            .await
            .map_err(|e| {
                error!("Failed to update ticket {}: {}", request.id, e);
                SmartTicketError::Database(e)
            })?;

        // Log changes
        let mut changes = Vec::new();
        if old_title != updated_ticket.title {
            changes.push((
                "title".to_string(),
                Some(old_title),
                Some(updated_ticket.title.clone()),
            ));
        }
        if old_description != updated_ticket.description {
            changes.push((
                "description".to_string(),
                Some(old_description),
                Some(updated_ticket.description.clone()),
            ));
        }
        if old_priority != updated_ticket.priority {
            changes.push((
                "priority".to_string(),
                Some(old_priority.to_string()),
                Some(updated_ticket.priority.to_string()),
            ));
        }
        if old_category_id != updated_ticket.category_id {
            changes.push((
                "category_id".to_string(),
                old_category_id.map(|u| u.to_string()),
                updated_ticket.category_id.map(|u| u.to_string()),
            ));
        }

        for (field, old_val, new_val) in changes {
            self.log_activity(
                updated_ticket.id,
                updated_by.clone(),
                format!("Updated {}", field),
                old_val,
                new_val,
                None,
            )
            .await?;
        }

        info!(
            "Updated ticket {} for tenant {}",
            updated_ticket.id, updated_ticket.tenant_id
        );
        Ok(updated_ticket)
    }

    /// List tickets with filtering and pagination
    #[instrument(skip(self))]
    pub async fn list_tickets(
        &self,
        filters: TicketSearchFilters,
        context: &TenantContext,
    ) -> Result<TicketListResponse> {
        // Validate tenant context
        if context.tenant_id != filters.tenant_id {
            return Err(SmartTicketError::Unauthorized(
                "Tenant ID mismatch".to_string(),
            ));
        }

        // Simplified query builder - build with basic filters only for now
        let (query, count_query) = self.build_list_tickets_query(&filters);

        // Get total count first
        let count: i64 = sqlx::query_scalar(&count_query)
            .bind(filters.tenant_id)
            .fetch_one(&self.pool)
            .await
            .map_err(|e| {
                error!("Failed to count tickets: {}", e);
                SmartTicketError::Database(e)
            })?;

        // Execute main query with pagination
        let page_size = filters.page_size.unwrap_or(50).min(100);
        let offset = filters.page_token.and_then(|t| t.parse().ok()).unwrap_or(0);

        let tickets = sqlx::query_as::<_, Ticket>(&query)
            .bind(filters.tenant_id)
            .bind(page_size)
            .bind(offset)
            .fetch_all(&self.pool)
            .await
            .map_err(|e| {
                error!("Failed to list tickets: {}", e);
                SmartTicketError::Database(e)
            })?;

        // Calculate next page token
        let next_page_token = if tickets.len() == page_size as usize {
            Some((page_size as i32).to_string())
        } else {
            None
        };

        info!(
            "Listed {} tickets for tenant {}",
            tickets.len(),
            filters.tenant_id
        );
        Ok(TicketListResponse {
            tickets,
            total_count: count,
            next_page_token,
        })
    }

    /// Delete a ticket (soft delete)
    #[instrument(skip(self))]
    pub async fn delete_ticket(
        &self,
        ticket_id: Uuid,
        context: &TenantContext,
        deleted_by: String,
    ) -> Result<()> {
        // Get ticket to log deletion
        let ticket = self.get_ticket(ticket_id, context).await?;

        // Soft delete
        let query = r#"
            UPDATE tickets
            SET is_deleted = true, updated_at = $1, updated_by = $2
            WHERE id = $3 AND tenant_id = $4 AND is_deleted = false
        "#;

        let rows_affected = sqlx::query(query)
            .bind(Utc::now())
            .bind(&deleted_by)
            .bind(ticket_id)
            .bind(context.tenant_id)
            .execute(&self.pool)
            .await
            .map_err(|e| {
                error!("Failed to delete ticket {}: {}", ticket_id, e);
                SmartTicketError::Database(e)
            })?
            .rows_affected();

        if rows_affected == 0 {
            return Err(SmartTicketError::not_found(
                "Ticket",
                &ticket_id.to_string(),
            ));
        }

        // Log activity
        self.log_activity(
            ticket_id,
            deleted_by,
            "Deleted".to_string(),
            None,
            None,
            Some(serde_json::json!({
                "title": ticket.title,
                "status": ticket.status
            })),
        )
        .await?;

        info!(
            "Deleted ticket {} for tenant {}",
            ticket_id, context.tenant_id
        );
        Ok(())
    }

    /// Assign ticket to an agent
    #[instrument(skip(self))]
    pub async fn assign_ticket(
        &self,
        ticket_id: Uuid,
        agent_id: Uuid,
        team_id: Option<Uuid>,
        context: &TenantContext,
        assigned_by: String,
    ) -> Result<Ticket> {
        // Get current ticket
        let mut ticket = self.get_ticket(ticket_id, context).await?;

        // Check if ticket can be assigned
        if !ticket.can_be_assigned() {
            return Err(SmartTicketError::Validation(
                "Ticket cannot be assigned in current status".to_string(),
            ));
        }

        let old_agent_id = ticket.assigned_agent_id;
        let _old_team_id = ticket.team_id;

        // Update assignment
        ticket.assign_to_agent(agent_id, assigned_by.clone())?;
        if let Some(team_id) = team_id {
            ticket.team_id = Some(team_id);
        }

        // Update in database
        let query = r#"
            UPDATE tickets
            SET assigned_agent_id = $2, team_id = $3, status = $4,
                updated_at = $5, updated_by = $6
            WHERE id = $1 AND tenant_id = $7 AND is_deleted = false
            RETURNING *
        "#;

        let updated_ticket = sqlx::query_as::<_, Ticket>(query)
            .bind(ticket.id)
            .bind(ticket.assigned_agent_id)
            .bind(ticket.team_id)
            .bind(ticket.status)
            .bind(ticket.updated_at)
            .bind(&ticket.updated_by)
            .bind(context.tenant_id)
            .fetch_one(&self.pool)
            .await
            .map_err(|e| {
                error!("Failed to assign ticket {}: {}", ticket_id, e);
                SmartTicketError::Database(e)
            })?;

        // Log activity
        self.log_activity(
            ticket_id,
            assigned_by.clone(),
            "Assigned".to_string(),
            old_agent_id.map(|u| u.to_string()),
            Some(agent_id.to_string()),
            Some(serde_json::json!({
                "team_id": team_id
            })),
        )
        .await?;

        info!(
            "Assigned ticket {} to agent {} for tenant {}",
            ticket_id, agent_id, context.tenant_id
        );
        Ok(updated_ticket)
    }

    /// Change ticket status
    #[instrument(skip(self))]
    pub async fn change_ticket_status(
        &self,
        ticket_id: Uuid,
        new_status: TicketStatus,
        comment: Option<String>,
        context: &TenantContext,
        changed_by: String,
    ) -> Result<Ticket> {
        // Get current ticket
        let mut ticket = self.get_ticket(ticket_id, context).await?;

        let old_status = ticket.status;

        // Update status
        ticket.update_status(new_status, changed_by.clone())?;

        // Update in database
        let query = r#"
            UPDATE tickets
            SET status = $2, resolved_at = $3, closed_at = $4,
                updated_at = $5, updated_by = $6
            WHERE id = $1 AND tenant_id = $7 AND is_deleted = false
            RETURNING *
        "#;

        let updated_ticket = sqlx::query_as::<_, Ticket>(query)
            .bind(ticket.id)
            .bind(ticket.status)
            .bind(ticket.resolved_at)
            .bind(ticket.closed_at)
            .bind(ticket.updated_at)
            .bind(&ticket.updated_by)
            .bind(context.tenant_id)
            .fetch_one(&self.pool)
            .await
            .map_err(|e| {
                error!("Failed to change status for ticket {}: {}", ticket_id, e);
                SmartTicketError::Database(e)
            })?;

        // Log activity
        self.log_activity(
            ticket_id,
            changed_by.clone(),
            "Status Changed".to_string(),
            Some(old_status.to_string()),
            Some(new_status.to_string()),
            Some(serde_json::json!({
                "comment": comment
            })),
        )
        .await?;

        info!(
            "Changed status of ticket {} from {:?} to {:?} for tenant {}",
            ticket_id, old_status, new_status, context.tenant_id
        );
        Ok(updated_ticket)
    }

    /// Add comment to ticket
    #[instrument(skip(self))]
    pub async fn add_comment(
        &self,
        ticket_id: Uuid,
        author_id: String,
        author_name: String,
        author_email: String,
        content: String,
        comment_type: CommentType,
        is_internal: bool,
        context: &TenantContext,
        created_by: String,
    ) -> Result<TicketComment> {
        // Verify ticket exists
        self.get_ticket(ticket_id, context).await?;

        let comment = TicketComment::new(
            ticket_id,
            author_id.clone(),
            author_name.clone(),
            author_email.clone(),
            content.clone(),
            comment_type,
            is_internal,
            created_by.clone(),
        );

        // Insert comment
        let query = r#"
            INSERT INTO ticket_comments (
                id, ticket_id, author_id, author_name, author_email,
                content, comment_type, is_internal, created_at, updated_at,
                is_deleted, created_by, updated_by
            ) VALUES (
                $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
            ) RETURNING *
        "#;

        let inserted_comment = sqlx::query_as::<_, TicketComment>(query)
            .bind(comment.id)
            .bind(comment.ticket_id)
            .bind(&comment.author_id)
            .bind(&comment.author_name)
            .bind(&comment.author_email)
            .bind(&comment.content)
            .bind(comment.comment_type)
            .bind(comment.is_internal)
            .bind(comment.created_at)
            .bind(comment.updated_at)
            .bind(comment.is_deleted)
            .bind(&comment.created_by)
            .bind(&comment.updated_by)
            .fetch_one(&self.pool)
            .await
            .map_err(|e| {
                error!("Failed to add comment to ticket {}: {}", ticket_id, e);
                SmartTicketError::Database(e)
            })?;

        // Log activity
        self.log_activity(
            ticket_id,
            created_by.clone(),
            format!(
                "{} Added",
                if is_internal {
                    "Internal Comment"
                } else {
                    "Comment"
                }
            ),
            None,
            None,
            Some(serde_json::json!({
                "author": author_name,
                "type": comment_type,
                "internal": is_internal,
                "content_preview": content.chars().take(100).collect::<String>()
            })),
        )
        .await?;

        info!(
            "Added comment to ticket {} by {} for tenant {}",
            ticket_id, author_name, context.tenant_id
        );
        Ok(inserted_comment)
    }

    /// Get ticket comments
    #[instrument(skip(self))]
    pub async fn get_comments(
        &self,
        ticket_id: Uuid,
        context: &TenantContext,
        page_size: Option<i32>,
        page_token: Option<String>,
    ) -> Result<Vec<TicketComment>> {
        // Verify ticket exists
        self.get_ticket(ticket_id, context).await?;

        let page_size = page_size.unwrap_or(50).min(100);
        let offset = page_token.and_then(|t| t.parse().ok()).unwrap_or(0);

        let query = r#"
            SELECT * FROM ticket_comments
            WHERE ticket_id = $1 AND is_deleted = false
            ORDER BY created_at ASC
            LIMIT $2 OFFSET $3
        "#;

        let comments = sqlx::query_as::<_, TicketComment>(query)
            .bind(ticket_id)
            .bind(page_size)
            .bind(offset)
            .fetch_all(&self.pool)
            .await
            .map_err(|e| {
                error!("Failed to get comments for ticket {}: {}", ticket_id, e);
                SmartTicketError::Database(e)
            })?;

        info!(
            "Retrieved {} comments for ticket {}",
            comments.len(),
            ticket_id
        );
        Ok(comments)
    }

    /// Get ticket statistics
    #[instrument(skip(self))]
    pub async fn get_ticket_stats(&self, context: &TenantContext) -> Result<TicketStats> {
        let tenant_id = context.tenant_id;

        // Get basic counts
        let counts_query = r#"
            SELECT
                COUNT(*) as total_tickets,
                COUNT(*) FILTER (WHERE status = 'Open') as open_tickets,
                COUNT(*) FILTER (WHERE status = 'InProgress') as in_progress_tickets,
                COUNT(*) FILTER (WHERE status = 'Resolved') as resolved_tickets,
                COUNT(*) FILTER (WHERE status = 'Closed') as closed_tickets,
                COUNT(*) FILTER (WHERE is_deleted = false AND
                    due_at IS NOT NULL AND due_at < NOW() AND
                    status NOT IN ('Resolved', 'Closed')) as overdue_tickets
            FROM tickets
            WHERE tenant_id = $1 AND is_deleted = false
        "#;

        let counts_row = sqlx::query(counts_query)
            .bind(tenant_id)
            .fetch_one(&self.pool)
            .await
            .map_err(|e| {
                error!("Failed to get ticket counts: {}", e);
                SmartTicketError::Database(e)
            })?;

        let total_tickets: i64 = counts_row.get("total_tickets");
        let open_tickets: i64 = counts_row.get("open_tickets");
        let in_progress_tickets: i64 = counts_row.get("in_progress_tickets");
        let resolved_tickets: i64 = counts_row.get("resolved_tickets");
        let closed_tickets: i64 = counts_row.get("closed_tickets");
        let overdue_tickets: i64 = counts_row.get("overdue_tickets");

        // Get priority distribution
        let priority_query = r#"
            SELECT priority, COUNT(*) as count
            FROM tickets
            WHERE tenant_id = $1 AND is_deleted = false
            GROUP BY priority
            ORDER BY priority
        "#;

        let priority_rows = sqlx::query(priority_query)
            .bind(tenant_id)
            .fetch_all(&self.pool)
            .await
            .map_err(|e| {
                error!("Failed to get priority distribution: {}", e);
                SmartTicketError::Database(e)
            })?;

        let tickets_by_priority: Vec<TicketPriorityStats> = priority_rows
            .into_iter()
            .map(|row| {
                let priority: TicketPriority = row.get("priority");
                let count: i64 = row.get("count");
                TicketPriorityStats {
                    priority,
                    count,
                    percentage: (count as f64 / total_tickets as f64) * 100.0,
                }
            })
            .collect();

        // Get type distribution
        let type_query = r#"
            SELECT ticket_type, COUNT(*) as count
            FROM tickets
            WHERE tenant_id = $1 AND is_deleted = false
            GROUP BY ticket_type
            ORDER BY ticket_type
        "#;

        let type_rows = sqlx::query(type_query)
            .bind(tenant_id)
            .fetch_all(&self.pool)
            .await
            .map_err(|e| {
                error!("Failed to get type distribution: {}", e);
                SmartTicketError::Database(e)
            })?;

        let tickets_by_type: Vec<TicketTypeStats> = type_rows
            .into_iter()
            .map(|row| {
                let ticket_type: TicketType = row.get("ticket_type");
                let count: i64 = row.get("count");
                TicketTypeStats {
                    ticket_type,
                    count,
                    percentage: (count as f64 / total_tickets as f64) * 100.0,
                }
            })
            .collect();

        // Get average resolution and response times
        let time_query = r#"
            SELECT
                AVG(EXTRACT(EPOCH FROM (resolved_at - created_at)) / 60) as avg_resolution_minutes,
                AVG(satisfaction_rating) as avg_satisfaction
            FROM tickets
            WHERE tickets.tenant_id = $1
                AND tickets.is_deleted = false
                AND tickets.status IN ('Resolved', 'Closed')
        "#;

        let time_row = sqlx::query(time_query)
            .bind(tenant_id)
            .fetch_one(&self.pool)
            .await
            .map_err(|e| {
                error!("Failed to get time statistics: {}", e);
                SmartTicketError::Database(e)
            })?;

        let average_resolution_time_minutes: Option<f64> = time_row.get("avg_resolution_minutes");
        let average_response_time_minutes: Option<f64> = time_row.get("avg_response_minutes");
        let satisfaction_score_average: Option<f64> = time_row.get("avg_satisfaction");

        info!(
            "Retrieved ticket statistics for tenant {}",
            context.tenant_id
        );
        Ok(TicketStats {
            total_tickets,
            open_tickets,
            in_progress_tickets,
            resolved_tickets,
            closed_tickets,
            overdue_tickets,
            tickets_by_priority,
            tickets_by_type,
            tickets_by_category: vec![], // TODO: Implement category stats
            average_resolution_time_minutes,
            average_response_time_minutes,
            satisfaction_score_average,
        })
    }

    /// Search tickets
    #[instrument(skip(self))]
    pub async fn search_tickets(
        &self,
        tenant_id: Uuid,
        query: &str,
        filters: Vec<String>,
        page_size: Option<i32>,
        page_token: Option<String>,
        sort_by: Option<String>,
        sort_desc: Option<bool>,
    ) -> Result<Vec<Ticket>> {
        let search_pattern = format!("%{}%", query);
        let mut sql_query = String::from(
            r#"
            SELECT DISTINCT
                tickets.id, tickets.tenant_id,
                tickets.contact_id as customer_id,
                tickets.assigned_to_id as assigned_agent_id,
                tickets.title, tickets.description, tickets.status, tickets.priority, tickets.severity,
                tickets.ticket_type, tickets.category_id, tickets.tags, tickets.external_reference,
                tickets.due_at as due_date, tickets.resolved_at, tickets.closed_at, tickets.resolution,
                tickets.is_deleted, tickets.created_at, tickets.updated_at,
                tickets.created_by, tickets.updated_by, tickets.created_by_id,
                tickets.ticket_number, tickets.custom_fields
            FROM tickets
            LEFT JOIN ticket_comments ON tickets.id = ticket_comments.ticket_id
            WHERE tickets.tenant_id = $1
                AND tickets.is_deleted = false
                AND (tickets.title ILIKE $2
                    OR tickets.description ILIKE $2
                    OR tickets.external_reference ILIKE $2
                    OR ticket_comments.content ILIKE $2)
        "#,
        );

        // Simplified search - ignore additional filters for now to avoid trait object issues
        // TODO: Implement proper query builder for complex search

        // Add ordering
        let default_field = "created_at".to_string();
        let sort_field = sort_by.as_ref().unwrap_or(&default_field);
        let order_desc = sort_desc.unwrap_or(true);
        sql_query.push_str(&format!(
            " ORDER BY tickets.{} {}",
            sort_field,
            if order_desc { "DESC" } else { "ASC" }
        ));

        // Add pagination
        let page_size = page_size.unwrap_or(50).min(100);
        let offset = page_token.and_then(|t| t.parse().ok()).unwrap_or(0);

        sql_query.push_str(&format!(" LIMIT ${} OFFSET ${}", 3, 4));

        // Execute search query
        let tickets = sqlx::query_as::<_, Ticket>(&sql_query)
            .bind(tenant_id)
            .bind(&search_pattern)
            .bind(page_size)
            .bind(offset)
            .fetch_all(&self.pool)
            .await
            .map_err(|e| {
                error!("Failed to search tickets: {}", e);
                SmartTicketError::Database(e)
            })?;

        info!(
            "Found {} tickets matching query '{}' for tenant {}",
            tickets.len(),
            query,
            tenant_id
        );
        Ok(tickets)
    }

    /// Log ticket activity
    #[instrument(skip(self))]
    async fn log_activity(
        &self,
        ticket_id: Uuid,
        actor_id: String,
        action: String,
        old_value: Option<String>,
        new_value: Option<String>,
        details: Option<serde_json::Value>,
    ) -> Result<()> {
        let query = r#"
            INSERT INTO ticket_activities (
                id, ticket_id, actor_id, actor_name, action,
                old_value, new_value, details, created_at, created_by
            ) VALUES (
                $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
            )
        "#;

        // Get actor name from users table (simplified for now)
        let actor_name = actor_id.clone(); // In real implementation, would fetch from users table

        sqlx::query(query)
            .bind(Uuid::new_v4())
            .bind(ticket_id)
            .bind(&actor_id)
            .bind(actor_name)
            .bind(&action)
            .bind(&old_value)
            .bind(&new_value)
            .bind(&details)
            .bind(Utc::now())
            .bind(&actor_id)
            .execute(&self.pool)
            .await
            .map_err(|e| {
                error!("Failed to log activity for ticket {}: {}", ticket_id, e);
                SmartTicketError::Database(e)
            })?;

        Ok(())
    }

    /// Build query for listing tickets (simplified version to avoid trait object issues)
    fn build_list_tickets_query(&self, filters: &TicketSearchFilters) -> (String, String) {
        let base_query = r#"
            SELECT
                id, tenant_id,
                contact_id as customer_id,
                assigned_to_id as assigned_agent_id,
                title, description, status, priority, severity,
                ticket_type, category_id, tags, external_reference,
                due_at as due_date, resolved_at, closed_at, resolution,
                is_deleted, created_at, updated_at,
                created_by, updated_by, created_by_id,
                ticket_number, custom_fields
            FROM tickets
            WHERE tenant_id = $1 AND is_deleted = false
        "#.to_string();
        let base_count =
            "SELECT COUNT(*) FROM tickets WHERE tenant_id = $1 AND is_deleted = false".to_string();

        let mut query_conditions = Vec::new();

        // Add basic filter conditions
        if let Some(_customer_id) = filters.customer_id {
            query_conditions.push("AND contact_id = $2".to_string());
        }
        if let Some(_assigned_agent_id) = filters.assigned_agent_id {
            query_conditions.push("AND assigned_to_id = $2".to_string());
        }
        if let Some(_status) = &filters.status {
            query_conditions.push("AND status = $2".to_string());
        }
        if let Some(_priority) = &filters.priority {
            query_conditions.push("AND priority = $2".to_string());
        }

        // For now, ignore complex filters to avoid trait object issues
        // TODO: Implement proper query builder for complex filters

        // Add ordering
        let default_order_by = "created_at".to_string();
        let order_by = filters.order_by.as_ref().unwrap_or(&default_order_by);
        let order_desc = filters.order_desc.unwrap_or(true);
        let order_clause = format!(
            " ORDER BY {} {}",
            order_by,
            if order_desc { "DESC" } else { "ASC" }
        );

        // Add pagination
        let limit_clause = " LIMIT $2 OFFSET $3".to_string();

        let query = format!(
            "{} {} {}",
            base_query,
            query_conditions.join(" "),
            order_clause + &limit_clause
        );
        let count_query = format!("{} {}", base_count, query_conditions.join(" "));

        (query, count_query)
    }
}
