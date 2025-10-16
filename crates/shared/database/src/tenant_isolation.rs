use smartticket_shared_error::{Result, SmartTicketError};
use sqlx::PgPool;
use uuid::Uuid;

#[derive(Debug, Clone)]
pub struct TenantContext {
    pub tenant_id: Uuid,
    pub user_id: Uuid,
    pub user_role: String,
}

impl TenantContext {
    pub fn new(tenant_id: Uuid, user_id: Uuid, user_role: String) -> Self {
        Self {
            tenant_id,
            user_id,
            user_role,
        }
    }

    pub fn is_admin(&self) -> bool {
        matches!(self.user_role.as_str(), "SuperAdmin" | "TenantAdmin")
    }

    pub fn is_support(&self) -> bool {
        matches!(
            self.user_role.as_str(),
            "SuperAdmin" | "TenantAdmin" | "SupportEngineer"
        )
    }

    pub fn can_access_tenant(&self, target_tenant_id: Uuid) -> bool {
        self.tenant_id == target_tenant_id || self.user_role == "SuperAdmin"
    }

    pub fn can_manage_users(&self) -> bool {
        matches!(self.user_role.as_str(), "SuperAdmin" | "TenantAdmin")
    }

    pub fn can_view_all_tickets(&self) -> bool {
        matches!(
            self.user_role.as_str(),
            "SuperAdmin" | "TenantAdmin" | "SupportEngineer"
        )
    }

    pub fn can_assign_tickets(&self) -> bool {
        matches!(
            self.user_role.as_str(),
            "SuperAdmin" | "TenantAdmin" | "SupportEngineer"
        )
    }

    pub fn can_manage_knowledge(&self) -> bool {
        matches!(
            self.user_role.as_str(),
            "SuperAdmin" | "TenantAdmin" | "SupportEngineer"
        )
    }
}

pub struct TenantIsolation;

impl TenantIsolation {
    pub fn verify_tenant_access(context: &TenantContext, resource_tenant_id: Uuid) -> Result<()> {
        if !context.can_access_tenant(resource_tenant_id) {
            return Err(SmartTicketError::TenantAccessDenied {
                tenant_id: resource_tenant_id,
                reason: "User does not have access to this tenant".to_string(),
            });
        }
        Ok(())
    }

    pub fn verify_user_management(context: &TenantContext) -> Result<()> {
        if !context.can_manage_users() {
            return Err(SmartTicketError::PermissionDenied(
                "User does not have permission to manage users".to_string(),
            ));
        }
        Ok(())
    }

    pub fn verify_ticket_access(
        context: &TenantContext,
        ticket_tenant_id: Uuid,
        ticket_contact_id: Option<Uuid>,
    ) -> Result<()> {
        // Check tenant access first
        Self::verify_tenant_access(context, ticket_tenant_id)?;

        // If not admin/support, can only access own tickets
        if !context.can_view_all_tickets() {
            if let Some(contact_id) = ticket_contact_id {
                if contact_id != context.user_id {
                    return Err(SmartTicketError::PermissionDenied(
                        "User can only access their own tickets".to_string(),
                    ));
                }
            } else {
                return Err(SmartTicketError::PermissionDenied(
                    "No contact information available for ticket".to_string(),
                ));
            }
        }

        Ok(())
    }

    pub fn verify_knowledge_access(
        context: &TenantContext,
        article_tenant_id: Uuid,
        article_visibility: &str,
    ) -> Result<()> {
        Self::verify_tenant_access(context, article_tenant_id)?;

        match article_visibility {
            "Public" => Ok(()),
            "Internal" | "Restricted" => {
                if context.can_view_all_tickets() {
                    Ok(())
                } else {
                    Err(SmartTicketError::PermissionDenied(
                        "User does not have permission to view internal knowledge articles"
                            .to_string(),
                    ))
                }
            }
            _ => Err(SmartTicketError::PermissionDenied(
                "Unknown knowledge article visibility level".to_string(),
            )),
        }
    }

    pub async fn verify_tenant_exists(pool: &PgPool, tenant_id: Uuid) -> Result<()> {
        let result = sqlx::query("SELECT id FROM tenants WHERE id = $1 AND is_active = true")
            .bind(tenant_id)
            .fetch_optional(pool)
            .await
            .map_err(|e| SmartTicketError::Database(e))?;

        if result.is_none() {
            return Err(SmartTicketError::TenantNotFound(tenant_id));
        }

        Ok(())
    }

    pub async fn verify_user_exists(pool: &PgPool, user_id: Uuid, tenant_id: Uuid) -> Result<bool> {
        let result = sqlx::query(
            "SELECT id FROM users WHERE id = $1 AND tenant_id = $2 AND is_active = true",
        )
        .bind(user_id)
        .bind(tenant_id)
        .fetch_optional(pool)
        .await
        .map_err(|e| SmartTicketError::Database(e))?;

        Ok(result.is_some())
    }

    pub fn apply_tenant_filter(query: &mut String, context: &TenantContext) {
        if context.user_role != "SuperAdmin" {
            query.push_str(&format!(" AND tenant_id = '{}' ", context.tenant_id));
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_tenant_context_permissions() {
        let admin = TenantContext::new(Uuid::new_v4(), Uuid::new_v4(), "TenantAdmin".to_string());
        assert!(admin.is_admin());
        assert!(admin.can_manage_users());
        assert!(admin.can_view_all_tickets());

        let support = TenantContext::new(
            Uuid::new_v4(),
            Uuid::new_v4(),
            "SupportEngineer".to_string(),
        );
        assert!(!support.is_admin());
        assert!(!support.can_manage_users());
        assert!(support.can_view_all_tickets());

        let customer =
            TenantContext::new(Uuid::new_v4(), Uuid::new_v4(), "CustomerUser".to_string());
        assert!(!customer.is_admin());
        assert!(!customer.can_manage_users());
        assert!(!customer.can_view_all_tickets());
    }

    #[test]
    fn test_tenant_access() {
        let tenant_id = Uuid::new_v4();
        let other_tenant_id = Uuid::new_v4();

        let context = TenantContext::new(tenant_id, Uuid::new_v4(), "CustomerUser".to_string());

        // Can access own tenant
        assert!(context.can_access_tenant(tenant_id));

        // Cannot access other tenant
        assert!(!context.can_access_tenant(other_tenant_id));

        // Super admin can access any tenant
        let super_admin =
            TenantContext::new(Uuid::new_v4(), Uuid::new_v4(), "SuperAdmin".to_string());
        assert!(super_admin.can_access_tenant(other_tenant_id));
    }
}
