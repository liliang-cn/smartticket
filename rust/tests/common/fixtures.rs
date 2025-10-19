use uuid::Uuid;
use sqlx::Row;
use smartticket_shared_database::models::*;
use smartticket_shared_database::DatabaseConnection;
use smartticket_shared_error::Result;

/// Create a test ticket
pub async fn create_test_ticket(
    db: &DatabaseConnection,
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

/// Create a test knowledge article
pub async fn create_test_knowledge_article(
    db: &DatabaseConnection,
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

/// Create a test user
pub async fn create_test_user(
    db: &DatabaseConnection,
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

/// Create a test SLA policy
pub async fn create_test_sla_policy(
    db: &DatabaseConnection,
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

/// Create a test tenant
pub async fn create_test_tenant(
    db: &DatabaseConnection,
    name: &str,
    domain: &str,
) -> Result<Uuid> {
    let tenant_id = Uuid::new_v4();

    sqlx::query!(
        "INSERT INTO tenants (id, name, domain, subscription_tier, max_users, data_residency_region, is_active)
         VALUES ($1, $2, $3, $4, $5, $6, $7)",
        tenant_id,
        name,
        domain,
        SubscriptionTier::Enterprise,
        1000i32,
        "EU",
        true
    )
    .execute(db.pool())
    .await?;

    Ok(tenant_id)
}