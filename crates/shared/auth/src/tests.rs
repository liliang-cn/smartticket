#[cfg(test)]
mod tests {
    use super::*;
    use crate::claims::{ApiKeyClaims, TenantContext, UserClaims};
    use crate::jwt::UserData;
    use crate::JwtService;
    use chrono::{Duration, Utc};
    use smartticket_shared_config::AuthConfig;
    use uuid::Uuid;

    fn create_test_jwt_service() -> JwtService {
        let config = AuthConfig {
            jwt_secret: "test-secret-key-must-be-at-least-32-characters-for-security".to_string(),
            jwt_expiration: 3600,
            refresh_expiration: 86400,
            issuer: "smartticket-test".to_string(),
            bcrypt_cost: 4,
            password_min_length: 6,
            password_require_special: false,
            password_require_numbers: false,
            password_require_uppercase: false,
            rate_limit: Default::default(),
        };

        JwtService::new(config).expect("Failed to create JWT service for testing")
    }

    fn create_test_user_data() -> UserData {
        UserData {
            email: "test@example.com".to_string(),
            username: "testuser".to_string(),
            full_name: "Test User".to_string(),
            role: "CustomerUser".to_string(),
            tenant_id: "tenant-123".to_string(),
            tenant_name: "Test Tenant".to_string(),
            permissions: vec!["tickets:read".to_string(), "tickets:write".to_string()],
        }
    }

    #[test]
    fn test_jwt_token_generation_and_verification() {
        let jwt_service = create_test_jwt_service();
        let user_data = create_test_user_data();
        let user_id = "user-123";

        // Generate token pair
        let token_pair = jwt_service
            .generate_token_pair(user_id, &user_data)
            .expect("Failed to generate token pair");

        // Verify access token
        let claims = jwt_service
            .verify_user_token(&token_pair.access_token)
            .expect("Failed to verify access token");

        assert_eq!(claims.sub, user_id);
        assert_eq!(claims.email, user_data.email);
        assert_eq!(claims.username, user_data.username);
        assert_eq!(claims.full_name, user_data.full_name);
        assert_eq!(claims.role, user_data.role);
        assert_eq!(claims.tenant_id, user_data.tenant_id);
        assert_eq!(claims.tenant_name, user_data.tenant_name);
        assert_eq!(claims.permissions, user_data.permissions);
        assert!(!claims.is_expired());

        // Verify refresh token
        let refresh_claims = jwt_service
            .verify_refresh_token(&token_pair.refresh_token)
            .expect("Failed to verify refresh token");

        assert_eq!(refresh_claims.sub, user_id);
        assert_eq!(refresh_claims.type_, "refresh");
        assert!(!refresh_claims.is_expired());

        // Verify token type is "Bearer"
        assert_eq!(token_pair.token_type, "Bearer");

        // Verify expiration time is in the future
        assert!(token_pair.expires_at > Utc::now());
    }

    #[test]
    fn test_expired_token_verification() {
        let jwt_service = create_test_jwt_service();
        let user_data = create_test_user_data();
        let user_id = "user-123";

        // Create a config with negative expiration to simulate expired token
        let mut config = AuthConfig {
            jwt_secret: "test-secret-key-must-be-at-least-32-characters-for-security".to_string(),
            jwt_expiration: 1, // Very short expiration
            refresh_expiration: 1,
            issuer: "smartticket-test".to_string(),
            bcrypt_cost: 4,
            password_min_length: 6,
            password_require_special: false,
            password_require_numbers: false,
            password_require_uppercase: false,
            rate_limit: Default::default(),
        };

        let expired_jwt_service =
            JwtService::new(config).expect("Failed to create expired JWT service");

        // Generate expired token pair
        let token_pair = expired_jwt_service
            .generate_token_pair(user_id, &user_data)
            .expect("Failed to generate expired token pair");

        // Try to verify expired access token - should fail
        let result = jwt_service.verify_user_token(&token_pair.access_token);
        assert!(result.is_err(), "Expired token should not be verifiable");

        // Try to verify expired refresh token - should fail
        let result = jwt_service.verify_refresh_token(&token_pair.refresh_token);
        assert!(
            result.is_err(),
            "Expired refresh token should not be verifiable"
        );
    }

    #[test]
    fn test_invalid_token() {
        let jwt_service = create_test_jwt_service();

        // Test with completely invalid token
        let invalid_tokens = vec![
            "",                        // Empty
            "invalid",                 // Invalid format
            "invalid.token",           // Incomplete token
            "invalid.token.signature", // Invalid payload
        ];

        for token in invalid_tokens {
            let result = jwt_service.verify_user_token(token);
            assert!(
                result.is_err(),
                "Invalid token '{}' should fail verification",
                token
            );
        }
    }

    #[test]
    fn test_user_claims_permissions() {
        let user_data = UserData {
            email: "admin@example.com".to_string(),
            username: "admin".to_string(),
            full_name: "Admin User".to_string(),
            role: "TenantAdmin".to_string(),
            tenant_id: "tenant-123".to_string(),
            tenant_name: "Test Tenant".to_string(),
            permissions: vec![
                "tickets:read".to_string(),
                "tickets:write".to_string(),
                "users:read".to_string(),
                "users:write".to_string(),
            ],
        };

        let now = Utc::now();
        let expires_at = now + Duration::hours(1);
        let claims = UserClaims::new(
            "user-123".to_string(),
            user_data.email,
            user_data.username,
            user_data.full_name,
            user_data.role,
            user_data.tenant_id,
            user_data.tenant_name,
            user_data.permissions,
            now,
            expires_at,
            "smartticket-test".to_string(),
            Uuid::new_v4().to_string(),
        );

        // Test permission checking
        assert!(claims.has_permission("tickets:read"));
        assert!(claims.has_permission("tickets:write"));
        assert!(claims.has_permission("users:read"));
        assert!(claims.has_permission("users:write"));
        assert!(!claims.has_permission("nonexistent:permission"));

        // Test permission combinations
        assert!(claims.has_any_permission(&["tickets:read", "nonexistent:permission"]));
        assert!(claims.has_any_permission(&["users:read", "users:write"]));
        assert!(!claims.has_any_permission(&["nonexistent:permission1", "nonexistent:permission2"]));

        assert!(claims.has_all_permissions(&["tickets:read", "tickets:write"]));
        assert!(!claims.has_all_permissions(&["tickets:read", "nonexistent:permission"]));

        // Test role checking
        assert!(claims.is_admin());
        assert!(!claims.is_support()); // TenantAdmin is not considered support in this context

        // Test permission set
        let permission_set = claims.get_permission_set();
        assert_eq!(permission_set.len(), 4);
        assert!(permission_set.contains("tickets:read"));
        assert!(permission_set.contains("tickets:write"));
        assert!(permission_set.contains("users:read"));
        assert!(permission_set.contains("users:write"));
    }

    #[test]
    fn test_tenant_context() {
        let user_data = UserData {
            email: "support@example.com".to_string(),
            username: "support".to_string(),
            full_name: "Support User".to_string(),
            role: "SupportEngineer".to_string(),
            tenant_id: "tenant-456".to_string(),
            tenant_name: "Support Tenant".to_string(),
            permissions: vec![
                "tickets:read".to_string(),
                "tickets:write".to_string(),
                "tickets:assign".to_string(),
                "knowledge:read".to_string(),
                "knowledge:write".to_string(),
            ],
        };

        let now = Utc::now();
        let expires_at = now + Duration::hours(1);
        let claims = UserClaims::new(
            "user-456".to_string(),
            user_data.email,
            user_data.username,
            user_data.full_name,
            user_data.role,
            user_data.tenant_id,
            user_data.tenant_name,
            user_data.permissions,
            now,
            expires_at,
            "smartticket-test".to_string(),
            Uuid::new_v4().to_string(),
        );

        let context = TenantContext::from_claims(&claims);

        // Test basic properties
        assert_eq!(context.tenant_id, "tenant-456");
        assert_eq!(context.tenant_name, "Support Tenant");
        assert_eq!(context.user_id, "user-456");
        assert_eq!(context.user_role, "SupportEngineer");

        // Test tenant access
        assert!(context.can_access_tenant("tenant-456"));
        assert!(!context.can_access_tenant("other-tenant"));

        // Test role checks
        assert!(!context.is_admin());
        assert!(context.is_support());

        // Test permission checks
        assert!(context.has_permission("tickets:read"));
        assert!(context.has_permission("tickets:assign"));
        assert!(!context.has_permission("users:write"));

        // Test capability checks
        assert!(!context.can_manage_users());
        assert!(context.can_view_all_tickets());
        assert!(context.can_assign_tickets());
        assert!(context.can_manage_knowledge());
        assert!(!context.can_manage_sla());
        assert!(!context.can_view_audit_logs());
    }

    #[test]
    fn test_api_key_claims() {
        let permissions = vec![
            "api:tickets:read".to_string(),
            "api:tickets:create".to_string(),
        ];

        let now = Utc::now();
        let expires_at = now + Duration::hours(1);
        let api_key_claims = ApiKeyClaims::new(
            "api-key-123".to_string(),
            "Test API Key".to_string(),
            "tenant-789".to_string(),
            permissions,
            now,
            expires_at,
            "smartticket-test".to_string(),
            Uuid::new_v4().to_string(),
        );

        // Test basic properties
        assert_eq!(api_key_claims.sub, "api-key-123");
        assert_eq!(api_key_claims.name, "Test API Key");
        assert_eq!(api_key_claims.tenant_id, "tenant-789");
        assert!(!api_key_claims.is_expired());

        // Test permission checking
        assert!(api_key_claims.has_permission("api:tickets:read"));
        assert!(api_key_claims.has_permission("api:tickets:create"));
        assert!(!api_key_claims.has_permission("api:tickets:delete"));

        // Test tenant context creation
        let context = api_key_claims.get_tenant_context();
        assert_eq!(context.tenant_id, "tenant-789");
        assert_eq!(context.user_id, "api-key-123");
        assert_eq!(context.user_role, "ApiKey");
        assert!(context.has_permission("api:tickets:read"));
    }

    #[test]
    fn test_extract_token_from_header() {
        let jwt_service = create_test_jwt_service();
        let user_data = create_test_user_data();
        let token_pair = jwt_service
            .generate_token_pair("user-123", &user_data)
            .expect("Failed to generate token pair");

        // Test valid header format
        let valid_header = format!("Bearer {}", token_pair.access_token);
        let extracted = jwt_service.extract_token_from_header(&valid_header);
        assert!(extracted.is_ok());
        assert_eq!(extracted.unwrap(), token_pair.access_token);

        // Test invalid header formats
        let invalid_headers = vec![
            "",                   // Empty
            "Bearer",             // Missing token
            "bearer token",       // Lowercase "bearer"
            "Token abc123",       // Wrong prefix
            "Bearer token extra", // Too many parts
        ];

        for header in invalid_headers {
            let result = jwt_service.extract_token_from_header(header);
            assert!(
                result.is_err(),
                "Invalid header '{}' should fail extraction",
                header
            );
        }
    }

    #[test]
    fn test_refresh_token_type_validation() {
        let jwt_service = create_test_jwt_service();
        let user_data = create_test_user_data();
        let user_id = "user-123";

        // Generate valid refresh token
        let token_pair = jwt_service
            .generate_token_pair(user_id, &user_data)
            .expect("Failed to generate token pair");

        // Try to verify refresh token as user token - should fail due to type mismatch
        let refresh_claims = jwt_service
            .verify_refresh_token(&token_pair.refresh_token)
            .expect("Failed to verify refresh token");

        assert_eq!(refresh_claims.type_, "refresh");

        // Try to use refresh token as access token - this should work at the JWT level
        // but the application logic should check the type
        let user_claims = jwt_service.verify_user_token(&token_pair.refresh_token);
        assert!(
            user_claims.is_err(),
            "Refresh token should not be verifiable as user token"
        );
    }

    #[test]
    fn test_different_roles_context() {
        let test_cases = vec![
            ("SuperAdmin", true, true, true, true, true, true),
            ("TenantAdmin", true, true, true, true, true, false),
            ("SupportEngineer", false, true, true, true, false, false),
            ("CustomerUser", false, false, false, false, false, false),
            ("Sales", false, false, false, false, false, false),
        ];

        for (
            role,
            can_manage_users,
            can_view_all_tickets,
            can_assign_tickets,
            can_manage_knowledge,
            can_manage_sla,
            can_view_audit_logs,
        ) in test_cases
        {
            let user_data = UserData {
                email: format!("{}@example.com", role.to_lowercase()),
                username: role.to_lowercase(),
                full_name: format!("{} User", role),
                role: role.to_string(),
                tenant_id: "tenant-test".to_string(),
                tenant_name: "Test Tenant".to_string(),
                permissions: vec!["tickets:read".to_string()],
            };

            let now = Utc::now();
            let expires_at = now + Duration::hours(1);
            let claims = UserClaims::new(
                format!("user-{}", role.to_lowercase()),
                user_data.email,
                user_data.username,
                user_data.full_name,
                user_data.role,
                user_data.tenant_id,
                user_data.tenant_name,
                user_data.permissions,
                now,
                expires_at,
                "smartticket-test".to_string(),
                Uuid::new_v4().to_string(),
            );

            let context = TenantContext::from_claims(&claims);

            assert_eq!(
                context.can_manage_users(),
                can_manage_users,
                "Role {} - can_manage_users failed",
                role
            );
            assert_eq!(
                context.can_view_all_tickets(),
                can_view_all_tickets,
                "Role {} - can_view_all_tickets failed",
                role
            );
            assert_eq!(
                context.can_assign_tickets(),
                can_assign_tickets,
                "Role {} - can_assign_tickets failed",
                role
            );
            assert_eq!(
                context.can_manage_knowledge(),
                can_manage_knowledge,
                "Role {} - can_manage_knowledge failed",
                role
            );
            assert_eq!(
                context.can_manage_sla(),
                can_manage_sla,
                "Role {} - can_manage_sla failed",
                role
            );
            assert_eq!(
                context.can_view_audit_logs(),
                can_view_audit_logs,
                "Role {} - can_view_audit_logs failed",
                role
            );
        }
    }
}
