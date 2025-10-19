use chrono::Utc;
use jsonwebtoken::{decode, encode, DecodingKey, EncodingKey, Header, Validation};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::models::UserRole;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Claims {
    pub sub: String, // User ID
    pub email: String,
    pub username: String,
    pub full_name: String,
    pub tenant_id: String,
    pub role: UserRole,
    pub permissions: Vec<String>,
    pub exp: usize,  // Expiration time
    pub iat: usize,  // Issued at
    pub iss: String, // Issuer
    pub aud: String, // Audience
}

#[derive(Clone)]
pub struct AuthService {
    encoding_key: EncodingKey,
    decoding_key: DecodingKey,
    issuer: String,
    audience: String,
}

impl AuthService {
    pub fn new(secret: &str, issuer: String, audience: String) -> Self {
        Self {
            encoding_key: EncodingKey::from_secret(secret.as_ref()),
            decoding_key: DecodingKey::from_secret(secret.as_ref()),
            issuer,
            audience,
        }
    }

    pub fn generate_token(
        &self,
        user_id: &Uuid,
        email: &str,
        username: &str,
        full_name: &str,
        tenant_id: &Uuid,
        role: &UserRole,
    ) -> Result<String, String> {
        let now = Utc::now();
        let exp = now + chrono::Duration::hours(24); // 24 hour expiration

        let permissions = self.get_permissions_for_role(role);

        let claims = Claims {
            sub: user_id.to_string(),
            email: email.to_string(),
            username: username.to_string(),
            full_name: full_name.to_string(),
            tenant_id: tenant_id.to_string(),
            role: role.clone(),
            permissions,
            exp: exp.timestamp() as usize,
            iat: now.timestamp() as usize,
            iss: self.issuer.clone(),
            aud: self.audience.clone(),
        };

        encode(&Header::default(), &claims, &self.encoding_key)
            .map_err(|e| format!("Failed to generate token: {}", e))
    }

    pub fn validate_token(&self, token: &str) -> Result<Claims, String> {
        let mut validation = Validation::new(jsonwebtoken::Algorithm::HS256);
        validation.set_issuer(&[&self.issuer]);
        validation.set_audience(&[&self.audience]);

        decode::<Claims>(token, &self.decoding_key, &validation)
            .map(|data| data.claims)
            .map_err(|e| format!("Invalid token: {}", e))
    }

    pub fn get_permissions_for_role(&self, role: &UserRole) -> Vec<String> {
        match role {
            UserRole::SuperAdmin => vec![
                "system:admin".to_string(),
                "tenant:create".to_string(),
                "tenant:update".to_string(),
                "tenant:delete".to_string(),
                "user:view".to_string(),
                "user:create".to_string(),
                "user:update".to_string(),
                "user:delete".to_string(),
                "user:view_own".to_string(),
                "ticket:create".to_string(),
                "ticket:view".to_string(),
                "ticket:update".to_string(),
                "ticket:delete".to_string(),
                "ticket:assign".to_string(),
                "knowledge:create".to_string(),
                "knowledge:view".to_string(),
                "knowledge:update".to_string(),
                "knowledge:delete".to_string(),
                "knowledge:publish".to_string(),
            ],
            UserRole::TenantAdmin => vec![
                "tenant:update".to_string(),
                "user:view".to_string(),
                "user:create".to_string(),
                "user:update".to_string(),
                "user:delete".to_string(),
                "user:view_own".to_string(),
                "ticket:create".to_string(),
                "ticket:view".to_string(),
                "ticket:update".to_string(),
                "ticket:delete".to_string(),
                "ticket:assign".to_string(),
                "knowledge:create".to_string(),
                "knowledge:view".to_string(),
                "knowledge:update".to_string(),
                "knowledge:delete".to_string(),
                "knowledge:publish".to_string(),
            ],
            UserRole::SupportEngineer => vec![
                "ticket:view".to_string(),
                "ticket:update".to_string(),
                "ticket:assign".to_string(),
                "knowledge:create".to_string(),
                "knowledge:view".to_string(),
                "knowledge:update".to_string(),
                "knowledge:publish".to_string(),
            ],
            UserRole::Sales => vec![
                "ticket:view".to_string(),
                "ticket:create".to_string(),
                "knowledge:view".to_string(),
            ],
            UserRole::CustomerUser => vec![
                "ticket:create".to_string(),
                "ticket:view_own".to_string(),
                "ticket:update_own".to_string(),
                "knowledge:view".to_string(),
            ],
        }
    }

    pub fn has_permission(&self, claims: &Claims, permission: &str) -> bool {
        claims.permissions.contains(&permission.to_string())
    }

    pub fn has_any_permission(&self, claims: &Claims, permissions: &[&str]) -> bool {
        permissions
            .iter()
            .any(|perm| claims.permissions.contains(&perm.to_string()))
    }

    pub fn has_all_permissions(&self, claims: &Claims, permissions: &[&str]) -> bool {
        permissions
            .iter()
            .all(|perm| claims.permissions.contains(&perm.to_string()))
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthUser {
    pub id: Uuid,
    pub tenant_id: Uuid,
    pub email: String,
    pub username: String,
    pub full_name: String,
    pub role: UserRole,
    pub permissions: Vec<String>,
}

impl From<Claims> for AuthUser {
    fn from(claims: Claims) -> Self {
        Self {
            id: Uuid::parse_str(&claims.sub).unwrap_or_default(),
            tenant_id: Uuid::parse_str(&claims.tenant_id).unwrap_or_default(),
            email: claims.email,
            username: claims.username,
            full_name: claims.full_name,
            role: claims.role,
            permissions: claims.permissions,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::models::UserRole;

    #[test]
    fn test_jwt_token_generation_and_validation() {
        let auth_service = AuthService::new(
            "test_secret_key_12345",
            "smartticket".to_string(),
            "smartticket-client".to_string(),
        );

        let user_id = Uuid::new_v4();
        let tenant_id = Uuid::new_v4();
        let token = auth_service
            .generate_token(
                &user_id,
                "test@example.com",
                "testuser",
                "Test User",
                &tenant_id,
                &UserRole::SupportEngineer,
            )
            .unwrap();

        let claims = auth_service.validate_token(&token).unwrap();
        assert_eq!(claims.sub, user_id.to_string());
        assert_eq!(claims.email, "test@example.com");
        assert!(matches!(claims.role, UserRole::SupportEngineer));
    }

    #[test]
    fn test_role_permissions() {
        let auth_service = AuthService::new(
            "test_secret_key_12345",
            "smartticket".to_string(),
            "smartticket-client".to_string(),
        );

        let admin_perms = auth_service.get_permissions_for_role(&UserRole::SuperAdmin);
        assert!(admin_perms.contains(&"system:admin".to_string()));

        let customer_perms = auth_service.get_permissions_for_role(&UserRole::CustomerUser);
        assert!(customer_perms.contains(&"ticket:create".to_string()));
        assert!(!customer_perms.contains(&"ticket:assign".to_string()));
    }
}
