use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::collections::HashSet;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserClaims {
    pub sub: String, // Subject (user ID)
    pub email: String,
    pub username: String,
    pub full_name: String,
    pub role: String,
    pub tenant_id: String,
    pub tenant_name: String,
    pub permissions: Vec<String>,
    pub iat: i64,    // Issued at
    pub exp: i64,    // Expiration
    pub iss: String, // Issuer
    pub jti: String, // JWT ID
}

impl UserClaims {
    pub fn new(
        user_id: String,
        email: String,
        username: String,
        full_name: String,
        role: String,
        tenant_id: String,
        tenant_name: String,
        permissions: Vec<String>,
        issued_at: DateTime<Utc>,
        expires_at: DateTime<Utc>,
        issuer: String,
        jwt_id: String,
    ) -> Self {
        Self {
            sub: user_id,
            email,
            username,
            full_name,
            role,
            tenant_id,
            tenant_name,
            permissions,
            iat: issued_at.timestamp(),
            exp: expires_at.timestamp(),
            iss: issuer,
            jti: jwt_id,
        }
    }

    pub fn is_expired(&self) -> bool {
        Utc::now().timestamp() > self.exp
    }

    pub fn has_permission(&self, permission: &str) -> bool {
        self.permissions.contains(&permission.to_string())
    }

    pub fn has_any_permission(&self, permissions: &[&str]) -> bool {
        permissions.iter().any(|p| self.has_permission(p))
    }

    pub fn has_all_permissions(&self, permissions: &[&str]) -> bool {
        permissions.iter().all(|p| self.has_permission(p))
    }

    pub fn is_admin(&self) -> bool {
        matches!(self.role.as_str(), "SuperAdmin" | "TenantAdmin")
    }

    pub fn is_support(&self) -> bool {
        matches!(
            self.role.as_str(),
            "SuperAdmin" | "SupportEngineer"  // TenantAdmin is not considered support
        )
    }

    pub fn get_permission_set(&self) -> HashSet<String> {
        self.permissions.iter().cloned().collect()
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RefreshTokenClaims {
    pub sub: String,   // Subject (user ID)
    pub type_: String, // Token type (should be "refresh")
    pub iat: i64,      // Issued at
    pub exp: i64,      // Expiration
    pub iss: String,   // Issuer
    pub jti: String,   // JWT ID
}

impl RefreshTokenClaims {
    pub fn new(
        user_id: String,
        issued_at: DateTime<Utc>,
        expires_at: DateTime<Utc>,
        issuer: String,
        jwt_id: String,
    ) -> Self {
        Self {
            sub: user_id,
            type_: "refresh".to_string(),
            iat: issued_at.timestamp(),
            exp: expires_at.timestamp(),
            iss: issuer,
            jti: jwt_id,
        }
    }

    pub fn is_expired(&self) -> bool {
        Utc::now().timestamp() > self.exp
    }
}

#[derive(Debug, Clone)]
pub struct TenantContext {
    pub tenant_id: String,
    pub tenant_name: String,
    pub user_id: String,
    pub user_role: String,
    pub permissions: HashSet<String>,
}

impl TenantContext {
    pub fn from_claims(claims: &UserClaims) -> Self {
        Self {
            tenant_id: claims.tenant_id.clone(),
            tenant_name: claims.tenant_name.clone(),
            user_id: claims.sub.clone(),
            user_role: claims.role.clone(),
            permissions: claims.get_permission_set(),
        }
    }

    pub fn can_access_tenant(&self, tenant_id: &str) -> bool {
        self.tenant_id == tenant_id || self.user_role == "SuperAdmin"
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

    pub fn has_permission(&self, permission: &str) -> bool {
        self.permissions.contains(permission)
    }

    pub fn can_manage_users(&self) -> bool {
        self.is_admin()
    }

    pub fn can_view_all_tickets(&self) -> bool {
        self.is_support()
    }

    pub fn can_assign_tickets(&self) -> bool {
        self.is_support()
    }

    pub fn can_manage_knowledge(&self) -> bool {
        self.is_support()
    }

    pub fn can_manage_sla(&self) -> bool {
        self.is_admin()
    }

    pub fn can_view_audit_logs(&self) -> bool {
        matches!(self.user_role.as_str(), "SuperAdmin")
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApiKeyClaims {
    pub sub: String,  // Subject (API key ID)
    pub name: String, // API key name
    pub tenant_id: String,
    pub permissions: Vec<String>,
    pub iat: i64,
    pub exp: i64,
    pub iss: String,
    pub jti: String,
}

impl ApiKeyClaims {
    pub fn new(
        key_id: String,
        name: String,
        tenant_id: String,
        permissions: Vec<String>,
        issued_at: DateTime<Utc>,
        expires_at: DateTime<Utc>,
        issuer: String,
        jwt_id: String,
    ) -> Self {
        Self {
            sub: key_id,
            name,
            tenant_id,
            permissions,
            iat: issued_at.timestamp(),
            exp: expires_at.timestamp(),
            iss: issuer,
            jti: jwt_id,
        }
    }

    pub fn is_expired(&self) -> bool {
        Utc::now().timestamp() > self.exp
    }

    pub fn has_permission(&self, permission: &str) -> bool {
        self.permissions.contains(&permission.to_string())
    }

    pub fn get_tenant_context(&self) -> TenantContext {
        TenantContext {
            tenant_id: self.tenant_id.clone(),
            tenant_name: "".to_string(), // API keys don't have tenant name
            user_id: self.sub.clone(),
            user_role: "ApiKey".to_string(),
            permissions: self.permissions.iter().cloned().collect(),
        }
    }
}
