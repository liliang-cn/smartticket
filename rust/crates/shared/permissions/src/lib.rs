use serde::{Deserialize, Serialize};
use std::collections::{HashMap, HashSet};
use smartticket_shared_auth::TenantContext;
use smartticket_shared_error::{SmartTicketError, Result};
use uuid::Uuid;

#[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub enum Resource {
    Ticket,
    KnowledgeArticle,
    User,
    Tenant,
    SLAPolicy,
    AuditLog,
    Configuration,
    APIKey,
}

#[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub enum Action {
    Read,
    Write,
    Create,
    Update,
    Delete,
    Admin,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Permission {
    pub resource: Resource,
    pub action: Action,
    pub conditions: Option<PermissionCondition>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PermissionCondition {
    pub tenant_owned: bool,
    pub department_filter: Option<String>,
    pub priority_filter: Option<String>,
}

#[derive(Debug, Clone)]
pub struct Role {
    pub name: String,
    pub permissions: HashSet<Permission>,
    pub is_system_role: bool,
}

impl Role {
    pub fn new(name: String, permissions: Vec<Permission>, is_system_role: bool) -> Self {
        Self {
            name,
            permissions: permissions.into_iter().collect(),
            is_system_role,
        }
    }

    pub fn has_permission(&self, permission: &Permission) -> bool {
        self.permissions.contains(permission)
    }
}

pub struct PermissionService {
    roles: HashMap<String, Role>,
}

impl PermissionService {
    pub fn new() -> Self {
        let mut service = Self {
            roles: HashMap::new(),
        };

        service.initialize_default_roles();
        service
    }

    /// Initialize default system roles
    fn initialize_default_roles(&mut self) {
        // Super Admin - full access to everything
        let super_admin_permissions = vec![
            Permission { resource: Resource::Ticket, action: Action::Admin, conditions: None },
            Permission { resource: Resource::KnowledgeArticle, action: Action::Admin, conditions: None },
            Permission { resource: Resource::User, action: Action::Admin, conditions: None },
            Permission { resource: Resource::Tenant, action: Action::Admin, conditions: None },
            Permission { resource: Resource::SLAPolicy, action: Action::Admin, conditions: None },
            Permission { resource: Resource::AuditLog, action: Action::Read, conditions: None },
            Permission { resource: Resource::Configuration, action: Action::Admin, conditions: None },
            Permission { resource: Resource::APIKey, action: Action::Admin, conditions: None },
        ];

        let super_admin_role = Role::new(
            "SuperAdmin".to_string(),
            super_admin_permissions,
            true,
        );

        // Tenant Admin - full access within their tenant
        let tenant_admin_permissions = vec![
            Permission {
                resource: Resource::Ticket,
                action: Action::Admin,
                conditions: Some(PermissionCondition {
                    tenant_owned: true,
                    department_filter: None,
                    priority_filter: None,
                }),
            },
            Permission {
                resource: Resource::KnowledgeArticle,
                action: Action::Admin,
                conditions: Some(PermissionCondition {
                    tenant_owned: true,
                    department_filter: None,
                    priority_filter: None,
                }),
            },
            Permission {
                resource: Resource::User,
                action: Action::Admin,
                conditions: Some(PermissionCondition {
                    tenant_owned: true,
                    department_filter: None,
                    priority_filter: None,
                }),
            },
            Permission {
                resource: Resource::Tenant,
                action: Action::Read,
                conditions: Some(PermissionCondition {
                    tenant_owned: true,
                    department_filter: None,
                    priority_filter: None,
                }),
            },
            Permission {
                resource: Resource::SLAPolicy,
                action: Action::Admin,
                conditions: Some(PermissionCondition {
                    tenant_owned: true,
                    department_filter: None,
                    priority_filter: None,
                }),
            },
            Permission {
                resource: Resource::AuditLog,
                action: Action::Read,
                conditions: Some(PermissionCondition {
                    tenant_owned: true,
                    department_filter: None,
                    priority_filter: None,
                }),
            },
            Permission {
                resource: Resource::APIKey,
                action: Action::Admin,
                conditions: Some(PermissionCondition {
                    tenant_owned: true,
                    department_filter: None,
                    priority_filter: None,
                }),
            },
        ];

        let tenant_admin_role = Role::new(
            "TenantAdmin".to_string(),
            tenant_admin_permissions,
            true,
        );

        // Support Engineer - access to tickets and knowledge within tenant
        let support_permissions = vec![
            Permission {
                resource: Resource::Ticket,
                action: Action::Read,
                conditions: Some(PermissionCondition {
                    tenant_owned: true,
                    department_filter: None,
                    priority_filter: None,
                }),
            },
            Permission {
                resource: Resource::Ticket,
                action: Action::Write,
                conditions: Some(PermissionCondition {
                    tenant_owned: true,
                    department_filter: None,
                    priority_filter: None,
                }),
            },
            Permission {
                resource: Resource::Ticket,
                action: Action::Create,
                conditions: Some(PermissionCondition {
                    tenant_owned: true,
                    department_filter: None,
                    priority_filter: None,
                }),
            },
            Permission {
                resource: Resource::Ticket,
                action: Action::Update,
                conditions: Some(PermissionCondition {
                    tenant_owned: true,
                    department_filter: None,
                    priority_filter: None,
                }),
            },
            Permission {
                resource: Resource::KnowledgeArticle,
                action: Action::Read,
                conditions: Some(PermissionCondition {
                    tenant_owned: true,
                    department_filter: None,
                    priority_filter: None,
                }),
            },
            Permission {
                resource: Resource::KnowledgeArticle,
                action: Action::Write,
                conditions: Some(PermissionCondition {
                    tenant_owned: true,
                    department_filter: None,
                    priority_filter: None,
                }),
            },
            Permission {
                resource: Resource::KnowledgeArticle,
                action: Action::Create,
                conditions: Some(PermissionCondition {
                    tenant_owned: true,
                    department_filter: None,
                    priority_filter: None,
                }),
            },
            Permission {
                resource: Resource::KnowledgeArticle,
                action: Action::Update,
                conditions: Some(PermissionCondition {
                    tenant_owned: true,
                    department_filter: None,
                    priority_filter: None,
                }),
            },
        ];

        let support_role = Role::new(
            "SupportEngineer".to_string(),
            support_permissions,
            true,
        );

        // Customer User - limited access to their own tickets and public knowledge
        let customer_permissions = vec![
            Permission {
                resource: Resource::Ticket,
                action: Action::Read,
                conditions: Some(PermissionCondition {
                    tenant_owned: true,
                    department_filter: None,
                    priority_filter: None,
                }),
            },
            Permission {
                resource: Resource::Ticket,
                action: Action::Create,
                conditions: Some(PermissionCondition {
                    tenant_owned: true,
                    department_filter: None,
                    priority_filter: None,
                }),
            },
            Permission {
                resource: Resource::Ticket,
                action: Action::Update,
                conditions: Some(PermissionCondition {
                    tenant_owned: true,
                    department_filter: None,
                    priority_filter: None,
                }),
            },
            Permission {
                resource: Resource::KnowledgeArticle,
                action: Action::Read,
                conditions: Some(PermissionCondition {
                    tenant_owned: true,
                    department_filter: None,
                    priority_filter: None,
                }),
            },
        ];

        let customer_role = Role::new(
            "CustomerUser".to_string(),
            customer_permissions,
            true,
        );

        // Sales - read-only access to tickets and knowledge
        let sales_permissions = vec![
            Permission {
                resource: Resource::Ticket,
                action: Action::Read,
                conditions: Some(PermissionCondition {
                    tenant_owned: true,
                    department_filter: None,
                    priority_filter: None,
                }),
            },
            Permission {
                resource: Resource::KnowledgeArticle,
                action: Action::Read,
                conditions: Some(PermissionCondition {
                    tenant_owned: true,
                    department_filter: None,
                    priority_filter: None,
                }),
            },
            Permission {
                resource: Resource::User,
                action: Action::Read,
                conditions: Some(PermissionCondition {
                    tenant_owned: true,
                    department_filter: None,
                    priority_filter: None,
                }),
            },
        ];

        let sales_role = Role::new(
            "Sales".to_string(),
            sales_permissions,
            true,
        );

        self.roles.insert("SuperAdmin".to_string(), super_admin_role);
        self.roles.insert("TenantAdmin".to_string(), tenant_admin_role);
        self.roles.insert("SupportEngineer".to_string(), support_role);
        self.roles.insert("CustomerUser".to_string(), customer_role);
        self.roles.insert("Sales".to_string(), sales_role);
    }

    /// Get role by name
    pub fn get_role(&self, role_name: &str) -> Option<&Role> {
        self.roles.get(role_name)
    }

    /// Check if a role has a specific permission
    pub fn role_has_permission(&self, role_name: &str, permission: &Permission) -> bool {
        if let Some(role) = self.get_role(role_name) {
            role.has_permission(permission)
        } else {
            false
        }
    }

    /// Check if user has permission based on tenant context
    pub fn check_permission(
        &self,
        context: &TenantContext,
        resource: Resource,
        action: Action,
        resource_tenant_id: Option<Uuid>,
        resource_owner_id: Option<Uuid>,
    ) -> Result<()> {
        // Get user's role permissions
        let permission = Permission {
            resource,
            action,
            conditions: Some(PermissionCondition {
                tenant_owned: true,
                department_filter: None,
                priority_filter: None,
            }),
        };

        // Check if user has the permission
        if !self.check_user_permission(context, &permission) {
            return Err(SmartTicketError::PermissionDenied(format!(
                "User {} does not have permission to perform {:?} on {:?}",
                context.user_id, action, resource
            )));
        }

        // Check tenant access
        if let Some(resource_tenant) = resource_tenant_id {
            let resource_tenant_str = resource_tenant.to_string();
            if !context.can_access_tenant(&resource_tenant_str) {
                return Err(SmartTicketError::TenantAccessDenied {
                    tenant_id: resource_tenant,
                    reason: "User does not have access to this tenant".to_string(),
                });
            }
        }

        // Check ownership conditions if required
        if !context.is_admin() {
            if let Some(owner_id) = resource_owner_id {
                if owner_id.to_string() != context.user_id {
                    // Check if user can access resources they don't own
                    if !self.can_access_foreign_resource(context, &permission) {
                        return Err(SmartTicketError::PermissionDenied(
                            "User can only access their own resources".to_string()
                        ));
                    }
                }
            }
        }

        Ok(())
    }

    /// Check if user has specific permission
    fn check_user_permission(&self, context: &TenantContext, permission: &Permission) -> bool {
        // Super admins have all permissions
        if context.user_role == "SuperAdmin" {
            return true;
        }

        // Check role-based permissions
        if self.role_has_permission(&context.user_role, permission) {
            return true;
        }

        // Check individual permissions from JWT
        let permission_string = format!("{:?}:{:?}", permission.resource, permission.action);
        context.has_permission(&permission_string)
    }

    /// Check if user can access resources they don't own
    fn can_access_foreign_resource(&self, context: &TenantContext, permission: &Permission) -> bool {
        match context.user_role.as_str() {
            "SuperAdmin" | "TenantAdmin" | "SupportEngineer" => {
                // Support staff can access most resources within their tenant
                matches!(
                    permission.resource,
                    Resource::Ticket | Resource::KnowledgeArticle | Resource::User
                )
            }
            "Sales" => {
                // Sales can only read resources
                matches!(permission.action, Action::Read)
            }
            _ => false,
        }
    }

    /// Add custom role
    pub fn add_role(&mut self, role: Role) -> Result<()> {
        if role.is_system_role {
            return Err(SmartTicketError::PermissionDenied(
                "Cannot modify system roles".to_string()
            ));
        }

        if self.roles.contains_key(&role.name) {
            return Err(SmartTicketError::Conflict(format!(
                "Role '{}' already exists",
                role.name
            )));
        }

        self.roles.insert(role.name.clone(), role);
        Ok(())
    }

    /// Update custom role
    pub fn update_role(&mut self, role_name: &str, new_permissions: Vec<Permission>) -> Result<()> {
        let role = self.roles.get_mut(role_name)
            .ok_or_else(|| SmartTicketError::NotFound {
                entity: "Role".to_string(),
                id: role_name.to_string(),
            })?;

        if role.is_system_role {
            return Err(SmartTicketError::PermissionDenied(
                "Cannot modify system roles".to_string()
            ));
        }

        role.permissions = new_permissions.into_iter().collect();
        Ok(())
    }

    /// Delete custom role
    pub fn delete_role(&mut self, role_name: &str) -> Result<()> {
        let role = self.roles.get(role_name)
            .ok_or_else(|| SmartTicketError::NotFound {
                entity: "Role".to_string(),
                id: role_name.to_string(),
            })?;

        if role.is_system_role {
            return Err(SmartTicketError::PermissionDenied(
                "Cannot delete system roles".to_string()
            ));
        }

        self.roles.remove(role_name);
        Ok(())
    }

    /// List all roles
    pub fn list_roles(&self) -> Vec<&Role> {
        self.roles.values().collect()
    }

    /// Get all permissions for a user
    pub fn get_user_permissions(&self, context: &TenantContext) -> HashSet<String> {
        let mut permissions = HashSet::new();

        // Add role-based permissions
        if let Some(role) = self.get_role(&context.user_role) {
            for permission in &role.permissions {
                let perm_string = format!("{:?}:{:?}", permission.resource, permission.action);
                permissions.insert(perm_string);
            }
        }

        // Add individual permissions from JWT
        for perm in &context.permissions {
            permissions.insert(perm.clone());
        }

        permissions
    }
}

pub trait PermissionChecker {
    fn check_permission(
        &self,
        context: &TenantContext,
        resource: Resource,
        action: Action,
        resource_tenant_id: Option<Uuid>,
        resource_owner_id: Option<Uuid>,
    ) -> Result<()>;
}

impl PermissionChecker for PermissionService {
    fn check_permission(
        &self,
        context: &TenantContext,
        resource: Resource,
        action: Action,
        resource_tenant_id: Option<Uuid>,
        resource_owner_id: Option<Uuid>,
    ) -> Result<()> {
        self.check_permission(context, resource, action, resource_tenant_id, resource_owner_id)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use uuid::Uuid;

    fn create_test_tenant_context(role: &str) -> TenantContext {
        TenantContext {
            tenant_id: "tenant-123".to_string(),
            tenant_name: "Test Tenant".to_string(),
            user_id: "user-456".to_string(),
            user_role: role.to_string(),
            permissions: HashSet::new(),
        }
    }

    #[test]
    fn test_default_roles_initialization() {
        let service = PermissionService::new();

        // Check that all default roles are initialized
        assert!(service.get_role("SuperAdmin").is_some());
        assert!(service.get_role("TenantAdmin").is_some());
        assert!(service.get_role("SupportEngineer").is_some());
        assert!(service.get_role("CustomerUser").is_some());
        assert!(service.get_role("Sales").is_some());
    }

    #[test]
    fn test_super_admin_permissions() {
        let service = PermissionService::new();
        let context = create_test_tenant_context("SuperAdmin");

        // Super admin should have access to everything
        let test_cases = vec![
            (Resource::Ticket, Action::Admin),
            (Resource::KnowledgeArticle, Action::Admin),
            (Resource::User, Action::Admin),
            (Resource::Tenant, Action::Admin),
            (Resource::SLAPolicy, Action::Admin),
            (Resource::AuditLog, Action::Read),
        ];

        for (resource, action) in test_cases {
            assert!(
                service.check_permission(&context, resource, action, None, None).is_ok(),
                "SuperAdmin should have {:?} permission on {:?}",
                action, resource
            );
        }
    }

    #[test]
    fn test_customer_user_permissions() {
        let service = PermissionService::new();
        let context = create_test_tenant_context("CustomerUser");
        let tenant_id = Uuid::new_v4();
        let user_id = Uuid::parse_str("user-456").unwrap();

        // Customer should be able to read their own tickets
        assert!(service.check_permission(
            &context,
            Resource::Ticket,
            Action::Read,
            Some(tenant_id),
            Some(user_id)
        ).is_ok());

        // Customer should not be able to access other users' tickets
        let other_user_id = Uuid::new_v4();
        assert!(service.check_permission(
            &context,
            Resource::Ticket,
            Action::Read,
            Some(tenant_id),
            Some(other_user_id)
        ).is_err());

        // Customer should not have admin permissions
        assert!(service.check_permission(
            &context,
            Resource::Ticket,
            Action::Admin,
            None,
            None
        ).is_err());
    }

    #[test]
    fn test_support_engineer_permissions() {
        let service = PermissionService::new();
        let context = create_test_tenant_context("SupportEngineer");
        let tenant_id = Uuid::new_v4();

        // Support engineer should be able to access all tickets in their tenant
        assert!(service.check_permission(
            &context,
            Resource::Ticket,
            Action::Read,
            Some(tenant_id),
            Some(Uuid::new_v4()) // Different owner
        ).is_ok());

        // Support engineer should not have tenant admin permissions
        assert!(service.check_permission(
            &context,
            Resource::Tenant,
            Action::Admin,
            None,
            None
        ).is_err());
    }

    #[test]
    fn test_custom_roles() {
        let mut service = PermissionService::new();

        let custom_permissions = vec![
            Permission {
                resource: Resource::Ticket,
                action: Action::Read,
                conditions: None,
            },
        ];

        let custom_role = Role::new(
            "CustomRole".to_string(),
            custom_permissions,
            false,
        );

        // Add custom role
        assert!(service.add_role(custom_role).is_ok());
        assert!(service.get_role("CustomRole").is_some());

        // Try to add system role (should fail)
        let system_role = Role::new(
            "AnotherSuperAdmin".to_string(),
            vec![],
            true,
        );
        assert!(service.add_role(system_role).is_err());

        // Update custom role
        let updated_permissions = vec![
            Permission {
                resource: Resource::Ticket,
                action: Action::Write,
                conditions: None,
            },
        ];

        assert!(service.update_role("CustomRole", updated_permissions).is_ok());
        assert!(service.update_role("SuperAdmin", vec![]).is_err());

        // Delete custom role
        assert!(service.delete_role("CustomRole").is_ok());
        assert!(service.delete_role("SuperAdmin").is_err());
    }

    #[test]
    fn test_tenant_isolation() {
        let service = PermissionService::new();
        let context = create_test_tenant_context("CustomerUser");

        let user_tenant = Uuid::parse_str("tenant-123").unwrap();
        let other_tenant = Uuid::new_v4();

        // User should access their own tenant
        assert!(service.check_permission(
            &context,
            Resource::Ticket,
            Action::Read,
            Some(user_tenant),
            Some(Uuid::new_v4())
        ).is_ok());

        // User should not access other tenants
        assert!(service.check_permission(
            &context,
            Resource::Ticket,
            Action::Read,
            Some(other_tenant),
            Some(Uuid::new_v4())
        ).is_err());
    }
}