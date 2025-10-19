//! Gateway Error Handling
//!
//! Comprehensive error handling for HTTP-to-gRPC gateway including
//! error mapping, response formatting, and error logging.

use axum::response::{IntoResponse, Response};
use axum::http::StatusCode;
use tonic::{Code, Status};
use crate::utils::response::{ApiError, ApiResponse};
use tracing::{error, warn, debug};

/// Gateway error types
#[derive(Debug, thiserror::Error)]
pub enum GatewayError {
    /// gRPC service error
    #[error("gRPC service error: {0}")]
    GrpcError(#[from] Status),

    /// Authentication error
    #[error("Authentication error: {0}")]
    AuthError(String),

    /// Authorization error
    #[error("Authorization error: {0}")]
    AuthorizationError(String),

    /// Validation error
    #[error("Validation error: {0}")]
    ValidationError(String),

    /// Tenant error
    #[error("Tenant error: {0}")]
    TenantError(String),

    /// Rate limiting error
    #[error("Rate limit exceeded")]
    RateLimitError,

    /// Service unavailable error
    #[error("Service unavailable: {0}")]
    ServiceUnavailable(String),

    /// Internal server error
    #[error("Internal server error: {0}")]
    InternalError(String),

    /// Not found error
    #[error("Resource not found: {0}")]
    NotFound(String),

    /// Conflict error
    #[error("Resource conflict: {0}")]
    Conflict(String),

    /// Bad request error
    #[error("Bad request: {0}")]
    BadRequest(String),
}

impl GatewayError {
    /// Convert to API error
    pub fn to_api_error(&self) -> ApiError {
        match self {
            GatewayError::GrpcError(status) => self.map_grpc_status(status),
            GatewayError::AuthError(msg) => ApiError::unauthorized(msg.clone()),
            GatewayError::AuthorizationError(msg) => ApiError::forbidden(msg.clone()),
            GatewayError::ValidationError(msg) => ApiError::validation(msg.clone(), None::<String>),
            GatewayError::TenantError(msg) => ApiError::validation(msg.clone(), Some("tenant".to_string())),
            GatewayError::RateLimitError => ApiError::new(
                "RATE_LIMIT_EXCEEDED",
                "Rate limit exceeded. Please try again later.",
                StatusCode::TOO_MANY_REQUESTS,
            ),
            GatewayError::ServiceUnavailable(msg) => ApiError::new(
                "SERVICE_UNAVAILABLE",
                format!("Service temporarily unavailable: {}", msg),
                StatusCode::SERVICE_UNAVAILABLE,
            ),
            GatewayError::InternalError(msg) => ApiError::internal(msg.clone()),
            GatewayError::NotFound(msg) => ApiError::not_found(msg.clone()),
            GatewayError::Conflict(msg) => ApiError::conflict(msg.clone()),
            GatewayError::BadRequest(msg) => ApiError::validation(msg.clone(), None::<String>),
        }
    }

    /// Map gRPC status to HTTP API error
    fn map_grpc_status(&self, status: &Status) -> ApiError {
        let (code, message, http_status) = match status.code() {
            Code::Ok => return ApiError::internal("Unexpected OK error"),
            Code::Cancelled => (
                "CANCELLED",
                "Request was cancelled by the client".to_string(),
                StatusCode::REQUEST_TIMEOUT,
            ),
            Code::Unknown => (
                "UNKNOWN",
                format!("Unknown error: {}", status.message()),
                StatusCode::INTERNAL_SERVER_ERROR,
            ),
            Code::InvalidArgument => (
                "INVALID_ARGUMENT",
                format!("Invalid argument: {}", status.message()),
                StatusCode::BAD_REQUEST,
            ),
            Code::DeadlineExceeded => (
                "DEADLINE_EXCEEDED",
                "Request deadline exceeded".to_string(),
                StatusCode::REQUEST_TIMEOUT,
            ),
            Code::NotFound => (
                "NOT_FOUND",
                format!("Resource not found: {}", status.message()),
                StatusCode::NOT_FOUND,
            ),
            Code::AlreadyExists => (
                "ALREADY_EXISTS",
                format!("Resource already exists: {}", status.message()),
                StatusCode::CONFLICT,
            ),
            Code::PermissionDenied => (
                "PERMISSION_DENIED",
                format!("Permission denied: {}", status.message()),
                StatusCode::FORBIDDEN,
            ),
            Code::ResourceExhausted => (
                "RESOURCE_EXHAUSTED",
                "Resource exhausted (quota exceeded)".to_string(),
                StatusCode::TOO_MANY_REQUESTS,
            ),
            Code::FailedPrecondition => (
                "FAILED_PRECONDITION",
                format!("Failed precondition: {}", status.message()),
                StatusCode::BAD_REQUEST,
            ),
            Code::Aborted => (
                "ABORTED",
                "Operation was aborted".to_string(),
                StatusCode::CONFLICT,
            ),
            Code::OutOfRange => (
                "OUT_OF_RANGE",
                format!("Out of range: {}", status.message()),
                StatusCode::BAD_REQUEST,
            ),
            Code::Unimplemented => (
                "UNIMPLEMENTED",
                format!("Operation not implemented: {}", status.message()),
                StatusCode::NOT_IMPLEMENTED,
            ),
            Code::Internal => (
                "INTERNAL",
                "Internal server error".to_string(),
                StatusCode::INTERNAL_SERVER_ERROR,
            ),
            Code::Unavailable => (
                "UNAVAILABLE",
                format!("Service unavailable: {}", status.message()),
                StatusCode::SERVICE_UNAVAILABLE,
            ),
            Code::DataLoss => (
                "DATA_LOSS",
                "Data loss occurred".to_string(),
                StatusCode::INTERNAL_SERVER_ERROR,
            ),
            Code::Unauthenticated => (
                "UNAUTHENTICATED",
                format!("Authentication required: {}", status.message()),
                StatusCode::UNAUTHORIZED,
            ),
        };

        let mut api_error = ApiError::new(code, message, http_status);
        if !status.details().is_empty() {
            if let Ok(details_str) = std::str::from_utf8(status.details()) {
                let mut detail_map = std::collections::HashMap::new();
                detail_map.insert("grpc_details".to_string(), serde_json::Value::String(details_str.to_string()));
                api_error = api_error.with_details(detail_map);
            }
        }
        api_error
    }

    /// Check if error should be logged as error vs warning
    fn is_server_error(&self) -> bool {
        matches!(
            self,
            GatewayError::GrpcError(_)
            | GatewayError::ServiceUnavailable(_)
            | GatewayError::InternalError(_)
        )
    }

    /// Log the error with appropriate level
    pub fn log(&self, context: &str) {
        if self.is_server_error() {
            error!(
                error = %self,
                context = context,
                "Gateway server error"
            );
        } else {
            warn!(
                error = %self,
                context = context,
                "Gateway client error"
            );
        }
    }
}

impl IntoResponse for GatewayError {
    fn into_response(self) -> Response {
        self.log("HTTP response");
        let api_error = self.to_api_error();
        ApiResponse::<()>::single_error(api_error).into_response()
    }
}

/// Error mapping utilities
pub struct ErrorMapper;

impl ErrorMapper {
    /// Map database errors to gateway errors
    pub fn map_database_error(err: &sqlx::Error) -> GatewayError {
        match err {
            sqlx::Error::RowNotFound => GatewayError::NotFound("Record not found".to_string()),
            sqlx::Error::Database(db_err) => {
                if db_err.is_unique_violation() {
                    GatewayError::Conflict("Resource already exists".to_string())
                } else if db_err.is_foreign_key_violation() {
                    GatewayError::ValidationError("Referenced resource does not exist".to_string())
                } else {
                    GatewayError::InternalError(format!("Database error: {}", db_err))
                }
            }
            sqlx::Error::PoolTimedOut => {
                GatewayError::ServiceUnavailable("Database connection timeout".to_string())
            }
            _ => GatewayError::InternalError(format!("Database error: {}", err)),
        }
    }

    /// Map Redis errors to gateway errors
    pub fn map_redis_error(err: &str) -> GatewayError {
        error!("Redis error: {}", err);
        GatewayError::ServiceUnavailable("Cache service unavailable".to_string())
    }

    /// Map JSON parsing errors to gateway errors
    pub fn map_json_error(err: &serde_json::Error) -> GatewayError {
        debug!("JSON parsing error: {}", err);
        GatewayError::BadRequest("Invalid JSON format".to_string())
    }
}

/// Error context for better debugging
#[derive(Debug, Clone)]
pub struct ErrorContext {
    pub service: String,
    pub operation: String,
    pub user_id: Option<String>,
    pub tenant_id: Option<String>,
    pub request_id: Option<String>,
}

impl ErrorContext {
    /// Create new error context
    pub fn new(service: &str, operation: &str) -> Self {
        Self {
            service: service.to_string(),
            operation: operation.to_string(),
            user_id: None,
            tenant_id: None,
            request_id: None,
        }
    }

    /// Add user ID to context
    pub fn with_user_id(mut self, user_id: &str) -> Self {
        self.user_id = Some(user_id.to_string());
        self
    }

    /// Add tenant ID to context
    pub fn with_tenant_id(mut self, tenant_id: &str) -> Self {
        self.tenant_id = Some(tenant_id.to_string());
        self
    }

    /// Add request ID to context
    pub fn with_request_id(mut self, request_id: &str) -> Self {
        self.request_id = Some(request_id.to_string());
        self
    }

    /// Format context for logging
    pub fn format(&self) -> String {
        let mut parts = vec![
            format!("service={}", self.service),
            format!("operation={}", self.operation),
        ];

        if let Some(ref user_id) = self.user_id {
            parts.push(format!("user_id={}", user_id));
        }
        if let Some(ref tenant_id) = self.tenant_id {
            parts.push(format!("tenant_id={}", tenant_id));
        }
        if let Some(ref request_id) = self.request_id {
            parts.push(format!("request_id={}", request_id));
        }

        parts.join(", ")
    }
}

/// Error handler trait for consistent error processing
pub trait ErrorHandler {
    /// Handle error with context
    fn handle_with_context(&self, error: GatewayError, context: &ErrorContext) -> GatewayError {
        let context_str = context.format();
        error.log(&context_str);
        error
    }

    /// Handle gRPC status with context
    fn handle_grpc_with_context(&self, status: Status, context: &ErrorContext) -> GatewayError {
        let gateway_error = GatewayError::GrpcError(status);
        self.handle_with_context(gateway_error, context)
    }
}

/// Default error handler implementation
pub struct DefaultErrorHandler;

impl ErrorHandler for DefaultErrorHandler {}

/// Error response builder for consistent error responses
pub struct ErrorResponseBuilder;

impl ErrorResponseBuilder {
    /// Create validation error response
    pub fn validation(field: &str, message: &str) -> GatewayError {
        GatewayError::ValidationError(format!("{}: {}", field, message))
    }

    /// Create authentication error response
    pub fn authentication(message: &str) -> GatewayError {
        GatewayError::AuthError(message.to_string())
    }

    /// Create authorization error response
    pub fn authorization(message: &str) -> GatewayError {
        GatewayError::AuthorizationError(message.to_string())
    }

    /// Create tenant error response
    pub fn tenant(message: &str) -> GatewayError {
        GatewayError::TenantError(message.to_string())
    }

    /// Create not found error response
    pub fn not_found(resource: &str) -> GatewayError {
        GatewayError::NotFound(format!("{} not found", resource))
    }

    /// Create conflict error response
    pub fn conflict(resource: &str) -> GatewayError {
        GatewayError::Conflict(format!("{} already exists", resource))
    }

    /// Create rate limit error response
    pub fn rate_limit() -> GatewayError {
        GatewayError::RateLimitError
    }

    /// Create service unavailable error response
    pub fn service_unavailable(service: &str) -> GatewayError {
        GatewayError::ServiceUnavailable(format!("{} is currently unavailable", service))
    }

    /// Create internal error response
    pub fn internal(message: &str) -> GatewayError {
        GatewayError::InternalError(message.to_string())
    }

    /// Create bad request error response
    pub fn bad_request(message: &str) -> GatewayError {
        GatewayError::BadRequest(message.to_string())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_grpc_error_mapping() {
        let not_found_status = Status::new(Code::NotFound, "User not found");
        let gateway_error = GatewayError::GrpcError(not_found_status);
        let api_error = gateway_error.to_api_error();

        assert_eq!(api_error.code, "NOT_FOUND");
        assert_eq!(api_error.status_code, StatusCode::NOT_FOUND);
    }

    #[test]
    fn test_error_context() {
        let context = ErrorContext::new("UserService", "GetUser")
            .with_user_id("user123")
            .with_tenant_id("tenant456")
            .with_request_id("req789");

        let formatted = context.format();
        assert!(formatted.contains("service=UserService"));
        assert!(formatted.contains("operation=GetUser"));
        assert!(formatted.contains("user_id=user123"));
        assert!(formatted.contains("tenant_id=tenant456"));
        assert!(formatted.contains("request_id=req789"));
    }

    #[test]
    fn test_error_response_builder() {
        let error = ErrorResponseBuilder::validation("email", "Invalid format");
        assert!(matches!(error, GatewayError::ValidationError(_)));

        let error = ErrorResponseBuilder::authentication("Invalid token");
        assert!(matches!(error, GatewayError::AuthError(_)));

        let error = ErrorResponseBuilder::not_found("User");
        assert!(matches!(error, GatewayError::NotFound(_)));
    }

    #[test]
    fn test_error_logging_levels() {
        let server_error = GatewayError::InternalError("Database failed".to_string());
        assert!(server_error.is_server_error());

        let client_error = GatewayError::ValidationError("Invalid input".to_string());
        assert!(!client_error.is_server_error());
    }
}