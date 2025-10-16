//! gRPC Role and Permission Service Implementation for SmartTicket
//!
//! This module implements the gRPC service handlers for role and permission management,
//! including CRUD operations for roles, permission assignments, and user role management.

use chrono;
use sqlx::Row;
use std::result::Result as StdResult;
use std::str::FromStr;
use std::sync::Arc;
use tonic::{Request, Response, Status};
use tracing::{error, info, instrument};
use uuid::Uuid;

// Database Role struct for sqlx queries
#[derive(Debug)]
struct DbRole {
    id: Uuid,
    tenant_id: Uuid,
    name: String,
    description: Option<String>,
    is_system_role: bool,
    is_active: bool,
    created_by: Uuid,
    updated_by: Uuid,
    created_at: Option<chrono::DateTime<chrono::Utc>>,
    updated_at: Option<chrono::DateTime<chrono::Utc>>,
}

// Implement sqlx::FromRow for DbRole
impl sqlx::FromRow<'_, sqlx::postgres::PgRow> for DbRole {
    fn from_row(row: &sqlx::postgres::PgRow) -> sqlx::Result<Self> {
        Ok(DbRole {
            id: row.try_get("id")?,
            tenant_id: row.try_get("tenant_id")?,
            name: row.try_get("name")?,
            description: row.try_get("description")?,
            is_system_role: row.try_get("is_system_role")?,
            is_active: row.try_get("is_active")?,
            created_by: row.try_get("created_by")?,
            updated_by: row.try_get("updated_by")?,
            created_at: row.try_get("created_at")?,
            updated_at: row.try_get("updated_at")?,
        })
    }
}

use crate::smartticket_v1::{
    role_permission_service_server::RolePermissionService, AssignPermissionsToRoleRequest,
    AssignPermissionsToRoleResponse, AssignRoleToUserRequest, AssignRoleToUserResponse,
    CreateRoleRequest, CreateRoleResponse, DeleteRoleRequest, DeleteRoleResponse,
    GetRolePermissionsRequest, GetRolePermissionsResponse, GetRoleRequest, GetRoleResponse,
    GetUserRolesRequest, GetUserRolesResponse, GetUsersWithRoleRequest, GetUsersWithRoleResponse,
    ListPermissionsRequest, ListPermissionsResponse, ListRolesRequest, ListRolesResponse,
    PaginationResponse, RemovePermissionsFromRoleRequest, RemovePermissionsFromRoleResponse,
    RemoveRoleFromUserRequest, RemoveRoleFromUserResponse, Response as ApiResponse, Role,
    RolePermission, UpdateRoleRequest, UpdateRoleResponse, User, UserRoleAssignment,
};
use crate::{PermissionCheck, RequestExt};
use smartticket_shared_database::AuthService;

/// gRPC Role and Permission Service implementation
pub struct RolePermissionGrpcService {
    #[allow(dead_code)]
    auth_service: Arc<AuthService>,
    db_pool: Arc<sqlx::PgPool>,
}

impl RolePermissionGrpcService {
    /// Create a new gRPC role and permission service
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

    /// Convert database role to gRPC role
    fn db_role_to_grpc(role: &str, description: &str, permissions: Vec<RolePermission>) -> Role {
        Role {
            id: role.to_string(),
            name: role.to_string(),
            description: description.to_string(),
            permissions,
            is_system_role: Self::is_system_role(role),
            is_active: true,
            created_at: Some(prost_types::Timestamp {
                seconds: chrono::Utc::now().timestamp(),
                nanos: 0,
            }),
            updated_at: Some(prost_types::Timestamp {
                seconds: chrono::Utc::now().timestamp(),
                nanos: 0,
            }),
            created_by: "system".to_string(),
            updated_by: "system".to_string(),
        }
    }

    /// Check if a role is a system role
    fn is_system_role(role_name: &str) -> bool {
        matches!(
            role_name,
            "CustomerUser" | "CustomerAdmin" | "SupportAgent" | "SupportManager" | "SystemAdmin"
        )
    }

    /// Get all available permissions
    fn get_all_permissions() -> Vec<RolePermission> {
        vec![
            RolePermission {
                id: "tickets:read".to_string(),
                name: "Read Tickets".to_string(),
                description: "View and read tickets".to_string(),
                resource: "tickets".to_string(),
                action: "read".to_string(),
                scopes: vec!["own".to_string(), "team".to_string(), "all".to_string()],
                is_system_permission: true,
                category: "tickets".to_string(),
            },
            RolePermission {
                id: "tickets:write".to_string(),
                name: "Write Tickets".to_string(),
                description: "Create and update tickets".to_string(),
                resource: "tickets".to_string(),
                action: "write".to_string(),
                scopes: vec!["own".to_string(), "team".to_string(), "all".to_string()],
                is_system_permission: true,
                category: "tickets".to_string(),
            },
            RolePermission {
                id: "tickets:delete".to_string(),
                name: "Delete Tickets".to_string(),
                description: "Delete tickets".to_string(),
                resource: "tickets".to_string(),
                action: "delete".to_string(),
                scopes: vec!["own".to_string(), "team".to_string(), "all".to_string()],
                is_system_permission: true,
                category: "tickets".to_string(),
            },
            RolePermission {
                id: "tickets:assign".to_string(),
                name: "Assign Tickets".to_string(),
                description: "Assign tickets to agents".to_string(),
                resource: "tickets".to_string(),
                action: "assign".to_string(),
                scopes: vec!["team".to_string(), "all".to_string()],
                is_system_permission: true,
                category: "tickets".to_string(),
            },
            RolePermission {
                id: "knowledge:read".to_string(),
                name: "Read Knowledge".to_string(),
                description: "View knowledge base articles".to_string(),
                resource: "knowledge".to_string(),
                action: "read".to_string(),
                scopes: vec![
                    "published".to_string(),
                    "draft".to_string(),
                    "all".to_string(),
                ],
                is_system_permission: true,
                category: "knowledge".to_string(),
            },
            RolePermission {
                id: "knowledge:write".to_string(),
                name: "Write Knowledge".to_string(),
                description: "Create and update knowledge articles".to_string(),
                resource: "knowledge".to_string(),
                action: "write".to_string(),
                scopes: vec!["own".to_string(), "team".to_string(), "all".to_string()],
                is_system_permission: true,
                category: "knowledge".to_string(),
            },
            RolePermission {
                id: "knowledge:publish".to_string(),
                name: "Publish Knowledge".to_string(),
                description: "Publish knowledge articles".to_string(),
                resource: "knowledge".to_string(),
                action: "publish".to_string(),
                scopes: vec!["team".to_string(), "all".to_string()],
                is_system_permission: true,
                category: "knowledge".to_string(),
            },
            RolePermission {
                id: "knowledge:delete".to_string(),
                name: "Delete Knowledge".to_string(),
                description: "Delete knowledge articles".to_string(),
                resource: "knowledge".to_string(),
                action: "delete".to_string(),
                scopes: vec!["own".to_string(), "team".to_string(), "all".to_string()],
                is_system_permission: true,
                category: "knowledge".to_string(),
            },
            RolePermission {
                id: "users:read".to_string(),
                name: "Read Users".to_string(),
                description: "View user information".to_string(),
                resource: "users".to_string(),
                action: "read".to_string(),
                scopes: vec!["own".to_string(), "team".to_string(), "all".to_string()],
                is_system_permission: true,
                category: "users".to_string(),
            },
            RolePermission {
                id: "users:write".to_string(),
                name: "Write Users".to_string(),
                description: "Create and update users".to_string(),
                resource: "users".to_string(),
                action: "write".to_string(),
                scopes: vec!["team".to_string(), "all".to_string()],
                is_system_permission: true,
                category: "users".to_string(),
            },
            RolePermission {
                id: "users:delete".to_string(),
                name: "Delete Users".to_string(),
                description: "Delete users".to_string(),
                resource: "users".to_string(),
                action: "delete".to_string(),
                scopes: vec!["team".to_string(), "all".to_string()],
                is_system_permission: true,
                category: "users".to_string(),
            },
            RolePermission {
                id: "tenant:read".to_string(),
                name: "Read Tenant".to_string(),
                description: "View tenant information".to_string(),
                resource: "tenant".to_string(),
                action: "read".to_string(),
                scopes: vec!["own".to_string()],
                is_system_permission: true,
                category: "tenant".to_string(),
            },
            RolePermission {
                id: "tenant:write".to_string(),
                name: "Write Tenant".to_string(),
                description: "Update tenant settings".to_string(),
                resource: "tenant".to_string(),
                action: "write".to_string(),
                scopes: vec!["own".to_string()],
                is_system_permission: true,
                category: "tenant".to_string(),
            },
            RolePermission {
                id: "tenant:delete".to_string(),
                name: "Delete Tenant".to_string(),
                description: "Delete tenant".to_string(),
                resource: "tenant".to_string(),
                action: "delete".to_string(),
                scopes: vec!["all".to_string()],
                is_system_permission: true,
                category: "tenant".to_string(),
            },
            RolePermission {
                id: "system:admin".to_string(),
                name: "System Admin".to_string(),
                description: "Full system administration".to_string(),
                resource: "system".to_string(),
                action: "admin".to_string(),
                scopes: vec!["all".to_string()],
                is_system_permission: true,
                category: "system".to_string(),
            },
            RolePermission {
                id: "system:audit".to_string(),
                name: "System Audit".to_string(),
                description: "View system audit logs".to_string(),
                resource: "system".to_string(),
                action: "audit".to_string(),
                scopes: vec!["all".to_string()],
                is_system_permission: true,
                category: "system".to_string(),
            },
            RolePermission {
                id: "system:reports".to_string(),
                name: "System Reports".to_string(),
                description: "View system reports".to_string(),
                resource: "system".to_string(),
                action: "reports".to_string(),
                scopes: vec!["all".to_string()],
                is_system_permission: true,
                category: "system".to_string(),
            },
        ]
    }

    /// Get permissions for a role
    fn get_role_permissions(role_name: &str) -> Vec<RolePermission> {
        let all_permissions = Self::get_all_permissions();
        let role_permission_ids = match role_name {
            "CustomerUser" => vec!["tickets:read", "knowledge:read"],
            "CustomerAdmin" => vec![
                "tickets:read",
                "tickets:write",
                "knowledge:read",
                "knowledge:write",
                "users:read",
            ],
            "SupportAgent" => vec![
                "tickets:read",
                "tickets:write",
                "tickets:assign",
                "knowledge:read",
                "knowledge:write",
            ],
            "SupportManager" => vec![
                "tickets:read",
                "tickets:write",
                "tickets:delete",
                "tickets:assign",
                "knowledge:read",
                "knowledge:write",
                "knowledge:publish",
                "users:read",
                "tenant:read",
            ],
            "SystemAdmin" => vec![
                "tickets:read",
                "tickets:write",
                "tickets:delete",
                "tickets:assign",
                "knowledge:read",
                "knowledge:write",
                "knowledge:publish",
                "knowledge:delete",
                "users:read",
                "users:write",
                "users:delete",
                "tenant:read",
                "tenant:write",
                "system:admin",
                "system:audit",
                "system:reports",
            ],
            _ => vec![],
        };

        all_permissions
            .into_iter()
            .filter(|p| role_permission_ids.contains(&p.id.as_str()))
            .collect()
    }

    /// Convert database user to gRPC user
    #[allow(dead_code)]
    fn db_user_to_grpc(user: User) -> User {
        User {
            id: user.id.to_string(),
            tenant_id: user.tenant_id.to_string(),
            email: user.email,
            username: user.username,
            full_name: user.full_name,
            role: user.role as i32,
            is_active: user.is_active,
            last_login_at: None, // TODO: Implement last login tracking
        }
    }
}

#[tonic::async_trait]
impl RolePermissionService for RolePermissionGrpcService {
    #[instrument(skip(self))]
    async fn create_role(
        &self,
        request: Request<CreateRoleRequest>,
    ) -> StdResult<Response<CreateRoleResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("system:admin") {
            return Err(e);
        }

        let auth_user = request.auth_user()?;
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!("Creating role: {} by: {}", req.name, auth_user.email);

        // Validate required fields
        if req.name.trim().is_empty() {
            let response = CreateRoleResponse {
                response: Some(Self::create_error_response(
                    "Role name is required",
                    &request_id,
                )),
                role: None,
            };
            return Ok(Response::new(response));
        }

        // Check if role already exists
        if Self::is_system_role(&req.name) {
            let response = CreateRoleResponse {
                response: Some(Self::create_error_response(
                    "Cannot create role with system-reserved name",
                    &request_id,
                )),
                role: None,
            };
            return Ok(Response::new(response));
        }

        // Insert role into database
        let role_id = Uuid::new_v4();
        let now = chrono::Utc::now();

        let query = r#"
            INSERT INTO roles (id, tenant_id, name, description, is_system_role, is_active, created_by, updated_by, created_at, updated_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
            RETURNING id, tenant_id, name, description, is_system_role, is_active, created_by, updated_by, created_at, updated_at
        "#;

        let _result_role = sqlx::query_as::<_, DbRole>(query)
            .bind(role_id)
            .bind(auth_user.tenant_id)
            .bind(&req.name)
            .bind(&req.description)
            .bind(false) // is_system_role
            .bind(true) // is_active
            .bind(auth_user.id)
            .bind(auth_user.id)
            .bind(now)
            .bind(now)
            .fetch_one(&*self.db_pool)
            .await
            .map_err(|e| {
                error!("Failed to create role in database: {}", e);
                Status::internal("Failed to create role")
            })?;

        // Insert role permissions
        if !req.permission_ids.is_empty() {
            for permission_id in &req.permission_ids {
                let perm_query = r#"
                    INSERT INTO role_permissions (id, role_id, permission_id, tenant_id, created_at, updated_at)
                    VALUES ($1, $2, $3, $4, $5, $6)
                "#;

                let _perm_result = sqlx::query(perm_query)
                    .bind(Uuid::new_v4())
                    .bind(role_id)
                    .bind(permission_id)
                    .bind(auth_user.tenant_id)
                    .bind(now)
                    .bind(now)
                    .execute(&*self.db_pool)
                    .await
                    .map_err(|e| {
                        error!("Failed to create role permission in database: {}", e);
                        Status::internal("Failed to create role permissions")
                    })?;
            }
        }

        // Get permissions for response
        let permissions = req
            .permission_ids
            .into_iter()
            .filter_map(|id| Self::get_all_permissions().into_iter().find(|p| p.id == id))
            .collect();

        let role = Role {
            id: role_id.to_string(),
            name: req.name.clone(),
            description: req.description.clone(),
            permissions,
            is_system_role: false,
            is_active: true,
            created_at: Some(prost_types::Timestamp {
                seconds: now.timestamp(),
                nanos: now.timestamp_subsec_nanos() as i32,
            }),
            updated_at: Some(prost_types::Timestamp {
                seconds: now.timestamp(),
                nanos: now.timestamp_subsec_nanos() as i32,
            }),
            created_by: auth_user.id.to_string(),
            updated_by: auth_user.id.to_string(),
        };

        let response = CreateRoleResponse {
            response: Some(Self::create_success_response(
                "Role created successfully",
                &request_id,
            )),
            role: Some(role),
        };

        Ok(Response::new(response))
    }

    #[instrument(skip(self))]
    async fn get_role(
        &self,
        request: Request<GetRoleRequest>,
    ) -> StdResult<Response<GetRoleResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("system:admin") {
            return Err(e);
        }

        let auth_user = request.auth_user()?;
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!("Getting role: {} by: {}", req.role_id, auth_user.email);

        // Parse role ID as UUID
        let role_uuid = match Uuid::from_str(&req.role_id) {
            Ok(id) => id,
            Err(_) => {
                // Try to find by name for system roles
                if Self::is_system_role(&req.role_id) {
                    let permissions = Self::get_role_permissions(&req.role_id);
                    let role = Self::db_role_to_grpc(&req.role_id, "System role", permissions);
                    let all_permissions = Self::get_all_permissions();

                    let response = GetRoleResponse {
                        response: Some(Self::create_success_response(
                            "Role retrieved successfully",
                            &request_id,
                        )),
                        role: Some(role),
                        all_permissions,
                    };
                    return Ok(Response::new(response));
                } else {
                    let response = GetRoleResponse {
                        response: Some(Self::create_error_response(
                            "Invalid role ID format",
                            &request_id,
                        )),
                        role: None,
                        all_permissions: vec![],
                    };
                    return Ok(Response::new(response));
                }
            }
        };

        // Retrieve role from database
        let query = r#"
            SELECT id, tenant_id, name, description, is_system_role, is_active,
                   created_by, updated_by, created_at, updated_at
            FROM roles
            WHERE id = $1 AND tenant_id = $2
        "#;

        let role_record = sqlx::query_as::<_, DbRole>(query)
            .bind(role_uuid)
            .bind(auth_user.tenant_id)
            .fetch_optional(&*self.db_pool)
            .await
            .map_err(|e| {
                error!("Failed to retrieve role from database: {}", e);
                Status::internal("Failed to retrieve role")
            })?;

        if let Some(role_data) = role_record {
            // Get role permissions
            let perm_query = r#"
                SELECT permission_id
                FROM role_permissions
                WHERE role_id = $1 AND tenant_id = $2
            "#;

            let permission_records = sqlx::query_scalar(perm_query)
                .bind(role_uuid)
                .bind(auth_user.tenant_id)
                .fetch_all(&*self.db_pool)
                .await
                .map_err(|e| {
                    error!("Failed to retrieve role permissions from database: {}", e);
                    Status::internal("Failed to retrieve role permissions")
                })?;

            let permissions: Vec<RolePermission> = permission_records
                .into_iter()
                .filter_map(|perm_id: String| {
                    Self::get_all_permissions()
                        .into_iter()
                        .find(|p| p.id == perm_id)
                })
                .collect();

            let role = Role {
                id: role_data.id.to_string(),
                name: role_data.name,
                description: role_data.description.unwrap_or_default(),
                permissions,
                is_system_role: role_data.is_system_role,
                is_active: role_data.is_active,
                created_at: role_data.created_at.map(|dt| prost_types::Timestamp {
                    seconds: dt.timestamp(),
                    nanos: dt.timestamp_subsec_nanos() as i32,
                }),
                updated_at: role_data.updated_at.map(|dt| prost_types::Timestamp {
                    seconds: dt.timestamp(),
                    nanos: dt.timestamp_subsec_nanos() as i32,
                }),
                created_by: role_data.created_by.to_string(),
                updated_by: role_data.updated_by.to_string(),
            };

            let all_permissions = Self::get_all_permissions();

            let response = GetRoleResponse {
                response: Some(Self::create_success_response(
                    "Role retrieved successfully",
                    &request_id,
                )),
                role: Some(role),
                all_permissions,
            };
            Ok(Response::new(response))
        } else {
            let response = GetRoleResponse {
                response: Some(Self::create_error_response("Role not found", &request_id)),
                role: None,
                all_permissions: vec![],
            };
            Ok(Response::new(response))
        }
    }

    #[instrument(skip(self))]
    async fn update_role(
        &self,
        request: Request<UpdateRoleRequest>,
    ) -> StdResult<Response<UpdateRoleResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("system:admin") {
            return Err(e);
        }

        let auth_user = request.auth_user()?;
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!("Updating role: {} by: {}", req.role_id, auth_user.email);

        // Check if trying to update system role
        if Self::is_system_role(&req.role_id) {
            let response = UpdateRoleResponse {
                response: Some(Self::create_error_response(
                    "Cannot modify system roles",
                    &request_id,
                )),
                role: None,
            };
            return Ok(Response::new(response));
        }

        // Parse role ID as UUID
        let role_uuid = match Uuid::from_str(&req.role_id) {
            Ok(id) => id,
            Err(_) => {
                let response = UpdateRoleResponse {
                    response: Some(Self::create_error_response(
                        "Invalid role ID format",
                        &request_id,
                    )),
                    role: None,
                };
                return Ok(Response::new(response));
            }
        };

        let now = chrono::Utc::now();

        // Check if role exists and get current data
        let select_query = r#"
            SELECT id, tenant_id, name, description, is_system_role, is_active,
                   created_by, updated_by, created_at, updated_at
            FROM roles
            WHERE id = $1 AND tenant_id = $2
        "#;

        let current_role = sqlx::query_as::<_, DbRole>(select_query)
            .bind(role_uuid)
            .bind(auth_user.tenant_id)
            .fetch_optional(&*self.db_pool)
            .await
            .map_err(|e| {
                error!("Failed to retrieve role from database: {}", e);
                Status::internal("Failed to retrieve role")
            })?;

        if let Some(_role_data) = current_role {
            // Update role in database
            let update_query = r#"
                UPDATE roles
                SET name = COALESCE($1, name),
                    description = COALESCE($2, description),
                    is_active = $3,
                    updated_by = $4,
                    updated_at = $5
                WHERE id = $6 AND tenant_id = $7
                RETURNING id, tenant_id, name, description, is_system_role, is_active,
                          created_by, updated_by, created_at, updated_at
            "#;

            let updated_role = sqlx::query_as::<_, DbRole>(update_query)
                .bind(if req.name.is_empty() {
                    None
                } else {
                    Some(&req.name)
                })
                .bind(if req.description.is_empty() {
                    None
                } else {
                    Some(&req.description)
                })
                .bind(req.is_active)
                .bind(auth_user.id)
                .bind(now)
                .bind(role_uuid)
                .bind(auth_user.tenant_id)
                .fetch_one(&*self.db_pool)
                .await
                .map_err(|e| {
                    error!("Failed to update role in database: {}", e);
                    Status::internal("Failed to update role")
                })?;

            // Get role permissions
            let perm_query = r#"
                SELECT permission_id
                FROM role_permissions
                WHERE role_id = $1 AND tenant_id = $2
            "#;

            let permission_records = sqlx::query_scalar(perm_query)
                .bind(role_uuid)
                .bind(auth_user.tenant_id)
                .fetch_all(&*self.db_pool)
                .await
                .map_err(|e| {
                    error!("Failed to retrieve role permissions from database: {}", e);
                    Status::internal("Failed to retrieve role permissions")
                })?;

            let permissions: Vec<RolePermission> = permission_records
                .into_iter()
                .filter_map(|perm_id: String| {
                    Self::get_all_permissions()
                        .into_iter()
                        .find(|p| p.id == perm_id)
                })
                .collect();

            let role = Role {
                id: updated_role.id.to_string(),
                name: updated_role.name,
                description: updated_role.description.unwrap_or_default(),
                permissions,
                is_system_role: updated_role.is_system_role,
                is_active: updated_role.is_active,
                created_at: updated_role.created_at.map(|dt| prost_types::Timestamp {
                    seconds: dt.timestamp(),
                    nanos: dt.timestamp_subsec_nanos() as i32,
                }),
                updated_at: updated_role.updated_at.map(|dt| prost_types::Timestamp {
                    seconds: dt.timestamp(),
                    nanos: dt.timestamp_subsec_nanos() as i32,
                }),
                created_by: updated_role.created_by.to_string(),
                updated_by: updated_role.updated_by.to_string(),
            };

            let response = UpdateRoleResponse {
                response: Some(Self::create_success_response(
                    "Role updated successfully",
                    &request_id,
                )),
                role: Some(role),
            };
            Ok(Response::new(response))
        } else {
            let response = UpdateRoleResponse {
                response: Some(Self::create_error_response("Role not found", &request_id)),
                role: None,
            };
            Ok(Response::new(response))
        }
    }

    #[instrument(skip(self))]
    async fn delete_role(
        &self,
        request: Request<DeleteRoleRequest>,
    ) -> StdResult<Response<DeleteRoleResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("system:admin") {
            return Err(e);
        }

        let auth_user = request.auth_user()?;
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!("Deleting role: {} by: {}", req.role_id, auth_user.email);

        // Check if trying to delete system role
        if Self::is_system_role(&req.role_id) && !req.force_delete {
            let response = DeleteRoleResponse {
                response: Some(Self::create_error_response(
                    "Cannot delete system roles",
                    &request_id,
                )),
            };
            return Ok(Response::new(response));
        }

        // Parse role ID as UUID
        let role_uuid = match Uuid::from_str(&req.role_id) {
            Ok(id) => id,
            Err(_) => {
                let response = DeleteRoleResponse {
                    response: Some(Self::create_error_response(
                        "Invalid role ID format",
                        &request_id,
                    )),
                };
                return Ok(Response::new(response));
            }
        };

        // Check if role exists
        let check_query = r#"
            SELECT id, tenant_id, is_system_role
            FROM roles
            WHERE id = $1 AND tenant_id = $2
        "#;

        let role_check = sqlx::query_as::<_, (Uuid, Uuid, bool)>(check_query)
            .bind(role_uuid)
            .bind(auth_user.tenant_id)
            .fetch_optional(&*self.db_pool)
            .await
            .map_err(|e| {
                error!("Failed to check role existence in database: {}", e);
                Status::internal("Failed to check role")
            })?;

        if let Some((_, _, is_system_role)) = role_check {
            // Check if trying to delete system role
            if is_system_role && !req.force_delete {
                let response = DeleteRoleResponse {
                    response: Some(Self::create_error_response(
                        "Cannot delete system roles without force",
                        &request_id,
                    )),
                };
                return Ok(Response::new(response));
            }

            // Delete role permissions first
            let delete_perms_query = r#"
                DELETE FROM role_permissions
                WHERE role_id = $1 AND tenant_id = $2
            "#;

            let _perms_result = sqlx::query(delete_perms_query)
                .bind(role_uuid)
                .bind(auth_user.tenant_id)
                .execute(&*self.db_pool)
                .await
                .map_err(|e| {
                    error!("Failed to delete role permissions from database: {}", e);
                    Status::internal("Failed to delete role permissions")
                })?;

            // Delete user role assignments
            let delete_assignments_query = r#"
                DELETE FROM user_role_assignments
                WHERE role_id = $1 AND tenant_id = $2
            "#;

            let _assignments_result = sqlx::query(delete_assignments_query)
                .bind(role_uuid)
                .bind(auth_user.tenant_id)
                .execute(&*self.db_pool)
                .await
                .map_err(|e| {
                    error!(
                        "Failed to delete user role assignments from database: {}",
                        e
                    );
                    Status::internal("Failed to delete user role assignments")
                })?;

            // Delete the role
            let delete_role_query = r#"
                DELETE FROM roles
                WHERE id = $1 AND tenant_id = $2
            "#;

            let _role_result = sqlx::query(delete_role_query)
                .bind(role_uuid)
                .bind(auth_user.tenant_id)
                .execute(&*self.db_pool)
                .await
                .map_err(|e| {
                    error!("Failed to delete role from database: {}", e);
                    Status::internal("Failed to delete role")
                })?;

            let response = DeleteRoleResponse {
                response: Some(Self::create_success_response(
                    "Role deleted successfully",
                    &request_id,
                )),
            };
            Ok(Response::new(response))
        } else {
            let response = DeleteRoleResponse {
                response: Some(Self::create_error_response("Role not found", &request_id)),
            };
            Ok(Response::new(response))
        }
    }

    #[instrument(skip(self))]
    async fn list_roles(
        &self,
        request: Request<ListRolesRequest>,
    ) -> StdResult<Response<ListRolesResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("system:admin") {
            return Err(e);
        }

        let auth_user = request.auth_user()?;
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!("Listing roles by: {}", auth_user.email);

        // Get custom roles from database
        let db_query = r#"
            SELECT id, tenant_id, name, description, is_system_role, is_active,
                   created_by, updated_by, created_at, updated_at
            FROM roles
            WHERE tenant_id = $1
            ORDER BY created_at DESC
        "#;

        let db_roles = sqlx::query_as::<_, DbRole>(db_query)
            .bind(auth_user.tenant_id)
            .fetch_all(&*self.db_pool)
            .await
            .map_err(|e| {
                error!("Failed to retrieve roles from database: {}", e);
                Status::internal("Failed to retrieve roles")
            })?;

        let mut roles = vec![];

        // Add database roles
        for role_data in db_roles {
            // Get role permissions
            let perm_query = r#"
                SELECT permission_id
                FROM role_permissions
                WHERE role_id = $1 AND tenant_id = $2
            "#;

            let permission_records = sqlx::query_scalar(perm_query)
                .bind(role_data.id)
                .bind(auth_user.tenant_id)
                .fetch_all(&*self.db_pool)
                .await
                .map_err(|e| {
                    error!("Failed to retrieve role permissions from database: {}", e);
                    Status::internal("Failed to retrieve role permissions")
                })?;

            let permissions: Vec<RolePermission> = permission_records
                .into_iter()
                .filter_map(|perm_id: String| {
                    Self::get_all_permissions()
                        .into_iter()
                        .find(|p| p.id == perm_id)
                })
                .collect();

            let role = Role {
                id: role_data.id.to_string(),
                name: role_data.name,
                description: role_data.description.unwrap_or_default(),
                permissions,
                is_system_role: role_data.is_system_role,
                is_active: role_data.is_active,
                created_at: role_data.created_at.map(|dt| prost_types::Timestamp {
                    seconds: dt.timestamp(),
                    nanos: dt.timestamp_subsec_nanos() as i32,
                }),
                updated_at: role_data.updated_at.map(|dt| prost_types::Timestamp {
                    seconds: dt.timestamp(),
                    nanos: dt.timestamp_subsec_nanos() as i32,
                }),
                created_by: role_data.created_by.to_string(),
                updated_by: role_data.updated_by.to_string(),
            };
            roles.push(role);
        }

        // Add system roles if requested
        if req.include_system_roles {
            let system_roles = vec![
                ("CustomerUser", "Basic customer access"),
                ("CustomerAdmin", "Customer administrator"),
                ("SupportAgent", "Support agent"),
                ("SupportManager", "Support manager"),
                ("SystemAdmin", "System administrator"),
            ];

            for (role_name, description) in system_roles {
                let permissions = Self::get_role_permissions(role_name);
                let role = Self::db_role_to_grpc(role_name, description, permissions);
                roles.push(role);
            }
        }

        // Apply search filter if provided
        if !req.search.is_empty() {
            roles.retain(|r| {
                r.name.to_lowercase().contains(&req.search.to_lowercase())
                    || r.description
                        .to_lowercase()
                        .contains(&req.search.to_lowercase())
            });
        }

        // Apply active filter if specified
        if !req.include_inactive {
            roles.retain(|r| r.is_active);
        }

        let total_count = roles.len() as i32;
        let page_size = req.pagination.as_ref().map(|p| p.page_size).unwrap_or(10);

        // Apply pagination
        let offset = req
            .pagination
            .as_ref()
            .and_then(|p| p.page_token.parse::<usize>().ok())
            .unwrap_or(0);
        let end_index = std::cmp::min(offset + page_size as usize, roles.len());
        let paginated_roles = if offset < roles.len() {
            roles[offset..end_index].to_vec()
        } else {
            vec![]
        };

        let next_page_token = if end_index < roles.len() {
            end_index.to_string()
        } else {
            "".to_string()
        };

        let response = ListRolesResponse {
            response: Some(Self::create_success_response(
                "Roles listed successfully",
                &request_id,
            )),
            roles: paginated_roles,
            pagination: Some(PaginationResponse {
                total_count,
                page_size,
                next_page_token,
                prev_page_token: if offset > 0 {
                    (offset.saturating_sub(page_size as usize)).to_string()
                } else {
                    "".to_string()
                },
            }),
        };

        Ok(Response::new(response))
    }

    #[instrument(skip(self))]
    async fn list_permissions(
        &self,
        request: Request<ListPermissionsRequest>,
    ) -> StdResult<Response<ListPermissionsResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("system:admin") {
            return Err(e);
        }

        let auth_user = request.auth_user()?;
        let req = request.into_inner();

        let request_id = req
            .metadata
            .as_ref()
            .map(|m| m.request_id.clone())
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        info!("Listing permissions by: {}", auth_user.email);

        let mut permissions = Self::get_all_permissions();

        // Apply category filter if provided
        if !req.category.is_empty() {
            permissions.retain(|p| p.category == req.category);
        }

        // Apply resource filter if provided
        if !req.resource.is_empty() {
            permissions.retain(|p| p.resource == req.resource);
        }

        // Apply system permission filter if specified
        if !req.include_system_permissions {
            permissions.retain(|p| !p.is_system_permission);
        }

        let total_count = permissions.len() as i32;
        let page_size = req.pagination.as_ref().map(|p| p.page_size).unwrap_or(10);

        let response = ListPermissionsResponse {
            response: Some(Self::create_success_response(
                "Permissions listed successfully",
                &request_id,
            )),
            permissions,
            pagination: Some(PaginationResponse {
                total_count,
                page_size,
                next_page_token: "".to_string(),
                prev_page_token: "".to_string(),
            }),
        };

        Ok(Response::new(response))
    }

    #[instrument(skip(self))]
    async fn get_role_permissions(
        &self,
        request: Request<GetRolePermissionsRequest>,
    ) -> StdResult<Response<GetRolePermissionsResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("system:admin") {
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
            "Getting permissions for role: {} by: {}",
            req.role_id, auth_user.email
        );

        let permissions = Self::get_role_permissions(&req.role_id);
        let role = Self::db_role_to_grpc(&req.role_id, "System role", vec![]);
        let available_permissions = Self::get_all_permissions();

        let response = GetRolePermissionsResponse {
            response: Some(Self::create_success_response(
                "Role permissions retrieved successfully",
                &request_id,
            )),
            role: Some(role),
            permissions,
            available_permissions,
        };

        Ok(Response::new(response))
    }

    #[instrument(skip(self))]
    async fn assign_permissions_to_role(
        &self,
        request: Request<AssignPermissionsToRoleRequest>,
    ) -> StdResult<Response<AssignPermissionsToRoleResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("system:admin") {
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
            "Assigning permissions to role: {} by: {}",
            req.role_id, auth_user.email
        );

        // Check if trying to modify system role
        if Self::is_system_role(&req.role_id) {
            let response = AssignPermissionsToRoleResponse {
                response: Some(Self::create_error_response(
                    "Cannot modify system role permissions",
                    &request_id,
                )),
                role: None,
                assigned_permissions: vec![],
            };
            return Ok(Response::new(response));
        }

        // Parse role ID as UUID
        let role_uuid = match Uuid::from_str(&req.role_id) {
            Ok(id) => id,
            Err(_) => {
                let response = AssignPermissionsToRoleResponse {
                    response: Some(Self::create_error_response(
                        "Invalid role ID format",
                        &request_id,
                    )),
                    role: None,
                    assigned_permissions: vec![],
                };
                return Ok(Response::new(response));
            }
        };

        let now = chrono::Utc::now();

        // Verify role exists
        let role_check = r#"
            SELECT id, name FROM roles WHERE id = $1 AND tenant_id = $2
        "#;

        let role_data = sqlx::query_as::<_, (Uuid, String)>(role_check)
            .bind(role_uuid)
            .bind(auth_user.tenant_id)
            .fetch_optional(&*self.db_pool)
            .await
            .map_err(|e| {
                error!("Failed to check role in database: {}", e);
                Status::internal("Failed to check role")
            })?;

        if role_data.is_none() {
            let response = AssignPermissionsToRoleResponse {
                response: Some(Self::create_error_response("Role not found", &request_id)),
                role: None,
                assigned_permissions: vec![],
            };
            return Ok(Response::new(response));
        }

        // Remove existing permissions
        let delete_perms = r#"
            DELETE FROM role_permissions WHERE role_id = $1 AND tenant_id = $2
        "#;

        sqlx::query(delete_perms)
            .bind(role_uuid)
            .bind(auth_user.tenant_id)
            .execute(&*self.db_pool)
            .await
            .map_err(|e| {
                error!("Failed to delete existing permissions: {}", e);
                Status::internal("Failed to delete existing permissions")
            })?;

        // Add new permissions
        for permission_id in &req.permission_ids {
            let insert_perm = r#"
                INSERT INTO role_permissions (id, role_id, permission_id, tenant_id, created_at, updated_at)
                VALUES ($1, $2, $3, $4, $5, $6)
                ON CONFLICT DO NOTHING
            "#;

            sqlx::query(insert_perm)
                .bind(Uuid::new_v4())
                .bind(role_uuid)
                .bind(permission_id)
                .bind(auth_user.tenant_id)
                .bind(now)
                .bind(now)
                .execute(&*self.db_pool)
                .await
                .map_err(|e| {
                    error!("Failed to assign permission: {}", e);
                    Status::internal("Failed to assign permission")
                })?;
        }

        // Get assigned permissions for response
        let permissions: Vec<RolePermission> = req
            .permission_ids
            .into_iter()
            .filter_map(|id| Self::get_all_permissions().into_iter().find(|p| p.id == id))
            .collect();

        let role = Role {
            id: req.role_id.clone(),
            name: role_data.unwrap().1,
            description: "Updated role".to_string(),
            permissions: permissions.clone(),
            is_system_role: false,
            is_active: true,
            created_at: Some(prost_types::Timestamp {
                seconds: now.timestamp(),
                nanos: now.timestamp_subsec_nanos() as i32,
            }),
            updated_at: Some(prost_types::Timestamp {
                seconds: now.timestamp(),
                nanos: now.timestamp_subsec_nanos() as i32,
            }),
            created_by: auth_user.id.to_string(),
            updated_by: auth_user.id.to_string(),
        };

        let response = AssignPermissionsToRoleResponse {
            response: Some(Self::create_success_response(
                "Permissions assigned successfully",
                &request_id,
            )),
            role: Some(role),
            assigned_permissions: permissions,
        };

        Ok(Response::new(response))
    }

    #[instrument(skip(self))]
    async fn remove_permissions_from_role(
        &self,
        request: Request<RemovePermissionsFromRoleRequest>,
    ) -> StdResult<Response<RemovePermissionsFromRoleResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("system:admin") {
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
            "Removing permissions from role: {} by: {}",
            req.role_id, auth_user.email
        );

        // Check if trying to modify system role
        if Self::is_system_role(&req.role_id) {
            let response = RemovePermissionsFromRoleResponse {
                response: Some(Self::create_error_response(
                    "Cannot modify system role permissions",
                    &request_id,
                )),
                role: None,
                remaining_permissions: vec![],
            };
            return Ok(Response::new(response));
        }

        // TODO: Implement actual permission removal in database
        let mut current_permissions = Self::get_role_permissions(&req.role_id);
        let permission_ids_to_remove: std::collections::HashSet<String> =
            req.permission_ids.into_iter().collect();
        current_permissions.retain(|p| !permission_ids_to_remove.contains(&p.id));

        let role = Self::db_role_to_grpc(&req.role_id, "Updated role", current_permissions.clone());

        let response = RemovePermissionsFromRoleResponse {
            response: Some(Self::create_success_response(
                "Permissions removed successfully",
                &request_id,
            )),
            role: Some(role),
            remaining_permissions: current_permissions,
        };

        Ok(Response::new(response))
    }

    #[instrument(skip(self))]
    async fn assign_role_to_user(
        &self,
        request: Request<AssignRoleToUserRequest>,
    ) -> StdResult<Response<AssignRoleToUserResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("users:write") {
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
            "Assigning role {} to user {} by: {}",
            req.role_id, req.user_id, auth_user.email
        );

        // Parse user ID
        let _user_id = match Uuid::from_str(&req.user_id) {
            Ok(id) => id,
            Err(_) => {
                let response = AssignRoleToUserResponse {
                    response: Some(Self::create_error_response(
                        "Invalid user ID format",
                        &request_id,
                    )),
                    assignment: None,
                    role: None,
                };
                return Ok(Response::new(response));
            }
        };

        // Verify user exists
        let user_check = r#"
            SELECT id, email FROM users WHERE id = $1 AND tenant_id = $2
        "#;

        let user_data = sqlx::query_as::<_, (Uuid, String)>(user_check)
            .bind(_user_id)
            .bind(auth_user.tenant_id)
            .fetch_optional(&*self.db_pool)
            .await
            .map_err(|e| {
                error!("Failed to check user in database: {}", e);
                Status::internal("Failed to check user")
            })?;

        if user_data.is_none() {
            let response = AssignRoleToUserResponse {
                response: Some(Self::create_error_response("User not found", &request_id)),
                assignment: None,
                role: None,
            };
            return Ok(Response::new(response));
        }

        // Parse role ID as UUID or handle system role names
        let role_uuid = match Uuid::from_str(&req.role_id) {
            Ok(id) => Some(id),
            Err(_) => {
                // For system roles, we don't store in database but return success
                if Self::is_system_role(&req.role_id) {
                    let assignment = UserRoleAssignment {
                        user_id: req.user_id,
                        role_id: req.role_id.clone(),
                        assigned_by: auth_user.id.to_string(),
                        assigned_at: Some(prost_types::Timestamp {
                            seconds: chrono::Utc::now().timestamp(),
                            nanos: 0,
                        }),
                        expires_at: req.expires_at,
                        is_active: true,
                        assignment_reason: req.reason,
                    };

                    let role = Self::db_role_to_grpc(&req.role_id, "System role", vec![]);

                    let response = AssignRoleToUserResponse {
                        response: Some(Self::create_success_response(
                            "System role assigned successfully",
                            &request_id,
                        )),
                        assignment: Some(assignment),
                        role: Some(role),
                    };
                    return Ok(Response::new(response));
                } else {
                    let response = AssignRoleToUserResponse {
                        response: Some(Self::create_error_response(
                            "Invalid role ID format",
                            &request_id,
                        )),
                        assignment: None,
                        role: None,
                    };
                    return Ok(Response::new(response));
                }
            }
        };

        let now = chrono::Utc::now();

        // Create role assignment in database
        let assignment_id = Uuid::new_v4();
        let insert_assignment = r#"
            INSERT INTO user_role_assignments (id, user_id, role_id, tenant_id, assigned_by, assigned_at, expires_at, is_active, assignment_reason)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
            ON CONFLICT (user_id, role_id, tenant_id) DO UPDATE SET
                assigned_by = EXCLUDED.assigned_by,
                assigned_at = EXCLUDED.assigned_at,
                expires_at = EXCLUDED.expires_at,
                is_active = EXCLUDED.is_active,
                assignment_reason = EXCLUDED.assignment_reason
        "#;

        sqlx::query(insert_assignment)
            .bind(assignment_id)
            .bind(_user_id)
            .bind(role_uuid.unwrap())
            .bind(auth_user.tenant_id)
            .bind(auth_user.id)
            .bind(now)
            .bind(req.expires_at.as_ref().map(|dt| {
                chrono::DateTime::<chrono::Utc>::from_timestamp(dt.seconds, dt.nanos as u32)
            }))
            .bind(true)
            .bind(&req.reason)
            .execute(&*self.db_pool)
            .await
            .map_err(|e| {
                error!("Failed to assign role to user: {}", e);
                Status::internal("Failed to assign role to user")
            })?;

        // Get role details for response
        let role_query = r#"
            SELECT id, name, description FROM roles WHERE id = $1 AND tenant_id = $2
        "#;

        let role_details = sqlx::query_as::<_, (Uuid, String, String)>(role_query)
            .bind(role_uuid.unwrap())
            .bind(auth_user.tenant_id)
            .fetch_optional(&*self.db_pool)
            .await
            .map_err(|e| {
                error!("Failed to get role details: {}", e);
                Status::internal("Failed to get role details")
            })?;

        if let Some((_, role_name, role_description)) = role_details {
            let assignment = UserRoleAssignment {
                user_id: req.user_id,
                role_id: req.role_id.clone(),
                assigned_by: auth_user.id.to_string(),
                assigned_at: Some(prost_types::Timestamp {
                    seconds: now.timestamp(),
                    nanos: now.timestamp_subsec_nanos() as i32,
                }),
                expires_at: req.expires_at,
                is_active: true,
                assignment_reason: req.reason,
            };

            let role = Role {
                id: req.role_id.clone(),
                name: role_name,
                description: role_description,
                permissions: vec![], // Permissions can be loaded separately if needed
                is_system_role: false,
                is_active: true,
                created_at: None,
                updated_at: None,
                created_by: auth_user.id.to_string(),
                updated_by: auth_user.id.to_string(),
            };

            let response = AssignRoleToUserResponse {
                response: Some(Self::create_success_response(
                    "Role assigned successfully",
                    &request_id,
                )),
                assignment: Some(assignment),
                role: Some(role),
            };
            Ok(Response::new(response))
        } else {
            let response = AssignRoleToUserResponse {
                response: Some(Self::create_error_response("Role not found", &request_id)),
                assignment: None,
                role: None,
            };
            Ok(Response::new(response))
        }
    }

    #[instrument(skip(self))]
    async fn remove_role_from_user(
        &self,
        request: Request<RemoveRoleFromUserRequest>,
    ) -> StdResult<Response<RemoveRoleFromUserResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("users:write") {
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
            "Removing role {} from user {} by: {}",
            req.role_id, req.user_id, auth_user.email
        );

        // TODO: Implement actual role removal in database
        let response = RemoveRoleFromUserResponse {
            response: Some(Self::create_success_response(
                "Role removed successfully",
                &request_id,
            )),
        };

        Ok(Response::new(response))
    }

    #[instrument(skip(self))]
    async fn get_user_roles(
        &self,
        request: Request<GetUserRolesRequest>,
    ) -> StdResult<Response<GetUserRolesResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("users:read") {
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
            "Getting roles for user: {} by: {}",
            req.user_id, auth_user.email
        );

        // TODO: Implement actual user role retrieval from database
        let role_name = "CustomerUser"; // Mock role
        let role = Self::db_role_to_grpc(role_name, "System role", vec![]);

        let assignment = UserRoleAssignment {
            user_id: req.user_id,
            role_id: role_name.to_string(),
            assigned_by: "system".to_string(),
            assigned_at: Some(prost_types::Timestamp {
                seconds: chrono::Utc::now().timestamp(),
                nanos: 0,
            }),
            expires_at: None,
            is_active: true,
            assignment_reason: "Initial assignment".to_string(),
        };

        let response = GetUserRolesResponse {
            response: Some(Self::create_success_response(
                "User roles retrieved successfully",
                &request_id,
            )),
            assignments: vec![assignment],
            roles: vec![role],
        };

        Ok(Response::new(response))
    }

    #[instrument(skip(self))]
    async fn get_users_with_role(
        &self,
        request: Request<GetUsersWithRoleRequest>,
    ) -> StdResult<Response<GetUsersWithRoleResponse>, Status> {
        // Check authentication and authorization
        if let Err(e) = request.check_permission("users:read") {
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
            "Getting users with role: {} by: {}",
            req.role_id, auth_user.email
        );

        // TODO: Implement actual user retrieval from database
        let users = vec![]; // Mock empty list for now
        let assignments = vec![]; // Mock empty list for now

        let response = GetUsersWithRoleResponse {
            response: Some(Self::create_success_response(
                "Users with role retrieved successfully",
                &request_id,
            )),
            users,
            pagination: Some(PaginationResponse {
                total_count: 0,
                page_size: req.pagination.as_ref().map(|p| p.page_size).unwrap_or(10),
                next_page_token: "".to_string(),
                prev_page_token: "".to_string(),
            }),
            assignments,
        };

        Ok(Response::new(response))
    }
}
