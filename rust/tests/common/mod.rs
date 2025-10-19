use std::sync::Arc;
use sqlx::{PgPool, Row};
use uuid::Uuid;
use smartticket_shared_database::DatabaseConnection;
use smartticket_shared_auth::{AuthService, TenantContext, UserData};
use smartticket_shared_error::Result;
use smartticket_shared_database::models::*;

pub mod fixtures;
pub mod assertions;

/// Test context for managing test environment
pub struct TestContext {
    pub db: Arc<DatabaseConnection>,
    pub jwt_service: Arc<AuthService>,
    pub test_tenant_id: Uuid,
    pub test_user_id: Uuid,
}

impl TestContext {
    /// Create a new test context
    pub async fn new() -> Result<Self> {
        // Initialize database connection
        let database_url = std::env::var("TEST_DATABASE_URL")
            .unwrap_or_else(|_| "postgresql://postgres:password@localhost:5433/smartticket_test".to_string());

        let pool = PgPool::connect(&database_url).await?;
        let db = Arc::new(DatabaseConnection::new(pool));

        // Initialize JWT service
        let jwt_service = Arc::new(AuthService::new(
            "test-secret-key-for-testing-only".to_string(),
            3600, // 1 hour
        ));

        // Create test tenant and user
        let test_tenant_id = Uuid::new_v4();
        let test_user_id = Uuid::new_v4();

        let mut context = Self {
            db,
            jwt_service,
            test_tenant_id,
            test_user_id,
        };

        context.setup_test_tenant().await?;
        context.setup_test_user().await?;

        Ok(context)
    }

    /// Setup test tenant
    async fn setup_test_tenant(&mut self) -> Result<()> {
        sqlx::query!(
            "INSERT INTO tenants (id, name, domain, subscription_tier, max_users, data_residency_region, is_active)
             VALUES ($1, $2, $3, $4, $5, $6, $7)",
            self.test_tenant_id,
            "Test Tenant",
            "test.smartticket.com",
            SubscriptionTier::Enterprise,
            100i32,
            "EU",
            true
        )
        .execute(self.db.pool())
        .await?;

        Ok(())
    }

    /// Setup test user
    async fn setup_test_user(&mut self) -> Result<()> {
        sqlx::query!(
            "INSERT INTO users (id, tenant_id, email, username, full_name, password_hash, role, is_active)
             VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
            self.test_user_id,
            self.test_tenant_id,
            "test@test.smartticket.com",
            "testuser",
            "Test User",
            "$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj6hsCm1HLKS",
            UserRole::TenantAdmin,
            true
        )
        .execute(self.db.pool())
        .await?;

        Ok(())
    }

    /// Generate a test JWT token
    pub fn generate_test_token(&self, user_data: UserData) -> String {
        self.jwt_service.generate_token(
            &user_data.user_id,
            &user_data.email,
            &user_data.username,
            &user_data.full_name,
            &user_data.tenant_id,
            &user_data.role,
        ).unwrap_or_else(|_| "test-token".to_string())
    }

    /// Clean up test data
    pub async fn cleanup(&mut self) -> Result<()> {
        // Clean up in reverse order of creation
        sqlx::query("DELETE FROM audit_logs WHERE tenant_id = $1")
            .bind(self.test_tenant_id)
            .execute(self.db.pool())
            .await?;

        sqlx::query("DELETE FROM knowledge_articles WHERE tenant_id = $1")
            .bind(self.test_tenant_id)
            .execute(self.db.pool())
            .await?;

        sqlx::query("DELETE FROM tickets WHERE tenant_id = $1")
            .bind(self.test_tenant_id)
            .execute(self.db.pool())
            .await?;

        sqlx::query("DELETE FROM users WHERE tenant_id = $1")
            .bind(self.test_tenant_id)
            .execute(self.db.pool())
            .await?;

        sqlx::query("DELETE FROM tenants WHERE id = $1")
            .bind(self.test_tenant_id)
            .execute(self.db.pool())
            .await?;

        Ok(())
    }
}