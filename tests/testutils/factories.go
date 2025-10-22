package testutils

import (
	"context"
	"fmt"
	"time"

	"github.com/company/smartticket/internal/database"
	"github.com/company/smartticket/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// DataFactory provides factory methods for creating test data.
type DataFactory struct {
	db        *database.Database
	tenant    *models.Tenant
	users     map[string]*models.User
	tickets   []*models.Ticket
	articles  []*models.KnowledgeArticle
	providers []*models.LLMProvider
}

// NewDataFactory creates a new data factory.
func NewDataFactory(db *database.Database) *DataFactory {
	return &DataFactory{
		db:    db,
		users: make(map[string]*models.User),
	}
}

// CreateTestData creates a comprehensive set of test data.
func (df *DataFactory) CreateTestData(ctx context.Context) error {
	// Create tenant
	if err := df.CreateTenant(ctx); err != nil {
		return fmt.Errorf("failed to create tenant: %w", err)
	}

	// Create users
	if err := df.CreateUsers(ctx); err != nil {
		return fmt.Errorf("failed to create users: %w", err)
	}

	// Create tickets
	if err := df.CreateTickets(ctx); err != nil {
		return fmt.Errorf("failed to create tickets: %w", err)
	}

	// Create knowledge articles
	if err := df.CreateKnowledgeArticles(ctx); err != nil {
		return fmt.Errorf("failed to create knowledge articles: %w", err)
	}

	// Create LLM providers
	if err := df.CreateLLMProviders(ctx); err != nil {
		return fmt.Errorf("failed to create LLM providers: %w", err)
	}

	return nil
}

// CreateTenant creates a test tenant.
func (df *DataFactory) CreateTenant(ctx context.Context) error {
	now := time.Now()
	df.tenant = &models.Tenant{
		BaseModel: models.BaseModel{
			ID:        1,
			CreatedAt: now,
			UpdatedAt: now,
		},
		Name:     "Test Corporation",
		Slug:     "test-corporation",
		Domain:   "test.example.com",
		Plan:     "enterprise",
		Settings: `{"timezone": "UTC", "locale": "en-US"}`,
		IsActive: true,
	}

	return df.db.Create(df.tenant).Error
}

// GetTenant returns the created tenant.
func (df *DataFactory) GetTenant() *models.Tenant {
	return df.tenant
}

// CreateUsers creates test users with different roles.
func (df *DataFactory) CreateUsers(ctx context.Context) error {
	roles := []string{"admin", "engineer", "support", "customer"}
	names := []string{"Admin", "Engineer", "Support", "Customer"}

	for i, role := range roles {
		now := time.Now()
		user := &models.User{
			BaseModel: models.BaseModel{
				ID:        uint(i + 2), // Start from ID 2 to avoid conflicts
				CreatedAt: now,
				UpdatedAt: now,
			},
			TenantID:     df.tenant.ID,
			Email:        fmt.Sprintf("%s@test.com", role),
			Username:     fmt.Sprintf("test_%s", role),
			PasswordHash: "hashed_password", // Will be set below
			FirstName:    names[i],
			LastName:     "User",
			IsActive:     true,
			Preferences:  `{"notifications": true, "theme": "light"}`,
		}

		// Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		user.PasswordHash = string(hashedPassword)

		if err := df.db.Create(user).Error; err != nil {
			return err
		}

		df.users[role] = user
	}

	return nil
}

// GetUser returns a user by role.
func (df *DataFactory) GetUser(role string) *models.User {
	return df.users[role]
}

// GetAllUsers returns all created users.
func (df *DataFactory) GetAllUsers() map[string]*models.User {
	return df.users
}

// CreateTickets creates test tickets with various statuses and priorities.
func (df *DataFactory) CreateTickets(ctx context.Context) error {
	statuses := []string{"open", "in_progress", "resolved", "closed"}
	priorities := []string{"low", "medium", "high", "critical"}
	severities := []string{"trivial", "minor", "major", "critical"}
	categories := []string{"bug", "feature", "question", "incident"}

	for i := 0; i < 20; i++ {
		now := time.Now()
		createdTime := now.Add(-time.Duration(i) * time.Hour)

		ticket := &models.Ticket{
			BaseModel: models.BaseModel{
				ID:        uint(i + 100), // Start from ID 100
				CreatedAt: createdTime,
				UpdatedAt: createdTime,
			},
			TenantID:       df.tenant.ID,
			TicketNumber:   fmt.Sprintf("TICKET-%04d", i+1),
			Title:          fmt.Sprintf("Test Ticket %d", i+1),
			Description:    fmt.Sprintf("This is test ticket number %d with various details", i+1),
			Status:         statuses[i%len(statuses)],
			Priority:       priorities[i%len(priorities)],
			Severity:       severities[i%len(severities)],
			Category:       categories[i%len(categories)],
			RequesterName:  "Test Customer",
			RequesterEmail: "customer@test.com",
		}

		// Assign tickets to engineers
		if i%2 == 0 && i > 0 {
			assignedTo := df.users["engineer"].ID
			ticket.AssignedTo = &assignedTo
		}

		if err := df.db.Create(ticket).Error; err != nil {
			return err
		}

		df.tickets = append(df.tickets, ticket)

		// Create some messages for tickets
		if i%3 == 0 {
			if err := df.createTicketMessages(ctx, ticket); err != nil {
				return err
			}
		}
	}

	return nil
}

// createTicketMessages creates messages for a ticket.
func (df *DataFactory) createTicketMessages(ctx context.Context, ticket *models.Ticket) error {
	messageCount := 1 + (int(ticket.ID) % 3) // 1-3 messages per ticket

	for i := 0; i < messageCount; i++ {
		now := time.Now()
		messageTime := now.Add(time.Duration(i) * time.Minute)

		message := &models.Message{
			BaseModel: models.BaseModel{
				ID:        uint(1000 + i), // Start from ID 1000
				CreatedAt: messageTime,
				UpdatedAt: messageTime,
			},
			TicketID:    ticket.ID,
			UserID:      df.users["customer"].ID,
			Content:     fmt.Sprintf("This is message %d for ticket %s", i+1, ticket.TicketNumber),
			ContentType: "text",
		}

		if err := df.db.Create(message).Error; err != nil {
			return err
		}
	}

	return nil
}

// GetTickets returns all created tickets.
func (df *DataFactory) GetTickets() []*models.Ticket {
	return df.tickets
}

// GetTicketByStatus returns tickets filtered by status.
func (df *DataFactory) GetTicketByStatus(status string) []*models.Ticket {
	var filtered []*models.Ticket
	for _, ticket := range df.tickets {
		if ticket.Status == status {
			filtered = append(filtered, ticket)
		}
	}
	return filtered
}

// CreateKnowledgeArticles creates test knowledge articles.
func (df *DataFactory) CreateKnowledgeArticles(ctx context.Context) error {
	categories := []string{"getting-started", "troubleshooting", "api", "features"}
	for i := 0; i < 10; i++ {
		now := time.Now()
		createdTime := now.Add(-time.Duration(i*24) * time.Hour)
		updatedTime := now.Add(-time.Duration(i*12) * time.Hour)

		article := &models.KnowledgeArticle{
			BaseModel: models.BaseModel{
				ID:        uint(i + 200), // Start from ID 200
				CreatedAt: createdTime,
				UpdatedAt: updatedTime,
			},
			TenantID:    df.tenant.ID,
			Title:       fmt.Sprintf("Knowledge Article %d", i+1),
			Slug:        fmt.Sprintf("knowledge-article-%d", i+1),
			Content:     fmt.Sprintf("This is the content of knowledge article %d. It contains useful information about the system.", i+1),
			Summary:     fmt.Sprintf("Summary of article %d", i+1),
			AuthorID:    df.users["engineer"].ID,
			Status:      "published",
			Visibility:  "public",
			AccessLevel: "all",
			Category:    categories[i%len(categories)],
			Tags:        fmt.Sprintf(`["tag%d", "tag%d"]`, i%5, (i+1)%5),
			Views:       i * 10,
		}

		if err := df.db.Create(article).Error; err != nil {
			return err
		}

		df.articles = append(df.articles, article)
	}

	return nil
}

// GetKnowledgeArticles returns all created knowledge articles.
func (df *DataFactory) GetKnowledgeArticles() []*models.KnowledgeArticle {
	return df.articles
}

// CreateLLMProviders creates test LLM providers.
func (df *DataFactory) CreateLLMProviders(ctx context.Context) error {
	providers := []struct {
		name         string
		model        string
		providerType string
	}{
		{"OpenAI GPT-4", "gpt-4", "openai"},
		{"Anthropic Claude", "claude-3-sonnet", "anthropic"},
		{"Local Ollama", "llama2", "ollama"},
	}

	for i, p := range providers {
		now := time.Now()
		provider := &models.LLMProvider{
			BaseModel: models.BaseModel{
				ID:        uint(i + 300), // Start from ID 300
				CreatedAt: now,
				UpdatedAt: now,
			},
			TenantID:      df.tenant.ID,
			Name:          p.name,
			ProviderType:  p.providerType,
			APIEndpoint:   fmt.Sprintf("https://api.%s.com/v1", p.providerType),
			APIKey:        fmt.Sprintf("test-api-key-for-%s", p.providerType),
			Model:         p.model,
			MaxTokens:     4096,
			Temperature:   0.7,
			TaskTypes:     `["chat", "summarization"]`,
			IsDefault:     i == 0, // First one is default
			IsEnabled:     true,
			Configuration: `{"timeout": 30, "retries": 3}`,
		}

		if err := df.db.Create(provider).Error; err != nil {
			return err
		}

		df.providers = append(df.providers, provider)
	}

	return nil
}

// GetLLMProviders returns all created LLM providers.
func (df *DataFactory) GetLLMProviders() []*models.LLMProvider {
	return df.providers
}

// CreateAuditLog creates an audit log entry.
func (df *DataFactory) CreateAuditLog(ctx context.Context, action, resourceType string, resourceID uint, details map[string]interface{}) error {
	detailsJSON := fmt.Sprintf(`{"action": "%s", "details": %v}`, action, details)

	now := time.Now()
	auditLog := &models.AuditLog{
		BaseModel: models.BaseModel{
			ID:        uint(400), // Fixed ID for simplicity
			CreatedAt: now,
			UpdatedAt: now,
		},
		TenantID:     df.tenant.ID,
		UserID:       df.users["admin"].ID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ResourceName: fmt.Sprintf("%s-%d", resourceType, resourceID),
		IPAddress:    "127.0.0.1",
		UserAgent:    "test-agent/1.0",
		Changes:      detailsJSON,
		RequestID:    fmt.Sprintf("req-%d", now.Unix()),
		Hash:         "audit-hash",
	}

	return df.db.Create(auditLog).Error
}

// CreatePermissions creates test permissions.
func (df *DataFactory) CreatePermissions(ctx context.Context) error {
	permissions := []struct {
		code        string
		description string
		category    string
	}{
		{"tickets:read", "Read tickets", "tickets"},
		{"tickets:write", "Create and update tickets", "tickets"},
		{"tickets:delete", "Delete tickets", "tickets"},
		{"users:read", "Read users", "users"},
		{"users:write", "Create and update users", "users"},
		{"users:delete", "Delete users", "users"},
		{"admin:all", "Full administrative access", "admin"},
	}

	for i, p := range permissions {
		now := time.Now()
		permission := &models.Permission{
			BaseModel: models.BaseModel{
				ID:        uint(i + 500), // Start from ID 500
				CreatedAt: now,
				UpdatedAt: now,
			},
			Code:        p.code,
			Name:        p.description,
			Description: p.description,
			Category:    p.category,
			IsSystem:    p.code == "admin:all", // Admin permission is system
		}

		if err := df.db.Create(permission).Error; err != nil {
			return err
		}
	}

	return nil
}

// ResetTestData clears all test data.
func (df *DataFactory) ResetTestData(ctx context.Context) error {
	// Delete in order of dependencies
	tables := []interface{}{
		&models.AuditLog{},
		&models.Message{},
		&models.Attachment{},
		&models.KnowledgeArticle{},
		&models.Ticket{},
		&models.UserPermission{},
		&models.UserRole{},
		&models.RolePermission{},
		&models.Role{},
		&models.Permission{},
		&models.User{},
		&models.LLMProvider{},
		&models.Tenant{},
	}

	for _, table := range tables {
		if err := df.db.Where("tenant_id = ?", df.tenant.ID).Delete(table).Error; err != nil {
			return err
		}
	}

	// Reset internal state
	df.tenant = nil
	df.users = make(map[string]*models.User)
	df.tickets = nil
	df.articles = nil
	df.providers = nil

	return nil
}

// MockFactory creates mock objects for testing.
type MockFactory struct{}

// NewMockFactory creates a new mock factory.
func NewMockFactory() *MockFactory {
	return &MockFactory{}
}

// CreateMockTicketService creates a mock ticket service.
func (mf *MockFactory) CreateMockTicketService() *MockTicketService {
	return &MockTicketService{
		tickets: make(map[string]*models.Ticket),
	}
}

// MockTicketService is a mock implementation of ticket service.
type MockTicketService struct {
	tickets map[string]*models.Ticket
}

func (m *MockTicketService) CreateTicket(ctx context.Context, ticket *models.Ticket) error {
	id := fmt.Sprintf("%d", ticket.ID)
	m.tickets[id] = ticket
	return nil
}

func (m *MockTicketService) GetTicket(ctx context.Context, id string) (*models.Ticket, error) {
	ticket, exists := m.tickets[id]
	if !exists {
		return nil, fmt.Errorf("ticket not found")
	}
	return ticket, nil
}

func (m *MockTicketService) UpdateTicket(ctx context.Context, ticket *models.Ticket) error {
	id := fmt.Sprintf("%d", ticket.ID)
	m.tickets[id] = ticket
	return nil
}

func (m *MockTicketService) DeleteTicket(ctx context.Context, id string) error {
	delete(m.tickets, id)
	return nil
}

func (m *MockTicketService) ListTickets(ctx context.Context, filter map[string]interface{}) ([]*models.Ticket, error) {
	var tickets []*models.Ticket
	for _, ticket := range m.tickets {
		tickets = append(tickets, ticket)
	}
	return tickets, nil
}

// CreateMockUserService creates a mock user service.
func (mf *MockFactory) CreateMockUserService() *MockUserService {
	return &MockUserService{
		users: make(map[string]*models.User),
	}
}

// MockUserService is a mock implementation of user service.
type MockUserService struct {
	users map[string]*models.User
}

func (m *MockUserService) CreateUser(ctx context.Context, user *models.User) error {
	id := fmt.Sprintf("%d", user.ID)
	m.users[id] = user
	return nil
}

func (m *MockUserService) GetUser(ctx context.Context, id string) (*models.User, error) {
	user, exists := m.users[id]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

func (m *MockUserService) UpdateUser(ctx context.Context, user *models.User) error {
	id := fmt.Sprintf("%d", user.ID)
	m.users[id] = user
	return nil
}

func (m *MockUserService) DeleteUser(ctx context.Context, id string) error {
	delete(m.users, id)
	return nil
}

func (m *MockUserService) ListUsers(ctx context.Context, filter map[string]interface{}) ([]*models.User, error) {
	var users []*models.User
	for _, user := range m.users {
		users = append(users, user)
	}
	return users, nil
}
