use smartticket_shared_config::{AppConfig, DatabaseConfig, RedisConfig};
use smartticket_shared_database::{DatabaseConnection, RedisService};
use smartticket_shared_auth::{JwtService, UserData};
use chrono::Utc;
use uuid::Uuid;
use sqlx::{PgPool, Row};
use std::sync::Once;

static INIT: Once = Once::new();

pub struct TestContext {
    pub config: AppConfig,
    pub db: DatabaseConnection,
    pub redis: RedisService,
    pub jwt_service: JwtService,
    pub test_tenant_id: Uuid,
    pub test_user_id: Uuid,
}

impl TestContext {
    pub async fn new() -> anyhow::Result<Self> {
        // Initialize test environment
        INIT.call_once(|| {
            std::env::set_var("ENVIRONMENT", "test");
        });

        let config = create_test_config();
        let db = DatabaseConnection::new(&config.database).await?;
        let redis = RedisService::new(&config.redis)?;
        let jwt_service = JwtService::new(config.auth.clone())?;

        // Run migrations
        db.run_migrations().await?;

        // Create test tenant and user
        let test_tenant_id = create_test_tenant(&db).await?;
        let test_user_id = create_test_user(&db, test_tenant_id).await?;

        Ok(Self {
            config,
            db,
            redis,
            jwt_service,
            test_tenant_id,
            test_user_id,
        })
    }

    pub async fn cleanup(&self) -> anyhow::Result<()> {
        // Clean up test data
        sqlx::query("TRUNCATE TABLE audit_logs CASCADE")
            .execute(self.db.pool())
            .await?;

        sqlx::query("TRUNCATE TABLE knowledge_article_views CASCADE")
            .execute(self.db.pool())
            .await?;

        sqlx::query("TRUNCATE TABLE knowledge_articles CASCADE")
            .execute(self.db.pool())
            .await?;

        sqlx::query("TRUNCATE TABLE sla_metrics CASCADE")
            .execute(self.db.pool())
            .await?;

        sqlx::query("TRUNCATE TABLE tickets CASCADE")
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

    pub fn generate_test_token(&self, user_data: UserData) -> String {
        self.jwt_service
            .generate_token_pair(&self.test_user_id.to_string(), &user_data)
            .unwrap()
            .access_token
    }

    pub fn get_default_user_data(&self) -> UserData {
        UserData {
            email: "test@example.com".to_string(),
            username: "testuser".to_string(),
            full_name: "Test User".to_string(),
            role: "CustomerUser".to_string(),
            tenant_id: self.test_tenant_id.to_string(),
            tenant_name: "Test Tenant".to_string(),
            permissions: vec!["tickets:read".to_string(), "tickets:write".to_string()],
        }
    }

    pub fn get_admin_user_data(&self) -> UserData {
        UserData {
            email: "admin@example.com".to_string(),
            username: "admin".to_string(),
            full_name: "Admin User".to_string(),
            role: "TenantAdmin".to_string(),
            tenant_id: self.test_tenant_id.to_string(),
            tenant_name: "Test Tenant".to_string(),
            permissions: vec![
                "tickets:read".to_string(),
                "tickets:write".to_string(),
                "users:read".to_string(),
                "users:write".to_string(),
                "sla:read".to_string(),
                "sla:write".to_string(),
            ],
        }
    }
}

fn create_test_config() -> AppConfig {
    AppConfig {
        environment: "test".to_string(),
        service_name: "smartticket-test".to_string(),
        version: "0.1.0-test".to_string(),
        database: DatabaseConfig {
            host: std::env::var("TEST_DB_HOST").unwrap_or_else(|_| "localhost".to_string()),
            port: std::env::var("TEST_DB_PORT")
                .unwrap_or_else(|_| "5432".to_string())
                .parse()
                .unwrap_or(5432),
            database_name: std::env::var("TEST_DB_NAME")
                .unwrap_or_else(|_| "smartticket_test".to_string()),
            username: std::env::var("TEST_DB_USER")
                .unwrap_or_else(|_| "postgres".to_string()),
            password: std::env::var("TEST_DB_PASSWORD")
                .unwrap_or_else(|_| "postgres".to_string()),
            ssl_mode: smartticket_shared_config::SslMode::Disable,
            max_connections: 5,
            min_connections: 1,
            connect_timeout: 10,
            idle_timeout: 300,
        },
        redis: RedisConfig {
            host: std::env::var("TEST_REDIS_HOST")
                .unwrap_or_else(|_| "localhost".to_string()),
            port: std::env::var("TEST_REDIS_PORT")
                .unwrap_or_else(|_| "6379".to_string())
                .parse()
                .unwrap_or(6379),
            username: None,
            password: None,
            database: 1, // Use different DB for tests
            max_connections: 3,
            connection_timeout: 3,
            command_timeout: 3,
        },
        ..AppConfig::default()
    }
}

async fn create_test_tenant(db: &DatabaseConnection) -> anyhow::Result<Uuid> {
    let tenant_id = Uuid::new_v4();

    sqlx::query!(
        r#"
        INSERT INTO tenants (id, name, domain, subscription_tier, max_users, data_residency_region)
        VALUES ($1, $2, $3, $4, $5, $6)
        "#,
        tenant_id,
        "Test Tenant",
        "test.smartticket.com",
        "Enterprise" as smartticket_shared_database::models::SubscriptionTier,
        1000i32,
        "EU"
    )
    .execute(db.pool())
    .await?;

    Ok(tenant_id)
}

async fn create_test_user(db: &DatabaseConnection, tenant_id: Uuid) -> anyhow::Result<Uuid> {
    let user_id = Uuid::new_v4();

    sqlx::query!(
        r#"
        INSERT INTO users (id, tenant_id, email, username, full_name, password_hash, role)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        "#,
        user_id,
        tenant_id,
        "test@example.com",
        "testuser",
        "Test User",
        "$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj6hsCm1HLKS", // password: test123
        "CustomerUser" as smartticket_shared_database::models::UserRole
    )
    .execute(db.pool())
    .await?;

    Ok(user_id)
}

#[cfg(test)]
pub mod fixtures {
    use super::*;
    use smartticket_shared_database::models::*;

    pub fn create_test_ticket(
        tenant_id: Uuid,
        contact_id: Uuid,
        created_by_id: Uuid,
    ) -> Ticket {
        Ticket::new(
            tenant_id,
            "TST-2025-00001".to_string(),
            "Test Ticket".to_string(),
            "This is a test ticket for unit testing".to_string(),
            TicketPriority::Normal,
            TicketSeverity::Low,
            contact_id,
            created_by_id,
        )
    }

    pub fn create_test_knowledge_article(
        tenant_id: Uuid,
        author_id: Uuid,
    ) -> KnowledgeArticle {
        KnowledgeArticle::new(
            tenant_id,
            "Test Article".to_string(),
            "This is test content for knowledge base article.".to_string(),
            author_id,
            "en".to_string(),
        )
    }

    pub fn create_test_sla_policy(
        tenant_id: Uuid,
        priority: TicketPriority,
        severity: TicketSeverity,
    ) -> SlaPolicy {
        let now = Utc::now();
        SlaPolicy {
            id: Uuid::new_v4(),
            tenant_id,
            name: "Test SLA Policy".to_string(),
            description: Some("Test SLA policy for unit testing".to_string()),
            priority,
            severity,
            response_time_minutes: 60,
            resolution_time_minutes: 480,
            business_hours_only: true,
            timezone: "UTC".to_string(),
            is_active: true,
            created_at: now,
            updated_at: now,
        }
    }
}

#[cfg(test)]
pub mod assertions {
    use super::*;
    use smartticket_shared_error::SmartTicketError;

    pub fn assert_error_type(error: &SmartTicketError, expected_type: &str) {
        match error {
            SmartTicketError::Validation(_) => assert_eq!(expected_type, "Validation"),
            SmartTicketError::NotFound { .. } => assert_eq!(expected_type, "NotFound"),
            SmartTicketError::PermissionDenied(_) => assert_eq!(expected_type, "PermissionDenied"),
            SmartTicketError::Unauthorized(_) => assert_eq!(expected_type, "Unauthorized"),
            SmartTicketError::Conflict(_) => assert_eq!(expected_type, "Conflict"),
            SmartTicketError::Database(_) => assert_eq!(expected_type, "Database"),
            SmartTicketError::Redis(_) => assert_eq!(expected_type, "Redis"),
            SmartTicketError::Jwt(_) => assert_eq!(expected_type, "Jwt"),
            _ => panic!("Unexpected error type: {:?}", error),
        }
    }

    pub fn assert_tenant_isolation(tenant_id: Uuid, resource_tenant_id: Uuid) {
        assert_eq!(
            tenant_id, resource_tenant_id,
            "Tenant isolation violation: resource belongs to different tenant"
        );
    }
}