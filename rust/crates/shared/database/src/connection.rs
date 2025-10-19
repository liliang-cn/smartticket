use smartticket_shared_config::{database::SslMode, DatabaseConfig};
use smartticket_shared_error::{Result, SmartTicketError};
use sqlx::{postgres::PgPoolOptions, PgPool, Row};
use std::time::Duration;
#[allow(unused_imports)]
use tracing::{info, warn};

pub struct DatabaseConnection {
    pool: PgPool,
}

impl DatabaseConnection {
    pub async fn new(config: &DatabaseConfig) -> Result<Self> {
        let database_url = format!(
            "postgresql://{}:{}@{}:{}/{}?sslmode={}",
            config.username,
            config.password,
            config.host,
            config.port,
            config.database_name,
            match config.ssl_mode {
                SslMode::Disable => "disable",
                SslMode::Prefer => "prefer",
                SslMode::Require => "require",
            }
        );

        info!(
            "Connecting to database: {}:{}/{}",
            config.host, config.port, config.database_name
        );

        let pool = PgPoolOptions::new()
            .max_connections(config.max_connections)
            .min_connections(config.min_connections)
            .acquire_timeout(Duration::from_secs(config.connect_timeout))
            .idle_timeout(Duration::from_secs(config.idle_timeout))
            .connect(&database_url)
            .await
            .map_err(|e| {
                SmartTicketError::Configuration(format!("Failed to connect to database: {}", e))
            })?;

        // Test the connection
        sqlx::query("SELECT 1")
            .fetch_one(&pool)
            .await
            .map_err(SmartTicketError::Database)?;

        info!("Successfully connected to database");

        Ok(Self { pool })
    }

    pub fn pool(&self) -> &PgPool {
        &self.pool
    }

    pub async fn health_check(&self) -> Result<()> {
        let result = sqlx::query("SELECT version()")
            .fetch_one(&self.pool)
            .await
            .map_err(SmartTicketError::Database)?;

        let version: String = result.get("version");
        info!("Database health check - version: {}", version);

        Ok(())
    }

    pub async fn close(&self) {
        info!("Closing database connections");
        self.pool.close().await;
    }

    // Migration helper
    pub async fn run_migrations(&self) -> Result<()> {
        info!("Running database migrations");
        sqlx::migrate!("./migrations")
            .run(&self.pool)
            .await
            .map_err(|e| SmartTicketError::Database(e.into()))?;
        info!("Database migrations completed successfully");
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_database_connection() {
        let config = DatabaseConfig::default();
        // This test requires a running PostgreSQL instance
        // In a real project, you'd use testcontainers or a test database

        // Skip test if no database is available
        match DatabaseConnection::new(&config).await {
            Ok(conn) => {
                assert!(conn.health_check().await.is_ok());
                conn.close().await;
            }
            Err(e) => {
                warn!("Skipping database test - no database available: {}", e);
            }
        }
    }
}
