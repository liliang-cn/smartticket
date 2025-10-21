package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/company/smartticket/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// MockPermissionService is a mock implementation of PermissionService
type MockPermissionService struct {
	mock.Mock
}

func (m *MockPermissionService) GetUserPermissions(ctx context.Context, userID uint, tenantID string) ([]models.Permission, error) {
	args := m.Called(ctx, userID, tenantID)
	return args.Get(0).([]models.Permission), args.Error(1)
}

func (m *MockPermissionService) GetUserRoles(ctx context.Context, userID uint, tenantID string) ([]models.Role, error) {
	args := m.Called(ctx, userID, tenantID)
	return args.Get(0).([]models.Role), args.Error(1)
}

func (m *MockPermissionService) GetDatabase() *gorm.DB {
	// Return a mock DB or nil for testing
	args := m.Called()
	return args.Get(0).(*gorm.DB)
}

func (m *MockPermissionService) HasPermission(ctx context.Context, userID uint, tenantID string, permissionCode string) (bool, error) {
	args := m.Called(ctx, userID, tenantID, permissionCode)
	return args.Bool(0), args.Error(1)
}

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Migrate essential models for middleware testing
	err = db.AutoMigrate(
		&models.Tenant{},
		&models.User{},
		&models.Ticket{},
		&models.Message{},
		&models.KnowledgeArticle{},
		&models.Permission{},
		&models.Role{},
		&models.RolePermission{},
		&models.UserPermission{},
		&models.UserRole{},
	)
	require.NoError(t, err)

	return db
}

// setupTestGin creates a gin test context
func setupTestGin() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	return c, w
}

// createTestUser creates a test user with all required fields
func createTestUser(id uint, email string) *models.User {
	return &models.User{
		BaseModel:    models.BaseModel{ID: id},
		TenantID:     1,
		Email:        email,
		Username:     "testuser",
		PasswordHash: "hashed_password",
		Role:         "customer",
		IsActive:     true,
	}
}

func TestPermissionMiddleware_RequirePermission(t *testing.T) {
	mockService := new(MockPermissionService)
	middleware := NewPermissionMiddleware(mockService)

	tests := []struct {
		name             string
		setupUser        func() *models.User
		setupTenant      func() string
		setupPermissions func() []models.Permission
		setupRoles       func() []models.Role
		requiredPerm     string
		expectedStatus   int
		expectedError    string
	}{
		{
			name: "User has required permission directly",
			setupUser: func() *models.User {
				return createTestUser(1, "test@example.com")
			},
			setupTenant: func() string {
				return "tenant-123"
			},
			setupPermissions: func() []models.Permission {
				return []models.Permission{
					{Code: "ticket:read", Name: "Read Tickets"},
				}
			},
			setupRoles:     func() []models.Role { return []models.Role{} },
			requiredPerm:   "ticket:read",
			expectedStatus: http.StatusOK,
		},
		{
			name: "User has required permission through role",
			setupUser: func() *models.User {
				return createTestUser(1, "test@example.com")
			},
			setupTenant: func() string {
				return "tenant-123"
			},
			setupPermissions: func() []models.Permission {
				return []models.Permission{}
			},
			setupRoles: func() []models.Role {
				role := models.Role{
					Name: "Support",
					Permissions: []models.Permission{
						{Code: "ticket:read", Name: "Read Tickets"},
					},
				}
				return []models.Role{role}
			},
			requiredPerm:   "ticket:read",
			expectedStatus: http.StatusOK,
		},
		{
			name: "User does not have required permission",
			setupUser: func() *models.User {
				return createTestUser(1, "test@example.com")
			},
			setupTenant: func() string {
				return "tenant-123"
			},
			setupPermissions: func() []models.Permission {
				return []models.Permission{
					{Code: "ticket:write", Name: "Write Tickets"},
				}
			},
			setupRoles:     func() []models.Role { return []models.Role{} },
			requiredPerm:   "ticket:read",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Insufficient permissions",
		},
		{
			name: "No user in context",
			setupUser: func() *models.User {
				return nil
			},
			setupTenant:      func() string { return "tenant-123" },
			setupPermissions: func() []models.Permission { return []models.Permission{} },
			setupRoles:       func() []models.Role { return []models.Role{} },
			requiredPerm:     "ticket:read",
			expectedStatus:   http.StatusUnauthorized,
			expectedError:    "Authentication required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := setupTestGin()

			// Setup user in context
			if tt.setupUser() != nil {
				c.Set("user", tt.setupUser())
				c.Set("tenant_id", tt.setupTenant())

				// Setup mock expectations
				mockService.On("GetUserPermissions", mock.Anything, tt.setupUser().ID, tt.setupTenant()).
					Return(tt.setupPermissions(), nil)
				mockService.On("GetUserRoles", mock.Anything, tt.setupUser().ID, tt.setupTenant()).
					Return(tt.setupRoles(), nil)
			}

			// Create middleware handler
			handler := middleware.RequirePermission(tt.requiredPerm)

			// Create a simple downstream handler
			handler(c)

			// Check response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				errorData, ok := response["error"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, tt.expectedError, errorData["message"])
			}

			// Verify mock expectations
			if tt.setupUser() != nil {
				mockService.AssertExpectations(t)
			}

			// Reset mock for next test
			mockService.ExpectedCalls = nil
		})
	}
}

func TestPermissionMiddleware_RequireAnyPermission(t *testing.T) {
	mockService := new(MockPermissionService)
	middleware := NewPermissionMiddleware(mockService)

	tests := []struct {
		name             string
		setupUser        func() *models.User
		setupTenant      func() string
		setupPermissions func() []models.Permission
		setupRoles       func() []models.Role
		requiredPerms    []string
		expectedStatus   int
		expectedError    string
	}{
		{
			name: "User has one of required permissions directly",
			setupUser: func() *models.User {
				return createTestUser(1, "test@example.com")
			},
			setupTenant: func() string {
				return "tenant-123"
			},
			setupPermissions: func() []models.Permission {
				return []models.Permission{
					{Code: "ticket:read", Name: "Read Tickets"},
				}
			},
			setupRoles:     func() []models.Role { return []models.Role{} },
			requiredPerms:  []string{"ticket:read", "ticket:write"},
			expectedStatus: http.StatusOK,
		},
		{
			name: "User has one of required permissions through role",
			setupUser: func() *models.User {
				return createTestUser(1, "test@example.com")
			},
			setupTenant: func() string {
				return "tenant-123"
			},
			setupPermissions: func() []models.Permission {
				return []models.Permission{}
			},
			setupRoles: func() []models.Role {
				role := models.Role{
					Name: "Support",
					Permissions: []models.Permission{
						{Code: "ticket:write", Name: "Write Tickets"},
					},
				}
				return []models.Role{role}
			},
			requiredPerms:  []string{"ticket:read", "ticket:write"},
			expectedStatus: http.StatusOK,
		},
		{
			name: "User has none of required permissions",
			setupUser: func() *models.User {
				return createTestUser(1, "test@example.com")
			},
			setupTenant: func() string {
				return "tenant-123"
			},
			setupPermissions: func() []models.Permission {
				return []models.Permission{
					{Code: "user:read", Name: "Read Users"},
				}
			},
			setupRoles:     func() []models.Role { return []models.Role{} },
			requiredPerms:  []string{"ticket:read", "ticket:write"},
			expectedStatus: http.StatusForbidden,
			expectedError:  "Insufficient permissions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := setupTestGin()

			// Setup user in context
			if tt.setupUser() != nil {
				c.Set("user", tt.setupUser())
				c.Set("tenant_id", tt.setupTenant())

				// Setup mock expectations
				mockService.On("GetUserPermissions", mock.Anything, tt.setupUser().ID, tt.setupTenant()).
					Return(tt.setupPermissions(), nil)
				mockService.On("GetUserRoles", mock.Anything, tt.setupUser().ID, tt.setupTenant()).
					Return(tt.setupRoles(), nil)
			}

			// Create middleware handler
			handler := middleware.RequireAnyPermission(tt.requiredPerms...)

			// Execute middleware
			handler(c)

			// Check response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				errorData, ok := response["error"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, tt.expectedError, errorData["message"])
			}

			// Verify mock expectations
			if tt.setupUser() != nil {
				mockService.AssertExpectations(t)
			}

			// Reset mock for next test
			mockService.ExpectedCalls = nil
		})
	}
}

func TestPermissionMiddleware_RequireOwnership(t *testing.T) {
	db := setupTestDB(t)
	mockService := new(MockPermissionService)
	middleware := NewPermissionMiddleware(mockService)

	// Setup mock to return test database
	mockService.On("GetDatabase").Return(db)

	tests := []struct {
		name           string
		setupUser      func() *models.User
		setupTenant    func() string
		setupData      func(t *testing.T, db *gorm.DB, user *models.User, tenantID string)
		resourceType   string
		resourceID     string
		expectedStatus int
		expectedError  string
	}{
		{
			name: "User owns ticket resource",
			setupUser: func() *models.User {
				return createTestUser(1, "test@example.com")
			},
			setupTenant: func() string {
				return "tenant-123"
			},
			setupData: func(t *testing.T, db *gorm.DB, user *models.User, tenantID string) {
				// Create a ticket owned by the user
				userIDStr := strconv.FormatUint(uint64(user.ID), 10)
				ticket := &models.Ticket{
					BaseModel:    models.BaseModel{CreatedBy: &userIDStr},
					TenantID:     1, // Simulate tenant ID conversion
					TicketNumber: "TICKET-001",
					Title:        "Test Ticket",
				}
				require.NoError(t, db.Create(ticket).Error)
			},
			resourceType:   "ticket",
			resourceID:     "1",
			expectedStatus: http.StatusOK,
		},
		{
			name: "User does not own ticket resource",
			setupUser: func() *models.User {
				return createTestUser(2, "other@example.com")
			},
			setupTenant: func() string {
				return "tenant-123"
			},
			setupData: func(t *testing.T, db *gorm.DB, user *models.User, tenantID string) {
				// Create a ticket owned by a different user
				diffUserID := "1"
				ticket := &models.Ticket{
					BaseModel:    models.BaseModel{CreatedBy: &diffUserID},
					TenantID:     1,
					TicketNumber: "TICKET-001",
					Title:        "Test Ticket",
				}
				require.NoError(t, db.Create(ticket).Error)
			},
			resourceType:   "ticket",
			resourceID:     "1",
			expectedStatus: http.StatusForbidden,
			expectedError:  "You can only access your own resources",
		},
		{
			name: "User has admin permission",
			setupUser: func() *models.User {
				return createTestUser(2, "admin@example.com")
			},
			setupTenant: func() string {
				return "tenant-123"
			},
			setupData: func(t *testing.T, db *gorm.DB, user *models.User, tenantID string) {
				// Create a ticket owned by a different user
				diffUserID := "1"
				ticket := &models.Ticket{
					BaseModel:    models.BaseModel{CreatedBy: &diffUserID},
					TenantID:     1,
					TicketNumber: "TICKET-001",
					Title:        "Test Ticket",
				}
				require.NoError(t, db.Create(ticket).Error)
			},
			resourceType:   "ticket",
			resourceID:     "1",
			expectedStatus: http.StatusOK,
		},
		{
			name: "No user in context",
			setupUser: func() *models.User {
				return nil
			},
			setupTenant:    func() string { return "tenant-123" },
			setupData:      func(t *testing.T, db *gorm.DB, user *models.User, tenantID string) {},
			resourceType:   "ticket",
			resourceID:     "1",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Authentication required",
		},
		{
			name: "Missing resource ID",
			setupUser: func() *models.User {
				return createTestUser(1, "test@example.com")
			},
			setupTenant:    func() string { return "tenant-123" },
			setupData:      func(t *testing.T, db *gorm.DB, user *models.User, tenantID string) {},
			resourceType:   "ticket",
			resourceID:     "", // Missing ID
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Resource ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := setupTestGin()

			// Set resource ID in path parameters
			if tt.resourceID != "" {
				c.Params = gin.Params{
					{Key: "id", Value: tt.resourceID},
				}
			}

			// Setup user in context
			if tt.setupUser() != nil {
				user := tt.setupUser()
				c.Set("user", user)
				c.Set("tenant_id", tt.setupTenant())

				// Setup admin permissions if this is the admin test case
				if tt.name == "User has admin permission" {
					mockService.On("GetUserPermissions", mock.Anything, user.ID, tt.setupTenant()).
						Return([]models.Permission{
							{Code: "admin:system", Name: "System Admin"},
						}, nil)
				} else {
					mockService.On("GetUserPermissions", mock.Anything, user.ID, tt.setupTenant()).
						Return([]models.Permission{}, nil)
				}

				// Setup test data
				tt.setupData(t, db, user, tt.setupTenant())
			}

			// Create middleware handler
			handler := middleware.RequireOwnership(tt.resourceType)

			// Execute middleware
			handler(c)

			// Check response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				errorData, ok := response["error"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, tt.expectedError, errorData["message"])
			}

			// Reset mock for next test
			mockService.ExpectedCalls = nil
		})
	}
}

func TestTenantMiddleware(t *testing.T) {
	tests := []struct {
		name            string
		headers         map[string]string
		expectedStatus  int
		expectedError   string
		shouldSetTenant bool
		expectedTenant  string
	}{
		{
			name: "Valid tenant ID",
			headers: map[string]string{
				"X-Tenant-ID": "tenant-123",
			},
			expectedStatus:  http.StatusOK,
			shouldSetTenant: true,
			expectedTenant:  "tenant-123",
		},
		{
			name:           "Missing tenant ID",
			headers:        map[string]string{},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Tenant ID is required",
		},
		{
			name: "Empty tenant ID",
			headers: map[string]string{
				"X-Tenant-ID": "",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Tenant ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Create request with headers
			req := httptest.NewRequest("GET", "/test", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}
			c.Request = req

			// Create middleware handler
			handler := TenantMiddleware()

			// Execute middleware
			handler(c)

			// Check response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				errorData, ok := response["error"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, tt.expectedError, errorData["message"])
			}

			if tt.shouldSetTenant {
				tenantID, exists := c.Get("tenant_id")
				assert.True(t, exists)
				assert.Equal(t, tt.expectedTenant, tenantID)
			}
		})
	}
}

func TestPermissionMiddleware_ResourceOwnershipTypes(t *testing.T) {
	db := setupTestDB(t)
	mockService := new(MockPermissionService)
	middleware := NewPermissionMiddleware(mockService)

	// Setup mock to return test database
	mockService.On("GetDatabase").Return(db)

	tests := []struct {
		name           string
		resourceType   string
		setupResource  func(t *testing.T, db *gorm.DB, user *models.User) uint
		userID         uint
		expectedStatus int
	}{
		{
			name:         "Message ownership",
			resourceType: "message",
			setupResource: func(t *testing.T, db *gorm.DB, user *models.User) uint {
				message := &models.Message{
					BaseModel: models.BaseModel{},
					TicketID:  1,
					UserID:    user.ID,
					Content:   "Test message",
				}
				require.NoError(t, db.Create(message).Error)
				return message.ID
			},
			userID:         1,
			expectedStatus: http.StatusOK,
		},
		{
			name:         "Knowledge article ownership",
			resourceType: "knowledge",
			setupResource: func(t *testing.T, db *gorm.DB, user *models.User) uint {
				article := &models.KnowledgeArticle{
					TenantID: 1,
					AuthorID: user.ID,
					Title:    "Test Article",
					Content:  "Test content",
				}
				require.NoError(t, db.Create(article).Error)
				return article.ID
			},
			userID:         1,
			expectedStatus: http.StatusOK,
		},
		{
			name:         "User profile ownership (same user)",
			resourceType: "user",
			setupResource: func(t *testing.T, db *gorm.DB, user *models.User) uint {
				return user.ID
			},
			userID:         1,
			expectedStatus: http.StatusOK,
		},
		{
			name:         "User profile ownership (different user)",
			resourceType: "user",
			setupResource: func(t *testing.T, db *gorm.DB, user *models.User) uint {
				// Create another user
				otherUser := &models.User{
					TenantID: 1,
					Email:    "other@example.com",
					Username: "otheruser",
					Role:     "customer",
					IsActive: true,
				}
				require.NoError(t, db.Create(otherUser).Error)
				return otherUser.ID
			},
			userID:         1,
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			user := createTestUser(tt.userID, "test@example.com")
			resourceID := tt.setupResource(t, db, user)

			// Setup context
			c.Set("user", user)
			c.Set("tenant_id", "1")
			c.Params = gin.Params{
				{Key: "id", Value: strconv.FormatUint(uint64(resourceID), 10)},
			}

			// Setup mock for admin permissions check (should return no admin perms)
			mockService.On("GetUserPermissions", mock.Anything, user.ID, "1").
				Return([]models.Permission{}, nil)

			// Create middleware handler
			handler := middleware.RequireOwnership(tt.resourceType)

			// Execute middleware
			handler(c)

			// Check response
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Reset mock for next test
			mockService.ExpectedCalls = nil
		})
	}
}

// Test error cases and edge conditions
func TestPermissionMiddleware_ErrorHandling(t *testing.T) {
	mockService := new(MockPermissionService)
	middleware := NewPermissionMiddleware(mockService)

	t.Run("Database error when checking permissions", func(t *testing.T) {
		c, w := setupTestGin()

		user := createTestUser(1, "test@example.com")
		c.Set("user", user)
		c.Set("tenant_id", "tenant-123")

		// Setup mock to return error
		mockService.On("GetUserPermissions", mock.Anything, user.ID, "tenant-123").
			Return([]models.Permission{}, assert.AnError)
		mockService.On("GetUserRoles", mock.Anything, user.ID, "tenant-123").
			Return([]models.Role{}, nil)

		// Create and execute middleware
		handler := middleware.RequirePermission("ticket:read")
		handler(c)

		// Check response
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		errorData, ok := response["error"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "INTERNAL_ERROR", errorData["code"])
		assert.Equal(t, "Failed to check permissions", errorData["message"])

		mockService.AssertExpectations(t)
	})

	t.Run("User type assertion error", func(t *testing.T) {
		c, _ := setupTestGin()

		// Set invalid user type in context
		c.Set("user", "invalid_user_type")
		c.Set("tenant_id", "tenant-123")

		// Create and execute middleware
		handler := middleware.RequirePermission("ticket:read")

		// This should cause a panic due to type assertion failure
		assert.Panics(t, func() {
			handler(c)
		})
	})
}
