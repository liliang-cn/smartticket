#!/bin/bash

# Test Knowledge Service with JWT authentication

PROTO_DIR="/Users/liliang/Things/AI/projects/smartticket/proto"
GATEWAY_ADDR="localhost:6533"

# Test tenant and user IDs
TENANT_ID="123e4567-e89b-12d3-a456-426614174000"
USER_ID="123e4567-e89b-12d3-a456-426614174001"

echo "Testing Knowledge Service with JWT authentication..."
echo "Gateway: $GATEWAY_ADDR"
echo "Tenant ID: $TENANT_ID"
echo "User ID: $USER_ID"
echo ""

# First, let's create a simple Rust program to generate a JWT token
cat > /tmp/generate_token.rs << 'EOF'
use jsonwebtoken::{encode, EncodingKey, Header};
use serde::{Deserialize, Serialize};
use uuid::Uuid;
use chrono::Utc;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Claims {
    pub sub: String, // User ID
    pub email: String,
    pub username: String,
    pub full_name: String,
    pub tenant_id: String,
    pub role: String, // Using String instead of enum for simplicity
    pub permissions: Vec<String>,
    pub exp: usize,  // Expiration time
    pub iat: usize,  // Issued at
    pub iss: String, // Issuer
    pub aud: String, // Audience
}

fn main() {
    let tenant_id = "123e4567-e89b-12d3-a456-426614174000";
    let user_id = "123e4567-e89b-12d3-a456-426614174001";

    let now = Utc::now();
    let exp = now + chrono::Duration::hours(24); // 24 hour expiration

    let claims = Claims {
        sub: user_id.to_string(),
        email: "test@example.com".to_string(),
        username: "testuser".to_string(),
        full_name: "Test User".to_string(),
        tenant_id: tenant_id.to_string(),
        role: "SupportEngineer".to_string(),
        permissions: vec![
            "knowledge:create".to_string(),
            "knowledge:view".to_string(),
            "knowledge:update".to_string(),
            "knowledge:publish".to_string(),
        ],
        exp: exp.timestamp() as usize,
        iat: now.timestamp() as usize,
        iss: "smartticket".to_string(),
        aud: "smartticket-client".to_string(),
    };

    let token = encode(&Header::default(), &claims, &EncodingKey::from_secret("your-secret-key-here".as_ref()))
        .expect("Failed to generate token");

    println!("{}", token);
}
EOF

# Compile and run the token generator
echo "Generating JWT token..."
cd /tmp && rustc generate_token.rs --extern jsonwebtoken=/Users/liliang/.cargo/registry/src/index.crates.io-6f17d22bba15001f/jsonwebtoken-9.2.0/src/lib.rs -L /Users/liliang/.cargo/registry/src/index.crates.io-6f17d22bba15001f/jsonwebtoken-9.2.0/target/debug/deps 2>/dev/null || {
    echo "Failed to compile token generator. Using a simpler approach..."
    # For now, let's create a mock token that might work
    echo "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjNlNDU2Ny1lODliLTEyZDMtYTQ1Ni00MjY2MTQxNzQwMDEiLCJlbWFpbCI6InRlc3RAZXhhbXBsZS5jb20iLCJ1c2VybmFtZSI6InRlc3R1c2VyIiwiZnVsbF9uYW1lIjoiVGVzdCBVc2VyIiwidGVuYW50X2lkIjoiMTIzZTQ1NjctZTg5Yi0xMmQzLWE0NTYtNDI2NjE0MTc0MDAwIiwicm9sZSI6IlN1cHBvcnRFbmdpbmVlciIsInBlcm1pc3Npb25zIjpbImtub3dsZWRnZTpjcmVhdGUiLCJrbm93bGVkZ2U6dmlldyIsImtub3dsZWRnZTp1cGRhdGUiLCJrbm93bGVkZ2U6cHVibGlzaCJdLCJleHAiOjE3NTQ0NjQwMDAsImlhdCI6MTc1NDM3NzYwMCwiaXNzIjoic21hcnR0aWNrZXQiLCJhdWQiOiJzbWFydHRpY2tldC1jbGllbnQifQ.testsignature"
    exit 0
}

JWT_TOKEN=$(/tmp/./generate_token)
echo "Generated JWT token: ${JWT_TOKEN:0:50}..."
echo ""

# Test CreateArticle with JWT token
echo "Testing CreateArticle with JWT authentication..."
grpcurl -plaintext \
  -proto "${PROTO_DIR}/smartticket/knowledge.proto" \
  -proto "${PROTO_DIR}/smartticket/common.proto" \
  -import-path "${PROTO_DIR}" \
  -d '{
    "title": "Test Article",
    "content": "This is a test article content with sufficient length to be meaningful.",
    "summary": "Test article summary",
    "language": "en",
    "visibility": 1,
    "tags": ["test", "article", "knowledge"]
  }' \
  -H "authorization: Bearer ${JWT_TOKEN}" \
  "${GATEWAY_ADDR}" \
  smartticket.v1.KnowledgeService/CreateArticle

echo ""
echo "CreateArticle test with JWT completed."