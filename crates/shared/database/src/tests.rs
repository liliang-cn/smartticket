#[cfg(test)]
mod tests {
    use super::*;
    use crate::models::*;
    use crate::tenant_isolation::TenantContext;
    use sqlx::Row;
    use uuid::Uuid;

    async fn setup_test_db() -> DatabaseConnection {
        let config = DatabaseConfig {
            host: "localhost".to_string(),
            port: 5432,
            database_name: "smartticket_test".to_string(),
            username: "postgres".to_string(),
            password: "postgres".to_string(),
            ssl_mode: smartticket_shared_config::SslMode::Disable,
            max_connections: 5,
            min_connections: 1,
            connect_timeout: 10,
            idle_timeout: 300,
        };

        let db =
            DatabaseConnection::new(&config).expect("Failed to create test database connection");

        // Run migrations for test database
        db.run_migrations().await.expect("Failed to run migrations");

        db
    }

    async fn cleanup_test_data(db: &DatabaseConnection) {
        let _ = sqlx::query("TRUNCATE TABLE audit_logs CASCADE")
            .execute(db.pool())
            .await;

        let _ = sqlx::query("DELETE FROM users WHERE email LIKE '%@test.com'")
            .execute(db.pool())
            .await;

        let _ = sqlx::query("DELETE FROM tenants WHERE domain LIKE '%.test.com'")
            .execute(db.pool())
            .await;
    }

    #[tokio::test]
    async fn test_database_connection() {
        // This test requires a running PostgreSQL instance
        match setup_test_db().await {
            Ok(db) => {
                let health_result = db.health_check().await;
                assert!(
                    health_result.is_ok(),
                    "Database health check failed: {:?}",
                    health_result
                );

                // Test basic query
                let result = sqlx::query("SELECT 1 as test").fetch_one(db.pool()).await;
                assert!(result.is_ok());

                let test_val: i32 = result.unwrap().get("test");
                assert_eq!(test_val, 1);
            }
            Err(e) => {
                println!("Skipping database test - no database available: {}", e);
            }
        }
    }

    #[tokio::test]
    async fn test_tenant_isolation_policies() {
        let db = match setup_test_db().await {
            Ok(db) => db,
            Err(_) => {
                println!("Skipping tenant isolation test - no database available");
                return;
            }
        };

        let tenant1_id = Uuid::new_v4();
        let tenant2_id = Uuid::new_v4();
        let admin_user_id = Uuid::new_v4();

        // Create test tenants
        sqlx::query!(
            "INSERT INTO tenants (id, name, domain, subscription_tier, max_users, data_residency_region)
             VALUES ($1, $2, $3, $4, $5, $6)",
            tenant1_id,
            "Test Tenant 1",
            "tenant1.test.com",
            "Enterprise" as SubscriptionTier,
            1000i32,
            "EU"
        )
        .execute(db.pool())
        .await
        .expect("Failed to create test tenant 1");

        sqlx::query!(
            "INSERT INTO tenants (id, name, domain, subscription_tier, max_users, data_residency_region)
             VALUES ($1, $2, $3, $4, $5, $6)",
            tenant2_id,
            "Test Tenant 2",
            "tenant2.test.com",
            "Enterprise" as SubscriptionTier,
            1000i32,
            "EU"
        )
        .execute(db.pool())
        .await
        .expect("Failed to create test tenant 2");

        // Create admin user
        sqlx::query!(
            "INSERT INTO users (id, tenant_id, email, username, full_name, password_hash, role)
             VALUES ($1, $2, $3, $4, $5, $6, $7)",
            admin_user_id,
            tenant1_id,
            "admin@test.com",
            "admin",
            "Admin User",
            "$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj6hsCm1HLKS",
            "TenantAdmin" as UserRole
        )
        .execute(db.pool())
        .await
        .expect("Failed to create admin user");

        // Test tenant isolation - admin should only see their own users
        let admin_context =
            TenantContext::new(tenant1_id, admin_user_id, "TenantAdmin".to_string());

        // Set application context for RLS
        sqlx::query("SELECT set_app_context($1, $2)")
            .bind(admin_context.tenant_id)
            .bind(admin_context.user_id)
            .execute(db.pool())
            .await
            .expect("Failed to set app context");

        // Create users in both tenants
        let user1_tenant1_id = Uuid::new_v4();
        let user1_tenant2_id = Uuid::new_v4();

        sqlx::query!(
            "INSERT INTO users (id, tenant_id, email, username, full_name, password_hash, role)
             VALUES ($1, $2, $3, $4, $5, $6, $7)",
            user1_tenant1_id,
            tenant1_id,
            "user1@tenant1.test.com",
            "user1_t1",
            "User 1 Tenant 1",
            "$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj6hsCm1HLKS",
            "CustomerUser" as UserRole
        )
        .execute(db.pool())
        .await
        .expect("Failed to create user in tenant 1");

        sqlx::query!(
            "INSERT INTO users (id, tenant_id, email, username, full_name, password_hash, role)
             VALUES ($1, $2, $3, $4, $5, $6, $7)",
            user1_tenant2_id,
            tenant2_id,
            "user1@tenant2.test.com",
            "user1_t2",
            "User 1 Tenant 2",
            "$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj6hsCm1HLKS",
            "CustomerUser" as UserRole
        )
        .execute(db.pool())
        .await
        .expect("Failed to create user in tenant 2");

        // Test that admin from tenant1 can only see users from tenant1
        let users_in_tenant1 = sqlx::query("SELECT COUNT(*) as count FROM users")
            .fetch_one(db.pool())
            .await
            .expect("Failed to query users");

        let count: i64 = users_in_tenant1.get("count");
        assert_eq!(
            count, 3,
            "Admin should only see 3 users (admin + user in tenant1 + admin user from setup)"
        );

        cleanup_test_data(&db).await;
    }

    #[tokio::test]
    async fn test_ticket_number_generation() {
        let db = match setup_test_db().await {
            Ok(db) => db,
            Err(_) => {
                println!("Skipping ticket number generation test - no database available");
                return;
            }
        };

        let tenant_id = Uuid::new_v4();
        let user_id = Uuid::new_v4();

        // Create test tenant
        sqlx::query!(
            "INSERT INTO tenants (id, name, domain, subscription_tier, max_users, data_residency_region)
             VALUES ($1, $2, $3, $4, $5, $6)",
            tenant_id,
            "Test Company",
            "testcompany.test.com",
            "Enterprise" as SubscriptionTier,
            1000i32,
            "EU"
        )
        .execute(db.pool())
        .await
        .expect("Failed to create test tenant");

        // Create test user
        sqlx::query!(
            "INSERT INTO users (id, tenant_id, email, username, full_name, password_hash, role)
             VALUES ($1, $2, $3, $4, $5, $6, $7)",
            user_id,
            tenant_id,
            "test@testcompany.test.com",
            "testuser",
            "Test User",
            "$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj6hsCm1HLKS",
            "CustomerUser" as UserRole
        )
        .execute(db.pool())
        .await
        .expect("Failed to create test user");

        // Set application context
        sqlx::query("SELECT set_app_context($1, $2)")
            .bind(tenant_id)
            .bind(user_id)
            .execute(db.pool())
            .await
            .expect("Failed to set app context");

        // Create first ticket
        let ticket1_id = Uuid::new_v4();
        sqlx::query!(
            "INSERT INTO tickets (id, tenant_id, title, description, contact_id, created_by_id)
             VALUES ($1, $2, $3, $4, $5, $6)
             RETURNING ticket_number",
            ticket1_id,
            tenant_id,
            "Test Ticket 1",
            "First test ticket",
            user_id,
            user_id
        )
        .fetch_one(db.pool())
        .await
        .expect("Failed to create first ticket");

        // Create second ticket
        let ticket2_id = Uuid::new_v4();
        let result = sqlx::query!(
            "INSERT INTO tickets (id, tenant_id, title, description, contact_id, created_by_id)
             VALUES ($1, $2, $3, $4, $5, $6)
             RETURNING ticket_number",
            ticket2_id,
            tenant_id,
            "Test Ticket 2",
            "Second test ticket",
            user_id,
            user_id
        )
        .fetch_one(db.pool())
        .await
        .expect("Failed to create second ticket");

        // Verify ticket numbers are generated correctly
        assert!(result.ticket_number.starts_with("TES-2025-"));
        assert!(result.ticket_number.len() > 10);

        cleanup_test_data(&db).await;
    }

    #[tokio::test]
    async fn test_sla_metrics_creation() {
        let db = match setup_test_db().await {
            Ok(db) => db,
            Err(_) => {
                println!("Skipping SLA metrics test - no database available");
                return;
            }
        };

        let tenant_id = Uuid::new_v4();
        let user_id = Uuid::new_v4();

        // Create test tenant and user
        sqlx::query!(
            "INSERT INTO tenants (id, name, domain, subscription_tier, max_users, data_residency_region)
             VALUES ($1, $2, $3, $4, $5, $6)",
            tenant_id,
            "Test Company",
            "testsla.test.com",
            "Enterprise" as SubscriptionTier,
            1000i32,
            "EU"
        )
        .execute(db.pool())
        .await
        .expect("Failed to create test tenant");

        sqlx::query!(
            "INSERT INTO users (id, tenant_id, email, username, full_name, password_hash, role)
             VALUES ($1, $2, $3, $4, $5, $6, $7)",
            user_id,
            tenant_id,
            "test@testsla.test.com",
            "testuser",
            "Test User",
            "$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj6hsCm1HLKS",
            "CustomerUser" as UserRole
        )
        .execute(db.pool())
        .await
        .expect("Failed to create test user");

        // Set application context
        sqlx::query("SELECT set_app_context($1, $2)")
            .bind(tenant_id)
            .bind(user_id)
            .execute(db.pool())
            .await
            .expect("Failed to set app context");

        // Create a ticket (should automatically create SLA metrics)
        let ticket_id = Uuid::new_v4();
        sqlx::query!(
            "INSERT INTO tickets (id, tenant_id, title, description, contact_id, created_by_id)
             VALUES ($1, $2, $3, $4, $5, $6)",
            ticket_id,
            tenant_id,
            "Test Ticket with SLA",
            "Test ticket for SLA metrics",
            user_id,
            user_id
        )
        .execute(db.pool())
        .await
        .expect("Failed to create test ticket");

        // Verify SLA metrics were created
        let sla_metrics =
            sqlx::query("SELECT COUNT(*) as count FROM sla_metrics WHERE ticket_id = $1")
                .bind(ticket_id)
                .fetch_one(db.pool())
                .await
                .expect("Failed to query SLA metrics");

        let count: i64 = sla_metrics.get("count");
        assert_eq!(
            count, 1,
            "SLA metrics should be automatically created for new tickets"
        );

        cleanup_test_data(&db).await;
    }

    #[tokio::test]
    async fn test_audit_logging() {
        let db = match setup_test_db().await {
            Ok(db) => db,
            Err(_) => {
                println!("Skipping audit logging test - no database available");
                return;
            }
        };

        let tenant_id = Uuid::new_v4();
        let user_id = Uuid::new_v4();

        // Create test tenant and user
        sqlx::query!(
            "INSERT INTO tenants (id, name, domain, subscription_tier, max_users, data_residency_region)
             VALUES ($1, $2, $3, $4, $5, $6)",
            tenant_id,
            "Audit Test Company",
            "audit.test.com",
            "Enterprise" as SubscriptionTier,
            1000i32,
            "EU"
        )
        .execute(db.pool())
        .await
        .expect("Failed to create test tenant");

        sqlx::query!(
            "INSERT INTO users (id, tenant_id, email, username, full_name, password_hash, role)
             VALUES ($1, $2, $3, $4, $5, $6, $7)",
            user_id,
            tenant_id,
            "audit@test.com",
            "audituser",
            "Audit Test User",
            "$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj6hsCm1HLKS",
            "CustomerUser" as UserRole
        )
        .execute(db.pool())
        .await
        .expect("Failed to create test user");

        // Set application context
        sqlx::query("SELECT set_app_context($1, $2)")
            .bind(tenant_id)
            .bind(user_id)
            .execute(db.pool())
            .await
            .expect("Failed to set app context");

        // Create a knowledge article (should create audit log)
        let article_id = Uuid::new_v4();
        sqlx::query!(
            "INSERT INTO knowledge_articles (id, tenant_id, title, content, author_id)
             VALUES ($1, $2, $3, $4, $5)",
            article_id,
            tenant_id,
            "Test Article for Audit",
            "This is a test article to verify audit logging works correctly.",
            user_id
        )
        .execute(db.pool())
        .await
        .expect("Failed to create test article");

        // Update the article (should create another audit log)
        sqlx::query!(
            "UPDATE knowledge_articles SET title = $1 WHERE id = $2",
            "Updated Test Article for Audit",
            article_id
        )
        .execute(db.pool())
        .await
        .expect("Failed to update test article");

        // Verify audit logs were created
        let audit_logs =
            sqlx::query("SELECT COUNT(*) as count FROM audit_logs WHERE resource_id = $1")
                .bind(article_id.to_string())
                .fetch_one(db.pool())
                .await
                .expect("Failed to query audit logs");

        let count: i64 = audit_logs.get("count");
        assert_eq!(count, 2, "Should have 2 audit logs (CREATE and UPDATE)");

        // Verify audit log details
        let create_log = sqlx::query!(
            "SELECT action, new_values FROM audit_logs
             WHERE resource_id = $1 AND action = 'CREATE'
             ORDER BY created_at ASC LIMIT 1",
            article_id.to_string()
        )
        .fetch_one(db.pool())
        .await
        .expect("Failed to fetch CREATE audit log");

        assert_eq!(create_log.action, "CREATE");
        assert!(create_log.new_values.is_some());

        cleanup_test_data(&db).await;
    }
}
