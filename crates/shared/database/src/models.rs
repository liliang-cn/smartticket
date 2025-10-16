use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use smartticket_shared_error::Result;
use sqlx::FromRow;
use uuid::Uuid;

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct Tenant {
    pub id: Uuid,
    pub name: String,
    pub domain: String,
    pub settings: serde_json::Value,
    pub subscription_tier: SubscriptionTier,
    pub max_users: i32,
    pub data_residency_region: String,
    pub is_active: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, sqlx::Type)]
#[sqlx(type_name = "subscription_tier")]
pub enum SubscriptionTier {
    #[sqlx(rename = "trial")]
    Trial,
    #[sqlx(rename = "standard")]
    Standard,
    #[sqlx(rename = "premium")]
    Premium,
    #[sqlx(rename = "enterprise")]
    Enterprise,
}

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct User {
    pub id: Uuid,
    pub tenant_id: Uuid,
    pub email: String,
    pub username: String,
    pub full_name: String,
    pub password_hash: String,
    pub role: UserRole,
    pub is_active: bool,
    pub last_login_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::Type)]
#[sqlx(type_name = "user_role")]
pub enum UserRole {
    #[sqlx(rename = "customer")]
    CustomerUser,
    #[sqlx(rename = "agent")]
    SupportEngineer,
    #[sqlx(rename = "team_lead")]
    Sales,
    #[sqlx(rename = "admin")]
    TenantAdmin,
    #[sqlx(rename = "system_admin")]
    SuperAdmin,
}

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct Ticket {
    pub id: Uuid,
    pub tenant_id: Uuid,
    pub ticket_number: String,
    pub title: String,
    pub description: String,
    pub status: TicketStatus,
    pub priority: TicketPriority,
    pub severity: TicketSeverity,
    pub category_id: Option<Uuid>,
    pub contact_id: Uuid,
    pub assigned_to_id: Option<Uuid>,
    pub created_by_id: Uuid,
    pub resolved_at: Option<DateTime<Utc>>,
    pub closed_at: Option<DateTime<Utc>>,
    pub due_at: Option<DateTime<Utc>>,
    pub resolution: Option<String>,
    pub tags: Vec<String>,
    pub custom_fields: serde_json::Value,
    pub is_deleted: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::Type)]
#[sqlx(type_name = "ticket_status")]
pub enum TicketStatus {
    New,
    Open,
    InProgress,
    PendingCustomer,
    PendingThirdParty,
    Resolved,
    Closed,
    Reopened,
}

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::Type)]
#[sqlx(type_name = "ticket_priority")]
pub enum TicketPriority {
    Low,
    Normal,
    High,
    Critical,
}

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::Type)]
#[sqlx(type_name = "ticket_severity")]
pub enum TicketSeverity {
    Low,
    Medium,
    High,
    Critical,
}

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct KnowledgeArticle {
    pub id: Uuid,
    pub tenant_id: Uuid,
    pub title: String,
    pub content: String,
    pub summary: Option<String>,
    pub category_id: Option<Uuid>,
    pub author_id: Uuid,
    pub status: KnowledgeStatus,
    pub visibility: KnowledgeVisibility,
    pub language: String,
    pub tags: Vec<String>,
    pub view_count: i32,
    pub helpful_count: i32,
    pub not_helpful_count: i32,
    pub version: i32,
    pub published_at: Option<DateTime<Utc>>,
    pub expires_at: Option<DateTime<Utc>>,
    pub is_deleted: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::Type)]
#[sqlx(type_name = "knowledge_status")]
pub enum KnowledgeStatus {
    Draft,
    Review,
    Published,
    Archived,
}

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::Type)]
#[sqlx(type_name = "knowledge_visibility")]
pub enum KnowledgeVisibility {
    Public,
    Internal,
    Restricted,
}

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct SlaPolicy {
    pub id: Uuid,
    pub tenant_id: Uuid,
    pub name: String,
    pub description: Option<String>,
    pub priority: TicketPriority,
    pub severity: TicketSeverity,
    pub response_time_minutes: i32,
    pub resolution_time_minutes: i32,
    pub business_hours_only: bool,
    pub timezone: String,
    pub is_active: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct SlaMetrics {
    pub id: Uuid,
    pub tenant_id: Uuid,
    pub ticket_id: Uuid,
    pub sla_policy_id: Uuid,
    pub response_due_at: DateTime<Utc>,
    pub resolution_due_at: DateTime<Utc>,
    pub first_response_at: Option<DateTime<Utc>>,
    pub resolved_at: Option<DateTime<Utc>>,
    pub response_breached: bool,
    pub resolution_breached: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct AuditLog {
    pub id: Uuid,
    pub tenant_id: Uuid,
    pub user_id: Option<Uuid>,
    pub action: String,
    pub resource_type: String,
    pub resource_id: String,
    pub old_values: Option<serde_json::Value>,
    pub new_values: Option<serde_json::Value>,
    pub ip_address: Option<String>,
    pub user_agent: Option<String>,
    pub created_at: DateTime<Utc>,
}

impl Tenant {
    pub fn new(
        name: String,
        domain: String,
        subscription_tier: SubscriptionTier,
        data_residency_region: String,
    ) -> Self {
        let now = Utc::now();
        Self {
            id: Uuid::new_v4(),
            name,
            domain,
            settings: serde_json::json!({}),
            subscription_tier,
            max_users: match subscription_tier {
                SubscriptionTier::Trial => 5,
                SubscriptionTier::Standard => 10,
                SubscriptionTier::Premium => 100,
                SubscriptionTier::Enterprise => 1000,
            },
            data_residency_region,
            is_active: true,
            created_at: now,
            updated_at: now,
        }
    }
}

impl User {
    pub fn new(
        tenant_id: Uuid,
        email: String,
        username: String,
        full_name: String,
        password_hash: String,
        role: UserRole,
    ) -> Self {
        let now = Utc::now();
        Self {
            id: Uuid::new_v4(),
            tenant_id,
            email,
            username,
            full_name,
            password_hash,
            role,
            is_active: true,
            last_login_at: None,
            created_at: now,
            updated_at: now,
        }
    }
}

impl Ticket {
    pub fn new(
        tenant_id: Uuid,
        ticket_number: String,
        title: String,
        description: String,
        priority: TicketPriority,
        severity: TicketSeverity,
        contact_id: Uuid,
        created_by_id: Uuid,
    ) -> Self {
        let now = Utc::now();
        Self {
            id: Uuid::new_v4(),
            tenant_id,
            ticket_number,
            title,
            description,
            status: TicketStatus::New,
            priority,
            severity,
            category_id: None,
            contact_id,
            assigned_to_id: None,
            created_by_id,
            resolved_at: None,
            closed_at: None,
            due_at: None,
            resolution: None,
            tags: vec![],
            custom_fields: serde_json::json!({}),
            is_deleted: false,
            created_at: now,
            updated_at: now,
        }
    }

    pub fn can_transition_to(&self, new_status: TicketStatus) -> Result<bool> {
        match (&self.status, new_status) {
            (TicketStatus::New, TicketStatus::Open) => Ok(true),
            (TicketStatus::New, TicketStatus::Closed) => Ok(true),
            (TicketStatus::Open, TicketStatus::InProgress) => Ok(true),
            (TicketStatus::Open, TicketStatus::PendingCustomer) => Ok(true),
            (TicketStatus::Open, TicketStatus::PendingThirdParty) => Ok(true),
            (TicketStatus::Open, TicketStatus::Closed) => Ok(true),
            (TicketStatus::InProgress, TicketStatus::PendingCustomer) => Ok(true),
            (TicketStatus::InProgress, TicketStatus::PendingThirdParty) => Ok(true),
            (TicketStatus::InProgress, TicketStatus::Resolved) => Ok(true),
            (TicketStatus::PendingCustomer, TicketStatus::InProgress) => Ok(true),
            (TicketStatus::PendingCustomer, TicketStatus::Closed) => Ok(true),
            (TicketStatus::PendingThirdParty, TicketStatus::InProgress) => Ok(true),
            (TicketStatus::Resolved, TicketStatus::Closed) => Ok(true),
            (TicketStatus::Resolved, TicketStatus::Reopened) => Ok(true),
            (TicketStatus::Closed, TicketStatus::Reopened) => Ok(true),
            (TicketStatus::Reopened, TicketStatus::Open) => Ok(true),
            _ => Ok(false),
        }
    }
}

impl KnowledgeArticle {
    pub fn new(
        tenant_id: Uuid,
        title: String,
        content: String,
        author_id: Uuid,
        language: String,
    ) -> Self {
        let now = Utc::now();
        Self {
            id: Uuid::new_v4(),
            tenant_id,
            title,
            content,
            summary: None,
            category_id: None,
            author_id,
            status: KnowledgeStatus::Draft,
            visibility: KnowledgeVisibility::Internal,
            language,
            tags: vec![],
            view_count: 0,
            helpful_count: 0,
            not_helpful_count: 0,
            version: 1,
            published_at: None,
            expires_at: None,
            is_deleted: false,
            created_at: now,
            updated_at: now,
        }
    }
}
