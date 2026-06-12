package models

import (
	"time"

	"gorm.io/gorm"
)

// Base model with common fields.
type BaseModel struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedBy *string        `gorm:"size:255" json:"created_by,omitempty"`
	UpdatedBy *string        `gorm:"size:255" json:"updated_by,omitempty"`
}

// User represents a user account.
// Customer represents a customer organization (a company the operator serves).
// Customer-role users belong to one Customer; their tickets are scoped to it.
type Customer struct {
	BaseModel
	Name string `gorm:"size:255;not null;index" json:"name"`
	// Code is an optional unique short code. It is a pointer so that multiple
	// customers without a code store NULL (distinct under the unique index)
	// rather than colliding on the empty string.
	Code        *string  `gorm:"size:100;uniqueIndex" json:"code,omitempty"`
	Domain      string   `gorm:"size:255;index" json:"domain"`
	IsActive    bool     `gorm:"default:true" json:"is_active"`
	Description string   `gorm:"type:text" json:"description"`
	Users       []User   `gorm:"foreignKey:CustomerID" json:"users,omitempty"`
	Tickets     []Ticket `gorm:"foreignKey:CustomerID" json:"tickets,omitempty"`
}

type User struct {
	BaseModel
	Email        string     `gorm:"size:255;not null;uniqueIndex" json:"email"`
	Username     string     `gorm:"size:100;not null;uniqueIndex" json:"username"`
	PasswordHash string     `gorm:"size:255;not null" json:"-"`
	FirstName    string     `gorm:"size:100" json:"first_name"`
	LastName     string     `gorm:"size:100" json:"last_name"`
	Role         string     `gorm:"size:50;default:'customer'" json:"role"`
	IsActive     bool       `gorm:"default:true" json:"is_active"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	Preferences  string     `gorm:"type:text" json:"preferences"` // JSON string
	// CustomerID links a customer-role user to the customer organization they
	// belong to. Nil for team users (admin/engineer).
	CustomerID *uint     `gorm:"index" json:"customer_id,omitempty"`
	Customer   *Customer `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	Tickets    []Ticket  `gorm:"foreignKey:CreatedBy;references:ID" json:"tickets,omitempty"`
	Messages   []Message `gorm:"foreignKey:UserID;references:ID" json:"messages,omitempty"`
	Roles      []Role    `gorm:"many2many:user_roles;" json:"roles,omitempty"`
}

// Ticket represents a support ticket.
type Ticket struct {
	BaseModel
	TicketNumber   string     `gorm:"size:50;not null;uniqueIndex" json:"ticket_number"`
	Title          string     `gorm:"size:255;not null" json:"title"`
	Description    string     `gorm:"type:text" json:"description"`
	Status         string     `gorm:"size:50;not null;default:'open'" json:"status"`     // open, in_progress, resolved, closed, cancelled
	Priority       string     `gorm:"size:20;not null;default:'medium'" json:"priority"` // low, medium, high, critical
	Severity       string     `gorm:"size:20;not null;default:'minor'" json:"severity"`  // trivial, minor, major, critical
	Category       string     `gorm:"size:100" json:"category"`
	Type           string     `gorm:"size:50" json:"type"`
	ProductID      *uint      `gorm:"index" json:"product_id"`
	Product        *Product   `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	ServiceID      *uint      `gorm:"index" json:"service_id"`
	Service        *Service   `gorm:"foreignKey:ServiceID" json:"service,omitempty"`
	CustomerID     *uint      `gorm:"index" json:"customer_id,omitempty"`
	Customer       *Customer  `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	AssignedTo     *uint      `gorm:"index" json:"assigned_to"`
	AssignedUser   *User      `gorm:"foreignKey:AssignedTo" json:"assigned_user,omitempty"`
	RequesterName  string     `gorm:"size:255" json:"requester_name"`
	RequesterEmail string     `gorm:"size:255" json:"requester_email"`
	Tags           string     `gorm:"type:text" json:"tags"`          // JSON array
	CustomFields   string     `gorm:"type:text" json:"custom_fields"` // JSON object
	IsDeleted      bool       `gorm:"default:false;index" json:"is_deleted"`
	ResolutionTime *time.Time `json:"resolution_time"`
	ResolvedAt     *time.Time `json:"resolved_at"`
	DueDate        *time.Time `json:"due_date"`
	SLAStatus      string     `gorm:"size:20;default:'within'" json:"sla_status"` // within, breached, warning
	// Parity fields — added together to avoid repeat migrations.
	Channel           string       `gorm:"size:30;default:'web'" json:"channel"` // web, email, web_widget
	ConversationToken string       `gorm:"type:text;index" json:"conversation_token,omitempty"`
	Summary           string       `gorm:"type:text" json:"summary,omitempty"`
	AssignedTeamID    *uint        `gorm:"index" json:"assigned_team_id,omitempty"`
	MergedIntoID      *uint        `gorm:"index" json:"merged_into_id,omitempty"`
	Messages          []Message    `gorm:"foreignKey:TicketID" json:"messages,omitempty"`
	Attachments       []Attachment `gorm:"foreignKey:TicketID" json:"attachments,omitempty"`
}

// Message represents a ticket message.
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

// Attachment represents a file attachment.
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

// TicketEvent is an append-only record of an operation performed on a ticket
// (creation, status/priority changes, assignment, replies and notes). It powers
// the ticket activity/history timeline.
type TicketEvent struct {
	BaseModel
	TicketID uint   `gorm:"index;not null" json:"ticket_id"`
	UserID   uint   `gorm:"index" json:"user_id"` // acting user; 0 = system
	User     *User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Action   string `gorm:"size:50;not null" json:"action"` // created, status, priority, severity, assigned, replied, note, updated
	Summary  string `gorm:"size:500" json:"summary"`
}

// KnowledgeArticle represents a knowledge base article.
type KnowledgeArticle struct {
	BaseModel
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

// LLMProvider represents a configured LLM provider.
type LLMProvider struct {
	BaseModel
	Name         string `gorm:"size:255;not null" json:"name"`
	ProviderType string `gorm:"size:50;not null" json:"provider_type"` // openai, azure, anthropic, deepseek, ollama, local
	APIEndpoint  string `gorm:"size:500" json:"api_endpoint"`
	// APIKey holds AES-GCM ciphertext; json:"-" so it never serializes to clients.
	APIKey string `gorm:"size:500" json:"-"`
	Model  string `gorm:"size:100" json:"model"`
	// Dimensions is the embedding output dimension (used when TaskTypes includes "embedding").
	Dimensions    int     `gorm:"default:1024" json:"dimensions"`
	MaxTokens     int     `gorm:"default:4096" json:"max_tokens"`
	Temperature   float64 `gorm:"default:0.7" json:"temperature"`
	TaskTypes     string  `gorm:"type:text" json:"task_types"` // JSON array
	IsDefault     bool    `gorm:"default:false" json:"is_default"`
	IsEnabled     bool    `gorm:"default:true" json:"is_enabled"`
	QuotaLimit    int     `gorm:"default:1000" json:"quota_limit"`
	QuotaUsed     int     `gorm:"default:0" json:"quota_used"`
	Configuration string  `gorm:"type:text" json:"configuration"` // JSON object
}

// ImportExportJob represents a data import/export job.
type ImportExportJob struct {
	BaseModel
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

// AuditLog represents an audit log entry.
type AuditLog struct {
	BaseModel
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

// AnalyticsEvent stores privacy-preserving website/demo analytics. It does not
// store raw IP addresses; VisitorHash is a daily hash of IP + user agent.
type AnalyticsEvent struct {
	BaseModel
	EventType   string `gorm:"size:30;not null;index" json:"event_type"` // pageview, click
	Path        string `gorm:"size:500;index" json:"path"`
	Title       string `gorm:"size:255" json:"title"`
	Referrer    string `gorm:"size:500;index" json:"referrer"`
	Source      string `gorm:"size:100;index" json:"source"`
	Locale      string `gorm:"size:20;index" json:"locale"`
	Target      string `gorm:"size:255;index" json:"target"`
	UserAgent   string `gorm:"size:500" json:"user_agent"`
	DeviceType  string `gorm:"size:30;index" json:"device_type"`
	VisitorHash string `gorm:"size:64;index" json:"visitor_hash"`
}

// APIKey represents an API key bound to a service-account user for external access.
// Authentication inherits the bound user's RBAC; revoke by setting IsActive=false.
type APIKey struct {
	BaseModel
	Name       string     `gorm:"size:255;not null" json:"name"`
	KeyHash    string     `gorm:"size:255;not null;uniqueIndex" json:"-"`
	KeyPrefix  string     `gorm:"size:20;not null" json:"key_prefix"`
	UserID     uint       `gorm:"index;not null" json:"user_id"` // bound service account; inherits its RBAC
	User       *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	IsActive   bool       `gorm:"default:true" json:"is_active"` // revoke = set false
	ExpiresAt  *time.Time `json:"expires_at"`                    // nil = never expires
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatorID  uint       `gorm:"index" json:"creator_id"`
	Creator    *User      `gorm:"foreignKey:CreatorID" json:"creator,omitempty"`
}

// SystemSetting represents system-wide settings.
type SystemSetting struct {
	BaseModel
	Key         string `gorm:"size:255;not null;uniqueIndex" json:"key"`
	Value       string `gorm:"type:text" json:"value"`
	Type        string `gorm:"size:50;not null;default:'string'" json:"type"` // string, int, bool, json
	Description string `gorm:"type:text" json:"description"`
	IsPublic    bool   `gorm:"default:false" json:"is_public"`
}

// Branding holds the org-wide white-label configuration for this single-tenant
// deployment: the displayed product/workspace names, the signal accent color
// and an optional uploaded logo. It is a singleton — exactly one row (ID 1) is
// ever created, lazily, by the branding service. Read access is public (the
// login page and app shell render it before authentication); writes are
// admin-only.
type Branding struct {
	BaseModel
	AppName       string `gorm:"size:100" json:"app_name"`
	AppSubtitle   string `gorm:"size:100" json:"app_subtitle"`
	WorkspaceName string `gorm:"size:120" json:"workspace_name"`
	PrimaryColor  string `gorm:"size:32" json:"primary_color"`
	LoginTagline  string `gorm:"size:200" json:"login_tagline"`
	LoginSubtext  string `gorm:"size:300" json:"login_subtext"`
	// LogoPath is the on-disk path of the uploaded logo (never exposed to
	// clients); LogoExt is its extension (incl. leading dot). Empty when no
	// logo has been uploaded.
	LogoPath string `gorm:"size:512" json:"-"`
	LogoExt  string `gorm:"size:16" json:"-"`
}

// AISettings is the singleton row holding deployment-wide AI feature toggles.
// Every AI capability is gated on these so operators decide what is enabled —
// AI is opt-in and fully under the deployment's control.
type AISettings struct {
	BaseModel
	// Enabled is the master switch; when false no AI feature runs.
	Enabled bool `gorm:"default:true" json:"enabled"`
	// SuggestReplies lets agents request an AI-drafted reply on a ticket.
	SuggestReplies bool `gorm:"default:true" json:"suggest_replies"`
	// KnowledgeAI enables semantic search + "ask" over the knowledge base.
	KnowledgeAI bool `gorm:"default:true" json:"knowledge_ai"`
	// AutoClassify lets AI suggest a category/priority on new tickets.
	AutoClassify bool `gorm:"default:false" json:"auto_classify"`
	// ReplyInstructions is optional operator guidance (tone, do/don'ts) injected
	// into the suggested-reply prompt.
	ReplyInstructions string `gorm:"type:text" json:"reply_instructions"`
	// AutoReplyEnabled lets the AI post a public reply automatically when
	// confidence >= AutoReplyConfidence and NeedsClarification is false.
	AutoReplyEnabled bool `gorm:"default:false" json:"auto_reply_enabled"`
	// AutoReplyConfidence is the minimum Draft.Confidence (0..1) required before
	// the orchestrator posts an automated reply without agent review.
	AutoReplyConfidence float64 `gorm:"default:0.75" json:"auto_reply_confidence"`
	// AutoResolveEnabled allows the orchestrator to mark a ticket resolved after
	// posting a high-confidence auto-reply (Phase 3; post-reply action today).
	AutoResolveEnabled bool `gorm:"default:false" json:"auto_resolve_enabled"`
	// MaxAutoRepliesPerTicket caps how many AI-authored messages the orchestrator
	// will post on a single ticket before handing off to a human agent.
	MaxAutoRepliesPerTicket int `gorm:"default:2" json:"max_auto_replies_per_ticket"`
	// AutoSummarizeOnResolve triggers an LLM summary of the conversation when a
	// ticket transitions to resolved status.
	AutoSummarizeOnResolve bool `gorm:"default:false" json:"auto_summarize_on_resolve"`
}

// Product represents a product or service offering.
type Product struct {
	BaseModel
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

// Service represents a specific service within a product.
type Service struct {
	BaseModel
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

// SLATemplate represents SLA模板.
type SLATemplate struct {
	BaseModel
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

// Subscription is a customer's purchased support/licensing subscription for a
// product, billed per node (or per cluster) over a term (typically annual).
type Subscription struct {
	BaseModel
	CustomerID    uint         `gorm:"index;not null" json:"customer_id"`
	Customer      *Customer    `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	ProductID     uint         `gorm:"index;not null" json:"product_id"`
	Product       *Product     `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	SLATemplateID *uint        `gorm:"index" json:"sla_template_id"`
	SLATemplate   *SLATemplate `gorm:"foreignKey:SLATemplateID" json:"sla_template,omitempty"`
	Plan          string       `gorm:"size:100" json:"plan"`                           // e.g. "Standard", "Premium 24x7"
	BillingUnit   string       `gorm:"size:20;default:'per_node'" json:"billing_unit"` // per_node | per_cluster
	NodeCount     int          `gorm:"default:1" json:"node_count"`
	BillingPeriod string       `gorm:"size:20;default:'annual'" json:"billing_period"` // annual | monthly
	StartsAt      time.Time    `json:"starts_at"`
	ExpiresAt     time.Time    `json:"expires_at"`
	Status        string       `gorm:"size:20;default:'active'" json:"status"` // active | expired | cancelled
	UnitPrice     float64      `json:"unit_price"`                             // price per unit (per node) per period
	Currency      string       `gorm:"size:10;default:'USD'" json:"currency"`
	Notes         string       `gorm:"type:text" json:"notes"`
}

// TableName overrides the default table name for Subscription.
func (Subscription) TableName() string { return "subscriptions" }

// SLARule represents具体的SLA规则.
type SLARule struct {
	BaseModel
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

// Permission represents a granular permission for resource-action combinations.
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

// Role represents role definitions with associated permission sets.
type Role struct {
	BaseModel
	Name        string       `gorm:"size:100;not null" json:"name"`
	Description string       `gorm:"type:text" json:"description"`
	IsSystem    bool         `gorm:"default:false" json:"is_system"` // System roles cannot be deleted
	IsActive    bool         `gorm:"default:true" json:"is_active"`
	CreatedBy   uint         `gorm:"index" json:"created_by"`
	Creator     *User        `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
}

// RolePermission represents many-to-many relationship between roles and permissions.
type RolePermission struct {
	BaseModel
	RoleID       uint       `gorm:"not null;index" json:"role_id"`
	Role         Role       `gorm:"foreignKey:RoleID;references:ID" json:"role,omitempty"`
	PermissionID uint       `gorm:"not null;index" json:"permission_id"`
	Permission   Permission `gorm:"foreignKey:PermissionID;references:ID" json:"permission,omitempty"`
	CreatedBy    uint       `gorm:"index" json:"created_by"`
	Granter      *User      `gorm:"foreignKey:CreatedBy;references:ID" json:"granter,omitempty"`
}

// UserPermission represents direct permission assignments to users (bypassing roles).
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

// UserRole represents many-to-many relationship between users and roles.
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

// Notification is an in-app notification for a single recipient user.
type Notification struct {
	BaseModel
	UserID  uint   `gorm:"index;not null" json:"user_id"` // recipient
	Type    string `gorm:"size:50;not null" json:"type"`  // ticket_reply, ticket_assigned, ticket_status, ticket_created
	Title   string `gorm:"size:255;not null" json:"title"`
	Body    string `gorm:"type:text" json:"body"`
	RefType string `gorm:"size:50" json:"ref_type"` // e.g. "ticket"
	RefID   uint   `gorm:"index" json:"ref_id"`     // e.g. ticket ID
	IsRead  bool   `gorm:"default:false;index" json:"is_read"`
}

func (Notification) TableName() string { return "notifications" }

// Team is a named group of users (agents/admins) used for ticket assignment and
// @mention routing. Teams are managed by admins.
type Team struct {
	BaseModel
	Name        string `gorm:"size:120;not null;uniqueIndex" json:"name"`
	Description string `gorm:"type:text" json:"description"`
}

// TeamMember records that a user belongs to a team. The composite unique index
// prevents duplicate memberships at the database level.
type TeamMember struct {
	BaseModel
	TeamID uint `gorm:"index;not null;uniqueIndex:idx_team_user" json:"team_id"`
	UserID uint `gorm:"index;not null;uniqueIndex:idx_team_user" json:"user_id"`
}

func (Team) TableName() string       { return "teams" }
func (TeamMember) TableName() string { return "team_members" }

// SatisfactionSurvey captures a post-resolution CSAT survey for a ticket.
// One survey per ticket: CreateForTicket is idempotent. Rating 0 means the
// customer has not yet answered.
type SatisfactionSurvey struct {
	BaseModel
	TicketID    uint       `gorm:"index;not null" json:"ticket_id"`
	Rating      int        `json:"rating"` // 1..5, 0 = not yet answered
	Comment     string     `gorm:"type:text" json:"comment"`
	Token       string     `gorm:"size:128;uniqueIndex" json:"-"` // public access token
	SentAt      *time.Time `json:"sent_at"`
	RespondedAt *time.Time `json:"responded_at"`
}

func (SatisfactionSurvey) TableName() string { return "satisfaction_surveys" }

// TicketLink records a directional relationship between two tickets.
// The composite unique index (source_id, target_id, type) prevents duplicate
// links of the same type between the same pair of tickets.
type TicketLink struct {
	BaseModel
	SourceID uint   `gorm:"index;not null;uniqueIndex:idx_link" json:"source_id"`
	TargetID uint   `gorm:"index;not null;uniqueIndex:idx_link" json:"target_id"`
	Type     string `gorm:"size:20;not null;uniqueIndex:idx_link" json:"type"` // related|duplicate|blocks
}

func (TicketLink) TableName() string { return "ticket_links" }

// AutomationRule defines a trigger-condition-action rule that runs automatically
// when a matching domain event fires.
type AutomationRule struct {
	BaseModel
	Name        string `gorm:"size:200;not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`
	Enabled     bool   `gorm:"default:true" json:"enabled"`
	// Event is the trigger event type, matching automation.EventType constants.
	// Values: ticket.created | ticket.updated | message.created | ticket.sla_warning | schedule
	Event string `gorm:"size:50;not null;index" json:"event"`
	// Match controls whether all (AND) or any (OR) conditions must pass.
	// Values: all | any
	Match string `gorm:"size:10;default:'all'" json:"match"`
	// Conditions is a JSON array of condition objects: [{field,op,value}]
	Conditions string `gorm:"type:text" json:"conditions"`
	// Actions is a JSON array of action objects: [{type,params}]
	Actions string `gorm:"type:text" json:"actions"`
	// Position controls rule evaluation order (ascending).
	Position int `gorm:"default:0;index" json:"position"`
}

func (AutomationRule) TableName() string { return "automation_rules" }

// Macro is a canned response template an agent can apply to a ticket reply.
// Shared=true macros are visible to all team members; Shared=false macros are
// private to their OwnerID. The Body supports {{variable}} placeholders
// substituted at apply-time (see internal/macro.Render).
// Actions is an optional JSON array of side-effects: [{type, params}].
type Macro struct {
	BaseModel
	Title      string `gorm:"size:200;not null" json:"title"`
	Category   string `gorm:"size:100;index" json:"category"`
	Body       string `gorm:"type:text;not null" json:"body"`
	Actions    string `gorm:"type:text" json:"actions"`
	Shared     bool   `gorm:"default:false" json:"shared"`
	OwnerID    uint   `gorm:"index" json:"owner_id"`
	UsageCount int    `gorm:"default:0" json:"usage_count"`
}

func (Macro) TableName() string { return "macros" }

// Webhook is an admin-configured outbound endpoint that receives ticket domain
// events as signed HTTP POSTs. Secret is the HMAC signing key (never exposed to
// clients). Events is a JSON array of subscribed event-type strings.
type Webhook struct {
	BaseModel
	Name      string `gorm:"size:255;not null" json:"name"`
	URL       string `gorm:"size:1024;not null" json:"url"`
	Secret    string `gorm:"size:255;not null" json:"-"`
	Events    string `gorm:"type:text" json:"events"`
	Active    bool   `gorm:"default:true" json:"active"`
	CreatorID uint   `gorm:"index" json:"creator_id"`
}

// WebhookDelivery is one queued/attempted delivery of an event to a Webhook.
// Status is pending/success/failed; the worker retries failed rows up to a cap.
type WebhookDelivery struct {
	BaseModel
	WebhookID     uint       `gorm:"index;not null" json:"webhook_id"`
	EventType     string     `gorm:"size:64;index" json:"event_type"`
	Payload       string     `gorm:"type:text" json:"payload"`
	Status        string     `gorm:"size:16;index;default:'pending'" json:"status"`
	StatusCode    int        `json:"status_code"`
	Attempts      int        `json:"attempts"`
	LastAttemptAt *time.Time `json:"last_attempt_at"`
	Error         string     `gorm:"type:text" json:"error"`
}

func (Webhook) TableName() string         { return "webhooks" }
func (WebhookDelivery) TableName() string { return "webhook_deliveries" }

// Table name overrides.
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
