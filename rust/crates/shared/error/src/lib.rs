use std::collections::HashMap;
use tonic::{Code, Status};
use uuid::Uuid;

#[derive(Debug, thiserror::Error)]
pub enum SmartTicketError {
    #[error("Validation error: {0}")]
    Validation(String),

    #[error("Not found: {entity} with id {id}")]
    NotFound { entity: String, id: String },

    #[error("Permission denied: {0}")]
    PermissionDenied(String),

    #[error("Unauthorized: {0}")]
    Unauthorized(String),

    #[error("Conflict: {0}")]
    Conflict(String),

    #[error("Rate limited: {0}")]
    RateLimited(String),

    #[error("Database error: {0}")]
    Database(#[from] sqlx::Error),

    #[error("Redis error: {0}")]
    Redis(#[from] redis::RedisError),

    #[error("Serialization error: {0}")]
    Serialization(#[from] serde_json::Error),

    #[error("JWT error: {0}")]
    Jwt(#[from] jsonwebtoken::errors::Error),

    #[error("Password hashing error: {0}")]
    PasswordHashing(String),

    #[error("Configuration error: {0}")]
    Configuration(String),

    #[error("External service error: {service} - {message}")]
    ExternalService { service: String, message: String },

    #[error("Internal server error: {0}")]
    Internal(String),

    #[error("Tenant not found: {0}")]
    TenantNotFound(Uuid),

    #[error("Access denied for tenant {tenant_id}: {reason}")]
    TenantAccessDenied { tenant_id: Uuid, reason: String },

    #[error("SLA violation: {0}")]
    SlaViolation(String),

    #[error("Ticket transition not allowed: from {from} to {to}")]
    InvalidTicketTransition { from: String, to: String },
}

impl SmartTicketError {
    pub fn not_found(entity: &str, id: &str) -> Self {
        Self::NotFound {
            entity: entity.to_string(),
            id: id.to_string(),
        }
    }

    pub fn permission_denied(reason: &str) -> Self {
        Self::PermissionDenied(reason.to_string())
    }

    pub fn validation(message: &str) -> Self {
        Self::Validation(message.to_string())
    }

    pub fn conflict(message: &str) -> Self {
        Self::Conflict(message.to_string())
    }

    pub fn external_service(service: &str, message: &str) -> Self {
        Self::ExternalService {
            service: service.to_string(),
            message: message.to_string(),
        }
    }

    pub fn error_code(&self) -> &'static str {
        match self {
            SmartTicketError::Validation(_) => "VALIDATION_ERROR",
            SmartTicketError::NotFound { .. } => "NOT_FOUND",
            SmartTicketError::PermissionDenied(_) => "PERMISSION_DENIED",
            SmartTicketError::Unauthorized(_) => "UNAUTHORIZED",
            SmartTicketError::Conflict(_) => "CONFLICT",
            SmartTicketError::RateLimited(_) => "RATE_LIMITED",
            SmartTicketError::Database(_) => "DATABASE_ERROR",
            SmartTicketError::Redis(_) => "REDIS_ERROR",
            SmartTicketError::Serialization(_) => "SERIALIZATION_ERROR",
            SmartTicketError::Jwt(_) => "JWT_ERROR",
            SmartTicketError::PasswordHashing(_) => "PASSWORD_HASHING_ERROR",
            SmartTicketError::Configuration(_) => "CONFIGURATION_ERROR",
            SmartTicketError::ExternalService { .. } => "EXTERNAL_SERVICE_ERROR",
            SmartTicketError::Internal(_) => "INTERNAL_ERROR",
            SmartTicketError::TenantNotFound(_) => "TENANT_NOT_FOUND",
            SmartTicketError::TenantAccessDenied { .. } => "TENANT_ACCESS_DENIED",
            SmartTicketError::SlaViolation(_) => "SLA_VIOLATION",
            SmartTicketError::InvalidTicketTransition { .. } => "INVALID_TICKET_TRANSITION",
        }
    }

    pub fn http_status(&self) -> u16 {
        match self {
            SmartTicketError::Validation(_) => 400,
            SmartTicketError::NotFound { .. } => 404,
            SmartTicketError::PermissionDenied(_) => 403,
            SmartTicketError::Unauthorized(_) => 401,
            SmartTicketError::Conflict(_) => 409,
            SmartTicketError::RateLimited(_) => 429,
            SmartTicketError::TenantNotFound(_) => 404,
            SmartTicketError::TenantAccessDenied { .. } => 403,
            SmartTicketError::Database(_) => 500,
            SmartTicketError::Redis(_) => 500,
            SmartTicketError::Serialization(_) => 500,
            SmartTicketError::Jwt(_) => 401,
            SmartTicketError::PasswordHashing(_) => 500,
            SmartTicketError::Configuration(_) => 500,
            SmartTicketError::ExternalService { .. } => 502,
            SmartTicketError::Internal(_) => 500,
            SmartTicketError::SlaViolation(_) => 400,
            SmartTicketError::InvalidTicketTransition { .. } => 400,
        }
    }

    pub fn grpc_status(&self) -> (Code, String) {
        match self {
            SmartTicketError::Validation(msg) => (Code::InvalidArgument, msg.clone()),
            SmartTicketError::NotFound { entity, id } => {
                (Code::NotFound, format!("{} {} not found", entity, id))
            }
            SmartTicketError::PermissionDenied(msg) => (Code::PermissionDenied, msg.clone()),
            SmartTicketError::Unauthorized(msg) => (Code::Unauthenticated, msg.clone()),
            SmartTicketError::Conflict(msg) => (Code::AlreadyExists, msg.clone()),
            SmartTicketError::RateLimited(msg) => (Code::ResourceExhausted, msg.clone()),
            SmartTicketError::Database(_) => (Code::Internal, "Database error".to_string()),
            SmartTicketError::Redis(_) => (Code::Internal, "Redis error".to_string()),
            SmartTicketError::Serialization(_) => {
                (Code::Internal, "Serialization error".to_string())
            }
            SmartTicketError::Jwt(_) => (Code::Unauthenticated, "JWT error".to_string()),
            SmartTicketError::PasswordHashing(_) => {
                (Code::Internal, "Password hashing error".to_string())
            }
            SmartTicketError::Configuration(msg) => (Code::Internal, msg.clone()),
            SmartTicketError::ExternalService { service, message } => (
                Code::Unavailable,
                format!("External service {}: {}", service, message),
            ),
            SmartTicketError::Internal(msg) => (Code::Internal, msg.clone()),
            SmartTicketError::TenantNotFound(_) => (Code::NotFound, "Tenant not found".to_string()),
            SmartTicketError::TenantAccessDenied { tenant_id, reason } => (
                Code::PermissionDenied,
                format!("Access denied for tenant {}: {}", tenant_id, reason),
            ),
            SmartTicketError::SlaViolation(msg) => (Code::FailedPrecondition, msg.clone()),
            SmartTicketError::InvalidTicketTransition { from, to } => (
                Code::FailedPrecondition,
                format!("Invalid transition from {} to {}", from, to),
            ),
        }
    }
}

impl From<String> for SmartTicketError {
    fn from(s: String) -> Self {
        SmartTicketError::Internal(s)
    }
}

impl From<&str> for SmartTicketError {
    fn from(s: &str) -> Self {
        SmartTicketError::Internal(s.to_string())
    }
}

impl From<SmartTicketError> for Status {
    fn from(err: SmartTicketError) -> Self {
        let (code, message) = err.grpc_status();
        Status::new(code, message)
    }
}

#[derive(Debug, serde::Serialize)]
pub struct ErrorDetail {
    pub code: String,
    pub message: String,
    pub details: Option<HashMap<String, serde_json::Value>>,
    pub request_id: Option<String>,
}

impl ErrorDetail {
    pub fn new(error: &SmartTicketError) -> Self {
        Self {
            code: error.error_code().to_string(),
            message: error.to_string(),
            details: None,
            request_id: None,
        }
    }

    pub fn with_details(mut self, details: HashMap<String, serde_json::Value>) -> Self {
        self.details = Some(details);
        self
    }

    pub fn with_request_id(mut self, request_id: String) -> Self {
        self.request_id = Some(request_id);
        self
    }
}

pub type Result<T> = std::result::Result<T, SmartTicketError>;
