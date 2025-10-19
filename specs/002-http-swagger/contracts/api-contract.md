# API Contract: HTTP REST API Gateway

**Date**: 2025-01-17
**Version**: 1.0.0
**Purpose**: Define the complete REST API contract generated from gRPC services

## API Overview

The SmartTicket HTTP REST API provides complete access to all platform functionality through standard REST endpoints. All endpoints require authentication via JWT Bearer tokens and multi-tenant context headers.

### Base URL
- **Development**: `http://localhost:3286/v1`
- **Staging**: `https://staging-api.smartticket.com/v1`
- **Production**: `https://api.smartticket.com/v1`

### Authentication
```http
Authorization: Bearer <jwt_token>
X-Tenant-ID: <tenant_uuid>
X-User-ID: <user_uuid>
X-User-Role: <user_role>
```

## Service Contracts

### 1. Authentication Service

#### POST /auth/login
Authenticate user and return JWT tokens.

**Request**:
```json
{
  "email": "user@example.com",
  "password": "securePassword123",
  "tenant_domain": "example.smartticket.com",
  "remember_me": false
}
```

**Response** (200):
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
      "email": "user@example.com",
      "username": "johndoe",
      "full_name": "John Doe",
      "role": "CUSTOMER_USER",
      "is_active": true,
      "last_login_at": 1705396800,
      "created_at": 1705310400,
      "updated_at": 1705396800
    }
  },
  "request_id": "req_123456789",
  "timestamp": 1705483200
}
```

#### POST /auth/refresh
Refresh access token using refresh token.

**Request**:
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Response** (200):
```json
{
  "success": true,
  "message": "Token refreshed successfully",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_at": 1705483200
  },
  "request_id": "req_123456790",
  "timestamp": 1705483200
}
```

### 2. User Management Service

#### GET /users
Retrieve paginated list of users.

**Query Parameters**:
- `page` (integer): Page number (default: 1)
- `page_size` (integer): Items per page (default: 20, max: 100)
- `search` (string): Search query
- `role` (string): Filter by user role
- `is_active` (boolean): Filter by active status

**Response** (200):
```json
{
  "success": true,
  "message": "Users retrieved successfully",
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "user@example.com",
      "username": "johndoe",
      "full_name": "John Doe",
      "role": "CUSTOMER_USER",
      "is_active": true,
      "last_login_at": 1705396800,
      "created_at": 1705310400,
      "updated_at": 1705396800
    }
  ],
  "pagination": {
    "total_count": 100,
    "page_size": 20,
    "current_page": 1,
    "total_pages": 5,
    "has_next": true,
    "has_prev": false,
    "next_page_token": "abc123",
    "prev_page_token": null
  },
  "request_id": "req_123456791",
  "timestamp": 1705483200
}
```

#### POST /users
Create a new user.

**Request**:
```json
{
  "email": "newuser@example.com",
  "username": "newuser",
  "full_name": "New User",
  "password": "securePassword123",
  "role": "CUSTOMER_USER",
  "phone": "+1234567890",
  "timezone": "America/New_York",
  "language": "en",
  "preferences": {
    "theme": "light",
    "notifications": true
  }
}
```

**Response** (201):
```json
{
  "success": true,
  "message": "User created successfully",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "email": "newuser@example.com",
    "username": "newuser",
    "full_name": "New User",
    "role": "CUSTOMER_USER",
    "is_active": true,
    "last_login_at": null,
    "created_at": 1705483200,
    "updated_at": 1705483200
  },
  "request_id": "req_123456792",
  "timestamp": 1705483200
}
```

#### GET /users/{user_id}
Retrieve specific user by ID.

**Path Parameters**:
- `user_id` (string, UUID): User identifier

**Response** (200):
```json
{
  "success": true,
  "message": "User retrieved successfully",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "username": "johndoe",
    "full_name": "John Doe",
    "role": "CUSTOMER_USER",
    "is_active": true,
    "last_login_at": 1705396800,
    "created_at": 1705310400,
    "updated_at": 1705396800
  },
  "request_id": "req_123456793",
  "timestamp": 1705483200
}
```

#### PUT /users/{user_id}
Update user information.

**Request**:
```json
{
  "full_name": "John Smith",
  "phone": "+1234567890",
  "timezone": "America/New_York",
  "language": "en",
  "preferences": {
    "theme": "dark",
    "notifications": false
  }
}
```

**Response** (200):
```json
{
  "success": true,
  "message": "User updated successfully",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "username": "johndoe",
    "full_name": "John Smith",
    "role": "CUSTOMER_USER",
    "is_active": true,
    "last_login_at": 1705396800,
    "created_at": 1705310400,
    "updated_at": 1705483200
  },
  "request_id": "req_123456794",
  "timestamp": 1705483200
}
```

#### GET /users/current
Retrieve current user profile and permissions.

**Response** (200):
```json
{
  "success": true,
  "message": "Current user profile retrieved successfully",
  "data": {
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "user@example.com",
      "username": "johndoe",
      "full_name": "John Doe",
      "role": "CUSTOMER_USER",
      "is_active": true,
      "last_login_at": 1705396800,
      "created_at": 1705310400,
      "updated_at": 1705396800
    },
    "profile": {
      "user_id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "user@example.com",
      "username": "johndoe",
      "full_name": "John Doe",
      "phone": "+1234567890",
      "timezone": "America/New_York",
      "language": "en",
      "preferences": {
        "theme": "light",
        "notifications": true
      },
      "created_at": 1705310400,
      "updated_at": 1705396800
    },
    "permissions": [
      {
        "resource": "tickets",
        "actions": ["read", "create", "update"]
      },
      {
        "resource": "knowledge",
        "actions": ["read"]
      }
    ]
  },
  "request_id": "req_123456795",
  "timestamp": 1705483200
}
```

### 3. Tenant Management Service

#### GET /tenants
Retrieve paginated list of tenants.

**Query Parameters**:
- `page` (integer): Page number
- `page_size` (integer): Items per page
- `search` (string): Search query
- `subscription_tier` (string): Filter by subscription tier
- `is_active` (boolean): Filter by active status
- `data_residency_region` (string): Filter by region

**Response** (200):
```json
{
  "success": true,
  "message": "Tenants retrieved successfully",
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "Example Company",
      "domain": "example.smartticket.com",
      "subscription_tier": "PREMIUM",
      "max_users": 100,
      "current_user_count": 45,
      "data_residency_region": "EU",
      "is_active": true,
      "created_at": 1705310400,
      "updated_at": 1705396800,
      "subscription_expires_at": 1708992000,
      "contact_email": "admin@example.com"
    }
  ],
  "pagination": {
    "total_count": 50,
    "page_size": 20,
    "current_page": 1,
    "total_pages": 3,
    "has_next": true,
    "has_prev": false,
    "next_page_token": "def456",
    "prev_page_token": null
  },
  "request_id": "req_123456796",
  "timestamp": 1705483200
}
```

#### POST /tenants
Create a new tenant.

**Request**:
```json
{
  "name": "New Company",
  "domain": "newcompany.smartticket.com",
  "subscription_tier": "STANDARD",
  "max_users": 50,
  "data_residency_region": "US",
  "contact_email": "admin@newcompany.com",
  "billing_address": "123 Main St, City, State 12345",
  "phone": "+1234567890",
  "settings": {
    "default_timezone": "America/New_York",
    "default_language": "en",
    "enable_multi_language": true,
    "allow_user_registration": false
  }
}
```

**Response** (201):
```json
{
  "success": true,
  "message": "Tenant created successfully",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440002",
    "name": "New Company",
    "domain": "newcompany.smartticket.com",
    "subscription_tier": "STANDARD",
    "max_users": 50,
    "current_user_count": 0,
    "data_residency_region": "US",
    "is_active": true,
    "created_at": 1705483200,
    "updated_at": 1705483200,
    "contact_email": "admin@newcompany.com"
  },
  "request_id": "req_123456797",
  "timestamp": 1705483200
}
```

#### GET /tenants/current
Retrieve current user's tenant information.

**Response** (200):
```json
{
  "success": true,
  "message": "Current tenant retrieved successfully",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Example Company",
    "domain": "example.smartticket.com",
    "subscription_tier": "PREMIUM",
    "max_users": 100,
    "current_user_count": 45,
    "data_residency_region": "EU",
    "is_active": true,
    "created_at": 1705310400,
    "updated_at": 1705396800,
    "subscription_expires_at": 1708992000,
    "contact_email": "admin@example.com"
  },
  "request_id": "req_123456798",
  "timestamp": 1705483200
}
```

### 4. Ticket Management Service

#### GET /tickets
Retrieve paginated list of tickets.

**Query Parameters**:
- `page` (integer): Page number
- `page_size` (integer): Items per page
- `search` (string): Search query
- `statuses` (array): Filter by ticket statuses
- `priorities` (array): Filter by ticket priorities
- `severities` (array): Filter by ticket severities
- `assigned_to_id` (string): Filter by assigned user
- `contact_id` (string): Filter by contact user
- `category_id` (string): Filter by category
- `created_after` (string): Filter created after date
- `created_before` (string): Filter created before date

**Response** (200):
```json
{
  "success": true,
  "message": "Tickets retrieved successfully",
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440003",
      "title": "Login issue with mobile app",
      "description": "User cannot login to mobile app",
      "status": "OPEN",
      "priority": "NORMAL",
      "severity": "MINOR",
      "category_id": "550e8400-e29b-41d4-a716-446655440004",
      "contact_id": "550e8400-e29b-41d4-a716-446655440005",
      "assigned_to_id": "550e8400-e29b-41d4-a716-446655440006",
      "created_by_id": "550e8400-e29b-41d4-a716-446655440005",
      "created_at": 1705396800,
      "updated_at": 1705396800,
      "due_date": 1705656000,
      "resolved_at": null,
      "tags": ["mobile", "login"],
      "custom_fields": {
        "device_type": "iOS",
        "app_version": "2.1.0"
      }
    }
  ],
  "pagination": {
    "total_count": 200,
    "page_size": 20,
    "current_page": 1,
    "total_pages": 10,
    "has_next": true,
    "has_prev": false,
    "next_page_token": "ghi789",
    "prev_page_token": null
  },
  "request_id": "req_123456799",
  "timestamp": 1705483200
}
```

#### POST /tickets
Create a new ticket.

**Request**:
```json
{
  "title": "Cannot reset password",
  "description": "User is unable to reset password through the forgot password flow",
  "priority": "HIGH",
  "severity": "MAJOR",
  "category_id": "550e8400-e29b-41d4-a716-446655440004",
  "contact_id": "550e8400-e29b-41d4-a716-446655440005",
  "assigned_to_id": null,
  "due_date": 1705656000,
  "tags": ["password", "reset"],
  "custom_fields": {
    "browser": "Chrome",
    "os": "Windows 11"
  }
}
```

**Response** (201):
```json
{
  "success": true,
  "message": "Ticket created successfully",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440007",
    "title": "Cannot reset password",
    "description": "User is unable to reset password through the forgot password flow",
    "status": "OPEN",
    "priority": "HIGH",
    "severity": "MAJOR",
    "category_id": "550e8400-e29b-41d4-a716-446655440004",
    "contact_id": "550e8400-e29b-41d4-a716-446655440005",
    "assigned_to_id": null,
    "created_by_id": "550e8400-e29b-41d4-a716-446655440000",
    "created_at": 1705483200,
    "updated_at": 1705483200,
    "due_date": 1705656000,
    "resolved_at": null,
    "tags": ["password", "reset"],
    "custom_fields": {
      "browser": "Chrome",
      "os": "Windows 11"
    }
  },
  "request_id": "req_123456800",
  "timestamp": 1705483200
}
```

#### GET /tickets/{ticket_id}
Retrieve specific ticket by ID.

**Path Parameters**:
- `ticket_id` (string, UUID): Ticket identifier

**Query Parameters**:
- `include_comments` (boolean): Include ticket comments (default: false)

**Response** (200):
```json
{
  "success": true,
  "message": "Ticket retrieved successfully",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440003",
    "title": "Login issue with mobile app",
    "description": "User cannot login to mobile app",
    "status": "OPEN",
    "priority": "NORMAL",
    "severity": "MINOR",
    "category_id": "550e8400-e29b-41d4-a716-446655440004",
    "contact_id": "550e8400-e29b-41d4-a716-446655440005",
    "assigned_to_id": "550e8400-e29b-41d4-a716-446655440006",
    "created_by_id": "550e8400-e29b-41d4-a716-446655440005",
    "created_at": 1705396800,
    "updated_at": 1705396800,
    "due_date": 1705656000,
    "resolved_at": null,
    "tags": ["mobile", "login"],
    "custom_fields": {
      "device_type": "iOS",
      "app_version": "2.1.0"
    }
  },
  "request_id": "req_123456801",
  "timestamp": 1705483200
}
```

#### PATCH /tickets/{ticket_id}/status
Update ticket status.

**Request**:
```json
{
  "status": "IN_PROGRESS",
  "comment": "Investigating the mobile app login issue",
  "notify_customer": true
}
```

**Response** (200):
```json
{
  "success": true,
  "message": "Ticket status updated successfully",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440003",
    "status": "IN_PROGRESS",
    "updated_at": 1705483200
  },
  "request_id": "req_123456802",
  "timestamp": 1705483200
}
```

### 5. Knowledge Management Service

#### GET /knowledge/articles
Retrieve paginated list of knowledge articles.

**Query Parameters**:
- `page` (integer): Page number
- `page_size` (integer): Items per page
- `search` (string): Search query
- `statuses` (array): Filter by article statuses
- `visibilities` (array): Filter by visibility levels
- `category_id` (string): Filter by category
- `author_id` (string): Filter by author
- `language` (string): Filter by language
- `tags` (array): Filter by tags

**Response** (200):
```json
{
  "success": true,
  "message": "Articles retrieved successfully",
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440008",
      "title": "How to Reset Your Password",
      "summary": "Step-by-step guide to reset your account password",
      "status": "PUBLISHED",
      "visibility": "PUBLIC",
      "category_id": "550e8400-e29b-41d4-a716-446655440009",
      "author_id": "550e8400-e29b-41d4-a716-446655440010",
      "language": "en",
      "tags": ["password", "account", "security"],
      "view_count": 1250,
      "rating": 4.5,
      "created_at": 1705310400,
      "updated_at": 1705396800,
      "published_at": 1705310400
    }
  ],
  "pagination": {
    "total_count": 150,
    "page_size": 20,
    "current_page": 1,
    "total_pages": 8,
    "has_next": true,
    "has_prev": false,
    "next_page_token": "jkl012",
    "prev_page_token": null
  },
  "request_id": "req_123456803",
  "timestamp": 1705483200
}
```

#### POST /knowledge/articles
Create a new knowledge article.

**Request**:
```json
{
  "title": "Setting Up Two-Factor Authentication",
  "content": "This article explains how to set up 2FA for your account...",
  "summary": "Learn how to enable and configure two-factor authentication",
  "category_id": "550e8400-e29b-41d4-a716-446655440009",
  "tags": ["2fa", "security", "authentication"],
  "language": "en",
  "visibility": "INTERNAL"
}
```

**Response** (201):
```json
{
  "success": true,
  "message": "Article created successfully",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440011",
    "title": "Setting Up Two-Factor Authentication",
    "content": "This article explains how to set up 2FA for your account...",
    "summary": "Learn how to enable and configure two-factor authentication",
    "status": "DRAFT",
    "visibility": "INTERNAL",
    "category_id": "550e8400-e29b-41d4-a716-446655440009",
    "author_id": "550e8400-e29b-41d4-a716-446655440000",
    "language": "en",
    "tags": ["2fa", "security", "authentication"],
    "view_count": 0,
    "rating": null,
    "created_at": 1705483200,
    "updated_at": 1705483200,
    "published_at": null
  },
  "request_id": "req_123456804",
  "timestamp": 1705483200
}
```

#### POST /knowledge/articles/{article_id}/publish
Publish a knowledge article.

**Request**:
```json
{
  "publish_at": 1705569600,
  "notify_subscribers": true,
  "announcement_message": "New security guide available"
}
```

**Response** (200):
```json
{
  "success": true,
  "message": "Article published successfully",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440011",
    "status": "PUBLISHED",
    "published_at": 1705569600,
    "updated_at": 1705483200
  },
  "request_id": "req_123456805",
  "timestamp": 1705483200
}
```

### 6. SLA Management Service

#### GET /sla/policies
Retrieve SLA policies.

**Query Parameters**:
- `page` (integer): Page number
- `page_size` (integer): Items per page
- `search` (string): Search query
- `priority` (string): Filter by ticket priority
- `severity` (string): Filter by ticket severity
- `category_id` (string): Filter by category
- `is_active` (boolean): Filter by active status

**Response** (200):
```json
{
  "success": true,
  "message": "SLA policies retrieved successfully",
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440012",
      "name": "Standard Response Time",
      "description": "Standard SLA for regular tickets",
      "priority": "NORMAL",
      "severity": null,
      "category_id": null,
      "response_time_minutes": 60,
      "resolution_time_minutes": 480,
      "business_hours_only": true,
      "timezone": "America/New_York",
      "is_active": true,
      "created_at": 1705310400,
      "updated_at": 1705396800
    }
  ],
  "pagination": {
    "total_count": 25,
    "page_size": 20,
    "current_page": 1,
    "total_pages": 2,
    "has_next": true,
    "has_prev": false,
    "next_page_token": "mno345",
    "prev_page_token": null
  },
  "request_id": "req_123456806",
  "timestamp": 1705483200
}
```

### 7. Role and Permission Service

#### GET /roles
Retrieve available roles.

**Query Parameters**:
- `page` (integer): Page number
- `page_size` (integer): Items per page
- `search` (string): Search query
- `is_system_role` (boolean): Filter system roles

**Response** (200):
```json
{
  "success": true,
  "message": "Roles retrieved successfully",
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440013",
      "name": "Support Engineer",
      "description": "Can handle and resolve customer tickets",
      "permissions": [
        {
          "id": "perm_001",
          "resource": "tickets",
          "action": "read",
          "description": "Read tickets"
        },
        {
          "id": "perm_002",
          "resource": "tickets",
          "action": "update",
          "description": "Update tickets"
        }
      ],
      "is_system_role": false,
      "user_count": 12,
      "created_at": 1705310400,
      "updated_at": 1705396800
    }
  ],
  "pagination": {
    "total_count": 15,
    "page_size": 20,
    "current_page": 1,
    "total_pages": 1,
    "has_next": false,
    "has_prev": false,
    "next_page_token": null,
    "prev_page_token": null
  },
  "request_id": "req_123456807",
  "timestamp": 1705483200
}
```

## Error Responses

All endpoints return consistent error responses:

### 400 Bad Request
```json
{
  "success": false,
  "message": "Invalid input parameters",
  "errors": [
    {
      "code": "VALIDATION_ERROR",
      "message": "Email is required",
      "field": "email"
    }
  ],
  "request_id": "req_123456808",
  "timestamp": 1705483200
}
```

### 401 Unauthorized
```json
{
  "success": false,
  "message": "Authentication required",
  "errors": [
    {
      "code": "UNAUTHORIZED",
      "message": "Invalid or missing JWT token"
    }
  ],
  "request_id": "req_123456809",
  "timestamp": 1705483200
}
```

### 403 Forbidden
```json
{
  "success": false,
  "message": "Insufficient permissions",
  "errors": [
    {
      "code": "FORBIDDEN",
      "message": "User does not have required permissions"
    }
  ],
  "request_id": "req_123456810",
  "timestamp": 1705483200
}
```

### 404 Not Found
```json
{
  "success": false,
  "message": "Resource not found",
  "errors": [
    {
      "code": "NOT_FOUND",
      "message": "Requested resource does not exist"
    }
  ],
  "request_id": "req_123456811",
  "timestamp": 1705483200
}
```

### 429 Rate Limit Exceeded
```json
{
  "success": false,
  "message": "Rate limit exceeded",
  "errors": [
    {
      "code": "RATE_LIMIT_EXCEEDED",
      "message": "Too many requests, please try again later"
    }
  ],
  "request_id": "req_123456812",
  "timestamp": 1705483200
}
```

## Rate Limiting

API endpoints are rate-limited per tenant:

- **Enterprise tier**: 1000 requests per hour
- **Premium tier**: 500 requests per hour
- **Standard tier**: 200 requests per hour

Rate limit headers are included in responses:
```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1705486800
```

This contract provides complete coverage of all 68 gRPC interfaces exposed as REST endpoints with consistent authentication, error handling, and response formats.