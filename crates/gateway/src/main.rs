//! SmartTicket Gateway Main Entry Point
//!
//! This is the main entry point for the SmartTicket gRPC Gateway service.
//! It initializes and starts all gRPC services.

use anyhow::Result;
use smartticket_core::services::ticket_service::TicketService;
use smartticket_shared_config::AppConfig;
use smartticket_shared_database::AuthService;
use std::sync::Arc;
use tracing::{error, info};
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

use smartticket_gateway::{http_server::HttpServer, GatewayServer};

#[tokio::main]
async fn main() -> Result<()> {
    // Initialize tracing
    tracing_subscriber::registry()
        .with(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| "smartticket_gateway=debug,tower_http=debug,axum=debug".into()),
        )
        .with(tracing_subscriber::fmt::layer())
        .init();

    info!("🚀 Starting SmartTicket Gateway Server...");

    // Load configuration
    let config = match AppConfig::load() {
        Ok(config) => {
            info!("✅ Configuration loaded successfully");
            config
        }
        Err(e) => {
            error!("❌ Failed to load configuration: {}", e);
            return Err(anyhow::anyhow!("Failed to load configuration: {}", e));
        }
    };

    info!("📋 Configuration:");
    info!("  - gRPC Port: {}", config.server.grpc_port);
    info!("  - HTTP Port: {}", config.server.http_port);
    info!(
        "  - Database: {}:{}",
        config.database.host, config.database.port
    );
    info!("  - Redis: {}:{}", config.redis.host, config.redis.port);

    // Initialize auth service
    let auth_service = Arc::new(AuthService::new(
        "your-secret-key-here",
        "smartticket".to_string(),
        "smartticket-client".to_string(),
    ));
    info!("✅ Auth service initialized");

    // Initialize database connection pool
    let database_url = format!(
        "postgresql://{}:{}@{}:{}/{}",
        config.database.username,
        config.database.password,
        config.database.host,
        config.database.port,
        config.database.database_name
    );

    info!("🔗 Connecting to database...");
    info!(
        "Database URL: postgresql://{}:***@{}:{}/{}",
        config.database.username,
        config.database.host,
        config.database.port,
        config.database.database_name
    );
    let db_pool = sqlx::postgres::PgPool::connect(&database_url)
        .await
        .map_err(|e| {
            error!("❌ Failed to connect to database: {}", e);
            anyhow::anyhow!("Database connection failed: {}", e)
        })?;
    info!("✅ Database connection established");

    // Create shared database pool
    let db_pool_shared = Arc::new(db_pool);
    info!("✅ Database connection pool created");

    // Initialize ticket service with database pool (clone the pool from Arc)
    let ticket_service = Arc::new(TicketService::new((*db_pool_shared).clone()));
    info!("✅ Ticket service initialized with database connection");

    // Create and start gateway server
    let gateway_server = GatewayServer::new(
        config.clone(),
        ticket_service.clone(),
        auth_service.clone(),
        db_pool_shared.clone(),
    );
    info!("🚀 Starting SmartTicket Gateway Server...");

    info!("🌟 SmartTicket Gateway is configured successfully!");
    info!("📍 gRPC listening on: 0.0.0.0:{}", config.server.grpc_port);
    info!("🔧 Available services:");
    info!("  - TicketService (✅ database operations)");
    info!("  - UserService (✅ database operations)");
    info!("  - KnowledgeService (✅ database operations - all 12 interfaces)");
    info!("  - TenantService (✅ database operations with real billing calculations)");
    info!("  - RolePermissionService (⚠️  mock implementation - needs database rewrite)");

    // Start both gRPC and HTTP servers concurrently
    info!("🚀 Starting servers...");

    let grpc_port = config.server.grpc_port;
    let http_port = config.server.http_port;

    // Spawn gRPC server
    let grpc_server = tokio::spawn(async move {
        if let Err(e) = gateway_server.start().await {
            error!("❌ Failed to start gRPC server: {}", e);
        }
    });

    // Spawn HTTP server
    let http_server = tokio::spawn(async move {
        if let Err(e) = HttpServer::start(http_port).await {
            error!("❌ Failed to start HTTP server: {}", e);
        }
    });

    info!("✅ SmartTicket Gateway servers started successfully");
    info!("📍 gRPC listening on: 0.0.0.0:{}", grpc_port);
    info!("🌐 HTTP listening on: http://0.0.0.0:{}", http_port);
    info!(
        "📚 API Documentation available at: http://0.0.0.0:{}/docs",
        http_port
    );

    // Wait for both servers
    tokio::select! {
        _ = grpc_server => {
            error!("gRPC server stopped");
        }
        _ = http_server => {
            error!("HTTP server stopped");
        }
    }

    Ok(())
}
