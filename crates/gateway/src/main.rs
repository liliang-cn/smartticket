//! SmartTicket HTTP Gateway
//!
//! HTTP-to-gRPC reverse proxy gateway that exposes all SmartTicket gRPC services
//! as REST APIs with automatic OpenAPI documentation generation.

#![recursion_limit = "256"]

mod gateway;
mod utils;
mod services;
mod grpc_client;
mod proto;

use std::sync::Arc;
use tracing::{info, error, Level};
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

use gateway::{GatewayConfig, HttpToGrpcGateway, gateway_router::create_gateway_router};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
    // Initialize logging
    init_logging();

    info!("🚀 Starting SmartTicket HTTP Gateway");

    // Load configuration
    let config = load_configuration().await?;
    info!("📋 Configuration loaded successfully");
    info!("🌐 HTTP port: {}", config.http_port);
    info!("🔗 gRPC endpoint: {}", config.grpc_endpoint);

    // Create and initialize gateway
    let mut gateway = HttpToGrpcGateway::new(config.clone());

    // Initialize gRPC connections
    info!("🔌 Initializing gRPC service connections...");
    gateway.initialize_connections().await?;
    info!("✅ gRPC connections established");

    // Create router
    let app = create_gateway_router(gateway, config.clone());

    // Bind listener
    let addr = format!("0.0.0.0:{}", config.http_port);
    let listener = tokio::net::TcpListener::bind(&addr).await?;
    info!("🌍 HTTP server listening on: {}", addr);

    // Print startup information
    print_startup_info(&config);

    // Start server
    info!("🎯 SmartTicket Gateway is ready to accept requests");
    axum::serve(listener, app)
        .await
        .map_err(|e| {
            error!("❌ Server error: {}", e);
            e
        })?;

    Ok(())
}

/// Initialize logging with appropriate level and formatting
fn init_logging() {
    let log_level = std::env::var("RUST_LOG")
        .unwrap_or_else(|_| "info".to_string())
        .parse::<Level>()
        .unwrap_or(Level::INFO);

    tracing_subscriber::registry()
        .with(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| tracing_subscriber::EnvFilter::new("smartticket_gateway=info"))
        )
        .with(tracing_subscriber::fmt::layer())
        .init();

    info!("📝 Logging initialized with level: {:?}", log_level);
}

/// Load configuration from environment variables and defaults
async fn load_configuration() -> Result<GatewayConfig, Box<dyn std::error::Error + Send + Sync>> {
    // Try to load from environment first
    let config = gateway::config::load_from_env()
        .unwrap_or_else(|_| {
            info!("⚠️  Using default configuration (could not load from environment)");
            GatewayConfig::default()
        });

    // Validate configuration
    gateway::config::validate_config(&config)
        .map_err(|e| {
            error!("❌ Configuration validation failed: {}", e);
            e
        })?;

    info!("✅ Configuration validated successfully");
    Ok(config)
}

/// Print startup information and available endpoints
fn print_startup_info(config: &GatewayConfig) {
    println!();
    println!("🎉 SmartTicket API Gateway is running!");
    println!();
    println!("📊 Service Information:");
    println!("   • Version: {}", env!("CARGO_PKG_VERSION"));
    println!("   • HTTP Port: {}", config.http_port);
    println!("   • gRPC Endpoint: {}", config.grpc_endpoint);
    println!("   • Max Request Size: {} MB", config.max_request_size / 1024 / 1024);
    println!("   • Request Timeout: {} seconds", config.timeout.as_secs());
    println!("   • Rate Limit: {} req/min (burst: {})",
        config.rate_limit.requests_per_minute,
        config.rate_limit.burst_size);
    println!();
    println!("🌐 Available Endpoints:");
    println!("   • Health Check: http://localhost:{}/health", config.http_port);
    println!("   • API Documentation: http://localhost:{}/docs", config.http_port);
    println!("   • OpenAPI YAML: http://localhost:{}/openapi.yaml", config.http_port);
    println!("   • OpenAPI JSON: http://localhost:{}/openapi.json", config.http_port);
    println!("   • Swagger UI: http://localhost:{}/swagger-ui.html", config.http_port);
    println!("   • Root Info: http://localhost:{}/", config.http_port);
    println!();
    println!("🔐 Authentication:");
    println!("   • Login: POST http://localhost:{}/auth/v1/login", config.http_port);
    println!("   • Refresh: POST http://localhost:{}/auth/v1/refresh", config.http_port);
    println!("   • Logout: POST http://localhost:{}/auth/v1/logout", config.http_port);
    println!();
    println!("👥 User Management:");
    println!("   • List Users: GET http://localhost:{}/v1/users", config.http_port);
    println!("   • Create User: POST http://localhost:{}/v1/users", config.http_port);
    println!("   • Get User: GET http://localhost:{}/v1/users/{{id}}", config.http_port);
    println!("   • Update User: PUT http://localhost:{}/v1/users/{{id}}", config.http_port);
    println!("   • Delete User: DELETE http://localhost:{}/v1/users/{{id}}", config.http_port);
    println!();
    println!("🎫 Ticket Management:");
    println!("   • List Tickets: GET http://localhost:{}/v1/tickets", config.http_port);
    println!("   • Create Ticket: POST http://localhost:{}/v1/tickets", config.http_port);
    println!();
    println!("📚 Knowledge Base:");
    println!("   • List Articles: GET http://localhost:{}/v1/knowledge/articles", config.http_port);
    println!();
    println!("📋 SLA Management:");
    println!("   • List Policies: GET http://localhost:{}/v1/sla/policies", config.http_port);
    println!();
    println!("🔑 Role Management:");
    println!("   • List Roles: GET http://localhost:{}/v1/roles", config.http_port);
    println!();
    println!("💡 Tips:");
    println!("   • Visit http://localhost:{}/docs for interactive API testing", config.http_port);
    println!("   • Use 'Authorization: Bearer <token>' header for authenticated requests");
    println!("   • Use 'X-Tenant-ID: <tenant_id>' header for multi-tenant requests");
    println!("   • Check the health endpoint to verify service status");
    println!();
    println!("Press Ctrl+C to stop the server");
    println!();
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_configuration_loading() {
        // Test that default configuration can be created
        let config = GatewayConfig::default();
        assert_eq!(config.http_port, 3286);
        assert_eq!(config.grpc_endpoint, "http://localhost:50051");
    }

    #[tokio::test]
    async fn test_gateway_creation() {
        let config = GatewayConfig::default();
        let gateway = HttpToGrpcGateway::new(config);

        // Test that gateway can be created without panicking
        assert_eq!(gateway.config.http_port, 3286);
    }

    #[test]
    fn test_logging_initialization() {
        // This test ensures logging doesn't panic
        init_logging();
        assert!(true); // If we reach here, logging was initialized successfully
    }
}