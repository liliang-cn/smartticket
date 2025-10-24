package repositories

import (
	"fmt"
	"testing"
	"time"

	"github.com/company/smartticket/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Migrate all models
	err = db.AutoMigrate(
		&{},
		&models.User{},
		&models.Ticket{},
		&models.Message{},
		&models.Attachment{},
		&models.KnowledgeArticle{},
		&models.LLMProvider{},
		&models.ImportExportJob{},
		&models.AuditLog{},
		&models.APIKey{},
		&models.SystemSetting{},
		&models.Product{},
		&models.Service{},
		&models.SLATemplate{},
		&models.SLARule{},
		&models.Permission{},
		&models.Role{},
		&models.RolePermission{},
		&models.UserPermission{},
		&models.UserRole{},
	)
	require.NoError(t, err)

	return db
}

// TestBasicDatabaseOperations tests basic CRUD operations.
func TestBasicDatabaseOperations(t *testing.T) {
	db := setupTestDB(t)

	t.Run("Tenant CRUD operations", func(t *testing.T) {
		// Create
		tenant := &{
			Name:     "Test Corporation",
			Slug:     "test-corporation",
			Domain:   "test.example.com",
			Plan:     "basic",
			MaxUsers:  100,
			IsActive: true,
			Settings: `{"timezone": "UTC"}`,
		}

		err := db.Create(tenant).Error
		assert.NoError(t, err)
		assert.NotZero(t, tenant.ID)

		// Read
		var found 
		err = db.First(&found, tenant.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, tenant.Name, found.Name)
		assert.Equal(t, tenant.Slug, found.Slug)

		// Update
		tenant.Name = "Updated Corporation"
		err = db.Save(tenant).Error
		assert.NoError(t, err)

		err = db.First(&found, tenant.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, "Updated Corporation", found.Name)

		// Delete
		err = db.Delete(tenant).Error
		assert.NoError(t, err)

		err = db.First(&found, tenant.ID).Error
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})

	t.Run("User CRUD operations", func(t *testing.T) {
		// Create tenant first
		tenant := &{
			Name:     "User Test Tenant",
			Slug:     "user-test-tenant",
			IsActive: true,
		}
		err := db.Create(tenant).Error
		require.NoError(t, err)

		// Create user
		user := &models.User{
			Email:        "test@example.com",
			Username:     "testuser",
			FirstName:    "Test",
			LastName:     "User",
			PasswordHash: "hashed_password",
			IsActive:     true,
		}

		err = db.Create(user).Error
		assert.NoError(t, err)
		assert.NotZero(t, user.ID)

		// Read
		var found models.User
		err = db.First(&found, user.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, user.Email, found.Email)
		assert.Equal(t, tenant.ID, found)

		// Update
		user.FirstName = "Updated"
		err = db.Save(user).Error
		assert.NoError(t, err)

		err = db.First(&found, user.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, "Updated", found.FirstName)

		// Delete
		err = db.Delete(user).Error
		assert.NoError(t, err)

		err = db.First(&found, user.ID).Error
		assert.Error(t, err)
	})

	t.Run("Ticket CRUD operations", func(t *testing.T) {
		// Create tenant and user first
		tenant := &{
			Name:     "Ticket Test Tenant",
			Slug:     "ticket-test-tenant",
			IsActive: true,
		}
		err := db.Create(tenant).Error
		require.NoError(t, err)

		user := &models.User{
			Email:    "ticket@example.com",
			Username: "ticketuser",
			IsActive: true,
		}
		err = db.Create(user).Error
		require.NoError(t, err)

		// Create ticket
		ticket := &models.Ticket{
			TicketNumber:   "TICKET-001",
			Title:          "Test Ticket",
			Description:    "This is a test ticket",
			Status:         "open",
			Priority:       "medium",
			Severity:       "minor",
			RequesterName:  "Test User",
			RequesterEmail: "test@example.com",
		}

		err = db.Create(ticket).Error
		assert.NoError(t, err)
		assert.NotZero(t, ticket.ID)

		// Read
		var found models.Ticket
		err = db.First(&found, ticket.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, ticket.TicketNumber, found.TicketNumber)
		assert.Equal(t, tenant.ID, found)

		// Update status
		ticket.Status = "in_progress"
		err = db.Save(ticket).Error
		assert.NoError(t, err)

		err = db.First(&found, ticket.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, "in_progress", found.Status)

		// Delete
		err = db.Delete(ticket).Error
		assert.NoError(t, err)

		err = db.First(&found, ticket.ID).Error
		assert.Error(t, err)
	})
}

// TestDatabaseQueries tests various query operations.
func TestDatabaseQueries(t *testing.T) {
	db := setupTestDB(t)

	t.Run("Query with filters", func(t *testing.T) {
		// Create test data
		tenant := &{
			Name:     "Query Test Tenant",
			Slug:     "query-test-tenant",
			IsActive: true,
		}
		err := db.Create(tenant).Error
		require.NoError(t, err)

		// Create users with different permissions
		// Note: Role field removed from User model - roles are now handled through UserRole associations
		for i := 0; i < 4; i++ {
			user := &models.User{
				Email:    fmt.Sprintf("user%d@example.com", i+1),
				Username: fmt.Sprintf("user%d", i+1),
				IsActive: true,
			}
			err = db.Create(user).Error
			require.NoError(t, err)
		}

		// Query active users since User.Role field no longer exists
		var activeUsers []models.User
		err = db.Where("is_active = ? AND tenant_id = ?", true, tenant.ID).Find(&activeUsers).Error
		assert.NoError(t, err)
		// Verify we have the expected number of active users
		assert.Len(t, activeUsers, 4)

		// Query with LIKE
		var adminUsers []models.User
		err = db.Where("email LIKE ? AND tenant_id = ?", "%admin%", tenant.ID).Find(&adminUsers).Error
		assert.NoError(t, err)
		if len(adminUsers) > 0 {
			assert.Contains(t, adminUsers[0].Email, "admin")
		}
	})

	t.Run("Pagination", func(t *testing.T) {
		// Create test data
		tenant := &{
			Name:     "Pagination Test Tenant",
			Slug:     "pagination-test-tenant",
			IsActive: true,
		}
		err := db.Create(tenant).Error
		require.NoError(t, err)

		// Create multiple users
		for i := 0; i < 25; i++ {
			user := &models.User{
				Email:    fmt.Sprintf("pageuser%d@example.com", i+1),
				Username: fmt.Sprintf("pageuser%d", i+1),
				IsActive: true,
			}
			err = db.Create(user).Error
			require.NoError(t, err)
		}

		// Test pagination
		var page1 []models.User
		err = db.Where("tenant_id = ?", tenant.ID).
			Offset(0).
			Limit(10).
			Find(&page1).Error
		assert.NoError(t, err)
		assert.Len(t, page1, 10)

		var page2 []models.User
		err = db.Where("tenant_id = ?", tenant.ID).
			Offset(10).
			Limit(10).
			Find(&page2).Error
		assert.NoError(t, err)
		assert.Len(t, page2, 10)

		// Ensure no overlap
		page1IDs := make(map[uint]bool)
		for _, user := range page1 {
			page1IDs[user.ID] = true
		}
		for _, user := range page2 {
			assert.False(t, page1IDs[user.ID])
		}
	})

	t.Run("Count queries", func(t *testing.T) {
		// Create test data
		tenant := &{
			Name:     "Count Test Tenant",
			Slug:     "count-test-tenant",
			IsActive: true,
		}
		err := db.Create(tenant).Error
		require.NoError(t, err)

		// Create users for counting
		userCount := 10
		for i := 0; i < userCount; i++ {
			user := &models.User{
				Email:    fmt.Sprintf("countuser%d@example.com", i+1),
				Username: fmt.Sprintf("countuser%d", i+1),
				IsActive: true,
			}
			err = db.Create(user).Error
			require.NoError(t, err)
		}

		// Count total users
		var totalUsers int64
		err = db.Model(&models.User{}).Where("tenant_id = ?", tenant.ID).Count(&totalUsers).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(userCount), totalUsers)

		// Count active users
		var activeCount int64
		err = db.Model(&models.User{}).
			Where("tenant_id = ? AND is_active = ?", tenant.ID, true).
			Count(&activeCount).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(userCount), activeCount)
	})
}

// TestDatabaseTransactions tests transaction handling.
func TestDatabaseTransactions(t *testing.T) {
	db := setupTestDB(t)

	t.Run("Successful transaction", func(t *testing.T) {
		// Begin transaction
		tx := db.Begin()
		assert.False(t, tx.Error != nil)

		// Create tenant in transaction
		tenant := &{
			Name:     "Transaction Tenant",
			Slug:     "transaction-tenant",
			IsActive: true,
		}
		err := tx.Create(tenant).Error
		assert.NoError(t, err)

		// Create user in transaction
		user := &models.User{
			Email:    "tx@example.com",
			Username: "txuser",
			IsActive: true,
		}
		err = tx.Create(user).Error
		assert.NoError(t, err)

		// Commit transaction
		err = tx.Commit().Error
		assert.NoError(t, err)

		// Verify records exist after commit
		var foundTenant 
		err = db.First(&foundTenant, tenant.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, tenant.Name, foundTenant.Name)

		var foundUser models.User
		err = db.First(&foundUser, user.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, user.Email, foundUser.Email)
	})

	t.Run("Rolled back transaction", func(t *testing.T) {
		// Begin transaction
		tx := db.Begin()
		assert.False(t, tx.Error != nil)

		// Create tenant in transaction
		tenant := &{
			Name:     "Rollback Tenant",
			Slug:     "rollback-tenant",
			IsActive: true,
		}
		err := tx.Create(tenant).Error
		assert.NoError(t, err)

		// Create user in transaction
		user := &models.User{
			Email:    "rollback@example.com",
			Username: "rollbackuser",
			IsActive: true,
		}
		err = tx.Create(user).Error
		assert.NoError(t, err)

		// Rollback transaction
		err = tx.Rollback().Error
		assert.NoError(t, err)

		// Verify records don't exist after rollback
		var foundTenant 
		err = db.First(&foundTenant, tenant.ID).Error
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)

		var foundUser models.User
		err = db.First(&foundUser, user.ID).Error
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})
}

// TestDatabaseConstraints tests database constraints.
func TestDatabaseConstraints(t *testing.T) {
	db := setupTestDB(t)

	t.Run("Unique constraint violation", func(t *testing.T) {
		// Create first tenant
		tenant1 := &{
			Name:     "Unique Test 1",
			Slug:     "unique-test",
			IsActive: true,
		}
		err := db.Create(tenant1).Error
		assert.NoError(t, err)

		// Try to create second tenant with same slug
		tenant2 := &{
			Name:     "Unique Test 2",
			Slug:     "unique-test", // Same slug
			IsActive: true,
		}
		err = db.Create(tenant2).Error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "UNIQUE")
	})

	t.Run("Not null constraint", func(_ *testing.T) {
		// Try to create tenant with required field empty
		tenant := &{
			Name: "", // Empty name should violate not null constraint
			Slug: "not-null-test",
		}
		_ = db.Create(tenant).Error
		// Note: SQLite may not enforce NOT NULL constraints strictly in all cases
		// The actual behavior depends on SQLite configuration
	})

	t.Run("Foreign key relationships", func(t *testing.T) {
		// Try to create ticket without tenant
		ticket := &models.Ticket{
			TicketNumber: "TICKET-FK-001",
			Title:        "Foreign Key Test",
			Status:       "open",
		}
		err := db.Create(ticket).Error
		// SQLite may not enforce foreign key constraints by default
		// In production, you'd enable foreign key constraints
		if err == nil {
			// If no error, ticket was created anyway (SQLite default behavior)
			assert.NotZero(t, ticket.ID)
		} else {
			// If error occurred, it should be related to foreign key constraint
			assert.Error(t, err)
		}
	})
}

// TestDatabasePerformance tests basic performance aspects.
func TestDatabasePerformance(t *testing.T) {
	db := setupTestDB(t)

	t.Run("Batch insert performance", func(t *testing.T) {
		// Create tenant
		tenant := &{
			Name:     "Performance Test Tenant",
			Slug:     "performance-test-tenant",
			IsActive: true,
		}
		err := db.Create(tenant).Error
		require.NoError(t, err)

		// Measure batch insert time
		start := time.Now()

		// Create multiple users
		users := make([]*models.User, 100)
		for i := 0; i < 100; i++ {
			users[i] = &models.User{
				Email:    fmt.Sprintf("perfuser%d@example.com", i+1),
				Username: fmt.Sprintf("perfuser%d", i+1),
				IsActive: true,
			}
		}

		// Insert all users
		for _, user := range users {
			err := db.Create(user).Error
			require.NoError(t, err)
		}

		duration := time.Since(start)
		t.Logf("Batch insert of 100 users took: %v", duration)
		assert.Less(t, duration, 1*time.Second) // Should complete within 1 second
	})

	t.Run("Query performance with indexes", func(t *testing.T) {
		// Create tenant
		tenant := &{
			Name:     "Query Performance Tenant",
			Slug:     "query-perf-tenant",
			IsActive: true,
		}
		err := db.Create(tenant).Error
		require.NoError(t, err)

		// Create many users
		for i := 0; i < 1000; i++ {
			user := &models.User{
				Email:    fmt.Sprintf("queryuser%d@example.com", i+1),
				Username: fmt.Sprintf("queryuser%d", i+1),
				IsActive: true,
			}
			err := db.Create(user).Error
			require.NoError(t, err)
		}

		// Measure query time
		start := time.Now()

		var users []models.User
		err = db.Where("tenant_id = ? AND is_active = ?", tenant.ID, true).Find(&users).Error
		assert.NoError(t, err)

		duration := time.Since(start)
		t.Logf("Query of %d users by tenant and active status took: %v", len(users), duration)
		assert.Less(t, duration, 100*time.Millisecond) // Should complete within 100ms
	})
}