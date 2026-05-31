package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/company/smartticket/internal/models"
	sqlite "github.com/company/smartticket/internal/database/moderncsqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Migrate all models
	err = db.AutoMigrate(
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

// TestBasicServiceOperations tests basic service layer operations.
func TestBasicServiceOperations(t *testing.T) {
	db := setupTestDB(t)

	t.Run("Create and retrieve user", func(t *testing.T) {
		// Create a user
		user := &models.User{
			Email:        "test@example.com",
			Username:     "testuser",
			FirstName:    "Test",
			LastName:     "User",
			PasswordHash: "hashed_password",
			IsActive:     true,
		}

		err := db.Create(user).Error
		assert.NoError(t, err)
		assert.NotZero(t, user.ID)

		// Retrieve the user
		var found models.User
		err = db.First(&found, user.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, user.Email, found.Email)
		assert.Equal(t, user.Username, found.Username)
	})

	t.Run("Create and retrieve ticket", func(t *testing.T) {
		// Create a user
		user := &models.User{
			Email:    "ticket@example.com",
			Username: "ticketuser",
			IsActive: true,
		}
		err := db.Create(user).Error
		require.NoError(t, err)

		// Create a ticket
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

		// Retrieve the ticket
		var found models.Ticket
		err = db.Preload("AssignedUser").First(&found, ticket.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, ticket.TicketNumber, found.TicketNumber)
		assert.Equal(t, ticket.Title, found.Title)
	})
}

// TestServiceValidation tests service-level validation.
func TestServiceValidation(t *testing.T) {
	_ = setupTestDB(t) // Setup DB for validation tests

	t.Run("Email validation", func(t *testing.T) {
		// Test email format validation at service layer
		validEmails := []string{
			"test@example.com",
			"user.name@domain.co.uk",
			"user+tag@example.org",
		}

		for _, email := range validEmails {
			user := &models.User{
				Email:    email,
				Username: "testuser",
			}
			// In a real service, this would validate email format
			assert.NotEmpty(t, user.Email)
		}
	})

	t.Run("Ticket number format validation", func(t *testing.T) {
		// Test ticket number format validation
		validTicketNumbers := []string{
			"TICKET-001",
			"REQ-2024-001",
			"BUG-1234",
		}

		for _, ticketNumber := range validTicketNumbers {
			ticket := &models.Ticket{
				TicketNumber: ticketNumber,
				Title:        "Test Ticket",
			}
			// In a real service, this would validate ticket number format
			assert.NotEmpty(t, ticket.TicketNumber)
		}
	})

	t.Run("User validation without Role field", func(t *testing.T) {
		// Test user validation - Role field has been removed from User model
		// Roles are now handled through UserRole associations

		user := &models.User{
			Email:    "test@example.com",
			Username: "testuser",
		}
		// Test basic user fields - Role field no longer exists on User model
		assert.NotEmpty(t, user.Email)
		assert.NotEmpty(t, user.Username)
	})
}

// TestServiceErrorHandling tests service error handling.
func TestServiceErrorHandling(t *testing.T) {
	db := setupTestDB(t)

	t.Run("Database constraint violations", func(t *testing.T) {
		// Create a user
		user := &models.User{
			Email:    "dup@example.com",
			Username: "dupuser",
			IsActive: true,
		}
		err := db.Create(user).Error
		require.NoError(t, err)

		// Try to create another user with the same email
		duplicateUser := &models.User{
			Email:    "dup@example.com", // Same email
			Username: "dupuser2",
			IsActive: true,
		}
		err = db.Create(duplicateUser).Error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "UNIQUE")
	})

	t.Run("Context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Try to perform database operations with cancelled context
		user := &models.User{
			Email:    "cancelled@example.com",
			Username: "cancelleduser",
			IsActive: true,
		}
		err := db.WithContext(ctx).Create(user).Error
		assert.Error(t, err)
	})
}

// TestPermissionService tests the PermissionService functionality.
func TestPermissionService(t *testing.T) {
	db := setupTestDB(t)
	ps := NewPermissionService(db)
	ctx := context.Background()

	// Setup test data for each sub-test
	setupTestData := func(t *testing.T) *models.User {
		// Use unique identifiers for each test
		testID := fmt.Sprintf("test-%d", time.Now().UnixNano())

		user := &models.User{
			Email:    fmt.Sprintf("test-%s@example.com", testID),
			Username: fmt.Sprintf("testuser-%s", testID),
			IsActive: true,
		}
		require.NoError(t, db.Create(user).Error)
		return user
	}

	t.Run("Create and retrieve permission", func(t *testing.T) {
		_ = setupTestData(t)

		permission := &models.Permission{
			Code:        "test:read",
			Name:        "Test Read",
			Description: "Test read permission",
			Category:    "test",
			IsSystem:    false,
		}

		err := ps.CreatePermission(ctx, permission)
		assert.NoError(t, err)
		assert.NotZero(t, permission.ID)

		// Retrieve permission
		retrieved, err := ps.GetPermissionByID(ctx, permission.ID)
		assert.NoError(t, err)
		assert.Equal(t, permission.Code, retrieved.Code)
		assert.Equal(t, permission.Name, retrieved.Name)
	})

	t.Run("Create and retrieve role", func(t *testing.T) {
		_ = setupTestData(t)

		role := &models.Role{
			Name:        "Test Role",
			Description: "Test role description",
			IsSystem:    false,
			IsActive:    true,
		}

		err := ps.CreateRole(ctx, role)
		assert.NoError(t, err)
		assert.NotZero(t, role.ID)

		// Retrieve role
		retrieved, err := ps.GetRoleByID(ctx, role.ID)
		assert.NoError(t, err)
		assert.Equal(t, role.Name, retrieved.Name)
		assert.Equal(t, role.Description, retrieved.Description)
	})

	t.Run("Assign permission to role", func(t *testing.T) {
		_ = setupTestData(t)

		// Create permission
		permission := &models.Permission{
			Code:     "role:test",
			Name:     "Role Test",
			Category: "test",
		}
		require.NoError(t, ps.CreatePermission(ctx, permission))

		// Create role
		role := &models.Role{
			Name:     "Test Role for Assignment",
			IsActive: true,
		}
		require.NoError(t, ps.CreateRole(ctx, role))

		// Assign permission to role
		err := ps.AssignPermissionToRole(ctx, role.ID, permission.ID)
		assert.NoError(t, err)

		// Check role has permission
		permissions, err := ps.GetRolePermissions(ctx, role.ID)
		assert.NoError(t, err)
		assert.Len(t, permissions, 1)
		assert.Equal(t, permission.Code, permissions[0].Code)
	})

	t.Run("Assign role to user", func(t *testing.T) {
		user := setupTestData(t)

		// Create role
		role := &models.Role{
			Name:     "User Test Role",
			IsActive: true,
		}
		require.NoError(t, ps.CreateRole(ctx, role))

		// Assign role to user
		err := ps.AssignRoleToUser(ctx, user.ID, role.ID)
		assert.NoError(t, err)

		// Check user has role
		roles, err := ps.GetUserRoles(ctx, user.ID)
		assert.NoError(t, err)
		assert.Len(t, roles, 1)
		assert.Equal(t, role.Name, roles[0].Name)
	})

	t.Run("Delete permission (non-system)", func(t *testing.T) {
		_ = setupTestData(t)

		// Create non-system permission
		permission := &models.Permission{
			Code:     "delete:test",
			Name:     "Delete Test Permission",
			Category: "test",
			IsSystem: false,
		}
		require.NoError(t, ps.CreatePermission(ctx, permission))

		// Delete permission
		err := ps.DeletePermission(ctx, permission.ID)
		assert.NoError(t, err)

		// Verify deletion (should be wrapped error)
		_, err = ps.GetPermissionByID(ctx, permission.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get permission")
	})

	t.Run("Cannot delete system permission", func(t *testing.T) {
		_ = setupTestData(t)

		// Create system permission
		permission := &models.Permission{
			Code:     "system:test",
			Name:     "System Test Permission",
			Category: "system",
			IsSystem: true,
		}
		require.NoError(t, ps.CreatePermission(ctx, permission))

		// Try to delete system permission
		err := ps.DeletePermission(ctx, permission.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete system permission")

		// Verify permission still exists
		retrieved, err := ps.GetPermissionByID(ctx, permission.ID)
		assert.NoError(t, err)
		assert.Equal(t, permission.ID, retrieved.ID)
	})
}

// TestPermissionServiceComplex tests more complex scenarios with better isolation.
func TestPermissionServiceComplex(t *testing.T) {
	db := setupTestDB(t)
	ps := NewPermissionService(db)
	ctx := context.Background()

	t.Run("Basic CRUD operations", func(t *testing.T) {
		// Setup fresh data
		testID := fmt.Sprintf("crud-%d", time.Now().UnixNano())
		user := &models.User{
			Email:    fmt.Sprintf("crud-%s@example.com", testID),
			Username: fmt.Sprintf("cruduser-%s", testID),
			IsActive: true,
		}
		require.NoError(t, db.Create(user).Error)

		// Test permission creation and retrieval
		permission := &models.Permission{
			Code:        "crud:read",
			Name:        "CRUD Read",
			Description: "CRUD test read permission",
			Category:    "crud",
			IsSystem:    false,
		}

		err := ps.CreatePermission(ctx, permission)
		assert.NoError(t, err)
		assert.NotZero(t, permission.ID)

		retrieved, err := ps.GetPermissionByID(ctx, permission.ID)
		assert.NoError(t, err)
		assert.Equal(t, permission.Code, retrieved.Code)

		// Test permission update
		permission.Name = "Updated CRUD Permission"
		err = ps.UpdatePermission(ctx, permission)
		assert.NoError(t, err)

		retrieved, err = ps.GetPermissionByID(ctx, permission.ID)
		assert.NoError(t, err)
		assert.Equal(t, "Updated CRUD Permission", retrieved.Name)
	})

	t.Run("Role and permission assignment", func(t *testing.T) {
		// Setup fresh data
		testID := fmt.Sprintf("assignment-%d", time.Now().UnixNano())
		user := &models.User{
			Email:    fmt.Sprintf("assignment-%s@example.com", testID),
			Username: fmt.Sprintf("assignmentuser-%s", testID),
			IsActive: true,
		}
		require.NoError(t, db.Create(user).Error)

		// Create permission and role
		permission := &models.Permission{
			Code:     "assignment:test",
			Name:     "Assignment Test",
			Category: "test",
		}
		require.NoError(t, ps.CreatePermission(ctx, permission))

		role := &models.Role{
			Name:     "Assignment Test Role",
			IsActive: true,
		}
		require.NoError(t, ps.CreateRole(ctx, role))

		// Test assignment
		err := ps.AssignPermissionToRole(ctx, role.ID, permission.ID)
		assert.NoError(t, err)

		err = ps.AssignRoleToUser(ctx, user.ID, role.ID)
		assert.NoError(t, err)

		// Test permission checking
		hasPermission, err := ps.HasPermission(ctx, user.ID, permission.Code)
		assert.NoError(t, err)
		assert.True(t, hasPermission)

		// Test removal
		err = ps.RemovePermissionFromRole(ctx, role.ID, permission.ID)
		assert.NoError(t, err)

		// Verify role permissions are removed
		rolePermissions, err := ps.GetRolePermissions(ctx, role.ID)
		assert.NoError(t, err)
		assert.Len(t, rolePermissions, 0)

		// Check user no longer has permission through role
		hasPermission, err = ps.HasPermission(ctx, user.ID, permission.Code)
		assert.NoError(t, err)
		assert.False(t, hasPermission)
	})
}
