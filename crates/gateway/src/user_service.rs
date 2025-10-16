//! gRPC User Service Implementation for SmartTicket
//!
//! This module implements the gRPC service handlers for user management,
//! including customers and engineers.

use bcrypt::{hash, DEFAULT_COST};
use prost_types;
use sqlx::Row;
use std::result::Result as StdResult;
use std::str::FromStr;
use std::sync::Arc;
use tonic::{Request, Response, Status};
use tracing::{error, info, instrument};
use uuid::Uuid;

use crate::smartticket_v1::{
    user_service_server::UserService, ChangePasswordRequest, ChangePasswordResponse,
    CreateUserRequest, CreateUserResponse, DeleteUserRequest, DeleteUserResponse,
    GetCurrentUserRequest, GetCurrentUserResponse, GetUserPermissionsRequest,
    GetUserPermissionsResponse, GetUserRequest, GetUserResponse, ListUsersRequest,
    ListUsersResponse, PaginationResponse, Permission as GrpcPermission, ResetPasswordRequest,
    ResetPasswordResponse, Response as ApiResponse, UpdateCurrentUserRequest,
    UpdateCurrentUserResponse, UpdateUserRequest, UpdateUserResponse, UpdateUserStatusRequest,
    UpdateUserStatusResponse, User as GrpcUser, UserProfile, UserRole as GrpcUserRole,
};
use crate::{PermissionCheck, RequestExt};
use smartticket_shared_database::{AuthService, Tenant, User, UserRole};
use smartticket_shared_error::{Result, SmartTicketError};

/// gRPC User Service implementation
pub struct UserGrpcService {
    #[allow(dead_code)]
    auth_service: Arc<AuthService>,
    db_pool: Arc<sqlx::PgPool>,
}

impl UserGrpcService {
    /// Create a new gRPC user service
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

    /// Convert database user to gRPC user
    fn db_user_to_grpc(user: &User, _tenant: Option<Tenant>) -> GrpcUser {
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

    /// Convert gRPC user role to database user role
    fn grpc_role_to_db(role: GrpcUserRole) -> Result<UserRole> {
        match role {
            GrpcUserRole::SuperAdmin => Ok(UserRole::SuperAdmin),
            GrpcUserRole::TenantAdmin => Ok(UserRole::TenantAdmin),
            GrpcUserRole::SupportEngineer => Ok(UserRole::SupportEngineer),
            GrpcUserRole::CustomerUser => Ok(UserRole::CustomerUser),
            GrpcUserRole::Sales => Ok(UserRole::Sales),
            GrpcUserRole::Unspecified => Ok(UserRole::CustomerUser), // Default to Customer
        }
    }

    /// Validate email format
    fn validate_email(email: &str) -> Result<()> {
        if email.trim().is_empty() {
            return Err(SmartTicketError::Validation(
                "Email is required".to_string(),
            ));
        }

        if !email.contains('@') || !email.contains('.') {
            return Err(SmartTicketError::Validation(
                "Invalid email format".to_string(),
            ));
        }

        Ok(())
    }

    /// Validate password strength
    fn validate_password(password: &str) -> Result<()> {
        if password.len() < 8 {
            return Err(SmartTicketError::Validation(
                "Password must be at least 8 characters long".to_string(),
            ));
        }

        // Basic password strength validation
        let has_letter = password.chars().any(|c| c.is_alphabetic());
        let has_digit = password.chars().any(|c| c.is_numeric());

        if !has_letter || !has_digit {
            return Err(SmartTicketError::Validation(
                "Password must contain at least one letter and one digit".to_string(),
            ));
        }

        Ok(())
    }

    /// Retrieve user profile from database (simplified version since user_profiles table doesn't exist)
    async fn get_user_profile_from_db(
        &self,
        _user_id: uuid::Uuid,
        _tenant_id: uuid::Uuid,
    ) -> StdResult<Option<UserProfile>, Status> {
        // Since user_profiles table doesn't exist, return None for now
        // The calling functions will fill in the basic user data
        Ok(None)
    }
}

#[tonic::async_trait]
impl UserService for UserGrpcService {
    #[instrument(skip(self))]
    async fn create_user(
        &self,
        request: Request<CreateUserRequest>,
    ) -> StdResult<Response<CreateUserResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("user:create") {
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
            "Creating user with email: {} by: {}",
            req.email, auth_user.email
        );

        // Validate required fields
        if let Err(e) = Self::validate_email(&req.email) {
            let response = CreateUserResponse {
                response: Some(Self::create_error_response(&e.to_string(), &request_id)),
                user: None,
            };
            return Ok(Response::new(response));
        }

        if let Err(e) = Self::validate_password(&req.password) {
            let response = CreateUserResponse {
                response: Some(Self::create_error_response(&e.to_string(), &request_id)),
                user: None,
            };
            return Ok(Response::new(response));
        }

        if req.full_name.trim().is_empty() {
            let response = CreateUserResponse {
                response: Some(Self::create_error_response(
                    "Full name is required",
                    &request_id,
                )),
                user: None,
            };
            return Ok(Response::new(response));
        }

        // Validate and convert role
        let user_role = match Self::grpc_role_to_db(req.role()) {
            Ok(role) => role,
            Err(e) => {
                let response = CreateUserResponse {
                    response: Some(Self::create_error_response(&e.to_string(), &request_id)),
                    user: None,
                };
                return Ok(Response::new(response));
            }
        };

        // Check if user can create this role (only SuperAdmin can create SuperAdmin, etc.)
        match (&auth_user.role, &user_role) {
            (UserRole::SuperAdmin, _) => {} // SystemAdmin can create any role
            (UserRole::TenantAdmin, UserRole::SuperAdmin) => {
                let response = CreateUserResponse {
                    response: Some(Self::create_error_response(
                        "TenantAdmin cannot create SuperAdmin users",
                        &request_id,
                    )),
                    user: None,
                };
                return Ok(Response::new(response));
            }
            (UserRole::SupportEngineer, UserRole::SuperAdmin | UserRole::TenantAdmin) => {
                let response = CreateUserResponse {
                    response: Some(Self::create_error_response(
                        "SupportEngineer cannot create admin users",
                        &request_id,
                    )),
                    user: None,
                };
                return Ok(Response::new(response));
            }
            (UserRole::CustomerUser | UserRole::Sales, _) => {
                let response = CreateUserResponse {
                    response: Some(Self::create_error_response(
                        "Insufficient permissions to create users",
                        &request_id,
                    )),
                    user: None,
                };
                return Ok(Response::new(response));
            }
            _ => {} // Allowed combinations
        }

        // Hash password
        let password_hash = match hash(&req.password, DEFAULT_COST) {
            Ok(hash) => hash,
            Err(e) => {
                error!("Failed to hash password: {}", e);
                let response = CreateUserResponse {
                    response: Some(Self::create_error_response(
                        "Failed to process password",
                        &request_id,
                    )),
                    user: None,
                };
                return Ok(Response::new(response));
            }
        };

        // Insert user into database
        let query = r#"
            INSERT INTO users (id, tenant_id, email, username, full_name, password_hash, role, is_active, created_at, updated_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
            RETURNING id, tenant_id, email, username, full_name, password_hash, role, is_active, last_login_at, created_at, updated_at
        "#;

        let new_user = User::new(
            auth_user.tenant_id,
            req.email.trim().to_lowercase(),
            req.username.trim().to_string(),
            req.full_name.trim().to_string(),
            password_hash,
            user_role,
        );

        let result_user = sqlx::query_as::<_, User>(query)
            .bind(new_user.id)
            .bind(new_user.tenant_id)
            .bind(&new_user.email)
            .bind(&new_user.username)
            .bind(&new_user.full_name)
            .bind(&new_user.password_hash)
            .bind(new_user.role)
            .bind(new_user.is_active)
            .bind(new_user.created_at)
            .bind(new_user.updated_at)
            .fetch_one(&*self.db_pool)
            .await
            .map_err(|e| {
                error!("Failed to create user in database: {}", e);
                Status::internal("Failed to create user")
            })?;

        // Convert to gRPC response
        let grpc_user = Self::db_user_to_grpc(&result_user, None);

        let response = CreateUserResponse {
            response: Some(Self::create_success_response(
                "User created successfully",
                &request_id,
            )),
            user: Some(grpc_user),
        };

        info!("Successfully created user: {}", req.email);
        Ok(Response::new(response))
    }

    #[instrument(skip(self))]
    async fn get_user(
        &self,
        request: Request<GetUserRequest>,
    ) -> StdResult<Response<GetUserResponse>, Status> {
        // Check authentication and authorization
        let auth_user = match request.auth_user() {
            Ok(user) => user,
            Err(e) => return Err(e),
        };

        let permission = if auth_user.permissions.contains(&"user:view".to_string()) ||
            matches!(auth_user.role, UserRole::SuperAdmin | UserRole::TenantAdmin) {
            "user:view"
        } else {
            "user:view_own"
        };

        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!("Getting user: {} by: {}", req.user_id, auth_user.email);

        // Parse user ID
        let user_id = match Uuid::from_str(&req.user_id) {
            Ok(id) => id,
            Err(_) => {
                let response = GetUserResponse {
                    response: Some(Self::create_error_response(
                        "Invalid user ID format",
                        &request_id,
                    )),
                    user: None,
                    profile: None,
                };
                return Ok(Response::new(response));
            }
        };

        // Check if user can access this user (only own profile unless admin)
        if permission == "user:view_own" && auth_user.id != user_id {
            let response = GetUserResponse {
                response: Some(Self::create_error_response(
                    "Cannot access other user profiles",
                    &request_id,
                )),
                user: None,
                profile: None,
            };
            return Ok(Response::new(response));
        }

        // Retrieve user from database
        let user = sqlx::query_as::<_, User>(
            r#"
            SELECT id, tenant_id, email, username, full_name, password_hash,
                   role, is_active, last_login_at, created_at, updated_at
            FROM users
            WHERE id = $1 AND tenant_id = $2 AND is_active = true
            "#,
        )
        .bind(user_id)
        .bind(auth_user.tenant_id)
        .fetch_optional(&*self.db_pool)
        .await
        .map_err(|e| {
            error!("Failed to retrieve user from database: {}", e);
            Status::internal("Failed to retrieve user")
        })?;

        let user = match user {
            Some(user) => user,
            None => {
                let response = GetUserResponse {
                    response: Some(Self::create_error_response("User not found", &request_id)),
                    user: None,
                    profile: None,
                };
                return Ok(Response::new(response));
            }
        };

        let grpc_user = Self::db_user_to_grpc(&user, None);

        // Retrieve user profile from database
        let profile = match self
            .get_user_profile_from_db(user.id, auth_user.tenant_id)
            .await
        {
            Ok(mut profile_opt) => {
                if let Some(ref mut profile) = profile_opt {
                    // Fill in user data from user record
                    profile.email = user.email.clone();
                    profile.username = user.username.clone();
                    profile.full_name = user.full_name.clone();
                }
                profile_opt
            }
            Err(e) => {
                error!("Failed to retrieve user profile: {}", e);
                None
            }
        };

        let response = GetUserResponse {
            response: Some(Self::create_success_response(
                "User retrieved successfully",
                &request_id,
            )),
            user: Some(grpc_user),
            profile,
        };

        Ok(Response::new(response))
    }

    #[instrument(skip(self))]
    async fn get_current_user(
        &self,
        request: Request<GetCurrentUserRequest>,
    ) -> StdResult<Response<GetCurrentUserResponse>, Status> {
        // Check authentication
        let auth_user = request.auth_user()?;
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!("Getting current user: {}", auth_user.email);

        // Retrieve current user from database
        let user = sqlx::query_as::<_, User>(
            r#"
            SELECT id, tenant_id, email, username, full_name, password_hash,
                   role, is_active, last_login_at, created_at, updated_at
            FROM users
            WHERE id = $1 AND tenant_id = $2 AND is_active = true
            "#,
        )
        .bind(auth_user.id)
        .bind(auth_user.tenant_id)
        .fetch_optional(&*self.db_pool)
        .await
        .map_err(|e| {
            error!("Failed to retrieve current user from database: {}", e);
            Status::internal("Failed to retrieve current user")
        })?;

        let user = match user {
            Some(user) => user,
            None => {
                let response = GetCurrentUserResponse {
                    response: Some(Self::create_error_response(
                        "Current user not found",
                        &request_id,
                    )),
                    user: None,
                    profile: None,
                    permissions: vec![],
                };
                return Ok(Response::new(response));
            }
        };

        let grpc_user = Self::db_user_to_grpc(&user, None);

        // Retrieve current user profile from database
        let profile = match self
            .get_user_profile_from_db(user.id, auth_user.tenant_id)
            .await
        {
            Ok(mut profile_opt) => {
                if let Some(ref mut profile) = profile_opt {
                    // Fill in user data from user record
                    profile.email = user.email.clone();
                    profile.username = user.username.clone();
                    profile.full_name = user.full_name.clone();
                }
                profile_opt
            }
            Err(e) => {
                error!("Failed to retrieve current user profile: {}", e);
                None
            }
        };

        // Convert permissions to gRPC format
        let grpc_permissions: Vec<GrpcPermission> = auth_user
            .permissions
            .into_iter()
            .map(|perm| GrpcPermission {
                resource: "ticket".to_string(), // TODO: Parse permission properly
                actions: vec![perm],
            })
            .collect();

        let response = GetCurrentUserResponse {
            response: Some(Self::create_success_response(
                "Current user retrieved successfully",
                &request_id,
            )),
            user: Some(grpc_user),
            profile,
            permissions: grpc_permissions,
        };

        Ok(Response::new(response))
    }

    #[instrument(skip(self))]
    async fn list_users(
        &self,
        request: Request<ListUsersRequest>,
    ) -> StdResult<Response<ListUsersResponse>, Status> {
        // Check authentication
        let auth_user = match request.auth_user() {
            Ok(user) => user,
            Err(e) => return Err(e),
        };

        // Check if user has permission to view users
        let has_permission = auth_user.permissions.contains(&"user:view".to_string()) ||
            matches!(auth_user.role, UserRole::SuperAdmin | UserRole::TenantAdmin);

        if !has_permission {
            return Err(Status::permission_denied("Insufficient permissions. Required: user:view"));
        }

        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!("Listing users by: {}", auth_user.email);

        // Implement actual user listing from database with filtering and pagination
        let page_size = req.pagination.as_ref().map_or(20, |p| p.page_size);
        let page_token = req.pagination.as_ref().and_then(|p| {
            if p.page_token.is_empty() {
                None
            } else {
                Some(p.page_token.clone())
            }
        });

        // Parse page token to get offset
        let offset = page_token
            .and_then(|token| token.parse::<usize>().ok())
            .unwrap_or(0);

        // Build base query
        let mut query_builder = sqlx::QueryBuilder::new(
            "SELECT id, tenant_id, email, username, full_name, password_hash, ",
        );
        query_builder.push("role, is_active, last_login_at, ");
        query_builder.push("created_at, updated_at FROM users WHERE tenant_id = ");
        query_builder.push_bind(auth_user.tenant_id);

        // Apply is_active filter
        query_builder.push(" AND is_active = ");
        query_builder.push_bind(req.is_active);

        // Apply role filter
        let role_strings: Vec<String> = req
            .roles
            .iter()
            .filter_map(|&role| match role.try_into() {
                Ok(GrpcUserRole::CustomerUser) => Some("customer".to_string()),
                Ok(GrpcUserRole::SupportEngineer) => Some("support_engineer".to_string()),
                Ok(GrpcUserRole::TenantAdmin) => Some("admin".to_string()),
                Ok(GrpcUserRole::SuperAdmin) => Some("system_admin".to_string()),
                Ok(GrpcUserRole::Sales) => Some("sales".to_string()),
                _ => None,
            })
            .collect();

        if !role_strings.is_empty() {
            query_builder.push(" AND role = ANY(");
            query_builder.push_bind(&role_strings);
            query_builder.push(")");
        }

        // Apply search filter
        let search_term = if !req.search.is_empty() {
            format!("%{}%", req.search)
        } else {
            String::new()
        };

        if !search_term.is_empty() {
            query_builder.push(" AND (email ILIKE ");
            query_builder.push_bind(&search_term);
            query_builder.push(" OR username ILIKE ");
            query_builder.push_bind(&search_term);
            query_builder.push(" OR full_name ILIKE ");
            query_builder.push_bind(&search_term);
            query_builder.push(")");
        }

        // Count total users for pagination - build separate count query to avoid parameter binding issues
        let mut count_query_builder = sqlx::QueryBuilder::new("SELECT COUNT(*) FROM users WHERE tenant_id = ");
        count_query_builder.push_bind(auth_user.tenant_id);

        // Apply is_active filter for count query
        count_query_builder.push(" AND is_active = ");
        count_query_builder.push_bind(req.is_active);

        // Apply role filter for count query
        if !role_strings.is_empty() {
            count_query_builder.push(" AND role = ANY(");
            count_query_builder.push_bind(&role_strings);
            count_query_builder.push(")");
        }

        // Apply search filter for count query
        if !search_term.is_empty() {
            count_query_builder.push(" AND (email ILIKE ");
            count_query_builder.push_bind(&search_term);
            count_query_builder.push(" OR username ILIKE ");
            count_query_builder.push_bind(&search_term);
            count_query_builder.push(" OR full_name ILIKE ");
            count_query_builder.push_bind(&search_term);
            count_query_builder.push(")");
        }

        let total_count: i64 = count_query_builder
            .build_query_scalar()
            .fetch_one(&*self.db_pool)
            .await
            .map_err(|e| {
                error!("Failed to count users: {}", e);
                Status::internal("Failed to count users")
            })?;

        // Add ordering and pagination
        query_builder.push(" ORDER BY created_at DESC LIMIT ");
        query_builder.push_bind(page_size as i64 + 1); // Fetch one extra to check if there are more
        query_builder.push(" OFFSET ");
        query_builder.push_bind(offset as i64);

        let query = query_builder.build_query_as::<User>();

        let users = query.fetch_all(&*self.db_pool).await.map_err(|e| {
            error!("Failed to retrieve users from database: {}", e);
            Status::internal("Failed to retrieve users")
        })?;

        // Determine if there are more users
        let has_more = users.len() > page_size as usize;
        let users_for_response = if has_more {
            users.into_iter().take(page_size as usize).collect()
        } else {
            users
        };

        // Convert to gRPC format
        let grpc_users: Vec<GrpcUser> = users_for_response
            .into_iter()
            .map(|user| Self::db_user_to_grpc(&user, None))
            .collect();

        // Generate next page token if there are more users
        let next_page_token = if has_more {
            (offset + page_size as usize).to_string()
        } else {
            String::new()
        };

        let response = ListUsersResponse {
            response: Some(Self::create_success_response(
                "Users listed successfully",
                &request_id,
            )),
            users: grpc_users,
            pagination: Some(PaginationResponse {
                total_count: total_count as i32,
                page_size,
                next_page_token,
                prev_page_token: if offset > 0 {
                    (offset - page_size as usize).to_string()
                } else {
                    String::new()
                },
            }),
        };

        Ok(Response::new(response))
    }

    #[instrument(skip(self))]
    async fn update_user_status(
        &self,
        request: Request<UpdateUserStatusRequest>,
    ) -> StdResult<Response<UpdateUserStatusResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("user:update") {
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
            "Updating user status: {} -> {} by: {}",
            req.user_id, req.is_active, auth_user.email
        );

        // Parse user ID
        let user_id = match Uuid::from_str(&req.user_id) {
            Ok(id) => id,
            Err(_) => {
                let response = UpdateUserStatusResponse {
                    response: Some(Self::create_error_response(
                        "Invalid user ID format",
                        &request_id,
                    )),
                    user: None,
                };
                return Ok(Response::new(response));
            }
        };

        // Prevent user from deactivating themselves
        if auth_user.id == user_id && !req.is_active {
            let response = UpdateUserStatusResponse {
                response: Some(Self::create_error_response(
                    "Cannot deactivate your own account",
                    &request_id,
                )),
                user: None,
            };
            return Ok(Response::new(response));
        }

        // Implement actual user status update in database
        let result = sqlx::query_as::<_, User>(
            r#"
            UPDATE users
            SET is_active = $1, updated_at = NOW()
            WHERE id = $2 AND tenant_id = $3
            RETURNING id, tenant_id, email, username, full_name, password_hash,
                      role, is_active, last_login_at, created_at, updated_at
            "#,
        )
        .bind(req.is_active)
        .bind(user_id)
        .bind(auth_user.tenant_id)
        .fetch_optional(&*self.db_pool)
        .await
        .map_err(|e| {
            error!("Failed to update user status in database: {}", e);
            Status::internal("Failed to update user status")
        })?;

        let user = match result {
            Some(user) => user,
            None => {
                let response = UpdateUserStatusResponse {
                    response: Some(Self::create_error_response("User not found", &request_id)),
                    user: None,
                };
                return Ok(Response::new(response));
            }
        };

        let grpc_user = Self::db_user_to_grpc(&user, None);

        let response = UpdateUserStatusResponse {
            response: Some(Self::create_success_response(
                if req.is_active {
                    "User activated successfully"
                } else {
                    "User deactivated successfully"
                },
                &request_id,
            )),
            user: Some(grpc_user),
        };

        Ok(Response::new(response))
    }

    #[instrument(skip(self))]
    async fn update_user(
        &self,
        request: Request<UpdateUserRequest>,
    ) -> StdResult<Response<UpdateUserResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("user:update") {
            return Err(e);
        }

        let auth_user = request.auth_user()?;
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!("Updating user: {} by: {}", req.user_id, auth_user.email);

        // Parse user ID
        let user_id = match Uuid::from_str(&req.user_id) {
            Ok(id) => id,
            Err(_) => {
                let response = UpdateUserResponse {
                    response: Some(Self::create_error_response(
                        "Invalid user ID format",
                        &request_id,
                    )),
                    user: None,
                };
                return Ok(Response::new(response));
            }
        };

        // Check if user can update this user (admin can update any user, users can update their own profile)
        let can_update_any_user = auth_user.permissions.contains(&"user:update".to_string());
        let is_own_profile = auth_user.id == user_id;

        if !can_update_any_user && !is_own_profile {
            let response = UpdateUserResponse {
                response: Some(Self::create_error_response(
                    "Insufficient permissions to update this user",
                    &request_id,
                )),
                user: None,
            };
            return Ok(Response::new(response));
        }

        // Validate and convert role if provided
        let user_role = if req.role != GrpcUserRole::Unspecified as i32 {
            match Self::grpc_role_to_db(req.role.try_into().unwrap_or(GrpcUserRole::Unspecified)) {
                Ok(role) => Some(role),
                Err(e) => {
                    let response = UpdateUserResponse {
                        response: Some(Self::create_error_response(&e.to_string(), &request_id)),
                        user: None,
                    };
                    return Ok(Response::new(response));
                }
            }
        } else {
            None // Keep existing role
        };

        // Check if user can assign this role
        if let Some(new_role) = &user_role {
            match (&auth_user.role, &new_role) {
                (UserRole::SuperAdmin, _) => {} // SuperAdmin can assign any role
                (UserRole::TenantAdmin, UserRole::SuperAdmin) => {
                    let response = UpdateUserResponse {
                        response: Some(Self::create_error_response(
                            "TenantAdmin cannot assign SuperAdmin role",
                            &request_id,
                        )),
                        user: None,
                    };
                    return Ok(Response::new(response));
                }
                (UserRole::SupportEngineer, UserRole::SuperAdmin | UserRole::TenantAdmin) => {
                    let response = UpdateUserResponse {
                        response: Some(Self::create_error_response(
                            "SupportEngineer cannot assign admin roles",
                            &request_id,
                        )),
                        user: None,
                    };
                    return Ok(Response::new(response));
                }
                (UserRole::CustomerUser | UserRole::Sales, _) => {
                    let response = UpdateUserResponse {
                        response: Some(Self::create_error_response(
                            "Insufficient permissions to assign roles",
                            &request_id,
                        )),
                        user: None,
                    };
                    return Ok(Response::new(response));
                }
                _ => {} // Allowed combinations
            }
        }

        // Implement actual user update in database
        let mut query_builder = sqlx::QueryBuilder::new("UPDATE users SET updated_at = NOW()");
        let mut has_updates = false;

        // Update email if provided
        if !req.email.is_empty() {
            if Self::validate_email(&req.email).is_ok() {
                query_builder.push(", email = ");
                query_builder.push_bind(req.email.trim().to_lowercase());
                has_updates = true;
            }
        }

        // Update username if provided
        if !req.username.is_empty() {
            query_builder.push(", username = ");
            query_builder.push_bind(req.username.trim().to_string());
            has_updates = true;
        }

        // Update full name if provided
        if !req.full_name.is_empty() {
            query_builder.push(", full_name = ");
            query_builder.push_bind(req.full_name.trim().to_string());
            has_updates = true;
        }

        // Update role if provided
        if let Some(new_role) = user_role {
            query_builder.push(", role = ");
            query_builder.push_bind(new_role);
            has_updates = true;
        }

        if !has_updates {
            let response = UpdateUserResponse {
                response: Some(Self::create_error_response(
                    "No valid fields to update",
                    &request_id,
                )),
                user: None,
            };
            return Ok(Response::new(response));
        }

        query_builder.push(" WHERE id = ");
        query_builder.push_bind(user_id);
        query_builder.push(" AND tenant_id = ");
        query_builder.push_bind(auth_user.tenant_id);
        query_builder.push(" RETURNING id, tenant_id, email, username, full_name, password_hash, role, is_active, last_login_at, created_at, updated_at");

        let query = query_builder.build_query_as::<User>();

        let user = query.fetch_optional(&*self.db_pool).await.map_err(|e| {
            error!("Failed to update user in database: {}", e);
            Status::internal("Failed to update user")
        })?;

        let user = match user {
            Some(user) => user,
            None => {
                let response = UpdateUserResponse {
                    response: Some(Self::create_error_response("User not found", &request_id)),
                    user: None,
                };
                return Ok(Response::new(response));
            }
        };

        let grpc_user = Self::db_user_to_grpc(&user, None);

        let response = UpdateUserResponse {
            response: Some(Self::create_success_response(
                "User updated successfully",
                &request_id,
            )),
            user: Some(grpc_user),
        };

        Ok(Response::new(response))
    }

    #[instrument(skip(self))]
    async fn update_current_user(
        &self,
        request: Request<UpdateCurrentUserRequest>,
    ) -> StdResult<Response<UpdateCurrentUserResponse>, Status> {
        // Check authentication
        let auth_user = request.auth_user()?;
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!("Updating current user profile by: {}", auth_user.email);

        // Implement actual user profile update in database
        let mut query_builder = sqlx::QueryBuilder::new("UPDATE users SET updated_at = NOW()");
        let mut has_updates = false;

        // Update full name if provided
        if !req.full_name.is_empty() {
            query_builder.push(", full_name = ");
            query_builder.push_bind(req.full_name.trim().to_string());
            has_updates = true;
        }

        if !has_updates {
            let response = UpdateCurrentUserResponse {
                response: Some(Self::create_error_response(
                    "No valid fields to update",
                    &request_id,
                )),
                profile: None,
            };
            return Ok(Response::new(response));
        }

        query_builder.push(" WHERE id = ");
        query_builder.push_bind(auth_user.id);
        query_builder.push(" AND tenant_id = ");
        query_builder.push_bind(auth_user.tenant_id);
        query_builder.push(" RETURNING id, tenant_id, email, username, full_name, password_hash, role, is_active, last_login_at, created_at, updated_at");

        let query = query_builder.build_query_as::<User>();

        let updated_user = query.fetch_optional(&*self.db_pool).await.map_err(|e| {
            error!("Failed to update user profile in database: {}", e);
            Status::internal("Failed to update user profile")
        })?;

        let updated_user = match updated_user {
            Some(user) => user,
            None => {
                let response = UpdateCurrentUserResponse {
                    response: Some(Self::create_error_response("User not found", &request_id)),
                    profile: None,
                };
                return Ok(Response::new(response));
            }
        };

        // Since user_preferences table doesn't exist, use empty preferences for now
        let preferences = Some(prost_types::Struct {
            fields: std::collections::BTreeMap::new(),
        });

        // Create profile response from updated user data
        let profile = UserProfile {
            user_id: auth_user.id.to_string(),
            email: updated_user.email.clone(),
            username: updated_user.username.clone(),
            full_name: updated_user.full_name.clone(),
            phone: req.phone.clone(),
            timezone: req.timezone.clone(),
            language: req.language.clone(),
            preferences,
            created_at: Some(prost_types::Timestamp {
                seconds: updated_user.created_at.timestamp(),
                nanos: updated_user.created_at.timestamp_subsec_nanos() as i32,
            }),
            updated_at: Some(prost_types::Timestamp {
                seconds: updated_user.updated_at.timestamp(),
                nanos: updated_user.updated_at.timestamp_subsec_nanos() as i32,
            }),
        };

        let response = UpdateCurrentUserResponse {
            response: Some(Self::create_success_response(
                "Profile updated successfully",
                &request_id,
            )),
            profile: Some(profile),
        };

        Ok(Response::new(response))
    }

    #[instrument(skip(self))]
    async fn delete_user(
        &self,
        request: Request<DeleteUserRequest>,
    ) -> StdResult<Response<DeleteUserResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("user:delete") {
            return Err(e);
        }

        let auth_user = request.auth_user()?;
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!("Deleting user: {} by: {}", req.user_id, auth_user.email);

        // Parse user ID
        let user_id = match Uuid::from_str(&req.user_id) {
            Ok(id) => id,
            Err(_) => {
                let response = DeleteUserResponse {
                    response: Some(Self::create_error_response(
                        "Invalid user ID format",
                        &request_id,
                    )),
                };
                return Ok(Response::new(response));
            }
        };

        // Prevent user from deleting themselves
        if auth_user.id == user_id {
            let response = DeleteUserResponse {
                response: Some(Self::create_error_response(
                    "Cannot delete your own account",
                    &request_id,
                )),
            };
            return Ok(Response::new(response));
        }

        // Implement actual user deletion (soft delete) in database
        let result = sqlx::query(
            r#"
            UPDATE users
            SET is_active = false, updated_at = NOW()
            WHERE id = $1 AND tenant_id = $2 AND is_active = true
            "#,
        )
        .bind(user_id)
        .bind(auth_user.tenant_id)
        .execute(&*self.db_pool)
        .await
        .map_err(|e| {
            error!("Failed to delete user in database: {}", e);
            Status::internal("Failed to delete user")
        })?;

        if result.rows_affected() == 0 {
            let response = DeleteUserResponse {
                response: Some(Self::create_error_response(
                    "User not found or already deleted",
                    &request_id,
                )),
            };
            return Ok(Response::new(response));
        }

        let response = DeleteUserResponse {
            response: Some(Self::create_success_response(
                "User deleted successfully",
                &request_id,
            )),
        };

        Ok(Response::new(response))
    }

    #[instrument(skip(self))]
    async fn change_password(
        &self,
        request: Request<ChangePasswordRequest>,
    ) -> StdResult<Response<ChangePasswordResponse>, Status> {
        // Check authentication
        let auth_user = request.auth_user()?;
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!("Changing password for user: {}", auth_user.email);

        // Validate password requirements
        if let Err(e) = Self::validate_password(&req.new_password) {
            let response = ChangePasswordResponse {
                response: Some(Self::create_error_response(&e.to_string(), &request_id)),
            };
            return Ok(Response::new(response));
        }

        // Implement actual password change in database
        // Hash the new password
        let new_password_hash = hash(&req.new_password, DEFAULT_COST).map_err(|e| {
            error!("Failed to hash new password: {}", e);
            Status::internal("Failed to process password")
        })?;

        // Update password in database
        let result = sqlx::query(
            r#"
            UPDATE users
            SET password_hash = $1, updated_at = NOW()
            WHERE id = $2 AND tenant_id = $3 AND is_active = true
            "#,
        )
        .bind(new_password_hash)
        .bind(auth_user.id)
        .bind(auth_user.tenant_id)
        .execute(&*self.db_pool)
        .await
        .map_err(|e| {
            error!("Failed to change password in database: {}", e);
            Status::internal("Failed to change password")
        })?;

        if result.rows_affected() == 0 {
            let response = ChangePasswordResponse {
                response: Some(Self::create_error_response("User not found", &request_id)),
            };
            return Ok(Response::new(response));
        }

        let response = ChangePasswordResponse {
            response: Some(Self::create_success_response(
                "Password changed successfully",
                &request_id,
            )),
        };

        Ok(Response::new(response))
    }

    #[instrument(skip(self))]
    async fn reset_password(
        &self,
        request: Request<ResetPasswordRequest>,
    ) -> StdResult<Response<ResetPasswordResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("user:reset_password") {
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
            "Resetting password for user: {} by: {}",
            req.user_id, auth_user.email
        );

        // Parse user ID
        let _user_id = match Uuid::from_str(&req.user_id) {
            Ok(id) => id,
            Err(_) => {
                let response = ResetPasswordResponse {
                    response: Some(Self::create_error_response(
                        "Invalid user ID format",
                        &request_id,
                    )),
                    temporary_password: String::new(),
                };
                return Ok(Response::new(response));
            }
        };

        // Generate temporary password
        let temporary_password = format!("temp_{}", Uuid::new_v4().to_string()[..8].to_string());
        let _hashed_temp_password = match hash(&temporary_password, DEFAULT_COST) {
            Ok(hash) => hash,
            Err(e) => {
                error!("Failed to hash temporary password: {}", e);
                let response = ResetPasswordResponse {
                    response: Some(Self::create_error_response(
                        "Failed to generate temporary password",
                        &request_id,
                    )),
                    temporary_password: String::new(),
                };
                return Ok(Response::new(response));
            }
        };

        // Implement actual password reset in database and send email
        // Update password in database with temporary password
        let result = sqlx::query(
            r#"
            UPDATE users
            SET password_hash = $1, updated_at = NOW()
            WHERE id = $2 AND tenant_id = $3 AND is_active = true
            "#,
        )
        .bind(&_hashed_temp_password)
        .bind(_user_id)
        .bind(auth_user.tenant_id)
        .execute(&*self.db_pool)
        .await
        .map_err(|e| {
            error!("Failed to reset password in database: {}", e);
            Status::internal("Failed to reset password")
        })?;

        if result.rows_affected() == 0 {
            let response = ResetPasswordResponse {
                response: Some(Self::create_error_response("User not found", &request_id)),
                temporary_password: String::new(),
            };
            return Ok(Response::new(response));
        }

        // TODO: Send email with temporary password
        info!(
            "Temporary password for user {}: {}",
            _user_id, temporary_password
        );

        let response = ResetPasswordResponse {
            response: Some(Self::create_success_response(
                "Password reset successfully",
                &request_id,
            )),
            temporary_password,
        };

        Ok(Response::new(response))
    }

    #[instrument(skip(self))]
    async fn get_user_permissions(
        &self,
        request: Request<GetUserPermissionsRequest>,
    ) -> StdResult<Response<GetUserPermissionsResponse>, Status> {
        // Check authentication
        let auth_user = request.auth_user()?;
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!("Getting permissions for user: {}", auth_user.email);

        // Parse user ID
        let _user_id = match Uuid::from_str(&req.user_id) {
            Ok(id) => id,
            Err(_) => {
                let response = GetUserPermissionsResponse {
                    response: Some(Self::create_error_response(
                        "Invalid user ID format",
                        &request_id,
                    )),
                    permissions: vec![],
                };
                return Ok(Response::new(response));
            }
        };

        // Implement actual user permissions retrieval from database
        let permissions = sqlx::query(
            r#"
            SELECT DISTINCT unnest(permissions) as permission
            FROM roles r
            JOIN user_role_assignments ura ON r.id = ura.role_id
            WHERE ura.user_id = $1
              AND r.tenant_id = $2
              AND ura.is_active = true
              AND r.is_active = true
              AND (ura.expires_at IS NULL OR ura.expires_at > NOW())
            "#,
        )
        .bind(_user_id)
        .bind(auth_user.tenant_id)
        .fetch_all(&*self.db_pool)
        .await
        .map_err(|e| {
            error!("Failed to retrieve user permissions from database: {}", e);
            Status::internal("Failed to retrieve user permissions")
        })?;

        // Group permissions by resource
        let mut permission_map: std::collections::HashMap<String, Vec<String>> =
            std::collections::HashMap::new();
        for row in permissions {
            let permission_str: String = row.get(0);
            // Parse permission format like "ticket:view" or "user:create"
            if let Some((resource, action)) = permission_str.split_once(':') {
                permission_map
                    .entry(resource.to_string())
                    .or_insert_with(Vec::new)
                    .push(action.to_string());
            }
        }

        // Convert to gRPC format
        let grpc_permissions: Vec<GrpcPermission> = permission_map
            .into_iter()
            .map(|(resource, actions)| GrpcPermission { resource, actions })
            .collect();

        let response = GetUserPermissionsResponse {
            response: Some(Self::create_success_response(
                "Permissions retrieved successfully",
                &request_id,
            )),
            permissions: grpc_permissions,
        };

        Ok(Response::new(response))
    }
}
