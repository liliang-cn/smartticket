package models

import (
	"time"

	"gorm.io/gorm"
)

// Base model with common fields
type BaseModel struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedBy *string        `gorm:"size:255" json:"created_by,omitempty"`
	UpdatedBy *string        `gorm:"size:255" json:"updated_by,omitempty"`
}

// Tenant represents a multi-tenant organization
type Tenant struct {
	BaseModel
	Name      string     `gorm:"size:255;not null" json:"name"`
	Slug      string     `gorm:"size:100;uniqueIndex;not null" json:"slug"`
	Domain    string     `gorm:"size:255" json:"domain"`
	Settings  string     `gorm:"type:text" json:"settings"` // JSON string
	Plan      string     `gorm:"size:50;default:'basic'" json:"plan"`
	MaxUsers  int        `gorm:"default:100" json:"max_users"`
	IsActive  bool       `gorm:"default:true" json:"is_active"`
	ExpiredAt *time.Time `json:"expired_at"`
	Users     []User     `gorm:"foreignKey:TenantID;references:ID" json:"users,omitempty"`
	Tickets   []Ticket   `gorm:"foreignKey:TenantID;references:ID" json:"tickets,omitempty"`
}

// User represents a user account
type User struct {
	BaseModel
	TenantID     uint       `gorm:"not null;index" json:"tenant_id"`
	Tenant       Tenant     `gorm:"foreignKey:TenantID;references:ID" json:"tenant,omitempty"`
	Email        string     `gorm:"size:255;not null;uniqueIndex:idx_tenant_email" json:"email"`
	Username     string     `gorm:"size:100;not null;uniqueIndex:idx_tenant_username" json:"username"`
	PasswordHash string     `gorm:"size:255;not null" json:"-"`
	FirstName    string     `gorm:"size:100" json:"first_name"`
	LastName     string     `gorm:"size:100" json:"last_name"`
	Role         string     `gorm:"size:50;not null;default:'customer'" json:"role"` // admin, engineer, support, customer, sales
	IsActive     bool       `gorm:"default:true" json:"is_active"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	Preferences  string     `gorm:"type:text" json:"preferences"` // JSON string
	Tickets      []Ticket   `gorm:"foreignKey:CreatedBy;references:ID" json:"tickets,omitempty"`
	Messages     []Message  `gorm:"foreignKey:UserID;references:ID" json:"messages,omitempty"`
}

// Ticket represents a support ticket
type Ticket struct {
	BaseModel
	TenantID       uint         `gorm:"not null;index" json:"tenant_id"`
	Tenant         Tenant       `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	TicketNumber   string       `gorm:"size:50;not null;uniqueIndex:idx_tenant_ticket" json:"ticket_number"`
	Title          string       `gorm:"size:255;not null" json:"title"`
	Description    string       `gorm:"type:text" json:"description"`
	Status         string       `gorm:"size:50;not null;default:'open'" json:"status"`     // open, in_progress, resolved, closed, cancelled
	Priority       string       `gorm:"size:20;not null;default:'medium'" json:"priority"` // low, medium, high, critical
	Severity       string       `gorm:"size:20;not null;default:'minor'" json:"severity"`  // trivial, minor, major, critical
	Category       string       `gorm:"size:100" json:"category"`
	Type           string       `gorm:"size:50" json:"type"`
	ProductID      *uint        `gorm:"index" json:"product_id"`
	Product        *Product     `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	ServiceID      *uint        `gorm:"index" json:"service_id"`
	Service        *Service     `gorm:"foreignKey:ServiceID" json:"service,omitempty"`
	AssignedTo     *uint        `gorm:"index" json:"assigned_to"`
	AssignedUser   *User        `gorm:"foreignKey:AssignedTo" json:"assigned_user,omitempty"`
	RequesterName  string       `gorm:"size:255" json:"requester_name"`
	RequesterEmail string       `gorm:"size:255" json:"requester_email"`
	Tags           string       `gorm:"type:text" json:"tags"`          // JSON array
	CustomFields   string       `gorm:"type:text" json:"custom_fields"` // JSON object
	IsDeleted      bool         `gorm:"default:false;index" json:"is_deleted"`
	ResolutionTime *time.Time   `json:"resolution_time"`
	ResolvedAt     *time.Time   `json:"resolved_at"`
	DueDate        *time.Time   `json:"due_date"`
	SLAStatus      string       `gorm:"size:20;default:'within'" json:"sla_status"` // within, breached, warning
	Messages       []Message    `gorm:"foreignKey:TicketID" json:"messages,omitempty"`
	Attachments    []Attachment `gorm:"foreignKey:TicketID" json:"attachments,omitempty"`
}

// Message represents a ticket message
type Message struct {
	BaseModel
	TicketID    uint         `gorm:"not null;index" json:"ticket_id"`
	Ticket      Ticket       `gorm:"foreignKey:TicketID" json:"ticket,omitempty"`
	UserID      uint         `gorm:"index" json:"user_id"`
	User        *User        `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Content     string       `gorm:"type:text;not null" json:"content"`
	ContentType string       `gorm:"size:50;default:'text'" json:"content_type"` // text, html, markdown
	IsInternal  bool         `gorm:"default:false" json:"is_internal"`
	IsFromAI    bool         `gorm:"default:false" json:"is_from_ai"`
	Attachments []Attachment `gorm:"foreignKey:MessageID" json:"attachments,omitempty"`
}

// Attachment represents a file attachment
type Attachment struct {
	BaseModel
	TicketID           uint              `gorm:"not null;index" json:"ticket_id"`
	Ticket             Ticket            `gorm:"foreignKey:TicketID" json:"ticket,omitempty"`
	MessageID          *uint             `gorm:"index" json:"message_id"`
	Message            *Message          `gorm:"foreignKey:MessageID" json:"message,omitempty"`
	KnowledgeArticleID *uint             `gorm:"index" json:"knowledge_article_id"`
	KnowledgeArticle   *KnowledgeArticle `gorm:"foreignKey:KnowledgeArticleID" json:"knowledge_article,omitempty"`
	FileName           string            `gorm:"size:255;not null" json:"file_name"`
	OriginalName       string            `gorm:"size:255;not null" json:"original_name"`
	FilePath           string            `gorm:"size:500;not null" json:"file_path"`
	FileSize           int64             `gorm:"not null" json:"file_size"`
	ContentType        string            `gorm:"size:100" json:"content_type"`
	Hash               string            `gorm:"size:64;index" json:"hash"` // SHA-256 hash
}

// KnowledgeArticle represents a knowledge base article
type KnowledgeArticle struct {
	BaseModel
	TenantID     uint              `gorm:"not null;index" json:"tenant_id"`
	Tenant       Tenant            `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Title        string            `gorm:"size:255;not null" json:"title"`
	Slug         string            `gorm:"size:255;not null;index" json:"slug"`
	Content      string            `gorm:"type:text" json:"content"`
	ContentType  string            `gorm:"size:50;default:'markdown'" json:"content_type"`
	Summary      string            `gorm:"type:text" json:"summary"`
	AuthorID     uint              `gorm:"index" json:"author_id"`
	Author       *User             `gorm:"foreignKey:AuthorID" json:"author,omitempty"`
	Status       string            `gorm:"size:50;not null;default:'draft'" json:"status"` // draft, published, archived
	Visibility   string            `gorm:"size:50;default:'public'" json:"visibility"`     // public, internal, private
	AccessLevel  string            `gorm:"size:50;default:'all'" json:"access_level"`      // all, agents, admins
	ProductID    *uint             `gorm:"index" json:"product_id"`
	Product      *Product          `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	ServiceID    *uint             `gorm:"index" json:"service_id"`
	Service      *Service          `gorm:"foreignKey:ServiceID" json:"service,omitempty"`
	Category     string            `gorm:"size:100" json:"category"`
	Tags         string            `gorm:"type:text" json:"tags"` // JSON array
	Views        int               `gorm:"default:0" json:"views"`
	HelpfulVotes int               `gorm:"default:0" json:"helpful_votes"`
	Version      int               `gorm:"default:1" json:"version"`
	ParentID     *uint             `gorm:"index" json:"parent_id"`
	Parent       *KnowledgeArticle `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Attachments  []Attachment      `gorm:"foreignKey:KnowledgeArticleID" json:"attachments,omitempty"`
}

// LLMProvider represents a configured LLM provider
type LLMProvider struct {
	BaseModel
	TenantID      uint    `gorm:"not null;index" json:"tenant_id"`
	Tenant        Tenant  `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Name          string  `gorm:"size:255;not null" json:"name"`
	ProviderType  string  `gorm:"size:50;not null" json:"provider_type"` // openai, azure, anthropic, deepseek, ollama, local
	APIEndpoint   string  `gorm:"size:500" json:"api_endpoint"`
	APIKey        string  `gorm:"size:500" json:"api_key"` // encrypted
	Model         string  `gorm:"size:100" json:"model"`
	MaxTokens     int     `gorm:"default:4096" json:"max_tokens"`
	Temperature   float64 `gorm:"default:0.7" json:"temperature"`
	TaskTypes     string  `gorm:"type:text" json:"task_types"` // JSON array
	IsDefault     bool    `gorm:"default:false" json:"is_default"`
	IsEnabled     bool    `gorm:"default:true" json:"is_enabled"`
	QuotaLimit    int     `gorm:"default:1000" json:"quota_limit"`
	QuotaUsed     int     `gorm:"default:0" json:"quota_used"`
	Configuration string  `gorm:"type:text" json:"configuration"` // JSON object
}

// ImportExportJob represents a data import/export job
type ImportExportJob struct {
	BaseModel
	TenantID         uint       `gorm:"not null;index" json:"tenant_id"`
	Tenant           Tenant     `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Type             string     `gorm:"size:20;not null" json:"type"`                     // import, export
	Status           string     `gorm:"size:50;not null;default:'pending'" json:"status"` // pending, running, completed, failed
	Progress         int        `gorm:"default:0" json:"progress"`
	TotalRecords     int        `gorm:"default:0" json:"total_records"`
	ProcessedRecords int        `gorm:"default:0" json:"processed_records"`
	FailedRecords    int        `gorm:"default:0" json:"failed_records"`
	SourceFormat     string     `gorm:"size:50" json:"source_format"` // csv, json, xml, zendesk, jira
	TargetFormat     string     `gorm:"size:50" json:"target_format"`
	FilePath         string     `gorm:"size:500" json:"file_path"`
	Configuration    string     `gorm:"type:text" json:"configuration"` // JSON object
	Error            string     `gorm:"type:text" json:"error"`
	StartedAt        *time.Time `json:"started_at"`
	CompletedAt      *time.Time `json:"completed_at"`
	StartedBy        uint       `gorm:"index" json:"started_by"`
	StartedByUser    *User      `gorm:"foreignKey:StartedBy" json:"started_by_user,omitempty"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	BaseModel
	TenantID     uint   `gorm:"not null;index" json:"tenant_id"`
	Tenant       Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	UserID       uint   `gorm:"index" json:"user_id"`
	User         *User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Action       string `gorm:"size:100;not null;index" json:"action"`
	ResourceType string `gorm:"size:100;not null;index" json:"resource_type"`
	ResourceID   uint   `gorm:"index" json:"resource_id"`
	ResourceName string `gorm:"size:255" json:"resource_name"`
	IPAddress    string `gorm:"size:45" json:"ip_address"`
	UserAgent    string `gorm:"size:500" json:"user_agent"`
	Changes      string `gorm:"type:text" json:"changes"`    // JSON object
	OldValues    string `gorm:"type:text" json:"old_values"` // JSON object
	NewValues    string `gorm:"type:text" json:"new_values"` // JSON object
	RequestID    string `gorm:"size:100;index" json:"request_id"`
	Hash         string `gorm:"size:64;index" json:"hash"` // SHA-256 hash for integrity
}

// APIKey represents an API key for external access
type APIKey struct {
	BaseModel
	TenantID    uint       `gorm:"not null;index" json:"tenant_id"`
	Tenant      Tenant     `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Name        string     `gorm:"size:255;not null" json:"name"`
	KeyHash     string     `gorm:"size:255;not null;uniqueIndex" json:"key_hash"`
	KeyPrefix   string     `gorm:"size:20;not null" json:"key_prefix"`
	Permissions string     `gorm:"type:text" json:"permissions"` // JSON array
	IsActive    bool       `gorm:"default:true" json:"is_active"`
	ExpiresAt   *time.Time `json:"expires_at"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	UsageCount  int        `gorm:"default:0" json:"usage_count"`
	CreatorID   uint       `gorm:"index" json:"creator_id"`
	Creator     *User      `gorm:"foreignKey:CreatorID" json:"creator,omitempty"`
}

// SystemSetting represents system-wide settings
type SystemSetting struct {
	BaseModel
	Key         string `gorm:"size:255;not null;uniqueIndex" json:"key"`
	Value       string `gorm:"type:text" json:"value"`
	Type        string `gorm:"size:50;not null;default:'string'" json:"type"` // string, int, bool, json
	Description string `gorm:"type:text" json:"description"`
	IsPublic    bool   `gorm:"default:false" json:"is_public"`
}

// Product represents a product or service offering
type Product struct {
	BaseModel
	TenantID          uint               `gorm:"not null;index" json:"tenant_id"`
	Tenant            Tenant             `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Name              string             `gorm:"size:255;not null" json:"name"`
	Code              string             `gorm:"size:100;not null" json:"code"`
	Description       string             `gorm:"type:text" json:"description"`
	Category          string             `gorm:"size:100" json:"category"`
	Version           string             `gorm:"size:50" json:"version"`
	Status            string             `gorm:"size:50;not null;default:'active'" json:"status"` // active, inactive, deprecated
	IsManaged         bool               `gorm:"default:false" json:"is_managed"`                 // 是否为托管服务
	SupportLevel      string             `gorm:"size:50;default:'basic'" json:"support_level"`    // basic, premium, enterprise
	Documentation     string             `gorm:"type:text" json:"documentation"`                  // 文档链接
	Configuration     string             `gorm:"type:text" json:"configuration"`                  // JSON配置对象
	Tags              string             `gorm:"type:text" json:"tags"`                           // JSON数组
	Services          []Service          `gorm:"foreignKey:ProductID" json:"services,omitempty"`
	KnowledgeArticles []KnowledgeArticle `gorm:"foreignKey:ProductID" json:"knowledge_articles,omitempty"`
}

// Service represents a specific service within a product
type Service struct {
	BaseModel
	TenantID          uint               `gorm:"not null;index" json:"tenant_id"`
	Tenant            Tenant             `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	ProductID         uint               `gorm:"index" json:"product_id"`
	Product           Product            `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Name              string             `gorm:"size:255;not null" json:"name"`
	Code              string             `gorm:"size:100;not null" json:"code"`
	Description       string             `gorm:"type:text" json:"description"`
	Type              string             `gorm:"size:50;not null" json:"type"`                    // infrastructure, application, support, consulting
	Status            string             `gorm:"size:50;not null;default:'active'" json:"status"` // active, inactive, maintenance
	Availability      string             `gorm:"size:50;default:'24x7'" json:"availability"`      // 24x7, business_hours, custom
	SupportChannels   string             `gorm:"type:text" json:"support_channels"`               // JSON数组: email, phone, chat, portal
	EscalationRules   string             `gorm:"type:text" json:"escalation_rules"`               // JSON配置对象
	Configuration     string             `gorm:"type:text" json:"configuration"`                  // JSON配置对象
	Tags              string             `gorm:"type:text" json:"tags"`                           // JSON数组
	Tickets           []Ticket           `gorm:"foreignKey:ServiceID" json:"tickets,omitempty"`
	KnowledgeArticles []KnowledgeArticle `gorm:"foreignKey:ServiceID" json:"knowledge_articles,omitempty"`
}

// SLATemplate represents SLA模板
type SLATemplate struct {
	BaseModel
	TenantID        uint      `gorm:"not null;index" json:"tenant_id"`
	Tenant          Tenant    `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Name            string    `gorm:"size:255;not null" json:"name"`
	Description     string    `gorm:"type:text" json:"description"`
	IsDefault       bool      `gorm:"default:false" json:"is_default"`
	IsActive        bool      `gorm:"default:true" json:"is_active"`
	PriorityLevels  string    `gorm:"type:text" json:"priority_levels"`  // JSON: low, medium, high, critical
	SeverityLevels  string    `gorm:"type:text" json:"severity_levels"`  // JSON: trivial, minor, major, critical
	ResponseTimes   string    `gorm:"type:text" json:"response_times"`   // JSON对象
	ResolutionTimes string    `gorm:"type:text" json:"resolution_times"` // JSON对象
	BusinessHours   string    `gorm:"type:text" json:"business_hours"`   // JSON配置对象
	Holidays        string    `gorm:"type:text" json:"holidays"`         // JSON数组
	Configuration   string    `gorm:"type:text" json:"configuration"`    // JSON配置对象
	SLARules        []SLARule `gorm:"foreignKey:SLATemplateID" json:"sla_rules,omitempty"`
}

// SLARule represents具体的SLA规则
type SLARule struct {
	BaseModel
	TenantID       uint        `gorm:"not null;index" json:"tenant_id"`
	Tenant         Tenant      `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	SLATemplateID  uint        `gorm:"index" json:"sla_template_id"`
	SLATemplate    SLATemplate `gorm:"foreignKey:SLATemplateID" json:"sla_template,omitempty"`
	ProductID      *uint       `gorm:"index" json:"product_id"`
	Product        *Product    `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	ServiceID      *uint       `gorm:"index" json:"service_id"`
	Service        *Service    `gorm:"foreignKey:ServiceID" json:"service,omitempty"`
	Priority       string      `gorm:"size:20;not null" json:"priority"`   // low, medium, high, critical
	Severity       string      `gorm:"size:20;not null" json:"severity"`   // trivial, minor, major, critical
	ResponseTime   int         `gorm:"not null" json:"response_time"`      // 响应时间(分钟)
	ResolutionTime int         `gorm:"not null" json:"resolution_time"`    // 解决时间(分钟)
	BusinessOnly   bool        `gorm:"default:false" json:"business_only"` // 仅工作时间
	IsActive       bool        `gorm:"default:true" json:"is_active"`
	Conditions     string      `gorm:"type:text" json:"conditions"` // JSON条件配置
}

// Permission represents a granular permission for resource-action combinations
type Permission struct {
	BaseModel
	Code            string           `gorm:"size:100;not null;uniqueIndex" json:"code"` // Format: resource:action
	Name            string           `gorm:"size:200;not null" json:"name"`
	Description     string           `gorm:"type:text" json:"description"`
	Category        string           `gorm:"size:50;not null" json:"category"` // tickets, users, knowledge, etc.
	IsSystem        bool             `gorm:"default:false" json:"is_system"`   // System permissions cannot be deleted
	RoleAssignments []RolePermission `gorm:"foreignKey:PermissionID" json:"role_assignments,omitempty"`
	UserAssignments []UserPermission `gorm:"foreignKey:PermissionID" json:"user_assignments,omitempty"`
}

// Role represents role definitions with associated permission sets
type Role struct {
	BaseModel
	TenantID    uint         `gorm:"not null;index" json:"tenant_id"`
	Tenant      Tenant       `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Name        string       `gorm:"size:100;not null" json:"name"`
	Description string       `gorm:"type:text" json:"description"`
	IsSystem    bool         `gorm:"default:false" json:"is_system"` // System roles cannot be deleted
	IsActive    bool         `gorm:"default:true" json:"is_active"`
	CreatedBy   uint         `gorm:"index" json:"created_by"`
	Creator     *User        `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
}

// RolePermission represents many-to-many relationship between roles and permissions
type RolePermission struct {
	BaseModel
	RoleID       uint       `gorm:"not null;index" json:"role_id"`
	Role         Role       `gorm:"foreignKey:RoleID;references:ID" json:"role,omitempty"`
	PermissionID uint       `gorm:"not null;index" json:"permission_id"`
	Permission   Permission `gorm:"foreignKey:PermissionID;references:ID" json:"permission,omitempty"`
	CreatedBy    uint       `gorm:"index" json:"created_by"`
	Granter      *User      `gorm:"foreignKey:CreatedBy;references:ID" json:"granter,omitempty"`
}

// UserPermission represents direct permission assignments to users (bypassing roles)
type UserPermission struct {
	BaseModel
	UserID       uint       `gorm:"not null;index" json:"user_id"`
	User         User       `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
	PermissionID uint       `gorm:"not null;index" json:"permission_id"`
	Permission   Permission `gorm:"foreignKey:PermissionID;references:ID" json:"permission,omitempty"`
	GrantedBy    uint       `gorm:"index" json:"granted_by"`
	ExpiresAt    *time.Time `json:"expires_at"` // Optional expiration
	Granter      *User      `gorm:"foreignKey:GrantedBy;references:ID" json:"granter,omitempty"`
}

// UserRole represents many-to-many relationship between users and roles
type UserRole struct {
	UserID     uint      `gorm:"primaryKey;not null;index" json:"user_id"`
	RoleID     uint      `gorm:"primaryKey;not null;index" json:"role_id"`
	AssignedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"assigned_at"`
	AssignedBy uint      `gorm:"not null;index" json:"assigned_by"`

	// Relationships
	User     User  `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
	Role     Role  `gorm:"foreignKey:RoleID;references:ID" json:"role,omitempty"`
	Assigner *User `gorm:"foreignKey:AssignedBy;references:ID" json:"assigner,omitempty"`
}

// Table name overrides
func (Tenant) TableName() string           { return "tenants" }
func (User) TableName() string             { return "users" }
func (Ticket) TableName() string           { return "tickets" }
func (Message) TableName() string          { return "messages" }
func (Attachment) TableName() string       { return "attachments" }
func (KnowledgeArticle) TableName() string { return "knowledge_articles" }
func (LLMProvider) TableName() string      { return "llm_providers" }
func (ImportExportJob) TableName() string  { return "import_export_jobs" }
func (AuditLog) TableName() string         { return "audit_logs" }
func (APIKey) TableName() string           { return "api_keys" }
func (SystemSetting) TableName() string    { return "system_settings" }
func (Product) TableName() string          { return "products" }
func (Service) TableName() string          { return "services" }
func (SLATemplate) TableName() string      { return "sla_templates" }
func (SLARule) TableName() string          { return "sla_rules" }
func (Permission) TableName() string       { return "permissions" }
func (Role) TableName() string             { return "roles" }
func (RolePermission) TableName() string   { return "role_permissions" }
func (UserPermission) TableName() string   { return "user_permissions" }
func (UserRole) TableName() string         { return "user_roles" }
