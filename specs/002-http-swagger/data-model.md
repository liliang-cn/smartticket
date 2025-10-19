# Data Model: HTTP REST API Gateway

**Date**: 2025-01-17
**Purpose**: Define data structures and entities for the HTTP-to-gRPC gateway implementation

## Core Gateway Entities

### 1. Gateway Configuration

```rust
pub struct GatewayConfig {
    pub http_port: u16,
    pub grpc_endpoint: String,
    pub cors_origins: Vec<String>,
    pub max_request_size: usize,
    pub timeout: Duration,
    pub rate_limit: RateLimitConfig,
    pub auth: AuthConfig,
    openapi: OpenApiConfig,
}

pub struct RateLimitConfig {
    pub requests_per_minute: u32,
    pub burst_size: u32,
}

pub struct AuthConfig {
    pub jwt_secret: String,
    pub token_expiry: Duration,
    pub refresh_expiry: Duration,
}

pub struct OpenApiConfig {
    pub auto_refresh: bool,
    pub include_examples: bool,
    pub servers: Vec<ServerInfo>,
}
```

### 2. Request Context

```rust
pub struct RequestContext {
    pub request_id: String,
    pub tenant_id: String,
    pub user_id: String,
    pub user_role: UserRole,
    pub timestamp: i64,
    pub metadata: HashMap<String, String>,
}

pub struct TenantContext {
    pub tenant_id: String,
    pub domain: String,
    pub subscription_tier: SubscriptionTier,
    pub max_users: u32,
    pub is_active: bool,
    pub settings: TenantSettings,
}
```

### 3. HTTP-to-gRPC Translation

```rust
pub struct RequestTranslation {
    pub http_method: HttpMethod,
    pub http_path: String,
    pub grpc_service: String,
    pub grpc_method: String,
    pub path_params: HashMap<String, String>,
    pub query_params: HashMap<String, String>,
    pub headers: HashMap<String, String>,
    pub body: Option<serde_json::Value>,
}

pub struct ResponseTranslation {
    pub grpc_status: GrpcStatus,
    pub http_status: StatusCode,
    pub headers: HashMap<String, String>,
    pub body: Option<serde_json::Value>,
    pub error: Option<ApiError>,
}
```

### 4. Error Handling

```rust
#[derive(Serialize, Deserialize)]
pub struct ApiResponse<T> {
    pub success: bool,
    pub message: String,
    pub data: Option<T>,
    pub errors: Vec<ApiError>,
    pub request_id: String,
    pub timestamp: i64,
}

#[derive(Serialize, Deserialize)]
pub struct ApiError {
    pub code: String,
    pub message: String,
    pub details: Option<serde_json::Value>,
    pub field: Option<String>,
}

pub enum GatewayError {
    AuthenticationError(String),
    AuthorizationError(String),
    ValidationError(String),
    TranslationError(String),
    ServiceError(String),
    RateLimitError(String),
}
```

## Service-Specific Data Models

### 1. Authentication Service

```rust
#[derive(Serialize, Deserialize)]
pub struct LoginRequest {
    pub email: String,
    pub password: String,
    pub tenant_domain: String,
    pub remember_me: Option<bool>,
}

#[derive(Serialize, Deserialize)]
pub struct LoginResponse {
    pub success: bool,
    pub access_token: String,
    pub refresh_token: String,
    pub expires_at: i64,
    pub user: UserInfo,
}

#[derive(Serialize, Deserialize)]
pub struct RefreshTokenRequest {
    pub refresh_token: String,
}

#[derive(Serialize, Deserialize)]
pub struct RefreshTokenResponse {
    pub success: bool,
    pub access_token: String,
    pub refresh_token: String,
    pub expires_at: i64,
}
```

### 2. User Service

```rust
#[derive(Serialize, Deserialize)]
pub struct CreateUserRequest {
    pub email: String,
    pub username: String,
    pub full_name: String,
    pub password: String,
    pub role: UserRole,
    pub phone: Option<String>,
    pub timezone: Option<String>,
    pub language: Option<String>,
    pub preferences: Option<HashMap<String, serde_json::Value>>,
}

#[derive(Serialize, Deserialize)]
pub struct UpdateUserRequest {
    pub email: Option<String>,
    pub username: Option<String>,
    pub full_name: Option<String>,
    pub role: Option<UserRole>,
    pub phone: Option<String>,
    pub timezone: Option<String>,
    pub language: Option<String>,
    pub preferences: Option<HashMap<String, serde_json::Value>>,
}

#[derive(Serialize, Deserialize)]
pub struct UserResponse {
    pub id: String,
    pub email: String,
    pub username: String,
    pub full_name: String,
    pub role: UserRole,
    pub is_active: bool,
    pub last_login_at: Option<i64>,
    pub created_at: i64,
    pub updated_at: i64,
}

#[derive(Serialize, Deserialize)]
pub enum UserRole {
    SUPER_ADMIN,
    TENANT_ADMIN,
    SUPPORT_ENGINEER,
    CUSTOMER_USER,
    SALES,
}

#[derive(Serialize, Deserialize)]
pub struct ChangePasswordRequest {
    pub current_password: String,
    pub new_password: String,
}
```

### 3. Tenant Service

```rust
#[derive(Serialize, Deserialize)]
pub struct CreateTenantRequest {
    pub name: String,
    pub domain: String,
    pub subscription_tier: SubscriptionTier,
    pub max_users: u32,
    pub data_residency_region: String,
    pub contact_email: String,
    pub billing_address: Option<String>,
    pub phone: Option<String>,
    pub settings: Option<TenantSettings>,
}

#[derive(Serialize, Deserialize)]
pub struct TenantResponse {
    pub id: String,
    pub name: String,
    pub domain: String,
    pub subscription_tier: SubscriptionTier,
    pub max_users: u32,
    pub current_user_count: u32,
    pub data_residency_region: String,
    pub is_active: bool,
    pub created_at: i64,
    pub updated_at: i64,
    pub subscription_expires_at: Option<i64>,
    pub contact_email: String,
}

#[derive(Serialize, Deserialize)]
pub enum SubscriptionTier {
    STANDARD,
    PREMIUM,
    ENTERPRISE,
}

#[derive(Serialize, Deserialize)]
pub struct TenantSettings {
    pub default_timezone: Option<String>,
    pub default_language: Option<String>,
    pub enable_multi_language: Option<bool>,
    pub allow_user_registration: Option<bool>,
    pub branding_logo_url: Option<String>,
    pub branding_color: Option<String>,
    pub custom_fields: Option<HashMap<String, serde_json::Value>>,
    pub security: Option<SecuritySettings>,
    pub notifications: Option<NotificationSettings>,
}
```

### 4. Ticket Service

```rust
#[derive(Serialize, Deserialize)]
pub struct CreateTicketRequest {
    pub title: String,
    pub description: String,
    pub priority: TicketPriority,
    pub severity: TicketSeverity,
    pub category_id: Option<String>,
    pub contact_id: String,
    pub assigned_to_id: Option<String>,
    pub due_date: Option<i64>,
    pub tags: Option<Vec<String>>,
    pub custom_fields: Option<HashMap<String, serde_json::Value>>,
}

#[derive(Serialize, Deserialize)]
pub struct TicketResponse {
    pub id: String,
    pub title: String,
    pub description: String,
    pub status: TicketStatus,
    pub priority: TicketPriority,
    pub severity: TicketSeverity,
    pub category_id: Option<String>,
    pub contact_id: String,
    pub assigned_to_id: Option<String>,
    pub created_by_id: String,
    pub created_at: i64,
    pub updated_at: i64,
    pub due_date: Option<i64>,
    pub resolved_at: Option<i64>,
    pub tags: Vec<String>,
    pub custom_fields: HashMap<String, serde_json::Value>,
}

#[derive(Serialize, Deserialize)]
pub enum TicketStatus {
    OPEN,
    IN_PROGRESS,
    PENDING_CUSTOMER,
    RESOLVED,
    CLOSED,
    REOPENED,
}

#[derive(Serialize, Deserialize)]
pub enum TicketPriority {
    LOW,
    NORMAL,
    HIGH,
    URGENT,
}

#[derive(Serialize, Deserialize)]
pub enum TicketSeverity {
    MINOR,
    MAJOR,
    CRITICAL,
    BLOCKER,
}

#[derive(Serialize, Deserialize)]
pub struct UpdateTicketStatusRequest {
    pub status: TicketStatus,
    pub comment: Option<String>,
    pub notify_customer: Option<bool>,
}
```

### 5. Knowledge Service

```rust
#[derive(Serialize, Deserialize)]
pub struct CreateArticleRequest {
    pub title: String,
    pub content: String,
    pub summary: Option<String>,
    pub category_id: Option<String>,
    pub tags: Option<Vec<String>>,
    pub language: Option<String>,
    pub visibility: KnowledgeVisibility,
    pub custom_fields: Option<HashMap<String, serde_json::Value>>,
}

#[derive(Serialize, Deserialize)]
pub struct ArticleResponse {
    pub id: String,
    pub title: String,
    pub content: String,
    pub summary: Option<String>,
    pub status: KnowledgeStatus,
    pub visibility: KnowledgeVisibility,
    pub category_id: Option<String>,
    pub author_id: String,
    pub language: Option<String>,
    pub tags: Vec<String>,
    pub view_count: u32,
    pub rating: Option<f32>,
    pub created_at: i64,
    pub updated_at: i64,
    pub published_at: Option<i64>,
    pub custom_fields: HashMap<String, serde_json::Value>,
}

#[derive(Serialize, Deserialize)]
pub enum KnowledgeStatus {
    DRAFT,
    PENDING_REVIEW,
    PUBLISHED,
    ARCHIVED,
}

#[derive(Serialize, Deserialize)]
pub enum KnowledgeVisibility {
    PUBLIC,
    INTERNAL,
    RESTRICTED,
    PRIVATE,
}

#[derive(Serialize, Deserialize)]
pub struct SearchArticlesRequest {
    pub query: String,
    pub category_id: Option<String>,
    pub tags: Option<Vec<String>>,
    pub language: Option<String>,
    pub status: Option<Vec<KnowledgeStatus>>,
    pub visibility: Option<Vec<KnowledgeVisibility>>,
    pub limit: Option<u32>,
    pub offset: Option<u32>,
}
```

### 6. SLA Service

```rust
#[derive(Serialize, Deserialize)]
pub struct CreateSlaPolicyRequest {
    pub name: String,
    pub description: String,
    pub priority: Option<TicketPriority>,
    pub severity: Option<TicketSeverity>,
    pub category_id: Option<String>,
    pub response_time_minutes: u32,
    pub resolution_time_minutes: u32,
    pub business_hours_only: bool,
    pub timezone: Option<String>,
    pub holiday_calendar_id: Option<String>,
    pub escalation_rules: Option<Vec<EscalationRule>>,
}

#[derive(Serialize, Deserialize)]
pub struct SlaPolicyResponse {
    pub id: String,
    pub name: String,
    pub description: String,
    pub priority: Option<TicketPriority>,
    pub severity: Option<TicketSeverity>,
    pub category_id: Option<String>,
    pub response_time_minutes: u32,
    pub resolution_time_minutes: u32,
    pub business_hours_only: bool,
    pub timezone: Option<String>,
    pub is_active: bool,
    pub created_at: i64,
    pub updated_at: i64,
}

#[derive(Serialize, Deserialize)]
pub struct EscalationRule {
    pub condition: String,
    pub delay_minutes: u32,
    pub escalate_to_role: UserRole,
    pub notify_users: Vec<String>,
}
```

### 7. Role and Permission Service

```rust
#[derive(Serialize, Deserialize)]
pub struct CreateRoleRequest {
    pub name: String,
    pub description: String,
    pub permissions: Vec<String>,
    pub is_system_role: bool,
}

#[derive(Serialize, Deserialize)]
pub struct RoleResponse {
    pub id: String,
    pub name: String,
    pub description: String,
    pub permissions: Vec<Permission>,
    pub is_system_role: bool,
    pub user_count: u32,
    pub created_at: i64,
    pub updated_at: i64,
}

#[derive(Serialize, Deserialize)]
pub struct Permission {
    pub id: String,
    pub resource: String,
    pub action: String,
    pub description: String,
}

#[derive(Serialize, Deserialize)]
pub struct UpdateRolePermissionsRequest {
    pub role_id: String,
    pub add_permissions: Vec<String>,
    pub remove_permissions: Vec<String>,
}
```

## Response Models

### Pagination

```rust
#[derive(Serialize, Deserialize)]
pub struct PaginatedResponse<T> {
    pub success: bool,
    pub message: String,
    pub data: Vec<T>,
    pub pagination: PaginationInfo,
    pub request_id: String,
    pub timestamp: i64,
}

#[derive(Serialize, Deserialize)]
pub struct PaginationInfo {
    pub total_count: u32,
    pub page_size: u32,
    pub current_page: u32,
    pub total_pages: u32,
    pub has_next: bool,
    pub has_prev: bool,
    pub next_page_token: Option<String>,
    pub prev_page_token: Option<String>,
}
```

### Search

```rust
#[derive(Serialize, Deserialize)]
pub struct SearchRequest {
    pub query: String,
    pub filters: Option<HashMap<String, serde_json::Value>>,
    pub sort_by: Option<String>,
    pub sort_order: Option<SortOrder>,
    pub limit: Option<u32>,
    pub offset: Option<u32>,
}

#[derive(Serialize, Deserialize)]
pub enum SortOrder {
    ASC,
    DESC,
}

#[derive(Serialize, Deserialize)]
pub struct SearchResponse<T> {
    pub success: bool,
    pub message: String,
    pub results: Vec<T>,
    pub total_count: u32,
    pub search_time_ms: u32,
    pub request_id: String,
    pub timestamp: i64,
}
```

## Configuration Models

### Server Configuration

```rust
#[derive(Serialize, Deserialize)]
pub struct ServerInfo {
    pub url: String,
    pub description: String,
}

#[derive(Serialize, Deserialize)]
pub struct OpenApiSpec {
    pub openapi: String,
    pub info: ApiInfo,
    pub servers: Vec<ServerInfo>,
    pub security: Vec<SecurityScheme>,
    pub paths: HashMap<String, PathItem>,
    pub components: Components,
}
```

### Security

```rust
#[derive(Serialize, Deserialize)]
pub struct SecuritySettings {
    pub require_2fa: Option<bool>,
    pub password_min_length: Option<u32>,
    pub require_password_change: Option<bool>,
    pub session_timeout_minutes: Option<u32>,
    pub ip_whitelist_enabled: Option<bool>,
    pub allowed_ip_ranges: Option<Vec<String>>,
}

#[derive(Serialize, Deserialize)]
pub struct NotificationSettings {
    pub email_notifications: Option<bool>,
    pub sms_notifications: Option<bool>,
    pub push_notifications: Option<bool>,
    pub default_from_email: Option<String>,
    pub default_from_name: Option<String>,
    pub notification_templates: Option<HashMap<String, serde_json::Value>>,
}
```

This data model provides the foundation for implementing the HTTP-to-gRPC gateway with comprehensive type safety and clear structure for all service interactions.