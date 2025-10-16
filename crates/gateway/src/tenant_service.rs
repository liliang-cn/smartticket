//! gRPC Tenant Service Implementation for SmartTicket
//!
//! This module implements the gRPC service handlers for tenant management,
//! including CRUD operations, subscription management, and billing.

use std::result::Result as StdResult;
use std::str::FromStr;
use std::sync::Arc;
use tonic::{Request, Response, Status};
use tracing::{error, info, instrument};
use uuid::Uuid;

use crate::smartticket_v1::{
    tenant_service_server::TenantService, BillingLineItem, CreateTenantRequest,
    CreateTenantResponse, DeleteTenantRequest, DeleteTenantResponse, GetCurrentTenantRequest,
    GetCurrentTenantResponse, GetTenantBillingRequest, GetTenantBillingResponse, GetTenantRequest,
    GetTenantResponse, GetTenantUsageRequest, GetTenantUsageResponse, ListTenantsRequest,
    ListTenantsResponse, NotificationSettings, PaginationResponse, PaymentMethod,
    Response as ApiResponse, SecuritySettings, SubscriptionTier as GrpcSubscriptionTier,
    TenantBilling, TenantInfo, TenantSettings, TenantUsage, UpdateSubscriptionRequest,
    UpdateSubscriptionResponse, UpdateTenantRequest, UpdateTenantResponse,
    UpdateTenantStatusRequest, UpdateTenantStatusResponse, UsageMetrics, User, UserActivitySummary,
};
use crate::{PermissionCheck, RequestExt};
use smartticket_shared_database::{AuthService, SubscriptionTier, Tenant};
use smartticket_shared_error::{Result, SmartTicketError};
use sqlx::Row;

/// gRPC Tenant Service implementation
pub struct TenantGrpcService {
    #[allow(dead_code)]
    auth_service: Arc<AuthService>,
    db_pool: Arc<sqlx::PgPool>,
}

impl TenantGrpcService {
    /// Create a new gRPC tenant service
    pub fn new(auth_service: Arc<AuthService>, db_pool: Arc<sqlx::PgPool>) -> Self {
        Self {
            auth_service,
            db_pool,
        }
    }

    /// Create success response
    fn create_success_response(message: &str, request_id: &str) -> ApiResponse {
        ApiResponse {
            success: true,
            message: message.to_string(),
            data: None,
            errors: vec![],
            request_id: request_id.to_string(),
        }
    }

    /// Create error response
    fn create_error_response(message: &str, request_id: &str) -> ApiResponse {
        ApiResponse {
            success: false,
            message: message.to_string(),
            data: None,
            errors: vec![crate::smartticket_v1::Error {
                code: "VALIDATION_ERROR".to_string(),
                message: message.to_string(),
                details: None,
            }],
            request_id: request_id.to_string(),
        }
    }

    /// Convert database tenant to gRPC tenant info
    async fn db_tenant_to_grpc_info(&self, tenant: Tenant) -> Result<TenantInfo> {
        // Get current user count for this tenant
        let user_count: i64 = sqlx::query_scalar(
            "SELECT COUNT(*) FROM users WHERE tenant_id = $1 AND is_active = true",
        )
        .bind(tenant.id)
        .fetch_one(&*self.db_pool)
        .await
        .unwrap_or(0);

        Ok(TenantInfo {
            id: tenant.id.to_string(),
            name: tenant.name,
            domain: tenant.domain,
            subscription_tier: match tenant.subscription_tier {
                SubscriptionTier::Trial => GrpcSubscriptionTier::Standard as i32, // Map Trial to Standard for gRPC
                SubscriptionTier::Standard => GrpcSubscriptionTier::Standard as i32,
                SubscriptionTier::Premium => GrpcSubscriptionTier::Premium as i32,
                SubscriptionTier::Enterprise => GrpcSubscriptionTier::Enterprise as i32,
            },
            max_users: tenant.max_users,
            data_residency_region: tenant.data_residency_region,
            is_active: tenant.is_active,
            created_at: Some(prost_types::Timestamp {
                seconds: tenant.created_at.timestamp(),
                nanos: tenant.created_at.timestamp_subsec_nanos() as i32,
            }),
            updated_at: Some(prost_types::Timestamp {
                seconds: tenant.updated_at.timestamp(),
                nanos: tenant.updated_at.timestamp_subsec_nanos() as i32,
            }),
            current_user_count: user_count as i32,
            subscription_expires_at: None, // TODO: Implement subscription expiration
            is_trial: false,               // TODO: Implement trial status
            contact_email: {
                // Get contact email from database
                sqlx::query_scalar(
                    "SELECT COALESCE(contact_email, '') FROM tenant_contact WHERE tenant_id = $1",
                )
                .bind(tenant.id)
                .fetch_one(&*self.db_pool)
                .await
                .unwrap_or_else(|_| "".to_string())
            },
            billing_address: {
                // Get billing address from database
                sqlx::query_scalar(
                    "SELECT COALESCE(billing_address, '') FROM tenant_billing WHERE tenant_id = $1",
                )
                .bind(tenant.id)
                .fetch_one(&*self.db_pool)
                .await
                .unwrap_or_else(|_| "".to_string())
            },
            phone: {
                // Get phone from database
                sqlx::query_scalar(
                    "SELECT COALESCE(phone, '') FROM tenant_contact WHERE tenant_id = $1",
                )
                .bind(tenant.id)
                .fetch_one(&*self.db_pool)
                .await
                .unwrap_or_else(|_| "".to_string())
            },
            settings: Some(TenantSettings {
                default_timezone: "UTC".to_string(),
                default_language: "en".to_string(),
                enable_multi_language: false,
                allow_user_registration: false,
                branding_logo_url: String::new(),
                branding_color: String::new(),
                custom_fields: None,
                security: Some(SecuritySettings {
                    require_2fa: false,
                    password_min_length: 8,
                    require_password_change: false,
                    session_timeout_minutes: 480,
                    ip_whitelist_enabled: false,
                    allowed_ip_ranges: vec![],
                }),
                notifications: Some(NotificationSettings {
                    email_notifications: true,
                    sms_notifications: false,
                    push_notifications: true,
                    default_from_email: {
                        // Get notification email from database or use tenant-specific default
                        sqlx::query_scalar("SELECT COALESCE(default_from_email, 'noreply@smartticket.local') FROM tenant_notification_settings WHERE tenant_id = $1")
                        .bind(tenant.id)
                        .fetch_one(&*self.db_pool)
                        .await
                        .unwrap_or_else(|_| "noreply@smartticket.local".to_string())
                    },
                    default_from_name: "SmartTicket".to_string(),
                    notification_templates: None,
                }),
            }),
        })
    }

    /// Convert gRPC subscription tier to database subscription tier
    fn grpc_subscription_tier_to_db(tier: GrpcSubscriptionTier) -> Result<SubscriptionTier> {
        match tier {
            GrpcSubscriptionTier::Standard => Ok(SubscriptionTier::Standard),
            GrpcSubscriptionTier::Premium => Ok(SubscriptionTier::Premium),
            GrpcSubscriptionTier::Enterprise => Ok(SubscriptionTier::Enterprise),
            GrpcSubscriptionTier::Unspecified => Ok(SubscriptionTier::Standard), // Default to Standard
        }
    }

    /// Validate domain format
    fn validate_domain(domain: &str) -> Result<()> {
        if domain.trim().is_empty() {
            return Err(SmartTicketError::Validation(
                "Domain is required".to_string(),
            ));
        }

        // Basic domain validation
        if !domain.contains('.') {
            return Err(SmartTicketError::Validation(
                "Invalid domain format".to_string(),
            ));
        }

        if domain.len() > 253 {
            return Err(SmartTicketError::Validation("Domain too long".to_string()));
        }

        Ok(())
    }

    /// Validate email format
    fn validate_email(email: &str) -> Result<()> {
        if email.trim().is_empty() {
            return Err(SmartTicketError::Validation(
                "Contact email is required".to_string(),
            ));
        }

        if !email.contains('@') || !email.contains('.') {
            return Err(SmartTicketError::Validation(
                "Invalid email format".to_string(),
            ));
        }

        Ok(())
    }

    /// Generate setup token for new tenant
    fn generate_setup_token() -> String {
        format!("setup_{}", Uuid::new_v4().to_string())
    }
}

#[tonic::async_trait]
impl TenantService for TenantGrpcService {
    #[instrument(skip(self))]
    async fn create_tenant(
        &self,
        request: Request<CreateTenantRequest>,
    ) -> StdResult<Response<CreateTenantResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("tenant:create") {
            return Err(e);
        }

        let auth_user = request.auth_user()?;
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!("Creating tenant: {} by: {}", req.name, auth_user.email);

        // Validate required fields
        if req.name.trim().is_empty() {
            let response = CreateTenantResponse {
                response: Some(Self::create_error_response(
                    "Tenant name is required",
                    &request_id,
                )),
                tenant: None,
                setup_token: String::new(),
            };
            return Ok(Response::new(response));
        }

        if let Err(e) = Self::validate_domain(&req.domain) {
            let response = CreateTenantResponse {
                response: Some(Self::create_error_response(&e.to_string(), &request_id)),
                tenant: None,
                setup_token: String::new(),
            };
            return Ok(Response::new(response));
        }

        if let Err(e) = Self::validate_email(&req.contact_email) {
            let response = CreateTenantResponse {
                response: Some(Self::create_error_response(&e.to_string(), &request_id)),
                tenant: None,
                setup_token: String::new(),
            };
            return Ok(Response::new(response));
        }

        // Validate and convert subscription tier
        let subscription_tier = match Self::grpc_subscription_tier_to_db(req.subscription_tier()) {
            Ok(tier) => tier,
            Err(e) => {
                let response = CreateTenantResponse {
                    response: Some(Self::create_error_response(&e.to_string(), &request_id)),
                    tenant: None,
                    setup_token: String::new(),
                };
                return Ok(Response::new(response));
            }
        };

        // Create new tenant
        let mut new_tenant = Tenant::new(
            req.name.trim().to_string(),
            req.domain.trim().to_lowercase(),
            subscription_tier,
            req.data_residency_region,
        );

        // Override max_users if provided
        new_tenant.max_users = req.max_users.max(1);

        // Insert tenant into database
        let query = r#"
            INSERT INTO tenants (
                id, name, domain, subscription_tier, max_users,
                data_residency_region, settings, is_active, created_at, updated_at
            ) VALUES (
                $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
            ) RETURNING *
        "#;

        let result_tenant = match sqlx::query_as::<_, Tenant>(query)
            .bind(new_tenant.id)
            .bind(&new_tenant.name)
            .bind(&new_tenant.domain)
            .bind(new_tenant.subscription_tier)
            .bind(new_tenant.max_users)
            .bind(&new_tenant.data_residency_region)
            .bind(&new_tenant.settings)
            .bind(new_tenant.is_active)
            .bind(new_tenant.created_at)
            .bind(new_tenant.updated_at)
            .fetch_one(&*self.db_pool)
            .await
        {
            Ok(tenant) => tenant,
            Err(e) => {
                error!("Failed to create tenant: {}", e);
                let response = CreateTenantResponse {
                    response: Some(Self::create_error_response(
                        "Failed to create tenant",
                        &request_id,
                    )),
                    tenant: None,
                    setup_token: String::new(),
                };
                return Err(Status::internal(format!("Database error: {}", e)));
            }
        };

        // Convert to gRPC response
        let grpc_tenant = self
            .db_tenant_to_grpc_info(result_tenant)
            .await
            .map_err(|e| Status::internal(format!("Failed to convert tenant: {}", e)))?;
        let setup_token = Self::generate_setup_token();

        let response = CreateTenantResponse {
            response: Some(Self::create_success_response(
                "Tenant created successfully",
                &request_id,
            )),
            tenant: Some(grpc_tenant),
            setup_token,
        };

        info!("Successfully created tenant: {}", req.name);
        Ok(Response::new(response))
    }

    #[instrument(skip(self))]
    async fn get_tenant(
        &self,
        request: Request<GetTenantRequest>,
    ) -> StdResult<Response<GetTenantResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("tenant:view") {
            return Err(e);
        }

        let auth_user = request.auth_user()?;
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!("Getting tenant: {} by: {}", req.tenant_id, auth_user.email);

        // Parse tenant ID
        let tenant_id = match Uuid::from_str(&req.tenant_id) {
            Ok(id) => id,
            Err(_) => {
                let response = GetTenantResponse {
                    response: Some(Self::create_error_response(
                        "Invalid tenant ID format",
                        &request_id,
                    )),
                    tenant: None,
                };
                return Ok(Response::new(response));
            }
        };

        // Retrieve tenant from database
        let query = "SELECT * FROM tenants WHERE id = $1";
        let tenant = sqlx::query_as::<_, Tenant>(query)
            .bind(tenant_id)
            .fetch_optional(&*self.db_pool)
            .await
            .map_err(|e| {
                error!("Failed to retrieve tenant: {}", e);
                Status::internal(format!("Database error: {}", e))
            })?;

        let tenant = match tenant {
            Some(t) => t,
            None => {
                let response = GetTenantResponse {
                    response: Some(Self::create_error_response("Tenant not found", &request_id)),
                    tenant: None,
                };
                return Ok(Response::new(response));
            }
        };

        let grpc_tenant = self
            .db_tenant_to_grpc_info(tenant)
            .await
            .map_err(|e| Status::internal(format!("Failed to convert tenant: {}", e)))?;

        let response = GetTenantResponse {
            response: Some(Self::create_success_response(
                "Tenant retrieved successfully",
                &request_id,
            )),
            tenant: Some(grpc_tenant),
        };

        Ok(Response::new(response))
    }

    #[instrument(skip(self))]
    async fn get_current_tenant(
        &self,
        request: Request<GetCurrentTenantRequest>,
    ) -> StdResult<Response<GetCurrentTenantResponse>, Status> {
        // Check authentication
        let auth_user = request.auth_user()?;
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!("Getting current tenant for user: {}", auth_user.email);

        // Retrieve current tenant from database
        let query = "SELECT * FROM tenants WHERE id = $1";
        let tenant = sqlx::query_as::<_, Tenant>(query)
            .bind(auth_user.tenant_id)
            .fetch_optional(&*self.db_pool)
            .await
            .map_err(|e| {
                error!("Failed to retrieve current tenant: {}", e);
                Status::internal(format!("Database error: {}", e))
            })?;

        let tenant = match tenant {
            Some(t) => t,
            None => {
                let response = GetCurrentTenantResponse {
                    response: Some(Self::create_error_response(
                        "Current tenant not found",
                        &request_id,
                    )),
                    tenant: None,
                    recent_users: vec![],
                };
                return Ok(Response::new(response));
            }
        };

        let grpc_tenant = self
            .db_tenant_to_grpc_info(tenant)
            .await
            .map_err(|e| Status::internal(format!("Failed to convert tenant: {}", e)))?;

        // Retrieve recent users for this tenant
        let recent_users_query = r#"
            SELECT id, email, full_name, is_active, last_login_at, created_at
            FROM users
            WHERE tenant_id = $1 AND is_active = true
            ORDER BY last_login_at DESC NULLS LAST
            LIMIT 5
        "#;

        let recent_users = sqlx::query(recent_users_query)
            .bind(auth_user.tenant_id)
            .fetch_all(&*self.db_pool)
            .await
            .map_err(|e| {
                error!("Failed to retrieve recent users: {}", e);
                Status::internal(format!("Database error: {}", e))
            })?
            .into_iter()
            .map(|row| User {
                id: row.get::<uuid::Uuid, _>("id").to_string(),
                tenant_id: auth_user.tenant_id.to_string(),
                email: row.get("email"),
                username: row.get("email"), // Use email as username fallback
                full_name: row.get("full_name"),
                role: 1, // Default role
                is_active: row.get("is_active"),
                last_login_at: row
                    .get::<Option<chrono::DateTime<chrono::Utc>>, _>("last_login_at")
                    .map(|dt| prost_types::Timestamp {
                        seconds: dt.timestamp(),
                        nanos: dt.timestamp_subsec_nanos() as i32,
                    }),
            })
            .collect();

        let response = GetCurrentTenantResponse {
            response: Some(Self::create_success_response(
                "Current tenant retrieved successfully",
                &request_id,
            )),
            tenant: Some(grpc_tenant),
            recent_users,
        };

        Ok(Response::new(response))
    }

    #[instrument(skip(self))]
    async fn update_tenant(
        &self,
        request: Request<UpdateTenantRequest>,
    ) -> StdResult<Response<UpdateTenantResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("tenant:update") {
            return Err(e);
        }

        let auth_user = request.auth_user()?;
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!("Updating tenant: {} by: {}", req.tenant_id, auth_user.email);

        // Parse tenant ID
        let tenant_id = match Uuid::from_str(&req.tenant_id) {
            Ok(id) => id,
            Err(_) => {
                let response = UpdateTenantResponse {
                    response: Some(Self::create_error_response(
                        "Invalid tenant ID format",
                        &request_id,
                    )),
                    tenant: None,
                };
                return Ok(Response::new(response));
            }
        };

        // Validate fields if provided
        if !req.name.is_empty() && req.name.trim().is_empty() {
            let response = UpdateTenantResponse {
                response: Some(Self::create_error_response(
                    "Tenant name cannot be empty",
                    &request_id,
                )),
                tenant: None,
            };
            return Ok(Response::new(response));
        }

        if !req.domain.is_empty() {
            if let Err(e) = Self::validate_domain(&req.domain) {
                let response = UpdateTenantResponse {
                    response: Some(Self::create_error_response(&e.to_string(), &request_id)),
                    tenant: None,
                };
                return Ok(Response::new(response));
            }
        }

        if !req.contact_email.is_empty() {
            if let Err(e) = Self::validate_email(&req.contact_email) {
                let response = UpdateTenantResponse {
                    response: Some(Self::create_error_response(&e.to_string(), &request_id)),
                    tenant: None,
                };
                return Ok(Response::new(response));
            }
        }

        // Update tenant in database
        let mut update_parts = Vec::new();
        let mut param_count = 1;

        if !req.name.is_empty() {
            update_parts.push(format!("name = ${}", param_count));
            param_count += 1;
        }
        if !req.domain.is_empty() {
            update_parts.push(format!("domain = ${}", param_count));
            param_count += 1;
        }

        if update_parts.is_empty() {
            // No actual update needed, just return the current tenant
            let current_query = "SELECT * FROM tenants WHERE id = $1";
            let current_tenant = sqlx::query_as::<_, Tenant>(current_query)
                .bind(tenant_id)
                .fetch_one(&*self.db_pool)
                .await
                .map_err(|e| {
                    error!("Failed to retrieve current tenant: {}", e);
                    Status::internal(format!("Database error: {}", e))
                })?;

            let grpc_tenant = self
                .db_tenant_to_grpc_info(current_tenant)
                .await
                .map_err(|e| Status::internal(format!("Failed to convert tenant: {}", e)))?;

            let response = UpdateTenantResponse {
                response: Some(Self::create_success_response(
                    "No updates needed",
                    &request_id,
                )),
                tenant: Some(grpc_tenant),
            };
            return Ok(Response::new(response));
        }

        update_parts.push(format!("updated_at = ${}", param_count));
        param_count += 1;

        let query = format!(
            "UPDATE tenants SET {} WHERE id = ${} RETURNING *",
            update_parts.join(", "),
            param_count
        );

        let mut query_builder = sqlx::query_as::<_, Tenant>(&query);

        if !req.name.is_empty() {
            query_builder = query_builder.bind(&req.name);
        }
        if !req.domain.is_empty() {
            query_builder = query_builder.bind(&req.domain);
        }
        query_builder = query_builder.bind(chrono::Utc::now());
        query_builder = query_builder.bind(tenant_id);

        let updated_tenant = query_builder.fetch_one(&*self.db_pool).await.map_err(|e| {
            error!("Failed to update tenant: {}", e);
            Status::internal(format!("Database error: {}", e))
        })?;

        let grpc_tenant = self
            .db_tenant_to_grpc_info(updated_tenant)
            .await
            .map_err(|e| Status::internal(format!("Failed to convert tenant: {}", e)))?;

        let response = UpdateTenantResponse {
            response: Some(Self::create_success_response(
                "Tenant updated successfully",
                &request_id,
            )),
            tenant: Some(grpc_tenant),
        };

        Ok(Response::new(response))
    }

    #[instrument(skip(self))]
    async fn list_tenants(
        &self,
        request: Request<ListTenantsRequest>,
    ) -> StdResult<Response<ListTenantsResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("tenant:view") {
            return Err(e);
        }

        let auth_user = request.auth_user()?;
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!("Listing tenants by: {}", auth_user.email);

        // Implement actual tenant listing from database with filtering and pagination
        let page_size = req.pagination.as_ref().map(|p| p.page_size).unwrap_or(10);
        let offset = req
            .pagination
            .as_ref()
            .and_then(|p| p.page_token.parse::<usize>().ok())
            .unwrap_or(0);

        let query = "SELECT * FROM tenants ORDER BY created_at DESC LIMIT $1 OFFSET $2";
        let tenants = sqlx::query_as::<_, Tenant>(query)
            .bind(page_size as i64)
            .bind(offset as i64)
            .fetch_all(&*self.db_pool)
            .await
            .map_err(|e| {
                error!("Failed to list tenants: {}", e);
                Status::internal(format!("Database error: {}", e))
            })?;

        // Get total count for pagination
        let total_count: i64 = sqlx::query_scalar("SELECT COUNT(*) FROM tenants")
            .fetch_one(&*self.db_pool)
            .await
            .map_err(|e| {
                error!("Failed to count tenants: {}", e);
                Status::internal(format!("Database error: {}", e))
            })?;

        // Convert tenants to gRPC format
        let mut grpc_tenants = Vec::new();
        for tenant in tenants {
            match self.db_tenant_to_grpc_info(tenant).await {
                Ok(grpc_tenant) => grpc_tenants.push(grpc_tenant),
                Err(e) => {
                    error!("Failed to convert tenant: {}", e);
                    // Continue with other tenants
                }
            }
        }

        let response = ListTenantsResponse {
            response: Some(Self::create_success_response(
                "Tenants listed successfully",
                &request_id,
            )),
            tenants: grpc_tenants,
            pagination: Some(PaginationResponse {
                total_count: total_count as i32,
                page_size,
                next_page_token: if (offset + page_size as usize) < total_count as usize {
                    (offset + page_size as usize).to_string()
                } else {
                    "".to_string()
                },
                prev_page_token: if offset > 0 {
                    offset.saturating_sub(page_size as usize).to_string()
                } else {
                    "".to_string()
                },
            }),
        };

        Ok(Response::new(response))
    }

    #[instrument(skip(self))]
    async fn delete_tenant(
        &self,
        request: Request<DeleteTenantRequest>,
    ) -> StdResult<Response<DeleteTenantResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("tenant:delete") {
            return Err(e);
        }

        let auth_user = request.auth_user()?;
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!("Deleting tenant: {} by: {}", req.tenant_id, auth_user.email);

        // Parse tenant ID
        let tenant_id = match Uuid::from_str(&req.tenant_id) {
            Ok(id) => id,
            Err(_) => {
                let response = DeleteTenantResponse {
                    response: Some(Self::create_error_response(
                        "Invalid tenant ID format",
                        &request_id,
                    )),
                };
                return Ok(Response::new(response));
            }
        };

        // Prevent user from deleting their own tenant
        if auth_user.tenant_id == tenant_id {
            let response = DeleteTenantResponse {
                response: Some(Self::create_error_response(
                    "Cannot delete your own tenant",
                    &request_id,
                )),
            };
            return Ok(Response::new(response));
        }

        // Implement actual tenant deletion (soft delete) in database
        let query =
            "UPDATE tenants SET is_active = false, updated_at = NOW() WHERE id = $1 RETURNING *";
        let deleted_tenant = sqlx::query_as::<_, Tenant>(query)
            .bind(tenant_id)
            .fetch_optional(&*self.db_pool)
            .await
            .map_err(|e| {
                error!("Failed to delete tenant: {}", e);
                Status::internal(format!("Database error: {}", e))
            })?;

        let response = if deleted_tenant.is_some() {
            DeleteTenantResponse {
                response: Some(Self::create_success_response(
                    "Tenant deleted successfully",
                    &request_id,
                )),
            }
        } else {
            DeleteTenantResponse {
                response: Some(Self::create_error_response("Tenant not found", &request_id)),
            }
        };

        Ok(Response::new(response))
    }

    #[instrument(skip(self))]
    async fn update_tenant_status(
        &self,
        request: Request<UpdateTenantStatusRequest>,
    ) -> StdResult<Response<UpdateTenantStatusResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("tenant:update") {
            return Err(e);
        }

        let auth_user = request.auth_user()?;
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!(
            "Updating tenant status: {} -> {} by: {}",
            req.tenant_id, req.is_active, auth_user.email
        );

        // Parse tenant ID
        let tenant_id = match Uuid::from_str(&req.tenant_id) {
            Ok(id) => id,
            Err(_) => {
                let response = UpdateTenantStatusResponse {
                    response: Some(Self::create_error_response(
                        "Invalid tenant ID format",
                        &request_id,
                    )),
                    tenant: None,
                };
                return Ok(Response::new(response));
            }
        };

        // Prevent user from deactivating their own tenant
        if auth_user.tenant_id == tenant_id && !req.is_active {
            let response = UpdateTenantStatusResponse {
                response: Some(Self::create_error_response(
                    "Cannot deactivate your own tenant",
                    &request_id,
                )),
                tenant: None,
            };
            return Ok(Response::new(response));
        }

        // Implement actual tenant status update in database
        let query =
            "UPDATE tenants SET is_active = $1, updated_at = NOW() WHERE id = $2 RETURNING *";
        let updated_tenant = sqlx::query_as::<_, Tenant>(query)
            .bind(req.is_active)
            .bind(tenant_id)
            .fetch_optional(&*self.db_pool)
            .await
            .map_err(|e| {
                error!("Failed to update tenant status: {}", e);
                Status::internal(format!("Database error: {}", e))
            })?;

        let grpc_tenant = match updated_tenant {
            Some(tenant) => self
                .db_tenant_to_grpc_info(tenant)
                .await
                .map_err(|e| Status::internal(format!("Failed to convert tenant: {}", e)))?,
            None => {
                let response = UpdateTenantStatusResponse {
                    response: Some(Self::create_error_response("Tenant not found", &request_id)),
                    tenant: None,
                };
                return Ok(Response::new(response));
            }
        };

        let response = UpdateTenantStatusResponse {
            response: Some(Self::create_success_response(
                if req.is_active {
                    "Tenant activated successfully"
                } else {
                    "Tenant deactivated successfully"
                },
                &request_id,
            )),
            tenant: Some(grpc_tenant),
        };

        Ok(Response::new(response))
    }

    #[instrument(skip(self))]
    async fn update_subscription(
        &self,
        request: Request<UpdateSubscriptionRequest>,
    ) -> StdResult<Response<UpdateSubscriptionResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("tenant:update") {
            return Err(e);
        }

        let auth_user = request.auth_user()?;
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!(
            "Updating subscription for tenant: {} by: {}",
            req.tenant_id, auth_user.email
        );

        // Parse tenant ID
        let tenant_id = match Uuid::from_str(&req.tenant_id) {
            Ok(id) => id,
            Err(_) => {
                let response = UpdateSubscriptionResponse {
                    response: Some(Self::create_error_response(
                        "Invalid tenant ID format",
                        &request_id,
                    )),
                    tenant: None,
                    billing: None,
                    will_invoice: false,
                };
                return Ok(Response::new(response));
            }
        };

        // Validate and convert subscription tier
        let subscription_tier = match Self::grpc_subscription_tier_to_db(req.new_tier()) {
            Ok(tier) => tier,
            Err(e) => {
                let response = UpdateSubscriptionResponse {
                    response: Some(Self::create_error_response(&e.to_string(), &request_id)),
                    tenant: None,
                    billing: None,
                    will_invoice: false,
                };
                return Ok(Response::new(response));
            }
        };

        // Implement actual subscription update in database
        let query = "UPDATE tenants SET subscription_tier = $1, max_users = $2, updated_at = NOW() WHERE id = $3 RETURNING *";
        let updated_tenant = sqlx::query_as::<_, Tenant>(query)
            .bind(subscription_tier)
            .bind(req.new_max_users.max(1))
            .bind(tenant_id)
            .fetch_optional(&*self.db_pool)
            .await
            .map_err(|e| {
                error!("Failed to update subscription: {}", e);
                Status::internal(format!("Database error: {}", e))
            })?;

        let grpc_tenant = match updated_tenant {
            Some(tenant) => self
                .db_tenant_to_grpc_info(tenant)
                .await
                .map_err(|e| Status::internal(format!("Failed to convert tenant: {}", e)))?,
            None => {
                let response = UpdateSubscriptionResponse {
                    response: Some(Self::create_error_response("Tenant not found", &request_id)),
                    tenant: None,
                    billing: None,
                    will_invoice: false,
                };
                return Ok(Response::new(response));
            }
        };

        // Calculate real billing information based on database data
        let billing_cycle_start = chrono::Utc::now() - chrono::Duration::days(30);
        let billing_cycle_end = chrono::Utc::now() + chrono::Duration::days(30);

        // Get actual usage statistics for billing calculation
        let current_users_count: i64 = sqlx::query_scalar(
            "SELECT COUNT(*) FROM users WHERE tenant_id = $1 AND is_active = true",
        )
        .bind(tenant_id)
        .fetch_one(&*self.db_pool)
        .await
        .map_err(|e| {
            error!("Failed to get user count for billing: {}", e);
            Status::internal(format!("Database error: {}", e))
        })?;

        // Calculate base cost and per-user cost based on subscription tier
        let (base_cost, per_user_cost, included_users) = match subscription_tier {
            SubscriptionTier::Trial => (0.0, 10.0, 5),
            SubscriptionTier::Standard => (99.0, 5.0, 10),
            SubscriptionTier::Premium => (299.0, 3.0, 25),
            SubscriptionTier::Enterprise => (999.0, 1.0, 100),
        };

        let current_users = current_users_count as i32;
        let additional_users = (current_users.saturating_sub(included_users)).max(0);
        let user_cost = additional_users as f64 * per_user_cost;
        let monthly_cost = base_cost + user_cost;

        // Create billing line items
        let mut line_items = vec![BillingLineItem {
            item_type: "subscription".to_string(),
            description: format!(
                "{} Plan (Monthly)",
                match subscription_tier {
                    SubscriptionTier::Trial => "Trial",
                    SubscriptionTier::Standard => "Standard",
                    SubscriptionTier::Premium => "Premium",
                    SubscriptionTier::Enterprise => "Enterprise",
                }
            ),
            quantity: 1,
            unit_price: base_cost,
            total_price: base_cost,
        }];

        // Add additional users line item if applicable
        if additional_users > 0 {
            line_items.push(BillingLineItem {
                item_type: "additional_users".to_string(),
                description: format!(
                    "Additional Users ({} × ${:.2})",
                    additional_users, per_user_cost
                ),
                quantity: additional_users,
                unit_price: per_user_cost,
                total_price: user_cost,
            });
        }

        // Get usage-based costs (simplified for now)
        let storage_usage_bytes: i64 = sqlx::query_scalar(
            "SELECT COALESCE(SUM(octet_length(t.content::text)), 0) FROM tickets t WHERE t.tenant_id = $1 AND t.created_at >= $2"
        )
            .bind(tenant_id)
            .bind(billing_cycle_start)
            .fetch_one(&*self.db_pool)
            .await
            .unwrap_or(0);

        // Calculate storage cost (e.g., $0.01 per GB after first 1GB)
        let storage_gb = storage_usage_bytes as f64 / (1024.0 * 1024.0 * 1024.0);
        let storage_cost = (storage_gb.max(1.0) - 1.0).max(0.0) * 0.01;

        if storage_cost > 0.0 {
            line_items.push(BillingLineItem {
                item_type: "storage".to_string(),
                description: format!("Storage ({:.2} GB × $0.01)", storage_gb.max(1.0)),
                quantity: (storage_gb.max(1.0) * 100.0) as i32, // Store in hundredths of GB
                unit_price: 0.01,
                total_price: storage_cost,
            });
        }

        let current_usage_cost = user_cost + storage_cost;

        let billing = TenantBilling {
            tenant_id: req.tenant_id,
            current_tier: req.new_tier,
            billing_cycle_start: Some(prost_types::Timestamp {
                seconds: billing_cycle_start.timestamp(),
                nanos: billing_cycle_start.timestamp_subsec_nanos() as i32,
            }),
            billing_cycle_end: Some(prost_types::Timestamp {
                seconds: billing_cycle_end.timestamp(),
                nanos: billing_cycle_end.timestamp_subsec_nanos() as i32,
            }),
            currency: "USD".to_string(),
            monthly_cost,
            current_usage_cost,
            line_items,
            payment_method: None, // TODO: Implement payment method retrieval from database
            next_billing_date: Some(prost_types::Timestamp {
                seconds: billing_cycle_end.timestamp(),
                nanos: billing_cycle_end.timestamp_subsec_nanos() as i32,
            }),
        };

        let response = UpdateSubscriptionResponse {
            response: Some(Self::create_success_response(
                "Subscription updated successfully",
                &request_id,
            )),
            tenant: Some(grpc_tenant),
            billing: Some(billing),
            will_invoice: true,
        };

        Ok(Response::new(response))
    }

    #[instrument(skip(self))]
    async fn get_tenant_usage(
        &self,
        request: Request<GetTenantUsageRequest>,
    ) -> StdResult<Response<GetTenantUsageResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("tenant:view") {
            return Err(e);
        }

        let auth_user = request.auth_user()?;
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!(
            "Getting usage for tenant: {} by: {}",
            req.tenant_id, auth_user.email
        );

        // Parse tenant ID
        let tenant_id = match Uuid::from_str(&req.tenant_id) {
            Ok(id) => id,
            Err(_) => {
                let response = GetTenantUsageResponse {
                    response: Some(Self::create_error_response(
                        "Invalid tenant ID format",
                        &request_id,
                    )),
                    usage: None,
                };
                return Ok(Response::new(response));
            }
        };

        // Implement actual usage retrieval from database
        let period_start = chrono::Utc::now() - chrono::Duration::days(30);
        let period_end = chrono::Utc::now();

        // Get user statistics
        let total_users: i64 =
            sqlx::query_scalar("SELECT COUNT(*) FROM users WHERE tenant_id = $1")
                .bind(tenant_id)
                .fetch_one(&*self.db_pool)
                .await
                .unwrap_or(0);

        let active_users: i64 = sqlx::query_scalar(
            "SELECT COUNT(*) FROM users WHERE tenant_id = $1 AND is_active = true",
        )
        .bind(tenant_id)
        .fetch_one(&*self.db_pool)
        .await
        .unwrap_or(0);

        // Get ticket statistics
        let total_tickets: i64 = sqlx::query_scalar(
            "SELECT COUNT(*) FROM tickets WHERE tenant_id = $1 AND is_deleted = false",
        )
        .bind(tenant_id)
        .fetch_one(&*self.db_pool)
        .await
        .unwrap_or(0);

        let open_tickets: i64 = sqlx::query_scalar("SELECT COUNT(*) FROM tickets WHERE tenant_id = $1 AND status != 'Closed' AND is_deleted = false")
            .bind(tenant_id)
            .fetch_one(&*self.db_pool)
            .await
            .unwrap_or(0);

        // Get knowledge article statistics
        let knowledge_articles: i64 = sqlx::query_scalar(
            "SELECT COUNT(*) FROM knowledge_articles WHERE tenant_id = $1 AND is_deleted = false",
        )
        .bind(tenant_id)
        .fetch_one(&*self.db_pool)
        .await
        .unwrap_or(0);

        // Get top users by ticket count
        let top_users_query = r#"
            SELECT
                u.id, u.full_name,
                COUNT(t.id) as tickets_handled,
                COUNT(CASE WHEN t.status = 'Closed' THEN 1 END) as tickets_resolved,
                AVG(EXTRACT(EPOCH FROM (t.updated_at - t.created_at))/3600) as avg_resolution_time
            FROM users u
            LEFT JOIN tickets t ON u.id = t.assigned_agent_id AND t.tenant_id = u.tenant_id
            WHERE u.tenant_id = $1 AND u.is_active = true
            GROUP BY u.id, u.full_name
            HAVING COUNT(t.id) > 0
            ORDER BY tickets_handled DESC
            LIMIT 5
        "#;

        let top_users = sqlx::query(top_users_query)
            .bind(tenant_id)
            .fetch_all(&*self.db_pool)
            .await
            .unwrap_or_default()
            .into_iter()
            .map(|row| UserActivitySummary {
                user_id: row.get::<uuid::Uuid, _>("id").to_string(),
                user_name: row.get("full_name"),
                tickets_handled: row.get::<i64, _>("tickets_handled") as i32,
                tickets_resolved: row.get::<i64, _>("tickets_resolved") as i32,
                avg_resolution_time_hours: row
                    .get::<Option<f64>, _>("avg_resolution_time")
                    .unwrap_or(0.0) as i32,
            })
            .collect();

        // Calculate actual storage usage
        let storage_used_bytes: i64 = sqlx::query_scalar(
            "SELECT COALESCE(SUM(
                octet_length(t.content::text) +
                octet_length(COALESCE(t.title::text, '')) +
                octet_length(COALESCE(t.description::text, ''))
            ), 0)
             FROM tickets t
             WHERE t.tenant_id = $1
               AND t.created_at >= $2
               AND t.created_at <= $3
               AND t.is_deleted = false",
        )
        .bind(tenant_id)
        .bind(period_start)
        .bind(period_end)
        .fetch_one(&*self.db_pool)
        .await
        .unwrap_or(0);

        // Add knowledge articles storage
        let knowledge_storage_bytes: i64 = sqlx::query_scalar(
            "SELECT COALESCE(SUM(
                octet_length(ka.content::text) +
                octet_length(COALESCE(ka.title::text, '')) +
                octet_length(COALESCE(ka.summary::text, ''))
            ), 0)
             FROM knowledge_articles ka
             WHERE ka.tenant_id = $1
               AND ka.created_at >= $2
               AND ka.created_at <= $3
               AND ka.deleted_at IS NULL",
        )
        .bind(tenant_id)
        .bind(period_start)
        .bind(period_end)
        .fetch_one(&*self.db_pool)
        .await
        .unwrap_or(0);

        let total_storage_used = storage_used_bytes + knowledge_storage_bytes;

        // Get actual resolution time statistics
        let (resolved_tickets, avg_resolution_time) = match sqlx::query(
            "SELECT
                COUNT(CASE WHEN t.status = 'Closed' AND t.resolved_at IS NOT NULL THEN 1 END) as resolved_count,
                AVG(EXTRACT(EPOCH FROM (t.resolved_at - t.created_at))/3600) as avg_resolution_hours
             FROM tickets t
             WHERE t.tenant_id = $1
               AND t.created_at >= $2
               AND t.created_at <= $3
               AND t.is_deleted = false"
        )
            .bind(tenant_id)
            .bind(period_start)
            .bind(period_end)
            .fetch_one(&*self.db_pool)
            .await
        {
            Ok(row) => (
                row.get::<Option<i64>, _>("resolved_count").unwrap_or(0) as i32,
                row.get::<Option<f64>, _>("avg_resolution_hours").unwrap_or(0.0) as f32,
            ),
            Err(_) => (0, 0.0), // Fallback values
        };

        // Get knowledge article view statistics
        let knowledge_views: i64 = sqlx::query_scalar(
            "SELECT COALESCE(SUM(view_count), 0)
             FROM knowledge_articles
             WHERE tenant_id = $1
               AND created_at >= $2
               AND created_at <= $3
               AND deleted_at IS NULL",
        )
        .bind(tenant_id)
        .bind(period_start)
        .bind(period_end)
        .fetch_one(&*self.db_pool)
        .await
        .unwrap_or(0);

        // Get customer satisfaction score (simplified - using rating system)
        let satisfaction_score = match sqlx::query(
            "SELECT
                COUNT(CASE WHEN ar.is_helpful = true THEN 1 END) as helpful_count,
                COUNT(CASE WHEN ar.is_helpful = false THEN 1 END) as not_helpful_count
             FROM article_ratings ar
             JOIN knowledge_articles ka ON ar.article_id = ka.id
             WHERE ka.tenant_id = $1
               AND ar.created_at >= $2
               AND ar.created_at <= $3",
        )
        .bind(tenant_id)
        .bind(period_start)
        .bind(period_end)
        .fetch_one(&*self.db_pool)
        .await
        {
            Ok(row) => {
                let helpful_count: i64 = row.get::<Option<i64>, _>("helpful_count").unwrap_or(0);
                let not_helpful_count: i64 =
                    row.get::<Option<i64>, _>("not_helpful_count").unwrap_or(0);
                let total_ratings = helpful_count + not_helpful_count;
                if total_ratings > 0 {
                    (helpful_count as f64 / total_ratings as f64 * 100.0) as f32
                } else {
                    0.0
                }
            }
            Err(_) => 0.0, // Fallback satisfaction score
        };

        let usage = TenantUsage {
            tenant_id: req.tenant_id,
            total_users: total_users as i32,
            active_users: active_users as i32,
            total_tickets: total_tickets as i32,
            open_tickets: open_tickets as i32,
            storage_used_bytes: total_storage_used as i64,
            api_calls_this_month: 0, // TODO: Implement API call tracking with audit logs
            period_start: Some(prost_types::Timestamp {
                seconds: period_start.timestamp(),
                nanos: 0,
            }),
            period_end: Some(prost_types::Timestamp {
                seconds: period_end.timestamp(),
                nanos: period_end.timestamp_subsec_nanos() as i32,
            }),
            metrics: Some(UsageMetrics {
                tickets_created: total_tickets as i32,
                tickets_resolved: resolved_tickets,
                avg_resolution_time_hours: (avg_resolution_time as i32),
                knowledge_articles_created: knowledge_articles as i32,
                knowledge_article_views: knowledge_views as i32,
                customer_satisfaction_score: (satisfaction_score as i32),
                top_users,
            }),
        };

        let response = GetTenantUsageResponse {
            response: Some(Self::create_success_response(
                "Usage retrieved successfully",
                &request_id,
            )),
            usage: Some(usage),
        };

        Ok(Response::new(response))
    }

    #[instrument(skip(self))]
    async fn get_tenant_billing(
        &self,
        request: Request<GetTenantBillingRequest>,
    ) -> StdResult<Response<GetTenantBillingResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("tenant:view") {
            return Err(e);
        }

        let auth_user = request.auth_user()?;
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!(
            "Getting billing for tenant: {} by: {}",
            req.tenant_id, auth_user.email
        );

        // Parse tenant ID
        let tenant_id = match Uuid::from_str(&req.tenant_id) {
            Ok(id) => id,
            Err(_) => {
                let response = GetTenantBillingResponse {
                    response: Some(Self::create_error_response(
                        "Invalid tenant ID format",
                        &request_id,
                    )),
                    billing: None,
                    upcoming_charges: vec![],
                };
                return Ok(Response::new(response));
            }
        };

        // Implement actual billing retrieval from database
        let tenant_info =
            sqlx::query("SELECT subscription_tier, max_users FROM tenants WHERE id = $1")
                .bind(tenant_id)
                .fetch_optional(&*self.db_pool)
                .await
                .unwrap_or_default();

        let (current_tier, max_users) = if let Some(row) = tenant_info {
            let tier: smartticket_shared_database::SubscriptionTier = row.get("subscription_tier");
            let grpc_tier = match tier {
                smartticket_shared_database::SubscriptionTier::Trial => {
                    GrpcSubscriptionTier::Standard // Map Trial to Standard for gRPC
                }
                smartticket_shared_database::SubscriptionTier::Standard => {
                    GrpcSubscriptionTier::Standard
                }
                smartticket_shared_database::SubscriptionTier::Premium => {
                    GrpcSubscriptionTier::Premium
                }
                smartticket_shared_database::SubscriptionTier::Enterprise => {
                    GrpcSubscriptionTier::Enterprise
                }
            };
            (grpc_tier as i32, row.get::<i32, _>("max_users"))
        } else {
            (GrpcSubscriptionTier::Standard as i32, 10)
        };

        // Calculate costs based on tier
        let (base_cost, per_user_cost) = match current_tier {
            t if t == GrpcSubscriptionTier::Standard as i32 => (99.0, 5.0),
            t if t == GrpcSubscriptionTier::Premium as i32 => (299.0, 3.0),
            t if t == GrpcSubscriptionTier::Enterprise as i32 => (999.0, 1.0),
            _ => (99.0, 5.0),
        };

        // Get current active users count
        let current_users_count: i64 = sqlx::query_scalar(
            "SELECT COUNT(*) FROM users WHERE tenant_id = $1 AND is_active = true",
        )
        .bind(tenant_id)
        .fetch_one(&*self.db_pool)
        .await
        .unwrap_or(0);

        let current_users = current_users_count as i32;
        let monthly_cost: f64 = base_cost
            + ((current_users.max(max_users) as f32 - 10.0).max(0.0) * per_user_cost as f32) as f64;

        let billing = TenantBilling {
            tenant_id: req.tenant_id,
            current_tier,
            billing_cycle_start: Some(prost_types::Timestamp {
                seconds: (chrono::Utc::now() - chrono::Duration::days(30)).timestamp(),
                nanos: 0,
            }),
            billing_cycle_end: Some(prost_types::Timestamp {
                seconds: (chrono::Utc::now() + chrono::Duration::days(30)).timestamp(),
                nanos: 0,
            }),
            currency: "USD".to_string(),
            monthly_cost,
            current_usage_cost: 0.0, // TODO: Implement usage cost calculation
            line_items: vec![BillingLineItem {
                item_type: "subscription".to_string(),
                description: format!(
                    "{} Plan (Monthly)",
                    match current_tier {
                        t if t == GrpcSubscriptionTier::Standard as i32 => "Standard",
                        t if t == GrpcSubscriptionTier::Premium as i32 => "Premium",
                        t if t == GrpcSubscriptionTier::Enterprise as i32 => "Enterprise",
                        _ => "Standard",
                    }
                ),
                quantity: 1,
                unit_price: base_cost,
                total_price: base_cost,
            }],
            payment_method: None, // TODO: Implement payment method storage
            next_billing_date: Some(prost_types::Timestamp {
                seconds: (chrono::Utc::now() + chrono::Duration::days(30)).timestamp(),
                nanos: 0,
            }),
        };

        // Calculate upcoming charges
        let upcoming_charges = vec![BillingLineItem {
            item_type: "subscription".to_string(),
            description: format!(
                "{} Plan (Monthly)",
                match current_tier {
                    t if t == GrpcSubscriptionTier::Standard as i32 => "Standard",
                    t if t == GrpcSubscriptionTier::Premium as i32 => "Premium",
                    t if t == GrpcSubscriptionTier::Enterprise as i32 => "Enterprise",
                    _ => "Standard",
                }
            ),
            quantity: 1,
            unit_price: base_cost,
            total_price: base_cost,
        }];

        let response = GetTenantBillingResponse {
            response: Some(Self::create_success_response(
                "Billing retrieved successfully",
                &request_id,
            )),
            billing: Some(billing),
            upcoming_charges,
        };

        Ok(Response::new(response))
    }
}
