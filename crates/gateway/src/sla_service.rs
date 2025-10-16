use tonic::{Request, Response, Status};
use uuid::Uuid;
use sqlx::{PgPool, Row};
use std::sync::Arc;
use chrono::{DateTime, Utc, Duration};

use smartticket_shared_database::{TenantContext, TicketPriority, TicketSeverity};

use crate::smartticket_v1::{
    sla_service_server::SlaService,
    CreateSlaPolicyRequest, CreateSlaPolicyResponse,
    GetSlaPolicyRequest, GetSlaPolicyResponse,
    UpdateSlaPolicyRequest, UpdateSlaPolicyResponse,
    ListSlaPoliciesRequest, ListSlaPoliciesResponse,
    DeleteSlaPolicyRequest, DeleteSlaPolicyResponse,
    GetSlaMetricsRequest, GetSlaMetricsResponse,
    GetSlaDashboardRequest, GetSlaDashboardResponse,
    GetSlaBreachesRequest, GetSlaBreachesResponse,
    UpdateSlaMetricsRequest, UpdateSlaMetricsResponse,
    SlaPolicy, SlaMetrics, SlaDashboard, SlaBreachAlert,
    Response as ApiResponse, PaginationResponse, RequestMetadata,
    SlaDashboardItem, SlaSummary, SlaTrend,
};

use crate::auth_middleware::extract_request_metadata;

#[derive(Debug, Clone)]
pub struct SlaGrpcService {
    pool: Arc<PgPool>,
}

impl SlaGrpcService {
    pub fn new(pool: Arc<PgPool>) -> Self {
        Self { pool }
    }

    fn success_response(message: &str, request_id: &str) -> ApiResponse {
        ApiResponse {
            success: true,
            message: message.to_string(),
            data: None,
            errors: vec![],
            request_id: request_id.to_string(),
        }
    }

    fn error_response(message: &str, request_id: &str) -> ApiResponse {
        ApiResponse {
            success: false,
            message: message.to_string(),
            data: None,
            errors: vec![],
            request_id: request_id.to_string(),
        }
    }

    fn map_priority_enum_to_string(priority: i32) -> Result<TicketPriority, Status> {
        match priority {
            1 => Ok(TicketPriority::Low),
            2 => Ok(TicketPriority::Normal),
            3 => Ok(TicketPriority::High),
            4 => Ok(TicketPriority::Critical),
            _ => Err(Status::invalid_argument("Invalid priority value")),
        }
    }

    fn map_string_to_priority_enum(priority: &str) -> i32 {
        match priority {
            "Low" => 1,
            "Normal" => 2,
            "High" => 3,
            "Critical" => 4,
            _ => 2, // Default to Normal
        }
    }

    fn map_severity_enum_to_string(severity: i32) -> Result<TicketSeverity, Status> {
        match severity {
            1 => Ok(TicketSeverity::Low),
            2 => Ok(TicketSeverity::Medium),
            3 => Ok(TicketSeverity::High),
            4 => Ok(TicketSeverity::Critical),
            _ => Err(Status::invalid_argument("Invalid severity value")),
        }
    }

    fn map_string_to_severity_enum(severity: &str) -> i32 {
        match severity {
            "Low" => 1,
            "Medium" => 2,
            "High" => 3,
            "Critical" => 4,
            _ => 2, // Default to Medium
        }
    }

    fn ticket_priority_to_enum(priority: &TicketPriority) -> i32 {
        match priority {
            TicketPriority::Low => 1,
            TicketPriority::Normal => 2,
            TicketPriority::High => 3,
            TicketPriority::Critical => 4,
        }
    }

    fn ticket_severity_to_enum(severity: &TicketSeverity) -> i32 {
        match severity {
            TicketSeverity::Low => 1,
            TicketSeverity::Medium => 2,
            TicketSeverity::High => 3,
            TicketSeverity::Critical => 4,
        }
    }

    fn ticket_priority_to_string(priority: &TicketPriority) -> String {
        match priority {
            TicketPriority::Low => "Low".to_string(),
            TicketPriority::Normal => "Normal".to_string(),
            TicketPriority::High => "High".to_string(),
            TicketPriority::Critical => "Critical".to_string(),
        }
    }

    fn chrono_to_prost_timestamp(dt: DateTime<Utc>) -> prost_types::Timestamp {
        prost_types::Timestamp {
            seconds: dt.timestamp(),
            nanos: dt.timestamp_subsec_nanos() as i32,
        }
    }

    fn calculate_business_hours_due(
        start_time: DateTime<Utc>,
        minutes: i32,
        timezone: &str,
    ) -> Result<DateTime<Utc>, Status> {
        // Simple implementation - in production this would handle business hours properly
        match timezone {
            "UTC" | "" => Ok(start_time + Duration::minutes(minutes as i64)),
            _ => {
                // For other timezones, convert to UTC (simplified)
                Ok(start_time + Duration::minutes(minutes as i64))
            }
        }
    }

    async fn extract_tenant_context(&self, metadata: &tonic::metadata::MetadataMap) -> Result<TenantContext, Status> {
        // Extract tenant and user IDs from metadata headers
        let tenant_id = metadata
            .get("x-tenant-id")
            .and_then(|value| value.to_str().ok())
            .and_then(|s| s.parse::<Uuid>().ok())
            .ok_or_else(|| Status::unauthenticated("Missing or invalid tenant ID"))?;

        let user_id = metadata
            .get("x-user-id")
            .and_then(|value| value.to_str().ok())
            .and_then(|s| s.parse::<Uuid>().ok())
            .ok_or_else(|| Status::unauthenticated("Missing or invalid user ID"))?;

        // Get user role from database
        let user_role_query = "SELECT role::text FROM users WHERE id = $1 AND tenant_id = $2";
        let user_role: String = sqlx::query(user_role_query)
            .bind(user_id)
            .bind(tenant_id)
            .fetch_one(&*self.pool)
            .await
            .map_err(|_| Status::unauthenticated("User not found"))?
            .get("role");

        Ok(TenantContext {
            tenant_id,
            user_id,
            user_role,
        })
    }
}

#[tonic::async_trait]
impl SlaService for SlaGrpcService {
    async fn create_sla_policy(
        &self,
        request: Request<CreateSlaPolicyRequest>,
    ) -> Result<Response<CreateSlaPolicyResponse>, Status> {
        let metadata = request.metadata();
        let tenant_context = self.extract_tenant_context(metadata).await?;
        let req = request.into_inner();

        // Validate required fields
        if req.name.is_empty() {
            return Err(Status::invalid_argument("Policy name is required"));
        }
        if req.response_time_minutes <= 0 {
            return Err(Status::invalid_argument("Response time must be positive"));
        }
        if req.resolution_time_minutes <= 0 {
            return Err(Status::invalid_argument("Resolution time must be positive"));
        }
        if req.resolution_time_minutes <= req.response_time_minutes {
            return Err(Status::invalid_argument("Resolution time must be greater than response time"));
        }

        let policy_id = Uuid::new_v4();
        let now = Utc::now();

        // Validate and convert enum values
        let priority_val = Self::map_priority_enum_to_string(req.priority)?;
        let severity_val = Self::map_severity_enum_to_string(req.severity)?;

        // Check for duplicate policy name within tenant
        let duplicate_check = sqlx::query(
            "SELECT id FROM sla_policies WHERE tenant_id = $1 AND name = $2 AND is_active = true"
        )
        .bind(tenant_context.tenant_id)
        .bind(&req.name)
        .fetch_optional(&*self.pool)
        .await
        .map_err(|e| Status::internal(format!("Database error: {}", e)))?;

        if duplicate_check.is_some() {
            return Err(Status::already_exists("SLA policy with this name already exists"));
        }

        let query = r#"
            INSERT INTO sla_policies (
                id, tenant_id, name, description, priority, severity,
                response_time_minutes, resolution_time_minutes, business_hours_only,
                timezone, is_active, created_at, updated_at
            ) VALUES (
                $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
            ) RETURNING id, created_at, updated_at
        "#;

        let row = sqlx::query(query)
            .bind(policy_id)
            .bind(tenant_context.tenant_id)
            .bind(&req.name)
            .bind(&req.description)
            .bind(&priority_val)
            .bind(&severity_val)
            .bind(req.response_time_minutes)
            .bind(req.resolution_time_minutes)
            .bind(req.business_hours_only)
            .bind(&req.timezone)
            .bind(true) // is_active
            .bind(now)
            .bind(now)
            .fetch_one(&*self.pool)
            .await
            .map_err(|e| Status::internal(format!("Failed to create SLA policy: {}", e)))?;

        let created_at: DateTime<Utc> = row.get("created_at");
        let updated_at: DateTime<Utc> = row.get("updated_at");

        let policy = SlaPolicy {
            id: policy_id.to_string(),
            tenant_id: tenant_context.tenant_id.to_string(),
            name: req.name,
            description: req.description,
            priority: req.priority,
            severity: req.severity,
            response_time_minutes: req.response_time_minutes,
            resolution_time_minutes: req.resolution_time_minutes,
            business_hours_only: req.business_hours_only,
            timezone: req.timezone,
            is_active: true,
            created_at: Some(Self::chrono_to_prost_timestamp(created_at)),
            updated_at: Some(Self::chrono_to_prost_timestamp(updated_at)),
        };

        let response = CreateSlaPolicyResponse {
            response: Some(Self::success_response(
                "SLA policy created successfully",
                &req.metadata.as_ref().map(|m| m.request_id.clone()).unwrap_or_default()
            )),
            policy: Some(policy),
        };

        Ok(Response::new(response))
    }

    async fn get_sla_policy(
        &self,
        request: Request<GetSlaPolicyRequest>,
    ) -> Result<Response<GetSlaPolicyResponse>, Status> {
        let metadata = request.metadata();
        let tenant_context = self.extract_tenant_context(metadata).await?;
        let req = request.into_inner();

        let policy_id = Uuid::parse_str(&req.policy_id)
            .map_err(|_| Status::invalid_argument("Invalid policy ID"))?;

        let query = r#"
            SELECT * FROM sla_policies
            WHERE id = $1 AND tenant_id = $2
        "#;

        let row = sqlx::query(query)
            .bind(policy_id)
            .bind(tenant_context.tenant_id)
            .fetch_optional(&*self.pool)
            .await
            .map_err(|e| Status::internal(format!("Database error: {}", e)))?
            .ok_or_else(|| Status::not_found("SLA policy not found"))?;

        let created_at: DateTime<Utc> = row.get("created_at");
        let updated_at: DateTime<Utc> = row.get("updated_at");

        let policy = SlaPolicy {
            id: row.get::<Uuid, _>("id").to_string(),
            tenant_id: row.get::<Uuid, _>("tenant_id").to_string(),
            name: row.get("name"),
            description: row.get("description"),
            priority: Self::ticket_priority_to_enum(&row.get::<TicketPriority, _>("priority")),
            severity: Self::ticket_severity_to_enum(&row.get::<TicketSeverity, _>("severity")),
            response_time_minutes: row.get("response_time_minutes"),
            resolution_time_minutes: row.get("resolution_time_minutes"),
            business_hours_only: row.get("business_hours_only"),
            timezone: row.get("timezone"),
            is_active: row.get("is_active"),
            created_at: Some(Self::chrono_to_prost_timestamp(created_at)),
            updated_at: Some(Self::chrono_to_prost_timestamp(updated_at)),
        };

        let response = GetSlaPolicyResponse {
            response: Some(Self::success_response(
                "SLA policy retrieved successfully",
                &req.metadata.as_ref().map(|m| m.request_id.clone()).unwrap_or_default()
            )),
            policy: Some(policy),
        };

        Ok(Response::new(response))
    }

    async fn update_sla_policy(
        &self,
        request: Request<UpdateSlaPolicyRequest>,
    ) -> Result<Response<UpdateSlaPolicyResponse>, Status> {
        let metadata = request.metadata();
        let tenant_context = self.extract_tenant_context(metadata).await?;
        let req = request.into_inner();

        let policy_id = Uuid::parse_str(&req.policy_id)
            .map_err(|_| Status::invalid_argument("Invalid policy ID"))?;

        let now = Utc::now();

        // Check if policy exists and belongs to tenant
        let check_query = "SELECT id FROM sla_policies WHERE id = $1 AND tenant_id = $2";
        let exists = sqlx::query(check_query)
            .bind(policy_id)
            .bind(tenant_context.tenant_id)
            .fetch_optional(&*self.pool)
            .await
            .map_err(|e| Status::internal(format!("Database error: {}", e)))?;

        if exists.is_none() {
            return Err(Status::not_found("SLA policy not found"));
        }

        // Check for duplicate name if changing
        if !req.name.is_empty() {
            let duplicate_check = sqlx::query(
                "SELECT id FROM sla_policies WHERE tenant_id = $1 AND name = $2 AND id != $3 AND is_active = true"
            )
            .bind(tenant_context.tenant_id)
            .bind(&req.name)
            .bind(policy_id)
            .fetch_optional(&*self.pool)
            .await
            .map_err(|e| Status::internal(format!("Database error: {}", e)))?;

            if duplicate_check.is_some() {
                return Err(Status::already_exists("SLA policy with this name already exists"));
            }
        }

        // Use a simpler approach - build the query with fixed parameter positions
        let mut set_clauses = Vec::new();
        let mut param_index = 1;

        if !req.name.is_empty() {
            set_clauses.push(format!("name = ${}", param_index));
            param_index += 1;
        }
        if req.response_time_minutes > 0 {
            set_clauses.push(format!("response_time_minutes = ${}", param_index));
            param_index += 1;
        }
        if req.resolution_time_minutes > 0 {
            set_clauses.push(format!("resolution_time_minutes = ${}", param_index));
            param_index += 1;
        }
        if req.business_hours_only {
            set_clauses.push(format!("business_hours_only = ${}", param_index));
            param_index += 1;
        }
        if !req.timezone.is_empty() {
            set_clauses.push(format!("timezone = ${}", param_index));
            param_index += 1;
        }

        if set_clauses.is_empty() {
            return Err(Status::invalid_argument("No fields to update"));
        }

        // Always update updated_at
        set_clauses.push(format!("updated_at = ${}", param_index));
        let id_param = param_index + 1;
        let tenant_param = param_index + 2;

        let query = format!(
            "UPDATE sla_policies SET {} WHERE id = ${} AND tenant_id = ${} RETURNING *",
            set_clauses.join(", "),
            id_param,
            tenant_param
        );

        // Debug: Print the query to see what's being generated
        eprintln!("DEBUG: Generated query: {}", query);
        eprintln!("DEBUG: Set clauses: {:?}", set_clauses);

        let mut query_builder = sqlx::query(&query);

        // Bind parameters in the same order as set_clauses
        if !req.name.is_empty() {
            query_builder = query_builder.bind(&req.name);
        }
        if req.response_time_minutes > 0 {
            query_builder = query_builder.bind(req.response_time_minutes);
        }
        if req.resolution_time_minutes > 0 {
            query_builder = query_builder.bind(req.resolution_time_minutes);
        }
        if req.business_hours_only {
            query_builder = query_builder.bind(req.business_hours_only);
        }
        if !req.timezone.is_empty() {
            query_builder = query_builder.bind(&req.timezone);
        }

        // Bind the always-updated fields
        query_builder = query_builder
            .bind(now)  // updated_at
            .bind(policy_id)  // WHERE id
            .bind(tenant_context.tenant_id);  // WHERE tenant_id

        let row = query_builder
            .fetch_one(&*self.pool)
            .await
            .map_err(|e| Status::internal(format!("Failed to update SLA policy: {}", e)))?;

        let created_at: DateTime<Utc> = row.get("created_at");
        let updated_at: DateTime<Utc> = row.get("updated_at");

        // Get enum types properly from database
        let priority_db: TicketPriority = row.get("priority");
        let severity_db: TicketSeverity = row.get("severity");

        let policy = SlaPolicy {
            id: row.get::<Uuid, _>("id").to_string(),
            tenant_id: row.get::<Uuid, _>("tenant_id").to_string(),
            name: row.get("name"),
            description: row.get("description"),
            priority: Self::ticket_priority_to_enum(&priority_db),
            severity: Self::ticket_severity_to_enum(&severity_db),
            response_time_minutes: row.get("response_time_minutes"),
            resolution_time_minutes: row.get("resolution_time_minutes"),
            business_hours_only: row.get("business_hours_only"),
            timezone: row.get("timezone"),
            is_active: row.get("is_active"),
            created_at: Some(Self::chrono_to_prost_timestamp(created_at)),
            updated_at: Some(Self::chrono_to_prost_timestamp(updated_at)),
        };

        let response = UpdateSlaPolicyResponse {
            response: Some(Self::success_response(
                "SLA policy updated successfully",
                &req.metadata.as_ref().map(|m| m.request_id.clone()).unwrap_or_default()
            )),
            policy: Some(policy),
        };

        Ok(Response::new(response))
    }

    async fn list_sla_policies(
        &self,
        request: Request<ListSlaPoliciesRequest>,
    ) -> Result<Response<ListSlaPoliciesResponse>, Status> {
        let metadata = request.metadata();
        let tenant_context = self.extract_tenant_context(metadata).await?;
        let req = request.into_inner();

        let page_size = req.pagination.as_ref()
            .map(|p| if p.page_size > 0 && p.page_size <= 100 { p.page_size } else { 20 })
            .unwrap_or(20);

        let page_token = req.pagination.as_ref()
            .and_then(|p| Some(p.page_token.clone()))
            .and_then(|token| token.parse::<i64>().ok())
            .unwrap_or(0);

        // Build WHERE clause
        let mut where_conditions = vec!["tenant_id = $1".to_string()];
        let mut bind_count = 2;

        if req.is_active {
            where_conditions.push(format!("is_active = ${}", bind_count));
            bind_count += 1;
        }

        if !req.priorities.is_empty() {
            // For simplicity, only handle first priority in list
            if let Some(priority_filter) = req.priorities.first() {
                where_conditions.push(format!("priority = ${}", bind_count));
                bind_count += 1;
            }
        }

        if !req.severities.is_empty() {
            // For simplicity, only handle first severity in list
            if let Some(severity_filter) = req.severities.first() {
                where_conditions.push(format!("severity = ${}", bind_count));
                bind_count += 1;
            }
        }

        let where_clause = where_conditions.join(" AND ");

        // Count query
        let count_query = format!("SELECT COUNT(*) as total FROM sla_policies WHERE {}", where_clause);
        let mut count_query_builder = sqlx::query(&count_query).bind(tenant_context.tenant_id);

        if req.is_active {
            count_query_builder = count_query_builder.bind(true);
        }
        if let Some(priority_filter) = req.priorities.first() {
            let priority_val = Self::map_priority_enum_to_string(*priority_filter)?;
            count_query_builder = count_query_builder.bind(priority_val);
        }
        if let Some(severity_filter) = req.severities.first() {
            let severity_val = Self::map_severity_enum_to_string(*severity_filter)?;
            count_query_builder = count_query_builder.bind(severity_val);
        }

        let total_count: i64 = count_query_builder
            .fetch_one(&*self.pool)
            .await
            .map_err(|e| Status::internal(format!("Failed to count SLA policies: {}", e)))?
            .get("total");

        // Main query
        let main_query = format!(
            "SELECT * FROM sla_policies WHERE {} ORDER BY created_at DESC LIMIT ${} OFFSET ${}",
            where_clause, bind_count, bind_count + 1
        );

        let mut query_builder = sqlx::query(&main_query).bind(tenant_context.tenant_id);

        if req.is_active {
            query_builder = query_builder.bind(true);
        }
        if let Some(priority_filter) = req.priorities.first() {
            let priority_val = Self::map_priority_enum_to_string(*priority_filter)?;
            query_builder = query_builder.bind(priority_val);
        }
        if let Some(severity_filter) = req.severities.first() {
            let severity_val = Self::map_severity_enum_to_string(*severity_filter)?;
            query_builder = query_builder.bind(severity_val);
        }

        query_builder = query_builder
            .bind(page_size as i64)
            .bind(page_token);

        let rows = query_builder
            .fetch_all(&*self.pool)
            .await
            .map_err(|e| Status::internal(format!("Failed to list SLA policies: {}", e)))?;

        let policies: Vec<SlaPolicy> = rows
            .into_iter()
            .map(|row| {
                let created_at: DateTime<Utc> = row.get("created_at");
                let updated_at: DateTime<Utc> = row.get("updated_at");

                SlaPolicy {
                    id: row.get::<Uuid, _>("id").to_string(),
                    tenant_id: row.get::<Uuid, _>("tenant_id").to_string(),
                    name: row.get("name"),
                    description: row.get("description"),
                    priority: Self::ticket_priority_to_enum(&row.get::<TicketPriority, _>("priority")),
                    severity: Self::ticket_severity_to_enum(&row.get::<TicketSeverity, _>("severity")),
                    response_time_minutes: row.get("response_time_minutes"),
                    resolution_time_minutes: row.get("resolution_time_minutes"),
                    business_hours_only: row.get("business_hours_only"),
                    timezone: row.get("timezone"),
                    is_active: row.get("is_active"),
                    created_at: Some(Self::chrono_to_prost_timestamp(created_at)),
                    updated_at: Some(Self::chrono_to_prost_timestamp(updated_at)),
                }
            })
            .collect();

        let next_page_token = if (page_token + page_size as i64) < total_count {
            (page_token + page_size as i64).to_string()
        } else {
            String::new()
        };

        let pagination = PaginationResponse {
            total_count: total_count as i32,
            page_size,
            next_page_token: if (page_token + page_size as i64) < total_count {
                (page_token + page_size as i64).to_string()
            } else {
                String::new()
            },
            prev_page_token: if page_token > 0 { (page_token - page_size as i64).to_string() } else { String::new() },
        };

        let response = ListSlaPoliciesResponse {
            response: Some(Self::success_response(
                "SLA policies listed successfully",
                &req.metadata.as_ref().map(|m| m.request_id.clone()).unwrap_or_default()
            )),
            policies,
            pagination: Some(pagination),
        };

        Ok(Response::new(response))
    }

    async fn delete_sla_policy(
        &self,
        request: Request<DeleteSlaPolicyRequest>,
    ) -> Result<Response<DeleteSlaPolicyResponse>, Status> {
        let metadata = request.metadata();
        let tenant_context = self.extract_tenant_context(metadata).await?;
        let req = request.into_inner();

        let policy_id = Uuid::parse_str(&req.policy_id)
            .map_err(|_| Status::invalid_argument("Invalid policy ID"))?;

        // Check if policy exists and belongs to tenant
        let check_query = "SELECT id FROM sla_policies WHERE id = $1 AND tenant_id = $2";
        let exists = sqlx::query(check_query)
            .bind(policy_id)
            .bind(tenant_context.tenant_id)
            .fetch_optional(&*self.pool)
            .await
            .map_err(|e| Status::internal(format!("Database error: {}", e)))?;

        if exists.is_none() {
            return Err(Status::not_found("SLA policy not found"));
        }

        // Check if policy is being used by active tickets
        let usage_check = sqlx::query(
            "SELECT COUNT(*) as count FROM ticket_sla ts
             JOIN tickets t ON ts.ticket_id = t.id
             WHERE ts.sla_policy_id = $1 AND t.status NOT IN ('Resolved', 'Closed')"
        )
        .bind(policy_id)
        .fetch_one(&*self.pool)
        .await
        .map_err(|e| Status::internal(format!("Database error: {}", e)))?;

        let active_usage: i64 = usage_check.get("count");
        if active_usage > 0 {
            return Err(Status::failed_precondition(
                format!("Cannot delete SLA policy - {} active tickets are using this policy", active_usage)
            ));
        }

        // Soft delete by setting is_active = false
        let delete_query = "UPDATE sla_policies SET is_active = false, updated_at = $1 WHERE id = $2 AND tenant_id = $3";

        let result = sqlx::query(delete_query)
            .bind(Utc::now())
            .bind(policy_id)
            .bind(tenant_context.tenant_id)
            .execute(&*self.pool)
            .await
            .map_err(|e| Status::internal(format!("Failed to delete SLA policy: {}", e)))?;

        if result.rows_affected() == 0 {
            return Err(Status::not_found("SLA policy not found"));
        }

        let response = DeleteSlaPolicyResponse {
            response: Some(Self::success_response(
                "SLA policy deleted successfully",
                &req.metadata.as_ref().map(|m| m.request_id.clone()).unwrap_or_default()
            )),
        };

        Ok(Response::new(response))
    }

    async fn get_sla_metrics(
        &self,
        request: Request<GetSlaMetricsRequest>,
    ) -> Result<Response<GetSlaMetricsResponse>, Status> {
        let metadata = request.metadata();
        let tenant_context = self.extract_tenant_context(metadata).await?;
        let req = request.into_inner();

        let ticket_id = Uuid::parse_str(&req.ticket_id)
            .map_err(|_| Status::invalid_argument("Invalid ticket ID"))?;

        // Verify ticket belongs to tenant
        let ticket_check = sqlx::query(
            "SELECT id, created_at FROM tickets WHERE id = $1 AND tenant_id = $2"
        )
        .bind(ticket_id)
        .bind(tenant_context.tenant_id)
        .fetch_optional(&*self.pool)
        .await
        .map_err(|e| Status::internal(format!("Database error: {}", e)))?;

        let (ticket_id, ticket_created_at) = match ticket_check {
            Some(row) => (row.get::<Uuid, _>("id"), row.get::<DateTime<Utc>, _>("created_at")),
            None => return Err(Status::not_found("Ticket not found")),
        };

        // Get SLA metrics for the ticket
        let metrics_query = r#"
            SELECT ts.*, sp.name as policy_name, sp.priority, sp.severity,
                   sp.response_time_minutes, sp.resolution_time_minutes, sp.business_hours_only
            FROM ticket_sla ts
            JOIN sla_policies sp ON ts.sla_policy_id = sp.id
            WHERE ts.ticket_id = $1
        "#;

        let metrics_row = sqlx::query(metrics_query)
            .bind(ticket_id)
            .fetch_optional(&*self.pool)
            .await
            .map_err(|e| Status::internal(format!("Database error: {}", e)))?;

        let metrics = if let Some(row) = metrics_row {
            let response_due: Option<DateTime<Utc>> = row.get("response_due");
            let resolution_due: Option<DateTime<Utc>> = row.get("resolution_due");
            let first_response: Option<DateTime<Utc>> = row.get("first_response_at");
            let resolved_at: Option<DateTime<Utc>> = row.get("resolved_at");

            // Calculate actual response and resolution times
            let actual_response_time = if let Some(response_time) = first_response {
                Some((response_time - ticket_created_at).num_minutes() as i32)
            } else {
                None
            };

            let actual_resolution_time = if let Some(resolved_time) = resolved_at {
                Some((resolved_time - ticket_created_at).num_minutes() as i32)
            } else {
                None
            };

            // Check for breaches
            let now = Utc::now();
            let response_breached = if let Some(due_time) = response_due {
                due_time < now && first_response.is_none()
            } else {
                false
            };

            let resolution_breached = if let Some(due_time) = resolution_due {
                due_time < now && resolved_at.is_none()
            } else {
                false
            };

            Some(SlaMetrics {
                id: row.get::<Uuid, _>("id").to_string(),
                tenant_id: tenant_context.tenant_id.to_string(),
                ticket_id: ticket_id.to_string(),
                sla_policy_id: row.get::<Uuid, _>("sla_policy_id").to_string(),
                response_due_at: response_due.map(Self::chrono_to_prost_timestamp),
                resolution_due_at: resolution_due.map(Self::chrono_to_prost_timestamp),
                first_response_at: first_response.map(Self::chrono_to_prost_timestamp),
                resolved_at: resolved_at.map(Self::chrono_to_prost_timestamp),
                response_breached,
                resolution_breached,
                response_time_minutes: actual_response_time.unwrap_or(0),
                resolution_time_minutes: actual_resolution_time.unwrap_or(0),
                created_at: Some(Self::chrono_to_prost_timestamp(row.get("created_at"))),
                updated_at: Some(Self::chrono_to_prost_timestamp(row.get("updated_at"))),
                sla_policy: Some(SlaPolicy {
                    id: row.get::<Uuid, _>("sla_policy_id").to_string(),
                    tenant_id: tenant_context.tenant_id.to_string(),
                    name: row.get("policy_name"),
                    description: String::new(),
                    priority: Self::ticket_priority_to_enum(&row.get::<TicketPriority, _>("priority")),
                    severity: Self::ticket_severity_to_enum(&row.get::<TicketSeverity, _>("severity")),
                    response_time_minutes: row.get("response_time_minutes"),
                    resolution_time_minutes: row.get("resolution_time_minutes"),
                    business_hours_only: row.get("business_hours_only"),
                    timezone: "UTC".to_string(),
                    is_active: true,
                    created_at: None,
                    updated_at: None,
                }),
                ticket: None, // We'll implement ticket retrieval later if needed
            })
        } else {
            None
        };

        let response = GetSlaMetricsResponse {
            response: Some(Self::success_response(
                "SLA metrics retrieved successfully",
                &req.metadata.as_ref().map(|m| m.request_id.clone()).unwrap_or_default()
            )),
            metrics,
        };

        Ok(Response::new(response))
    }

    async fn get_sla_dashboard(
        &self,
        request: Request<GetSlaDashboardRequest>,
    ) -> Result<Response<GetSlaDashboardResponse>, Status> {
        let metadata = request.metadata();
        let tenant_context = self.extract_tenant_context(metadata).await?;
        let req = request.into_inner();

        let now = Utc::now();
        let start_date = req.start_date
            .map(|ts| chrono::DateTime::from_timestamp(ts.seconds, ts.nanos as u32).unwrap_or(now))
            .unwrap_or_else(|| now - Duration::days(30));

        // Get SLA compliance data
        let compliance_query = r#"
            SELECT
                sp.priority,
                COUNT(*) as total_tickets,
                COUNT(CASE WHEN ts.is_response_met = true THEN 1 END) as response_met,
                COUNT(CASE WHEN ts.is_resolution_met = true THEN 1 END) as resolution_met
            FROM tickets t
            LEFT JOIN ticket_sla ts ON t.id = ts.ticket_id
            LEFT JOIN sla_policies sp ON ts.sla_policy_id = sp.id
            WHERE t.tenant_id = $1
            AND t.created_at >= $2
            AND t.status NOT IN ('New')
            GROUP BY sp.priority
        "#;

        let compliance_rows = sqlx::query(compliance_query)
            .bind(tenant_context.tenant_id)
            .bind(start_date)
            .fetch_all(&*self.pool)
            .await
            .map_err(|e| Status::internal(format!("Database error: {}", e)))?;

        let mut compliance = Vec::new();
        let mut total_tickets = 0;
        let mut total_response_met = 0;
        let mut total_resolution_met = 0;

        for row in compliance_rows {
            let total: i64 = row.get("total_tickets");
            let response_met: i64 = row.get("response_met");
            let resolution_met: i64 = row.get("resolution_met");

            if total > 0 {
                let response_percentage = (response_met as f64 / total as f64) * 100.0;
                let resolution_percentage = (resolution_met as f64 / total as f64) * 100.0;

                compliance.push(SlaDashboardItem {
                    category: Self::ticket_priority_to_string(&row.get::<TicketPriority, _>("priority")),
                    total: total as i32,
                    achieved: response_met as i32,
                    percentage: response_percentage,
                });

                total_tickets += total;
                total_response_met += response_met;
                total_resolution_met += resolution_met;
            }
        }

        // Get breach data
        let breach_query = r#"
            SELECT
                sp.priority,
                COUNT(*) as total_breaches
            FROM tickets t
            JOIN ticket_sla ts ON t.id = ts.ticket_id
            JOIN sla_policies sp ON ts.sla_policy_id = sp.id
            WHERE t.tenant_id = $1
            AND t.created_at >= $2
            AND (ts.response_breached = true OR ts.resolution_breached = true)
            GROUP BY sp.priority
        "#;

        let breach_rows = sqlx::query(breach_query)
            .bind(tenant_context.tenant_id)
            .bind(start_date)
            .fetch_all(&*self.pool)
            .await
            .map_err(|e| Status::internal(format!("Database error: {}", e)))?;

        let mut breaches = Vec::new();
        for row in breach_rows {
            breaches.push(SlaDashboardItem {
                category: Self::ticket_priority_to_string(&row.get::<TicketPriority, _>("priority")),
                total: row.get::<i64, _>("total_breaches") as i32,
                achieved: 0,
                percentage: 0.0,
            });
        }

        // Create summary
        let response_compliance_rate = if total_tickets > 0 {
            (total_response_met as f64 / total_tickets as f64) * 100.0
        } else {
            0.0
        };
        let resolution_compliance_rate = if total_tickets > 0 {
            (total_resolution_met as f64 / total_tickets as f64) * 100.0
        } else {
            0.0
        };

        let summary = SlaSummary {
            total_tickets: total_tickets as i32,
            response_breaches: 0, // Calculate if needed
            resolution_breaches: breaches.iter().map(|b| b.total).sum(),
            response_compliance_rate,
            resolution_compliance_rate,
            overdue_tickets: 0, // Calculate if needed
        };

        // Generate trend data (simplified - daily compliance rates)
        let trend_query = r#"
            SELECT
                DATE(t.created_at) as date,
                COUNT(*) as total,
                COUNT(CASE WHEN ts.is_response_met = true THEN 1 END) as met
            FROM tickets t
            LEFT JOIN ticket_sla ts ON t.id = ts.ticket_id
            WHERE t.tenant_id = $1
            AND t.created_at >= $2
            GROUP BY DATE(t.created_at)
            ORDER BY date
            LIMIT 30
        "#;

        let trend_rows = sqlx::query(trend_query)
            .bind(tenant_context.tenant_id)
            .bind(start_date)
            .fetch_all(&*self.pool)
            .await
            .map_err(|e| Status::internal(format!("Database error: {}", e)))?;

        let trend: Vec<SlaTrend> = trend_rows
            .into_iter()
            .map(|row| {
                let total: i64 = row.get("total");
                let met: i64 = row.get("met");
                let percentage = if total > 0 { (met as f64 / total as f64) * 100.0 } else { 0.0 };

                SlaTrend {
                    period: row.get::<chrono::NaiveDate, _>("date").to_string(),
                    response_compliance: percentage,
                    resolution_compliance: percentage,
                    total_tickets: total as i32,
                }
            })
            .collect();

        let dashboard = SlaDashboard {
            compliance,
            breaches,
            summary: Some(summary),
            trend,
        };

        let response = GetSlaDashboardResponse {
            response: Some(Self::success_response(
                "SLA dashboard data retrieved successfully",
                &req.metadata.as_ref().map(|m| m.request_id.clone()).unwrap_or_default()
            )),
            dashboard: Some(dashboard),
        };

        Ok(Response::new(response))
    }

    async fn get_sla_breaches(
        &self,
        request: Request<GetSlaBreachesRequest>,
    ) -> Result<Response<GetSlaBreachesResponse>, Status> {
        let metadata = request.metadata();
        let tenant_context = self.extract_tenant_context(metadata).await?;
        let req = request.into_inner();

        let page_size = req.pagination.as_ref()
            .map(|p| if p.page_size > 0 && p.page_size <= 100 { p.page_size } else { 20 })
            .unwrap_or(20);

        let page_token = req.pagination.as_ref()
            .and_then(|p| Some(p.page_token.clone()))
            .and_then(|token| token.parse::<i64>().ok())
            .unwrap_or(0);

        // Build WHERE clause
        let mut where_conditions = vec!["t.tenant_id = $1".to_string()];
        let mut bind_count = 2;

        where_conditions.push("(ts.response_breached = true OR ts.resolution_breached = true)".to_string());

        let where_clause = where_conditions.join(" AND ");

        // Count query
        let count_query = format!(
            "SELECT COUNT(*) as total
             FROM tickets t
             JOIN ticket_sla ts ON t.id = ts.ticket_id
             JOIN sla_policies sp ON ts.sla_policy_id = sp.id
             WHERE {}", where_clause
        );

        let mut count_query_builder = sqlx::query(&count_query).bind(tenant_context.tenant_id);

        // No additional filters based on current proto definition

        let total_count: i64 = count_query_builder
            .fetch_one(&*self.pool)
            .await
            .map_err(|e| Status::internal(format!("Failed to count SLA breaches: {}", e)))?
            .get("total");

        // Main query
        let main_query = format!(
            r#"
            SELECT
                t.id as ticket_id, t.title as ticket_title, t.created_at as ticket_created_at,
                ts.*, sp.name as policy_name, sp.priority, sp.severity,
                sp.response_time_minutes, sp.resolution_time_minutes
            FROM tickets t
            JOIN ticket_sla ts ON t.id = ts.ticket_id
            JOIN sla_policies sp ON ts.sla_policy_id = sp.id
            WHERE {}
            ORDER BY ts.created_at DESC
            LIMIT ${} OFFSET ${}
            "#,
            where_clause, bind_count, bind_count + 1
        );

        let mut query_builder = sqlx::query(&main_query).bind(tenant_context.tenant_id);

        // No additional filters based on current proto definition

        query_builder = query_builder
            .bind(page_size as i64)
            .bind(page_token);

        let rows = query_builder
            .fetch_all(&*self.pool)
            .await
            .map_err(|e| Status::internal(format!("Failed to fetch SLA breaches: {}", e)))?;

        let breaches: Vec<SlaBreachAlert> = rows
            .into_iter()
            .map(|row| {
                let created_at: DateTime<Utc> = row.get("created_at");
                let ticket_created_at: DateTime<Utc> = row.get("ticket_created_at");
                let now = Utc::now();

                // Determine breach type
            let breach_type = if row.get("response_breached") && row.get("resolution_breached") {
                "Response and Resolution"
            } else if row.get("response_breached") {
                "Response"
            } else {
                "Resolution"
            };

            // Calculate due time and breach time
            let due_at = if row.get("response_breached") {
                row.get::<Option<DateTime<Utc>>, _>("response_due")
            } else {
                row.get::<Option<DateTime<Utc>>, _>("resolution_due")
            };

            let breach_time = Some(Self::chrono_to_prost_timestamp(now));

            // Calculate minutes overdue
            let minutes_overdue = if let Some(due_time) = due_at {
                (now.signed_duration_since(due_time).num_minutes() as i32).max(0)
            } else {
                0
            };

            SlaBreachAlert {
                id: row.get::<Uuid, _>("id").to_string(),
                tenant_id: tenant_context.tenant_id.to_string(),
                ticket_id: row.get::<Uuid, _>("ticket_id").to_string(),
                sla_policy_id: row.get::<Uuid, _>("sla_policy_id").to_string(),
                breach_type: breach_type.to_string(),
                due_at: due_at.map(Self::chrono_to_prost_timestamp),
                breach_time,
                is_overdue: minutes_overdue > 0,
                minutes_overdue,
                created_at: Some(Self::chrono_to_prost_timestamp(created_at)),
                ticket: None, // Not included in breach alerts
                sla_policy: None, // Not included in breach alerts
            }
            })
            .collect();

        let next_page_token = if (page_token + page_size as i64) < total_count {
            (page_token + page_size as i64).to_string()
        } else {
            String::new()
        };

        let pagination = PaginationResponse {
            total_count: total_count as i32,
            page_size,
            next_page_token: if (page_token + page_size as i64) < total_count {
                (page_token + page_size as i64).to_string()
            } else {
                String::new()
            },
            prev_page_token: if page_token > 0 { (page_token - page_size as i64).to_string() } else { String::new() },
        };

        let response = GetSlaBreachesResponse {
            response: Some(Self::success_response(
                "SLA breaches retrieved successfully",
                &req.metadata.as_ref().map(|m| m.request_id.clone()).unwrap_or_default()
            )),
            breaches,
            pagination: Some(pagination),
        };

        Ok(Response::new(response))
    }

    async fn update_sla_metrics(
        &self,
        request: Request<UpdateSlaMetricsRequest>,
    ) -> Result<Response<UpdateSlaMetricsResponse>, Status> {
        let metadata = request.metadata();
        let tenant_context = self.extract_tenant_context(metadata).await?;
        let req = request.into_inner();

        let ticket_id = Uuid::parse_str(&req.ticket_id)
            .map_err(|_| Status::invalid_argument("Invalid ticket ID"))?;

        // Verify ticket belongs to tenant
        let ticket_check = sqlx::query(
            "SELECT id FROM tickets WHERE id = $1 AND tenant_id = $2"
        )
        .bind(ticket_id)
        .bind(tenant_context.tenant_id)
        .fetch_optional(&*self.pool)
        .await
        .map_err(|e| Status::internal(format!("Database error: {}", e)))?;

        if ticket_check.is_none() {
            return Err(Status::not_found("Ticket not found"));
        }

        let now = Utc::now();

        // Update SLA metrics based on event type
        let event_time = req.event_time
            .map(|ts| chrono::DateTime::from_timestamp(ts.seconds, ts.nanos as u32).unwrap_or(now))
            .unwrap_or(now);

        let (first_response_at, resolved_at) = match req.event_type.as_str() {
            "first_response" => (Some(event_time), None),
            "resolved" => (None, Some(event_time)),
            _ => return Err(Status::invalid_argument("Invalid event type. Must be 'first_response' or 'resolved'")),
        };

        let update_query = r#"
            UPDATE ticket_sla SET
                first_response_at = COALESCE($1, first_response_at),
                resolved_at = COALESCE($2, resolved_at),
                updated_at = $3
            WHERE ticket_id = $4
            RETURNING *
        "#;

        let row = sqlx::query(update_query)
            .bind(first_response_at)
            .bind(resolved_at)
            .bind(now)
            .bind(ticket_id)
            .fetch_optional(&*self.pool)
            .await
            .map_err(|e| Status::internal(format!("Failed to update SLA metrics: {}", e)))?;

        let metrics = if let Some(row) = row {
            let response_due: Option<DateTime<Utc>> = row.get("response_due");
            let resolution_due: Option<DateTime<Utc>> = row.get("resolution_due");
            let first_response: Option<DateTime<Utc>> = row.get("first_response_at");
            let resolved_at: Option<DateTime<Utc>> = row.get("resolved_at");

            Some(SlaMetrics {
                id: row.get::<Uuid, _>("id").to_string(),
                tenant_id: tenant_context.tenant_id.to_string(),
                ticket_id: ticket_id.to_string(),
                sla_policy_id: row.get::<Uuid, _>("sla_policy_id").to_string(),
                response_due_at: response_due.map(Self::chrono_to_prost_timestamp),
                resolution_due_at: resolution_due.map(Self::chrono_to_prost_timestamp),
                first_response_at: first_response.map(Self::chrono_to_prost_timestamp),
                resolved_at: resolved_at.map(Self::chrono_to_prost_timestamp),
                response_breached: row.get("response_breached"),
                resolution_breached: row.get("resolution_breached"),
                response_time_minutes: row.get("actual_response_time"),
                resolution_time_minutes: row.get("actual_resolution_time"),
                created_at: Some(Self::chrono_to_prost_timestamp(row.get("created_at"))),
                updated_at: Some(Self::chrono_to_prost_timestamp(row.get("updated_at"))),
                sla_policy: None, // Not included in update response
                ticket: None, // Not included in update response
            })
        } else {
            return Err(Status::not_found("SLA metrics not found for this ticket"));
        };

        let response = UpdateSlaMetricsResponse {
            response: Some(Self::success_response(
                "SLA metrics updated successfully",
                &req.metadata.as_ref().map(|m| m.request_id.clone()).unwrap_or_default()
            )),
            metrics,
        };

        Ok(Response::new(response))
    }
}