//! OpenAPI Documentation Generator
//!
//! Generates comprehensive OpenAPI 3.0 specifications from gRPC service definitions
//! with dynamic content generation, multi-tenant support, and rich documentation.

use serde_json::{json, Value};
use std::collections::HashMap;
use chrono::{DateTime, Utc};
use crate::gateway::GatewayConfig;

/// Dynamic OpenAPI specification generator
pub struct OpenApiGenerator {
    config: GatewayConfig,
    generation_timestamp: DateTime<Utc>,
}

/// OpenAPI generation options
#[derive(Debug, Clone)]
pub struct OpenApiOptions {
    pub include_examples: bool,
    pub include_deprecated: bool,
    pub expand_depth: i32,
    pub include_internal_endpoints: bool,
    pub server_environment: ServerEnvironment,
}

#[derive(Debug, Clone)]
pub enum ServerEnvironment {
    Development,
    Staging,
    Production,
    Custom(String),
}

impl Default for OpenApiOptions {
    fn default() -> Self {
        Self {
            include_examples: true,
            include_deprecated: false,
            expand_depth: 2,
            include_internal_endpoints: false,
            server_environment: ServerEnvironment::Development,
        }
    }
}

impl OpenApiGenerator {
    /// Create new OpenAPI generator with current timestamp
    pub fn new(config: GatewayConfig) -> Self {
        Self {
            config,
            generation_timestamp: Utc::now(),
        }
    }

    /// Create new OpenAPI generator with custom options
    pub fn with_options(config: GatewayConfig, options: OpenApiOptions) -> Self {
        Self {
            config,
            generation_timestamp: Utc::now(),
        }
    }

    /// Generate complete OpenAPI specification with optimizations
    pub fn generate_spec(&self) -> Value {
        let mut spec = json!({
            "openapi": "3.0.3",
            "info": self.generate_api_info(),
            "servers": self.generate_servers(),
            "paths": self.generate_paths(),
            "components": self.generate_components(),
            "tags": self.generate_tags(),
            "security": [
                {"BearerAuth": []},
                {"TenantAuth": []}
            ]
        });

        // Add extensions for additional metadata
        spec["x-generated-at"] = Value::String(self.generation_timestamp.to_rfc3339());
        spec["x-generator"] = Value::String("SmartTicket Gateway".to_string());
        spec["x-version"] = Value::String(env!("CARGO_PKG_VERSION").to_string());

        // Add performance hints
        spec["x-performance"] = json!({
            "load_time_optimized": true,
            "lazy_loading_enabled": true,
            "virtual_scrolling": true,
            "caching_enabled": true
        });

        spec
    }

    /// Generate optimized OpenAPI specification for better performance
    pub fn generate_optimized_spec(&self) -> Value {
        let mut spec = self.generate_spec();

        // Add compression hints
        spec["x-compression"] = json!({
            "enabled": true,
            "algorithms": ["gzip", "br"],
            "min_size_bytes": 1024
        });

        // Add caching directives
        spec["x-cache-control"] = json!({
            "max-age": 300, // 5 minutes
            "stale-while-revalidate": 60, // 1 minute
            "vary": ["Authorization", "X-Tenant-ID"]
        });

        // Add progressive loading hints
        spec["x-progressive-loading"] = json!({
            "enabled": true,
            "chunks": [
                "core",
                "auth",
                "models",
                "advanced"
            ]
        });

        // Add bundle optimization hints
        spec["x-bundle"] = json!({
            "tree-shaking": true,
            "chunk-splitting": true,
            "code-splitting": true
        });

        spec
    }

    /// Generate API information with dynamic content
    fn generate_api_info(&self) -> Value {
        let mut description = r#"
B2B multi-tenant ticketing and knowledge collaboration platform API.

This API provides comprehensive functionality for:
- Multi-tenant management with data isolation
- Ticket lifecycle management with SLA tracking
- Knowledge base management and search
- User and role management
- Real-time notifications and updates
- Advanced analytics and reporting

## Authentication
Most endpoints require JWT authentication. Include the token in the Authorization header:
```
Authorization: Bearer <your-jwt-token>
```

## Multi-tenancy
All requests must include the tenant identifier:
```
X-Tenant-ID: <your-tenant-id>
```

## Rate Limiting
API endpoints are rate-limited to ensure fair usage. Check the `X-RateLimit-*` headers for current limits.

## Pagination
List endpoints support pagination using `page` and `page_size` parameters.
        "#.to_string();

        // Add dynamic information based on environment
        if self.config.http_port != 3286 {
            description.push_str(&format!("\n\n**Development Server**: Running on port {}", self.config.http_port));
        }

        json!({
            "title": "🚀 SmartTicket API Gateway",
            "description": description,
            "version": "1.0.0",
            "termsOfService": "https://smartticket.com/terms",
            "contact": {
                "name": "SmartTicket API Support",
                "email": "api-support@smartticket.com",
                "url": "https://smartticket.com/support",
                "x-twitter": "smartticketapi"
            },
            "license": {
                "name": "Commercial License",
                "url": "https://smartticket.com/license"
            },
            "x-logo": {
                "url": "https://smartticket.com/logo.png",
                "backgroundColor": "#667eea",
                "altText": "SmartTicket API"
            }
        })
    }

    /// Generate server information dynamically
    fn generate_servers(&self) -> Vec<Value> {
        let mut servers = vec![
            json!({
                "url": format!("http://localhost:{}/v1", self.config.http_port),
                "description": "Development server",
                "x-server-type": "development"
            })
        ];

        // Add environment-specific servers
        if cfg!(debug_assertions) {
            servers.push(json!({
                "url": "http://localhost:3286/v1",
                "description": "Local development",
                "x-server-type": "local"
            }));
        }

        servers.extend_from_slice(&[
            json!({
                "url": "https://staging-api.smartticket.com/v1",
                "description": "Staging environment",
                "x-server-type": "staging"
            }),
            json!({
                "url": "https://api.smartticket.com/v1",
                "description": "Production environment",
                "x-server-type": "production"
            })
        ]);

        servers
    }

    /// Generate API tags for organization
    fn generate_tags(&self) -> Vec<Value> {
        vec![
            json!({
                "name": "Authentication",
                "description": "User authentication and token management",
                "x-icon": "🔐"
            }),
            json!({
                "name": "User Management",
                "description": "User CRUD operations and profile management",
                "x-icon": "👥"
            }),
            json!({
                "name": "Tenant Management",
                "description": "Multi-tenant configuration and management",
                "x-icon": "🏢"
            }),
            json!({
                "name": "Ticket Management",
                "description": "Ticket lifecycle and operations",
                "x-icon": "🎫"
            }),
            json!({
                "name": "Knowledge Base",
                "description": "Knowledge articles and documentation",
                "x-icon": "📚"
            }),
            json!({
                "name": "SLA Management",
                "description": "Service Level Agreement configuration and monitoring",
                "x-icon": "📊"
            }),
            json!({
                "name": "Roles & Permissions",
                "description": "Role-based access control",
                "x-icon": "🔑"
            }),
            json!({
                "name": "Health & Monitoring",
                "description": "System health and monitoring endpoints",
                "x-icon": "💓"
            })
        ]
    }

    /// Generate comprehensive API paths
    fn generate_paths(&self) -> Value {
        let mut paths = serde_json::Map::new();

        // Health check endpoint
        paths.insert("/health".to_string(), json!({
            "get": {
                "summary": "Health Check",
                "description": "Check if the API and its services are healthy",
                "tags": ["Health"],
                "responses": {
                    "200": {
                        "description": "Service is healthy",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/HealthResponse" }
                            }
                        }
                    },
                    "503": {
                        "description": "Service is unhealthy",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/ApiResponse" }
                            }
                        }
                    }
                }
            }
        }));

        // Authentication endpoints
        for (path, endpoint) in self.generate_auth_paths() {
            paths.insert(path, endpoint);
        }

        // User management endpoints
        for (path, endpoint) in self.generate_user_paths() {
            paths.insert(path, endpoint);
        }

        // Tenant management endpoints
        for (path, endpoint) in self.generate_tenant_paths() {
            paths.insert(path, endpoint);
        }

        // Ticket management endpoints
        for (path, endpoint) in self.generate_ticket_paths() {
            paths.insert(path, endpoint);
        }

        // Knowledge base endpoints
        for (path, endpoint) in self.generate_knowledge_paths() {
            paths.insert(path, endpoint);
        }

        // SLA management endpoints
        for (path, endpoint) in self.generate_sla_paths() {
            paths.insert(path, endpoint);
        }

        // Role and permission endpoints
        for (path, endpoint) in self.generate_role_permission_paths() {
            paths.insert(path, endpoint);
        }

        Value::Object(paths)
    }

    /// Generate enhanced authentication paths
    fn generate_auth_paths(&self) -> HashMap<String, Value> {
        let mut paths = HashMap::new();

        paths.insert("/auth/v1/login".to_string(), json!({
            "post": {
                "summary": "User Login",
                "description": "Authenticate user credentials and return JWT access token",
                "tags": ["Authentication"],
                "operationId": "loginUser",
                "requestBody": {
                    "required": true,
                    "description": "User login credentials",
                    "content": {
                        "application/json": {
                            "schema": { "$ref": "#/components/schemas/LoginRequest" },
                            "example": {
                                "email": "john.doe@company.com",
                                "password": "SecurePass123!",
                                "tenant_id": "tenant-uuid-here"
                            }
                        }
                    }
                },
                "responses": {
                    "200": {
                        "description": "Login successful - JWT tokens returned",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/LoginResponse" },
                                "example": {
                                    "success": true,
                                    "data": {
                                        "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
                                        "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
                                        "expires_in": 3600,
                                        "token_type": "Bearer",
                                        "user": {
                                            "id": "123e4567-e89b-12d3-a456-426614174000",
                                            "email": "john.doe@company.com",
                                            "first_name": "John",
                                            "last_name": "Doe",
                                            "role": "engineer",
                                            "tenant_id": "tenant-uuid-here"
                                        }
                                    },
                                    "message": "Login successful",
                                    "timestamp": 1640995200
                                }
                            }
                        }
                    },
                    "400": {
                        "description": "Bad request - Invalid input data",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/ApiResponse" },
                                "example": {
                                    "success": false,
                                    "errors": [{
                                        "code": "VALIDATION_ERROR",
                                        "message": "Invalid email format",
                                        "field": "email"
                                    }]
                                }
                            }
                        }
                    },
                    "401": {
                        "description": "Unauthorized - Invalid credentials",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/ApiResponse" },
                                "example": {
                                    "success": false,
                                    "errors": [{
                                        "code": "INVALID_CREDENTIALS",
                                        "message": "Email or password is incorrect"
                                    }]
                                }
                            }
                        }
                    },
                    "429": {
                        "description": "Too many requests - Rate limit exceeded",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/ApiResponse" },
                                "example": {
                                    "success": false,
                                    "errors": [{
                                        "code": "RATE_LIMIT_EXCEEDED",
                                        "message": "Too many login attempts. Please try again later."
                                    }]
                                }
                            }
                        }
                    }
                }
            }
        }));

        paths.insert("/auth/v1/refresh".to_string(), json!({
            "post": {
                "summary": "Refresh Access Token",
                "description": "Use refresh token to obtain a new access token",
                "tags": ["Authentication"],
                "operationId": "refreshToken",
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "requestBody": {
                    "required": true,
                    "description": "Refresh token for obtaining new access token",
                    "content": {
                        "application/json": {
                            "schema": { "$ref": "#/components/schemas/RefreshTokenRequest" },
                            "example": {
                                "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
                            }
                        }
                    }
                },
                "responses": {
                    "200": {
                        "description": "Token refreshed successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/RefreshTokenResponse" },
                                "example": {
                                    "success": true,
                                    "data": {
                                        "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
                                        "expires_in": 3600,
                                        "token_type": "Bearer"
                                    },
                                    "message": "Token refreshed successfully"
                                }
                            }
                        }
                    },
                    "401": {
                        "description": "Invalid or expired refresh token",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/ApiResponse" }
                            }
                        }
                    }
                }
            }
        }));

        paths.insert("/auth/v1/logout".to_string(), json!({
            "post": {
                "summary": "User Logout",
                "description": "Invalidate current user session and tokens",
                "tags": ["Authentication"],
                "operationId": "logoutUser",
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "responses": {
                    "200": {
                        "description": "Logout successful",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/ApiResponse" },
                                "example": {
                                    "success": true,
                                    "message": "Logout successful"
                                }
                            }
                        }
                    },
                    "401": {
                        "description": "Invalid or expired token",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/ApiResponse" }
                            }
                        }
                    }
                }
            }
        }));

        paths
    }

    /// Generate user management paths
    fn generate_user_paths(&self) -> HashMap<String, Value> {
        let mut paths = HashMap::new();

        let users_endpoint = json!({
            "get": {
                "summary": "List Users",
                "description": "Retrieve paginated list of users",
                "tags": ["User Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "page",
                        "in": "query",
                        "description": "Page number",
                        "schema": { "type": "integer", "minimum": 1, "default": 1 }
                    },
                    {
                        "name": "page_size",
                        "in": "query",
                        "description": "Number of items per page",
                        "schema": { "type": "integer", "minimum": 1, "maximum": 100, "default": 20 }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Users retrieved successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/PaginatedUsersResponse" }
                            }
                        }
                    }
                }
            },
            "post": {
                "summary": "Create User",
                "description": "Create a new user",
                "tags": ["User Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "requestBody": {
                    "required": true,
                    "content": {
                        "application/json": {
                            "schema": { "$ref": "#/components/schemas/CreateUserRequest" }
                        }
                    }
                },
                "responses": {
                    "201": {
                        "description": "User created successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/UserResponse" }
                            }
                        }
                    }
                }
            }
        });
        paths.insert("/v1/users".to_string(), users_endpoint);

        paths
    }

    /// Generate tenant management paths
    fn generate_tenant_paths(&self) -> HashMap<String, Value> {
        let mut paths = HashMap::new();

        // Tenant management endpoints
        paths.insert("/v1/tenants".to_string(), json!({
            "get": {
                "summary": "List Tenants",
                "description": "Retrieve paginated list of tenants",
                "tags": ["Tenant Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "page",
                        "in": "query",
                        "schema": { "type": "integer", "minimum": 1, "default": 1 }
                    },
                    {
                        "name": "page_size",
                        "in": "query",
                        "schema": { "type": "integer", "minimum": 1, "maximum": 100, "default": 20 }
                    },
                    {
                        "name": "status",
                        "in": "query",
                        "schema": {
                            "type": "string",
                            "enum": ["active", "inactive", "suspended"]
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Tenants retrieved successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/PaginatedTenantsResponse" }
                            }
                        }
                    }
                }
            },
            "post": {
                "summary": "Create Tenant",
                "description": "Create a new tenant",
                "tags": ["Tenant Management"],
                "security": [{"BearerAuth": []}],
                "requestBody": {
                    "required": true,
                    "content": {
                        "application/json": {
                            "schema": { "$ref": "#/components/schemas/CreateTenantRequest" }
                        }
                    }
                },
                "responses": {
                    "201": {
                        "description": "Tenant created successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/TenantResponse" }
                            }
                        }
                    }
                }
            }
        }));

        // Specific tenant operations
        paths.insert("/v1/tenants/{tenant_id}".to_string(), json!({
            "get": {
                "summary": "Get Tenant",
                "description": "Retrieve details of a specific tenant",
                "tags": ["Tenant Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "tenant_id",
                        "in": "path",
                        "required": true,
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Tenant retrieved successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/TenantResponse" }
                            }
                        }
                    }
                }
            },
            "put": {
                "summary": "Update Tenant",
                "description": "Update tenant information",
                "tags": ["Tenant Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "tenant_id",
                        "in": "path",
                        "required": true,
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "requestBody": {
                    "required": true,
                    "content": {
                        "application/json": {
                            "schema": { "$ref": "#/components/schemas/UpdateTenantRequest" }
                        }
                    }
                },
                "responses": {
                    "200": {
                        "description": "Tenant updated successfully"
                    }
                }
            },
            "delete": {
                "summary": "Delete Tenant",
                "description": "Delete a tenant",
                "tags": ["Tenant Management"],
                "security": [{"BearerAuth": []}],
                "parameters": [
                    {
                        "name": "tenant_id",
                        "in": "path",
                        "required": true,
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "responses": {
                    "204": {
                        "description": "Tenant deleted successfully"
                    }
                }
            }
        }));

        // Tenant users management
        paths.insert("/v1/tenants/{tenant_id}/users".to_string(), json!({
            "get": {
                "summary": "Get Tenant Users",
                "description": "Retrieve users belonging to a specific tenant",
                "tags": ["Tenant Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "tenant_id",
                        "in": "path",
                        "required": true,
                        "schema": { "type": "string", "format": "uuid" }
                    },
                    {
                        "name": "page",
                        "in": "query",
                        "schema": { "type": "integer", "minimum": 1, "default": 1 }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Tenant users retrieved successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/PaginatedUsersResponse" }
                            }
                        }
                    }
                }
            },
            "post": {
                "summary": "Add Tenant User",
                "description": "Add a user to a tenant",
                "tags": ["Tenant Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "tenant_id",
                        "in": "path",
                        "required": true,
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "requestBody": {
                    "required": true,
                    "content": {
                        "application/json": {
                            "schema": { "$ref": "#/components/schemas/AddTenantUserRequest" }
                        }
                    }
                },
                "responses": {
                    "201": {
                        "description": "User added to tenant successfully"
                    }
                }
            }
        }));

        paths.insert("/v1/tenants/{tenant_id}/users/{user_id}".to_string(), json!({
            "delete": {
                "summary": "Remove Tenant User",
                "description": "Remove a user from a tenant",
                "tags": ["Tenant Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "tenant_id",
                        "in": "path",
                        "required": true,
                        "schema": { "type": "string", "format": "uuid" }
                    },
                    {
                        "name": "user_id",
                        "in": "path",
                        "required": true,
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "responses": {
                    "204": {
                        "description": "User removed from tenant successfully"
                    }
                }
            }
        }));

        // Tenant settings
        paths.insert("/v1/tenants/{tenant_id}/settings".to_string(), json!({
            "get": {
                "summary": "Get Tenant Settings",
                "description": "Retrieve tenant configuration settings",
                "tags": ["Tenant Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "tenant_id",
                        "in": "path",
                        "required": true,
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Tenant settings retrieved successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/TenantSettingsResponse" }
                            }
                        }
                    }
                }
            },
            "put": {
                "summary": "Update Tenant Settings",
                "description": "Update tenant configuration settings",
                "tags": ["Tenant Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "tenant_id",
                        "in": "path",
                        "required": true,
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "requestBody": {
                    "required": true,
                    "content": {
                        "application/json": {
                            "schema": { "$ref": "#/components/schemas/UpdateTenantSettingsRequest" }
                        }
                    }
                },
                "responses": {
                    "200": {
                        "description": "Tenant settings updated successfully"
                    }
                }
            }
        }));

        // Tenant status management
        paths.insert("/v1/tenants/{tenant_id}/status".to_string(), json!({
            "put": {
                "summary": "Update Tenant Status",
                "description": "Update tenant status (activate/suspend)",
                "tags": ["Tenant Management"],
                "security": [{"BearerAuth": []}],
                "parameters": [
                    {
                        "name": "tenant_id",
                        "in": "path",
                        "required": true,
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "requestBody": {
                    "required": true,
                    "content": {
                        "application/json": {
                            "schema": { "$ref": "#/components/schemas/UpdateTenantStatusRequest" }
                        }
                    }
                },
                "responses": {
                    "200": {
                        "description": "Tenant status updated successfully"
                    }
                }
            }
        }));

        paths
    }

    /// Generate comprehensive ticket management paths
    fn generate_ticket_paths(&self) -> HashMap<String, Value> {
        let mut paths = HashMap::new();

        // List tickets with filtering and pagination
        paths.insert("/v1/tickets".to_string(), json!({
            "get": {
                "summary": "List Tickets",
                "description": "Retrieve paginated list of tickets with optional filtering",
                "tags": ["Ticket Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "page",
                        "in": "query",
                        "description": "Page number",
                        "schema": { "type": "integer", "minimum": 1, "default": 1 }
                    },
                    {
                        "name": "page_size",
                        "in": "query",
                        "description": "Number of items per page (max 100)",
                        "schema": { "type": "integer", "minimum": 1, "maximum": 100, "default": 20 }
                    },
                    {
                        "name": "status",
                        "in": "query",
                        "description": "Filter by ticket status",
                        "schema": {
                            "type": "string",
                            "enum": ["open", "in_progress", "pending_customer", "resolved", "closed", "cancelled"]
                        }
                    },
                    {
                        "name": "priority",
                        "in": "query",
                        "description": "Filter by priority level",
                        "schema": {
                            "type": "string",
                            "enum": ["low", "medium", "high", "urgent", "critical"]
                        }
                    },
                    {
                        "name": "assigned_to",
                        "in": "query",
                        "description": "Filter by assigned user ID",
                        "schema": { "type": "string", "format": "uuid" }
                    },
                    {
                        "name": "customer_id",
                        "in": "query",
                        "description": "Filter by customer ID",
                        "schema": { "type": "string", "format": "uuid" }
                    },
                    {
                        "name": "created_after",
                        "in": "query",
                        "description": "Filter tickets created after this timestamp",
                        "schema": { "type": "integer", "format": "int64" }
                    },
                    {
                        "name": "created_before",
                        "in": "query",
                        "description": "Filter tickets created before this timestamp",
                        "schema": { "type": "integer", "format": "int64" }
                    },
                    {
                        "name": "search",
                        "in": "query",
                        "description": "Search in ticket title and description",
                        "schema": { "type": "string" }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Tickets retrieved successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/PaginatedTicketsResponse" },
                                "example": {
                                    "success": true,
                                    "data": {
                                        "items": [
                                            {
                                                "id": "ticket-123",
                                                "title": "Login issue with CRM system",
                                                "description": "User cannot access CRM portal",
                                                "status": "open",
                                                "priority": "high",
                                                "customer_id": "customer-456",
                                                "assigned_to": "user-789",
                                                "created_at": 1640995200,
                                                "updated_at": 1640998800,
                                                "due_date": 1641081600,
                                                "sla_status": "on_track",
                                                "tags": ["login", "crm", "urgent"]
                                            }
                                        ],
                                        "pagination": {
                                            "page": 1,
                                            "page_size": 20,
                                            "total": 156,
                                            "total_pages": 8,
                                            "has_next": true,
                                            "has_prev": false
                                        }
                                    }
                                }
                            }
                        }
                    },
                    "401": {
                        "description": "Unauthorized - Invalid or missing authentication"
                    },
                    "403": {
                        "description": "Forbidden - Insufficient permissions to view tickets"
                    }
                }
            },
            "post": {
                "summary": "Create Ticket",
                "description": "Create a new support ticket",
                "tags": ["Ticket Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "requestBody": {
                    "required": true,
                    "description": "Ticket creation details",
                    "content": {
                        "application/json": {
                            "schema": { "$ref": "#/components/schemas/CreateTicketRequest" },
                            "example": {
                                "title": "Unable to export reports",
                                "description": "When trying to export monthly reports, the system shows an error message",
                                "priority": "medium",
                                "customer_id": "customer-456",
                                "assigned_to": "user-789",
                                "category": "technical",
                                "tags": ["reports", "export", "error"],
                                "custom_fields": {
                                    "browser": "Chrome 96.0",
                                    "error_code": "EXP-501"
                                }
                            }
                        }
                    }
                },
                "responses": {
                    "201": {
                        "description": "Ticket created successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/TicketResponse" }
                            }
                        }
                    },
                    "400": {
                        "description": "Bad request - Invalid ticket data"
                    }
                }
            }
        }));

        // Get specific ticket
        paths.insert("/v1/tickets/{ticket_id}".to_string(), json!({
            "get": {
                "summary": "Get Ticket Details",
                "description": "Retrieve detailed information about a specific ticket",
                "tags": ["Ticket Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "ticket_id",
                        "in": "path",
                        "required": true,
                        "description": "Unique identifier of the ticket",
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Ticket details retrieved successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/TicketDetailsResponse" }
                            }
                        }
                    },
                    "404": {
                        "description": "Ticket not found"
                    }
                }
            },
            "put": {
                "summary": "Update Ticket",
                "description": "Update ticket information",
                "tags": ["Ticket Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "ticket_id",
                        "in": "path",
                        "required": true,
                        "description": "Unique identifier of the ticket",
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "requestBody": {
                    "required": true,
                    "content": {
                        "application/json": {
                            "schema": { "$ref": "#/components/schemas/UpdateTicketRequest" }
                        }
                    }
                },
                "responses": {
                    "200": {
                        "description": "Ticket updated successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/TicketResponse" }
                            }
                        }
                    },
                    "404": {
                        "description": "Ticket not found"
                    }
                }
            },
            "delete": {
                "summary": "Delete Ticket",
                "description": "Soft delete a ticket (mark as deleted)",
                "tags": ["Ticket Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "ticket_id",
                        "in": "path",
                        "required": true,
                        "description": "Unique identifier of the ticket",
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "responses": {
                    "204": {
                        "description": "Ticket deleted successfully"
                    },
                    "404": {
                        "description": "Ticket not found"
                    }
                }
            }
        }));

        // Ticket status transitions
        paths.insert("/v1/tickets/{ticket_id}/status".to_string(), json!({
            "patch": {
                "summary": "Update Ticket Status",
                "description": "Change the status of a ticket with automatic SLA recalculation",
                "tags": ["Ticket Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "ticket_id",
                        "in": "path",
                        "required": true,
                        "description": "Unique identifier of the ticket",
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "requestBody": {
                    "required": true,
                    "content": {
                        "application/json": {
                            "schema": { "$ref": "#/components/schemas/UpdateTicketStatusRequest" },
                            "example": {
                                "status": "in_progress",
                                "comment": "Started investigating the issue. Found root cause in database connection.",
                                "notify_customer": true
                            }
                        }
                    }
                },
                "responses": {
                    "200": {
                        "description": "Ticket status updated successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/TicketResponse" }
                            }
                        }
                    }
                }
            }
        }));

        // Ticket assignments
        paths.insert("/v1/tickets/{ticket_id}/assign".to_string(), json!({
            "patch": {
                "summary": "Assign Ticket",
                "description": "Assign or reassign a ticket to a user",
                "tags": ["Ticket Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "ticket_id",
                        "in": "path",
                        "required": true,
                        "description": "Unique identifier of the ticket",
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "requestBody": {
                    "required": true,
                    "content": {
                        "application/json": {
                            "schema": { "$ref": "#/components/schemas/AssignTicketRequest" },
                            "example": {
                                "assigned_to": "user-789",
                                "comment": "Assigning to senior engineer due to complexity"
                            }
                        }
                    }
                },
                "responses": {
                    "200": {
                        "description": "Ticket assigned successfully"
                    }
                }
            }
        }));

        // Ticket comments
        paths.insert("/v1/tickets/{ticket_id}/comments".to_string(), json!({
            "get": {
                "summary": "List Ticket Comments",
                "description": "Retrieve all comments for a specific ticket",
                "tags": ["Ticket Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "ticket_id",
                        "in": "path",
                        "required": true,
                        "description": "Unique identifier of the ticket",
                        "schema": { "type": "string", "format": "uuid" }
                    },
                    {
                        "name": "page",
                        "in": "query",
                        "description": "Page number",
                        "schema": { "type": "integer", "minimum": 1, "default": 1 }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Comments retrieved successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/PaginatedCommentsResponse" }
                            }
                        }
                    }
                }
            },
            "post": {
                "summary": "Add Ticket Comment",
                "description": "Add a new comment to a ticket",
                "tags": ["Ticket Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "ticket_id",
                        "in": "path",
                        "required": true,
                        "description": "Unique identifier of the ticket",
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "requestBody": {
                    "required": true,
                    "content": {
                        "application/json": {
                            "schema": { "$ref": "#/components/schemas/CreateCommentRequest" },
                            "example": {
                                "content": "I've identified the issue. The database connection pool is exhausted. Restarting the service should resolve this temporarily.",
                                "is_internal": false,
                                "attachments": [
                                    {
                                        "name": "error-log.txt",
                                        "url": "https://storage.example.com/attachments/error-log-123.txt",
                                        "size": 2048
                                    }
                                ]
                            }
                        }
                    }
                },
                "responses": {
                    "201": {
                        "description": "Comment added successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/CommentResponse" }
                            }
                        }
                    }
                }
            }
        }));

        paths
    }

    /// Generate knowledge base paths
    fn generate_knowledge_paths(&self) -> HashMap<String, Value> {
        let mut paths = HashMap::new();

        // Knowledge articles management
        paths.insert("/v1/knowledge/articles".to_string(), json!({
            "get": {
                "summary": "Search Knowledge Articles",
                "description": "Search knowledge base articles with filters and pagination",
                "tags": ["Knowledge Base"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "q",
                        "in": "query",
                        "description": "Search query for articles",
                        "schema": { "type": "string" }
                    },
                    {
                        "name": "category",
                        "in": "query",
                        "description": "Filter by category",
                        "schema": {
                            "type": "string",
                            "enum": ["getting-started", "troubleshooting", "best-practices", "api-reference", "tutorials"]
                        }
                    },
                    {
                        "name": "status",
                        "in": "query",
                        "description": "Filter by publication status",
                        "schema": {
                            "type": "string",
                            "enum": ["draft", "published", "archived"]
                        }
                    },
                    {
                        "name": "page",
                        "in": "query",
                        "description": "Page number",
                        "schema": { "type": "integer", "minimum": 1, "default": 1 }
                    },
                    {
                        "name": "page_size",
                        "in": "query",
                        "description": "Number of items per page",
                        "schema": { "type": "integer", "minimum": 1, "maximum": 100, "default": 20 }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Articles retrieved successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/PaginatedKnowledgeArticlesResponse" }
                            }
                        }
                    }
                }
            },
            "post": {
                "summary": "Create Knowledge Article",
                "description": "Create a new knowledge base article",
                "tags": ["Knowledge Base"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "requestBody": {
                    "required": true,
                    "content": {
                        "application/json": {
                            "schema": { "$ref": "#/components/schemas/CreateKnowledgeArticleRequest" }
                        }
                    }
                },
                "responses": {
                    "201": {
                        "description": "Article created successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/KnowledgeArticleResponse" }
                            }
                        }
                    }
                }
            }
        }));

        // Specific article operations
        paths.insert("/v1/knowledge/articles/{article_id}".to_string(), json!({
            "get": {
                "summary": "Get Knowledge Article",
                "description": "Retrieve a specific knowledge article",
                "tags": ["Knowledge Base"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "article_id",
                        "in": "path",
                        "required": true,
                        "description": "Unique identifier of the article",
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Article retrieved successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/KnowledgeArticleResponse" }
                            }
                        }
                    }
                }
            },
            "put": {
                "summary": "Update Knowledge Article",
                "description": "Update an existing knowledge article",
                "tags": ["Knowledge Base"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "article_id",
                        "in": "path",
                        "required": true,
                        "description": "Unique identifier of the article",
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "requestBody": {
                    "required": true,
                    "content": {
                        "application/json": {
                            "schema": { "$ref": "#/components/schemas/UpdateKnowledgeArticleRequest" }
                        }
                    }
                },
                "responses": {
                    "200": {
                        "description": "Article updated successfully"
                    }
                }
            },
            "delete": {
                "summary": "Delete Knowledge Article",
                "description": "Delete a knowledge article",
                "tags": ["Knowledge Base"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "article_id",
                        "in": "path",
                        "required": true,
                        "description": "Unique identifier of the article",
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "responses": {
                    "204": {
                        "description": "Article deleted successfully"
                    }
                }
            }
        }));

        // Article feedback
        paths.insert("/v1/knowledge/articles/{article_id}/feedback".to_string(), json!({
            "post": {
                "summary": "Submit Article Feedback",
                "description": "Submit feedback on a knowledge article",
                "tags": ["Knowledge Base"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "article_id",
                        "in": "path",
                        "required": true,
                        "description": "Unique identifier of the article",
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "requestBody": {
                    "required": true,
                    "content": {
                        "application/json": {
                            "schema": { "$ref": "#/components/schemas/ArticleFeedbackRequest" }
                        }
                    }
                },
                "responses": {
                    "201": {
                        "description": "Feedback submitted successfully"
                    }
                }
            }
        }));

        paths
    }

    /// Generate SLA management paths
    fn generate_sla_paths(&self) -> HashMap<String, Value> {
        let mut paths = HashMap::new();

        // SLA policies
        paths.insert("/v1/sla/policies".to_string(), json!({
            "get": {
                "summary": "List SLA Policies",
                "description": "Retrieve paginated list of SLA policies",
                "tags": ["SLA Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "page",
                        "in": "query",
                        "schema": { "type": "integer", "minimum": 1, "default": 1 }
                    },
                    {
                        "name": "page_size",
                        "in": "query",
                        "schema": { "type": "integer", "minimum": 1, "maximum": 100, "default": 20 }
                    },
                    {
                        "name": "status",
                        "in": "query",
                        "schema": {
                            "type": "string",
                            "enum": ["active", "inactive", "draft"]
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "SLA policies retrieved successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/PaginatedSLAPoliciesResponse" }
                            }
                        }
                    }
                }
            },
            "post": {
                "summary": "Create SLA Policy",
                "description": "Create a new SLA policy",
                "tags": ["SLA Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "requestBody": {
                    "required": true,
                    "content": {
                        "application/json": {
                            "schema": { "$ref": "#/components/schemas/CreateSLAPolicyRequest" }
                        }
                    }
                },
                "responses": {
                    "201": {
                        "description": "SLA policy created successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/SLAPolicyResponse" }
                            }
                        }
                    }
                }
            }
        }));

        // Specific SLA policy operations
        paths.insert("/v1/sla/policies/{policy_id}".to_string(), json!({
            "get": {
                "summary": "Get SLA Policy",
                "description": "Retrieve details of a specific SLA policy",
                "tags": ["SLA Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "policy_id",
                        "in": "path",
                        "required": true,
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "SLA policy retrieved successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/SLAPolicyResponse" }
                            }
                        }
                    }
                }
            },
            "put": {
                "summary": "Update SLA Policy",
                "description": "Update an existing SLA policy",
                "tags": ["SLA Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "policy_id",
                        "in": "path",
                        "required": true,
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "requestBody": {
                    "required": true,
                    "content": {
                        "application/json": {
                            "schema": { "$ref": "#/components/schemas/UpdateSLAPolicyRequest" }
                        }
                    }
                },
                "responses": {
                    "200": {
                        "description": "SLA policy updated successfully"
                    }
                }
            },
            "delete": {
                "summary": "Delete SLA Policy",
                "description": "Delete an SLA policy",
                "tags": ["SLA Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "policy_id",
                        "in": "path",
                        "required": true,
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "responses": {
                    "204": {
                        "description": "SLA policy deleted successfully"
                    }
                }
            }
        }));

        // SLA policy activation/deactivation
        paths.insert("/v1/sla/policies/{policy_id}/activate".to_string(), json!({
            "post": {
                "summary": "Activate SLA Policy",
                "description": "Activate an SLA policy",
                "tags": ["SLA Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "policy_id",
                        "in": "path",
                        "required": true,
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "SLA policy activated successfully"
                    }
                }
            }
        }));

        paths.insert("/v1/sla/policies/{policy_id}/deactivate".to_string(), json!({
            "post": {
                "summary": "Deactivate SLA Policy",
                "description": "Deactivate an SLA policy",
                "tags": ["SLA Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "policy_id",
                        "in": "path",
                        "required": true,
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "SLA policy deactivated successfully"
                    }
                }
            }
        }));

        // SLA agreements
        paths.insert("/v1/sla/agreements".to_string(), json!({
            "get": {
                "summary": "List SLA Agreements",
                "description": "Retrieve list of SLA agreements",
                "tags": ["SLA Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "page",
                        "in": "query",
                        "schema": { "type": "integer", "minimum": 1, "default": 1 }
                    },
                    {
                        "name": "page_size",
                        "in": "query",
                        "schema": { "type": "integer", "minimum": 1, "maximum": 100, "default": 20 }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "SLA agreements retrieved successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/PaginatedSLAAgreementsResponse" }
                            }
                        }
                    }
                }
            },
            "post": {
                "summary": "Create SLA Agreement",
                "description": "Create a new SLA agreement",
                "tags": ["SLA Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "requestBody": {
                    "required": true,
                    "content": {
                        "application/json": {
                            "schema": { "$ref": "#/components/schemas/CreateSLAAgreementRequest" }
                        }
                    }
                },
                "responses": {
                    "201": {
                        "description": "SLA agreement created successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/SLAAgreementResponse" }
                            }
                        }
                    }
                }
            }
        }));

        paths.insert("/v1/sla/agreements/{agreement_id}".to_string(), json!({
            "get": {
                "summary": "Get SLA Agreement",
                "description": "Retrieve details of a specific SLA agreement",
                "tags": ["SLA Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "agreement_id",
                        "in": "path",
                        "required": true,
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "SLA agreement retrieved successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/SLAAgreementResponse" }
                            }
                        }
                    }
                }
            },
            "put": {
                "summary": "Update SLA Agreement",
                "description": "Update an existing SLA agreement",
                "tags": ["SLA Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "agreement_id",
                        "in": "path",
                        "required": true,
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "requestBody": {
                    "required": true,
                    "content": {
                        "application/json": {
                            "schema": { "$ref": "#/components/schemas/UpdateSLAAgreementRequest" }
                        }
                    }
                },
                "responses": {
                    "200": {
                        "description": "SLA agreement updated successfully"
                    }
                }
            }
        }));

        // SLA metrics
        paths.insert("/v1/sla/metrics".to_string(), json!({
            "get": {
                "summary": "Get SLA Metrics",
                "description": "Retrieve SLA performance metrics",
                "tags": ["SLA Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "time_period",
                        "in": "query",
                        "description": "Time period for metrics (7d, 30d, 90d)",
                        "schema": {
                            "type": "string",
                            "enum": ["7d", "30d", "90d"],
                            "default": "30d"
                        }
                    },
                    {
                        "name": "policy_id",
                        "in": "query",
                        "description": "Filter by specific policy",
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "SLA metrics retrieved successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/SLAMetricsResponse" }
                            }
                        }
                    }
                }
            }
        }));

        // SLA breaches
        paths.insert("/v1/sla/breaches".to_string(), json!({
            "get": {
                "summary": "List SLA Breaches",
                "description": "Retrieve list of SLA breaches",
                "tags": ["SLA Management"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "page",
                        "in": "query",
                        "schema": { "type": "integer", "minimum": 1, "default": 1 }
                    },
                    {
                        "name": "page_size",
                        "in": "query",
                        "schema": { "type": "integer", "minimum": 1, "maximum": 100, "default": 20 }
                    },
                    {
                        "name": "severity",
                        "in": "query",
                        "schema": {
                            "type": "string",
                            "enum": ["minor", "major", "critical"]
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "SLA breaches retrieved successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/PaginatedSLABreachesResponse" }
                            }
                        }
                    }
                }
            }
        }));

        paths
    }

    /// Generate role and permission paths
    fn generate_role_permission_paths(&self) -> HashMap<String, Value> {
        let mut paths = HashMap::new();

        // Role management endpoints
        paths.insert("/v1/roles".to_string(), json!({
            "get": {
                "summary": "List Roles",
                "description": "Retrieve paginated list of roles",
                "tags": ["Roles & Permissions"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "page",
                        "in": "query",
                        "schema": { "type": "integer", "minimum": 1, "default": 1 }
                    },
                    {
                        "name": "page_size",
                        "in": "query",
                        "schema": { "type": "integer", "minimum": 1, "maximum": 100, "default": 20 }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Roles retrieved successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/PaginatedRolesResponse" }
                            }
                        }
                    }
                }
            },
            "post": {
                "summary": "Create Role",
                "description": "Create a new role",
                "tags": ["Roles & Permissions"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "requestBody": {
                    "required": true,
                    "content": {
                        "application/json": {
                            "schema": { "$ref": "#/components/schemas/CreateRoleRequest" }
                        }
                    }
                },
                "responses": {
                    "201": {
                        "description": "Role created successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/RoleResponse" }
                            }
                        }
                    }
                }
            }
        }));

        // Specific role operations
        paths.insert("/v1/roles/{role_id}".to_string(), json!({
            "get": {
                "summary": "Get Role",
                "description": "Retrieve details of a specific role",
                "tags": ["Roles & Permissions"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "role_id",
                        "in": "path",
                        "required": true,
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Role retrieved successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/RoleResponse" }
                            }
                        }
                    }
                }
            },
            "put": {
                "summary": "Update Role",
                "description": "Update an existing role",
                "tags": ["Roles & Permissions"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "role_id",
                        "in": "path",
                        "required": true,
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "requestBody": {
                    "required": true,
                    "content": {
                        "application/json": {
                            "schema": { "$ref": "#/components/schemas/UpdateRoleRequest" }
                        }
                    }
                },
                "responses": {
                    "200": {
                        "description": "Role updated successfully"
                    }
                }
            },
            "delete": {
                "summary": "Delete Role",
                "description": "Delete a role",
                "tags": ["Roles & Permissions"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "role_id",
                        "in": "path",
                        "required": true,
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "responses": {
                    "204": {
                        "description": "Role deleted successfully"
                    }
                }
            }
        }));

        // Role permissions management
        paths.insert("/v1/roles/{role_id}/permissions".to_string(), json!({
            "get": {
                "summary": "Get Role Permissions",
                "description": "Retrieve permissions for a specific role",
                "tags": ["Roles & Permissions"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "role_id",
                        "in": "path",
                        "required": true,
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Role permissions retrieved successfully",
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/RolePermissionsResponse" }
                            }
                        }
                    }
                }
            },
            "post": {
                "summary": "Add Role Permission",
                "description": "Add a permission to a role",
                "tags": ["Roles & Permissions"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "role_id",
                        "in": "path",
                        "required": true,
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "requestBody": {
                    "required": true,
                    "content": {
                        "application/json": {
                            "schema": { "$ref": "#/components/schemas/AddRolePermissionRequest" }
                        }
                    }
                },
                "responses": {
                    "201": {
                        "description": "Permission added to role successfully"
                    }
                }
            }
        }));

        paths.insert("/v1/roles/{role_id}/permissions/{permission_id}".to_string(), json!({
            "delete": {
                "summary": "Remove Role Permission",
                "description": "Remove a permission from a role",
                "tags": ["Roles & Permissions"],
                "security": [{"BearerAuth": []}, {"TenantAuth": []}],
                "parameters": [
                    {
                        "name": "role_id",
                        "in": "path",
                        "required": true,
                        "schema": { "type": "string", "format": "uuid" }
                    },
                    {
                        "name": "permission_id",
                        "in": "path",
                        "required": true,
                        "schema": { "type": "string", "format": "uuid" }
                    }
                ],
                "responses": {
                    "204": {
                        "description": "Permission removed from role successfully"
                    }
                }
            }
        }));

        paths
    }

    /// Generate comprehensive OpenAPI components
    fn generate_components(&self) -> Value {
        json!({
            "securitySchemes": {
                "BearerAuth": {
                    "type": "http",
                    "scheme": "bearer",
                    "bearerFormat": "JWT",
                    "description": "JWT access token obtained from login. Include in Authorization header.",
                    "x-example": "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
                },
                "TenantAuth": {
                    "type": "apiKey",
                    "in": "header",
                    "name": "X-Tenant-ID",
                    "description": "Tenant identifier for multi-tenant requests. Required for all authenticated endpoints.",
                    "x-example": "X-Tenant-ID: 123e4567-e89b-12d3-a456-426614174000"
                }
            },
            "schemas": {
                "ApiResponse": {
                    "type": "object",
                    "description": "Standard API response wrapper",
                    "properties": {
                        "success": {
                            "type": "boolean",
                            "description": "Whether the request was successful"
                        },
                        "message": {
                            "type": "string",
                            "description": "Human-readable message describing the result"
                        },
                        "data": {
                            "type": "object",
                            "description": "Response data payload",
                            "additionalProperties": true
                        },
                        "errors": {
                            "type": "array",
                            "description": "List of errors if request failed",
                            "items": { "$ref": "#/components/schemas/ApiError" }
                        },
                        "request_id": {
                            "type": "string",
                            "format": "uuid",
                            "description": "Unique identifier for tracking the request"
                        },
                        "timestamp": {
                            "type": "integer",
                            "description": "Unix timestamp of the response"
                        }
                    },
                    "required": ["success", "timestamp"]
                },
                "ApiError": {
                    "type": "object",
                    "required": ["code", "message"],
                    "properties": {
                        "code": { "type": "string" },
                        "message": { "type": "string" },
                        "details": { "type": "object", "additionalProperties": true },
                        "field": { "type": "string" }
                    }
                },
                "HealthResponse": {
                    "type": "object",
                    "properties": {
                        "status": { "type": "string" },
                        "version": { "type": "string" },
                        "timestamp": { "type": "integer" },
                        "services": {
                            "type": "object",
                            "additionalProperties": {
                                "$ref": "#/components/schemas/ServiceHealth"
                            }
                        }
                    }
                },
                "ServiceHealth": {
                    "type": "object",
                    "properties": {
                        "status": { "type": "string" },
                        "latency_ms": { "type": "integer" },
                        "error": { "type": "string" }
                    }
                },
                "LoginRequest": {
                    "type": "object",
                    "required": ["email", "password", "tenant_id"],
                    "properties": {
                        "email": { "type": "string", "format": "email" },
                        "password": { "type": "string", "minLength": 8 },
                        "tenant_id": { "type": "string" }
                    }
                },
                "LoginResponse": {
                    "type": "object",
                    "properties": {
                        "access_token": { "type": "string" },
                        "refresh_token": { "type": "string" },
                        "expires_in": { "type": "integer" },
                        "user": { "$ref": "#/components/schemas/User" }
                    }
                },
                "RefreshTokenRequest": {
                    "type": "object",
                    "required": ["refresh_token"],
                    "properties": {
                        "refresh_token": { "type": "string" }
                    }
                },
                "RefreshTokenResponse": {
                    "type": "object",
                    "properties": {
                        "access_token": { "type": "string" },
                        "expires_in": { "type": "integer" }
                    }
                },
                "User": {
                    "type": "object",
                    "properties": {
                        "id": { "type": "string", "format": "uuid" },
                        "email": { "type": "string", "format": "email" },
                        "first_name": { "type": "string" },
                        "last_name": { "type": "string" },
                        "role": { "type": "string", "enum": ["admin", "customer", "engineer", "se", "sales"] },
                        "is_active": { "type": "boolean" },
                        "created_at": { "type": "integer" },
                        "updated_at": { "type": "integer" }
                    }
                },
                "CreateUserRequest": {
                    "type": "object",
                    "required": ["email", "first_name", "last_name", "role"],
                    "properties": {
                        "email": { "type": "string", "format": "email" },
                        "first_name": { "type": "string" },
                        "last_name": { "type": "string" },
                        "role": { "type": "string", "enum": ["admin", "customer", "engineer", "se", "sales"] },
                        "password": { "type": "string", "minLength": 8 }
                    }
                },
                "UpdateUserRequest": {
                    "type": "object",
                    "properties": {
                        "first_name": { "type": "string" },
                        "last_name": { "type": "string" },
                        "role": { "type": "string", "enum": ["admin", "customer", "engineer", "se", "sales"] },
                        "is_active": { "type": "boolean" }
                    }
                },
                "UserResponse": {
                    "allOf": [
                        { "$ref": "#/components/schemas/ApiResponse" },
                        {
                            "type": "object",
                            "properties": {
                                "data": { "$ref": "#/components/schemas/User" }
                            }
                        }
                    ]
                },
                "PaginatedUsersResponse": {
                    "allOf": [
                        { "$ref": "#/components/schemas/ApiResponse" },
                        {
                            "type": "object",
                            "properties": {
                                "data": {
                                    "type": "object",
                                    "properties": {
                                        "items": {
                                            "type": "array",
                                            "items": { "$ref": "#/components/schemas/User" }
                                        },
                                        "pagination": { "$ref": "#/components/schemas/PaginationInfo" }
                                    }
                                }
                            }
                        }
                    ]
                },
                "PaginationInfo": {
                    "type": "object",
                    "properties": {
                        "page": { "type": "integer" },
                        "page_size": { "type": "integer" },
                        "total": { "type": "integer" },
                        "total_pages": { "type": "integer" },
                        "has_next": { "type": "boolean" },
                        "has_prev": { "type": "boolean" }
                    }
                },
                "Ticket": {
                    "type": "object",
                    "description": "Core ticket model",
                    "properties": {
                        "id": {
                            "type": "string",
                            "format": "uuid",
                            "description": "Unique ticket identifier"
                        },
                        "title": {
                            "type": "string",
                            "description": "Brief summary of the issue"
                        },
                        "description": {
                            "type": "string",
                            "description": "Detailed description of the issue"
                        },
                        "status": {
                            "type": "string",
                            "enum": ["open", "in_progress", "pending_customer", "resolved", "closed", "cancelled"],
                            "description": "Current ticket status"
                        },
                        "priority": {
                            "type": "string",
                            "enum": ["low", "medium", "high", "urgent", "critical"],
                            "description": "Ticket priority level"
                        },
                        "category": {
                            "type": "string",
                            "enum": ["technical", "billing", "feature_request", "bug_report", "other"],
                            "description": "Ticket category"
                        },
                        "customer_id": {
                            "type": "string",
                            "format": "uuid",
                            "description": "ID of the customer who created the ticket"
                        },
                        "assigned_to": {
                            "type": "string",
                            "format": "uuid",
                            "description": "ID of the assigned support agent"
                        },
                        "created_by": {
                            "type": "string",
                            "format": "uuid",
                            "description": "ID of the user who created the ticket"
                        },
                        "created_at": {
                            "type": "integer",
                            "format": "int64",
                            "description": "Unix timestamp when ticket was created"
                        },
                        "updated_at": {
                            "type": "integer",
                            "format": "int64",
                            "description": "Unix timestamp when ticket was last updated"
                        },
                        "due_date": {
                            "type": "integer",
                            "format": "int64",
                            "description": "Unix timestamp when ticket is due"
                        },
                        "resolved_at": {
                            "type": "integer",
                            "format": "int64",
                            "description": "Unix timestamp when ticket was resolved"
                        },
                        "sla_status": {
                            "type": "string",
                            "enum": ["on_track", "at_risk", "breached"],
                            "description": "SLA compliance status"
                        },
                        "tags": {
                            "type": "array",
                            "items": { "type": "string" },
                            "description": "Tags for categorization and search"
                        },
                        "custom_fields": {
                            "type": "object",
                            "additionalProperties": true,
                            "description": "Custom fields for tenant-specific data"
                        }
                    }
                },
                "TicketDetails": {
                    "allOf": [
                        { "$ref": "#/components/schemas/Ticket" },
                        {
                            "type": "object",
                            "properties": {
                                "comments": {
                                    "type": "array",
                                    "items": { "$ref": "#/components/schemas/Comment" },
                                    "description": "All comments on this ticket"
                                },
                                "attachments": {
                                    "type": "array",
                                    "items": { "$ref": "#/components/schemas/Attachment" },
                                    "description": "Files attached to this ticket"
                                },
                                "sla_info": {
                                    "$ref": "#/components/schemas/SLAInfo",
                                    "description": "SLA tracking information"
                                },
                                "activity_log": {
                                    "type": "array",
                                    "items": { "$ref": "#/components/schemas/TicketActivity" },
                                    "description": "Activity history for this ticket"
                                }
                            }
                        }
                    ]
                },
                "CreateTicketRequest": {
                    "type": "object",
                    "required": ["title", "description", "customer_id"],
                    "properties": {
                        "title": {
                            "type": "string",
                            "minLength": 5,
                            "maxLength": 200,
                            "description": "Brief summary of the issue"
                        },
                        "description": {
                            "type": "string",
                            "minLength": 10,
                            "description": "Detailed description of the issue"
                        },
                        "priority": {
                            "type": "string",
                            "enum": ["low", "medium", "high", "urgent", "critical"],
                            "default": "medium",
                            "description": "Ticket priority level"
                        },
                        "category": {
                            "type": "string",
                            "enum": ["technical", "billing", "feature_request", "bug_report", "other"],
                            "default": "technical",
                            "description": "Ticket category"
                        },
                        "customer_id": {
                            "type": "string",
                            "format": "uuid",
                            "description": "ID of the customer creating the ticket"
                        },
                        "assigned_to": {
                            "type": "string",
                            "format": "uuid",
                            "description": "ID of the support agent to assign (optional)"
                        },
                        "tags": {
                            "type": "array",
                            "items": { "type": "string" },
                            "description": "Tags for categorization and search"
                        },
                        "custom_fields": {
                            "type": "object",
                            "additionalProperties": true,
                            "description": "Custom fields for tenant-specific data"
                        },
                        "attachments": {
                            "type": "array",
                            "items": { "$ref": "#/components/schemas/Attachment" },
                            "description": "Initial attachments"
                        }
                    }
                },
                "UpdateTicketRequest": {
                    "type": "object",
                    "properties": {
                        "title": {
                            "type": "string",
                            "minLength": 5,
                            "maxLength": 200,
                            "description": "Updated ticket title"
                        },
                        "description": {
                            "type": "string",
                            "minLength": 10,
                            "description": "Updated ticket description"
                        },
                        "priority": {
                            "type": "string",
                            "enum": ["low", "medium", "high", "urgent", "critical"],
                            "description": "Updated ticket priority"
                        },
                        "category": {
                            "type": "string",
                            "enum": ["technical", "billing", "feature_request", "bug_report", "other"],
                            "description": "Updated ticket category"
                        },
                        "assigned_to": {
                            "type": "string",
                            "format": "uuid",
                            "description": "Updated assignment"
                        },
                        "tags": {
                            "type": "array",
                            "items": { "type": "string" },
                            "description": "Updated tags"
                        },
                        "custom_fields": {
                            "type": "object",
                            "additionalProperties": true,
                            "description": "Updated custom fields"
                        }
                    }
                },
                "UpdateTicketStatusRequest": {
                    "type": "object",
                    "required": ["status"],
                    "properties": {
                        "status": {
                            "type": "string",
                            "enum": ["open", "in_progress", "pending_customer", "resolved", "closed", "cancelled"],
                            "description": "New ticket status"
                        },
                        "comment": {
                            "type": "string",
                            "description": "Comment explaining the status change"
                        },
                        "notify_customer": {
                            "type": "boolean",
                            "default": true,
                            "description": "Whether to notify the customer of this change"
                        },
                        "resolution": {
                            "type": "string",
                            "description": "Resolution details (required when marking as resolved)"
                        }
                    }
                },
                "AssignTicketRequest": {
                    "type": "object",
                    "required": ["assigned_to"],
                    "properties": {
                        "assigned_to": {
                            "type": "string",
                            "format": "uuid",
                            "description": "ID of the user to assign the ticket to"
                        },
                        "comment": {
                            "type": "string",
                            "description": "Comment explaining the assignment"
                        },
                        "notify_assignee": {
                            "type": "boolean",
                            "default": true,
                            "description": "Whether to notify the assigned user"
                        }
                    }
                },
                "Comment": {
                    "type": "object",
                    "properties": {
                        "id": {
                            "type": "string",
                            "format": "uuid",
                            "description": "Unique comment identifier"
                        },
                        "ticket_id": {
                            "type": "string",
                            "format": "uuid",
                            "description": "ID of the ticket this comment belongs to"
                        },
                        "content": {
                            "type": "string",
                            "description": "Comment content"
                        },
                        "author_id": {
                            "type": "string",
                            "format": "uuid",
                            "description": "ID of the comment author"
                        },
                        "author_name": {
                            "type": "string",
                            "description": "Name of the comment author"
                        },
                        "is_internal": {
                            "type": "boolean",
                            "description": "Whether this comment is internal-only"
                        },
                        "created_at": {
                            "type": "integer",
                            "format": "int64",
                            "description": "Unix timestamp when comment was created"
                        },
                        "updated_at": {
                            "type": "integer",
                            "format": "int64",
                            "description": "Unix timestamp when comment was last updated"
                        },
                        "attachments": {
                            "type": "array",
                            "items": { "$ref": "#/components/schemas/Attachment" },
                            "description": "Files attached to this comment"
                        }
                    }
                },
                "CreateCommentRequest": {
                    "type": "object",
                    "required": ["content"],
                    "properties": {
                        "content": {
                            "type": "string",
                            "minLength": 1,
                            "description": "Comment content"
                        },
                        "is_internal": {
                            "type": "boolean",
                            "default": false,
                            "description": "Whether this comment should be internal-only"
                        },
                        "attachments": {
                            "type": "array",
                            "items": { "$ref": "#/components/schemas/Attachment" },
                            "description": "Files to attach to this comment"
                        }
                    }
                },
                "Attachment": {
                    "type": "object",
                    "properties": {
                        "id": {
                            "type": "string",
                            "format": "uuid",
                            "description": "Unique attachment identifier"
                        },
                        "name": {
                            "type": "string",
                            "description": "Original filename"
                        },
                        "url": {
                            "type": "string",
                            "format": "uri",
                            "description": "URL to download the attachment"
                        },
                        "size": {
                            "type": "integer",
                            "description": "File size in bytes"
                        },
                        "content_type": {
                            "type": "string",
                            "description": "MIME content type"
                        },
                        "uploaded_by": {
                            "type": "string",
                            "format": "uuid",
                            "description": "ID of user who uploaded the file"
                        },
                        "uploaded_at": {
                            "type": "integer",
                            "format": "int64",
                            "description": "Unix timestamp when file was uploaded"
                        }
                    }
                },
                "SLAInfo": {
                    "type": "object",
                    "properties": {
                        "policy_id": {
                            "type": "string",
                            "format": "uuid",
                            "description": "ID of the SLA policy applied"
                        },
                        "policy_name": {
                            "type": "string",
                            "description": "Name of the SLA policy"
                        },
                        "response_time_target": {
                            "type": "integer",
                            "description": "Target response time in minutes"
                        },
                        "resolution_time_target": {
                            "type": "integer",
                            "description": "Target resolution time in minutes"
                        },
                        "response_time_actual": {
                            "type": "integer",
                            "description": "Actual response time in minutes"
                        },
                        "resolution_time_actual": {
                            "type": "integer",
                            "description": "Actual resolution time in minutes"
                        },
                        "first_response_at": {
                            "type": "integer",
                            "format": "int64",
                            "description": "Unix timestamp of first response"
                        },
                        "breached": {
                            "type": "boolean",
                            "description": "Whether SLA was breached"
                        },
                        "breach_reason": {
                            "type": "string",
                            "description": "Reason for SLA breach (if applicable)"
                        }
                    }
                },
                "TicketActivity": {
                    "type": "object",
                    "properties": {
                        "id": {
                            "type": "string",
                            "format": "uuid",
                            "description": "Unique activity identifier"
                        },
                        "ticket_id": {
                            "type": "string",
                            "format": "uuid",
                            "description": "ID of the ticket"
                        },
                        "action": {
                            "type": "string",
                            "enum": ["created", "updated", "assigned", "status_changed", "commented", "resolved", "closed"],
                            "description": "Type of activity"
                        },
                        "description": {
                            "type": "string",
                            "description": "Human-readable description of the activity"
                        },
                        "user_id": {
                            "type": "string",
                            "format": "uuid",
                            "description": "ID of the user who performed the action"
                        },
                        "user_name": {
                            "type": "string",
                            "description": "Name of the user who performed the action"
                        },
                        "old_value": {
                            "type": "string",
                            "description": "Previous value (for change activities)"
                        },
                        "new_value": {
                            "type": "string",
                            "description": "New value (for change activities)"
                        },
                        "created_at": {
                            "type": "integer",
                            "format": "int64",
                            "description": "Unix timestamp when activity occurred"
                        }
                    }
                },
                "TicketResponse": {
                    "allOf": [
                        { "$ref": "#/components/schemas/ApiResponse" },
                        {
                            "type": "object",
                            "properties": {
                                "data": { "$ref": "#/components/schemas/Ticket" }
                            }
                        }
                    ]
                },
                "TicketDetailsResponse": {
                    "allOf": [
                        { "$ref": "#/components/schemas/ApiResponse" },
                        {
                            "type": "object",
                            "properties": {
                                "data": { "$ref": "#/components/schemas/TicketDetails" }
                            }
                        }
                    ]
                },
                "PaginatedTicketsResponse": {
                    "allOf": [
                        { "$ref": "#/components/schemas/ApiResponse" },
                        {
                            "type": "object",
                            "properties": {
                                "data": {
                                    "type": "object",
                                    "properties": {
                                        "items": {
                                            "type": "array",
                                            "items": { "$ref": "#/components/schemas/Ticket" }
                                        },
                                        "pagination": { "$ref": "#/components/schemas/PaginationInfo" }
                                    }
                                }
                            }
                        }
                    ]
                },
                "PaginatedCommentsResponse": {
                    "allOf": [
                        { "$ref": "#/components/schemas/ApiResponse" },
                        {
                            "type": "object",
                            "properties": {
                                "data": {
                                    "type": "object",
                                    "properties": {
                                        "items": {
                                            "type": "array",
                                            "items": { "$ref": "#/components/schemas/Comment" }
                                        },
                                        "pagination": { "$ref": "#/components/schemas/PaginationInfo" }
                                    }
                                }
                            }
                        }
                    ]
                },
                "CommentResponse": {
                    "allOf": [
                        { "$ref": "#/components/schemas/ApiResponse" },
                        {
                            "type": "object",
                            "properties": {
                                "data": { "$ref": "#/components/schemas/Comment" }
                            }
                        }
                    ]
                },
                // Knowledge Base schemas
                "KnowledgeArticle": {
                    "type": "object",
                    "properties": {
                        "id": { "type": "string", "format": "uuid" },
                        "title": { "type": "string" },
                        "content": { "type": "string" },
                        "category": {
                            "type": "string",
                            "enum": ["getting-started", "troubleshooting", "best-practices", "api-reference", "tutorials"]
                        },
                        "status": {
                            "type": "string",
                            "enum": ["draft", "published", "archived"]
                        },
                        "author_id": { "type": "string", "format": "uuid" },
                        "tags": {
                            "type": "array",
                            "items": { "type": "string" }
                        },
                        "view_count": { "type": "integer" },
                        "helpful_votes": { "type": "integer" },
                        "created_at": { "type": "integer" },
                        "updated_at": { "type": "integer" }
                    }
                },
                "CreateKnowledgeArticleRequest": {
                    "type": "object",
                    "required": ["title", "content", "category"],
                    "properties": {
                        "title": { "type": "string" },
                        "content": { "type": "string" },
                        "category": {
                            "type": "string",
                            "enum": ["getting-started", "troubleshooting", "best-practices", "api-reference", "tutorials"]
                        },
                        "tags": {
                            "type": "array",
                            "items": { "type": "string" }
                        }
                    }
                },
                "UpdateKnowledgeArticleRequest": {
                    "type": "object",
                    "properties": {
                        "title": { "type": "string" },
                        "content": { "type": "string" },
                        "category": {
                            "type": "string",
                            "enum": ["getting-started", "troubleshooting", "best-practices", "api-reference", "tutorials"]
                        },
                        "status": {
                            "type": "string",
                            "enum": ["draft", "published", "archived"]
                        },
                        "tags": {
                            "type": "array",
                            "items": { "type": "string" }
                        }
                    }
                },
                "KnowledgeArticleResponse": {
                    "allOf": [
                        { "$ref": "#/components/schemas/ApiResponse" },
                        {
                            "type": "object",
                            "properties": {
                                "data": { "$ref": "#/components/schemas/KnowledgeArticle" }
                            }
                        }
                    ]
                },
                "PaginatedKnowledgeArticlesResponse": {
                    "allOf": [
                        { "$ref": "#/components/schemas/ApiResponse" },
                        {
                            "type": "object",
                            "properties": {
                                "data": {
                                    "type": "object",
                                    "properties": {
                                        "items": {
                                            "type": "array",
                                            "items": { "$ref": "#/components/schemas/KnowledgeArticle" }
                                        },
                                        "pagination": { "$ref": "#/components/schemas/PaginationInfo" }
                                    }
                                }
                            }
                        }
                    ]
                },
                "ArticleFeedbackRequest": {
                    "type": "object",
                    "required": ["rating"],
                    "properties": {
                        "rating": {
                            "type": "integer",
                            "minimum": 1,
                            "maximum": 5
                        },
                        "comment": { "type": "string" }
                    }
                },
                // SLA Management schemas
                "SLAPolicy": {
                    "type": "object",
                    "properties": {
                        "id": { "type": "string", "format": "uuid" },
                        "name": { "type": "string" },
                        "description": { "type": "string" },
                        "response_time_minutes": { "type": "integer" },
                        "resolution_time_minutes": { "type": "integer" },
                        "business_hours_only": { "type": "boolean" },
                        "priority_levels": {
                            "type": "array",
                            "items": { "type": "string" }
                        },
                        "ticket_categories": {
                            "type": "array",
                            "items": { "type": "string" }
                        },
                        "status": {
                            "type": "string",
                            "enum": ["active", "inactive", "draft"]
                        },
                        "created_at": { "type": "integer" },
                        "updated_at": { "type": "integer" }
                    }
                },
                "CreateSLAPolicyRequest": {
                    "type": "object",
                    "required": ["name", "description", "response_time_minutes", "resolution_time_minutes"],
                    "properties": {
                        "name": { "type": "string" },
                        "description": { "type": "string" },
                        "response_time_minutes": { "type": "integer" },
                        "resolution_time_minutes": { "type": "integer" },
                        "business_hours_only": { "type": "boolean" },
                        "priority_levels": {
                            "type": "array",
                            "items": { "type": "string" }
                        },
                        "ticket_categories": {
                            "type": "array",
                            "items": { "type": "string" }
                        }
                    }
                },
                "UpdateSLAPolicyRequest": {
                    "type": "object",
                    "properties": {
                        "name": { "type": "string" },
                        "description": { "type": "string" },
                        "response_time_minutes": { "type": "integer" },
                        "resolution_time_minutes": { "type": "integer" },
                        "business_hours_only": { "type": "boolean" },
                        "priority_levels": {
                            "type": "array",
                            "items": { "type": "string" }
                        },
                        "ticket_categories": {
                            "type": "array",
                            "items": { "type": "string" }
                        },
                        "status": {
                            "type": "string",
                            "enum": ["active", "inactive", "draft"]
                        }
                    }
                },
                "SLAPolicyResponse": {
                    "allOf": [
                        { "$ref": "#/components/schemas/ApiResponse" },
                        {
                            "type": "object",
                            "properties": {
                                "data": { "$ref": "#/components/schemas/SLAPolicy" }
                            }
                        }
                    ]
                },
                "PaginatedSLAPoliciesResponse": {
                    "allOf": [
                        { "$ref": "#/components/schemas/ApiResponse" },
                        {
                            "type": "object",
                            "properties": {
                                "data": {
                                    "type": "object",
                                    "properties": {
                                        "items": {
                                            "type": "array",
                                            "items": { "$ref": "#/components/schemas/SLAPolicy" }
                                        },
                                        "pagination": { "$ref": "#/components/schemas/PaginationInfo" }
                                    }
                                }
                            }
                        }
                    ]
                },
                "SLAAgreement": {
                    "type": "object",
                    "properties": {
                        "id": { "type": "string", "format": "uuid" },
                        "tenant_id": { "type": "string", "format": "uuid" },
                        "policy_id": { "type": "string", "format": "uuid" },
                        "policy_name": { "type": "string" },
                        "customer_id": { "type": "string", "format": "uuid" },
                        "start_date": { "type": "integer" },
                        "end_date": { "type": "integer" },
                        "custom_terms": { "type": "string" },
                        "status": {
                            "type": "string",
                            "enum": ["active", "expired", "terminated"]
                        },
                        "created_at": { "type": "integer" },
                        "updated_at": { "type": "integer" }
                    }
                },
                "CreateSLAAgreementRequest": {
                    "type": "object",
                    "required": ["tenant_id", "policy_id", "customer_id"],
                    "properties": {
                        "tenant_id": { "type": "string", "format": "uuid" },
                        "policy_id": { "type": "string", "format": "uuid" },
                        "customer_id": { "type": "string", "format": "uuid" },
                        "start_date": { "type": "integer" },
                        "end_date": { "type": "integer" },
                        "custom_terms": { "type": "string" }
                    }
                },
                "UpdateSLAAgreementRequest": {
                    "type": "object",
                    "properties": {
                        "start_date": { "type": "integer" },
                        "end_date": { "type": "integer" },
                        "custom_terms": { "type": "string" },
                        "status": {
                            "type": "string",
                            "enum": ["active", "expired", "terminated"]
                        }
                    }
                },
                "SLAAgreementResponse": {
                    "allOf": [
                        { "$ref": "#/components/schemas/ApiResponse" },
                        {
                            "type": "object",
                            "properties": {
                                "data": { "$ref": "#/components/schemas/SLAAgreement" }
                            }
                        }
                    ]
                },
                "PaginatedSLAAgreementsResponse": {
                    "allOf": [
                        { "$ref": "#/components/schemas/ApiResponse" },
                        {
                            "type": "object",
                            "properties": {
                                "data": {
                                    "type": "object",
                                    "properties": {
                                        "items": {
                                            "type": "array",
                                            "items": { "$ref": "#/components/schemas/SLAAgreement" }
                                        },
                                        "pagination": { "$ref": "#/components/schemas/PaginationInfo" }
                                    }
                                }
                            }
                        }
                    ]
                },
                "SLAMetrics": {
                    "type": "object",
                    "properties": {
                        "total_tickets": { "type": "integer" },
                        "resolved_within_sla": { "type": "integer" },
                        "breached_sla": { "type": "integer" },
                        "average_response_time": { "type": "number" },
                        "average_resolution_time": { "type": "number" },
                        "sla_compliance_rate": { "type": "number" },
                        "period_start": { "type": "integer" },
                        "period_end": { "type": "integer" }
                    }
                },
                "SLAMetricsResponse": {
                    "allOf": [
                        { "$ref": "#/components/schemas/ApiResponse" },
                        {
                            "type": "object",
                            "properties": {
                                "data": { "$ref": "#/components/schemas/SLAMetrics" }
                            }
                        }
                    ]
                },
                "SLABreach": {
                    "type": "object",
                    "properties": {
                        "id": { "type": "string", "format": "uuid" },
                        "ticket_id": { "type": "string", "format": "uuid" },
                        "policy_id": { "type": "string", "format": "uuid" },
                        "breach_type": {
                            "type": "string",
                            "enum": ["response_time", "resolution_time"]
                        },
                        "severity": {
                            "type": "string",
                            "enum": ["minor", "major", "critical"]
                        },
                        "breach_time": { "type": "integer" },
                        "reason": { "type": "string" },
                        "resolved": { "type": "boolean" },
                        "created_at": { "type": "integer" }
                    }
                },
                "PaginatedSLABreachesResponse": {
                    "allOf": [
                        { "$ref": "#/components/schemas/ApiResponse" },
                        {
                            "type": "object",
                            "properties": {
                                "data": {
                                    "type": "object",
                                    "properties": {
                                        "items": {
                                            "type": "array",
                                            "items": { "$ref": "#/components/schemas/SLABreach" }
                                        },
                                        "pagination": { "$ref": "#/components/schemas/PaginationInfo" }
                                    }
                                }
                            }
                        }
                    ]
                },
                // Tenant Management schemas
                "Tenant": {
                    "type": "object",
                    "properties": {
                        "id": { "type": "string", "format": "uuid" },
                        "name": { "type": "string" },
                        "domain": { "type": "string" },
                        "status": {
                            "type": "string",
                            "enum": ["active", "inactive", "suspended"]
                        },
                        "plan": {
                            "type": "string",
                            "enum": ["basic", "professional", "enterprise"]
                        },
                        "max_users": { "type": "integer" },
                        "current_users": { "type": "integer" },
                        "settings": {
                            "type": "object",
                            "additionalProperties": true
                        },
                        "created_at": { "type": "integer" },
                        "updated_at": { "type": "integer" }
                    }
                },
                "CreateTenantRequest": {
                    "type": "object",
                    "required": ["name", "domain", "plan"],
                    "properties": {
                        "name": { "type": "string" },
                        "domain": { "type": "string" },
                        "plan": {
                            "type": "string",
                            "enum": ["basic", "professional", "enterprise"]
                        },
                        "max_users": { "type": "integer" },
                        "settings": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                },
                "UpdateTenantRequest": {
                    "type": "object",
                    "properties": {
                        "name": { "type": "string" },
                        "domain": { "type": "string" },
                        "status": {
                            "type": "string",
                            "enum": ["active", "inactive", "suspended"]
                        },
                        "plan": {
                            "type": "string",
                            "enum": ["basic", "professional", "enterprise"]
                        },
                        "max_users": { "type": "integer" }
                    }
                },
                "TenantResponse": {
                    "allOf": [
                        { "$ref": "#/components/schemas/ApiResponse" },
                        {
                            "type": "object",
                            "properties": {
                                "data": { "$ref": "#/components/schemas/Tenant" }
                            }
                        }
                    ]
                },
                "PaginatedTenantsResponse": {
                    "allOf": [
                        { "$ref": "#/components/schemas/ApiResponse" },
                        {
                            "type": "object",
                            "properties": {
                                "data": {
                                    "type": "object",
                                    "properties": {
                                        "items": {
                                            "type": "array",
                                            "items": { "$ref": "#/components/schemas/Tenant" }
                                        },
                                        "pagination": { "$ref": "#/components/schemas/PaginationInfo" }
                                    }
                                }
                            }
                        }
                    ]
                },
                "AddTenantUserRequest": {
                    "type": "object",
                    "required": ["user_id", "role"],
                    "properties": {
                        "user_id": { "type": "string", "format": "uuid" },
                        "role": {
                            "type": "string",
                            "enum": ["admin", "member", "viewer"]
                        }
                    }
                },
                "TenantSettings": {
                    "type": "object",
                    "properties": {
                        "timezone": { "type": "string" },
                        "locale": { "type": "string" },
                        "email_notifications": { "type": "boolean" },
                        "two_factor_auth": { "type": "boolean" },
                        "session_timeout_minutes": { "type": "integer" },
                        "custom_settings": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                },
                "UpdateTenantSettingsRequest": {
                    "type": "object",
                    "properties": {
                        "timezone": { "type": "string" },
                        "locale": { "type": "string" },
                        "email_notifications": { "type": "boolean" },
                        "two_factor_auth": { "type": "boolean" },
                        "session_timeout_minutes": { "type": "integer" },
                        "custom_settings": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                },
                "TenantSettingsResponse": {
                    "allOf": [
                        { "$ref": "#/components/schemas/ApiResponse" },
                        {
                            "type": "object",
                            "properties": {
                                "data": { "$ref": "#/components/schemas/TenantSettings" }
                            }
                        }
                    ]
                },
                "UpdateTenantStatusRequest": {
                    "type": "object",
                    "required": ["status"],
                    "properties": {
                        "status": {
                            "type": "string",
                            "enum": ["active", "inactive", "suspended"]
                        },
                        "reason": { "type": "string" }
                    }
                },
                // Role & Permission schemas
                "Role": {
                    "type": "object",
                    "properties": {
                        "id": { "type": "string", "format": "uuid" },
                        "name": { "type": "string" },
                        "description": { "type": "string" },
                        "is_system_role": { "type": "boolean" },
                        "tenant_id": { "type": "string", "format": "uuid" },
                        "permissions": {
                            "type": "array",
                            "items": { "$ref": "#/components/schemas/Permission" }
                        },
                        "user_count": { "type": "integer" },
                        "created_at": { "type": "integer" },
                        "updated_at": { "type": "integer" }
                    }
                },
                "Permission": {
                    "type": "object",
                    "properties": {
                        "id": { "type": "string", "format": "uuid" },
                        "name": { "type": "string" },
                        "resource": { "type": "string" },
                        "action": { "type": "string" },
                        "description": { "type": "string" }
                    }
                },
                "CreateRoleRequest": {
                    "type": "object",
                    "required": ["name", "description"],
                    "properties": {
                        "name": { "type": "string" },
                        "description": { "type": "string" },
                        "permission_ids": {
                            "type": "array",
                            "items": { "type": "string", "format": "uuid" }
                        }
                    }
                },
                "UpdateRoleRequest": {
                    "type": "object",
                    "properties": {
                        "name": { "type": "string" },
                        "description": { "type": "string" },
                        "permission_ids": {
                            "type": "array",
                            "items": { "type": "string", "format": "uuid" }
                        }
                    }
                },
                "RoleResponse": {
                    "allOf": [
                        { "$ref": "#/components/schemas/ApiResponse" },
                        {
                            "type": "object",
                            "properties": {
                                "data": { "$ref": "#/components/schemas/Role" }
                            }
                        }
                    ]
                },
                "PaginatedRolesResponse": {
                    "allOf": [
                        { "$ref": "#/components/schemas/ApiResponse" },
                        {
                            "type": "object",
                            "properties": {
                                "data": {
                                    "type": "object",
                                    "properties": {
                                        "items": {
                                            "type": "array",
                                            "items": { "$ref": "#/components/schemas/Role" }
                                        },
                                        "pagination": { "$ref": "#/components/schemas/PaginationInfo" }
                                    }
                                }
                            }
                        }
                    ]
                },
                "RolePermissionsResponse": {
                    "allOf": [
                        { "$ref": "#/components/schemas/ApiResponse" },
                        {
                            "type": "object",
                            "properties": {
                                "data": {
                                    "type": "object",
                                    "properties": {
                                        "permissions": {
                                            "type": "array",
                                            "items": { "$ref": "#/components/schemas/Permission" }
                                        }
                                    }
                                }
                            }
                        }
                    ]
                },
                "AddRolePermissionRequest": {
                    "type": "object",
                    "required": ["permission_id"],
                    "properties": {
                        "permission_id": { "type": "string", "format": "uuid" }
                    }
                }
            }
        })
    }

    /// Export OpenAPI specification as JSON string
    pub fn export_json(&self) -> Result<String, serde_json::Error> {
        let spec = self.generate_spec();
        serde_json::to_string_pretty(&spec)
    }

    /// Export OpenAPI specification as YAML string
    pub fn export_yaml(&self) -> Result<String, serde_yaml::Error> {
        let spec = self.generate_spec();
        serde_yaml::to_string(&spec)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_openapi_generation() {
        let config = GatewayConfig::default();
        let generator = OpenApiGenerator::new(config);
        let spec = generator.generate_spec();

        assert_eq!(spec.openapi, "3.0.3");
        assert_eq!(spec.info.title, "SmartTicket API");
        assert!(!spec.servers.is_empty());
        assert!(!spec.paths.is_empty());
        assert!(!spec.components.is_null());
    }

    #[test]
    fn test_json_export() {
        let config = GatewayConfig::default();
        let generator = OpenApiGenerator::new(config);
        let json_str = generator.export_json().unwrap();

        assert!(json_str.contains("\"openapi\": \"3.0.3\""));
        assert!(json_str.contains("SmartTicket API"));
    }

    #[test]
    fn test_yaml_export() {
        let config = GatewayConfig::default();
        let generator = OpenApiGenerator::new(config);
        let yaml_str = generator.export_yaml().unwrap();

        assert!(yaml_str.contains("openapi: 3.0.3"));
        assert!(yaml_str.contains("SmartTicket API"));
    }
}