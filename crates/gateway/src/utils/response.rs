//! HTTP Response Utilities
//!
//! Standardized response formats for all HTTP endpoints in the gateway.
//! Provides consistent structure for success, error, and paginated responses.

use axum::response::{IntoResponse, Response};
use axum::http::StatusCode;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::collections::HashMap;
use uuid::Uuid;

/// Standard API response wrapper
#[derive(Debug, Clone, Serialize)]
pub struct ApiResponse<T> {
    pub success: bool,
    pub message: String,
    pub data: Option<T>,
    pub errors: Vec<ApiError>,
    pub request_id: String,
    pub timestamp: i64,
}

impl<T> ApiResponse<T> {
    /// Create a successful response
    pub fn success(data: T) -> Self {
        Self {
            success: true,
            message: "Operation completed successfully".to_string(),
            data: Some(data),
            errors: Vec::new(),
            request_id: Uuid::new_v4().to_string(),
            timestamp: chrono::Utc::now().timestamp(),
        }
    }

    /// Create a successful response with custom message
    pub fn success_with_message(data: T, message: impl Into<String>) -> Self {
        Self {
            success: true,
            message: message.into(),
            data: Some(data),
            errors: Vec::new(),
            request_id: Uuid::new_v4().to_string(),
            timestamp: chrono::Utc::now().timestamp(),
        }
    }

    /// Create an error response
    pub fn error(errors: Vec<ApiError>) -> Self {
        Self {
            success: false,
            message: "Operation failed".to_string(),
            data: None,
            errors,
            request_id: Uuid::new_v4().to_string(),
            timestamp: chrono::Utc::now().timestamp(),
        }
    }

    /// Create an error response with custom message
    pub fn error_with_message(errors: Vec<ApiError>, message: impl Into<String>) -> Self {
        Self {
            success: false,
            message: message.into(),
            data: None,
            errors,
            request_id: Uuid::new_v4().to_string(),
            timestamp: chrono::Utc::now().timestamp(),
        }
    }

    /// Create a single error response
    pub fn single_error(error: ApiError) -> Self {
        Self {
            success: false,
            message: "Operation failed".to_string(),
            data: None,
            errors: vec![error],
            request_id: Uuid::new_v4().to_string(),
            timestamp: chrono::Utc::now().timestamp(),
        }
    }
}

impl<T> IntoResponse for ApiResponse<T>
where
    T: Serialize,
{
    fn into_response(self) -> Response {
        let status = if self.success {
            StatusCode::OK
        } else {
            self.errors.first()
                .map(|e| e.status_code)
                .unwrap_or(StatusCode::INTERNAL_SERVER_ERROR)
        };

        (status, axum::Json(self)).into_response()
    }
}

/// Standardized API error structure
#[derive(Debug, Clone, Serialize)]
pub struct ApiError {
    pub code: String,
    pub message: String,
    pub details: Option<HashMap<String, Value>>,
    pub field: Option<String>,
    #[serde(skip)]
    pub status_code: StatusCode,
}

impl ApiError {
    /// Create a new API error
    pub fn new(
        code: impl Into<String>,
        message: impl Into<String>,
        status_code: StatusCode,
    ) -> Self {
        Self {
            code: code.into(),
            message: message.into(),
            details: None,
            field: None,
            status_code,
        }
    }

    /// Create a validation error
    pub fn validation(message: impl Into<String>, field: Option<impl Into<String>>) -> Self {
        Self {
            code: "VALIDATION_ERROR".to_string(),
            message: message.into(),
            details: None,
            field: field.map(|f| f.into()),
            status_code: StatusCode::BAD_REQUEST,
        }
    }

    /// Create a not found error
    pub fn not_found(resource: impl Into<String>) -> Self {
        Self {
            code: "NOT_FOUND".to_string(),
            message: format!("{} not found", resource.into()),
            details: None,
            field: None,
            status_code: StatusCode::NOT_FOUND,
        }
    }

    /// Create an unauthorized error
    pub fn unauthorized(message: impl Into<String>) -> Self {
        Self {
            code: "UNAUTHORIZED".to_string(),
            message: message.into(),
            details: None,
            field: None,
            status_code: StatusCode::UNAUTHORIZED,
        }
    }

    /// Create a forbidden error
    pub fn forbidden(message: impl Into<String>) -> Self {
        Self {
            code: "FORBIDDEN".to_string(),
            message: message.into(),
            details: None,
            field: None,
            status_code: StatusCode::FORBIDDEN,
        }
    }

    /// Create a conflict error
    pub fn conflict(message: impl Into<String>) -> Self {
        Self {
            code: "CONFLICT".to_string(),
            message: message.into(),
            details: None,
            field: None,
            status_code: StatusCode::CONFLICT,
        }
    }

    /// Create an internal server error
    pub fn internal(message: impl Into<String>) -> Self {
        Self {
            code: "INTERNAL_ERROR".to_string(),
            message: message.into(),
            details: None,
            field: None,
            status_code: StatusCode::INTERNAL_SERVER_ERROR,
        }
    }

    /// Add details to the error
    pub fn with_details(mut self, details: HashMap<String, Value>) -> Self {
        self.details = Some(details);
        self
    }

    /// Add a field to the error
    pub fn with_field(mut self, field: impl Into<String>) -> Self {
        self.field = Some(field.into());
        self
    }
}

/// Paginated response wrapper
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PaginatedResponse<T> {
    pub items: Vec<T>,
    pub pagination: PaginationInfo,
}

/// Pagination information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PaginationInfo {
    pub page: u32,
    pub page_size: u32,
    pub total: u64,
    pub total_pages: u32,
    pub has_next: bool,
    pub has_prev: bool,
}

impl PaginationInfo {
    /// Create pagination info
    pub fn new(page: u32, page_size: u32, total: u64) -> Self {
        let total_pages = ((total as f64) / (page_size as f64)).ceil() as u32;
        let total_pages = if total_pages == 0 { 1 } else { total_pages };

        Self {
            page,
            page_size,
            total,
            total_pages,
            has_next: page < total_pages,
            has_prev: page > 1,
        }
    }
}

/// Empty response for successful operations with no data
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EmptyResponse {}

/// Health check response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HealthResponse {
    pub status: String,
    pub version: String,
    pub timestamp: i64,
    pub services: HashMap<String, ServiceHealth>,
}

/// Service health information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServiceHealth {
    pub status: String,
    pub latency_ms: Option<u64>,
    pub error: Option<String>,
}

impl HealthResponse {
    /// Create a health response
    pub fn new() -> Self {
        Self {
            status: "healthy".to_string(),
            version: env!("CARGO_PKG_VERSION").to_string(),
            timestamp: chrono::Utc::now().timestamp(),
            services: HashMap::new(),
        }
    }

    /// Add service health information
    pub fn with_service(mut self, name: impl Into<String>, health: ServiceHealth) -> Self {
        self.services.insert(name.into(), health);
        self
    }

    /// Set overall status to unhealthy
    pub fn unhealthy(mut self) -> Self {
        self.status = "unhealthy".to_string();
        self
    }
}

/// OpenAPI document response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OpenApiDocument {
    pub openapi: String,
    pub info: ApiInfo,
    pub servers: Vec<ServerInfo>,
    pub paths: HashMap<String, Value>,
    pub components: Value,
}

/// API information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApiInfo {
    pub title: String,
    pub description: String,
    pub version: String,
    pub contact: Option<ContactInfo>,
    pub license: Option<LicenseInfo>,
}

/// Contact information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ContactInfo {
    pub name: String,
    pub email: String,
    pub url: Option<String>,
}

/// License information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LicenseInfo {
    pub name: String,
    pub url: Option<String>,
}

/// Server information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServerInfo {
    pub url: String,
    pub description: String,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_api_response_success() {
        let data = "test data".to_string();
        let response = ApiResponse::success(data.clone());

        assert!(response.success);
        assert_eq!(response.data.unwrap(), data);
        assert!(response.errors.is_empty());
    }

    #[test]
    fn test_api_response_error() {
        let error = ApiError::validation("Invalid input", Some("field_name"));
        let response = ApiResponse::single_error(error);

        assert!(!response.success);
        assert!(response.data.is_none());
        assert_eq!(response.errors.len(), 1);
    }

    #[test]
    fn test_pagination_info() {
        let pagination = PaginationInfo::new(2, 10, 25);

        assert_eq!(pagination.page, 2);
        assert_eq!(pagination.page_size, 10);
        assert_eq!(pagination.total, 25);
        assert_eq!(pagination.total_pages, 3);
        assert!(pagination.has_next);
        assert!(pagination.has_prev);
    }

    #[test]
    fn test_api_error_creation() {
        let error = ApiError::not_found("User");

        assert_eq!(error.code, "NOT_FOUND");
        assert_eq!(error.message, "User not found");
        assert_eq!(error.status_code, StatusCode::NOT_FOUND);
    }
}