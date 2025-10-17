//! gRPC Auth Service Implementation for SmartTicket
//!
//! This module implements the gRPC service handlers for authentication,
//! including login and token refresh.

use bcrypt::verify;
use std::result::Result as StdResult;
use std::str::FromStr;
use std::sync::Arc;
use tonic::{Request, Response, Status};
use tracing::{error, info, instrument};
use uuid::Uuid;

use crate::smartticket_v1::{
    auth_service_server::AuthService as GrpcAuthService, LoginRequest, LoginResponse, RefreshTokenRequest,
    RefreshTokenResponse, Response as ApiResponse, User as GrpcUser, UserRole as GrpcUserRole,
};
use smartticket_shared_database::{AuthService, User, UserRole};

/// gRPC Auth Service implementation
pub struct AuthGrpcService {
    auth_service: Arc<AuthService>,
    db_pool: Arc<sqlx::PgPool>,
}

impl AuthGrpcService {
    /// Create a new gRPC auth service
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
                code: "AUTH_ERROR".to_string(),
                message: message.to_string(),
                details: None,
            }],
            request_id: request_id.to_string(),
        }
    }

    /// Convert database user to gRPC user
    fn db_user_to_grpc(user: &User) -> GrpcUser {
        GrpcUser {
            id: user.id.to_string(),
            tenant_id: user.tenant_id.to_string(),
            email: user.email.clone(),
            username: user.username.clone(),
            full_name: user.full_name.clone(),
            role: match user.role {
                UserRole::SuperAdmin => GrpcUserRole::SuperAdmin as i32,
                UserRole::TenantAdmin => GrpcUserRole::TenantAdmin as i32,
                UserRole::SupportEngineer => GrpcUserRole::SupportEngineer as i32,
                UserRole::CustomerUser => GrpcUserRole::CustomerUser as i32,
                UserRole::Sales => GrpcUserRole::Sales as i32,
            },
            is_active: user.is_active,
            last_login_at: user.last_login_at.map(|dt| prost_types::Timestamp {
                seconds: dt.timestamp(),
                nanos: dt.timestamp_subsec_nanos() as i32,
            }),
        }
    }

    /// Authenticate user with email and password
    async fn authenticate_user(
        &self,
        email: &str,
        password: &str,
        tenant_domain: &str,
    ) -> StdResult<Option<User>, Status> {
        // Set tenant context for RLS
        // Note: We'll handle this after finding the tenant

        // Find tenant by domain
        let tenant = sqlx::query_as::<_, smartticket_shared_database::Tenant>(
            "SELECT id, name, domain, subscription_tier, max_users, data_residency_region, is_active, created_at, updated_at, settings FROM tenants WHERE domain = $1 AND is_active = true"
        )
        .bind(tenant_domain)
        .fetch_optional(&*self.db_pool)
        .await
        .map_err(|e| {
            error!("Failed to find tenant: {}", e);
            Status::internal("Authentication failed")
        })?;

        let tenant = match tenant {
            Some(tenant) => tenant,
            None => return Ok(None),
        };

        // First test manual query without RLS context to understand the issue
        let user = sqlx::query_as::<_, User>(
            r#"
            SELECT id, tenant_id, email, username, full_name, password_hash,
                   role, is_active, last_login_at, created_at, updated_at
            FROM users
            WHERE email = $1 AND tenant_id = $2 AND is_active = true
            "#,
        )
        .bind(email.to_lowercase())
        .bind(&tenant.id)
        .fetch_optional(&*self.db_pool)
        .await
        .map_err(|e| {
            error!("Failed to find user: {}", e);
            Status::internal("Authentication failed")
        })?;

        match user {
            Some(user) => {
                info!("Debug: User found in database, verifying password for email: {}", user.email);
                info!("Debug: Password hash from DB: {}", user.password_hash);
                info!("Debug: Input password length: {}", password.len());

                // Verify password
                match verify(password, &user.password_hash) {
                    Ok(true) => {
                        info!("Debug: Password verification SUCCESS for email: {}", user.email);
                        Ok(Some(user))
                    },
                    Ok(false) => {
                        error!("Debug: Password verification FAILED for email: {} - password mismatch", user.email);
                        Ok(None)
                    },
                    Err(e) => {
                        error!("Debug: Password verification ERROR for email: {} - error: {}", user.email, e);
                        Err(Status::internal("Authentication failed"))
                    }
                }
            }
            None => {
                error!("Debug: User not found in database for email: {}", email);
                Ok(None)
            },
        }
    }

    /// Update user's last login time
    async fn update_last_login(&self, user_id: Uuid, tenant_id: Uuid) -> Result<(), Status> {
        sqlx::query(
            "UPDATE users SET last_login_at = NOW() WHERE id = $1 AND tenant_id = $2"
        )
        .bind(user_id)
        .bind(tenant_id)
        .execute(&*self.db_pool)
        .await
        .map_err(|e| {
            error!("Failed to update last login: {}", e);
            Status::internal("Failed to update login timestamp")
        })?;

        Ok(())
    }
}

#[tonic::async_trait]
impl GrpcAuthService for AuthGrpcService {
    #[instrument(skip(self))]
    async fn login(
        &self,
        request: Request<LoginRequest>,
    ) -> StdResult<Response<LoginResponse>, Status> {
        let req = request.into_inner();
        let request_id = Uuid::new_v4().to_string();

        info!(
            "Login attempt for email: {} in tenant: {}",
            req.email, req.tenant_domain
        );

        // Validate input
        if req.email.trim().is_empty() {
            let response = LoginResponse {
                response: Some(Self::create_error_response(
                    "Email is required",
                    &request_id,
                )),
                access_token: String::new(),
                refresh_token: String::new(),
                user: None,
                expires_at: None,
            };
            return Ok(Response::new(response));
        }

        if req.password.trim().is_empty() {
            let response = LoginResponse {
                response: Some(Self::create_error_response(
                    "Password is required",
                    &request_id,
                )),
                access_token: String::new(),
                refresh_token: String::new(),
                user: None,
                expires_at: None,
            };
            return Ok(Response::new(response));
        }

        // Authenticate user
        info!("Debug: Starting authentication for email: {}", req.email);
        let user = match self
            .authenticate_user(&req.email, &req.password, &req.tenant_domain)
            .await
        {
            Ok(Some(user)) => {
                info!("Debug: User found: {}", user.email);
                user
            },
            Ok(None) => {
                error!("Debug: User not found for email: {} in tenant: {}", req.email, req.tenant_domain);
                let response = LoginResponse {
                    response: Some(Self::create_error_response(
                        "Invalid email, password, or tenant domain",
                        &request_id,
                    )),
                    access_token: String::new(),
                    refresh_token: String::new(),
                    user: None,
                    expires_at: None,
                };
                return Ok(Response::new(response));
            }
            Err(e) => {
                error!("Debug: Authentication error: {}", e);
                return Err(e);
            },
        };

        // Generate JWT tokens
        let access_token = match self.auth_service.generate_token(
            &user.id,
            &user.email,
            &user.username,
            &user.full_name,
            &user.tenant_id,
            &user.role,
        ) {
            Ok(token) => token,
            Err(e) => {
                error!("Failed to generate access token: {}", e);
                let response = LoginResponse {
                    response: Some(Self::create_error_response(
                        "Failed to generate access token",
                        &request_id,
                    )),
                    access_token: String::new(),
                    refresh_token: String::new(),
                    user: None,
                    expires_at: None,
                };
                return Ok(Response::new(response));
            }
        };

        // Generate refresh token (same as access token for now)
        let refresh_token = access_token.clone();

        // Update last login time
        if let Err(e) = self.update_last_login(user.id, user.tenant_id).await {
            error!("Failed to update last login: {}", e);
            // Don't fail the login, just log the error
        }

        // Convert user to gRPC format
        let grpc_user = Self::db_user_to_grpc(&user);

        // Calculate expiration time (24 hours from now)
        let expires_at = chrono::Utc::now() + chrono::Duration::hours(24);

        let response = LoginResponse {
            response: Some(Self::create_success_response(
                "Login successful",
                &request_id,
            )),
            access_token,
            refresh_token,
            user: Some(grpc_user),
            expires_at: Some(prost_types::Timestamp {
                seconds: expires_at.timestamp(),
                nanos: expires_at.timestamp_subsec_nanos() as i32,
            }),
        };

        info!("User logged in successfully: {}", req.email);
        Ok(Response::new(response))
    }

    #[instrument(skip(self))]
    async fn refresh_token(
        &self,
        request: Request<RefreshTokenRequest>,
    ) -> StdResult<Response<RefreshTokenResponse>, Status> {
        let req = request.into_inner();
        let request_id = Uuid::new_v4().to_string();

        info!("Token refresh request received");

        // Validate refresh token
        let claims = match self.auth_service.validate_token(&req.refresh_token) {
            Ok(claims) => claims,
            Err(e) => {
                error!("Invalid refresh token: {}", e);
                let response = RefreshTokenResponse {
                    response: Some(Self::create_error_response(
                        "Invalid or expired refresh token",
                        &request_id,
                    )),
                    access_token: String::new(),
                    refresh_token: String::new(),
                    expires_at: None,
                };
                return Ok(Response::new(response));
            }
        };

        // Parse user ID and tenant ID from claims
        let user_id = match Uuid::from_str(&claims.sub) {
            Ok(id) => id,
            Err(_) => {
                let response = RefreshTokenResponse {
                    response: Some(Self::create_error_response(
                        "Invalid user ID in token",
                        &request_id,
                    )),
                    access_token: String::new(),
                    refresh_token: String::new(),
                    expires_at: None,
                };
                return Ok(Response::new(response));
            }
        };

        let tenant_id = match Uuid::from_str(&claims.tenant_id) {
            Ok(id) => id,
            Err(_) => {
                let response = RefreshTokenResponse {
                    response: Some(Self::create_error_response(
                        "Invalid tenant ID in token",
                        &request_id,
                    )),
                    access_token: String::new(),
                    refresh_token: String::new(),
                    expires_at: None,
                };
                return Ok(Response::new(response));
            }
        };

        // Verify user still exists and is active
        let user = sqlx::query_as::<_, User>(
            r#"
            SELECT id, tenant_id, email, username, full_name, password_hash,
                   role, is_active, last_login_at, created_at, updated_at
            FROM users
            WHERE id = $1 AND tenant_id = $2 AND is_active = true
            "#,
        )
        .bind(user_id)
        .bind(tenant_id)
        .fetch_optional(&*self.db_pool)
        .await
        .map_err(|e| {
            error!("Failed to verify user: {}", e);
            Status::internal("Token refresh failed")
        })?;

        let user = match user {
            Some(user) => user,
            None => {
                let response = RefreshTokenResponse {
                    response: Some(Self::create_error_response(
                        "User not found or inactive",
                        &request_id,
                    )),
                    access_token: String::new(),
                    refresh_token: String::new(),
                    expires_at: None,
                };
                return Ok(Response::new(response));
            }
        };

        // Generate new access token
        let new_access_token = match self.auth_service.generate_token(
            &user.id,
            &user.email,
            &user.username,
            &user.full_name,
            &user.tenant_id,
            &user.role,
        ) {
            Ok(token) => token,
            Err(e) => {
                error!("Failed to generate new access token: {}", e);
                let response = RefreshTokenResponse {
                    response: Some(Self::create_error_response(
                        "Failed to generate new access token",
                        &request_id,
                    )),
                    access_token: String::new(),
                    refresh_token: String::new(),
                    expires_at: None,
                };
                return Ok(Response::new(response));
            }
        };

        // Generate new refresh token
        let new_refresh_token = new_access_token.clone();

        // Calculate expiration time (24 hours from now)
        let expires_at = chrono::Utc::now() + chrono::Duration::hours(24);

        let response = RefreshTokenResponse {
            response: Some(Self::create_success_response(
                "Token refreshed successfully",
                &request_id,
            )),
            access_token: new_access_token,
            refresh_token: new_refresh_token,
            expires_at: Some(prost_types::Timestamp {
                seconds: expires_at.timestamp(),
                nanos: expires_at.timestamp_subsec_nanos() as i32,
            }),
        };

        info!("Token refreshed successfully for user: {}", user.email);
        Ok(Response::new(response))
    }
}