//! Password hashing and validation

use smartticket_shared_error::{Result, SmartTicketError};
use tracing::error;
use validator::ValidateEmail;

/// Password hashing and validation service
pub struct PasswordService {
    /// Bcrypt work factor (higher is more secure but slower)
    cost: u32,
}

impl PasswordService {
    pub fn new(cost: u32) -> Self {
        Self { cost }
    }

    /// Hash a password using bcrypt
    pub fn hash_password(&self, password: &str) -> Result<String> {
        if password.len() < 8 {
            return Err(SmartTicketError::Validation(
                "Password must be at least 8 characters long".to_string(),
            ));
        }

        bcrypt::hash(password, self.cost).map_err(|e| {
            error!("Failed to hash password: {}", e);
            SmartTicketError::Internal(format!("Password hashing failed: {}", e))
        })
    }

    /// Verify a password against a hash
    pub fn verify_password(&self, password: &str, hash: &str) -> Result<bool> {
        bcrypt::verify(password, hash).map_err(|e| {
            error!("Failed to verify password: {}", e);
            SmartTicketError::Internal(format!("Password verification failed: {}", e))
        })
    }

    /// Generate a secure random password
    pub fn generate_password(&self, length: usize) -> String {
        use rand::Rng;
        const LOWERCASE: &[u8] = b"abcdefghijklmnopqrstuvwxyz";
        const UPPERCASE: &[u8] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZ";
        const DIGITS: &[u8] = b"0123456789";
        const SPECIAL: &[u8] = b"!@#$%^&*()";
        const ALL_CHARS: &[u8] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*()";
        let mut rng = rand::thread_rng();

        if length < 4 {
            // For very short passwords, just use random characters from all sets
            return (0..length)
                .map(|_| {
                    let idx = rng.gen_range(0..ALL_CHARS.len());
                    ALL_CHARS[idx] as char
                })
                .collect();
        }

        // Ensure at least one of each character type
        let mut password_chars = Vec::with_capacity(length);

        // Add one guaranteed character from each set
        password_chars.push(LOWERCASE[rng.gen_range(0..LOWERCASE.len())] as char);
        password_chars.push(UPPERCASE[rng.gen_range(0..UPPERCASE.len())] as char);
        password_chars.push(DIGITS[rng.gen_range(0..DIGITS.len())] as char);
        password_chars.push(SPECIAL[rng.gen_range(0..SPECIAL.len())] as char);

        // Fill the rest with random characters from all sets
        for _ in 4..length {
            let idx = rng.gen_range(0..ALL_CHARS.len());
            password_chars.push(ALL_CHARS[idx] as char);
        }

        // Shuffle the characters to avoid predictable patterns
        use rand::seq::SliceRandom;
        password_chars.shuffle(&mut rng);

        password_chars.into_iter().collect()
    }

    /// Validate password strength
    pub fn validate_password_strength(&self, password: &str) -> Result<()> {
        let mut score = 0;
        let mut feedback = Vec::new();

        // Length check
        if password.len() >= 12 {
            score += 2;
        } else if password.len() >= 8 {
            score += 1;
            feedback.push("Use at least 12 characters for better security");
        }

        // Character variety
        if password.chars().any(|c| c.is_uppercase()) {
            score += 1;
        } else {
            feedback.push("Include uppercase letters");
        }

        if password.chars().any(|c| c.is_lowercase()) {
            score += 1;
        } else {
            feedback.push("Include lowercase letters");
        }

        if password.chars().any(|c| c.is_numeric()) {
            score += 1;
        } else {
            feedback.push("Include numbers");
        }

        if password
            .chars()
            .any(|c| "!@#$%^&*()_+-=[]{}|;:,.<>?".contains(c))
        {
            score += 1;
        } else {
            feedback.push("Include special characters");
        }

        // Check for common weak passwords
        let common_passwords = [
            "password",
            "123456",
            "12345678",
            "qwerty",
            "abc123",
            "letmein",
            "admin",
            "welcome",
            "password123",
        ];

        if common_passwords.contains(&password.to_lowercase().as_str()) {
            return Err(SmartTicketError::Validation(
                "Password is too common or weak".to_string(),
            ));
        }

        // Check for repeated patterns
        if password
            .chars()
            .collect::<Vec<char>>()
            .windows(3)
            .any(|window| window[0] == window[1] && window[1] == window[2])
        {
            feedback.push("Avoid repeated character patterns");
            score -= 1;
        }

        if score < 3 {
            return Err(SmartTicketError::Validation(format!(
                "Password is too weak. Please improve: {}",
                feedback.join(", ")
            )));
        }

        Ok(())
    }

    /// Check if email format is valid
    pub fn validate_email_format(&self, email: &str) -> Result<()> {
        if email.validate_email() {
            Ok(())
        } else {
            Err(SmartTicketError::Validation(
                "Invalid email format".to_string(),
            ))
        }
    }
}

impl Default for PasswordService {
    fn default() -> Self {
        Self::new(12) // bcrypt cost of 12 is secure for production
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_password_hashing_and_verification() {
        let service = PasswordService::new(4); // Lower cost for faster tests
        let password = "testpassword123";

        let hash = service.hash_password(password).unwrap();
        assert_ne!(hash, password);
        assert!(hash.starts_with("$2b$")); // bcrypt hash prefix

        let is_valid = service.verify_password(password, &hash).unwrap();
        assert!(is_valid);

        let is_invalid = service.verify_password("wrongpassword", &hash).unwrap();
        assert!(!is_invalid);
    }

    #[test]
    fn test_password_generation() {
        let service = PasswordService::new(4);
        let password = service.generate_password(16);

        assert_eq!(password.len(), 16);
        assert!(password.chars().any(|c| c.is_ascii_alphabetic()));
        assert!(password.chars().any(|c| c.is_ascii_digit()));
        assert!(password
            .chars()
            .any(|c| "!@#$%^&*()_+-=[]{}|;:,.<>?".contains(c)));
    }

    #[test]
    fn test_password_strength_validation() {
        let service = PasswordService::new(4);

        // Test weak passwords
        let weak_passwords = vec!["short", "password", "12345678", "abc123", "qwertyui"];

        for password in weak_passwords {
            assert!(
                service.validate_password_strength(password).is_err(),
                "Password '{}' should be rejected as weak",
                password
            );
        }

        // Test strong passwords
        let strong_passwords = vec![
            "MyStr0ngP@ssw0rd!",
            "Complex!123Password",
            "S3cur3P@ssw0rdWithNumbers",
        ];

        for password in strong_passwords {
            assert!(
                service.validate_password_strength(password).is_ok(),
                "Password '{}' should pass strength validation",
                password
            );
        }
    }

    #[test]
    fn test_email_validation() {
        let service = PasswordService::new(4);

        let valid_emails = vec![
            "user@example.com",
            "test.email@domain.co.uk",
            "user+tag@domain.org",
        ];

        let invalid_emails = vec!["invalid-email", "@domain.com", "user@", "user@.com"];

        for email in valid_emails {
            assert!(
                service.validate_email_format(email).is_ok(),
                "Email '{}' should be valid",
                email
            );
        }

        for email in invalid_emails {
            assert!(
                service.validate_email_format(email).is_err(),
                "Email '{}' should be invalid",
                email
            );
        }
    }
}
