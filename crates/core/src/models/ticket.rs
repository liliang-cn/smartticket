use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::FromRow;
use uuid::Uuid;

/// Ticket status enumeration
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, sqlx::Type)]
#[sqlx(type_name = "ticket_status")]
pub enum TicketStatus {
    Unspecified = 0,
    Open = 1,
    InProgress = 2,
    PendingCustomer = 3,
    PendingThirdParty = 4,
    Resolved = 5,
    Closed = 6,
    Reopened = 7,
}

impl Default for TicketStatus {
    fn default() -> Self {
        TicketStatus::Open
    }
}

/// Ticket priority enumeration
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, sqlx::Type)]
#[sqlx(type_name = "ticket_priority")]
pub enum TicketPriority {
    Unspecified = 0,
    Low = 1,
    Normal = 2,
    High = 3,
    Urgent = 4,
    Critical = 5,
}

impl Default for TicketPriority {
    fn default() -> Self {
        TicketPriority::Normal
    }
}

/// Ticket type enumeration
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, sqlx::Type)]
#[sqlx(type_name = "ticket_type", rename_all = "snake_case")]
pub enum TicketType {
    Unspecified = 0,
    Incident = 1,
    ServiceRequest = 2,
    Problem = 3,
    Change = 4,
    Question = 5,
}

impl Default for TicketType {
    fn default() -> Self {
        TicketType::Incident
    }
}

/// Comment type enumeration
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, sqlx::Type)]
#[sqlx(type_name = "comment_type", rename_all = "snake_case")]
pub enum CommentType {
    Unspecified = 0,
    Public = 1,
    Internal = 2,
    System = 3,
    Resolution = 4,
}

impl Default for CommentType {
    fn default() -> Self {
        CommentType::Public
    }
}

/// SLA status enumeration
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, sqlx::Type)]
#[sqlx(type_name = "sla_status", rename_all = "snake_case")]
pub enum SLAStatus {
    Unspecified = 0,
    Ok = 1,
    Warning = 2,
    Breached = 3,
    Paused = 4,
}

impl Default for SLAStatus {
    fn default() -> Self {
        SLAStatus::Ok
    }
}

/// Ticket database model
#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct Ticket {
    pub id: Uuid,
    pub tenant_id: Uuid,
    pub customer_id: Uuid, // This field maps to database contact_id column
    pub assigned_agent_id: Option<Uuid>,
    pub team_id: Option<Uuid>,
    pub title: String,
    pub description: String,
    pub status: TicketStatus,
    pub priority: TicketPriority,
    pub ticket_type: TicketType,
    pub category_id: Option<Uuid>,
    pub tags: Vec<String>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub due_date: Option<DateTime<Utc>>,
    pub resolved_at: Option<DateTime<Utc>>,
    pub closed_at: Option<DateTime<Utc>>,
    pub resolution: Option<String>,
    pub satisfaction_rating: Option<i64>,
    pub external_reference: Option<String>,
    pub custom_fields: Option<serde_json::Value>,
    pub is_deleted: bool,
    pub created_by: Option<String>,
    pub updated_by: Option<String>,
}

/// Ticket comment database model
#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct TicketComment {
    pub id: Uuid,
    pub ticket_id: Uuid,
    pub author_id: String,
    pub author_name: String,
    pub author_email: String,
    pub content: String,
    pub comment_type: CommentType,
    pub is_internal: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub is_deleted: bool,
    pub created_by: String,
    pub updated_by: String,
}

/// Ticket SLA information
#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct TicketSLA {
    pub id: Uuid,
    pub ticket_id: Uuid,
    pub sla_policy_id: Uuid,
    pub response_due: DateTime<Utc>,
    pub resolution_due: DateTime<Utc>,
    pub next_breach_time: Option<DateTime<Utc>>,
    pub status: SLAStatus,
    pub minutes_to_response_breach: Option<i64>,
    pub minutes_to_resolution_breach: Option<i64>,
    pub actual_response_time: Option<DateTime<Utc>>,
    pub actual_resolution_time: Option<DateTime<Utc>>,
    pub is_response_met: bool,
    pub is_resolution_met: bool,
    pub breach_count: i32,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub created_by: String,
    pub updated_by: String,
}

/// Ticket category model
#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct TicketCategory {
    pub id: Uuid,
    pub tenant_id: Uuid,
    pub name: String,
    pub description: Option<String>,
    pub parent_id: Option<Uuid>,
    pub color: Option<String>,
    pub icon: Option<String>,
    pub is_active: bool,
    pub sort_order: i32,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub created_by: String,
    pub updated_by: String,
}

/// SLA policy model
#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct SLAPolicy {
    pub id: Uuid,
    pub tenant_id: Uuid,
    pub name: String,
    pub description: Option<String>,
    pub is_active: bool,
    pub response_time_minutes: i32,
    pub resolution_time_minutes: i32,
    pub business_hours_only: bool,
    pub priority_multipliers: Option<serde_json::Value>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub created_by: String,
    pub updated_by: String,
}

/// Ticket activity log for audit trail
#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct TicketActivity {
    pub id: Uuid,
    pub ticket_id: Uuid,
    pub actor_id: String,
    pub actor_name: String,
    pub action: String,
    pub old_value: Option<String>,
    pub new_value: Option<String>,
    pub details: Option<serde_json::Value>,
    pub created_at: DateTime<Utc>,
    pub created_by: String,
}

/// Ticket attachment model
#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct TicketAttachment {
    pub id: Uuid,
    pub ticket_id: Uuid,
    pub comment_id: Option<Uuid>,
    pub filename: String,
    pub original_filename: String,
    pub content_type: String,
    pub file_size: i64,
    pub storage_url: String,
    pub thumbnail_url: Option<String>,
    pub is_inline: bool,
    pub created_at: DateTime<Utc>,
    pub created_by: String,
}

/// Ticket statistics for reporting
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TicketStats {
    pub total_tickets: i64,
    pub open_tickets: i64,
    pub in_progress_tickets: i64,
    pub resolved_tickets: i64,
    pub closed_tickets: i64,
    pub overdue_tickets: i64,
    pub tickets_by_priority: Vec<TicketPriorityStats>,
    pub tickets_by_type: Vec<TicketTypeStats>,
    pub tickets_by_category: Vec<TicketCategoryStats>,
    pub average_resolution_time_minutes: Option<f64>,
    pub average_response_time_minutes: Option<f64>,
    pub satisfaction_score_average: Option<f64>,
}

/// Ticket priority statistics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TicketPriorityStats {
    pub priority: TicketPriority,
    pub count: i64,
    pub percentage: f64,
}

/// Ticket type statistics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TicketTypeStats {
    pub ticket_type: TicketType,
    pub count: i64,
    pub percentage: f64,
}

/// Ticket category statistics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TicketCategoryStats {
    pub category_id: Uuid,
    pub category_name: String,
    pub count: i64,
    pub percentage: f64,
}

/// Create ticket request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateTicketRequest {
    pub tenant_id: Uuid,
    pub customer_id: Uuid,
    pub title: String,
    pub description: String,
    pub priority: TicketPriority,
    pub ticket_type: TicketType,
    pub category_id: Option<Uuid>,
    pub tags: Vec<String>,
    pub team_id: Option<Uuid>,
    pub due_date: Option<DateTime<Utc>>,
    pub external_reference: Option<String>,
}

/// Update ticket request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateTicketRequest {
    pub id: Uuid,
    pub tenant_id: Uuid,
    pub title: Option<String>,
    pub description: Option<String>,
    pub priority: Option<TicketPriority>,
    pub category_id: Option<Uuid>,
    pub tags: Option<Vec<String>>,
    pub due_date: Option<DateTime<Utc>>,
    pub external_reference: Option<String>,
}

/// Ticket search filters
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TicketSearchFilters {
    pub tenant_id: Uuid,
    pub customer_id: Option<Uuid>,
    pub assigned_agent_id: Option<Uuid>,
    pub team_id: Option<Uuid>,
    pub status: Option<TicketStatus>,
    pub priority: Option<TicketPriority>,
    pub ticket_type: Option<TicketType>,
    pub category_id: Option<Uuid>,
    pub created_after: Option<DateTime<Utc>>,
    pub created_before: Option<DateTime<Utc>>,
    pub updated_after: Option<DateTime<Utc>>,
    pub updated_before: Option<DateTime<Utc>>,
    pub search_query: Option<String>,
    pub tags: Vec<String>,
    pub page_size: Option<i32>,
    pub page_token: Option<String>,
    pub order_by: Option<String>,
    pub order_desc: Option<bool>,
}

/// Paginated ticket list response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TicketListResponse {
    pub tickets: Vec<Ticket>,
    pub total_count: i64,
    pub next_page_token: Option<String>,
}

impl Ticket {
    /// Create a new ticket
    pub fn new(request: CreateTicketRequest, created_by: String) -> Self {
        let now = Utc::now();
        Self {
            id: Uuid::new_v4(),
            tenant_id: request.tenant_id,
            customer_id: request.customer_id,
            assigned_agent_id: None,
            team_id: request.team_id,
            title: request.title,
            description: request.description,
            status: TicketStatus::Open,
            priority: request.priority,
            ticket_type: request.ticket_type,
            category_id: request.category_id,
            tags: request.tags,
            created_at: now,
            updated_at: now,
            due_date: request.due_date,
            resolved_at: None,
            closed_at: None,
            resolution: None,
            satisfaction_rating: None,
            external_reference: request.external_reference,
            custom_fields: None,
            is_deleted: false,
            created_by: Some(created_by.clone()),
            updated_by: Some(created_by),
        }
    }

    /// Check if ticket is in a final state
    pub fn is_closed(&self) -> bool {
        matches!(self.status, TicketStatus::Closed)
    }

    /// Check if ticket is resolved
    pub fn is_resolved(&self) -> bool {
        matches!(self.status, TicketStatus::Resolved | TicketStatus::Closed)
    }

    /// Check if ticket is actively being worked on
    pub fn is_active(&self) -> bool {
        !matches!(self.status, TicketStatus::Resolved | TicketStatus::Closed)
    }

    /// Get the time since creation
    pub fn age(&self) -> chrono::Duration {
        Utc::now() - self.created_at
    }

    /// Get the time since last update
    pub fn time_since_update(&self) -> chrono::Duration {
        Utc::now() - self.updated_at
    }

    /// Check if ticket is overdue based on due date
    pub fn is_overdue(&self) -> bool {
        if let Some(due_date) = self.due_date {
            Utc::now() > due_date && self.is_active()
        } else {
            false
        }
    }

    /// Check if ticket can be assigned to an agent
    pub fn can_be_assigned(&self) -> bool {
        matches!(self.status, TicketStatus::Open | TicketStatus::Reopened)
    }

    /// Check if ticket status can be changed to the target status
    pub fn can_change_status(&self, target_status: TicketStatus) -> bool {
        match (self.status, target_status) {
            // From Open can go to any state except Unspecified
            (TicketStatus::Open, target) if target != TicketStatus::Unspecified => true,

            // From InProgress can go to most states
            (TicketStatus::InProgress, TicketStatus::PendingCustomer) => true,
            (TicketStatus::InProgress, TicketStatus::PendingThirdParty) => true,
            (TicketStatus::InProgress, TicketStatus::Resolved) => true,
            (TicketStatus::InProgress, TicketStatus::Closed) => true,

            // From pending states can go back to InProgress or resolved/closed
            (TicketStatus::PendingCustomer, TicketStatus::InProgress) => true,
            (TicketStatus::PendingCustomer, TicketStatus::Resolved) => true,
            (TicketStatus::PendingCustomer, TicketStatus::Closed) => true,

            (TicketStatus::PendingThirdParty, TicketStatus::InProgress) => true,
            (TicketStatus::PendingThirdParty, TicketStatus::Resolved) => true,
            (TicketStatus::PendingThirdParty, TicketStatus::Closed) => true,

            // From Resolved can go to Closed or be reopened
            (TicketStatus::Resolved, TicketStatus::Closed) => true,
            (TicketStatus::Resolved, TicketStatus::Reopened) => true,

            // From Closed can be reopened
            (TicketStatus::Closed, TicketStatus::Reopened) => true,

            // From Reopened can go to Open or InProgress
            (TicketStatus::Reopened, TicketStatus::Open) => true,
            (TicketStatus::Reopened, TicketStatus::InProgress) => true,

            // Same status is allowed (no-op)
            (current, target) if current == target => true,

            // All other transitions are not allowed
            _ => false,
        }
    }

    /// Update ticket status with validation
    pub fn update_status(
        &mut self,
        new_status: TicketStatus,
        updated_by: String,
    ) -> Result<(), String> {
        if !self.can_change_status(new_status) {
            return Err(format!(
                "Cannot change status from {:?} to {:?}",
                self.status, new_status
            ));
        }

        let now = Utc::now();

        // Update timestamps based on status changes
        match new_status {
            TicketStatus::Resolved => {
                if self.resolved_at.is_none() {
                    self.resolved_at = Some(now);
                }
            }
            TicketStatus::Closed => {
                if self.closed_at.is_none() {
                    self.closed_at = Some(now);
                }
                if self.resolved_at.is_none() {
                    self.resolved_at = Some(now);
                }
            }
            TicketStatus::Reopened => {
                // Clear resolution timestamps when reopening
                self.resolved_at = None;
                self.closed_at = None;
            }
            _ => {}
        }

        self.status = new_status;
        self.updated_at = now;
        self.updated_by = Some(updated_by);

        Ok(())
    }

    /// Assign ticket to an agent
    pub fn assign_to_agent(&mut self, agent_id: Uuid, updated_by: String) -> Result<(), String> {
        if !self.can_be_assigned() {
            return Err("Ticket cannot be assigned in current status".to_string());
        }

        self.assigned_agent_id = Some(agent_id);
        self.updated_at = Utc::now();
        self.updated_by = Some(updated_by);

        // Auto-transition to InProgress if currently Open
        if self.status == TicketStatus::Open {
            self.status = TicketStatus::InProgress;
        }

        Ok(())
    }

    /// Unassign ticket from agent
    pub fn unassign(&mut self, updated_by: String) {
        self.assigned_agent_id = None;
        self.updated_at = Utc::now();
        self.updated_by = Some(updated_by);
    }

    /// Add a tag to the ticket
    pub fn add_tag(&mut self, tag: String, updated_by: String) {
        if !self.tags.contains(&tag) {
            self.tags.push(tag);
            self.updated_at = Utc::now();
            self.updated_by = Some(updated_by);
        }
    }

    /// Remove a tag from the ticket
    pub fn remove_tag(&mut self, tag: &str, updated_by: String) {
        if let Some(index) = self.tags.iter().position(|t| t == tag) {
            self.tags.remove(index);
            self.updated_at = Utc::now();
            self.updated_by = Some(updated_by);
        }
    }

    /// Update ticket resolution
    pub fn set_resolution(&mut self, resolution: String, updated_by: String) {
        self.resolution = Some(resolution);
        self.updated_at = Utc::now();
        self.updated_by = Some(updated_by);
    }

    /// Set satisfaction rating
    pub fn set_satisfaction_rating(&mut self, rating: i64, updated_by: String) {
        self.satisfaction_rating = Some(rating.clamp(1, 5));
        self.updated_at = Utc::now();
        self.updated_by = Some(updated_by);
    }
}

impl TicketComment {
    /// Create a new comment
    pub fn new(
        ticket_id: Uuid,
        author_id: String,
        author_name: String,
        author_email: String,
        content: String,
        comment_type: CommentType,
        is_internal: bool,
        created_by: String,
    ) -> Self {
        let now = Utc::now();
        Self {
            id: Uuid::new_v4(),
            ticket_id,
            author_id,
            author_name,
            author_email,
            content,
            comment_type,
            is_internal,
            created_at: now,
            updated_at: now,
            is_deleted: false,
            created_by: created_by.clone(),
            updated_by: created_by,
        }
    }

    /// Update comment content
    pub fn update_content(&mut self, content: String, updated_by: String) {
        self.content = content;
        self.updated_at = Utc::now();
        self.updated_by = updated_by;
    }

    /// Mark comment as deleted (soft delete)
    pub fn delete(&mut self, updated_by: String) {
        self.is_deleted = true;
        self.updated_at = Utc::now();
        self.updated_by = updated_by;
    }
}

impl TicketSLA {
    /// Create new SLA entry for a ticket
    pub fn new(
        ticket_id: Uuid,
        sla_policy_id: Uuid,
        response_due: DateTime<Utc>,
        resolution_due: DateTime<Utc>,
        created_by: String,
    ) -> Self {
        let now = Utc::now();
        Self {
            id: Uuid::new_v4(),
            ticket_id,
            sla_policy_id,
            response_due,
            resolution_due,
            next_breach_time: Some(response_due),
            status: SLAStatus::Ok,
            minutes_to_response_breach: Some((response_due - now).num_minutes()),
            minutes_to_resolution_breach: Some((resolution_due - now).num_minutes()),
            actual_response_time: None,
            actual_resolution_time: None,
            is_response_met: false,
            is_resolution_met: false,
            breach_count: 0,
            created_at: now,
            updated_at: now,
            created_by: created_by.clone(),
            updated_by: created_by,
        }
    }

    /// Check if response SLA is met
    pub fn check_response_sla(&mut self) {
        let now = Utc::now();
        if self.actual_response_time.is_none() && now > self.response_due {
            self.status = SLAStatus::Breached;
            self.breach_count += 1;
            self.minutes_to_response_breach = Some(0);
        } else if self.actual_response_time.is_none() {
            self.minutes_to_response_breach = Some((self.response_due - now).num_minutes());

            // Update status based on proximity to breach
            let minutes_to_breach = (self.response_due - now).num_minutes();
            self.status = if minutes_to_breach <= 0 {
                SLAStatus::Breached
            } else if minutes_to_breach <= 60 {
                SLAStatus::Warning
            } else {
                SLAStatus::Ok
            };
        }
    }

    /// Check if resolution SLA is met
    pub fn check_resolution_sla(&mut self) {
        let now = Utc::now();
        if self.actual_resolution_time.is_some() {
            self.is_resolution_met = self.actual_resolution_time.unwrap() <= self.resolution_due;
        } else if now > self.resolution_due {
            self.status = SLAStatus::Breached;
            self.breach_count += 1;
            self.minutes_to_resolution_breach = Some(0);
        } else {
            self.minutes_to_resolution_breach = Some((self.resolution_due - now).num_minutes());
        }
    }

    /// Update next breach time
    pub fn update_next_breach_time(&mut self) {
        let now = Utc::now();
        if self.actual_response_time.is_none() && now < self.response_due {
            self.next_breach_time = Some(self.response_due);
        } else if self.actual_response_time.is_some() && now < self.resolution_due {
            self.next_breach_time = Some(self.resolution_due);
        } else {
            self.next_breach_time = None;
        }
    }

    /// Mark response time as met
    pub fn mark_response_met(&mut self, updated_by: String) {
        self.actual_response_time = Some(Utc::now());
        self.is_response_met = true;
        self.updated_at = Utc::now();
        self.updated_by = updated_by;
        self.update_next_breach_time();
    }

    /// Mark resolution time as met
    pub fn mark_resolution_met(&mut self, updated_by: String) {
        self.actual_resolution_time = Some(Utc::now());
        self.is_resolution_met = self.actual_resolution_time.unwrap() <= self.resolution_due;
        self.updated_at = Utc::now();
        self.updated_by = updated_by;
        self.update_next_breach_time();
    }
}

impl std::fmt::Display for TicketStatus {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            TicketStatus::Unspecified => write!(f, "New"), // Map to New as first status
            TicketStatus::Open => write!(f, "Open"),
            TicketStatus::InProgress => write!(f, "InProgress"),
            TicketStatus::PendingCustomer => write!(f, "PendingCustomer"),
            TicketStatus::PendingThirdParty => write!(f, "PendingThirdParty"),
            TicketStatus::Resolved => write!(f, "Resolved"),
            TicketStatus::Closed => write!(f, "Closed"),
            TicketStatus::Reopened => write!(f, "Reopened"),
        }
    }
}

impl std::fmt::Display for TicketPriority {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            TicketPriority::Unspecified => write!(f, "Low"), // Map to Low as default
            TicketPriority::Low => write!(f, "Low"),
            TicketPriority::Normal => write!(f, "Normal"),
            TicketPriority::High => write!(f, "High"),
            TicketPriority::Urgent => write!(f, "High"), // Map Urgent to High
            TicketPriority::Critical => write!(f, "Critical"),
        }
    }
}

impl std::fmt::Display for TicketType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            TicketType::Unspecified => write!(f, "unspecified"),
            TicketType::Incident => write!(f, "incident"),
            TicketType::ServiceRequest => write!(f, "service_request"),
            TicketType::Problem => write!(f, "problem"),
            TicketType::Change => write!(f, "change"),
            TicketType::Question => write!(f, "question"),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_ticket_creation() {
        let request = CreateTicketRequest {
            tenant_id: Uuid::new_v4(),
            customer_id: Uuid::new_v4(),
            title: "Test Ticket".to_string(),
            description: "Test Description".to_string(),
            priority: TicketPriority::Normal,
            ticket_type: TicketType::Incident,
            category_id: None,
            tags: vec!["test".to_string()],
            team_id: None,
            due_date: None,
            external_reference: None,
        };

        let ticket = Ticket::new(request, "test_user".to_string());

        assert_eq!(ticket.status, TicketStatus::Open);
        assert_eq!(ticket.priority, TicketPriority::Normal);
        assert_eq!(ticket.ticket_type, TicketType::Incident);
        assert!(ticket.tags.contains(&"test".to_string()));
        assert!(ticket.is_active());
        assert!(!ticket.is_resolved());
        assert!(!ticket.is_closed());
    }

    #[test]
    fn test_ticket_status_transitions() {
        let mut ticket = Ticket {
            id: Uuid::new_v4(),
            tenant_id: Uuid::new_v4(),
            customer_id: Uuid::new_v4(),
            assigned_agent_id: None,
            team_id: None,
            title: "Test".to_string(),
            description: "Test".to_string(),
            status: TicketStatus::Open,
            priority: TicketPriority::Normal,
            ticket_type: TicketType::Incident,
            category_id: None,
            tags: vec![],
            created_at: Utc::now(),
            updated_at: Utc::now(),
            due_date: None,
            resolved_at: None,
            closed_at: None,
            resolution: None,
            satisfaction_rating: None,
            external_reference: None,
            custom_fields: None,
            is_deleted: false,
            created_by: Some("test".to_string()),
            updated_by: Some("test".to_string()),
        };

        // Valid transitions
        assert!(ticket
            .update_status(TicketStatus::InProgress, "test".to_string())
            .is_ok());
        assert_eq!(ticket.status, TicketStatus::InProgress);

        assert!(ticket
            .update_status(TicketStatus::Resolved, "test".to_string())
            .is_ok());
        assert_eq!(ticket.status, TicketStatus::Resolved);
        assert!(ticket.resolved_at.is_some());

        assert!(ticket
            .update_status(TicketStatus::Closed, "test".to_string())
            .is_ok());
        assert_eq!(ticket.status, TicketStatus::Closed);
        assert!(ticket.closed_at.is_some());

        // Invalid transition
        assert!(ticket
            .update_status(TicketStatus::Open, "test".to_string())
            .is_err());
    }

    #[test]
    fn test_ticket_assignment() {
        let mut ticket = Ticket {
            id: Uuid::new_v4(),
            tenant_id: Uuid::new_v4(),
            customer_id: Uuid::new_v4(),
            assigned_agent_id: None,
            team_id: None,
            title: "Test".to_string(),
            description: "Test".to_string(),
            status: TicketStatus::Open,
            priority: TicketPriority::Normal,
            ticket_type: TicketType::Incident,
            category_id: None,
            tags: vec![],
            created_at: Utc::now(),
            updated_at: Utc::now(),
            due_date: None,
            resolved_at: None,
            closed_at: None,
            resolution: None,
            satisfaction_rating: None,
            external_reference: None,
            custom_fields: None,
            is_deleted: false,
            created_by: Some("test".to_string()),
            updated_by: Some("test".to_string()),
        };

        let agent_id = Uuid::new_v4();
        assert!(ticket.assign_to_agent(agent_id, "test".to_string()).is_ok());
        assert_eq!(ticket.assigned_agent_id, Some(agent_id));
        assert_eq!(ticket.status, TicketStatus::InProgress);

        ticket.unassign("test".to_string());
        assert_eq!(ticket.assigned_agent_id, None);
    }

    #[test]
    fn test_sla_calculations() {
        let now = Utc::now();
        let response_due = now + chrono::Duration::hours(2); // 2 hours to avoid Warning
        let resolution_due = now + chrono::Duration::hours(24);

        let mut sla = TicketSLA::new(
            Uuid::new_v4(),
            Uuid::new_v4(),
            response_due,
            resolution_due,
            "test".to_string(),
        );

        assert_eq!(sla.status, SLAStatus::Ok);
        assert!(sla.minutes_to_response_breach.unwrap() > 0);

        // Check response SLA
        sla.check_response_sla();
        assert_eq!(sla.status, SLAStatus::Ok); // Should be Ok with 2 hours buffer

        // Mark response as met
        sla.mark_response_met("test".to_string());
        assert!(sla.is_response_met);
        assert!(sla.actual_response_time.is_some());

        // Check resolution SLA
        sla.check_resolution_sla();
        assert_eq!(sla.status, SLAStatus::Ok);
        assert!(sla.minutes_to_resolution_breach.unwrap() > 0);
    }
}
