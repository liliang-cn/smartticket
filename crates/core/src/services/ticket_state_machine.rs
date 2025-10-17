use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use tracing::{info, instrument, warn};
use uuid::Uuid;

use crate::models::ticket::*;
use smartticket_shared_error::{Result, SmartTicketError};

/// Ticket state machine for managing status transitions
pub struct TicketStateMachine {
    transition_rules: HashMap<TicketStatus, Vec<TicketStatusTransition>>,
}

impl TicketStateMachine {
    /// Create a new ticket state machine with default rules
    pub fn new() -> Self {
        let mut machine = Self {
            transition_rules: HashMap::new(),
        };
        machine.setup_default_transitions();
        machine
    }

    /// Setup default ticket status transition rules
    fn setup_default_transitions(&mut self) {
        // From Open
        self.add_transition(
            TicketStatus::Open,
            TicketStatus::InProgress,
            TransitionType::Automatic,
            "Ticket is being worked on".to_string(),
        );
        self.add_transition(
            TicketStatus::Open,
            TicketStatus::Closed,
            TransitionType::Manual,
            "Close immediately (e.g., duplicate, spam)".to_string(),
        );
        self.add_transition(
            TicketStatus::Open,
            TicketStatus::Resolved,
            TransitionType::Manual,
            "Quick resolution without assignment".to_string(),
        );

        // From InProgress
        self.add_transition(
            TicketStatus::InProgress,
            TicketStatus::PendingCustomer,
            TransitionType::Manual,
            "Waiting for customer response".to_string(),
        );
        self.add_transition(
            TicketStatus::InProgress,
            TicketStatus::PendingThirdParty,
            TransitionType::Manual,
            "Waiting for third-party response".to_string(),
        );
        self.add_transition(
            TicketStatus::InProgress,
            TicketStatus::Resolved,
            TransitionType::Manual,
            "Issue has been resolved".to_string(),
        );
        self.add_transition(
            TicketStatus::InProgress,
            TicketStatus::Closed,
            TransitionType::Manual,
            "Issue is closed".to_string(),
        );

        // From PendingCustomer
        self.add_transition(
            TicketStatus::PendingCustomer,
            TicketStatus::InProgress,
            TransitionType::Automatic,
            "Customer responded, continue work".to_string(),
        );
        self.add_transition(
            TicketStatus::PendingCustomer,
            TicketStatus::Resolved,
            TransitionType::Manual,
            "Resolved based on customer feedback".to_string(),
        );
        self.add_transition(
            TicketStatus::PendingCustomer,
            TicketStatus::Closed,
            TransitionType::Manual,
            "Closed due to inactivity".to_string(),
        );

        // From PendingThirdParty
        self.add_transition(
            TicketStatus::PendingThirdParty,
            TicketStatus::InProgress,
            TransitionType::Automatic,
            "Third-party responded, continue work".to_string(),
        );
        self.add_transition(
            TicketStatus::PendingThirdParty,
            TicketStatus::Resolved,
            TransitionType::Manual,
            "Resolved by third-party".to_string(),
        );
        self.add_transition(
            TicketStatus::PendingThirdParty,
            TicketStatus::Closed,
            TransitionType::Manual,
            "Closed by third-party".to_string(),
        );

        // From Resolved
        self.add_transition(
            TicketStatus::Resolved,
            TicketStatus::Closed,
            TransitionType::Automatic,
            "Auto-close after waiting period".to_string(),
        );
        self.add_transition(
            TicketStatus::Resolved,
            TicketStatus::Reopened,
            TransitionType::Manual,
            "Customer reopened the ticket".to_string(),
        );

        // From Closed
        self.add_transition(
            TicketStatus::Closed,
            TicketStatus::Reopened,
            TransitionType::Manual,
            "Ticket reopened by staff or customer".to_string(),
        );

        // From Reopened
        self.add_transition(
            TicketStatus::Reopened,
            TicketStatus::Open,
            TransitionType::Automatic,
            "Ticket reopened, back to queue".to_string(),
        );
        self.add_transition(
            TicketStatus::Reopened,
            TicketStatus::InProgress,
            TransitionType::Manual,
            "Immediately assigned and worked on".to_string(),
        );
    }

    /// Add a transition rule
    fn add_transition(
        &mut self,
        from_status: TicketStatus,
        to_status: TicketStatus,
        transition_type: TransitionType,
        description: String,
    ) {
        let transition = TicketStatusTransition {
            from_status,
            to_status,
            transition_type,
            description,
            conditions: Vec::new(),
            actions: Vec::new(),
        };

        self.transition_rules
            .entry(from_status)
            .or_insert_with(Vec::new)
            .push(transition);
    }

    /// Validate if a status transition is allowed
    #[instrument(skip(self))]
    pub fn validate_transition(
        &self,
        current_status: TicketStatus,
        target_status: TicketStatus,
        context: &TransitionContext,
    ) -> Result<TransitionValidation> {
        // Check if it's the same status (no-op)
        if current_status == target_status {
            return Ok(TransitionValidation {
                is_valid: true,
                reason: "No status change".to_string(),
                required_actions: Vec::new(),
                warnings: Vec::new(),
            });
        }

        // Check if transition exists
        if let Some(transitions) = self.transition_rules.get(&current_status) {
            for transition in transitions {
                if transition.to_status == target_status {
                    // Check conditions
                    let (conditions_met, condition_warnings) =
                        self.check_conditions(&transition.conditions, context);

                    if conditions_met {
                        let required_actions = transition.actions.clone();
                        let mut warnings = condition_warnings;

                        // Add contextual warnings
                        if target_status == TicketStatus::Closed
                            && context.sla_status == SLAStatus::Breached
                        {
                            warnings.push("Closing ticket with breached SLA".to_string());
                        }

                        if target_status == TicketStatus::Resolved && context.comments.is_empty() {
                            warnings
                                .push("Resolving ticket without resolution comment".to_string());
                        }

                        return Ok(TransitionValidation {
                            is_valid: true,
                            reason: transition.description.clone(),
                            required_actions,
                            warnings,
                        });
                    }
                }
            }
        }

        Err(SmartTicketError::Validation(format!(
            "Invalid status transition from {:?} to {:?}",
            current_status, target_status
        )))
    }

    /// Execute a status transition
    #[instrument(skip(self))]
    pub async fn execute_transition(
        &self,
        ticket: &mut Ticket,
        target_status: TicketStatus,
        context: &TransitionContext,
        executed_by: String,
    ) -> Result<TransitionResult> {
        // Validate transition
        let validation = self.validate_transition(ticket.status, target_status, context)?;

        if !validation.is_valid {
            return Err(SmartTicketError::Validation(validation.reason));
        }

        let old_status = ticket.status;
        let transition_time = Utc::now();

        // Execute the transition
        ticket.update_status(target_status, executed_by.clone())?;

        // Execute required actions
        let mut executed_actions = Vec::new();
        for action in validation.required_actions {
            match action {
                TransitionAction::AssignToAgent(agent_id) => {
                    if let Err(e) = ticket.assign_to_agent(agent_id, executed_by.clone()) {
                        warn!("Failed to assign ticket to agent: {}", e);
                    } else {
                        executed_actions.push(format!("Assigned to agent {}", agent_id));
                    }
                }
                TransitionAction::SetResolution(resolution) => {
                    ticket.set_resolution(resolution, executed_by.clone());
                    executed_actions.push("Resolution set".to_string());
                }
                TransitionAction::NotifyCustomer => {
                    // In real implementation, would send notification
                    executed_actions.push("Customer notified".to_string());
                }
                TransitionAction::UpdateSLA => {
                    // In real implementation, would update SLA
                    executed_actions.push("SLA updated".to_string());
                }
                TransitionAction::LogActivity(message) => {
                    // In real implementation, would log activity
                    executed_actions.push(format!("Activity logged: {}", message));
                }
                TransitionAction::Escalate(reason) => {
                    // In real implementation, would escalate ticket
                    executed_actions.push(format!("Ticket escalated: {}", reason));
                }
            }
        }

        info!(
            "Executed status transition for ticket {}: {:?} -> {:?}",
            ticket.id, old_status, target_status
        );

        Ok(TransitionResult {
            ticket_id: ticket.id,
            old_status,
            new_status: target_status,
            transition_time,
            executed_by,
            executed_actions,
            warnings: validation.warnings,
        })
    }

    /// Get possible transitions from current status
    pub fn get_possible_transitions(&self, current_status: TicketStatus) -> Vec<TicketStatus> {
        self.transition_rules
            .get(&current_status)
            .map_or(Vec::new(), |transitions| {
                transitions.iter().map(|t| t.to_status).collect()
            })
    }

    /// Get transition details
    pub fn get_transition_details(
        &self,
        current_status: TicketStatus,
        target_status: TicketStatus,
    ) -> Option<&TicketStatusTransition> {
        self.transition_rules
            .get(&current_status)
            .and_then(|transitions| transitions.iter().find(|t| t.to_status == target_status))
    }

    /// Check transition conditions
    fn check_conditions(
        &self,
        conditions: &[TransitionCondition],
        context: &TransitionContext,
    ) -> (bool, Vec<String>) {
        let mut all_met = true;
        let mut warnings = Vec::new();

        for condition in conditions {
            match condition {
                TransitionCondition::HasAssignee => {
                    if context.assigned_agent_id.is_none() {
                        all_met = false;
                        warnings.push("Ticket must be assigned to an agent".to_string());
                    }
                }
                TransitionCondition::HasResolution => {
                    if context.resolution.is_none() {
                        all_met = false;
                        warnings.push("Resolution must be provided".to_string());
                    }
                }
                TransitionCondition::HasComments => {
                    if context.comments.is_empty() {
                        all_met = false;
                        warnings.push("At least one comment is required".to_string());
                    }
                }
                TransitionCondition::CustomerNotified => {
                    if !context.customer_notified {
                        all_met = false;
                        warnings.push("Customer must be notified before closing".to_string());
                    }
                }
                TransitionCondition::SLANotBreached => {
                    if context.sla_status == SLAStatus::Breached {
                        warnings.push("SLA is breached".to_string());
                    }
                }
                TransitionCondition::AllAttachmentsProcessed => {
                    if !context.all_attachments_processed {
                        warnings.push("Some attachments are not processed".to_string());
                    }
                } // Removed CustomCheck for serialization compatibility
            }
        }

        (all_met, warnings)
    }

    /// Check if ticket is in a final state
    pub fn is_final_state(&self, status: TicketStatus) -> bool {
        matches!(status, TicketStatus::Closed)
    }

    /// Check if ticket can be assigned
    pub fn can_be_assigned(&self, status: TicketStatus) -> bool {
        matches!(status, TicketStatus::Open | TicketStatus::Reopened)
    }

    /// Get next auto-close time for resolved tickets
    pub fn get_auto_close_time(&self, resolved_at: DateTime<Utc>) -> DateTime<Utc> {
        // Auto-close resolved tickets after 7 days
        resolved_at + chrono::Duration::days(7)
    }

    /// Check if ticket should be auto-closed
    pub fn should_auto_close(&self, ticket: &Ticket) -> bool {
        if ticket.status != TicketStatus::Resolved {
            return false;
        }

        if let Some(resolved_at) = ticket.resolved_at {
            let auto_close_time = self.get_auto_close_time(resolved_at);
            Utc::now() > auto_close_time
        } else {
            false
        }
    }
}

impl Default for TicketStateMachine {
    fn default() -> Self {
        Self::new()
    }
}

/// Ticket status transition definition
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TicketStatusTransition {
    pub from_status: TicketStatus,
    pub to_status: TicketStatus,
    pub transition_type: TransitionType,
    pub description: String,
    pub conditions: Vec<TransitionCondition>,
    pub actions: Vec<TransitionAction>,
}

/// Transition type
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum TransitionType {
    /// Manual transition by user
    Manual,
    /// Automatic transition by system
    Automatic,
    /// Conditional transition based on rules
    Conditional,
}

/// Transition condition
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum TransitionCondition {
    /// Ticket must have an assignee
    HasAssignee,
    /// Resolution must be provided
    HasResolution,
    /// Comments must be present
    HasComments,
    /// Customer must be notified
    CustomerNotified,
    /// SLA must not be breached
    SLANotBreached,
    /// All attachments must be processed
    AllAttachmentsProcessed,
}

/// Transition action
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum TransitionAction {
    /// Assign ticket to specific agent
    AssignToAgent(Uuid),
    /// Set resolution text
    SetResolution(String),
    /// Notify customer
    NotifyCustomer,
    /// Update SLA records
    UpdateSLA,
    /// Log activity
    LogActivity(String),
    /// Escalate ticket
    Escalate(String),
}

/// Context for transition validation
#[derive(Debug, Clone)]
pub struct TransitionContext {
    pub tenant_id: Uuid,
    pub user_id: String,
    pub user_role: String,
    pub assigned_agent_id: Option<Uuid>,
    pub comments: Vec<String>,
    pub resolution: Option<String>,
    pub customer_notified: bool,
    pub sla_status: SLAStatus,
    pub all_attachments_processed: bool,
    pub additional_data: HashMap<String, String>,
}

/// Transition validation result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransitionValidation {
    pub is_valid: bool,
    pub reason: String,
    pub required_actions: Vec<TransitionAction>,
    pub warnings: Vec<String>,
}

/// Transition execution result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransitionResult {
    pub ticket_id: Uuid,
    pub old_status: TicketStatus,
    pub new_status: TicketStatus,
    pub transition_time: DateTime<Utc>,
    pub executed_by: String,
    pub executed_actions: Vec<String>,
    pub warnings: Vec<String>,
}

/// Ticket workflow configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TicketWorkflow {
    pub id: Uuid,
    pub tenant_id: Uuid,
    pub name: String,
    pub description: Option<String>,
    pub is_active: bool,
    pub priority_rules: Vec<PriorityRule>,
    pub assignment_rules: Vec<AssignmentRule>,
    pub notification_rules: Vec<NotificationRule>,
    pub escalation_rules: Vec<EscalationRule>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub created_by: String,
    pub updated_by: String,
}

/// Priority rule for automatic ticket prioritization
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PriorityRule {
    pub id: Uuid,
    pub workflow_id: Uuid,
    pub name: String,
    pub conditions: Vec<String>, // SQL WHERE conditions
    pub target_priority: TicketPriority,
    pub is_active: bool,
    pub sort_order: i32,
}

/// Assignment rule for automatic ticket assignment
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AssignmentRule {
    pub id: Uuid,
    pub workflow_id: Uuid,
    pub name: String,
    pub conditions: Vec<String>, // SQL WHERE conditions
    pub assignment_type: AssignmentType,
    pub target_agent_id: Option<Uuid>,
    pub target_team_id: Option<Uuid>,
    pub is_active: bool,
    pub sort_order: i32,
}

/// Assignment type
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum AssignmentType {
    /// Assign to specific agent
    Agent,
    /// Assign to specific team
    Team,
    /// Round-robin within team
    RoundRobin,
    /// Load-based assignment
    LoadBalanced,
    /// Skill-based assignment
    SkillBased,
}

/// Notification rule
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationRule {
    pub id: Uuid,
    pub workflow_id: Uuid,
    pub name: String,
    pub trigger_event: String,
    pub conditions: Vec<String>,
    pub recipients: Vec<String>,
    pub template_id: Uuid,
    pub is_active: bool,
}

/// Escalation rule
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EscalationRule {
    pub id: Uuid,
    pub workflow_id: Uuid,
    pub name: String,
    pub trigger_conditions: Vec<String>,
    pub escalation_type: EscalationType,
    pub target_agent_id: Option<Uuid>,
    pub target_team_id: Option<Uuid>,
    pub notify_manager: bool,
    pub is_active: bool,
}

/// Escalation type
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum EscalationType {
    /// Reassign to higher level agent
    Reassign,
    /// Notify manager
    NotifyManager,
    /// Create escalation ticket
    CreateEscalation,
    /// Auto-escalate to team lead
    AutoEscalate,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_state_machine_transitions() {
        let machine = TicketStateMachine::new();

        // Test valid transitions
        let context = TransitionContext {
            tenant_id: Uuid::new_v4(),
            user_id: "test_user".to_string(),
            user_role: "agent".to_string(),
            assigned_agent_id: Some(Uuid::new_v4()),
            comments: vec!["Test comment".to_string()],
            resolution: Some("Test resolution".to_string()),
            customer_notified: true,
            sla_status: SLAStatus::Ok,
            all_attachments_processed: true,
            additional_data: HashMap::new(),
        };

        // Valid: Open -> InProgress
        let result =
            machine.validate_transition(TicketStatus::Open, TicketStatus::InProgress, &context);
        assert!(result.is_ok());

        // Valid: InProgress -> Resolved
        let result =
            machine.validate_transition(TicketStatus::InProgress, TicketStatus::Resolved, &context);
        assert!(result.is_ok());

        // Invalid: Closed -> Open (must go through Reopened)
        let result =
            machine.validate_transition(TicketStatus::Closed, TicketStatus::Open, &context);
        assert!(result.is_err());
    }

    #[test]
    fn test_get_possible_transitions() {
        let machine = TicketStateMachine::new();

        // From Open status
        let transitions = machine.get_possible_transitions(TicketStatus::Open);
        assert!(transitions.contains(&TicketStatus::InProgress));
        assert!(transitions.contains(&TicketStatus::Resolved));
        assert!(transitions.contains(&TicketStatus::Closed));

        // From Closed status
        let transitions = machine.get_possible_transitions(TicketStatus::Closed);
        assert!(transitions.contains(&TicketStatus::Reopened));
        assert_eq!(transitions.len(), 1);
    }

    #[test]
    fn test_business_hours_calculation() {
        let machine = TicketStateMachine::new();
        let resolved_at = Utc::now();
        let auto_close_time = machine.get_auto_close_time(resolved_at);

        // Should be 7 days later
        let diff = auto_close_time - resolved_at;
        assert_eq!(diff.num_days(), 7);
    }

    #[test]
    fn test_should_auto_close() {
        let machine = TicketStateMachine::new();

        // Resolved ticket from 8 days ago should auto-close
        let mut ticket = Ticket {
            id: Uuid::new_v4(),
            tenant_id: Uuid::new_v4(),
            customer_id: Uuid::new_v4(),
            assigned_agent_id: None,
            team_id: None,
            title: "Test".to_string(),
            description: "Test".to_string(),
            status: TicketStatus::Resolved,
            priority: TicketPriority::Normal,
            ticket_type: TicketType::Incident,
            category_id: None,
            tags: vec![],
            created_at: Utc::now() - chrono::Duration::days(10),
            updated_at: Utc::now() - chrono::Duration::days(8),
            due_date: None,
            resolved_at: Some(Utc::now() - chrono::Duration::days(8)),
            closed_at: None,
            resolution: None,
            satisfaction_rating: None,
            external_reference: None,
            custom_fields: None,
            is_deleted: false,
            created_by: Some("test".to_string()),
            updated_by: Some("test".to_string()),
        };

        assert!(machine.should_auto_close(&ticket));

        // Recently resolved ticket should not auto-close
        ticket.resolved_at = Some(Utc::now());
        assert!(!machine.should_auto_close(&ticket));

        // Non-resolved ticket should not auto-close
        ticket.status = TicketStatus::Open;
        ticket.resolved_at = Some(Utc::now() - chrono::Duration::days(10));
        assert!(!machine.should_auto_close(&ticket));
    }
}
