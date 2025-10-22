package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SeedData represents the complete seed data structure.
type SeedData struct {
	Tenants           []Tenant
	Users             []User
	Tickets           []Ticket
	KnowledgeArticles []KnowledgeArticle
	Settings          []Setting
	LLMProviders      []LLMProvider
	TicketCategories  []TicketCategory
	TicketStatuses    []TicketStatus
}

// Tenant represents a tenant record.
type Tenant struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name      string    `json:"name" gorm:"not null;type:varchar(255)"`
	Domain    string    `json:"domain" gorm:"uniqueIndex;not null;type:varchar(255)"`
	Status    string    `json:"status" gorm:"default:active;type:varchar(50)"`
	Settings  string    `json:"settings" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt time.Time `json:"deleted_at" gorm:"index"`
}

// User represents a user record.
type User struct {
	ID        string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TenantID  string     `json:"tenant_id" gorm:"not null;index;type:varchar(36)"`
	Email     string     `json:"email" gorm:"not null;uniqueIndex;type:varchar(255)"`
	Name      string     `json:"name" gorm:"not null;type:varchar(255)"`
	// Role field removed from User model - now handled by UserRole associations
	// Role assignment is now done through the UserRole table in the main application
	Status    string     `json:"status" gorm:"default:active;type:varchar(50)"`
	Password  string     `json:"password" gorm:"not null;type:varchar(255)"`
	LastLogin *time.Time `json:"last_login"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt time.Time  `json:"deleted_at" gorm:"index"`
}

// Ticket represents a ticket record.
type Ticket struct {
	ID          string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TenantID    string     `json:"tenant_id" gorm:"not null;index;type:varchar(36)"`
	Number      string     `json:"number" gorm:"not null;uniqueIndex:idx_tenant_number;type:varchar(50)"`
	Title       string     `json:"title" gorm:"not null;type:varchar(255)"`
	Description string     `json:"description" gorm:"type:text"`
	Status      string     `json:"status" gorm:"not null;index;type:varchar(50)"`
	Priority    string     `json:"priority" gorm:"not null;type:varchar(50)"`
	Severity    string     `json:"severity" gorm:"not null;type:varchar(50)"`
	Category    string     `json:"category" gorm:"not null;type:varchar(100)"`
	CreatedBy   string     `json:"created_by" gorm:"not null;type:varchar(36)"`
	AssignedTo  *string    `json:"assigned_to" gorm:"type:varchar(36)"`
	DueDate     *time.Time `json:"due_date"`
	ResolvedAt  *time.Time `json:"resolved_at"`
	ArchivedAt  *time.Time `json:"archived_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   time.Time  `json:"deleted_at" gorm:"index"`
}

// KnowledgeArticle represents a knowledge article.
type KnowledgeArticle struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TenantID    string    `json:"tenant_id" gorm:"not null;index;type:varchar(36)"`
	Title       string    `json:"title" gorm:"not null;type:varchar(255)"`
	Content     string    `json:"content" gorm:"type:text"`
	Summary     string    `json:"summary" gorm:"type:text"`
	Category    string    `json:"category" gorm:"not null;type:varchar(100)"`
	Tags        string    `json:"tags" gorm:"type:text"`
	Status      string    `json:"status" gorm:"not null;default:draft;type:varchar(50)"`
	AuthorID    string    `json:"author_id" gorm:"not null;type:varchar(36)"`
	ViewCount   int       `json:"view_count" gorm:"default:0"`
	IsPublished bool      `json:"is_published" gorm:"default:false"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   time.Time `json:"deleted_at" gorm:"index"`
}

// Setting represents a setting record.
type Setting struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TenantID  string    `json:"tenant_id" gorm:"not null;index;type:varchar(36)"`
	Key       string    `json:"key" gorm:"not null;uniqueIndex:idx_tenant_key;type:varchar(255)"`
	Value     string    `json:"value" gorm:"type:text"`
	Type      string    `json:"type" gorm:"not null;type:varchar(50)"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// LLMProvider represents an LLM provider configuration.
type LLMProvider struct {
	ID           string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TenantID     string    `json:"tenant_id" gorm:"not null;index;type:varchar(36)"`
	Name         string    `json:"name" gorm:"not null;type:varchar(255)"`
	ProviderType string    `json:"provider_type" gorm:"not null;type:varchar(50)"`
	APIEndpoint  string    `json:"api_endpoint" gorm:"type:varchar(500)"`
	APIKey       string    `json:"api_key" gorm:"type:varchar(500)"`
	Model        string    `json:"model" gorm:"type:varchar(100)"`
	MaxTokens    int       `json:"max_tokens" gorm:"default:4096"`
	Temperature  float64   `json:"temperature" gorm:"default:0.7"`
	TaskTypes    string    `json:"task_types" gorm:"type:text"`
	IsDefault    bool      `json:"is_default" gorm:"default:false"`
	IsEnabled    bool      `json:"is_enabled" gorm:"default:true"`
	QuotaLimit   int       `json:"quota_limit" gorm:"default:10000"`
	QuotaUsed    int       `json:"quota_used" gorm:"default:0"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	DeletedAt    time.Time `json:"deleted_at" gorm:"index"`
}

// TicketCategory represents a ticket category.
type TicketCategory struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TenantID  string    `json:"tenant_id" gorm:"not null;index;type:varchar(36)"`
	Name      string    `json:"name" gorm:"not null;type:varchar(100)"`
	Color     string    `json:"color" gorm:"default:#007bff;type:varchar(7)"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TicketStatus represents a ticket status.
type TicketStatus struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TenantID    string    `json:"tenant_id" gorm:"not null;index;type:varchar(36)"`
	Name        string    `json:"name" gorm:"not null;type:varchar(50)"`
	Description string    `json:"description" gorm:"type:text"`
	Color       string    `json:"color" gorm:"default:#6c757d;type:varchar(7)"`
	IsDefault   bool      `json:"is_default" gorm:"default:false"`
	IsFinal     bool      `json:"is_final" gorm:"default:false"`
	Order       int       `json:"order" gorm:"default:0"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// GenerateSeedData generates complete seed data for the system.
func GenerateSeedData() *SeedData {
	// Generate tenants
	tenants := generateTenants()

	// Generate users for each tenant
	users := make([]User, 0)
	for _, tenant := range tenants {
		users = append(users, generateUsers(tenant.ID)...)
	}

	// Generate ticket categories for each tenant
	categories := make([]TicketCategory, 0)
	for _, tenant := range tenants {
		categories = append(categories, generateTicketCategories(tenant.ID)...)
	}

	// Generate ticket statuses for each tenant
	statuses := make([]TicketStatus, 0)
	for _, tenant := range tenants {
		statuses = append(statuses, generateTicketStatuses(tenant.ID)...)
	}

	// Generate tickets for each tenant
	tickets := make([]Ticket, 0)
	for i, tenant := range tenants {
		// Get admin user for this tenant
		adminID := ""
		for _, user := range users {
			// Since User.Role field has been removed, we'll use the first user as admin for seed data
			// In the actual application, roles are handled through UserRole associations
			if user.TenantID == tenant.ID && adminID == "" {
				adminID = user.ID
				break
			}
		}

		if adminID != "" {
			tickets = append(tickets, generateTickets(tenant.ID, adminID, i+1)...)
		}
	}

	// Generate knowledge articles for each tenant
	articles := make([]KnowledgeArticle, 0)
	for _, tenant := range tenants {
		// Get admin user for this tenant
		adminID := ""
		for _, user := range users {
			// Since User.Role field has been removed, we'll use the first user as admin for seed data
			// In the actual application, roles are handled through UserRole associations
			if user.TenantID == tenant.ID && adminID == "" {
				adminID = user.ID
				break
			}
		}

		if adminID != "" {
			articles = append(articles, generateKnowledgeArticles(tenant.ID, adminID)...)
		}
	}

	// Generate settings for each tenant
	settings := make([]Setting, 0)
	for _, tenant := range tenants {
		settings = append(settings, generateSettings(tenant.ID)...)
	}

	// Generate LLM providers for each tenant
	llmProviders := make([]LLMProvider, 0)
	for _, tenant := range tenants {
		llmProviders = append(llmProviders, generateLLMProviders(tenant.ID)...)
	}

	return &SeedData{
		Tenants:           tenants,
		Users:             users,
		Tickets:           tickets,
		KnowledgeArticles: articles,
		Settings:          settings,
		LLMProviders:      llmProviders,
		TicketCategories:  categories,
		TicketStatuses:    statuses,
	}
}

// generateTenants generates sample tenants.
func generateTenants() []Tenant {
	now := time.Now()

	return []Tenant{
		{
			ID:        uuid.New().String(),
			Name:      "Acme Corporation",
			Domain:    "acme.example.com",
			Status:    "active",
			Settings:  `{"timezone": "America/New_York", "language": "en"}`,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        uuid.New().String(),
			Name:      "Globex Inc",
			Domain:    "globex.example.com",
			Status:    "active",
			Settings:  `{"timezone": "America/Los_Angeles", "language": "en"}`,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        uuid.New().String(),
			Name:      "Stark Industries",
			Domain:    "stark.example.com",
			Status:    "active",
			Settings:  `{"timezone": "America/New_York", "language": "en"}`,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

// generateUsers generates sample users for a tenant.
func generateUsers(tenantID string) []User {
	now := time.Now()

	// Hash password
	password, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

	// Create tenant-specific emails to avoid uniqueness conflicts
	tenantDomain := "tenant" + tenantID[:8] + ".example.com"

	return []User{
		{
			ID:        uuid.New().String(),
			TenantID:  tenantID,
			Email:     "admin@" + tenantDomain,
			Name:      "System Administrator",
			// Role field removed - roles are now handled through UserRole associations in main application
			Status:    "active",
			Password:  string(password),
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        uuid.New().String(),
			TenantID:  tenantID,
			Email:     "john.smith@" + tenantDomain,
			Name:      "John Smith",
			// Role field removed - roles are now handled through UserRole associations in main application
			Status:    "active",
			Password:  string(password),
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        uuid.New().String(),
			TenantID:  tenantID,
			Email:     "sarah.jones@" + tenantDomain,
			Name:      "Sarah Jones",
			// Role field removed - roles are now handled through UserRole associations in main application
			Status:    "active",
			Password:  string(password),
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        uuid.New().String(),
			TenantID:  tenantID,
			Email:     "mike.wilson@" + tenantDomain,
			Name:      "Mike Wilson",
			// Role field removed - roles are now handled through UserRole associations in main application
			Status:    "active",
			Password:  string(password),
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        uuid.New().String(),
			TenantID:  tenantID,
			Email:     "jane.doe@" + tenantDomain,
			Name:      "Jane Doe",
			// Role field removed - roles are now handled through UserRole associations in main application
			Status:    "active",
			Password:  string(password),
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

// generateTicketCategories generates sample ticket categories.
func generateTicketCategories(tenantID string) []TicketCategory {
	now := time.Now()

	return []TicketCategory{
		{
			ID:        uuid.New().String(),
			TenantID:  tenantID,
			Name:      "Bug Report",
			Color:     "#dc3545",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        uuid.New().String(),
			TenantID:  tenantID,
			Name:      "Feature Request",
			Color:     "#28a745",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        uuid.New().String(),
			TenantID:  tenantID,
			Name:      "Technical Issue",
			Color:     "#ffc107",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        uuid.New().String(),
			TenantID:  tenantID,
			Name:      "General Inquiry",
			Color:     "#17a2b8",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

// generateTicketStatuses generates sample ticket statuses.
func generateTicketStatuses(tenantID string) []TicketStatus {
	now := time.Now()

	return []TicketStatus{
		{
			ID:          uuid.New().String(),
			TenantID:    tenantID,
			Name:        "Open",
			Description: "Ticket has been opened and is awaiting triage",
			Color:       "#007bff",
			IsDefault:   true,
			IsFinal:     false,
			Order:       1,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          uuid.New().String(),
			TenantID:    tenantID,
			Name:        "In Progress",
			Description: "Ticket is being worked on",
			Color:       "#ffc107",
			IsDefault:   false,
			IsFinal:     false,
			Order:       2,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          uuid.New().String(),
			TenantID:    tenantID,
			Name:        "Pending Customer Response",
			Description: "Waiting for customer to respond",
			Color:       "#fd7e14",
			IsDefault:   false,
			IsFinal:     false,
			Order:       3,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          uuid.New().String(),
			TenantID:    tenantID,
			Name:        "Resolved",
			Description: "Ticket has been resolved",
			Color:       "#28a745",
			IsDefault:   false,
			IsFinal:     true,
			Order:       4,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          uuid.New().String(),
			TenantID:    tenantID,
			Name:        "Closed",
			Description: "Ticket has been closed",
			Color:       "#6c757d",
			IsDefault:   false,
			IsFinal:     true,
			Order:       5,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}
}

// generateTickets generates sample tickets for a tenant.
func generateTickets(tenantID, createdBy string, tenantIndex int) []Ticket {
	now := time.Now()

	return []Ticket{
		{
			ID:          uuid.New().String(),
			TenantID:    tenantID,
			Number:      fmt.Sprintf("T%d-001", tenantIndex),
			Title:       "Unable to login to system",
			Description: "I've been trying to login for the past hour but keep getting an invalid credentials error. I'm sure my password is correct.",
			Status:      "Open",
			Priority:    "high",
			Severity:    "medium",
			Category:    "Technical Issue",
			CreatedBy:   createdBy,
			CreatedAt:   now.Add(-2 * time.Hour),
			UpdatedAt:   now.Add(-2 * time.Hour),
		},
		{
			ID:          uuid.New().String(),
			TenantID:    tenantID,
			Number:      fmt.Sprintf("T%d-002", tenantIndex),
			Title:       "System running slowly",
			Description: "The system has been very slow today. Pages are taking a long time to load.",
			Status:      "In Progress",
			Priority:    "medium",
			Severity:    "low",
			Category:    "Technical Issue",
			CreatedBy:   createdBy,
			CreatedAt:   now.Add(-4 * time.Hour),
			UpdatedAt:   now.Add(-1 * time.Hour),
		},
		{
			ID:          uuid.New().String(),
			TenantID:    tenantID,
			Number:      fmt.Sprintf("T%d-003", tenantIndex),
			Title:       "Add dark mode feature",
			Description: "It would be great to have a dark mode option for the interface to reduce eye strain during extended use.",
			Status:      "Pending Customer Response",
			Priority:    "low",
			Severity:    "low",
			Category:    "Feature Request",
			CreatedBy:   createdBy,
			CreatedAt:   now.Add(-1 * 24 * time.Hour),
			UpdatedAt:   now.Add(-12 * time.Hour),
		},
		{
			ID:          uuid.New().String(),
			TenantID:    tenantID,
			Number:      fmt.Sprintf("T%d-004", tenantIndex),
			Title:       "Export functionality not working",
			Description: "When I try to export tickets to CSV, I get an error message saying the export failed.",
			Status:      "Resolved",
			Priority:    "medium",
			Severity:    "medium",
			Category:    "Bug Report",
			CreatedBy:   createdBy,
			CreatedAt:   now.Add(-3 * 24 * time.Hour),
			UpdatedAt:   now.Add(-2 * 24 * time.Hour),
		},
	}
}

// generateKnowledgeArticles generates sample knowledge articles.
func generateKnowledgeArticles(tenantID, authorID string) []KnowledgeArticle {
	now := time.Now()

	return []KnowledgeArticle{
		{
			ID:       uuid.New().String(),
			TenantID: tenantID,
			Title:    "How to Reset Your Password",
			Content: `# How to Reset Your Password

If you've forgotten your password or need to change it, follow these steps:

1. Go to the login page
2. Click on "Forgot Password"
3. Enter your email address
4. Check your email for a reset link
5. Click the reset link and enter your new password
6. Log in with your new password

## Tips for Strong Passwords
- Use at least 12 characters
- Include uppercase and lowercase letters
- Include numbers and special characters
- Don't use personal information
- Don't reuse passwords from other services

If you continue to have trouble, please contact support.`,
			Summary:     "Step-by-step guide for resetting your account password",
			Category:    "User Guide",
			Tags:        `["password", "login", "security", "user guide"]`,
			Status:      "published",
			AuthorID:    authorID,
			ViewCount:   156,
			IsPublished: true,
			CreatedAt:   now.Add(-7 * 24 * time.Hour),
			UpdatedAt:   now.Add(-7 * 24 * time.Hour),
		},
		{
			ID:       uuid.New().String(),
			TenantID: tenantID,
			Title:    "Common Login Issues and Solutions",
			Content: `# Common Login Issues and Solutions

This article covers the most common login problems and their solutions.

## 1. Incorrect Password
**Problem**: Getting "invalid credentials" error
**Solution**:
- Check that caps lock is off
- Reset your password if you've forgotten it
- Ensure you're using the correct email address

## 2. Account Locked
**Problem**: Account temporarily locked due to failed attempts
**Solution**:
- Wait 15 minutes for automatic unlock
- Contact administrator if still locked

## 3. Browser Issues
**Problem**: Login button not working or page not loading
**Solution**:
- Clear browser cache and cookies
- Try a different browser
- Disable browser extensions temporarily

## 4. Network Issues
**Problem**: Can't connect to the server
**Solution**:
- Check your internet connection
- Try accessing from a different network
- Contact IT if network issues persist

## 5. Two-Factor Authentication Problems
**Problem**: Not receiving 2FA codes
**Solution**:
- Check phone signal and SMS delivery
- Ensure time is correct on your device
- Try backup codes if available

For additional help, create a support ticket.`,
			Summary:     "Troubleshooting guide for common login problems",
			Category:    "Troubleshooting",
			Tags:        `["login", "troubleshooting", "2fa", "browser"]`,
			Status:      "published",
			AuthorID:    authorID,
			ViewCount:   89,
			IsPublished: true,
			CreatedAt:   now.Add(-5 * 24 * time.Hour),
			UpdatedAt:   now.Add(-2 * 24 * time.Hour),
		},
	}
}

// generateSettings generates default settings for a tenant.
func generateSettings(tenantID string) []Setting {
	now := time.Now()

	return []Setting{
		{
			ID:        uuid.New().String(),
			TenantID:  tenantID,
			Key:       fmt.Sprintf("%s.system.timezone", tenantID),
			Value:     "America/New_York",
			Type:      "string",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        uuid.New().String(),
			TenantID:  tenantID,
			Key:       fmt.Sprintf("%s.system.language", tenantID),
			Value:     "en",
			Type:      "string",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        uuid.New().String(),
			TenantID:  tenantID,
			Key:       fmt.Sprintf("%s.tickets.auto_number", tenantID),
			Value:     "true",
			Type:      "boolean",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        uuid.New().String(),
			TenantID:  tenantID,
			Key:       fmt.Sprintf("%s.notifications.email_enabled", tenantID),
			Value:     "true",
			Type:      "boolean",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        uuid.New().String(),
			TenantID:  tenantID,
			Key:       fmt.Sprintf("%s.security.session_timeout", tenantID),
			Value:     "3600",
			Type:      "integer",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

// generateLLMProviders generates sample LLM providers.
func generateLLMProviders(tenantID string) []LLMProvider {
	now := time.Now()

	return []LLMProvider{
		{
			ID:           uuid.New().String(),
			TenantID:     tenantID,
			Name:         "OpenAI GPT",
			ProviderType: "openai",
			APIEndpoint:  "https://api.openai.com/v1",
			Model:        "gpt-4o-mini",
			MaxTokens:    4096,
			Temperature:  0.7,
			TaskTypes:    `["chat", "generation", "summarization"]`,
			IsDefault:    true,
			IsEnabled:    false,
			QuotaLimit:   10000,
			QuotaUsed:    0,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			ID:           uuid.New().String(),
			TenantID:     tenantID,
			Name:         "Local Ollama",
			ProviderType: "ollama",
			APIEndpoint:  "http://localhost:11434",
			Model:        "llama3:latest",
			MaxTokens:    2048,
			Temperature:  0.8,
			TaskTypes:    `["chat", "generation"]`,
			IsDefault:    false,
			IsEnabled:    true,
			QuotaLimit:   1000,
			QuotaUsed:    0,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}
}

// SaveSeedData saves seed data to a JSON file.
func SaveSeedData(data *SeedData, filename string) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal seed data: %w", err)
	}

	err = os.WriteFile(filename, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write seed data file: %w", err)
	}

	log.Printf("Seed data saved to %s", filename)
	return nil
}

// LoadSeedData loads seed data from a JSON file.
func LoadSeedData(filename string) (*SeedData, error) {
	jsonData, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read seed data file: %w", err)
	}

	var data SeedData
	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal seed data: %w", err)
	}

	log.Printf("Seed data loaded from %s", filename)
	return &data, nil
}

// SeedDatabase seeds the database with the provided data.
func SeedDatabase(db *gorm.DB, data *SeedData) error {
	ctx := context.Background()

	log.Println("Starting database seeding...")

	// Seed tenants
	if err := seedTenants(ctx, db, data.Tenants); err != nil {
		return fmt.Errorf("failed to seed tenants: %w", err)
	}

	// Seed ticket categories
	if err := seedTicketCategories(ctx, db, data.TicketCategories); err != nil {
		return fmt.Errorf("failed to seed ticket categories: %w", err)
	}

	// Seed ticket statuses
	if err := seedTicketStatuses(ctx, db, data.TicketStatuses); err != nil {
		return fmt.Errorf("failed to seed ticket statuses: %w", err)
	}

	// Seed users
	if err := seedUsers(ctx, db, data.Users); err != nil {
		return fmt.Errorf("failed to seed users: %w", err)
	}

	// Seed tickets
	if err := seedTickets(ctx, db, data.Tickets); err != nil {
		return fmt.Errorf("failed to seed tickets: %w", err)
	}

	// Seed knowledge articles
	if err := seedKnowledgeArticles(ctx, db, data.KnowledgeArticles); err != nil {
		return fmt.Errorf("failed to seed knowledge articles: %w", err)
	}

	// Seed settings
	if err := seedSettings(ctx, db, data.Settings); err != nil {
		return fmt.Errorf("failed to seed settings: %w", err)
	}

	// Seed LLM providers
	if err := seedLLMProviders(ctx, db, data.LLMProviders); err != nil {
		return fmt.Errorf("failed to seed LLM providers: %w", err)
	}

	log.Println("Database seeding completed successfully!")
	return nil
}

// Helper functions to seed each table.
func seedTenants(ctx context.Context, db *gorm.DB, tenants []Tenant) error {
	for _, tenant := range tenants {
		if err := db.Create(&tenant).Error; err != nil {
			return err
		}
	}
	log.Printf("Seeded %d tenants", len(tenants))
	return nil
}

func seedUsers(ctx context.Context, db *gorm.DB, users []User) error {
	for _, user := range users {
		if err := db.Create(&user).Error; err != nil {
			return err
		}
	}
	log.Printf("Seeded %d users", len(users))
	return nil
}

func seedTicketCategories(ctx context.Context, db *gorm.DB, categories []TicketCategory) error {
	for _, category := range categories {
		if err := db.Create(&category).Error; err != nil {
			return err
		}
	}
	log.Printf("Seeded %d ticket categories", len(categories))
	return nil
}

func seedTicketStatuses(ctx context.Context, db *gorm.DB, statuses []TicketStatus) error {
	for _, status := range statuses {
		if err := db.Create(&status).Error; err != nil {
			return err
		}
	}
	log.Printf("Seeded %d ticket statuses", len(statuses))
	return nil
}

func seedTickets(ctx context.Context, db *gorm.DB, tickets []Ticket) error {
	for _, ticket := range tickets {
		if err := db.Create(&ticket).Error; err != nil {
			return err
		}
	}
	log.Printf("Seeded %d tickets", len(tickets))
	return nil
}

func seedKnowledgeArticles(ctx context.Context, db *gorm.DB, articles []KnowledgeArticle) error {
	for _, article := range articles {
		if err := db.Create(&article).Error; err != nil {
			return err
		}
	}
	log.Printf("Seeded %d knowledge articles", len(articles))
	return nil
}

func seedSettings(ctx context.Context, db *gorm.DB, settings []Setting) error {
	for _, setting := range settings {
		if err := db.Create(&setting).Error; err != nil {
			return err
		}
	}
	log.Printf("Seeded %d settings", len(settings))
	return nil
}

func seedLLMProviders(ctx context.Context, db *gorm.DB, providers []LLMProvider) error {
	for _, provider := range providers {
		if err := db.Create(&provider).Error; err != nil {
			return err
		}
	}
	log.Printf("Seeded %d LLM providers", len(providers))
	return nil
}
