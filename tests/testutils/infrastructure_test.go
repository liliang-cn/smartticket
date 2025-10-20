package testutils

import (
	"testing"

	"github.com/company/smartticket/internal/config"
	"github.com/company/smartticket/internal/database"
	"github.com/company/smartticket/tests/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestTestDatabase(t *testing.T) {
	td := NewTestDatabase(t)
	defer func() { _ = td.Close() }()

	// Test that database is created and healthy
	assert.NotNil(t, td)
	assert.True(t, td.IsHealthy())

	// Test database stats
	stats := td.Stats()
	assert.NotNil(t, stats)
	assert.Equal(t, 10, stats["max_open_connections"]) // Test config has 10 connections
}

func TestTestConfig(t *testing.T) {
	tc := NewTestConfig(t)
	defer func() { _ = tc.Close() }()

	// Test that config is created
	assert.NotNil(t, tc)
	assert.NotNil(t, tc.Config)
	assert.Equal(t, "test", tc.Environment)

	// Test server config
	assert.Equal(t, "localhost", tc.Server.Host)
	assert.Equal(t, 0, tc.Server.Port) // Random port for testing

	// Test database config
	assert.Equal(t, "sqlite", tc.Database.Type)
	assert.Equal(t, "silent", tc.Database.LogLevel)
}

func TestTestServer(t *testing.T) {
	ts := NewTestServer(t)
	defer func() { _ = ts.Close() }()

	// Test that server is created
	assert.NotNil(t, ts)
	assert.NotEmpty(t, ts.GetURL())
	assert.NotNil(t, ts.GetEngine())
	assert.NotNil(t, ts.GetConfig())
	assert.NotNil(t, ts.GetDatabase())

	// Test HTTP client
	client := NewHTTPClient(ts)
	assert.NotNil(t, client)

	// Test basic request
	resp, err := client.Get("/")
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	defer func() { _ = resp.Body.Close() }()
}

func TestFixtures(t *testing.T) {
	// Test tenant fixture
	tenant := fixtures.NewTestTenant()
	assert.NotEmpty(t, tenant.ID, "Tenant ID should not be empty")
	assert.Equal(t, "Test Tenant", tenant.Name)
	assert.Equal(t, "test.example.com", tenant.Domain)
	assert.Equal(t, "active", tenant.Status)

	// Test user fixture
	user := fixtures.NewTestUser(tenant.ID)
	assert.NotEmpty(t, user.ID, "User ID should not be empty")
	assert.Equal(t, tenant.ID, user.TenantID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "admin", user.Role)

	// Test ticket fixture
	ticket := fixtures.NewTestTicket(tenant.ID, user.ID)
	assert.NotEmpty(t, ticket.ID, "Ticket ID should not be empty")
	assert.Equal(t, tenant.ID, ticket.TenantID)
	assert.Equal(t, user.ID, ticket.CreatedBy)
	assert.Equal(t, "TICKET-001", ticket.Number)
	assert.Equal(t, "open", ticket.Status)

	// Test sample fixtures
	tenants := fixtures.GetSampleTenants()
	assert.Len(t, tenants, 2, "Should have 2 sample tenants")
	assert.NotEqual(t, tenants[0].Name, tenants[1].Name)

	users := fixtures.GetSampleUsers(tenant.ID)
	assert.Len(t, users, 3, "Should have 3 sample users")
	roles := make(map[string]bool)
	for _, u := range users {
		roles[u.Role] = true
	}
	assert.True(t, roles["admin"])
	assert.True(t, roles["engineer"])
	assert.True(t, roles["customer"])

	tickets := fixtures.GetSampleTickets(tenant.ID, user.ID)
	assert.Len(t, tickets, 3, "Should have 3 sample tickets")
	assert.NotEqual(t, tickets[0].Title, tickets[1].Title)
}

func TestTestRunner(t *testing.T) {
	runner := NewTestRunner(".")

	// Test dependency check
	result := runner.CheckTestDependencies()
	assert.NoError(t, result.Error)
	assert.Contains(t, result.Output, "All test dependencies")

	// Test that runner can execute basic commands
	result = runner.runCommand("go", []string{"version"})
	assert.NoError(t, result.Error)
	assert.NotEmpty(t, result.Output)
}

func TestWithTestDatabase(t *testing.T) {
	WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		assert.NotNil(t, db)
		assert.True(t, db.IsHealthy())
	})
}

func TestWithTestConfig(t *testing.T) {
	WithTestConfig(t, func(t *testing.T, cfg *config.Config) {
		assert.NotNil(t, cfg)
		assert.Equal(t, "test", cfg.Environment)
	})
}

func TestWithTestServer(t *testing.T) {
	WithTestServer(t, func(t *testing.T, ts *TestServer) {
		assert.NotNil(t, ts)
		assert.NotEmpty(t, ts.GetURL())

		client := NewHTTPClient(ts)
		resp, err := client.Get("/")
		assert.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
	})
}

func TestTestServerWaitForReady(t *testing.T) {
	ts := NewTestServer(t)
	defer func() { _ = ts.Close() }()

	// Test that server URL is accessible (basic connectivity)
	client := NewHTTPClient(ts)
	resp, err := client.Get("/")
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	defer func() { _ = resp.Body.Close() }()
}

func TestTestDatabaseIsolation(t *testing.T) {
	// Create two separate test databases
	td1 := NewTestDatabase(t)
	defer func() { _ = td1.Close() }()

	td2 := NewTestDatabase(t)
	defer func() { _ = td2.Close() }()

	// They should have different configs
	assert.NotEqual(t, td1.GetConfig().ConnectionURL, td2.GetConfig().ConnectionURL)

	// Both should be healthy
	assert.True(t, td1.IsHealthy())
	assert.True(t, td2.IsHealthy())
}

func TestTestConfigIsolation(t *testing.T) {
	// Create two separate test configs
	tc1 := NewTestConfig(t)
	defer func() { _ = tc1.Close() }()

	tc2 := NewTestConfig(t)
	defer func() { _ = tc2.Close() }()

	// They should have different temp directories
	assert.NotEqual(t, tc1.tempDir, tc2.tempDir)

	// Both should have same structure but different instances
	assert.Equal(t, tc1.Environment, tc2.Environment)
	assert.Equal(t, tc1.Server.Host, tc2.Server.Host)
}
