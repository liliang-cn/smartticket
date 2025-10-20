package user

import (
	"fmt"
	"testing"
	"time"

	"github.com/company/smartticket/internal/auth"
	"github.com/company/smartticket/internal/database"
	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/tests/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserService_CreateUser(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		// Setup
		authRepo := auth.NewRepository(db.DB)
		authService := auth.NewService(db.DB, "test-secret", time.Hour, time.Hour*24, "test-issuer")
		service := NewService(db.DB, authRepo, authService)

		tenant := createTestTenant(t, db)

		// Test data
		req := &CreateUserRequest{
			Email:     "newuser@example.com",
			Username:  "newuser",
			FirstName: "New",
			LastName:  "User",
			Role:      "engineer",
			Password:  "Password123!",
		}

		// Execute
		result, err := service.CreateUser(tenant.ID, req)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, req.Email, result.Email)
		assert.Equal(t, req.FirstName, result.FirstName)
		assert.Equal(t, req.LastName, result.LastName)
		assert.Equal(t, req.Role, result.Role)
		assert.Equal(t, tenant.ID, result.TenantID)
		assert.True(t, result.IsActive)
		assert.NotZero(t, result.ID)

		// Verify password is hashed
		var user models.User
		err = db.DB.First(&user, result.ID).Error
		require.NoError(t, err)
		assert.NotEqual(t, req.Password, user.PasswordHash)
		assert.True(t, len(user.PasswordHash) > 50) // bcrypt hash length
	})
}

func TestUserService_GetUser(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		// Setup
		authRepo := auth.NewRepository(db.DB)
		authService := auth.NewService(db.DB, "test-secret", time.Hour, time.Hour*24, "test-issuer")
		service := NewService(db.DB, authRepo, authService)

		tenant := createTestTenant(t, db)
		user := createTestUser(t, db, tenant.ID)

		// Execute
		result, err := service.GetUser(tenant.ID, user.ID)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, user.ID, result.ID)
		assert.Equal(t, user.Email, result.Email)
		assert.Equal(t, tenant.ID, result.TenantID)
	})
}

func TestUserService_ListUsers(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		// Setup
		authRepo := auth.NewRepository(db.DB)
		authService := auth.NewService(db.DB, "test-secret", time.Hour, time.Hour*24, "test-issuer")
		service := NewService(db.DB, authRepo, authService)

		tenant := createTestTenant(t, db)

		// Create multiple users
		createTestUser(t, db, tenant.ID)
		createTestUser(t, db, tenant.ID)
		createTestUser(t, db, tenant.ID)

		// Execute
		req := &UserListRequest{
			Page:     1,
			PageSize: 10,
		}
		result, err := service.ListUsers(tenant.ID, req)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestUserService_UpdateUser(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		// Setup
		authRepo := auth.NewRepository(db.DB)
		authService := auth.NewService(db.DB, "test-secret", time.Hour, time.Hour*24, "test-issuer")
		service := NewService(db.DB, authRepo, authService)

		tenant := createTestTenant(t, db)
		user := createTestUser(t, db, tenant.ID)

		// Test data
		req := &UpdateUserRequest{
			FirstName: "Updated",
			LastName:  "Name",
			Role:      "support",
		}

		// Execute
		result, err := service.UpdateUser(tenant.ID, user.ID, req)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, req.FirstName, result.FirstName)
		assert.Equal(t, req.LastName, result.LastName)
		assert.Equal(t, req.Role, result.Role)
	})
}

func TestUserService_DeleteUser(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		// Setup
		authRepo := auth.NewRepository(db.DB)
		authService := auth.NewService(db.DB, "test-secret", time.Hour, time.Hour*24, "test-issuer")
		service := NewService(db.DB, authRepo, authService)

		tenant := createTestTenant(t, db)
		user := createTestUser(t, db, tenant.ID)

		// Execute
		err := service.DeleteUser(tenant.ID, user.ID)

		// Assert
		require.NoError(t, err)

		// Verify user is soft deleted
		var deletedUser models.User
		err = db.DB.Unscoped().First(&deletedUser, user.ID).Error
		require.NoError(t, err)
		assert.NotNil(t, deletedUser.DeletedAt)
	})
}

func TestUserService_ActivateUser(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		// Setup
		authRepo := auth.NewRepository(db.DB)
		authService := auth.NewService(db.DB, "test-secret", time.Hour, time.Hour*24, "test-issuer")
		service := NewService(db.DB, authRepo, authService)

		tenant := createTestTenant(t, db)
		user := createTestUser(t, db, tenant.ID)
		user.IsActive = false
		db.DB.Save(user)

		// Execute
		err := service.ActivateUser(tenant.ID, user.ID)

		// Assert
		require.NoError(t, err)

		// Verify user is activated
		var updatedUser models.User
		err = db.DB.First(&updatedUser, user.ID).Error
		require.NoError(t, err)
		assert.True(t, updatedUser.IsActive)
	})
}

func TestUserService_DeactivateUser(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		// Setup
		authRepo := auth.NewRepository(db.DB)
		authService := auth.NewService(db.DB, "test-secret", time.Hour, time.Hour*24, "test-issuer")
		service := NewService(db.DB, authRepo, authService)

		tenant := createTestTenant(t, db)
		user := createTestUser(t, db, tenant.ID)

		// Execute
		err := service.DeactivateUser(tenant.ID, user.ID)

		// Assert
		require.NoError(t, err)

		// Verify user is deactivated
		var updatedUser models.User
		err = db.DB.First(&updatedUser, user.ID).Error
		require.NoError(t, err)
		assert.False(t, updatedUser.IsActive)
	})
}

func TestUserService_GetUserStats(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		// Setup
		authRepo := auth.NewRepository(db.DB)
		authService := auth.NewService(db.DB, "test-secret", time.Hour, time.Hour*24, "test-issuer")
		service := NewService(db.DB, authRepo, authService)

		tenant := createTestTenant(t, db)

		// Create users with different roles and statuses
		createTestUserWithRole(t, db, tenant.ID, "admin", true)
		createTestUserWithRole(t, db, tenant.ID, "engineer", true)
		createTestUserWithRole(t, db, tenant.ID, "support", true)
		createTestUserWithRole(t, db, tenant.ID, "customer", false) // inactive

		// Execute
		stats, err := service.GetUserStats(tenant.ID)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, stats)
		assert.Equal(t, int64(4), stats["total_users"])
		assert.Equal(t, int64(3), stats["active_users"])
		assert.Equal(t, int64(1), stats["users_admin"])
		assert.Equal(t, int64(1), stats["users_engineer"])
		assert.Equal(t, int64(1), stats["users_support"])
		assert.Equal(t, int64(0), stats["users_customer"]) // Customer is inactive, so not counted in active users by role
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
	// Generate unique email and username using timestamp
	timestamp := time.Now().UnixNano()
	user := &models.User{
		TenantID:     tenantID,
		Email:        fmt.Sprintf("test-%d@example.com", timestamp),
		Username:     fmt.Sprintf("testuser-%d", timestamp),
		FirstName:    "Test",
		LastName:     "User",
		Role:         "admin",
		PasswordHash: "$2a$10$dummy.hash.for.testing",
		IsActive:     true,
	}

	err := db.DB.Create(user).Error
	require.NoError(t, err)

	return user
}

func createTestUserWithRole(t *testing.T, db *database.Database, tenantID uint, role string, isActive bool) *models.User {
	// Generate unique email and username using timestamp
	timestamp := time.Now().UnixNano()
	user := &models.User{
		TenantID:     tenantID,
		Email:        fmt.Sprintf("%s-%d@example.com", role, timestamp),
		Username:     fmt.Sprintf("%s-user-%d", role, timestamp),
		FirstName:    role,
		LastName:     "User",
		Role:         role,
		PasswordHash: "$2a$10$dummy.hash.for.testing",
	}

	err := db.DB.Create(user).Error
	require.NoError(t, err)

	// Update IsActive field separately to avoid GORM default override
	if user.IsActive != isActive {
		err = db.DB.Model(user).Update("is_active", isActive).Error
		require.NoError(t, err)
		// Refresh user object
		err = db.DB.First(user, user.ID).Error
		require.NoError(t, err)
	}

	return user
}
