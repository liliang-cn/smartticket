//! Test Knowledge Service with proper JWT authentication

use smartticket_shared_database::AuthService;
use uuid::Uuid;
use smartticket_shared_database::models::UserRole;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Initialize auth service with same parameters as server
    let auth_service = AuthService::new(
        "your-secret-key-here",
        "smartticket".to_string(),
        "smartticket-client".to_string(),
    );

    // Test user and tenant IDs
    let tenant_id = Uuid::parse_str("123e4567-e89b-12d3-a456-426614174000")?;
    let user_id = Uuid::parse_str("123e4567-e89b-12d3-a456-426614174001")?;

    // Generate JWT token
    let token = auth_service.generate_token(
        &user_id,
        "test@example.com",
        "testuser",
        "Test User",
        &tenant_id,
        &UserRole::SupportEngineer,
    )?;

    println!("Generated JWT token: {}", token);
    println!("Token length: {}", token.len());

    // Test token validation
    let claims = auth_service.validate_token(&token)?;
    println!("Token validation successful!");
    println!("User ID: {}", claims.sub);
    println!("Email: {}", claims.email);
    println!("Tenant ID: {}", claims.tenant_id);
    println!("Role: {:?}", claims.role);
    println!("Permissions: {:?}", claims.permissions);

    Ok(())
}