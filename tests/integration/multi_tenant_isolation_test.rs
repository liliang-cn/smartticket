use std::collections::HashMap;
use tests::common::{TestContext, fixtures, assertions};
use smartticket_shared_database::models::*;
use smartticket_shared_auth::{TenantContext, UserData};
use smartticket_shared_error::{SmartTicketError, Result};
use uuid::Uuid;
use sqlx::Row;

/// Test multi-tenant isolation across all data access patterns
#[tokio::test]
async fn test_comprehensive_multi_tenant_isolation() -> Result<()> {
    let mut context = TestContext::new().await?;

    // Create two separate tenants
    let tenant1_id = Uuid::new_v4();
    let tenant2_id = Uuid::new_v4();

    // Setup tenant 1
    sqlx::query!(
        "INSERT INTO tenants (id, name, domain, subscription_tier, max_users, data_residency_region)
         VALUES ($1, $2, $3, $4, $5, $6)",
        tenant1_id,
        "Company A",
        "companya.smartticket.com",
        SubscriptionTier::Enterprise,
        1000i32,
        "EU"
    )
    .execute(context.db.pool())
    .await?;

    // Setup tenant 2
    sqlx::query!(
        "INSERT INTO tenants (id, name, domain, subscription_tier, max_users, data_residency_region)
         VALUES ($1, $2, $3, $4, $5, $6)",
        tenant2_id,
        "Company B",
        "companyb.smartticket.com",
        SubscriptionTier::Premium,
        500i32,
        "EU"
    )
    .execute(context.db.pool())
    .await?;

    // Create users for both tenants
    let tenant1_admin_id = create_test_user(
        &context.db,
        tenant1_id,
        "admin@companya.com",
        "admin_a",
        "Admin A",
        UserRole::TenantAdmin,
    ).await?;

    let tenant2_admin_id = create_test_user(
        &context.db,
        tenant2_id,
        "admin@companyb.com",
        "admin_b",
        "Admin B",
        UserRole::TenantAdmin,
    ).await?;

    let tenant1_user_id = create_test_user(
        &context.db,
        tenant1_id,
        "user@companya.com",
        "user_a",
        "User A",
        UserRole::CustomerUser,
    ).await?;

    let tenant2_user_id = create_test_user(
        &context.db,
        tenant2_id,
        "user@companyb.com",
        "user_b",
        "User B",
        UserRole::CustomerUser,
    ).await?;

    // Test 1: Admin from tenant 1 should only see tenant 1 data
    let tenant1_context = TenantContext::new(
        tenant1_id,
        tenant1_admin_id,
        "TenantAdmin".to_string(),
    );

    sqlx::query("SELECT set_app_context($1, $2)")
        .bind(tenant1_context.tenant_id)
        .bind(tenant1_context.user_id)
        .execute(context.db.pool())
        .await?;

    let tenant1_user_count: i64 = sqlx::query("SELECT COUNT(*) FROM users")
        .fetch_one(context.db.pool())
        .await?
        .get("count");

    assert_eq!(tenant1_user_count, 2, "Tenant 1 admin should see exactly 2 users");

    // Test 2: Admin from tenant 2 should only see tenant 2 data
    let tenant2_context = TenantContext::new(
        tenant2_id,
        tenant2_admin_id,
        "TenantAdmin".to_string(),
    );

    sqlx::query("SELECT set_app_context($1, $2)")
        .bind(tenant2_context.tenant_id)
        .bind(tenant2_context.user_id)
        .execute(context.db.pool())
        .await?;

    let tenant2_user_count: i64 = sqlx::query("SELECT COUNT(*) FROM users")
        .fetch_one(context.db.pool())
        .await?
        .get("count");

    assert_eq!(tenant2_user_count, 2, "Tenant 2 admin should see exactly 2 users");

    // Test 3: Customer user should only see their own tickets
    let ticket1_id = create_test_ticket(
        &context.db,
        tenant1_id,
        tenant1_user_id,
        tenant1_user_id,
        "Ticket from Company A",
    ).await?;

    let ticket2_id = create_test_ticket(
        &context.db,
        tenant2_id,
        tenant2_user_id,
        tenant2_user_id,
        "Ticket from Company B",
    ).await?;

    // Customer from tenant 1 context
    let tenant1_customer_context = TenantContext::new(
        tenant1_id,
        tenant1_user_id,
        "CustomerUser".to_string(),
    );

    sqlx::query("SELECT set_app_context($1, $2)")
        .bind(tenant1_customer_context.tenant_id)
        .bind(tenant1_customer_context.user_id)
        .execute(context.db.pool())
        .await?;

    let customer_ticket_count: i64 = sqlx::query("SELECT COUNT(*) FROM tickets")
        .fetch_one(context.db.pool())
        .await?
        .get("count");

    assert_eq!(customer_ticket_count, 1, "Customer should see only their own ticket");

    // Test 4: Knowledge article isolation
    let article1_id = create_test_knowledge_article(
        &context.db,
        tenant1_id,
        tenant1_admin_id,
        "Internal Knowledge - Company A",
        KnowledgeVisibility::Internal,
    ).await?;

    let article2_id = create_test_knowledge_article(
        &context.db,
        tenant2_id,
        tenant2_admin_id,
        "Internal Knowledge - Company B",
        KnowledgeVisibility::Internal,
    ).await?;

    // Admin from tenant 1 should only see tenant 1 articles
    sqlx::query("SELECT set_app_context($1, $2)")
        .bind(tenant1_context.tenant_id)
        .bind(tenant1_context.user_id)
        .execute(context.db.pool())
        .await?;

    let tenant1_article_count: i64 = sqlx::query("SELECT COUNT(*) FROM knowledge_articles")
        .fetch_one(context.db.pool())
        .await?
        .get("count");

    assert_eq!(tenant1_article_count, 1, "Tenant 1 admin should see only 1 knowledge article");

    // Test 5: SLA policy isolation
    create_test_sla_policy(
        &context.db,
        tenant1_id,
        TicketPriority::High,
        TicketSeverity::High,
        "Company A High Priority SLA",
    ).await?;

    create_test_sla_policy(
        &context.db,
        tenant2_id,
        TicketPriority::High,
        TicketSeverity::High,
        "Company B High Priority SLA",
    ).await?;

    let tenant1_sla_count: i64 = sqlx::query("SELECT COUNT(*) FROM sla_policies")
        .fetch_one(context.db.pool())
        .await?
        .get("count");

    assert_eq!(tenant1_sla_count, 1, "Tenant 1 should see only their own SLA policies");

    // Test 6: Audit log isolation
    // Admin from tenant 1 should only see audit logs for tenant 1
    let tenant1_audit_count: i64 = sqlx::query("SELECT COUNT(*) FROM audit_logs")
        .fetch_one(context.db.pool())
        .await?
        .get("count");

    // Should have audit logs for all operations performed by tenant 1
    assert!(tenant1_audit_count > 0, "Should have audit logs for tenant 1 operations");

    context.cleanup().await?;
    Ok(())
}

/// Test cross-tenant data access attempts (should fail)
#[tokio::test]
async fn test_cross_tenant_access_violations() -> Result<()> {
    let mut context = TestContext::new().await?;

    let tenant1_id = Uuid::new_v4();
    let tenant2_id = Uuid::new_v4();

    // Setup tenants
    sqlx::query!(
        "INSERT INTO tenants (id, name, domain, subscription_tier, max_users, data_residency_region)
         VALUES ($1, $2, $3, $4, $5, $6)",
        tenant1_id,
        "Company A",
        "companya.smartticket.com",
        SubscriptionTier::Enterprise,
        1000i32,
        "EU"
    )
    .execute(context.db.pool())
    .await?;

    sqlx::query!(
        "INSERT INTO tenants (id, name, domain, subscription_tier, max_users, data_residency_region)
         VALUES ($1, $2, $3, $4, $5, $6)",
        tenant2_id,
        "Company B",
        "companyb.smartticket.com",
        SubscriptionTier::Premium,
        500i32,
        "EU"
    )
    .execute(context.db.pool())
    .await?;

    // Create admin for tenant 1
    let tenant1_admin_id = create_test_user(
        &context.db,
        tenant1_id,
        "admin@companya.com",
        "admin_a",
        "Admin A",
        UserRole::TenantAdmin,
    ).await?;

    // Create user in tenant 2
    let tenant2_user_id = create_test_user(
        &context.db,
        tenant2_id,
        "user@companyb.com",
        "user_b",
        "User B",
        UserRole::CustomerUser,
    ).await?;

    // Set context as tenant 1 admin
    let tenant1_context = TenantContext::new(
        tenant1_id,
        tenant1_admin_id,
        "TenantAdmin".to_string(),
    );

    sqlx::query("SELECT set_app_context($1, $2)")
        .bind(tenant1_context.tenant_id)
        .bind(tenant1_context.user_id)
        .execute(context.db.pool())
        .await?;

    // Test: Try to access user from different tenant (should return empty)
    let cross_tenant_user = sqlx::query("SELECT * FROM users WHERE id = $1")
        .bind(tenant2_user_id)
        .fetch_optional(context.db.pool())
        .await?;

    assert!(cross_tenant_user.is_none(), "Should not be able to access user from different tenant");

    // Test: Create ticket for different tenant user (should fail due to RLS)
    let result = sqlx::query!(
        "INSERT INTO tickets (id, tenant_id, title, description, contact_id, created_by_id)
         VALUES ($1, $2, $3, $4, $5, $6)",
        Uuid::new_v4(),
        tenant2_id,  // Wrong tenant
        "Cross-tenant ticket",
        "This should fail",
        tenant2_user_id,
        tenant1_admin_id
    )
    .execute(context.db.pool())
    .await;

    assert!(result.is_err(), "Should not be able to create ticket for different tenant");

    context.cleanup().await?;
    Ok(())
}

/// Test SuperAdmin role can access all tenants
#[tokio::test]
async fn test_superadmin_cross_tenant_access() -> Result<()> {
    let mut context = TestContext::new().await?;

    let tenant1_id = Uuid::new_v4();
    let tenant2_id = Uuid::new_v4();

    // Setup tenants
    sqlx::query!(
        "INSERT INTO tenants (id, name, domain, subscription_tier, max_users, data_residency_region)
         VALUES ($1, $2, $3, $4, $5, $6)",
        tenant1_id,
        "Company A",
        "companya.smartticket.com",
        SubscriptionTier::Enterprise,
        1000i32,
        "EU"
    )
    .execute(context.db.pool())
    .await?;

    sqlx::query!(
        "INSERT INTO tenants (id, name, domain, subscription_tier, max_users, data_residency_region)
         VALUES ($1, $2, $3, $4, $5, $6)",
        tenant2_id,
        "Company B",
        "companyb.smartticket.com",
        SubscriptionTier::Premium,
        500i32,
        "EU"
    )
    .execute(context.db.pool())
    .await?;

    // Create SuperAdmin user
    let superadmin_id = create_test_user(
        &context.db,
        tenant1_id,
        "superadmin@smartticket.com",
        "superadmin",
        "Super Admin",
        UserRole::SuperAdmin,
    ).await?;

    // Create regular users in both tenants
    let tenant1_user_id = create_test_user(
        &context.db,
        tenant1_id,
        "user@companya.com",
        "user_a",
        "User A",
        UserRole::CustomerUser,
    ).await?;

    let tenant2_user_id = create_test_user(
        &context.db,
        tenant2_id,
        "user@companyb.com",
        "user_b",
        "User B",
        UserRole::CustomerUser,
    ).await?;

    // Set context as SuperAdmin
    let superadmin_context = TenantContext::new(
        tenant1_id,
        superadmin_id,
        "SuperAdmin".to_string(),
    );

    sqlx::query("SELECT set_app_context($1, $2)")
        .bind(superadmin_context.tenant_id)
        .bind(superadmin_context.user_id)
        .execute(context.db.pool())
        .await?;

    // Test: SuperAdmin should see all users across all tenants
    let total_users: i64 = sqlx::query("SELECT COUNT(*) FROM users")
        .fetch_one(context.db.pool())
        .await?
        .get("count");

    assert_eq!(total_users, 3, "SuperAdmin should see all users across all tenants");

    // Test: SuperAdmin should see all tickets across all tenants
    let ticket1_id = create_test_ticket(
        &context.db,
        tenant1_id,
        tenant1_user_id,
        tenant1_user_id,
        "Ticket from Company A",
    ).await?;

    let ticket2_id = create_test_ticket(
        &context.db,
        tenant2_id,
        tenant2_user_id,
        tenant2_user_id,
        "Ticket from Company B",
    ).await?;

    let total_tickets: i64 = sqlx::query("SELECT COUNT(*) FROM tickets")
        .fetch_one(context.db.pool())
        .await?
        .get("count");

    assert_eq!(total_tickets, 2, "SuperAdmin should see all tickets across all tenants");

    context.cleanup().await?;
    Ok(())
}

/// Test JWT token tenant isolation
#[tokio::test]
async fn test_jwt_tenant_isolation() -> Result<()> {
    let mut context = TestContext::new().await?;

    let tenant1_id = Uuid::new_v4();
    let tenant2_id = Uuid::new_v4();

    // Setup tenants
    sqlx::query!(
        "INSERT INTO tenants (id, name, domain, subscription_tier, max_users, data_residency_region)
         VALUES ($1, $2, $3, $4, $5, $6)",
        tenant1_id,
        "Company A",
        "companya.smartticket.com",
        SubscriptionTier::Enterprise,
        1000i32,
        "EU"
    )
    .execute(context.db.pool())
    .await?;

    sqlx::query!(
        "INSERT INTO tenants (id, name, domain, subscription_tier, max_users, data_residency_region)
         VALUES ($1, $2, $3, $4, $5, $6)",
        tenant2_id,
        "Company B",
        "companyb.smartticket.com",
        SubscriptionTier::Premium,
        500i32,
        "EU"
    )
    .execute(context.db.pool())
    .await?;

    // Create users
    let tenant1_user_id = create_test_user(
        &context.db,
        tenant1_id,
        "user@companya.com",
        "user_a",
        "User A",
        UserRole::CustomerUser,
    ).await?;

    let tenant2_user_id = create_test_user(
        &context.db,
        tenant2_id,
        "user@companyb.com",
        "user_b",
        "User B",
        UserRole::CustomerUser,
    ).await?;

    // Generate JWT tokens for both users
    let tenant1_user_data = UserData {
        email: "user@companya.com".to_string(),
        username: "user_a".to_string(),
        full_name: "User A".to_string(),
        role: "CustomerUser".to_string(),
        tenant_id: tenant1_id.to_string(),
        tenant_name: "Company A".to_string(),
        permissions: vec!["tickets:read".to_string()],
    };

    let tenant2_user_data = UserData {
        email: "user@companyb.com".to_string(),
        username: "user_b".to_string(),
        full_name: "User B".to_string(),
        role: "CustomerUser".to_string(),
        tenant_id: tenant2_id.to_string(),
        tenant_name: "Company B".to_string(),
        permissions: vec!["tickets:read".to_string()],
    };

    let token1 = context.generate_test_token(tenant1_user_data);
    let token2 = context.generate_test_token(tenant2_user_data);

    // Verify token1 contains tenant1 info
    let claims1 = context.jwt_service.verify_user_token(&token1)?;
    assert_eq!(claims1.tenant_id, tenant1_id.to_string());
    assert_eq!(claims1.tenant_name, "Company A");

    // Verify token2 contains tenant2 info
    let claims2 = context.jwt_service.verify_user_token(&token2)?;
    assert_eq!(claims2.tenant_id, tenant2_id.to_string());
    assert_eq!(claims2.tenant_name, "Company B");

    // Test tenant context isolation
    let context1 = TenantContext::from_claims(&claims1);
    let context2 = TenantContext::from_claims(&claims2);

    assert!(!context1.can_access_tenant(&tenant2_id.to_string()));
    assert!(!context2.can_access_tenant(&tenant1_id.to_string()));
    assert!(context1.can_access_tenant(&tenant1_id.to_string()));
    assert!(context2.can_access_tenant(&tenant2_id.to_string()));

    context.cleanup().await?;
    Ok(())
}

// Helper functions for test data creation
async fn create_test_user(
    db: &smartticket_shared_database::DatabaseConnection,
    tenant_id: Uuid,
    email: &str,
    username: &str,
    full_name: &str,
    role: UserRole,
) -> Result<Uuid> {
    let user_id = Uuid::new_v4();

    sqlx::query!(
        "INSERT INTO users (id, tenant_id, email, username, full_name, password_hash, role)
         VALUES ($1, $2, $3, $4, $5, $6, $7)",
        user_id,
        tenant_id,
        email,
        username,
        full_name,
        "$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj6hsCm1HLKS",
        role as _
    )
    .execute(db.pool())
    .await?;

    Ok(user_id)
}

async fn create_test_ticket(
    db: &smartticket_shared_database::DatabaseConnection,
    tenant_id: Uuid,
    contact_id: Uuid,
    created_by_id: Uuid,
    title: &str,
) -> Result<Uuid> {
    let ticket_id = Uuid::new_v4();

    sqlx::query!(
        "INSERT INTO tickets (id, tenant_id, title, description, contact_id, created_by_id)
         VALUES ($1, $2, $3, $4, $5, $6)",
        ticket_id,
        tenant_id,
        title,
        format!("Test ticket: {}", title),
        contact_id,
        created_by_id
    )
    .execute(db.pool())
    .await?;

    Ok(ticket_id)
}

async fn create_test_knowledge_article(
    db: &smartticket_shared_database::DatabaseConnection,
    tenant_id: Uuid,
    author_id: Uuid,
    title: &str,
    visibility: KnowledgeVisibility,
) -> Result<Uuid> {
    let article_id = Uuid::new_v4();

    sqlx::query!(
        "INSERT INTO knowledge_articles (id, tenant_id, title, content, author_id, visibility)
         VALUES ($1, $2, $3, $4, $5, $6)",
        article_id,
        tenant_id,
        title,
        format!("Test content for: {}", title),
        author_id,
        visibility as _
    )
    .execute(db.pool())
    .await?;

    Ok(article_id)
}

async fn create_test_sla_policy(
    db: &smartticket_shared_database::DatabaseConnection,
    tenant_id: Uuid,
    priority: TicketPriority,
    severity: TicketSeverity,
    name: &str,
) -> Result<Uuid> {
    let policy_id = Uuid::new_v4();

    sqlx::query!(
        "INSERT INTO sla_policies (id, tenant_id, name, priority, severity, response_time_minutes, resolution_time_minutes)
         VALUES ($1, $2, $3, $4, $5, $6, $7)",
        policy_id,
        tenant_id,
        name,
        priority as _,
        severity as _,
        60i32,
        480i32
    )
    .execute(db.pool())
    .await?;

    Ok(policy_id)
}