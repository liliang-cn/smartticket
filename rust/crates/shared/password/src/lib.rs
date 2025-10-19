use argonautica::{Hasher, Verifier};
use rand::Rng;
use smartticket_shared_config::AuthConfig;
use smartticket_shared_error::{SmartTicketError, Result};
use tracing::{info, error};
use validator::validate_email;

pub struct PasswordService {
    hasher: Hasher,
    config: AuthConfig,
}

impl PasswordService {
    pub fn new(config: AuthConfig) -> Result<Self> {
        let hasher = Hasher::default()
            .configure_non_secret_argon2()
            .map_err(|e| {
                error!("Failed to configure password hasher: {}", e);
                SmartTicketError::PasswordHashing(e)
            })?;

        Ok(Self { hasher, config })
    }

    /// Hash a password with the configured parameters
    pub fn hash_password(&self, password: &str) -> Result<String> {
        // Validate password requirements
        self.validate_password(password)?;

        let hash = self.hasher
            .with_password(password)
            .with_secret_key(&self.config.jwt_secret)
            .hash()
            .map_err(|e| {
                error!("Failed to hash password: {}", e);
                SmartTicketError::PasswordHashing(e)
            })?;

        info!("Password hashed successfully");
        Ok(hash)
    }

    /// Verify a password against a hash
    pub fn verify_password(&self, password: &str, hash: &str) -> Result<bool> {
        let is_valid = self.hasher
            .with_password(password)
            .with_hash(hash)
            .with_secret_key(&self.config.jwt_secret)
            .verify()
            .map_err(|e| {
                error!("Failed to verify password: {}", e);
                SmartTicketError::PasswordHashing(e)
            })?;

        Ok(is_valid)
    }

    /// Generate a random password
    pub fn generate_password(&self, length: usize) -> String {
        const CHARSET: &[u8] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*";
        let mut rng = rand::thread_rng();

        (0..length)
            .map(|_| {
                let idx = rng.gen_range(0..CHARSET.len());
                CHARSET[idx] as char
            })
            .collect()
    }

    /// Validate password against configured requirements
    fn validate_password(&self, password: &str) -> Result<()> {
        if password.len() < self.config.password_min_length {
            return Err(SmartTicketError::Validation(format!(
                "Password must be at least {} characters long",
                self.config.password_min_length
            )));
        }

        if self.config.password_require_uppercase && !password.chars().any(|c| c.is_uppercase()) {
            return Err(SmartTicketError::Validation(
                "Password must contain at least one uppercase letter".to_string()
            ));
        }

        if self.config.password_require_numbers && !password.chars().any(|c| c.is_numeric()) {
            return Err(SmartTicketError::Validation(
                "Password must contain at least one number".to_string()
            ));
        }

        if self.config.password_require_special && !password.chars().any(|c| "!@#$%^&*()_+-=[]{}|;:,.<>?".contains(c)) {
            return Err(SmartTicketError::Validation(
                "Password must contain at least one special character".to_string()
            ));
        }

        Ok(())
    }

    /// Check if a password is strong (additional heuristics)
    pub fn is_strong_password(&self, password: &str) -> bool {
        // Basic requirements
        if let Err(_) = self.validate_password(password) {
            return false;
        }

        // Additional strength checks
        let has_lowercase = password.chars().any(|c| c.is_lowercase());
        let has_uppercase = password.chars().any(|c| c.is_uppercase());
        let has_numbers = password.chars().any(|c| c.is_numeric());
        let has_special = password.chars().any(|c| "!@#$%^&*()_+-=[]{}|;:,.<>?".contains(c));

        // Score based on character variety
        let mut score = 0;
        if has_lowercase { score += 1; }
        if has_uppercase { score += 1; }
        if has_numbers { score += 1; }
        if has_special { score += 1; }

        // Length bonus
        if password.len() >= 12 { score += 1; }
        if password.len() >= 16 { score += 1; }

        score >= 4 // Must meet at least 4 criteria
    }

    /// Get password strength description
    pub fn get_password_strength(&self, password: &str) -> PasswordStrength {
        if !self.is_strong_password(password) {
            return PasswordStrength::Weak;
        }

        let has_lowercase = password.chars().any(|c| c.is_lowercase());
        let has_uppercase = password.chars().any(|c| c.is_uppercase());
        let has_numbers = password.chars().any(|c| c.is_numeric());
        let has_special = password.chars().any(|c| "!@#$%^&*()_+-=[]{}|;:,.<>?".contains(c));

        let score = [has_lowercase, has_uppercase, has_numbers, has_special]
            .iter()
            .filter(|&&x| x)
            .count();

        if password.len() >= 16 && score >= 4 {
            PasswordStrength::VeryStrong
        } else if password.len() >= 12 && score >= 3 {
            PasswordStrength::Strong
        } else if password.len() >= 8 && score >= 2 {
            PasswordStrength::Medium
        } else {
            PasswordStrength::Weak
        }
    }
}

#[derive(Debug, Clone, PartialEq)]
pub enum PasswordStrength {
    Weak,
    Medium,
    Strong,
    VeryStrong,
}

impl PasswordStrength {
    pub fn description(&self) -> &'static str {
        match self {
            PasswordStrength::Weak => "Weak - does not meet security requirements",
            PasswordStrength::Medium => "Medium - meets basic requirements",
            PasswordStrength::Strong => "Strong - good security characteristics",
            PasswordStrength::VeryStrong => "Very Strong - excellent security characteristics",
        }
    }

    pub fn color(&self) -> &'static str {
        match self {
            PasswordStrength::Weak => "red",
            PasswordStrength::Medium => "yellow",
            PasswordStrength::Strong => "blue",
            PasswordStrength::VeryStrong => "green",
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn create_test_config() -> AuthConfig {
        AuthConfig {
            jwt_secret: "test-secret-key-must-be-at-least-32-characters-for-security".to_string(),
            jwt_expiration: 3600,
            refresh_expiration: 86400,
            issuer: "smartticket-test".to_string(),
            bcrypt_cost: 4, // Lower cost for faster tests
            password_min_length: 6,
            password_require_special: false,
            password_require_numbers: false,
            password_require_uppercase: false,
            rate_limit: Default::default(),
        }
    }

    #[test]
    fn test_password_hashing_and_verification() {
        let config = create_test_config();
        let service = PasswordService::new(config).unwrap();
        let password = "test123";

        let hash = service.hash_password(password).unwrap();
        assert!(!hash.is_empty());
        assert!(hash.len() > 50); // Argon2 hashes are long

        let is_valid = service.verify_password(password, &hash).unwrap();
        assert!(is_valid);

        let is_invalid = service.verify_password("wrongpassword", &hash).unwrap();
        assert!(!is_invalid);
    }

    #[test]
    fn test_password_validation() {
        let config = create_test_config();
        let service = PasswordService::new(config).unwrap();

        // Test minimum length
        assert!(service.validate_password("12345").is_err()); // Too short
        assert!(service.validate_password("123456").is_ok()); // Minimum length

        // Test different passwords
        let valid_passwords = vec![
            "password123",
            "MySecurePassword",
            "Complex!Password123",
            "aB3$", // Minimum requirements
        ];

        for password in valid_passwords {
            assert!(service.validate_password(password).is_ok(),
                    "Password '{}' should be valid", password);
        }
    }

    #[test]
    fn test_password_generation() {
        let config = create_test_config();
        let service = PasswordService::new(config).unwrap();

        for length in [8, 12, 16, 20] {
            let password = service.generate_password(length);
            assert_eq!(password.len(), length);
            assert!(service.validate_password(&password).is_ok());
        }
    }

    #[test]
    fn test_password_strength() {
        let config = create_test_config();
        let service = PasswordService::new(config).unwrap();

        let test_cases = vec![
            ("weak", PasswordStrength::Weak),
            ("Password123", PasswordStrength::Medium),
            ("StrongPassword123!", PasswordStrength::Strong),
            ("VeryStrongPassword123!@#", PasswordStrength::VeryStrong),
        ];

        for (password, expected_strength) in test_cases {
            let strength = service.get_password_strength(password);
            assert_eq!(strength, expected_strength,
                       "Password '{}' should have strength {:?}", password, expected_strength);
        }
    }

    #[test]
    fn test_password_requirements_strict() {
        let mut config = create_test_config();
        config.password_min_length = 8;
        config.password_require_uppercase = true;
        config.password_require_numbers = true;
        config.password_require_special = true;

        let service = PasswordService::new(config).unwrap();

        // Test strict requirements
        assert!(service.validate_password("weak").is_err()); // Too short, no uppercase, no numbers, no special
        assert!(service.validate_password("weakpassword").is_err()); // No uppercase, no numbers, no special
        assert!(service.validate_password("Weakpassword").is_err()); // No numbers, no special
        assert!(service.validate_password("Weakpassword1").is_err()); // No special
        assert!(service.validate_password("Weakpassword1!").is_ok()); // All requirements met
    }
}