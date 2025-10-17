//! SmartTicket Core Service
//!
//! This crate contains the core business logic for the SmartTicket platform,
//! including ticket management, knowledge management, and SLA functionality.

pub mod models;
pub mod services;

pub use models::ticket::*;
pub use services::ticket_service::TicketService;

use smartticket_shared_config::AppConfig;
use smartticket_shared_database::DatabaseConnection;
use smartticket_shared_error::Result;
use tracing::info;

/// Core service instance
pub struct CoreService {
    pub ticket_service: TicketService,
    db_connection: DatabaseConnection,
}

impl CoreService {
    /// Create a new core service instance
    pub async fn new(config: &AppConfig) -> Result<Self> {
        info!("Initializing Core Service");

        // Initialize database connection
        let db_connection = DatabaseConnection::new(&config.database).await?;

        // Run migrations
        db_connection.run_migrations().await?;

        // Create ticket service
        let ticket_service = TicketService::new(db_connection.pool().clone());

        info!("Core Service initialized successfully");

        Ok(Self {
            ticket_service,
            db_connection,
        })
    }

    /// Get database connection reference
    pub fn get_db_connection(&self) -> &DatabaseConnection {
        &self.db_connection
    }

    /// Health check for the core service
    pub async fn health_check(&self) -> Result<()> {
        self.db_connection.health_check().await?;
        info!("Core service health check passed");
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    #[tokio::test]
    async fn test_core_service_creation() {
        // This test would require a real database connection
        // In a real implementation, you would use a test database
        // or mock the database connection
    }
}
