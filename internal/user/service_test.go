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

		// Seed the role required for assignment
		createTestRole(t, db, "customer")

		// Test data
		req := &CreateUserRequest{
			Email:     "newuser@example.com",
			Username:  "newuser",
			FirstName: "New",
			LastName:  "User",
			Password:  "Password123!",
			Role:      "customer",
			IsActive:  true,
		}

		// Execute
		result, err := service.CreateUser(req)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, req.Email, result.Email)
		assert.Equal(t, req.FirstName, result.FirstName)
		assert.Equal(t, req.LastName, result.LastName)
		// Role is now handled through UserRole associations, not direct User field
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

		user := createTestUser(t, db)

		// Execute
		result, err := service.GetUser(user.ID)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, user.ID, result.ID)
		assert.Equal(t, user.Email, result.Email)
	})
}

func TestUserService_ListUsers(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		// Setup
		authRepo := auth.NewRepository(db.DB)
		authService := auth.NewService(db.DB, "test-secret", time.Hour, time.Hour*24, "test-issuer")
		service := NewService(db.DB, authRepo, authService)

		// Create multiple users
		createTestUser(t, db)
		createTestUser(t, db)
		createTestUser(t, db)

		// Execute
		req := &UserListRequest{
			Page:     1,
			PageSize: 10,
		}
		result, err := service.ListUsers(req)

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

		user := createTestUser(t, db)

		// Test data
		req := &UpdateUserRequest{
			FirstName: "Updated",
			LastName:  "Name",
		}

		// Execute
		result, err := service.UpdateUser(user.ID, req)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, req.FirstName, result.FirstName)
		assert.Equal(t, req.LastName, result.LastName)
		// Role is now handled through UserRole associations, not direct User field
	})
}

func TestUserService_DeleteUser(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		// Setup
		authRepo := auth.NewRepository(db.DB)
		authService := auth.NewService(db.DB, "test-secret", time.Hour, time.Hour*24, "test-issuer")
		service := NewService(db.DB, authRepo, authService)

		user := createTestUser(t, db)

		// Execute
		err := service.DeleteUser(user.ID)

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

		user := createTestUser(t, db)
		user.IsActive = false
		db.Save(user)

		// Execute
		err := service.ActivateUser(user.ID)

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

		user := createTestUser(t, db)

		// Execute
		err := service.DeactivateUser(user.ID)

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

		// Create users with different statuses
		createTestUserWithRole(t, db, "admin", true)
		createTestUserWithRole(t, db, "engineer", true)
		createTestUserWithRole(t, db, "support", true)
		createTestUserWithRole(t, db, "customer", false) // inactive

		// Execute
		stats, err := service.GetUserStats()

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, stats)
		assert.Equal(t, int64(4), stats["total_users"])
		assert.Equal(t, int64(3), stats["active_users"])
	})
}

// Helper functions for creating test data

func createTestRole(t *testing.T, db *database.Database, name string) *models.Role {
	role := &models.Role{
		Name:     name,
		IsActive: true,
	}
	err := db.DB.Create(role).Error
	require.NoError(t, err)
	return role
}

func createTestUser(t *testing.T, db *database.Database) *models.User {
	// Generate unique email and username using timestamp
	timestamp := time.Now().UnixNano()
	user := &models.User{
		Email:        fmt.Sprintf("test-%d@example.com", timestamp),
		Username:     fmt.Sprintf("testuser-%d", timestamp),
		FirstName:    "Test",
		LastName:     "User",
		PasswordHash: "$2a$10$dummy.hash.for.testing",
		IsActive:     true,
	}

	err := db.DB.Create(user).Error
	require.NoError(t, err)

	return user
}

func createTestUserWithRole(t *testing.T, db *database.Database, role string, isActive bool) *models.User {
	// Generate unique email and username using timestamp
	timestamp := time.Now().UnixNano()
	user := &models.User{
		Email:        fmt.Sprintf("%s-%d@example.com", role, timestamp),
		Username:     fmt.Sprintf("%s-user-%d", role, timestamp),
		FirstName:    role,
		LastName:     "User",
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
