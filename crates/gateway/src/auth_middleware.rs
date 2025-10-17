use std::sync::Arc;
use tonic::metadata::MetadataMap;
use tonic::service::Interceptor;
use tonic::{Request, Status};

use crate::smartticket_v1::RequestMetadata;
use smartticket_shared_database::{AuthService, AuthUser, Claims};

/// JWT Authentication middleware for gRPC services
#[derive(Clone)]
#[allow(dead_code)]
pub struct AuthMiddleware<S> {
    inner: S,
    auth_service: Arc<AuthService>,
}

impl<S> AuthMiddleware<S> {
    pub fn new(service: S, auth_service: Arc<AuthService>) -> Self {
        Self {
            inner: service,
            auth_service,
        }
    }

    /// Extract JWT token from gRPC metadata
    #[allow(dead_code)]
    fn extract_token_from_metadata(metadata: &MetadataMap) -> Result<String, Status> {
        // Look for Authorization header
        if let Some(auth_header) = metadata.get("authorization") {
            let auth_str = auth_header
                .to_str()
                .map_err(|_| Status::unauthenticated("Invalid Authorization header format"))?;

            if auth_str.starts_with("Bearer ") {
                return Ok(auth_str[7..].to_string());
            }
        }

        // Alternative: look for x-auth-token header
        if let Some(token_header) = metadata.get("x-auth-token") {
            return Ok(token_header
                .to_str()
                .map_err(|_| Status::unauthenticated("Invalid x-auth-token header format"))?
                .to_string());
        }

        Err(Status::unauthenticated("Missing authentication token"))
    }

    /// Validate JWT token and extract user claims
    #[allow(dead_code)]
    fn validate_token(&self, token: &str) -> Result<Claims, Status> {
        self.auth_service
            .validate_token(token)
            .map_err(|e| Status::unauthenticated(format!("Invalid token: {}", e)))
    }

    /// Extract tenant context from request metadata
    #[allow(dead_code)]
    fn extract_tenant_context(&self, metadata: &MetadataMap) -> Result<TenantContext, Status> {
        let tenant_id = metadata
            .get("x-tenant-id")
            .ok_or_else(|| Status::invalid_argument("Missing x-tenant-id header"))?
            .to_str()
            .map_err(|_| Status::invalid_argument("Invalid tenant ID format"))?;

        let user_id = metadata
            .get("x-user-id")
            .ok_or_else(|| Status::invalid_argument("Missing x-user-id header"))?
            .to_str()
            .map_err(|_| Status::invalid_argument("Invalid user ID format"))?;

        let user_role = metadata
            .get("x-user-role")
            .and_then(|v| v.to_str().ok())
            .unwrap_or("CustomerUser")
            .to_string();

        let tenant_id = uuid::Uuid::parse_str(tenant_id)
            .map_err(|_| Status::invalid_argument("Invalid tenant ID UUID format"))?;

        let user_id = uuid::Uuid::parse_str(user_id)
            .map_err(|_| Status::invalid_argument("Invalid user ID UUID format"))?;

        Ok(TenantContext {
            tenant_id,
            user_id,
            user_role,
        })
    }
}

/// Tenant context extracted from authentication
#[derive(Debug, Clone)]
pub struct TenantContext {
    pub tenant_id: uuid::Uuid,
    pub user_id: uuid::Uuid,
    pub user_role: String,
}

impl TenantContext {
    /// Check if user has admin privileges
    pub fn is_admin(&self) -> bool {
        matches!(self.user_role.as_str(), "SuperAdmin" | "TenantAdmin")
    }

    /// Check if user is support staff
    pub fn is_support(&self) -> bool {
        matches!(
            self.user_role.as_str(),
            "SuperAdmin" | "TenantAdmin" | "SupportEngineer"
        )
    }

    /// Check if user is customer
    pub fn is_customer(&self) -> bool {
        self.user_role.as_str() == "CustomerUser"
    }

    /// Check if user can view all tickets (not just their own)
    pub fn can_view_all_tickets(&self) -> bool {
        matches!(
            self.user_role.as_str(),
            "SuperAdmin" | "TenantAdmin" | "SupportEngineer"
        )
    }

    /// Check if user can assign tickets
    pub fn can_assign_tickets(&self) -> bool {
        matches!(
            self.user_role.as_str(),
            "SuperAdmin" | "TenantAdmin" | "SupportEngineer"
        )
    }

    /// Check if user can delete tickets
    pub fn can_delete_tickets(&self) -> bool {
        matches!(self.user_role.as_str(), "SuperAdmin" | "TenantAdmin")
    }

    /// Check if user can manage users
    pub fn can_manage_users(&self) -> bool {
        matches!(self.user_role.as_str(), "SuperAdmin" | "TenantAdmin")
    }

    /// Check if user can manage knowledge base
    pub fn can_manage_knowledge(&self) -> bool {
        matches!(
            self.user_role.as_str(),
            "SuperAdmin" | "TenantAdmin" | "SupportEngineer"
        )
    }
}

/// Request extension trait for tenant context
pub trait RequestExt {
    fn tenant_context(&self) -> Result<TenantContext, Status>;
    fn auth_user(&self) -> Result<AuthUser, Status>;
}

impl<T> RequestExt for Request<T> {
    fn tenant_context(&self) -> Result<TenantContext, Status> {
        self.extensions()
            .get::<TenantContext>()
            .cloned()
            .ok_or_else(|| Status::internal("Tenant context not found"))
    }

    fn auth_user(&self) -> Result<AuthUser, Status> {
        self.extensions()
            .get::<AuthUser>()
            .cloned()
            .ok_or_else(|| Status::internal("Auth user not found"))
    }
}

/// Trait for permission checking in gRPC services
pub trait PermissionCheck {
    fn check_permission(&self, permission: &str) -> Result<(), Status>;
    fn check_any_permission(&self, permissions: &[&str]) -> Result<(), Status>;
    fn check_all_permissions(&self, permissions: &[&str]) -> Result<(), Status>;
}

impl<T> PermissionCheck for Request<T> {
    fn check_permission(&self, permission: &str) -> Result<(), Status> {
        let auth_user = self.auth_user()?;
        if auth_user.permissions.contains(&permission.to_string()) {
            Ok(())
        } else {
            Err(Status::permission_denied(format!(
                "Insufficient permissions. Required: {}",
                permission
            )))
        }
    }

    fn check_any_permission(&self, permissions: &[&str]) -> Result<(), Status> {
        let auth_user = self.auth_user()?;
        if permissions
            .iter()
            .any(|perm| auth_user.permissions.contains(&perm.to_string()))
        {
            Ok(())
        } else {
            Err(Status::permission_denied(format!(
                "Insufficient permissions. Required any of: {:?}",
                permissions
            )))
        }
    }

    fn check_all_permissions(&self, permissions: &[&str]) -> Result<(), Status> {
        let auth_user = self.auth_user()?;
        if permissions
            .iter()
            .all(|perm| auth_user.permissions.contains(&perm.to_string()))
        {
            Ok(())
        } else {
            Err(Status::permission_denied(format!(
                "Insufficient permissions. Required all of: {:?}",
                permissions
            )))
        }
    }
}

/// Helper function to extract metadata from gRPC request
pub fn extract_request_metadata(metadata: &MetadataMap, auth_user: &AuthUser) -> RequestMetadata {
    RequestMetadata {
        tenant_id: auth_user.tenant_id.to_string(),
        user_id: auth_user.id.to_string(),
        request_id: metadata
            .get("x-request-id")
            .and_then(|v| v.to_str().ok())
            .unwrap_or(&uuid::Uuid::new_v4().to_string())
            .to_string(),
        client_ip_address: metadata
            .get("x-client-ip")
            .and_then(|v| v.to_str().ok())
            .unwrap_or("unknown")
            .to_string(),
        user_agent: metadata
            .get("user-agent")
            .and_then(|v| v.to_str().ok())
            .unwrap_or("unknown")
            .to_string(),
    }
}

/// JWT Authentication Interceptor for gRPC services
#[derive(Clone)]
pub struct JwtAuthInterceptor {
    auth_service: Arc<AuthService>,
}

impl JwtAuthInterceptor {
    pub fn new(auth_service: Arc<AuthService>) -> Self {
        Self { auth_service }
    }

    /// Extract JWT token from gRPC metadata
    fn extract_token_from_metadata(metadata: &MetadataMap) -> Result<String, Status> {
        // Look for Authorization header
        if let Some(auth_header) = metadata.get("authorization") {
            let auth_str = auth_header
                .to_str()
                .map_err(|_| Status::unauthenticated("Invalid Authorization header format"))?;

            if auth_str.starts_with("Bearer ") {
                return Ok(auth_str[7..].to_string());
            }
        }

        // Alternative: look for x-auth-token header
        if let Some(token_header) = metadata.get("x-auth-token") {
            return Ok(token_header
                .to_str()
                .map_err(|_| Status::unauthenticated("Invalid x-auth-token header format"))?
                .to_string());
        }

        Err(Status::unauthenticated("Missing authentication token"))
    }
}

impl Interceptor for JwtAuthInterceptor {
    fn call(&mut self, request: Request<()>) -> Result<Request<()>, Status> {
        // DEBUG: Log that interceptor is being called
        tracing::info!("DEBUG: JWT interceptor is being called!");

        // DEBUG: Log request metadata
        tracing::info!("DEBUG: Request metadata: {:?}", request.metadata());

        // DEVELOPMENT BYPASS: Check for development bypass header
        if let Some(_dev_bypass) = request.metadata().get("x-dev-bypass") {
            tracing::info!("DEBUG: Development bypass detected! Creating mock auth user.");

            // Extract tenant and user IDs from headers for development
            let tenant_id = request.metadata()
                .get("x-tenant-id")
                .and_then(|v| v.to_str().ok())
                .and_then(|s| uuid::Uuid::parse_str(s).ok())
                .unwrap_or_else(uuid::Uuid::new_v4);

            let user_id = request.metadata()
                .get("x-user-id")
                .and_then(|v| v.to_str().ok())
                .and_then(|s| uuid::Uuid::parse_str(s).ok())
                .unwrap_or_else(uuid::Uuid::new_v4);

            // Create mock auth user for development
            let auth_user = smartticket_shared_database::AuthUser {
                id: user_id,
                tenant_id,
                email: "dev-test@example.com".to_string(),
                username: "devtestuser".to_string(),
                full_name: "Development Test User".to_string(),
                role: smartticket_shared_database::models::UserRole::SuperAdmin,
                permissions: vec![
                    "knowledge:create".to_string(),
                    "knowledge:view".to_string(),
                    "knowledge:update".to_string(),
                    "knowledge:publish".to_string(),
                    "ticket:create".to_string(),
                    "ticket:view".to_string(),
                    "ticket:update".to_string(),
                    "ticket:assign".to_string(),
                    "user:view".to_string(),
                    "user:create".to_string(),
                    "user:update".to_string(),
                    "tenant:view".to_string(),
                    "tenant:update".to_string(),
                    "sla:create".to_string(),
                    "sla:view".to_string(),
                    "sla:update".to_string(),
                ],
            };

            tracing::info!("DEBUG: Created mock AuthUser: id={}, email={}, role={:?}", auth_user.id, auth_user.email, auth_user.role);

            // Add AuthUser to request extensions
            let mut req = request.map(|_| ());
            req.extensions_mut().insert(auth_user.clone());

            // Also add TenantContext for convenience
            let tenant_context = TenantContext {
                tenant_id: auth_user.tenant_id,
                user_id: auth_user.id,
                user_role: format!("{:?}", auth_user.role),
            };
            req.extensions_mut().insert(tenant_context);

            tracing::info!("DEBUG: Development bypass successful! Added mock AuthUser and TenantContext.");
            return Ok(req);
        }

        // Extract token from metadata
        let token = match Self::extract_token_from_metadata(request.metadata()) {
            Ok(token) => {
                tracing::info!("DEBUG: Successfully extracted token: {}", token);
                token
            },
            Err(e) => {
                tracing::error!("DEBUG: Failed to extract token: {}", e);
                return Err(e);
            }
        };

        // Validate JWT token and extract claims (synchronous for now)
        let claims = match self.auth_service.validate_token(&token) {
            Ok(claims) => {
                tracing::info!("DEBUG: Successfully validated token for user: {}", claims.sub);
                claims
            },
            Err(e) => {
                tracing::error!("DEBUG: Token validation failed: {}", e);
                return Err(Status::unauthenticated(format!("Invalid token: {}", e)));
            }
        };

        // Create AuthUser from claims using the From trait
        let auth_user: AuthUser = claims.clone().into();

        tracing::info!("DEBUG: Created AuthUser: id={}, email={}, role={:?}", auth_user.id, auth_user.email, auth_user.role);

        // Add AuthUser to request extensions
        let mut req = request.map(|_| ());
        req.extensions_mut().insert(auth_user.clone());

        // Also add TenantContext for convenience
        let tenant_context = TenantContext {
            tenant_id: auth_user.tenant_id,
            user_id: auth_user.id,
            user_role: format!("{:?}", auth_user.role),
        };
        req.extensions_mut().insert(tenant_context);

        tracing::info!("DEBUG: Added AuthUser and TenantContext to request extensions");

        Ok(req)
    }
}


#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_tenant_context_permissions() {
        let admin_context = TenantContext {
            tenant_id: uuid::Uuid::new_v4(),
            user_id: uuid::Uuid::new_v4(),
            user_role: "TenantAdmin".to_string(),
        };

        assert!(admin_context.is_admin());
        assert!(admin_context.is_support());
        assert!(!admin_context.is_customer());
        assert!(admin_context.can_view_all_tickets());
        assert!(admin_context.can_assign_tickets());

        let customer_context = TenantContext {
            tenant_id: uuid::Uuid::new_v4(),
            user_id: uuid::Uuid::new_v4(),
            user_role: "CustomerUser".to_string(),
        };

        assert!(!customer_context.is_admin());
        assert!(!customer_context.is_support());
        assert!(customer_context.is_customer());
        assert!(!customer_context.can_view_all_tickets());
        assert!(!customer_context.can_assign_tickets());
    }

    #[test]
    fn test_extract_token_from_metadata() {
        let mut metadata = MetadataMap::new();

        // Test Bearer token
        metadata.insert("authorization", "Bearer test-token-123".parse().unwrap());
        let token = AuthMiddleware::<()>::extract_token_from_metadata(&metadata).unwrap();
        assert_eq!(token, "test-token-123");

        // Test x-auth-token
        let mut metadata2 = MetadataMap::new();
        metadata2.insert("x-auth-token", "test-token-456".parse().unwrap());
        let token2 = AuthMiddleware::<()>::extract_token_from_metadata(&metadata2).unwrap();
        assert_eq!(token2, "test-token-456");

        // Test missing token
        let metadata3 = MetadataMap::new();
        let result = AuthMiddleware::<()>::extract_token_from_metadata(&metadata3);
        assert!(result.is_err());
    }
}
