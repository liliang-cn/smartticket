//! Gateway Middleware
//!
//! Authentication, authorization, and request processing middleware for the HTTP gateway.

use axum::{
    extract::{Request, State},
    http::{header, StatusCode},
    middleware::Next,
    response::{Response, IntoResponse},
    Json,
};
use axum::http::HeaderMap;
use tower_http::cors::CorsLayer;
use std::sync::Arc;
use jsonwebtoken::{decode, encode, DecodingKey, EncodingKey, Header, Validation};
use serde::{Deserialize, Serialize};
use chrono::{Duration, Utc};
use uuid::Uuid;

use crate::gateway::{GatewayConfig, error::GatewayError};
use crate::utils::response::{ApiResponse, ApiError};

/// JWT Claims structure
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct Claims {
    /// Subject (user ID)
    pub sub: String,
    /// Tenant ID
    pub tenant_id: String,
    /// User roles
    pub roles: Vec<String>,
    /// User permissions
    pub permissions: Vec<String>,
    /// Issued at timestamp
    pub iat: i64,
    /// Expiration timestamp
    pub exp: i64,
    /// Issuer
    pub iss: String,
    /// Audience
    pub aud: String,
    /// JWT ID (unique token identifier)
    pub jti: String,
    /// Token type (access, refresh)
    pub token_type: String,
    /// Session ID
    pub session_id: Option<String>,
}

/// Authentication context for requests
#[derive(Debug, Clone)]
pub struct AuthContext {
    pub user_id: String,
    pub tenant_id: String,
    pub roles: Vec<String>,
    pub permissions: Vec<String>,
    pub session_id: Option<String>,
    pub token_jti: String,
}

/// Token management service
#[derive(Clone)]
pub struct TokenManager {
    encoding_key: EncodingKey,
    decoding_key: DecodingKey,
    config: Arc<GatewayConfig>,
}

impl TokenManager {
    /// Create new token manager
    pub fn new(config: Arc<GatewayConfig>) -> Self {
        let jwt_secret = &config.auth.jwt_secret;

        Self {
            encoding_key: EncodingKey::from_secret(jwt_secret.as_ref()),
            decoding_key: DecodingKey::from_secret(jwt_secret.as_ref()),
            config,
        }
    }

    /// Generate access token
    pub fn generate_access_token(
        &self,
        user_id: &str,
        tenant_id: &str,
        roles: Vec<String>,
        permissions: Vec<String>,
        session_id: Option<String>,
    ) -> Result<String, GatewayError> {
        let now = Utc::now();
        let exp = now + Duration::hours(1); // 1 hour expiry for access tokens

        let claims = Claims {
            sub: user_id.to_string(),
            tenant_id: tenant_id.to_string(),
            roles,
            permissions,
            iat: now.timestamp(),
            exp: exp.timestamp(),
            iss: "smartticket-gateway".to_string(),
            aud: "smartticket-api".to_string(),
            jti: Uuid::new_v4().to_string(),
            token_type: "access".to_string(),
            session_id,
        };

        self.generate_token(&claims)
    }

    /// Generate refresh token
    pub fn generate_refresh_token(
        &self,
        user_id: &str,
        tenant_id: &str,
        session_id: Option<String>,
    ) -> Result<String, GatewayError> {
        let now = Utc::now();
        let exp = now + Duration::days(7); // 7 days expiry for refresh tokens

        let claims = Claims {
            sub: user_id.to_string(),
            tenant_id: tenant_id.to_string(),
            roles: vec![],
            permissions: vec![],
            iat: now.timestamp(),
            exp: exp.timestamp(),
            iss: "smartticket-gateway".to_string(),
            aud: "smartticket-api".to_string(),
            jti: Uuid::new_v4().to_string(),
            token_type: "refresh".to_string(),
            session_id,
        };

        self.generate_token(&claims)
    }

    /// Generate token from claims
    fn generate_token(&self, claims: &Claims) -> Result<String, GatewayError> {
        encode(
            &Header::default(),
            claims,
            &self.encoding_key,
        ).map_err(|e| GatewayError::AuthError(format!("Token generation failed: {}", e)))
    }

    /// Validate and decode token
    pub fn validate_token(&self, token: &str) -> Result<Claims, GatewayError> {
        let token_data = decode::<Claims>(
            token,
            &self.decoding_key,
            &Validation::new(jsonwebtoken::Algorithm::HS256),
        ).map_err(|e| {
            match e.kind() {
                jsonwebtoken::errors::ErrorKind::ExpiredSignature => {
                    GatewayError::AuthError("Token has expired".to_string())
                }
                jsonwebtoken::errors::ErrorKind::InvalidToken => {
                    GatewayError::AuthError("Invalid token format".to_string())
                }
                jsonwebtoken::errors::ErrorKind::InvalidSignature => {
                    GatewayError::AuthError("Invalid token signature".to_string())
                }
                _ => GatewayError::AuthError(format!("Token validation failed: {}", e))
            }
        })?;

        Ok(token_data.claims)
    }

    /// Extract token from Authorization header
    pub fn extract_token_from_header(headers: &HeaderMap) -> Option<String> {
        headers
            .get(header::AUTHORIZATION)
            .and_then(|h| h.to_str().ok())
            .and_then(|auth_header| {
                if auth_header.starts_with("Bearer ") {
                    Some(auth_header[7..].to_string())
                } else {
                    None
                }
            })
    }

    /// Create authentication context from claims
    pub fn create_auth_context(&self, claims: &Claims) -> AuthContext {
        AuthContext {
            user_id: claims.sub.clone(),
            tenant_id: claims.tenant_id.clone(),
            roles: claims.roles.clone(),
            permissions: claims.permissions.clone(),
            session_id: claims.session_id.clone(),
            token_jti: claims.jti.clone(),
        }
    }

    /// Check if user has required permission
    pub fn has_permission(&self, auth_context: &AuthContext, required_permission: &str) -> bool {
        auth_context.permissions.contains(&required_permission.to_string())
    }

    /// Check if user has required role
    pub fn has_role(&self, auth_context: &AuthContext, required_role: &str) -> bool {
        auth_context.roles.contains(&required_role.to_string())
    }

    /// Check if user is admin
    pub fn is_admin(&self, auth_context: &AuthContext) -> bool {
        self.has_role(auth_context, "admin") || self.has_role(auth_context, "super_admin")
    }

    /// Check if user can access tenant resource
    pub fn can_access_tenant(&self, auth_context: &AuthContext, tenant_id: &str) -> bool {
        auth_context.tenant_id == tenant_id || self.is_admin(auth_context)
    }
}

/// Enhanced JWT authentication middleware
pub async fn jwt_auth_middleware(
    State(token_manager): State<Arc<TokenManager>>,
    request: Request,
    next: Next,
) -> Result<Response, GatewayError> {
    // Skip authentication for certain endpoints
    let path = request.uri().path();
    if should_skip_auth(path) {
        return Ok(next.run(request).await);
    }

    // Extract token from header
    let token = TokenManager::extract_token_from_header(request.headers())
        .ok_or_else(|| GatewayError::AuthError("Missing or invalid Authorization header".to_string()))?;

    // Validate token
    let claims = token_manager.validate_token(&token)?;

    // Create auth context
    let auth_context = token_manager.create_auth_context(&claims);

    // Log successful authentication
    tracing::info!(
        user_id = %auth_context.user_id,
        tenant_id = %auth_context.tenant_id,
        path = %path,
        "JWT authentication successful"
    );

    // Continue with the request, adding auth context to extensions
    let mut request = request;
    request.extensions_mut().insert(auth_context.clone());

    // Add user context headers
    if let Ok(user_id_header) = auth_context.user_id.parse() {
        request.headers_mut().insert("x-user-id", user_id_header);
    }
    if let Ok(tenant_id_header) = auth_context.tenant_id.parse() {
        request.headers_mut().insert("x-tenant-id", tenant_id_header);
    }
    if let Ok(roles_header) = auth_context.roles.join(",").parse() {
        request.headers_mut().insert("x-user-roles", roles_header);
    }

    Ok(next.run(request).await)
}

/// Check if authentication should be skipped for the given path
fn should_skip_auth(path: &str) -> bool {
    let skip_paths = [
        "/health",
        "/docs",
        "/openapi.yaml",
        "/openapi.json",
        "/swagger-ui.html",
        "/",
        "/auth/v1/login",
        "/auth/v1/register",
        "/auth/v1/forgot-password",
        "/auth/v1/reset-password",
    ];

    skip_paths.iter().any(|&skip_path| path == skip_path || path.starts_with(skip_path))
}

/// Enhanced tenant validation middleware
pub async fn tenant_middleware(
    State(_token_manager): State<Arc<TokenManager>>,
    request: Request,
    next: Next,
) -> Response {
    // Extract tenant from header or auth context
    let tenant_id = request
        .headers()
        .get("x-tenant-id")
        .and_then(|h| h.to_str().ok())
        .map(|s| s.to_string())
        .or_else(|| {
            request.extensions().get::<AuthContext>()
                .map(|ctx| ctx.tenant_id.clone())
        });

    match tenant_id {
        Some(tenant) => {
            // Validate tenant ID format
            if is_valid_tenant_id(&tenant) {
                tracing::debug!(tenant_id = %tenant, "Tenant validation successful");
                next.run(request).await
            } else {
                tracing::warn!(tenant_id = %tenant, "Invalid tenant ID format");
                create_error_response(StatusCode::BAD_REQUEST, "Invalid tenant ID format")
            }
        }
        None => {
            tracing::warn!("Missing tenant ID");
            create_error_response(StatusCode::BAD_REQUEST, "Missing tenant ID")
        }
    }
}

/// Validate tenant ID format
fn is_valid_tenant_id(tenant_id: &str) -> bool {
    // Tenant ID should be alphanumeric with optional hyphens, 3-50 characters
    let re = regex::Regex::new(r"^[a-zA-Z0-9-]{3,50}$").unwrap();
    re.is_match(tenant_id)
}

/// Create error response
fn create_error_response(status: StatusCode, message: &str) -> Response {
    let api_error = ApiError::new(
        "TENANT_VALIDATION_ERROR",
        message,
        status,
    );

    let response = ApiResponse::<()>::error(vec![api_error]);
    (status, Json(response)).into_response()
}

/// Role-based authorization middleware
pub async fn require_role_middleware(
    request: Request,
    next: Next,
    required_role: &'static str,
) -> Result<Response, GatewayError> {
    let auth_context = request.extensions().get::<AuthContext>()
        .ok_or_else(|| GatewayError::AuthorizationError("Authentication required".to_string()))?;

    if !auth_context.roles.contains(&required_role.to_string()) {
        return Err(GatewayError::AuthorizationError(
            format!("Insufficient privileges. Required role: {}", required_role)
        ));
    }

    Ok(next.run(request).await)
}

/// Permission-based authorization middleware
pub async fn require_permission_middleware(
    request: Request,
    next: Next,
    required_permission: &'static str,
) -> Result<Response, GatewayError> {
    let auth_context = request.extensions().get::<AuthContext>()
        .ok_or_else(|| GatewayError::AuthorizationError("Authentication required".to_string()))?;

    if !auth_context.permissions.contains(&required_permission.to_string()) {
        return Err(GatewayError::AuthorizationError(
            format!("Insufficient privileges. Required permission: {}", required_permission)
        ));
    }

    Ok(next.run(request).await)
}

/// Multi-tenant authorization middleware
pub async fn tenant_authorization_middleware(
    request: Request,
    next: Next,
) -> Result<Response, GatewayError> {
    let auth_context = request.extensions().get::<AuthContext>()
        .ok_or_else(|| GatewayError::AuthorizationError("Authentication required".to_string()))?;

    // Extract requested tenant from path or header
    let requested_tenant = request
        .headers()
        .get("x-requested-tenant-id")
        .and_then(|h| h.to_str().ok())
        .unwrap_or(&auth_context.tenant_id);

    // Check if user can access the requested tenant
    if auth_context.tenant_id != requested_tenant && !auth_context.roles.contains(&"admin".to_string()) {
        return Err(GatewayError::AuthorizationError(
            "Access denied: Cannot access cross-tenant resources".to_string()
        ));
    }

    Ok(next.run(request).await)
}

/// Rate limiting middleware
pub async fn rate_limit_middleware(
    request: Request,
    next: Next,
) -> Response {
    // TODO: Implement rate limiting based on tenant and user
    // For now, just pass through
    next.run(request).await
}

/// CORS configuration
pub fn create_cors_layer() -> CorsLayer {
    CorsLayer::new()
        .allow_origin([
            "http://localhost:3000".parse().unwrap(),
            "http://localhost:3286".parse().unwrap(),
            "http://localhost:7218".parse().unwrap(),
        ])
        .allow_methods([
            axum::http::Method::GET,
            axum::http::Method::POST,
            axum::http::Method::PUT,
            axum::http::Method::DELETE,
            axum::http::Method::PATCH,
            axum::http::Method::OPTIONS,
        ])
        .allow_headers([
            header::AUTHORIZATION,
            header::ACCEPT,
            header::CONTENT_TYPE,
            header::ORIGIN,
            header::USER_AGENT,
            "x-tenant-id".parse().unwrap(),
            "x-user-id".parse().unwrap(),
            "x-request-id".parse().unwrap(),
            "x-user-roles".parse().unwrap(),
        ])
        .allow_credentials(true)
}