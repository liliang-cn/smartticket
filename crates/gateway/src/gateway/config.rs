//! Gateway Configuration
//!
//! Configuration management for the HTTP-to-gRPC gateway including
//! authentication, rate limiting, and OpenAPI settings.

use std::time::Duration;
use serde::{Deserialize, Serialize};

/// Gateway Configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GatewayConfig {
    pub http_port: u16,
    pub grpc_endpoint: String,
    pub cors_origins: Vec<String>,
    pub max_request_size: usize,
    pub timeout: Duration,
    pub rate_limit: RateLimitConfig,
    pub auth: AuthConfig,
    pub openapi: OpenApiConfig,
}

impl Default for GatewayConfig {
    fn default() -> Self {
        Self {
            http_port: 3286, // Non-standard port per project rules
            grpc_endpoint: "http://localhost:50051".to_string(),
            cors_origins: vec![
                "http://localhost:3000".parse().unwrap(),
                "http://localhost:3286".parse().unwrap(),
            ],
            max_request_size: 10 * 1024 * 1024, // 10MB
            timeout: Duration::from_secs(30),
            rate_limit: RateLimitConfig::default(),
            auth: AuthConfig::default(),
            openapi: OpenApiConfig::default(),
        }
    }
}

/// Rate limiting configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RateLimitConfig {
    pub requests_per_minute: u32,
    pub burst_size: u32,
}

impl Default for RateLimitConfig {
    fn default() -> Self {
        Self {
            requests_per_minute: 100,
            burst_size: 20,
        }
    }
}

/// Authentication configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthConfig {
    pub jwt_secret: String,
    pub token_expiry: Duration,
    pub refresh_expiry: Duration,
}

impl Default for AuthConfig {
    fn default() -> Self {
        Self {
            jwt_secret: "your-secret-key-here-change-in-production".to_string(),
            token_expiry: Duration::from_secs(3600), // 1 hour
            refresh_expiry: Duration::from_secs(86400), // 24 hours
        }
    }
}

/// OpenAPI documentation configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OpenApiConfig {
    pub auto_refresh: bool,
    pub include_examples: bool,
    pub servers: Vec<ServerInfo>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServerInfo {
    pub url: String,
    pub description: String,
}

impl Default for OpenApiConfig {
    fn default() -> Self {
        Self {
            auto_refresh: true,
            include_examples: true,
            servers: vec![
                ServerInfo {
                    url: "http://localhost:3286".to_string(),
                    description: "Development server".to_string(),
                },
                ServerInfo {
                    url: "https://staging-api.smartticket.com/v1".to_string(),
                    description: "Staging server".to_string(),
                },
                ServerInfo {
                    url: "https://api.smartticket.com/v1".to_string(),
                    description: "Production server".to_string(),
                },
            ],
        }
    }
}

/// Load gateway configuration from environment variables
pub fn load_from_env() -> Result<GatewayConfig, Box<dyn std::error::Error + Send + Sync>> {
    let mut config = GatewayConfig::default();

    // Load HTTP port
    if let Ok(port) = std::env::var("HTTP_PORT") {
        config.http_port = port.parse()?;
    }

    // Load gRPC endpoint
    if let Ok(endpoint) = std::env::var("GRPC_ENDPOINT") {
        config.grpc_endpoint = endpoint;
    }

    // Load JWT secret
    if let Ok(secret) = std::env::var("JWT_SECRET") {
        config.auth.jwt_secret = secret;
    }

    // Load rate limits
    if let Ok(rpm) = std::env::var("RATE_LIMIT_RPM") {
        config.rate_limit.requests_per_minute = rpm.parse()?;
    }

    if let Ok(burst) = std::env::var("RATE_LIMIT_BURST") {
        config.rate_limit.burst_size = burst.parse()?;
    }

    // Load request size limit
    if let Ok(size) = std::env::var("MAX_REQUEST_SIZE") {
        config.max_request_size = size.parse()?;
    }

    // Load timeout
    if let Ok(timeout_secs) = std::env::var("REQUEST_TIMEOUT") {
        config.timeout = Duration::from_secs(timeout_secs.parse()?);
    }

    Ok(config)
}

/// Validate gateway configuration
pub fn validate_config(config: &GatewayConfig) -> Result<(), String> {
    // Validate port range
    if config.http_port < 1024 || config.http_port > 65535 {
        return Err("HTTP port must be between 1024 and 65535".to_string());
    }

    // Validate rate limits
    if config.rate_limit.requests_per_minute == 0 {
        return Err("Rate limit requests per minute must be greater than 0".to_string());
    }

    if config.rate_limit.burst_size == 0 {
        return Err("Rate limit burst size must be greater than 0".to_string());
    }

    // Validate JWT secret
    if config.auth.jwt_secret.len() < 32 {
        return Err("JWT secret must be at least 32 characters long".to_string());
    }

    // Validate request size
    if config.max_request_size == 0 {
        return Err("Maximum request size must be greater than 0".to_string());
    }

    // Validate timeout
    if config.timeout.as_secs() == 0 {
        return Err("Request timeout must be greater than 0 seconds".to_string());
    }

    // Validate gRPC endpoint
    if !config.grpc_endpoint.starts_with("http://") && !config.grpc_endpoint.starts_with("https://") {
        return Err("gRPC endpoint must start with http:// or https://".to_string());
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_default_config() {
        let config = GatewayConfig::default();
        assert_eq!(config.http_port, 3286);
        assert_eq!(config.grpc_endpoint, "http://localhost:50051");
        assert_eq!(config.max_request_size, 10 * 1024 * 1024);
        assert_eq!(config.rate_limit.requests_per_minute, 100);
        assert_eq!(config.auth.token_expiry, Duration::from_secs(3600));
    }

    #[test]
    fn test_env_config_loading() {
        // Set some environment variables
        std::env::set_var("HTTP_PORT", "3287");
        std::env::set_var("JWT_SECRET", "test-jwt-secret-key-that-is-long-enough");
        std::env::set_var("RATE_LIMIT_RPM", "200");

        let config = load_from_env().unwrap();
        assert_eq!(config.http_port, 3287);
        assert_eq!(config.rate_limit.requests_per_minute, 200);
        assert_eq!(config.auth.jwt_secret, "test-jwt-secret-key-that-is-long-enough");
    }

    #[test]
    fn test_config_validation() {
        let mut config = GatewayConfig::default();

        // Test valid config
        assert!(validate_config(&config).is_ok());

        // Test invalid port
        config.http_port = 80; // Common port - should fail
        assert!(validate_config(&config).is_err());

        // Test empty JWT secret
        config.http_port = 3286; // Reset to valid
        config.auth.jwt_secret = "short".to_string();
        assert!(validate_config(&config).is_err());

        // Test zero rate limit
        config.auth.jwt_secret = "valid-jwt-secret-key-that-is-long-enough".to_string();
        config.rate_limit.requests_per_minute = 0;
        assert!(validate_config(&config).is_err());
    }
}