package fixtures

import (
	"time"

	"github.com/google/uuid"
)

// TestTenant represents a test tenant fixture
type TestTenant struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Domain    string    `json:"domain"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TestUser represents a test user fixture
type TestUser struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TestTicket represents a test ticket fixture
type TestTicket struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Number      string    `json:"number"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Priority    string    `json:"priority"`
	Severity    string    `json:"severity"`
	Category    string    `json:"category"`
	CreatedBy   string    `json:"created_by"`
	AssignedTo  *string   `json:"assigned_to"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewTestTenant creates a new test tenant fixture
func NewTestTenant() TestTenant {
	now := time.Now()
	return TestTenant{
		ID:        uuid.New().String(),
		Name:      "Test Tenant",
		Domain:    "test.example.com",
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewTestUser creates a new test user fixture
func NewTestUser(tenantID string) TestUser {
	now := time.Now()
	return TestUser{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		Email:     "test@example.com",
		Name:      "Test User",
		Role:      "admin",
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewTestTicket creates a new test ticket fixture
func NewTestTicket(tenantID, createdBy string) TestTicket {
	now := time.Now()
	return TestTicket{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Number:      "TICKET-001",
		Title:       "Test Ticket",
		Description: "This is a test ticket for testing purposes",
		Status:      "open",
		Priority:    "medium",
		Severity:    "low",
		Category:    "general",
		CreatedBy:   createdBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// GetSampleTenants returns a slice of sample tenant fixtures
func GetSampleTenants() []TestTenant {
	tenant1 := NewTestTenant()
	tenant1.Name = "Acme Corporation"
	tenant1.Domain = "acme.example.com"

	tenant2 := NewTestTenant()
	tenant2.Name = "Globex Inc"
	tenant2.Domain = "globex.example.com"

	return []TestTenant{tenant1, tenant2}
}

// GetSampleUsers returns a slice of sample user fixtures
func GetSampleUsers(tenantID string) []TestUser {
	users := make([]TestUser, 3)

	users[0] = NewTestUser(tenantID)
	users[0].Email = "admin@example.com"
	users[0].Name = "Admin User"
	users[0].Role = "admin"

	users[1] = NewTestUser(tenantID)
	users[1].Email = "engineer@example.com"
	users[1].Name = "Engineer User"
	users[1].Role = "engineer"

	users[2] = NewTestUser(tenantID)
	users[2].Email = "customer@example.com"
	users[2].Name = "Customer User"
	users[2].Role = "customer"

	return users
}

// GetSampleTickets returns a slice of sample ticket fixtures
func GetSampleTickets(tenantID, createdBy string) []TestTicket {
	tickets := make([]TestTicket, 3)

	tickets[0] = NewTestTicket(tenantID, createdBy)
	tickets[0].Title = "Login Issue"
	tickets[0].Description = "User cannot login to the system"
	tickets[0].Priority = "high"
	tickets[0].Severity = "medium"

	tickets[1] = NewTestTicket(tenantID, createdBy)
	tickets[1].Title = "Performance Problem"
	tickets[1].Description = "System is running slowly"
	tickets[1].Priority = "medium"
	tickets[1].Severity = "low"

	tickets[2] = NewTestTicket(tenantID, createdBy)
	tickets[2].Title = "Feature Request"
	tickets[2].Description = "Add new feature to improve user experience"
	tickets[2].Priority = "low"
	tickets[2].Severity = "low"

	return tickets
}
