package auth

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

func setupAuthTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Enable foreign key constraints
	db.Exec("PRAGMA foreign_keys = ON")

	// Migrate required models
	err = db.AutoMigrate(
		&models.Tenant{},
		&models.User{},
		&models.APIKey{},
		&models.AuditLog{},
	)
	require.NoError(t, err)

	return db
}

func TestNewRepository(t *testing.T) {
	db := setupAuthTestDB(t)

	repo := NewRepository(db)
	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}

func TestRepository_CreateUser(t *testing.T) {
	db := setupAuthTestDB(t)
	repo := NewRepository(db)

	t.Run("Create valid user", func(t *testing.T) {
		// Create tenant first
		tenant := &models.Tenant{
			Name:     "Test Tenant",
			Slug:     "test-tenant",
			IsActive: true,
		}
		err := db.Create(tenant).Error
		require.NoError(t, err)

		user := &models.User{
			TenantID:     tenant.ID,
			Email:        "test@example.com",
			Username:     "testuser",
			FirstName:    "Test",
			LastName:     "User",
			Role:         "customer",
			PasswordHash: "hashed_password",
			IsActive:     true,
		}

		err = repo.CreateUser(user)
		assert.NoError(t, err)
		assert.NotZero(t, user.ID)
		assert.NotZero(t, user.CreatedAt)
		assert.NotZero(t, user.UpdatedAt)
	})

	t.Run("Create user without tenant", func(t *testing.T) {
		user := &models.User{
			Email:        "notenant@example.com",
			Username:     "notenant",
			PasswordHash: "hashed_password",
			IsActive:     true,
		}

		err := repo.CreateUser(user)
		assert.Error(t, err)
	})
}

func TestRepository_GetUserByID(t *testing.T) {
	db := setupAuthTestDB(t)
	repo := NewRepository(db)

	// Setup test data
	tenant := &models.Tenant{
		Name:     "Test Tenant",
		Slug:     "test-tenant",
		IsActive: true,
	}
	err := db.Create(tenant).Error
	require.NoError(t, err)

	user := &models.User{
		TenantID:     tenant.ID,
		Email:        "test@example.com",
		Username:     "testuser",
		FirstName:    "Test",
		LastName:     "User",
		Role:         "customer",
		PasswordHash: "hashed_password",
		IsActive:     true,
	}
	err = repo.CreateUser(user)
	require.NoError(t, err)

	t.Run("Get existing user", func(t *testing.T) {
		found, err := repo.GetUserByID(user.ID, user.TenantID)
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, user.Email, found.Email)
		assert.Equal(t, user.Username, found.Username)
	})

	t.Run("Get non-existent user", func(t *testing.T) {
		found, err := repo.GetUserByID(99999, user.TenantID)
		assert.Error(t, err)
		assert.Nil(t, found)
	})
}

func TestRepository_GetUserByEmail(t *testing.T) {
	db := setupAuthTestDB(t)
	repo := NewRepository(db)

	// Setup test data
	tenant := &models.Tenant{
		Name:     "Test Tenant",
		Slug:     "test-tenant",
		IsActive: true,
	}
	err := db.Create(tenant).Error
	require.NoError(t, err)

	user := &models.User{
		TenantID:     tenant.ID,
		Email:        "test@example.com",
		Username:     "testuser",
		FirstName:    "Test",
		LastName:     "User",
		Role:         "customer",
		PasswordHash: "hashed_password",
		IsActive:     true,
	}
	err = repo.CreateUser(user)
	require.NoError(t, err)

	t.Run("Get existing user by email", func(t *testing.T) {
		found, err := repo.GetUserByEmail(user.Email, user.TenantID)
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, user.ID, found.ID)
		assert.Equal(t, user.Username, found.Username)
	})

	t.Run("Get non-existent user by email", func(t *testing.T) {
		found, err := repo.GetUserByEmail("nonexistent@example.com", user.TenantID)
		assert.Error(t, err)
		assert.Nil(t, found)
	})
}

func TestRepository_GetUserByUsername(t *testing.T) {
	db := setupAuthTestDB(t)
	repo := NewRepository(db)

	// Setup test data
	tenant := &models.Tenant{
		Name:     "Test Tenant",
		Slug:     "test-tenant",
		IsActive: true,
	}
	err := db.Create(tenant).Error
	require.NoError(t, err)

	user := &models.User{
		TenantID:     tenant.ID,
		Email:        "test@example.com",
		Username:     "testuser",
		FirstName:    "Test",
		LastName:     "User",
		Role:         "customer",
		PasswordHash: "hashed_password",
		IsActive:     true,
	}
	err = repo.CreateUser(user)
	require.NoError(t, err)

	t.Run("Get existing user by username", func(t *testing.T) {
		found, err := repo.GetUserByUsername(user.Username, user.TenantID)
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, user.ID, found.ID)
		assert.Equal(t, user.Email, found.Email)
	})

	t.Run("Get non-existent user by username", func(t *testing.T) {
		found, err := repo.GetUserByUsername("nonexistent", user.TenantID)
		assert.Error(t, err)
		assert.Nil(t, found)
	})
}

func TestRepository_UpdateUser(t *testing.T) {
	db := setupAuthTestDB(t)
	repo := NewRepository(db)

	// Setup test data
	tenant := &models.Tenant{
		Name:     "Test Tenant",
		Slug:     "test-tenant",
		IsActive: true,
	}
	err := db.Create(tenant).Error
	require.NoError(t, err)

	user := &models.User{
		TenantID:     tenant.ID,
		Email:        "test@example.com",
		Username:     "testuser",
		FirstName:    "Test",
		LastName:     "User",
		Role:         "customer",
		PasswordHash: "hashed_password",
		IsActive:     true,
	}
	err = repo.CreateUser(user)
	require.NoError(t, err)

	t.Run("Update existing user", func(t *testing.T) {
		user.FirstName = "Updated"
		user.LastName = "Name"
		user.Role = "admin"

		err := repo.UpdateUser(user)
		assert.NoError(t, err)

		// Verify update
		updated, err := repo.GetUserByID(user.ID, user.TenantID)
		assert.NoError(t, err)
		assert.Equal(t, "Updated", updated.FirstName)
		assert.Equal(t, "Name", updated.LastName)
		assert.Equal(t, "admin", updated.Role)
	})

	t.Run("Update non-existent user", func(t *testing.T) {
		nonExistentUser := &models.User{
			Email: "nonexistent@example.com",
		}
		// Set the ID field from the embedded BaseModel
		nonExistentUser.ID = 99999

		err := repo.UpdateUser(nonExistentUser)
		assert.Error(t, err)
	})
}

func TestRepository_UpdatePassword(t *testing.T) {
	db := setupAuthTestDB(t)
	repo := NewRepository(db)

	// Setup test data
	tenant := &models.Tenant{
		Name:     "Test Tenant",
		Slug:     "test-tenant",
		IsActive: true,
	}
	err := db.Create(tenant).Error
	require.NoError(t, err)

	user := &models.User{
		TenantID:     tenant.ID,
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: "old_password_hash",
		IsActive:     true,
	}
	err = repo.CreateUser(user)
	require.NoError(t, err)

	t.Run("Update user password", func(t *testing.T) {
		newPasswordHash := "new_password_hash"
		err := repo.UpdatePassword(user.ID, newPasswordHash)
		assert.NoError(t, err)

		// Verify update
		updated, err := repo.GetUserByID(user.ID, user.TenantID)
		assert.NoError(t, err)
		assert.Equal(t, newPasswordHash, updated.PasswordHash)
	})

	t.Run("Update password for non-existent user", func(t *testing.T) {
		err := repo.UpdatePassword(99999, "new_password")
		assert.Error(t, err)
	})
}

func TestRepository_UpdateLastLogin(t *testing.T) {
	db := setupAuthTestDB(t)
	repo := NewRepository(db)

	// Setup test data
	tenant := &models.Tenant{
		Name:     "Test Tenant",
		Slug:     "test-tenant",
		IsActive: true,
	}
	err := db.Create(tenant).Error
	require.NoError(t, err)

	user := &models.User{
		TenantID:     tenant.ID,
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: "hashed_password",
		IsActive:     true,
	}
	err = repo.CreateUser(user)
	require.NoError(t, err)

	t.Run("Update last login", func(t *testing.T) {
		loginTime := time.Now()
		err := repo.UpdateLastLogin(user.ID)
		assert.NoError(t, err)

		// Verify update
		updated, err := repo.GetUserByID(user.ID, user.TenantID)
		assert.NoError(t, err)
		assert.NotNil(t, updated.LastLoginAt)
		assert.WithinDuration(t, loginTime, *updated.LastLoginAt, time.Second)
	})

	t.Run("Update last login for non-existent user", func(t *testing.T) {
		err := repo.UpdateLastLogin(99999)
		assert.Error(t, err)
	})
}

func TestRepository_CheckEmailExists(t *testing.T) {
	db := setupAuthTestDB(t)
	repo := NewRepository(db)

	// Setup test data
	tenant := &models.Tenant{
		Name:     "Test Tenant",
		Slug:     "test-tenant",
		IsActive: true,
	}
	err := db.Create(tenant).Error
	require.NoError(t, err)

	user := &models.User{
		TenantID:     tenant.ID,
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: "hashed_password",
		IsActive:     true,
	}
	err = repo.CreateUser(user)
	require.NoError(t, err)

	t.Run("Check existing email", func(t *testing.T) {
		exists, err := repo.CheckEmailExists(user.Email, user.TenantID)
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("Check non-existent email", func(t *testing.T) {
		exists, err := repo.CheckEmailExists("nonexistent@example.com", user.TenantID)
		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestRepository_CheckUsernameExists(t *testing.T) {
	db := setupAuthTestDB(t)
	repo := NewRepository(db)

	// Setup test data
	tenant := &models.Tenant{
		Name:     "Test Tenant",
		Slug:     "test-tenant",
		IsActive: true,
	}
	err := db.Create(tenant).Error
	require.NoError(t, err)

	user := &models.User{
		TenantID:     tenant.ID,
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: "hashed_password",
		IsActive:     true,
	}
	err = repo.CreateUser(user)
	require.NoError(t, err)

	t.Run("Check existing username", func(t *testing.T) {
		exists, err := repo.CheckUsernameExists(user.Username, user.TenantID)
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("Check non-existent username", func(t *testing.T) {
		exists, err := repo.CheckUsernameExists("nonexistent", user.TenantID)
		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestRepository_CreateTenant(t *testing.T) {
	db := setupAuthTestDB(t)
	repo := NewRepository(db)

	t.Run("Create valid tenant", func(t *testing.T) {
		tenant := &models.Tenant{
			Name:     "Test Tenant",
			Slug:     "test-tenant",
			Domain:   "test.example.com",
			Plan:     "basic",
			MaxUsers: 100,
			IsActive: true,
		}

		err := repo.CreateTenant(tenant)
		assert.NoError(t, err)
		assert.NotZero(t, tenant.ID)
		assert.NotZero(t, tenant.CreatedAt)
		assert.NotZero(t, tenant.UpdatedAt)
	})

	t.Run("Create tenant with duplicate slug", func(t *testing.T) {
		tenant1 := &models.Tenant{
			Name:     "Tenant 1",
			Slug:     "duplicate-slug",
			IsActive: true,
		}
		err := repo.CreateTenant(tenant1)
		assert.NoError(t, err)

		tenant2 := &models.Tenant{
			Name:     "Tenant 2",
			Slug:     "duplicate-slug", // Same slug
			IsActive: true,
		}
		err = repo.CreateTenant(tenant2)
		assert.Error(t, err) // Should fail due to unique constraint
	})
}

func TestRepository_GetTenantByID(t *testing.T) {
	db := setupAuthTestDB(t)
	repo := NewRepository(db)

	// Setup test data
	tenant := &models.Tenant{
		Name:     "Test Tenant",
		Slug:     "test-tenant",
		IsActive: true,
	}
	err := repo.CreateTenant(tenant)
	require.NoError(t, err)

	t.Run("Get existing tenant", func(t *testing.T) {
		found, err := repo.GetTenantByID(tenant.ID)
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, tenant.Name, found.Name)
		assert.Equal(t, tenant.Slug, found.Slug)
	})

	t.Run("Get non-existent tenant", func(t *testing.T) {
		found, err := repo.GetTenantByID(99999)
		assert.Error(t, err)
		assert.Nil(t, found)
	})
}

func TestRepository_GetTenantByDomain(t *testing.T) {
	db := setupAuthTestDB(t)
	repo := NewRepository(db)

	// Setup test data
	tenant := &models.Tenant{
		Name:     "Test Tenant",
		Slug:     "test-tenant",
		Domain:   "test.example.com",
		IsActive: true,
	}
	err := repo.CreateTenant(tenant)
	require.NoError(t, err)

	t.Run("Get existing tenant by domain", func(t *testing.T) {
		found, err := repo.GetTenantByDomain(tenant.Domain)
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, tenant.ID, found.ID)
		assert.Equal(t, tenant.Name, found.Name)
	})

	t.Run("Get non-existent tenant by domain", func(t *testing.T) {
		found, err := repo.GetTenantByDomain("nonexistent.example.com")
		assert.Error(t, err)
		assert.Nil(t, found)
	})
}

func TestRepository_ListUsers(t *testing.T) {
	db := setupAuthTestDB(t)
	repo := NewRepository(db)

	// Setup test data
	tenant := &models.Tenant{
		Name:     "Test Tenant",
		Slug:     "test-tenant",
		IsActive: true,
	}
	err := db.Create(tenant).Error
	require.NoError(t, err)

	// Create multiple users
	users := make([]*models.User, 5)
	for i := 0; i < 5; i++ {
		users[i] = &models.User{
			TenantID:     tenant.ID,
			Email:        fmt.Sprintf("user%d@example.com", i+1),
			Username:     fmt.Sprintf("user%d", i+1),
			FirstName:    fmt.Sprintf("User%d", i+1),
			Role:         "customer",
			PasswordHash: "hashed_password",
			IsActive:     true,
		}
		err = repo.CreateUser(users[i])
		require.NoError(t, err)
	}

	t.Run("List all users", func(t *testing.T) {
		allUsers, total, err := repo.ListUsers(tenant.ID, 1, 10, nil)
		assert.NoError(t, err)
		assert.Len(t, allUsers, 5)
		assert.Equal(t, int64(5), total)
	})

	t.Run("List users with role filter", func(t *testing.T) {
		filters := map[string]interface{}{"role": "customer"}
		filteredUsers, total, err := repo.ListUsers(tenant.ID, 1, 10, filters)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(filteredUsers), 5)
		assert.GreaterOrEqual(t, total, int64(5))
	})

	t.Run("List users with pagination", func(t *testing.T) {
		usersPage, total, err := repo.ListUsers(tenant.ID, 1, 3, nil)
		assert.NoError(t, err)
		assert.Len(t, usersPage, 3) // Exactly 3 users due to page size
		assert.Equal(t, int64(5), total)
	})
}

func TestRepository_GetUserStats(t *testing.T) {
	db := setupAuthTestDB(t)
	repo := NewRepository(db)

	// Setup test data
	tenant := &models.Tenant{
		Name:     "Test Tenant",
		Slug:     "test-tenant",
		IsActive: true,
	}
	err := db.Create(tenant).Error
	require.NoError(t, err)

	// Create users with different roles
	roles := []string{"admin", "engineer", "support", "customer"}
	for _, role := range roles {
		user := &models.User{
			TenantID:     tenant.ID,
			Email:        fmt.Sprintf("%s@example.com", role),
			Username:     role,
			Role:         role,
			PasswordHash: "hashed_password",
			IsActive:     true,
		}
		err = repo.CreateUser(user)
		require.NoError(t, err)
	}

	t.Run("Get user statistics", func(t *testing.T) {
		stats, err := repo.GetUserStats(tenant.ID)
		assert.NoError(t, err)
		assert.NotNil(t, stats)
		assert.GreaterOrEqual(t, stats["total_users"], int64(4))
		assert.GreaterOrEqual(t, stats["active_users"], int64(4))
	})
}
