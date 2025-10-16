use serde::{Deserialize, Serialize};
use smartticket_shared_error::{Result, SmartTicketError};
use std::collections::HashSet;

/// Permission levels in the system
#[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub enum Permission {
    // Ticket permissions
    #[serde(rename = "tickets:read")]
    TicketsRead,
    #[serde(rename = "tickets:write")]
    TicketsWrite,
    #[serde(rename = "tickets:delete")]
    TicketsDelete,
    #[serde(rename = "tickets:assign")]
    TicketsAssign,

    // Knowledge base permissions
    #[serde(rename = "knowledge:read")]
    KnowledgeRead,
    #[serde(rename = "knowledge:write")]
    KnowledgeWrite,
    #[serde(rename = "knowledge:publish")]
    KnowledgePublish,
    #[serde(rename = "knowledge:delete")]
    KnowledgeDelete,

    // User management permissions
    #[serde(rename = "users:read")]
    UsersRead,
    #[serde(rename = "users:write")]
    UsersWrite,
    #[serde(rename = "users:delete")]
    UsersDelete,

    // Tenant management permissions
    #[serde(rename = "tenant:read")]
    TenantRead,
    #[serde(rename = "tenant:write")]
    TenantWrite,
    #[serde(rename = "tenant:delete")]
    TenantDelete,

    // System permissions
    #[serde(rename = "system:admin")]
    SystemAdmin,
    #[serde(rename = "system:audit")]
    SystemAudit,
    #[serde(rename = "system:reports")]
    SystemReports,
}

impl Permission {
    /// Get the string representation of the permission
    pub fn as_str(&self) -> &'static str {
        match self {
            Permission::TicketsRead => "tickets:read",
            Permission::TicketsWrite => "tickets:write",
            Permission::TicketsDelete => "tickets:delete",
            Permission::TicketsAssign => "tickets:assign",
            Permission::KnowledgeRead => "knowledge:read",
            Permission::KnowledgeWrite => "knowledge:write",
            Permission::KnowledgePublish => "knowledge:publish",
            Permission::KnowledgeDelete => "knowledge:delete",
            Permission::UsersRead => "users:read",
            Permission::UsersWrite => "users:write",
            Permission::UsersDelete => "users:delete",
            Permission::TenantRead => "tenant:read",
            Permission::TenantWrite => "tenant:write",
            Permission::TenantDelete => "tenant:delete",
            Permission::SystemAdmin => "system:admin",
            Permission::SystemAudit => "system:audit",
            Permission::SystemReports => "system:reports",
        }
    }

    /// Parse a string into a permission
    pub fn from_str(s: &str) -> Result<Self> {
        match s {
            "tickets:read" => Ok(Permission::TicketsRead),
            "tickets:write" => Ok(Permission::TicketsWrite),
            "tickets:delete" => Ok(Permission::TicketsDelete),
            "tickets:assign" => Ok(Permission::TicketsAssign),
            "knowledge:read" => Ok(Permission::KnowledgeRead),
            "knowledge:write" => Ok(Permission::KnowledgeWrite),
            "knowledge:publish" => Ok(Permission::KnowledgePublish),
            "knowledge:delete" => Ok(Permission::KnowledgeDelete),
            "users:read" => Ok(Permission::UsersRead),
            "users:write" => Ok(Permission::UsersWrite),
            "users:delete" => Ok(Permission::UsersDelete),
            "tenant:read" => Ok(Permission::TenantRead),
            "tenant:write" => Ok(Permission::TenantWrite),
            "tenant:delete" => Ok(Permission::TenantDelete),
            "system:admin" => Ok(Permission::SystemAdmin),
            "system:audit" => Ok(Permission::SystemAudit),
            "system:reports" => Ok(Permission::SystemReports),
            _ => Err(SmartTicketError::Validation(format!(
                "Unknown permission: {}",
                s
            ))),
        }
    }
}

/// User roles with associated permissions
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum Role {
    #[serde(rename = "CustomerUser")]
    CustomerUser,
    #[serde(rename = "CustomerAdmin")]
    CustomerAdmin,
    #[serde(rename = "SupportAgent")]
    SupportAgent,
    #[serde(rename = "SupportManager")]
    SupportManager,
    #[serde(rename = "SystemAdmin")]
    SystemAdmin,
}

impl Role {
    /// Get all permissions for this role
    pub fn permissions(&self) -> HashSet<Permission> {
        match self {
            Role::CustomerUser => {
                let mut perms = HashSet::new();
                perms.insert(Permission::TicketsRead);
                perms.insert(Permission::KnowledgeRead);
                perms
            }
            Role::CustomerAdmin => {
                let mut perms = HashSet::new();
                perms.insert(Permission::TicketsRead);
                perms.insert(Permission::TicketsWrite);
                perms.insert(Permission::KnowledgeRead);
                perms.insert(Permission::KnowledgeWrite);
                perms.insert(Permission::UsersRead);
                perms
            }
            Role::SupportAgent => {
                let mut perms = HashSet::new();
                perms.insert(Permission::TicketsRead);
                perms.insert(Permission::TicketsWrite);
                perms.insert(Permission::TicketsAssign);
                perms.insert(Permission::KnowledgeRead);
                perms.insert(Permission::KnowledgeWrite);
                perms
            }
            Role::SupportManager => {
                let mut perms = HashSet::new();
                perms.insert(Permission::TicketsRead);
                perms.insert(Permission::TicketsWrite);
                perms.insert(Permission::TicketsDelete);
                perms.insert(Permission::TicketsAssign);
                perms.insert(Permission::KnowledgeRead);
                perms.insert(Permission::KnowledgeWrite);
                perms.insert(Permission::KnowledgePublish);
                perms.insert(Permission::UsersRead);
                perms.insert(Permission::TenantRead);
                perms
            }
            Role::SystemAdmin => {
                let mut perms = HashSet::new();
                perms.insert(Permission::TicketsRead);
                perms.insert(Permission::TicketsWrite);
                perms.insert(Permission::TicketsDelete);
                perms.insert(Permission::TicketsAssign);
                perms.insert(Permission::KnowledgeRead);
                perms.insert(Permission::KnowledgeWrite);
                perms.insert(Permission::KnowledgePublish);
                perms.insert(Permission::KnowledgeDelete);
                perms.insert(Permission::UsersRead);
                perms.insert(Permission::UsersWrite);
                perms.insert(Permission::UsersDelete);
                perms.insert(Permission::TenantRead);
                perms.insert(Permission::TenantWrite);
                perms.insert(Permission::SystemAdmin);
                perms.insert(Permission::SystemAudit);
                perms.insert(Permission::SystemReports);
                perms
            }
        }
    }
}

/// Permission checker service
#[derive(Debug, Clone)]
pub struct PermissionService {
    role_cache: std::collections::HashMap<String, HashSet<Permission>>,
}

impl PermissionService {
    pub fn new() -> Self {
        Self {
            role_cache: std::collections::HashMap::new(),
        }
    }

    /// Check if a user has a specific permission
    pub fn has_permission(&self, user_permissions: &[String], permission: &Permission) -> bool {
        user_permissions
            .iter()
            .any(|p| Permission::from_str(p).map_or(false, |perm| perm == *permission))
    }

    /// Check if a user has any of the specified permissions
    pub fn has_any_permission(
        &self,
        user_permissions: &[String],
        permissions: &[Permission],
    ) -> bool {
        permissions
            .iter()
            .any(|perm| self.has_permission(user_permissions, perm))
    }

    /// Check if a user has all of the specified permissions
    pub fn has_all_permissions(
        &self,
        user_permissions: &[String],
        permissions: &[Permission],
    ) -> bool {
        permissions
            .iter()
            .all(|perm| self.has_permission(user_permissions, perm))
    }

    /// Get permissions for a role
    pub fn get_role_permissions(&self, role: &Role) -> HashSet<Permission> {
        // Use cache if available
        let role_str = serde_json::to_string(role).unwrap_or_default();
        if !self.role_cache.contains_key(&role_str) {
            let _permissions = role.permissions();
            // Note: This would need mutable access in a real implementation
            // For now, we'll just compute directly
        }

        // Return the permissions directly
        role.permissions()
    }
}

impl Default for PermissionService {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_permission_parsing() {
        assert_eq!(
            Permission::from_str("tickets:read").unwrap(),
            Permission::TicketsRead
        );
        assert_eq!(
            Permission::from_str("system:admin").unwrap(),
            Permission::SystemAdmin
        );
        assert!(Permission::from_str("invalid:permission").is_err());
    }

    #[test]
    fn test_role_permissions() {
        let customer_user_perms = Role::CustomerUser.permissions();
        assert!(customer_user_perms.contains(&Permission::TicketsRead));
        assert!(customer_user_perms.contains(&Permission::KnowledgeRead));
        assert!(!customer_user_perms.contains(&Permission::TicketsWrite));

        let system_admin_perms = Role::SystemAdmin.permissions();
        assert!(system_admin_perms.contains(&Permission::SystemAdmin));
        assert!(system_admin_perms.contains(&Permission::TicketsDelete));
        assert!(system_admin_perms.contains(&Permission::UsersWrite));
    }

    #[test]
    fn test_permission_service() {
        let service = PermissionService::new();
        let user_permissions = vec![
            "tickets:read".to_string(),
            "tickets:write".to_string(),
            "knowledge:read".to_string(),
        ];

        assert!(service.has_permission(&user_permissions, &Permission::TicketsRead));
        assert!(service.has_permission(&user_permissions, &Permission::TicketsWrite));
        assert!(!service.has_permission(&user_permissions, &Permission::TicketsDelete));

        // Test any permission
        let required_permissions = vec![Permission::TicketsRead, Permission::TicketsDelete];
        assert!(service.has_any_permission(&user_permissions, &required_permissions));

        // Test all permissions
        let required_permissions = vec![Permission::TicketsRead, Permission::TicketsWrite];
        assert!(service.has_all_permissions(&user_permissions, &required_permissions));

        let required_permissions = vec![Permission::TicketsRead, Permission::TicketsDelete];
        assert!(!service.has_all_permissions(&user_permissions, &required_permissions));
    }
}
