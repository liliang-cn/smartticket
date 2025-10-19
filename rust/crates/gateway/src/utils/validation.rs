//! Request Validation Utilities
//!
//! Provides validation helpers for HTTP requests including parameter validation,
//! query parameter parsing, and request body validation.

use axum::extract::{Query, State};
use axum::http::HeaderMap;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use uuid::Uuid;

/// Validation error types
#[derive(Debug, Clone)]
pub enum ValidationError {
    MissingRequiredField(String),
    InvalidFormat(String, String),
    OutOfRange(String, String),
    InvalidLength(String, usize, usize),
    InvalidEnumValue(String, Vec<String>),
    InvalidUuid(String),
    InvalidEmail(String),
    InvalidTenantId(String),
}

impl std::fmt::Display for ValidationError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            ValidationError::MissingRequiredField(field) => write!(f, "Missing required field: {}", field),
            ValidationError::InvalidFormat(field, expected) => write!(f, "Invalid format for field '{}'. Expected: {}", field, expected),
            ValidationError::OutOfRange(field, range) => write!(f, "Field '{}' is out of range: {}", field, range),
            ValidationError::InvalidLength(field, min, max) => write!(f, "Field '{}' length must be between {} and {}", field, min, max),
            ValidationError::InvalidEnumValue(field, values) => write!(f, "Field '{}' must be one of: {}", field, values.join(", ")),
            ValidationError::InvalidUuid(field) => write!(f, "Field '{}' must be a valid UUID", field),
            ValidationError::InvalidEmail(field) => write!(f, "Field '{}' must be a valid email address", field),
            ValidationError::InvalidTenantId(field) => write!(f, "Field '{}' must be a valid tenant identifier", field),
        }
    }
}

impl ValidationError {
    /// Convert validation error to API error
    pub fn to_api_error(self) -> crate::utils::response::ApiError {
        match self {
            ValidationError::MissingRequiredField(field) => {
                crate::utils::response::ApiError::validation(
                    format!("Required field '{}' is missing", field),
                    Some(field),
                )
            }
            ValidationError::InvalidFormat(field, expected) => {
                crate::utils::response::ApiError::validation(
                    format!("Field '{}' has invalid format. Expected: {}", field, expected),
                    Some(field),
                )
            }
            ValidationError::OutOfRange(field, range) => {
                crate::utils::response::ApiError::validation(
                    format!("Field '{}' is out of valid range: {}", field, range),
                    Some(field),
                )
            }
            ValidationError::InvalidLength(field, min, max) => {
                crate::utils::response::ApiError::validation(
                    format!("Field '{}' length must be between {} and {}", field, min, max),
                    Some(field),
                )
            }
            ValidationError::InvalidEnumValue(field, valid_values) => {
                crate::utils::response::ApiError::validation(
                    format!("Field '{}' must be one of: {}", field, valid_values.join(", ")),
                    Some(field),
                )
            }
            ValidationError::InvalidUuid(field) => {
                crate::utils::response::ApiError::validation(
                    format!("Field '{}' must be a valid UUID", field),
                    Some(field),
                )
            }
            ValidationError::InvalidEmail(field) => {
                crate::utils::response::ApiError::validation(
                    format!("Field '{}' must be a valid email address", field),
                    Some(field),
                )
            }
            ValidationError::InvalidTenantId(field) => {
                crate::utils::response::ApiError::validation(
                    format!("Field '{}' must be a valid tenant identifier", field),
                    Some(field),
                )
            }
        }
    }
}

/// Common validation functions
pub struct Validator;

impl Validator {
    /// Validate required string field
    pub fn required_string(value: &Option<String>, field_name: &str) -> Result<String, ValidationError> {
        match value {
            Some(s) if !s.trim().is_empty() => Ok(s.clone()),
            _ => Err(ValidationError::MissingRequiredField(field_name.to_string())),
        }
    }

    /// Validate optional string field
    pub fn optional_string(value: &Option<String>) -> Option<String> {
        value.as_ref().and_then(|s| {
            let trimmed = s.trim();
            if trimmed.is_empty() { None } else { Some(trimmed.to_string()) }
        })
    }

    /// Validate UUID field
    pub fn uuid(value: &str, field_name: &str) -> Result<Uuid, ValidationError> {
        Uuid::parse_str(value).map_err(|_| ValidationError::InvalidUuid(field_name.to_string()))
    }

    /// Validate email format
    pub fn email(value: &str, field_name: &str) -> Result<String, ValidationError> {
        if Self::is_valid_email(value) {
            Ok(value.to_string())
        } else {
            Err(ValidationError::InvalidEmail(field_name.to_string()))
        }
    }

    /// Validate tenant ID format
    pub fn tenant_id(value: &str, field_name: &str) -> Result<String, ValidationError> {
        if Self::is_valid_tenant_id(value) {
            Ok(value.to_string())
        } else {
            Err(ValidationError::InvalidTenantId(field_name.to_string()))
        }
    }

    /// Validate string length
    pub fn string_length(value: &str, field_name: &str, min: usize, max: usize) -> Result<String, ValidationError> {
        if value.len() < min || value.len() > max {
            Err(ValidationError::InvalidLength(field_name.to_string(), min, max))
        } else {
            Ok(value.to_string())
        }
    }

    /// Validate numeric range
    pub fn numeric_range<T>(value: T, field_name: &str, min: T, max: T) -> Result<T, ValidationError>
    where
        T: PartialOrd + std::fmt::Display,
    {
        if value < min || value > max {
            Err(ValidationError::OutOfRange(
                field_name.to_string(),
                format!("{} to {}", min, max),
            ))
        } else {
            Ok(value)
        }
    }

    /// Validate enum value
    pub fn enum_value<T: ToString>(
        value: &str,
        field_name: &str,
        valid_values: &[T],
    ) -> Result<String, ValidationError> {
        let valid_strings: Vec<String> = valid_values.iter().map(|v| v.to_string()).collect();
        if valid_strings.contains(&value.to_string()) {
            Ok(value.to_string())
        } else {
            Err(ValidationError::InvalidEnumValue(field_name.to_string(), valid_strings))
        }
    }

    /// Validate pagination parameters
    pub fn pagination(page: Option<u32>, page_size: Option<u32>) -> (u32, u32) {
        let page = page.unwrap_or(1).max(1);
        let page_size = page_size.unwrap_or(20).clamp(1, 100);
        (page, page_size)
    }

    /// Check if email format is valid
    fn is_valid_email(email: &str) -> bool {
        // Simple email validation - can be enhanced with regex if needed
        email.contains('@') && email.contains('.') && email.len() > 5 && email.len() < 254
    }

    /// Check if tenant ID format is valid
    fn is_valid_tenant_id(tenant_id: &str) -> bool {
        // Tenant ID should be alphanumeric with possible hyphens/underscores
        tenant_id.len() >= 3
            && tenant_id.len() <= 50
            && tenant_id.chars().all(|c| c.is_alphanumeric() || c == '-' || c == '_')
    }
}

/// Query parameter validation structures
#[derive(Debug, Deserialize)]
pub struct PaginationQuery {
    pub page: Option<u32>,
    pub page_size: Option<u32>,
}

#[derive(Debug, Deserialize)]
pub struct SortQuery {
    pub sort_by: Option<String>,
    pub sort_order: Option<SortOrder>,
}

#[derive(Debug, Deserialize, Clone, Copy)]
#[serde(rename_all = "lowercase")]
pub enum SortOrder {
    Asc,
    Desc,
}

#[derive(Debug, Deserialize)]
pub struct FilterQuery {
    pub filter: Option<String>,
    pub search: Option<String>,
}

/// Combined query parameters
#[derive(Debug, Deserialize)]
pub struct ListQueryParams {
    #[serde(flatten)]
    pub pagination: PaginationQuery,
    #[serde(flatten)]
    pub sort: SortQuery,
    #[serde(flatten)]
    pub filter: FilterQuery,
}

impl ListQueryParams {
    /// Validate and extract pagination parameters
    pub fn pagination(&self) -> (u32, u32) {
        Validator::pagination(self.pagination.page, self.pagination.page_size)
    }

    /// Get sort parameters with defaults
    pub fn sort_params(&self, default_field: &str) -> (String, SortOrder) {
        let field = self.sort.sort_by.as_ref().map(|s| s.as_str()).unwrap_or(default_field).to_string();
        let order = self.sort.sort_order.unwrap_or(SortOrder::Asc);
        (field, order)
    }
}

/// Header validation utilities
pub struct HeaderValidator;

impl HeaderValidator {
    /// Extract and validate tenant ID from headers
    pub fn tenant_id(headers: &HeaderMap) -> Result<String, ValidationError> {
        let tenant_id = headers
            .get("x-tenant-id")
            .and_then(|h| h.to_str().ok())
            .ok_or_else(|| ValidationError::MissingRequiredField("x-tenant-id".to_string()))?;

        Validator::tenant_id(tenant_id, "x-tenant-id")
    }

    /// Extract and validate authorization token
    pub fn auth_token(headers: &HeaderMap) -> Result<String, ValidationError> {
        let auth_header = headers
            .get("authorization")
            .and_then(|h| h.to_str().ok())
            .ok_or_else(|| ValidationError::MissingRequiredField("authorization".to_string()))?;

        if !auth_header.starts_with("Bearer ") {
            return Err(ValidationError::InvalidFormat(
                "authorization".to_string(),
                "Bearer <token>".to_string(),
            ));
        }

        Ok(auth_header[7..].to_string())
    }

    /// Extract request ID from headers
    pub fn request_id(headers: &HeaderMap) -> Option<String> {
        headers
            .get("x-request-id")
            .and_then(|h| h.to_str().ok())
            .map(|s| s.to_string())
    }

    /// Extract user ID from headers
    pub fn user_id(headers: &HeaderMap) -> Result<Uuid, ValidationError> {
        let user_id = headers
            .get("x-user-id")
            .and_then(|h| h.to_str().ok())
            .ok_or_else(|| ValidationError::MissingRequiredField("x-user-id".to_string()))?;

        Validator::uuid(user_id, "x-user-id")
    }

    /// Extract user roles from headers
    pub fn user_roles(headers: &HeaderMap) -> Vec<String> {
        headers
            .get("x-user-roles")
            .and_then(|h| h.to_str().ok())
            .map(|roles| {
                roles
                    .split(',')
                    .map(|role| role.trim().to_string())
                    .filter(|role| !role.is_empty())
                    .collect()
            })
            .unwrap_or_default()
    }
}

/// Request validation context
#[derive(Debug)]
pub struct ValidationContext {
    pub tenant_id: String,
    pub user_id: Uuid,
    pub user_roles: Vec<String>,
    pub request_id: Option<String>,
}

impl ValidationContext {
    /// Create validation context from headers
    pub fn from_headers(headers: &HeaderMap) -> Result<Self, Vec<ValidationError>> {
        let mut errors = Vec::new();

        let tenant_id = match HeaderValidator::tenant_id(headers) {
            Ok(id) => id,
            Err(e) => {
                errors.push(e);
                String::new()
            }
        };

        let user_id = match HeaderValidator::user_id(headers) {
            Ok(id) => id,
            Err(e) => {
                errors.push(e);
                Uuid::new_v4() // Placeholder
            }
        };

        let user_roles = HeaderValidator::user_roles(headers);
        let request_id = HeaderValidator::request_id(headers);

        if errors.is_empty() {
            Ok(Self {
                tenant_id,
                user_id,
                user_roles,
                request_id,
            })
        } else {
            Err(errors)
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_required_string_validation() {
        assert!(Validator::required_string(&Some("test".to_string()), "field").is_ok());
        assert!(Validator::required_string(&Some("   ".to_string()), "field").is_err());
        assert!(Validator::required_string(&None, "field").is_err());
    }

    #[test]
    fn test_uuid_validation() {
        let valid_uuid = "550e8400-e29b-41d4-a716-446655440000";
        let invalid_uuid = "invalid-uuid";

        assert!(Validator::uuid(valid_uuid, "id").is_ok());
        assert!(Validator::uuid(invalid_uuid, "id").is_err());
    }

    #[test]
    fn test_email_validation() {
        assert!(Validator::email("test@example.com", "email").is_ok());
        assert!(Validator::email("invalid-email", "email").is_err());
    }

    #[test]
    fn test_pagination_validation() {
        let (page, size) = Validator::pagination(Some(0), Some(200));
        assert_eq!(page, 1); // Should be corrected to minimum 1
        assert_eq!(size, 100); // Should be clamped to maximum 100
    }

    #[test]
    fn test_enum_validation() {
        let valid_values = vec!["active", "inactive", "pending"];

        assert!(Validator::enum_value("active", "status", &valid_values).is_ok());
        assert!(Validator::enum_value("invalid", "status", &valid_values).is_err());
    }
}