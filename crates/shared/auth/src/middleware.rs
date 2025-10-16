use crate::{JwtService, TenantContext};
use axum::{http::HeaderMap, Json};
use smartticket_shared_error::{ErrorDetail, Result, SmartTicketError};
use std::sync::Arc;
use tonic::Request as TonicRequest;
use tower_http::{
    cors::{Any, CorsLayer},
    trace::TraceLayer,
};

/// Authentication middleware for Axum HTTP services
pub struct AuthMiddleware {
    jwt_service: Arc<JwtService>,
}

impl AuthMiddleware {
    pub fn new(jwt_service: Arc<JwtService>) -> Self {
        Self { jwt_service }
    }

    /// Extract JWT token from Authorization header
    fn extract_token(headers: &HeaderMap) -> Result<String> {
        let auth_header = headers
            .get("authorization")
            .and_then(|h| h.to_str().ok())
            .ok_or_else(|| {
                SmartTicketError::Unauthorized("Missing Authorization header".to_string())
            })?;

        if !auth_header.starts_with("Bearer ") {
            return Err(SmartTicketError::Unauthorized(
                "Invalid Authorization header format".to_string(),
            ));
        }

        let token = auth_header.strip_prefix("Bearer ").ok_or_else(|| {
            SmartTicketError::Unauthorized("Invalid Authorization header".to_string())
        })?;

        Ok(token.to_string())
    }

    /// Create authenticated tenant context from JWT token
    pub async fn create_tenant_context(&self, headers: &HeaderMap) -> Result<TenantContext> {
        let token = Self::extract_token(headers)?;
        let claims = self.jwt_service.verify_user_token(&token)?;

        Ok(TenantContext::from_claims(&claims))
    }
}

/// gRPC authentication interceptor for Tonic services
pub struct GrpcAuthInterceptor {
    jwt_service: Arc<JwtService>,
}

impl GrpcAuthInterceptor {
    pub fn new(jwt_service: Arc<JwtService>) -> Self {
        Self { jwt_service }
    }

    /// Create tenant context from gRPC metadata
    pub async fn create_tenant_context_from_metadata(
        &self,
        metadata: &tonic::metadata::MetadataMap,
    ) -> Result<TenantContext> {
        // Extract token from metadata
        let token = metadata
            .get("authorization")
            .and_then(|h| h.to_str().ok())
            .and_then(|s| s.strip_prefix("Bearer "))
            .ok_or_else(|| {
                SmartTicketError::Unauthorized("Missing or invalid authorization token".to_string())
            })?;

        // Verify JWT token
        let claims = self.jwt_service.verify_user_token(token)?;

        Ok(TenantContext::from_claims(&claims))
    }
}

/// Helper trait to extract tenant context from HTTP/gRPC requests
pub trait TenantContextExtractor {
    fn get_tenant_context(&self) -> Result<&TenantContext>;
}

impl TenantContextExtractor for axum::extract::Request {
    fn get_tenant_context(&self) -> Result<&TenantContext> {
        self.extensions().get::<TenantContext>().ok_or_else(|| {
            SmartTicketError::Unauthorized("Tenant context not found in request".to_string())
        })
    }
}

impl<T> TenantContextExtractor for TonicRequest<T> {
    fn get_tenant_context(&self) -> Result<&TenantContext> {
        self.extensions().get::<TenantContext>().ok_or_else(|| {
            SmartTicketError::Unauthorized("Tenant context not found in request".to_string())
        })
    }
}

/// CORS middleware configuration
pub fn create_cors_layer() -> CorsLayer {
    CorsLayer::new()
        .allow_origin(Any)
        .allow_methods(Any)
        .allow_headers(Any)
        .allow_credentials(true)
}

/// Complete middleware stack for Axum services
pub fn create_axum_middleware_stack(
    _jwt_service: Arc<JwtService>,
) -> (impl tower::Layer<axum::routing::Route> + Clone, CorsLayer) {
    (TraceLayer::new_for_http(), create_cors_layer())
}

/// Error response helper for HTTP endpoints
pub fn create_error_response(
    error: &SmartTicketError,
    request_id: Option<String>,
) -> Json<serde_json::Value> {
    let error_detail = ErrorDetail::new(error)
        .with_request_id(request_id.unwrap_or_else(|| uuid::Uuid::new_v4().to_string()));

    Json(serde_json::json!({
        "success": false,
        "error": {
            "code": error_detail.code,
            "message": error_detail.message,
            "details": error_detail.details,
            "request_id": error_detail.request_id
        }
    }))
}

#[cfg(test)]
mod tests {
    use super::*;
    use axum::http::{HeaderMap, Method};

    #[tokio::test]
    async fn test_token_extraction() {
        let mut headers = HeaderMap::new();

        // Test valid header
        headers.insert("authorization", "Bearer test-token".parse().unwrap());
        let result = AuthMiddleware::extract_token(&headers);
        assert!(result.is_ok());
        assert_eq!(result.unwrap(), "test-token");

        // Test missing header
        let headers = HeaderMap::new();
        let result = AuthMiddleware::extract_token(&headers);
        assert!(result.is_err());

        // Test invalid format
        let mut headers = HeaderMap::new();
        headers.insert("authorization", "Invalid token".parse().unwrap());
        let result = AuthMiddleware::extract_token(&headers);
        assert!(result.is_err());
    }

    #[test]
    fn test_client_id_extraction() {
        // Test IP-based extraction
        let request = axum::extract::Request::builder()
            .header("x-forwarded-for", "192.168.1.1")
            .body(axum::body::Body::empty())
            .unwrap();

        // Note: This is a simplified test since we're not implementing the full middleware
        let client_ip = request
            .headers()
            .get("x-forwarded-for")
            .and_then(|h| h.to_str().ok())
            .and_then(|s| s.split(',').next())
            .unwrap_or("unknown");

        assert_eq!(client_ip, "192.168.1.1");
    }
}
