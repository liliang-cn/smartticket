package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAuthTestHandlers(t *testing.T) (*Handlers, *gin.Engine) {
	gin.SetMode(gin.TestMode)

	// Create mock service
	service := &MockAuthService{}
	handlers := NewHandlers(service)

	// Create test router
	router := gin.New()
	return handlers, router
}

// MockAuthService implements AuthServiceInterface for testing
type MockAuthService struct {
	users       map[string]*MockUser
	tokens      map[string]*MockToken
	shouldError bool
}

type MockUser struct {
	ID       uint
	Email    string
	Username string
	Password string
	Role     string
	IsActive bool
}

type MockToken struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

func (m *MockAuthService) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	if m.shouldError {
		return nil, assert.AnError
	}

	user, exists := m.users[email]
	if !exists {
		return nil, assert.AnError
	}

	if user.Password != password {
		return nil, assert.AnError
	}

	if !user.IsActive {
		return nil, assert.AnError
	}

	token := &MockToken{
		AccessToken:  "mock-access-token",
		RefreshToken: "mock-refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	}

	m.tokens[token.AccessToken] = token

	return &LoginResponse{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		User: &UserInfo{
			ID:       user.ID,
			Email:    user.Email,
			Username: user.Username,
			Role:     user.Role,
		},
	}, nil
}

func (m *MockAuthService) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	if m.shouldError {
		return nil, assert.AnError
	}

	for _, token := range m.tokens {
		if token.RefreshToken == refreshToken && token.ExpiresAt.After(time.Now()) {
			return &TokenResponse{
				AccessToken:  "new-mock-access-token",
				RefreshToken: "new-mock-refresh-token",
				TokenType:    "Bearer",
				ExpiresIn:    3600,
			}, nil
		}
	}

	return nil, assert.AnError
}

func (m *MockAuthService) ValidateToken(ctx context.Context, token string) (*UserInfo, error) {
	if m.shouldError {
		return nil, assert.AnError
	}

	tokenData, exists := m.tokens[token]
	if !exists || tokenData.ExpiresAt.Before(time.Now()) {
		return nil, assert.AnError
	}

	// Return a mock user for valid token
	return &UserInfo{
		ID:       1,
		Email:    "test@example.com",
		Username: "testuser",
		Role:     "admin",
	}, nil
}

func (m *MockAuthService) GetUserInfo(ctx context.Context, userID uint) (*UserInfo, error) {
	if m.shouldError {
		return nil, assert.AnError
	}

	for _, user := range m.users {
		if user.ID == userID {
			return &UserInfo{
				ID:       user.ID,
				Email:    user.Email,
				Username: user.Username,
				Role:     user.Role,
			}, nil
		}
	}

	return nil, assert.AnError
}

func (m *MockAuthService) ChangePassword(ctx context.Context, userID uint, oldPassword, newPassword string) error {
	if m.shouldError {
		return assert.AnError
	}

	for _, user := range m.users {
		if user.ID == userID && user.Password == oldPassword {
			user.Password = newPassword
			return nil
		}
	}

	return assert.AnError
}

func (m *MockAuthService) AddUser(email, username, password string) *MockUser {
	if m.users == nil {
		m.users = make(map[string]*MockUser)
	}
	if m.tokens == nil {
		m.tokens = make(map[string]*MockToken)
	}

	user := &MockUser{
		ID:       uint(len(m.users) + 1),
		Email:    email,
		Username: username,
		Password: password,
		Role:     "customer",
		IsActive: true,
	}

	m.users[email] = user
	return user
}

func (m *MockAuthService) SetShouldError(shouldError bool) {
	m.shouldError = shouldError
}

func TestNewHandlers(t *testing.T) {
	service := &MockAuthService{}
	handlers := NewHandlers(service)

	assert.NotNil(t, handlers)
	assert.Equal(t, service, handlers.service)
}

func TestHandlers_Login(t *testing.T) {
	handlers, router := setupAuthTestHandlers(t)
	mockService := handlers.service.(*MockAuthService)

	// Add mock user
	mockService.AddUser("test@example.com", "testuser", "password123")

	// Setup routes
	router.POST("/auth/login", handlers.Login)

	t.Run("Successful login", func(t *testing.T) {
		loginData := map[string]string{
			"email":    "test@example.com",
			"password": "password123",
		}

		jsonData, _ := json.Marshal(loginData)
		req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})
		assert.NotEmpty(t, data["access_token"])
		assert.NotEmpty(t, data["refresh_token"])
	})

	t.Run("Invalid credentials", func(t *testing.T) {
		loginData := map[string]string{
			"email":    "test@example.com",
			"password": "wrongpassword",
		}

		jsonData, _ := json.Marshal(loginData)
		req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Missing required fields", func(t *testing.T) {
		loginData := map[string]string{
			"email": "test@example.com",
			// Missing password
		}

		jsonData, _ := json.Marshal(loginData)
		req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

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

func TestHandlers_RefreshToken(t *testing.T) {
	handlers, router := setupAuthTestHandlers(t)
	mockService := handlers.service.(*MockAuthService)

	// Setup routes
	router.POST("/auth/refresh", handlers.RefreshToken)

	t.Run("Successful token refresh", func(t *testing.T) {
		// First login to get a refresh token
		mockService.AddUser("test@example.com", "testuser", "password123")

		loginResponse, err := mockService.Login(context.Background(), "test@example.com", "password123")
		require.NoError(t, err)

		refreshData := map[string]string{
			"refresh_token": loginResponse.RefreshToken,
		}

		jsonData, _ := json.Marshal(refreshData)
		req, _ := http.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})
		assert.NotEmpty(t, data["access_token"])
		assert.NotEmpty(t, data["refresh_token"])
	})

	t.Run("Invalid refresh token", func(t *testing.T) {
		refreshData := map[string]string{
			"refresh_token": "invalid-refresh-token",
		}

		jsonData, _ := json.Marshal(refreshData)
		req, _ := http.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestHandlers_ValidateToken(t *testing.T) {
	handlers, router := setupAuthTestHandlers(t)
	mockService := handlers.service.(*MockAuthService)

	// Setup routes
	router.GET("/auth/validate", handlers.ValidateToken)

	t.Run("Valid token", func(t *testing.T) {
		// First login to get a token
		mockService.AddUser("test@example.com", "testuser", "password123")

		loginResponse, err := mockService.Login(context.Background(), "test@example.com", "password123")
		require.NoError(t, err)

		req, _ := http.NewRequest("GET", "/auth/validate", nil)
		req.Header.Set("Authorization", "Bearer "+loginResponse.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})
		assert.Equal(t, "test@example.com", data["email"])
		assert.Equal(t, "testuser", data["username"])
	})

	t.Run("Invalid token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/auth/validate", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Missing token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/auth/validate", nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestHandlers_GetProfile(t *testing.T) {
	handlers, router := setupAuthTestHandlers(t)
	mockService := handlers.service.(*MockAuthService)

	// Setup routes
	router.GET("/auth/profile", handlers.GetProfile)

	t.Run("Get user profile", func(t *testing.T) {
		// First login to get a token
		user := mockService.AddUser("test@example.com", "testuser", "password123")

		loginResponse, err := mockService.Login(context.Background(), "test@example.com", "password123")
		require.NoError(t, err)

		req, _ := http.NewRequest("GET", "/auth/profile", nil)
		req.Header.Set("Authorization", "Bearer "+loginResponse.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(user.ID), data["id"])
		assert.Equal(t, user.Email, data["email"])
		assert.Equal(t, user.Username, data["username"])
	})
}

func TestHandlers_ChangePassword(t *testing.T) {
	handlers, router := setupAuthTestHandlers(t)
	mockService := handlers.service.(*MockAuthService)

	// Setup routes
	router.POST("/auth/change-password", handlers.ChangePassword)

	t.Run("Successful password change", func(t *testing.T) {
		// First login to get a token
		mockService.AddUser("test@example.com", "testuser", "password123")

		loginResponse, err := mockService.Login(context.Background(), "test@example.com", "password123")
		require.NoError(t, err)

		passwordData := map[string]string{
			"old_password": "password123",
			"new_password": "newpassword123",
		}

		jsonData, _ := json.Marshal(passwordData)
		req, _ := http.NewRequest("POST", "/auth/change-password", bytes.NewBuffer(jsonData))
		req.Header.Set("Authorization", "Bearer "+loginResponse.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
	})

	t.Run("Wrong old password", func(t *testing.T) {
		mockService.AddUser("test2@example.com", "testuser2", "password123")

		loginResponse, err := mockService.Login(context.Background(), "test2@example.com", "password123")
		require.NoError(t, err)

		passwordData := map[string]string{
			"old_password": "wrongpassword",
			"new_password": "newpassword123",
		}

		jsonData, _ := json.Marshal(passwordData)
		req, _ := http.NewRequest("POST", "/auth/change-password", bytes.NewBuffer(jsonData))
		req.Header.Set("Authorization", "Bearer "+loginResponse.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Missing required fields", func(t *testing.T) {
		mockService.AddUser("test3@example.com", "testuser3", "password123")

		loginResponse, err := mockService.Login(context.Background(), "test3@example.com", "password123")
		require.NoError(t, err)

		passwordData := map[string]string{
			"old_password": "password123",
			// Missing new_password
		}

		jsonData, _ := json.Marshal(passwordData)
		req, _ := http.NewRequest("POST", "/auth/change-password", bytes.NewBuffer(jsonData))
		req.Header.Set("Authorization", "Bearer "+loginResponse.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandlers_GetMe(t *testing.T) {
	handlers, router := setupAuthTestHandlers(t)
	mockService := handlers.service.(*MockAuthService)

	// Setup routes
	router.GET("/auth/me", handlers.GetMe)

	t.Run("Get current user info", func(t *testing.T) {
		// First login to get a token
		user := mockService.AddUser("test@example.com", "testuser", "password123")

		loginResponse, err := mockService.Login(context.Background(), "test@example.com", "password123")
		require.NoError(t, err)

		req, _ := http.NewRequest("GET", "/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+loginResponse.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(user.ID), data["id"])
		assert.Equal(t, user.Email, data["email"])
		assert.Equal(t, user.Username, data["username"])
	})
}

func TestHandlers_Logout(t *testing.T) {
	handlers, router := setupAuthTestHandlers(t)
	mockService := handlers.service.(*MockAuthService)

	// Setup routes
	router.POST("/auth/logout", handlers.Logout)

	t.Run("Successful logout", func(t *testing.T) {
		// First login to get a token
		mockService.AddUser("test@example.com", "testuser", "password123")

		loginResponse, err := mockService.Login(context.Background(), "test@example.com", "password123")
		require.NoError(t, err)

		req, _ := http.NewRequest("POST", "/auth/logout", nil)
		req.Header.Set("Authorization", "Bearer "+loginResponse.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		assert.Equal(t, "Logged out successfully", response["message"])
	})
}
