# Quick Start Guide: HTTP REST API Gateway

**Date**: 2025-01-17
**Purpose**: Get started quickly with the SmartTicket HTTP REST API

## Overview

The SmartTicket HTTP REST API provides complete access to all platform functionality through standard REST endpoints. This guide will help you get started with authentication, making your first API calls, and understanding the core concepts.

## Prerequisites

- SmartTicket gateway server running on port 3286
- Valid tenant account
- User credentials with appropriate permissions
- HTTP client (curl, Postman, or any programming language)

## Base URL

```
Development: http://localhost:3286/v1
Staging:    https://staging-api.smartticket.com/v1
Production: https://api.smartticket.com/v1
```

## Authentication

### 1. Login to Get Access Token

Make a POST request to `/auth/login`:

```bash
curl -X POST http://localhost:3286/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@test.smartticket.com",
    "password": "admin123",
    "tenant_domain": "test.smartticket.com",
    "remember_me": false
  }'
```

**Response**:
```json
{
  "success": true,
  "message": "Login successful",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_at": 1705483200,
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "admin@test.smartticket.com",
      "username": "admin",
      "full_name": "Admin User",
      "role": "TENANT_ADMIN",
      "is_active": true
    }
  },
  "request_id": "req_123456789",
  "timestamp": 1705483200
}
```

### 2. Use Access Token for API Calls

Include the access token in the Authorization header:

```bash
curl -X GET http://localhost:3286/v1/users/current \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -H "X-User-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -H "X-User-Role: TENANT_ADMIN"
```

### 3. Refresh Access Token

When your access token expires, use the refresh token:

```bash
curl -X POST http://localhost:3286/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
  }'
```

## Quick API Examples

### Get Current User Profile

```bash
curl -X GET http://localhost:3286/v1/users/current \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "X-Tenant-ID: YOUR_TENANT_ID" \
  -H "X-User-ID: YOUR_USER_ID" \
  -H "X-User-Role: YOUR_USER_ROLE"
```

### List Users (Paginated)

```bash
curl -X GET "http://localhost:3286/v1/users?page=1&page_size=10" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "X-Tenant-ID: YOUR_TENANT_ID" \
  -H "X-User-ID: YOUR_USER_ID" \
  -H "X-User-Role: YOUR_USER_ROLE"
```

### Create a New User

```bash
curl -X POST http://localhost:3286/v1/users \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: YOUR_TENANT_ID" \
  -H "X-User-ID: YOUR_USER_ID" \
  -H "X-User-Role: YOUR_USER_ROLE" \
  -d '{
    "email": "newuser@example.com",
    "username": "newuser",
    "full_name": "New User",
    "password": "SecurePassword123!",
    "role": "CUSTOMER_USER",
    "phone": "+1234567890"
  }'
```

### Create a Ticket

```bash
curl -X POST http://localhost:3286/v1/tickets \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: YOUR_TENANT_ID" \
  -H "X-User-ID: YOUR_USER_ID" \
  -H "X-User-Role: YOUR_USER_ROLE" \
  -d '{
    "title": "Login issue with mobile app",
    "description": "User cannot login to mobile app after recent update",
    "priority": "HIGH",
    "severity": "MAJOR",
    "contact_id": "USER_ID_HERE",
    "tags": ["mobile", "login"]
  }'
```

### List Tickets

```bash
curl -X GET "http://localhost:3286/v1/tickets?page=1&page_size=20&status=OPEN" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "X-Tenant-ID: YOUR_TENANT_ID" \
  -H "X-User-ID: YOUR_USER_ID" \
  -H "X-User-Role: YOUR_USER_ROLE"
```

### Create Knowledge Article

```bash
curl -X POST http://localhost:3286/v1/knowledge/articles \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: YOUR_TENANT_ID" \
  -H "X-User-ID: YOUR_USER_ID" \
  -H "X-User-Role: YOUR_USER_ROLE" \
  -d '{
    "title": "How to Reset Your Password",
    "content": "Step-by-step guide to reset your password...",
    "summary": "Learn how to reset your account password",
    "tags": ["password", "security"],
    "visibility": "PUBLIC"
  }'
```

## API Documentation

### Interactive Swagger UI

Access the interactive API documentation at:
```
http://localhost:3286/docs
```

The Swagger UI provides:
- Complete API documentation
- Interactive "Try it out" functionality
- Request/response examples
- Authentication setup

### OpenAPI Specification

Download the complete OpenAPI specification:
```bash
curl -X GET http://localhost:3286/v1/openapi.yaml
```

## Common Patterns

### Error Handling

All API responses follow a consistent format:

```json
{
  "success": true|false,
  "message": "Human-readable message",
  "data": { ... },           // Only for successful responses
  "errors": [ ... ],         // Only for error responses
  "request_id": "unique_id",
  "timestamp": 1705483200
}
```

### Pagination

List endpoints support pagination:

```json
{
  "success": true,
  "data": [ ... ],
  "pagination": {
    "total_count": 100,
    "page_size": 20,
    "current_page": 1,
    "total_pages": 5,
    "has_next": true,
    "has_prev": false,
    "next_page_token": "abc123",
    "prev_page_token": null
  }
}
```

### Search

Most list endpoints support search:

```bash
curl -X GET "http://localhost:3286/v1/tickets?search=login&page=1&page_size=10"
```

### Filtering

Use query parameters to filter results:

```bash
# Filter tickets by status and priority
curl -X GET "http://localhost:3286/v1/tickets?statuses=OPEN&priorities=HIGH"

# Filter users by role
curl -X GET "http://localhost:3286/v1/users?role=CUSTOMER_USER"
```

## Rate Limiting

API endpoints are rate-limited per tenant:

- **Enterprise**: 1000 requests/hour
- **Premium**: 500 requests/hour
- **Standard**: 200 requests/hour

Check your rate limit status:
```bash
curl -I http://localhost:3286/v1/users/current \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"

# Response headers:
# X-RateLimit-Limit: 1000
# X-RateLimit-Remaining: 999
# X-RateLimit-Reset: 1705486800
```

## Code Examples

### JavaScript/TypeScript

```javascript
// Login and get token
const loginResponse = await fetch('http://localhost:3286/v1/auth/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    email: 'admin@test.smartticket.com',
    password: 'admin123',
    tenant_domain: 'test.smartticket.com'
  })
});

const { data } = await loginResponse.json();
const token = data.access_token;

// Make authenticated request
const usersResponse = await fetch('http://localhost:3286/v1/users', {
  headers: {
    'Authorization': `Bearer ${token}`,
    'X-Tenant-ID': data.user.id,
    'X-User-ID': data.user.id,
    'X-User-Role': data.user.role
  }
});

const users = await usersResponse.json();
console.log(users);
```

### Python

```python
import requests

# Login and get token
login_response = requests.post('http://localhost:3286/v1/auth/login', json={
    'email': 'admin@test.smartticket.com',
    'password': 'admin123',
    'tenant_domain': 'test.smartticket.com'
})

login_data = login_response.json()
token = login_data['data']['access_token']
user_id = login_data['data']['user']['id']
user_role = login_data['data']['user']['role']

# Make authenticated request
headers = {
    'Authorization': f'Bearer {token}',
    'X-Tenant-ID': user_id,
    'X-User-ID': user_id,
    'X-User-Role': user_role
}

users_response = requests.get('http://localhost:3286/v1/users', headers=headers)
users = users_response.json()
print(users)
```

### cURL Script

```bash
#!/bin/bash

# Configuration
API_BASE="http://localhost:3286/v1"
EMAIL="admin@test.smartticket.com"
PASSWORD="admin123"
TENANT_DOMAIN="test.smartticket.com"

# Login
echo "Logging in..."
LOGIN_RESPONSE=$(curl -s -X POST "$API_BASE/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\",\"tenant_domain\":\"$TENANT_DOMAIN\"}")

# Extract tokens
ACCESS_TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.data.access_token')
USER_ID=$(echo $LOGIN_RESPONSE | jq -r '.data.user.id')
USER_ROLE=$(echo $LOGIN_RESPONSE | jq -r '.data.user.role')

echo "Logged in successfully!"

# Get current user
echo "Getting current user profile..."
curl -s -X GET "$API_BASE/users/current" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "X-Tenant-ID: $USER_ID" \
  -H "X-User-ID: $USER_ID" \
  -H "X-User-Role: $USER_ROLE" | jq .
```

## Multi-Tenancy

All API requests must include tenant context headers:

- `X-Tenant-ID`: Your tenant UUID
- `X-User-ID`: Your user UUID
- `X-User-Role`: Your user role

These headers ensure proper data isolation and permission enforcement.

## Troubleshooting

### Common Issues

1. **401 Unauthorized**: Check your access token is valid and not expired
2. **403 Forbidden**: Verify you have the required permissions for the operation
3. **404 Not Found**: Ensure the resource ID exists and you have access to it
4. **429 Rate Limited**: Wait for the rate limit window to reset

### Debug Tips

- Always include the `request_id` from error responses in support requests
- Check the API documentation at `/docs` for required parameters
- Verify your tenant context headers are set correctly
- Use the Swagger UI "Try it out" feature to test requests

## Support

- **API Documentation**: http://localhost:3286/docs
- **OpenAPI Spec**: http://localhost:3286/v1/openapi.yaml
- **Support Email**: api-support@smartticket.com

This quick start guide should help you get up and running quickly with the SmartTicket HTTP REST API. For more detailed information, refer to the complete API documentation.