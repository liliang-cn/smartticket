package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/company/smartticket/internal/database"
	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/tests/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthService_Login(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		// Setup
		service := NewService(db.DB, "test-secret", time.Hour, time.Hour*24, "test-issuer")
		user := createTestUser(t, db, "test@example.com", "password123")

		// Test data
		req := &LoginRequest{
			Email:    "test@example.com",
			Password: "password123",
		}

		// Execute
		result, err := service.Login(req, "127.0.0.1", "test-agent")

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.Tokens)
		assert.NotEmpty(t, result.Tokens.AccessToken)
		assert.NotEmpty(t, result.Tokens.RefreshToken)
		assert.Equal(t, user.ID, result.User.ID)
		assert.Equal(t, user.Email, result.User.Email)
	})
}

func TestAuthService_Login_InvalidCredentials(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		// Setup
		service := NewService(db.DB, "test-secret", time.Hour, time.Hour*24, "test-issuer")
		createTestUser(t, db, "test@example.com", "password123")

		// Test data
		req := &LoginRequest{
			Email:    "test@example.com",
			Password: "wrongpassword",
		}

		// Execute
		result, err := service.Login(req, "127.0.0.1", "test-agent")

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid email or password")
	})
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		// Setup
		service := NewService(db.DB, "test-secret", time.Hour, time.Hour*24, "test-issuer")

		// Test data
		req := &LoginRequest{
			Email:    "nonexistent@example.com",
			Password: "password123",
		}

		// Execute
		result, err := service.Login(req, "127.0.0.1", "test-agent")

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid email or password")
	})
}

func TestAuthService_Login_InactiveUser(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		// Setup
		service := NewService(db.DB, "test-secret", time.Hour, time.Hour*24, "test-issuer")
		user := createTestUser(t, db, "test@example.com", "password123")
		user.IsActive = false
		db.Save(user)

		// Test data
		req := &LoginRequest{
			Email:    "test@example.com",
			Password: "password123",
		}

		// Execute
		result, err := service.Login(req, "127.0.0.1", "test-agent")

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid email or password")
	})
}

func TestAuthService_RefreshToken(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		// Setup
		service := NewService(db.DB, "test-secret", time.Hour, time.Hour*24, "test-issuer")
		_ = createTestUser(t, db, "test@example.com", "password123")

		// First login to get tokens
		loginReq := &LoginRequest{
			Email:    "test@example.com",
			Password: "password123",
		}
		loginResult, err := service.Login(loginReq, "127.0.0.1", "test-agent")
		require.NoError(t, err)

		// Execute
		result, err := service.RefreshToken(loginResult.Tokens.RefreshToken)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.AccessToken)
		// Note: Access tokens might be the same if generated within the same second,
		// which is normal for testing with fixed timestamps
	})
}

func TestAuthService_RefreshToken_InvalidToken(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		// Setup
		service := NewService(db.DB, "test-secret", time.Hour, time.Hour*24, "test-issuer")

		// Execute
		result, err := service.RefreshToken("invalid.refresh.token")

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		// Check for any of the possible error messages for invalid tokens
		errorMsg := err.Error()
		assert.True(t,
			strings.Contains(errorMsg, "invalid token") ||
				strings.Contains(errorMsg, "token is malformed") ||
				strings.Contains(errorMsg, "invalid refresh token"),
			"Expected invalid token error, got: %s", errorMsg)
	})
}

func TestAuthService_ValidateToken(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		// Setup
		service := NewService(db.DB, "test-secret", time.Hour, time.Hour*24, "test-issuer")
		user := createTestUser(t, db, "test@example.com", "password123")

		// First login to get token
		loginReq := &LoginRequest{
			Email:    "test@example.com",
			Password: "password123",
		}
		loginResult, err := service.Login(loginReq, "127.0.0.1", "test-agent")
		require.NoError(t, err)

		// Execute
		result, err := service.ValidateToken(loginResult.Tokens.AccessToken)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, user.ID, result.UserID)
		assert.Equal(t, user.Email, result.Email)
		// Role is now determined by GetUserEffectiveRole, not stored on User model
		// We expect "customer" as default role since no role assignments are created in test
		assert.Equal(t, "customer", result.Role)
	})
}

func TestAuthService_ValidateToken_InvalidToken(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		// Setup
		service := NewService(db.DB, "test-secret", time.Hour, time.Hour*24, "test-issuer")

		// Execute
		result, err := service.ValidateToken("invalid.access.token")

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		// Check for any of the possible error messages for invalid tokens
		errorMsg := err.Error()
		assert.True(t,
			strings.Contains(errorMsg, "invalid token") ||
				strings.Contains(errorMsg, "token is malformed") ||
				strings.Contains(errorMsg, "invalid access token"),
			"Expected invalid token error, got: %s", errorMsg)
	})
}

func TestAuthService_GetUserInfo(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		// Setup
		service := NewService(db.DB, "test-secret", time.Hour, time.Hour*24, "test-issuer")
		user := createTestUser(t, db, "test@example.com", "password123")

		// Execute
		result, err := service.GetUserInfo(user.ID)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, user.ID, result.ID)
		assert.Equal(t, user.Email, result.Email)
	})
}

func TestAuthService_GetUserInfo_UserNotFound(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		// Setup
		service := NewService(db.DB, "test-secret", time.Hour, time.Hour*24, "test-issuer")

		// Execute
		result, err := service.GetUserInfo(999999)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not found")
	})
}

// Helper functions for creating test data

func createTestUser(t *testing.T, db *database.Database, email, password string) *models.User {
	// Create a temporary service just for hashing password
	tempService := NewService(db.DB, "test-secret", time.Hour, time.Hour*24, "test-issuer")

	// Hash password
	hashedPassword, err := tempService.hashPassword(password)
	require.NoError(t, err)

	user := &models.User{
		Email:        email,
		Username:     "testuser",
		FirstName:    "Test",
		LastName:     "User",
		PasswordHash: hashedPassword,
		IsActive:     true,
	}

	err = db.DB.Create(user).Error
	require.NoError(t, err)

	return user
}
