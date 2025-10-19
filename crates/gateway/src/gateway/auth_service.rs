//! Authentication Service
//!
//! Handles user authentication, token management, and authorization logic.

use axum::{
    extract::{State, Request},
    http::{StatusCode, HeaderMap},
    response::{Json, Response},
    middleware::Next,
};
use serde::{Deserialize, Serialize};
use uuid::Uuid;
use chrono::{Duration, Utc};
use bcrypt;
use std::sync::Arc;

use crate::gateway::middleware::{TokenManager, AuthContext, Claims};
use crate::gateway::{GatewayConfig, error::GatewayError};
use crate::utils::response::{ApiResponse, ApiError};

/// Authentication service
#[derive(Clone)]
pub struct AuthService {
    token_manager: Arc<TokenManager>,
    config: Arc<GatewayConfig>,
}

/// Login request payload
#[derive(Debug, Deserialize, Serialize)]
pub struct LoginRequest {
    pub email: String,
    pub password: String,
    pub tenant_id: Option<String>,
    pub remember_me: Option<bool>,
}

/// Login response payload
#[derive(Debug, Deserialize, Serialize)]
pub struct LoginResponse {
    pub access_token: String,
    pub refresh_token: String,
    pub token_type: String,
    pub expires_in: i64,
    pub user_info: UserInfo,
    pub session_id: String,
}

/// User information response
#[derive(Debug, Deserialize, Serialize)]
pub struct UserInfo {
    pub id: String,
    pub email: String,
    pub name: String,
    pub tenant_id: String,
    pub roles: Vec<String>,
    pub permissions: Vec<String>,
    pub last_login: Option<String>,
    pub created_at: String,
}

/// Refresh token request
#[derive(Debug, Deserialize, Serialize)]
pub struct RefreshTokenRequest {
    pub refresh_token: String,
}

/// Token refresh response
#[derive(Debug, Deserialize, Serialize)]
pub struct TokenRefreshResponse {
    pub access_token: String,
    pub refresh_token: Option<String>,
    pub token_type: String,
    pub expires_in: i64,
}

/// Logout request
#[derive(Debug, Deserialize, Serialize)]
pub struct LogoutRequest {
    pub refresh_token: Option<String>,
    pub logout_all_sessions: Option<bool>,
}

/// User registration request
#[derive(Debug, Deserialize, Serialize)]
pub struct RegisterRequest {
    pub email: String,
    pub password: String,
    pub name: String,
    pub tenant_id: String,
    pub invite_code: Option<String>,
}

/// Password change request
#[derive(Debug, Deserialize, Serialize)]
pub struct ChangePasswordRequest {
    pub current_password: String,
    pub new_password: String,
    pub confirm_password: String,
}

/// Forgot password request
#[derive(Debug, Deserialize, Serialize)]
pub struct ForgotPasswordRequest {
    pub email: String,
    pub tenant_id: String,
}

/// Reset password request
#[derive(Debug, Deserialize, Serialize)]
pub struct ResetPasswordRequest {
    pub token: String,
    pub new_password: String,
    pub confirm_password: String,
}

impl AuthService {
    /// Create new authentication service
    pub fn new(config: Arc<GatewayConfig>) -> Self {
        let token_manager = Arc::new(TokenManager::new(config.clone()));

        Self {
            token_manager,
            config,
        }
    }

    /// Authenticate user and generate tokens
    pub async fn login(&self, request: LoginRequest) -> Result<LoginResponse, GatewayError> {
        // Validate input
        if request.email.is_empty() || request.password.is_empty() {
            return Err(GatewayError::ValidationError("Email and password are required".to_string()));
        }

        // TODO: Implement actual user authentication against database
        // For now, we'll simulate authentication
        let user_info = self.authenticate_user(&request.email, &request.password, request.tenant_id.as_deref()).await?;

        // Generate session ID
        let session_id = Uuid::new_v4().to_string();

        // Generate tokens
        let access_token = self.token_manager.generate_access_token(
            &user_info.id,
            &user_info.tenant_id,
            user_info.roles.clone(),
            user_info.permissions.clone(),
            Some(session_id.clone()),
        )?;

        let refresh_token = self.token_manager.generate_refresh_token(
            &user_info.id,
            &user_info.tenant_id,
            Some(session_id.clone()),
        )?;

        // Calculate token expiry
        let expires_in = 3600; // 1 hour in seconds

        tracing::info!(
            user_id = %user_info.id,
            tenant_id = %user_info.tenant_id,
            session_id = %session_id,
            "User login successful"
        );

        Ok(LoginResponse {
            access_token,
            refresh_token,
            token_type: "Bearer".to_string(),
            expires_in,
            user_info,
            session_id,
        })
    }

    /// Refresh access token using refresh token
    pub async fn refresh_token(&self, request: RefreshTokenRequest) -> Result<TokenRefreshResponse, GatewayError> {
        // Validate refresh token
        let claims = self.token_manager.validate_token(&request.refresh_token)?;

        // Ensure it's a refresh token
        if claims.token_type != "refresh" {
            return Err(GatewayError::AuthError("Invalid token type for refresh".to_string()));
        }

        // TODO: Check if session is still valid and not revoked
        // For now, we'll generate new tokens

        // Get user information (simplified - in real implementation, fetch from database)
        let user_info = self.get_user_info(&claims.sub, &claims.tenant_id).await?;

        // Generate new access token
        let access_token = self.token_manager.generate_access_token(
            &user_info.id,
            &user_info.tenant_id,
            user_info.roles.clone(),
            user_info.permissions.clone(),
            claims.session_id.clone(),
        )?;

        // Optionally generate new refresh token (token rotation)
        let refresh_token = if should_rotate_refresh_token() {
            Some(self.token_manager.generate_refresh_token(
                &user_info.id,
                &user_info.tenant_id,
                claims.session_id.clone(),
            )?)
        } else {
            None
        };

        tracing::info!(
            user_id = %user_info.id,
            tenant_id = %user_info.tenant_id,
            "Token refresh successful"
        );

        Ok(TokenRefreshResponse {
            access_token,
            refresh_token,
            token_type: "Bearer".to_string(),
            expires_in: 3600,
        })
    }

    /// Logout user and invalidate tokens
    pub async fn logout(&self, request: LogoutRequest, auth_context: &AuthContext) -> Result<(), GatewayError> {
        tracing::info!(
            user_id = %auth_context.user_id,
            tenant_id = %auth_context.tenant_id,
            session_id = ?auth_context.session_id,
            "User logout"
        );

        // TODO: Implement token invalidation
        // - Add tokens to blacklist/revocation list
        // - Invalidate session in database
        // - Clear user cache if needed

        if request.logout_all_sessions.unwrap_or(false) {
            // TODO: Invalidate all user sessions
            tracing::info!(user_id = %auth_context.user_id, "Invalidating all user sessions");
        } else {
            // TODO: Invalidate current session only
            tracing::info!(session_id = ?auth_context.session_id, "Invalidating current session");
        }

        Ok(())
    }

    /// Register new user
    pub async fn register(&self, request: RegisterRequest) -> Result<UserInfo, GatewayError> {
        // Validate input
        self.validate_registration_request(&request)?;

        // Check if user already exists
        if self.user_exists(&request.email, &request.tenant_id).await? {
            return Err(GatewayError::Conflict("User already exists".to_string()));
        }

        // Hash password
        let password_hash = bcrypt::hash(&request.password, bcrypt::DEFAULT_COST)
            .map_err(|e| GatewayError::InternalError(format!("Password hashing failed: {}", e)))?;

        // TODO: Create user in database
        let user_info = self.create_user(&request, &password_hash).await?;

        tracing::info!(
            user_id = %user_info.id,
            tenant_id = %user_info.tenant_id,
            email = %request.email,
            "User registration successful"
        );

        Ok(user_info)
    }

    /// Change user password
    pub async fn change_password(
        &self,
        request: ChangePasswordRequest,
        auth_context: &AuthContext,
    ) -> Result<(), GatewayError> {
        // Validate input
        if request.new_password != request.confirm_password {
            return Err(GatewayError::ValidationError("Password confirmation does not match".to_string()));
        }

        if request.new_password.len() < 8 {
            return Err(GatewayError::ValidationError("Password must be at least 8 characters long".to_string()));
        }

        // TODO: Verify current password against database
        // TODO: Update password in database
        // TODO: Invalidate all other sessions (optional security measure)

        tracing::info!(
            user_id = %auth_context.user_id,
            "Password change successful"
        );

        Ok(())
    }

    /// Handle forgot password request
    pub async fn forgot_password(&self, request: ForgotPasswordRequest) -> Result<(), GatewayError> {
        // TODO: Check if user exists
        // TODO: Generate password reset token
        // TODO: Send reset email
        // TODO: Store reset token with expiry

        tracing::info!(
            email = %request.email,
            tenant_id = %request.tenant_id,
            "Password reset requested"
        );

        // Always return success to prevent user enumeration
        Ok(())
    }

    /// Handle password reset
    pub async fn reset_password(&self, request: ResetPasswordRequest) -> Result<(), GatewayError> {
        // Validate input
        if request.new_password != request.confirm_password {
            return Err(GatewayError::ValidationError("Password confirmation does not match".to_string()));
        }

        // TODO: Validate reset token
        // TODO: Check token expiry
        // TODO: Update password in database
        // TODO: Invalidate token

        tracing::info!(
            "Password reset completed (token validation omitted for security)"
        );

        Ok(())
    }

    // Private helper methods

    /// Authenticate user credentials
    async fn authenticate_user(
        &self,
        email: &str,
        password: &str,
        tenant_id: Option<&str>,
    ) -> Result<UserInfo, GatewayError> {
        // TODO: Implement actual authentication against database
        // For now, simulate authentication with mock user

        // Mock authentication - in real implementation, this would query the database
        if email == "admin@smartticket.com" && password == "admin123" {
            let tenant_id = tenant_id.unwrap_or("default-tenant").to_string();

            Ok(UserInfo {
                id: "user-123".to_string(),
                email: email.to_string(),
                name: "Administrator".to_string(),
                tenant_id: tenant_id.clone(),
                roles: vec!["admin".to_string(), "user".to_string()],
                permissions: vec![
                    "users.read".to_string(),
                    "users.write".to_string(),
                    "tickets.read".to_string(),
                    "tickets.write".to_string(),
                    "admin.panel".to_string(),
                ],
                last_login: Some(Utc::now().to_rfc3339()),
                created_at: "2023-01-01T00:00:00Z".to_string(),
            })
        } else if email == "user@smartticket.com" && password == "user123" {
            let tenant_id = tenant_id.unwrap_or("default-tenant").to_string();

            Ok(UserInfo {
                id: "user-456".to_string(),
                email: email.to_string(),
                name: "Test User".to_string(),
                tenant_id: tenant_id.clone(),
                roles: vec!["user".to_string()],
                permissions: vec![
                    "tickets.read".to_string(),
                    "tickets.write".to_string(),
                ],
                last_login: Some(Utc::now().to_rfc3339()),
                created_at: "2023-01-01T00:00:00Z".to_string(),
            })
        } else {
            Err(GatewayError::AuthError("Invalid credentials".to_string()))
        }
    }

    /// Get user information from database
    async fn get_user_info(&self, user_id: &str, tenant_id: &str) -> Result<UserInfo, GatewayError> {
        // TODO: Implement actual database query
        // For now, return mock data
        Ok(UserInfo {
            id: user_id.to_string(),
            email: "user@example.com".to_string(),
            name: "Test User".to_string(),
            tenant_id: tenant_id.to_string(),
            roles: vec!["user".to_string()],
            permissions: vec!["tickets.read".to_string(), "tickets.write".to_string()],
            last_login: Some(Utc::now().to_rfc3339()),
            created_at: "2023-01-01T00:00:00Z".to_string(),
        })
    }

    /// Check if user already exists
    async fn user_exists(&self, email: &str, tenant_id: &str) -> Result<bool, GatewayError> {
        // TODO: Implement actual database query
        // For now, return false
        Ok(false)
    }

    /// Create new user in database
    async fn create_user(&self, request: &RegisterRequest, password_hash: &str) -> Result<UserInfo, GatewayError> {
        // TODO: Implement actual database insertion
        // For now, return mock user data
        Ok(UserInfo {
            id: Uuid::new_v4().to_string(),
            email: request.email.clone(),
            name: request.name.clone(),
            tenant_id: request.tenant_id.clone(),
            roles: vec!["user".to_string()],
            permissions: vec!["tickets.read".to_string(), "tickets.write".to_string()],
            last_login: None,
            created_at: Utc::now().to_rfc3339(),
        })
    }

    /// Validate registration request
    fn validate_registration_request(&self, request: &RegisterRequest) -> Result<(), GatewayError> {
        if request.email.is_empty() {
            return Err(GatewayError::ValidationError("Email is required".to_string()));
        }

        if !request.email.contains('@') {
            return Err(GatewayError::ValidationError("Invalid email format".to_string()));
        }

        if request.password.len() < 8 {
            return Err(GatewayError::ValidationError("Password must be at least 8 characters long".to_string()));
        }

        if request.name.is_empty() {
            return Err(GatewayError::ValidationError("Name is required".to_string()));
        }

        if request.tenant_id.is_empty() {
            return Err(GatewayError::ValidationError("Tenant ID is required".to_string()));
        }

        Ok(())
    }
}

/// Check if refresh token should be rotated
fn should_rotate_refresh_token() -> bool {
    // TODO: Implement token rotation policy
    // For security, it's good to rotate refresh tokens periodically
    true
}

/// Extract authentication context from request
pub fn extract_auth_context(request: &Request) -> Result<AuthContext, GatewayError> {
    request.extensions().get::<AuthContext>()
        .cloned()
        .ok_or_else(|| GatewayError::AuthError("Authentication context not found".to_string()))
}

/// Middleware to ensure user is authenticated
pub async fn require_auth_middleware(
    request: Request,
    next: Next,
) -> Result<Response, GatewayError> {
    let auth_context = extract_auth_context(&request)?;

    tracing::debug!(
        user_id = %auth_context.user_id,
        tenant_id = %auth_context.tenant_id,
        "Authentication middleware passed"
    );

    Ok(next.run(request).await)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_validate_registration_request() {
        let config = Arc::new(GatewayConfig::default());
        let auth_service = AuthService::new(config);

        let valid_request = RegisterRequest {
            email: "test@example.com".to_string(),
            password: "password123".to_string(),
            name: "Test User".to_string(),
            tenant_id: "test-tenant".to_string(),
            invite_code: None,
        };

        assert!(auth_service.validate_registration_request(&valid_request).is_ok());

        let invalid_request = RegisterRequest {
            email: "invalid-email".to_string(),
            password: "123".to_string(),
            name: "".to_string(),
            tenant_id: "".to_string(),
            invite_code: None,
        };

        assert!(auth_service.validate_registration_request(&invalid_request).is_err());
    }

    #[test]
    fn test_should_rotate_refresh_token() {
        assert!(should_rotate_refresh_token());
    }
}