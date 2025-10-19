use serde::{Deserialize, Serialize};
use validator::Validate;

#[derive(Debug, Clone, Serialize, Deserialize, Validate)]
pub struct AuthConfig {
    #[validate(length(min = 32))]
    pub jwt_secret: String,
    pub jwt_expiration: u64,
    pub refresh_expiration: u64,
    pub issuer: String,
    pub bcrypt_cost: u32,
    pub password_min_length: usize,
    pub password_require_special: bool,
    pub password_require_numbers: bool,
    pub password_require_uppercase: bool,
    pub rate_limit: RateLimitConfig,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RateLimitConfig {
    pub enabled: bool,
    pub requests_per_minute: u32,
    pub burst_size: u32,
}

impl Default for RateLimitConfig {
    fn default() -> Self {
        Self {
            enabled: true,
            requests_per_minute: 60,
            burst_size: 10,
        }
    }
}

impl Default for AuthConfig {
    fn default() -> Self {
        Self {
            jwt_secret: "your-super-secret-jwt-key-change-this-in-production".to_string(),
            jwt_expiration: 3600,          // 1 hour
            refresh_expiration: 86400 * 7, // 7 days
            issuer: "smartticket".to_string(),
            bcrypt_cost: 12,
            password_min_length: 8,
            password_require_special: true,
            password_require_numbers: true,
            password_require_uppercase: true,
            rate_limit: RateLimitConfig {
                enabled: true,
                requests_per_minute: 60,
                burst_size: 10,
            },
        }
    }
}
