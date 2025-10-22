package ticket

import (
	"fmt"
	"testing"
	"time"

	"github.com/company/smartticket/internal/database"
	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/sla"
	"github.com/company/smartticket/tests/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTicketService_CreateTicket(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		// Setup
		slaCalc := sla.NewCalculator(db.DB)
		service := NewService(db.DB, slaCalc)

		// Create test tenant and user
		tenant := createTestTenant(t, db)
		user := createTestUser(t, db, tenant.ID)

		// Test data
		req := &CreateTicketRequest{
			Title:          "Test Ticket",
			Description:    "This is a test ticket description",
			Priority:       "medium",
			Severity:       "minor",
			Category:       "technical",
			RequesterName:  "John Doe",
			RequesterEmail: "john@example.com",
		}

		// Execute
		result, err := service.CreateTicket(tenant.ID, user.ID, req)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, req.Title, result.Title)
		assert.Equal(t, req.Description, result.Description)
		assert.Equal(t, "open", result.Status)
		assert.NotZero(t, result.ID)
		assert.NotZero(t, result.CreatedAt)
		assert.NotEmpty(t, result.TicketNumber)
	})
}

func TestTicketService_GetTicket(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		// Setup
		slaCalc := sla.NewCalculator(db.DB)
		service := NewService(db.DB, slaCalc)

		tenant := createTestTenant(t, db)
		user := createTestUser(t, db, tenant.ID)

		// Create a test ticket
		ticket := createTestTicket(t, db, tenant.ID, user.ID)

		// Execute
		result, err := service.GetTicket(tenant.ID, ticket.ID)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, ticket.ID, result.ID)
		assert.Equal(t, ticket.Title, result.Title)
		// Note: TicketResponse doesn't include TenantID directly, but we can verify the ticket was created for the right tenant
	})
}

func TestTicketService_GetTicket_NotFound(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		// Setup
		slaCalc := sla.NewCalculator(db.DB)
		service := NewService(db.DB, slaCalc)

		tenant := createTestTenant(t, db)

		// Execute
		result, err := service.GetTicket(tenant.ID, 999999)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestTicketService_ListTickets(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		// Setup
		slaCalc := sla.NewCalculator(db.DB)
		service := NewService(db.DB, slaCalc)

		tenant := createTestTenant(t, db)
		user := createTestUser(t, db, tenant.ID)

		// Create multiple test tickets
		_ = createTestTicket(t, db, tenant.ID, user.ID)
		_ = createTestTicket(t, db, tenant.ID, user.ID)
		_ = createTestTicket(t, db, tenant.ID, user.ID) // Third ticket

		// Execute with filters map
		filters := map[string]interface{}{}
		result, err := service.ListTickets(tenant.ID, 1, 20, filters)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Data, 3) // All tickets should be returned
		assert.Equal(t, int64(3), result.Total)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 20, result.PageSize)
	})
}

func TestTicketService_UpdateTicket(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		// Setup
		slaCalc := sla.NewCalculator(db.DB)
		service := NewService(db.DB, slaCalc)

		tenant := createTestTenant(t, db)
		user := createTestUser(t, db, tenant.ID)
		ticket := createTestTicket(t, db, tenant.ID, user.ID)

		// Test data
		req := &UpdateTicketRequest{
			Title:    "Updated Ticket Title",
			Status:   "in_progress",
			Priority: "high",
		}

		// Execute
		result, err := service.UpdateTicket(tenant.ID, ticket.ID, user.ID, req)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, req.Title, result.Title)
		assert.Equal(t, req.Status, result.Status)
		assert.Equal(t, req.Priority, result.Priority)
		// Note: UpdatedAt comparison not needed as it's always updated
	})
}

func TestTicketService_AssignTicket(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		// Setup
		slaCalc := sla.NewCalculator(db.DB)
		service := NewService(db.DB, slaCalc)

		tenant := createTestTenant(t, db)
		user1 := createTestUser(t, db, tenant.ID)
		user2 := createTestUser(t, db, tenant.ID)
		ticket := createTestTicket(t, db, tenant.ID, user1.ID)

		// Execute
		err := service.AssignTicket(tenant.ID, ticket.ID, user2.ID)

		// Assert
		require.NoError(t, err)

		// Verify the assignment by getting the ticket
		updatedTicket, err := service.GetTicket(tenant.ID, ticket.ID)
		require.NoError(t, err)
		assert.NotNil(t, updatedTicket)
		require.NotNil(t, updatedTicket.AssignedTo)
		assert.Equal(t, user2.ID, *updatedTicket.AssignedTo)
	})
}

func TestTicketService_DeleteTicket(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		// Setup
		slaCalc := sla.NewCalculator(db.DB)
		service := NewService(db.DB, slaCalc)

		tenant := createTestTenant(t, db)
		user := createTestUser(t, db, tenant.ID)
		ticket := createTestTicket(t, db, tenant.ID, user.ID)

		// Execute
		err := service.DeleteTicket(tenant.ID, ticket.ID)

		// Assert
		require.NoError(t, err)

		// Verify ticket is soft deleted
		var deletedTicket models.Ticket
		err = db.DB.Unscoped().First(&deletedTicket, ticket.ID).Error
		require.NoError(t, err)
		assert.NotNil(t, deletedTicket.DeletedAt)
	})
}

func TestTicketService_GetTicketStats(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		// Setup
		slaCalc := sla.NewCalculator(db.DB)
		service := NewService(db.DB, slaCalc)

		tenant := createTestTenant(t, db)
		user := createTestUser(t, db, tenant.ID)

		// Create tickets with different statuses
		createTestTicketWithStatus(t, db, tenant.ID, user.ID, "open")
		createTestTicketWithStatus(t, db, tenant.ID, user.ID, "in_progress")
		createTestTicketWithStatus(t, db, tenant.ID, user.ID, "resolved")
		createTestTicketWithStatus(t, db, tenant.ID, user.ID, "closed")

		// Execute
		stats, err := service.GetTicketStats(tenant.ID)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, stats)
		// Stats return as map[string]interface{}, check correct keys
		assert.Equal(t, int64(4), stats["total_tickets"])
		assert.Equal(t, int64(1), stats["open_tickets"])
		assert.Equal(t, int64(1), stats["in_progress_tickets"])
		assert.Equal(t, int64(1), stats["resolved_tickets"])
		assert.Equal(t, int64(1), stats["closed_tickets"])
	})
}

// Helper functions for creating test data

func createTestTenant(t *testing.T, db *database.Database) *models.Tenant {
	// Generate unique tenant values using timestamp
	timestamp := time.Now().UnixNano()
	tenant := &models.Tenant{
		Name:     fmt.Sprintf("Test Tenant %d", timestamp),
		Slug:     fmt.Sprintf("test-tenant-%d", timestamp),
		Domain:   fmt.Sprintf("test%d.example.com", timestamp),
		Plan:     "basic",
		IsActive: true,
	}

	err := db.DB.Create(tenant).Error
	require.NoError(t, err)

	return tenant
}

func createTestUser(t *testing.T, db *database.Database, tenantID uint) *models.User {
	// Generate unique email using timestamp
	timestamp := time.Now().UnixNano()
	user := &models.User{
		TenantID:  tenantID,
		Email:     fmt.Sprintf("test-%d@example.com", timestamp),
		Username:  fmt.Sprintf("testuser-%d", timestamp),
		FirstName: "Test",
		LastName:  "User",
		IsActive:  true,
	}

	err := db.DB.Create(user).Error
	require.NoError(t, err)

	return user
}

func createTestTicket(t *testing.T, db *database.Database, tenantID, userID uint) *models.Ticket {
	ticket := &models.Ticket{
		TenantID:       tenantID,
		TicketNumber:   generateTicketNumber(tenantID),
		Title:          "Test Ticket",
		Description:    "This is a test ticket",
		Status:         "open",
		Priority:       "medium",
		Severity:       "minor",
		Category:       "technical",
		RequesterName:  "John Doe",
		RequesterEmail: "john@example.com",
		IsDeleted:      false,
	}

	err := db.DB.Create(ticket).Error
	require.NoError(t, err)

	return ticket
}

func createTestTicketWithStatus(t *testing.T, db *database.Database, tenantID, userID uint, status string) *models.Ticket {
	ticket := &models.Ticket{
		TenantID:       tenantID,
		TicketNumber:   generateTicketNumber(tenantID),
		Title:          "Test Ticket",
		Description:    "This is a test ticket",
		Status:         status,
		Priority:       "medium",
		Severity:       "minor",
		Category:       "technical",
		RequesterName:  "John Doe",
		RequesterEmail: "john@example.com",
		IsDeleted:      false,
	}

	if status == "resolved" || status == "closed" {
		now := time.Now()
		ticket.ResolvedAt = &now
		ticket.ResolutionTime = &now
	}

	err := db.DB.Create(ticket).Error
	require.NoError(t, err)

	return ticket
}

func generateTicketNumber(tenantID uint) string {
	// Generate unique ticket number using timestamp
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("TK-%d-%d", tenantID, timestamp%100000)
}
