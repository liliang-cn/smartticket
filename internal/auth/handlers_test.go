package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/company/smartticket/internal/database"
	"github.com/company/smartticket/tests/testutils"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test basic handler initialization without requiring a full service.
func TestNewHandlers_NilService(t *testing.T) {
	// Test with nil service - should still create handlers
	handlers := NewHandlers(nil)
	assert.NotNil(t, handlers)
}

// Test basic handler structure.
func TestHandlers_HandlerExists(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create handlers with nil service for structural testing
	handlers := NewHandlers(nil)
	router := gin.New()

	// Test that all handler methods exist and can be registered
	router.POST("/auth/login", handlers.Login)
	router.POST("/auth/refresh", handlers.RefreshToken)
	router.POST("/auth/logout", handlers.Logout)
	router.GET("/auth/profile", handlers.GetProfile)
	router.GET("/auth/me", handlers.GetMe)
	router.GET("/auth/validate", handlers.ValidateToken)
	router.POST("/auth/change-password", handlers.ChangePassword)

	assert.NotNil(t, router)
}

// Test that GetMe returns the full UserInfo so clients keep identity on reload.
func TestHandlers_GetMe_ReturnsFullUserInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		service := NewService(db.DB, "test-secret", time.Hour, time.Hour*24, "test-issuer")
		user := createTestUser(t, db, "me@example.com", "password123")

		handlers := NewHandlers(service)
		router := gin.New()
		router.GET("/auth/me", func(c *gin.Context) {
			c.Set("user_id", user.ID)
			c.Set("request_id", "req-123")
			handlers.GetMe(c)
		})

		req, _ := http.NewRequest("GET", "/auth/me", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Success   bool     `json:"success"`
			RequestID string   `json:"request_id"`
			Data      UserInfo `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

		assert.True(t, resp.Success)
		assert.Equal(t, "req-123", resp.RequestID)
		assert.Equal(t, user.ID, resp.Data.ID)
		assert.Equal(t, user.Email, resp.Data.Email)
		assert.Equal(t, user.Username, resp.Data.Username)
		assert.Equal(t, user.FirstName, resp.Data.FirstName)
		assert.Equal(t, user.LastName, resp.Data.LastName)
	})
}

// Test that GetMe rejects an invalid user context.
func TestHandlers_GetMe_InvalidContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handlers := NewHandlers(nil)
	router := gin.New()
	router.GET("/auth/me", handlers.GetMe)

	req, _ := http.NewRequest("GET", "/auth/me", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// Test login handler with basic request validation.
func TestHandlers_Login_BasicValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create handlers with nil service - we're just testing request validation
	handlers := NewHandlers(nil)
	router := gin.New()
	router.POST("/auth/login", handlers.Login)

	t.Run("Missing required fields", func(t *testing.T) {
		// Test with missing password
		loginData := map[string]string{
			"email": "test@example.com",
			// Missing password
		}

		jsonData, _ := json.Marshal(loginData)
		req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 400 due to binding validation
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer([]byte("{invalid json")))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// Test refresh token handler with basic request validation.
func TestHandlers_RefreshToken_BasicValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handlers := NewHandlers(nil)
	router := gin.New()
	router.POST("/auth/refresh", handlers.RefreshToken)

	t.Run("Missing refresh token", func(t *testing.T) {
		refreshData := map[string]string{} // Empty request

		jsonData, _ := json.Marshal(refreshData)
		req, _ := http.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/auth/refresh", bytes.NewBuffer([]byte("{invalid json")))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// Test change password handler with basic request validation.
func TestHandlers_ChangePassword_BasicValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handlers := NewHandlers(nil)
	router := gin.New()
	router.POST("/auth/change-password", handlers.ChangePassword)

	t.Run("Missing required fields", func(t *testing.T) {
		passwordData := map[string]string{
			"current_password": "password123",
			// Missing new_password and confirm_password
		}

		jsonData, _ := json.Marshal(passwordData)
		req, _ := http.NewRequest("POST", "/auth/change-password", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Password mismatch", func(t *testing.T) {
		passwordData := map[string]string{
			"current_password": "password123",
			"new_password":     "newpassword123",
			"confirm_password": "differentpassword",
		}

		jsonData, _ := json.Marshal(passwordData)
		req, _ := http.NewRequest("POST", "/auth/change-password", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
