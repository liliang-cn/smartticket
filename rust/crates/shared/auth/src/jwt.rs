use base64::engine::general_purpose::URL_SAFE_NO_PAD;
use base64::Engine;
use chrono::{DateTime, Duration, Utc};
use jsonwebtoken::{decode, encode, Algorithm, DecodingKey, EncodingKey, Header, Validation};
use serde::{Deserialize, Serialize};
use smartticket_shared_config::AuthConfig;
use smartticket_shared_error::{Result, SmartTicketError};
use tracing::{error, info};
use uuid::Uuid;

use super::claims::{ApiKeyClaims, RefreshTokenClaims, UserClaims};

pub struct JwtService {
    config: AuthConfig,
    encoding_key: EncodingKey,
    decoding_key: DecodingKey,
    validation: Validation,
}

impl JwtService {
    pub fn new(config: AuthConfig) -> Result<Self> {
        let encoding_key = EncodingKey::from_secret(config.jwt_secret.as_ref());
        let decoding_key = DecodingKey::from_secret(config.jwt_secret.as_ref());

        let mut validation = Validation::new(Algorithm::HS256);
        validation.set_issuer(&[&config.issuer]);
        validation.validate_exp = true;
        // validate_iat is not available in current version, so we skip it

        Ok(Self {
            config,
            encoding_key,
            decoding_key,
            validation,
        })
    }

    pub fn generate_user_token(&self, user_claims: UserClaims) -> Result<String> {
        let header = Header::default();

        encode(&header, &user_claims, &self.encoding_key).map_err(|e| {
            error!("Failed to generate user token: {}", e);
            SmartTicketError::Jwt(e)
        })
    }

    pub fn generate_refresh_token(&self, refresh_claims: RefreshTokenClaims) -> Result<String> {
        let header = Header::default();

        encode(&header, &refresh_claims, &self.encoding_key).map_err(|e| {
            error!("Failed to generate refresh token: {}", e);
            SmartTicketError::Jwt(e)
        })
    }

    pub fn generate_api_key_token(&self, api_key_claims: ApiKeyClaims) -> Result<String> {
        let header = Header::default();

        encode(&header, &api_key_claims, &self.encoding_key).map_err(|e| {
            error!("Failed to generate API key token: {}", e);
            SmartTicketError::Jwt(e)
        })
    }

    pub fn verify_user_token(&self, token: &str) -> Result<UserClaims> {
        let token_data = decode::<UserClaims>(token, &self.decoding_key, &self.validation)
            .map_err(|e| {
                error!("Failed to verify user token: {}", e);
                SmartTicketError::Jwt(e)
            })?;

        if token_data.claims.is_expired() {
            return Err(SmartTicketError::Unauthorized(
                "Token has expired".to_string(),
            ));
        }

        info!(
            "Successfully verified user token for user: {}",
            token_data.claims.sub
        );
        Ok(token_data.claims)
    }

    pub fn verify_refresh_token(&self, token: &str) -> Result<RefreshTokenClaims> {
        let token_data = decode::<RefreshTokenClaims>(token, &self.decoding_key, &self.validation)
            .map_err(|e| {
                error!("Failed to verify refresh token: {}", e);
                SmartTicketError::Jwt(e)
            })?;

        if token_data.claims.type_ != "refresh" {
            return Err(SmartTicketError::Unauthorized(
                "Invalid token type".to_string(),
            ));
        }

        if token_data.claims.is_expired() {
            return Err(SmartTicketError::Unauthorized(
                "Refresh token has expired".to_string(),
            ));
        }

        info!(
            "Successfully verified refresh token for user: {}",
            token_data.claims.sub
        );
        Ok(token_data.claims)
    }

    pub fn verify_api_key_token(&self, token: &str) -> Result<ApiKeyClaims> {
        let token_data = decode::<ApiKeyClaims>(token, &self.decoding_key, &self.validation)
            .map_err(|e| {
                error!("Failed to verify API key token: {}", e);
                SmartTicketError::Jwt(e)
            })?;

        if token_data.claims.is_expired() {
            return Err(SmartTicketError::Unauthorized(
                "API key has expired".to_string(),
            ));
        }

        info!(
            "Successfully verified API key token: {}",
            token_data.claims.sub
        );
        Ok(token_data.claims)
    }

    pub fn generate_token_pair(&self, user_id: &str, user_data: &UserData) -> Result<TokenPair> {
        let now = Utc::now();
        let expires_at = now + Duration::seconds(self.config.jwt_expiration as i64);
        let refresh_expires_at = now + Duration::seconds(self.config.refresh_expiration as i64);
        let jti = Uuid::new_v4().to_string();
        let refresh_jti = Uuid::new_v4().to_string();

        let user_claims = UserClaims::new(
            user_id.to_string(),
            user_data.email.clone(),
            user_data.username.clone(),
            user_data.full_name.clone(),
            user_data.role.clone(),
            user_data.tenant_id.clone(),
            user_data.tenant_name.clone(),
            user_data.permissions.clone(),
            now,
            expires_at,
            self.config.issuer.clone(),
            jti,
        );

        let refresh_claims = RefreshTokenClaims::new(
            user_id.to_string(),
            now,
            refresh_expires_at,
            self.config.issuer.clone(),
            refresh_jti,
        );

        let access_token = self.generate_user_token(user_claims)?;
        let refresh_token = self.generate_refresh_token(refresh_claims)?;

        Ok(TokenPair {
            access_token,
            refresh_token,
            expires_at,
            token_type: "Bearer".to_string(),
        })
    }

    pub fn extract_token_from_header(&self, auth_header: &str) -> Result<String> {
        if !auth_header.starts_with("Bearer ") {
            return Err(SmartTicketError::Unauthorized(
                "Invalid authorization header format".to_string(),
            ));
        }

        let token_part = auth_header.strip_prefix("Bearer ").ok_or_else(|| {
            SmartTicketError::Unauthorized("Invalid authorization header".to_string())
        })?;

        // Ensure there's exactly one token (no extra spaces or content)
        if token_part.trim().is_empty() {
            return Err(SmartTicketError::Unauthorized(
                "Missing token in authorization header".to_string(),
            ));
        }

        // Check if there are extra spaces or content after the token
        if token_part.chars().any(|c| c.is_whitespace()) {
            return Err(SmartTicketError::Unauthorized(
                "Invalid authorization header format - too many parts".to_string(),
            ));
        }

        Ok(token_part.to_string())
    }
}

#[derive(Debug, Clone)]
pub struct UserData {
    pub email: String,
    pub username: String,
    pub full_name: String,
    pub role: String,
    pub tenant_id: String,
    pub tenant_name: String,
    pub permissions: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenPair {
    pub access_token: String,
    pub refresh_token: String,
    pub expires_at: DateTime<Utc>,
    pub token_type: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct TokenInfo {
    pub jti: String,
    pub sub: String,
    pub iat: i64,
    pub exp: i64,
    pub iss: String,
}

impl JwtService {
    pub fn get_token_info(&self, token: &str) -> Result<TokenInfo> {
        let _header = jsonwebtoken::decode_header(token).map_err(SmartTicketError::Jwt)?;

        // We can't decode without knowing the exact structure, so we'll extract basic info
        let token_parts: Vec<&str> = token.split('.').collect();
        if token_parts.len() != 3 {
            return Err(SmartTicketError::Jwt(jsonwebtoken::errors::Error::from(
                jsonwebtoken::errors::ErrorKind::InvalidToken,
            )));
        }

        // Decode payload (middle part)
        let payload = URL_SAFE_NO_PAD.decode(token_parts[1]).map_err(|_| {
            SmartTicketError::Jwt(jsonwebtoken::errors::Error::from(
                jsonwebtoken::errors::ErrorKind::InvalidToken,
            ))
        })?;

        let payload_str = String::from_utf8(payload).map_err(|_| {
            SmartTicketError::Jwt(jsonwebtoken::errors::Error::from(
                jsonwebtoken::errors::ErrorKind::InvalidToken,
            ))
        })?;

        let token_info: TokenInfo = serde_json::from_str(&payload_str).map_err(|e| {
            error!("Failed to parse token payload: {}", e);
            SmartTicketError::Jwt(jsonwebtoken::errors::Error::from(
                jsonwebtoken::errors::ErrorKind::InvalidToken,
            ))
        })?;

        Ok(token_info)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn create_test_config() -> AuthConfig {
        AuthConfig {
            jwt_secret: "test-secret-key-must-be-at-least-32-characters".to_string(),
            jwt_expiration: 3600,
            refresh_expiration: 86400,
            issuer: "test".to_string(),
            bcrypt_cost: 4,
            password_min_length: 8,
            password_require_special: true,
            password_require_numbers: true,
            password_require_uppercase: true,
            rate_limit: Default::default(),
        }
    }

    #[test]
    fn test_generate_and_verify_user_token() {
        let config = create_test_config();
        let jwt_service = JwtService::new(config).unwrap();

        let user_data = UserData {
            email: "test@example.com".to_string(),
            username: "testuser".to_string(),
            full_name: "Test User".to_string(),
            role: "CustomerUser".to_string(),
            tenant_id: "tenant-123".to_string(),
            tenant_name: "Test Tenant".to_string(),
            permissions: vec!["tickets:read".to_string()],
        };

        let token_pair = jwt_service
            .generate_token_pair("user-123", &user_data)
            .unwrap();

        // Verify the access token
        let claims = jwt_service
            .verify_user_token(&token_pair.access_token)
            .unwrap();
        assert_eq!(claims.sub, "user-123");
        assert_eq!(claims.email, "test@example.com");
        assert!(!claims.is_expired());

        // Verify the refresh token
        let refresh_claims = jwt_service
            .verify_refresh_token(&token_pair.refresh_token)
            .unwrap();
        assert_eq!(refresh_claims.sub, "user-123");
        assert_eq!(refresh_claims.type_, "refresh");
        assert!(!refresh_claims.is_expired());
    }

    #[test]
    fn test_expired_token() {
        let config = create_test_config();
        // Create a token that's already expired
        let mut config_expired = config;
        config_expired.jwt_expiration = 1; // Very short expiration to make it expired

        let jwt_service = JwtService::new(config_expired).unwrap();

        let user_data = UserData {
            email: "test@example.com".to_string(),
            username: "testuser".to_string(),
            full_name: "Test User".to_string(),
            role: "CustomerUser".to_string(),
            tenant_id: "tenant-123".to_string(),
            tenant_name: "Test Tenant".to_string(),
            permissions: vec!["tickets:read".to_string()],
        };

        let token_pair = jwt_service
            .generate_token_pair("user-123", &user_data)
            .unwrap();

        // Wait for token to expire
        std::thread::sleep(std::time::Duration::from_secs(2));

        // Should fail to verify expired token
        let result = jwt_service.verify_user_token(&token_pair.access_token);
        assert!(result.is_err());
    }
}
