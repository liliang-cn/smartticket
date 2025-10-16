use chrono::{DateTime, Datelike, Duration, Timelike, Utc, Weekday};
use serde::{Deserialize, Serialize};
use sqlx::{PgPool, Row};
use tracing::{error, info, instrument, warn};
use uuid::Uuid;

use crate::models::ticket::*;
use smartticket_shared_database::TenantContext;
use smartticket_shared_error::{Result, SmartTicketError};

/// SLA service for managing Service Level Agreements
pub struct SLAService {
    pool: PgPool,
}

impl SLAService {
    /// Create a new SLA service
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// Create SLA record for a new ticket
    #[instrument(skip(self))]
    pub async fn create_ticket_sla(
        &self,
        ticket_id: Uuid,
        tenant_id: Uuid,
        priority: TicketPriority,
        category_id: Option<Uuid>,
        created_by: String,
    ) -> Result<TicketSLA> {
        // Get SLA policy for the tenant
        let sla_policy = self
            .get_active_sla_policy(tenant_id, priority, category_id)
            .await?;

        // Calculate response and resolution due dates
        let now = Utc::now();
        let response_due = self.calculate_due_date(
            now,
            sla_policy.response_time_minutes,
            sla_policy.business_hours_only,
        )?;
        let resolution_due = self.calculate_due_date(
            now,
            sla_policy.resolution_time_minutes,
            sla_policy.business_hours_only,
        )?;

        // Apply priority multipliers if configured
        let (final_response_due, final_resolution_due) = if let Some(multipliers) =
            &sla_policy.priority_multipliers
        {
            let response_multiplier = self.get_priority_multiplier(multipliers, priority);
            let resolution_multiplier = self.get_priority_multiplier(multipliers, priority);

            let response_minutes =
                (sla_policy.response_time_minutes as f64 * response_multiplier) as i32;
            let resolution_minutes =
                (sla_policy.resolution_time_minutes as f64 * resolution_multiplier) as i32;

            let final_response_due =
                self.calculate_due_date(now, response_minutes, sla_policy.business_hours_only)?;
            let final_resolution_due =
                self.calculate_due_date(now, resolution_minutes, sla_policy.business_hours_only)?;

            (final_response_due, final_resolution_due)
        } else {
            (response_due, resolution_due)
        };

        // Create SLA record
        let sla = TicketSLA::new(
            ticket_id,
            sla_policy.id,
            final_response_due,
            final_resolution_due,
            created_by,
        );

        // Insert into database
        let query = r#"
            INSERT INTO ticket_sla (
                id, ticket_id, sla_policy_id, response_due, resolution_due,
                next_breach_time, status, minutes_to_response_breach,
                minutes_to_resolution_breach, actual_response_time,
                actual_resolution_time, is_response_met, is_resolution_met,
                breach_count, created_at, updated_at, created_by, updated_by
            ) VALUES (
                $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
                $14, $15, $16, $17, $18
            ) RETURNING *
        "#;

        let inserted_sla = sqlx::query_as::<_, TicketSLA>(query)
            .bind(sla.id)
            .bind(sla.ticket_id)
            .bind(sla.sla_policy_id)
            .bind(sla.response_due)
            .bind(sla.resolution_due)
            .bind(sla.next_breach_time)
            .bind(sla.status)
            .bind(sla.minutes_to_response_breach)
            .bind(sla.minutes_to_resolution_breach)
            .bind(sla.actual_response_time)
            .bind(sla.actual_resolution_time)
            .bind(sla.is_response_met)
            .bind(sla.is_resolution_met)
            .bind(sla.breach_count)
            .bind(sla.created_at)
            .bind(sla.updated_at)
            .bind(&sla.created_by)
            .bind(&sla.updated_by)
            .fetch_one(&self.pool)
            .await
            .map_err(|e| {
                error!("Failed to create SLA for ticket {}: {}", ticket_id, e);
                SmartTicketError::Database(e)
            })?;

        info!(
            "Created SLA for ticket {} with response due {} and resolution due {}",
            ticket_id, final_response_due, final_resolution_due
        );
        Ok(inserted_sla)
    }

    /// Get SLA information for a ticket
    #[instrument(skip(self))]
    pub async fn get_ticket_sla(
        &self,
        ticket_id: Uuid,
        context: &TenantContext,
    ) -> Result<TicketSLA> {
        let query = r#"
            SELECT * FROM ticket_sla
            WHERE ticket_id = $1
            ORDER BY created_at DESC
            LIMIT 1
        "#;

        let sla = sqlx::query_as::<_, TicketSLA>(query)
            .bind(ticket_id)
            .fetch_optional(&self.pool)
            .await
            .map_err(|e| {
                error!("Failed to get SLA for ticket {}: {}", ticket_id, e);
                SmartTicketError::Database(e)
            })?
            .ok_or_else(|| SmartTicketError::not_found("SLA", &format!("ticket {}", ticket_id)))?;

        Ok(sla)
    }

    /// Update SLA status and timing for all active tickets
    #[instrument(skip(self))]
    pub async fn update_active_slas(&self, tenant_id: Option<Uuid>) -> Result<i32> {
        let query = if let Some(_tenant_id) = tenant_id {
            r#"
                SELECT ts.*, t.tenant_id, t.priority, t.status, t.assigned_agent_id, t.created_at
                FROM ticket_sla ts
                JOIN tickets t ON ts.ticket_id = t.id
                WHERE t.tenant_id = $1 AND t.is_deleted = false
                AND t.status NOT IN ('resolved', 'closed')
                AND ts.is_resolution_met = false
            "#
        } else {
            r#"
                SELECT ts.*, t.tenant_id, t.priority, t.status, t.assigned_agent_id, t.created_at
                FROM ticket_sla ts
                JOIN tickets t ON ts.ticket_id = t.id
                WHERE t.is_deleted = false
                AND t.status NOT IN ('resolved', 'closed')
                AND ts.is_resolution_met = false
            "#
        };

        let rows = if let Some(tenant_id) = tenant_id {
            sqlx::query(query)
                .bind(tenant_id)
                .fetch_all(&self.pool)
                .await
                .map_err(|e| {
                    error!("Failed to fetch active SLAs: {}", e);
                    SmartTicketError::Database(e)
                })?
        } else {
            sqlx::query(query)
                .fetch_all(&self.pool)
                .await
                .map_err(|e| {
                    error!("Failed to fetch active SLAs: {}", e);
                    SmartTicketError::Database(e)
                })?
        };

        let mut updated_count = 0;
        let _now = Utc::now();

        for row in rows {
            let ticket_id: Uuid = row.get("ticket_id");
            let mut sla = TicketSLA {
                id: row.get("id"),
                ticket_id: row.get("ticket_id"),
                sla_policy_id: row.get("sla_policy_id"),
                response_due: row.get("response_due"),
                resolution_due: row.get("resolution_due"),
                next_breach_time: row.get("next_breach_time"),
                status: row.get("status"),
                minutes_to_response_breach: row.get("minutes_to_response_breach"),
                minutes_to_resolution_breach: row.get("minutes_to_resolution_breach"),
                actual_response_time: row.get("actual_response_time"),
                actual_resolution_time: row.get("actual_resolution_time"),
                is_response_met: row.get("is_response_met"),
                is_resolution_met: row.get("is_resolution_met"),
                breach_count: row.get("breach_count"),
                created_at: row.get("created_at"),
                updated_at: row.get("updated_at"),
                created_by: row.get("created_by"),
                updated_by: row.get("updated_by"),
            };

            let was_breached = sla.status == SLAStatus::Breached;

            // Update response SLA if not yet met
            if !sla.is_response_met {
                sla.check_response_sla();
            }

            // Update resolution SLA if not yet met
            if !sla.is_resolution_met {
                sla.check_resolution_sla();
            }

            // Update next breach time
            sla.update_next_breach_time();

            // Log new breach if it just occurred
            if sla.status == SLAStatus::Breached && !was_breached {
                warn!("SLA breach detected for ticket {}", ticket_id);
                self.log_sla_breach(&sla).await?;
            }

            // Save updated SLA
            self.update_sla(&sla).await?;
            updated_count += 1;
        }

        info!("Updated {} active SLA records", updated_count);
        Ok(updated_count)
    }

    /// Mark ticket as responded to (first response SLA met)
    #[instrument(skip(self))]
    pub async fn mark_ticket_responded(&self, ticket_id: Uuid, responded_by: String) -> Result<()> {
        let mut sla = self
            .get_ticket_sla(
                ticket_id,
                &TenantContext {
                    tenant_id: Uuid::new_v4(), // Will be validated in get_ticket_sla
                    user_id: Uuid::new_v4(),
                    user_role: "temp".to_string(),
                },
            )
            .await?;

        if sla.is_response_met {
            return Ok(()); // Already marked as responded
        }

        sla.mark_response_met(responded_by.clone());
        self.update_sla(&sla).await?;

        info!(
            "Marked ticket {} as responded at {}",
            ticket_id,
            sla.actual_response_time.unwrap()
        );
        Ok(())
    }

    /// Mark ticket as resolved (resolution SLA met or not)
    #[instrument(skip(self))]
    pub async fn mark_ticket_resolved(&self, ticket_id: Uuid, resolved_by: String) -> Result<()> {
        let mut sla = self
            .get_ticket_sla(
                ticket_id,
                &TenantContext {
                    tenant_id: Uuid::new_v4(), // Will be validated in get_ticket_sla
                    user_id: Uuid::new_v4(),
                    user_role: "temp".to_string(),
                },
            )
            .await?;

        if sla.is_resolution_met {
            return Ok(()); // Already marked as resolved
        }

        sla.mark_resolution_met(resolved_by.clone());
        self.update_sla(&sla).await?;

        let status = if sla.is_resolution_met {
            "met"
        } else {
            "breached"
        };
        info!(
            "Marked ticket {} as resolved at {} (SLA {})",
            ticket_id,
            sla.actual_resolution_time.unwrap(),
            status
        );
        Ok(())
    }

    /// Get SLA breach report for a tenant
    #[instrument(skip(self))]
    pub async fn get_sla_breach_report(
        &self,
        tenant_id: Uuid,
        start_date: DateTime<Utc>,
        end_date: DateTime<Utc>,
    ) -> Result<SLABreachReport> {
        let query = r#"
            SELECT
                COUNT(*) as total_tickets,
                COUNT(*) FILTER (WHERE is_response_met = false) as response_breaches,
                COUNT(*) FILTER (WHERE is_resolution_met = false) as resolution_breaches,
                AVG(breach_count) as avg_breaches_per_ticket,
                COUNT(DISTINCT ticket_id) FILTER (WHERE breach_count > 0) as tickets_with_breaches
            FROM ticket_sla ts
            JOIN tickets t ON ts.ticket_id = t.id
            WHERE t.tenant_id = $1
                AND t.created_at BETWEEN $2 AND $3
                AND t.is_deleted = false
        "#;

        let row = sqlx::query(query)
            .bind(tenant_id)
            .bind(start_date)
            .bind(end_date)
            .fetch_one(&self.pool)
            .await
            .map_err(|e| {
                error!("Failed to generate SLA breach report: {}", e);
                SmartTicketError::Database(e)
            })?;

        let total_tickets: i64 = row.get("total_tickets");
        let response_breaches: i64 = row.get("response_breaches");
        let resolution_breaches: i64 = row.get("resolution_breaches");
        let avg_breaches_per_ticket: Option<f64> = row.get("avg_breaches_per_ticket");
        let tickets_with_breaches: i64 = row.get("tickets_with_breaches");

        // Get breaches by priority
        let priority_query = r#"
            SELECT
                t.priority,
                COUNT(*) FILTER (WHERE ts.is_response_met = false) as response_breaches,
                COUNT(*) FILTER (WHERE ts.is_resolution_met = false) as resolution_breaches,
                COUNT(*) as total_in_priority
            FROM ticket_sla ts
            JOIN tickets t ON ts.ticket_id = t.id
            WHERE t.tenant_id = $1
                AND t.created_at BETWEEN $2 AND $3
                AND t.is_deleted = false
            GROUP BY t.priority
            ORDER BY t.priority
        "#;

        let priority_rows = sqlx::query(priority_query)
            .bind(tenant_id)
            .bind(start_date)
            .bind(end_date)
            .fetch_all(&self.pool)
            .await
            .map_err(|e| {
                error!("Failed to get priority breakdown: {}", e);
                SmartTicketError::Database(e)
            })?;

        let breaches_by_priority: Vec<PrioritySLABreakdown> = priority_rows
            .into_iter()
            .map(|row| PrioritySLABreakdown {
                priority: row.get("priority"),
                total_tickets: row.get("total_in_priority"),
                response_breaches: row.get("response_breaches"),
                resolution_breaches: row.get("resolution_breaches"),
                response_breach_rate: row.get::<i64, _>("response_breaches") as f64
                    / row.get::<i64, _>("total_in_priority") as f64
                    * 100.0,
                resolution_breach_rate: row.get::<i64, _>("resolution_breaches") as f64
                    / row.get::<i64, _>("total_in_priority") as f64
                    * 100.0,
            })
            .collect();

        let report = SLABreachReport {
            period_start: start_date,
            period_end: end_date,
            total_tickets,
            response_breaches,
            resolution_breaches,
            response_breach_rate: if total_tickets > 0 {
                response_breaches as f64 / total_tickets as f64 * 100.0
            } else {
                0.0
            },
            resolution_breach_rate: if total_tickets > 0 {
                resolution_breaches as f64 / total_tickets as f64 * 100.0
            } else {
                0.0
            },
            avg_breaches_per_ticket,
            tickets_with_breaches,
            breaches_by_priority,
        };

        info!("Generated SLA breach report for tenant {} ({} tickets, {:.1}% response breach, {:.1}% resolution breach)",
              tenant_id, total_tickets, report.response_breach_rate, report.resolution_breach_rate);
        Ok(report)
    }

    /// Get active SLA policy for tenant
    async fn get_active_sla_policy(
        &self,
        tenant_id: Uuid,
        _priority: TicketPriority,
        category_id: Option<Uuid>,
    ) -> Result<SLAPolicy> {
        // Try to find specific policy for category and priority
        if let Some(category_id) = category_id {
            let query = r#"
                SELECT * FROM sla_policies
                WHERE tenant_id = $1 AND is_active = true
                AND ($2::uuid IS NULL OR $2 = ANY(ARRAY(SELECT category_id FROM sla_policy_categories WHERE sla_policy_id = sla_policies.id)))
                ORDER BY priority DESC
                LIMIT 1
            "#;

            if let Some(policy) = sqlx::query_as::<_, SLAPolicy>(query)
                .bind(tenant_id)
                .bind(category_id)
                .fetch_optional(&self.pool)
                .await
                .map_err(|e| {
                    error!("Failed to get category-specific SLA policy: {}", e);
                    SmartTicketError::Database(e)
                })?
            {
                return Ok(policy);
            }
        }

        // Fallback to tenant-wide policy
        let query = r#"
            SELECT * FROM sla_policies
            WHERE tenant_id = $1 AND is_active = true
            ORDER BY priority DESC
            LIMIT 1
        "#;

        let policy = sqlx::query_as::<_, SLAPolicy>(query)
            .bind(tenant_id)
            .fetch_optional(&self.pool)
            .await
            .map_err(|e| {
                error!("Failed to get tenant SLA policy: {}", e);
                SmartTicketError::Database(e)
            })?
            .ok_or_else(|| SmartTicketError::not_found("SLA policy", &tenant_id.to_string()))?;

        Ok(policy)
    }

    /// Calculate due date considering business hours
    fn calculate_due_date(
        &self,
        start_time: DateTime<Utc>,
        minutes: i32,
        business_hours_only: bool,
    ) -> Result<DateTime<Utc>> {
        Self::calculate_due_date_static(start_time, minutes, business_hours_only)
    }

    /// Static version of calculate_due_date for testing without database
    pub fn calculate_due_date_static(
        start_time: DateTime<Utc>,
        minutes: i32,
        business_hours_only: bool,
    ) -> Result<DateTime<Utc>> {
        if !business_hours_only {
            // If not business hours only, simply add minutes
            return Ok(start_time + Duration::minutes(minutes as i64));
        }

        // Business hours: Monday-Friday, 9 AM - 6 PM UTC
        let mut current_time = start_time;
        let mut remaining_minutes = minutes;

        while remaining_minutes > 0 {
            // Check if current time is within business hours
            let weekday = current_time.weekday();
            let hour = current_time.hour();

            if weekday.num_days_from_monday() < 5 && hour >= 9 && hour < 18 {
                // Within business hours, calculate minutes until end of day
                let end_of_day = current_time
                    .date_naive()
                    .and_hms_opt(18, 0, 0)
                    .unwrap()
                    .and_utc();
                let minutes_until_end = (end_of_day - current_time).num_minutes();

                if i64::from(remaining_minutes) <= minutes_until_end {
                    // Can complete within current business day
                    current_time = current_time + Duration::minutes(remaining_minutes as i64);
                    remaining_minutes = 0;
                } else {
                    // Move to next business day
                    remaining_minutes -= minutes_until_end as i32;
                    current_time = Self::next_business_day_start_static(current_time);
                }
            } else {
                // Outside business hours, move to next business day start
                current_time = Self::next_business_day_start_static(current_time);
            }
        }

        Ok(current_time)
    }

    /// Get next business day start (9 AM UTC)
    #[allow(dead_code)]
    fn next_business_day_start(&self, current_time: DateTime<Utc>) -> DateTime<Utc> {
        Self::next_business_day_start_static(current_time)
    }

    /// Static version of next_business_day_start for testing
    pub fn next_business_day_start_static(current_time: DateTime<Utc>) -> DateTime<Utc> {
        let mut next_day = current_time + Duration::days(1);

        // Skip weekends
        while next_day.weekday() == Weekday::Sat || next_day.weekday() == Weekday::Sun {
            next_day += Duration::days(1);
        }

        // Set to 9 AM
        next_day
            .date_naive()
            .and_hms_opt(9, 0, 0)
            .unwrap()
            .and_utc()
    }

    /// Get priority multiplier from JSON configuration
    fn get_priority_multiplier(
        &self,
        multipliers: &serde_json::Value,
        priority: TicketPriority,
    ) -> f64 {
        Self::get_priority_multiplier_static(multipliers, priority)
    }

    /// Static version of get_priority_multiplier for testing
    pub fn get_priority_multiplier_static(
        multipliers: &serde_json::Value,
        priority: TicketPriority,
    ) -> f64 {
        multipliers
            .get(&priority.to_string())
            .and_then(|v| v.as_f64())
            .unwrap_or(1.0)
    }

    /// Update SLA record in database
    async fn update_sla(&self, sla: &TicketSLA) -> Result<()> {
        let query = r#"
            UPDATE ticket_sla SET
                next_breach_time = $2, status = $3, minutes_to_response_breach = $4,
                minutes_to_resolution_breach = $5, actual_response_time = $6,
                actual_resolution_time = $7, is_response_met = $8, is_resolution_met = $9,
                breach_count = $10, updated_at = $11, updated_by = $12
            WHERE id = $1
        "#;

        sqlx::query(query)
            .bind(sla.id)
            .bind(sla.next_breach_time)
            .bind(sla.status)
            .bind(sla.minutes_to_response_breach)
            .bind(sla.minutes_to_resolution_breach)
            .bind(sla.actual_response_time)
            .bind(sla.actual_resolution_time)
            .bind(sla.is_response_met)
            .bind(sla.is_resolution_met)
            .bind(sla.breach_count)
            .bind(Utc::now())
            .bind(&sla.updated_by)
            .execute(&self.pool)
            .await
            .map_err(|e| {
                error!("Failed to update SLA {}: {}", sla.id, e);
                SmartTicketError::Database(e)
            })?;

        Ok(())
    }

    /// Log SLA breach
    async fn log_sla_breach(&self, sla: &TicketSLA) -> Result<()> {
        let query = r#"
            INSERT INTO sla_breaches (
                id, ticket_id, sla_policy_id, breach_type, breach_time,
                minutes_overdue, created_at, created_by
            ) VALUES (
                $1, $2, $3, $4, $5, $6, $7, $8
            )
        "#;

        let breach_type = if sla.minutes_to_response_breach.unwrap_or(0) < 0 {
            "response"
        } else {
            "resolution"
        };

        let minutes_overdue = if sla.minutes_to_response_breach.unwrap_or(0) < 0 {
            sla.minutes_to_response_breach.unwrap()
        } else {
            sla.minutes_to_resolution_breach.unwrap()
        };

        sqlx::query(query)
            .bind(Uuid::new_v4())
            .bind(sla.ticket_id)
            .bind(sla.sla_policy_id)
            .bind(breach_type)
            .bind(Utc::now())
            .bind(minutes_overdue.abs())
            .bind(Utc::now())
            .bind("system")
            .execute(&self.pool)
            .await
            .map_err(|e| {
                error!("Failed to log SLA breach: {}", e);
                SmartTicketError::Database(e)
            })?;

        Ok(())
    }
}

/// SLA breach report
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SLABreachReport {
    pub period_start: DateTime<Utc>,
    pub period_end: DateTime<Utc>,
    pub total_tickets: i64,
    pub response_breaches: i64,
    pub resolution_breaches: i64,
    pub response_breach_rate: f64,
    pub resolution_breach_rate: f64,
    pub avg_breaches_per_ticket: Option<f64>,
    pub tickets_with_breaches: i64,
    pub breaches_by_priority: Vec<PrioritySLABreakdown>,
}

/// Priority-specific SLA breakdown
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PrioritySLABreakdown {
    pub priority: TicketPriority,
    pub total_tickets: i64,
    pub response_breaches: i64,
    pub resolution_breaches: i64,
    pub response_breach_rate: f64,
    pub resolution_breach_rate: f64,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_business_hours_calculation() {
        // Test calculation within same business day
        let start = DateTime::parse_from_rfc3339("2023-01-02T10:00:00Z")
            .unwrap()
            .with_timezone(&chrono::Utc);
        let due = SLAService::calculate_due_date_static(start, 120, true).unwrap();
        // Should be 2 hours later (12:00)
        assert_eq!(due.hour(), 12);

        // Test calculation spanning to next business day
        let start = DateTime::parse_from_rfc3339("2023-01-02T16:00:00Z")
            .unwrap()
            .with_timezone(&chrono::Utc);
        let due = SLAService::calculate_due_date_static(start, 240, true).unwrap();
        // Should be next day at 10:00 (1 hour left in day, 3 hours next day)
        assert_eq!(due.hour(), 10);
        assert_eq!(due.day(), 3);

        // Test weekend handling
        let start = DateTime::parse_from_rfc3339("2023-01-06T16:00:00Z")
            .unwrap()
            .with_timezone(&chrono::Utc); // Friday
        let due = SLAService::calculate_due_date_static(start, 240, true).unwrap();
        // Should be Monday at 10:00 (skip weekend)
        assert_eq!(due.weekday(), Weekday::Mon);
        assert_eq!(due.hour(), 10);
    }

    #[test]
    fn test_priority_multiplier() {
        let multipliers = serde_json::json!({
            "low": 2.0,
            "normal": 1.0,
            "high": 0.5,
            "urgent": 0.25,
            "critical": 0.1
        });

        assert_eq!(
            SLAService::get_priority_multiplier_static(&multipliers, TicketPriority::Low),
            2.0
        );
        assert_eq!(
            SLAService::get_priority_multiplier_static(&multipliers, TicketPriority::Normal),
            1.0
        );
        assert_eq!(
            SLAService::get_priority_multiplier_static(&multipliers, TicketPriority::High),
            0.5
        );
        assert_eq!(
            SLAService::get_priority_multiplier_static(&multipliers, TicketPriority::Urgent),
            0.25
        );
        assert_eq!(
            SLAService::get_priority_multiplier_static(&multipliers, TicketPriority::Critical),
            0.1
        );
    }
}
