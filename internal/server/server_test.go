package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/company/smartticket/internal/config"
	"github.com/company/smartticket/internal/database"
)

// createTestConfig creates a test configuration.
func createTestConfig() *config.Config {
	return &config.Config{
		Environment: "test",
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 0, // Random port for testing
		},
		Database: config.DatabaseConfig{
			Type:          "sqlite",
			ConnectionURL: ":memory:",
		},
		Logger: config.LoggerConfig{
			Level: "debug",
		},
		RateLimit: config.RateLimitConfig{
			RequestsPerSecond: 1000, // High limit for testing
			Burst:             2000,
		},
	}
}

// createTestDB creates a test database connection.
func createTestDB(t *testing.T) *database.Database {
	cfg := &config.DatabaseConfig{
		Type:            "sqlite",
		ConnectionURL:   ":memory:",
		MaxConnections:  1,
		MaxIdleConns:    1,
		ConnMaxLifetime: 3600,
		LogLevel:        "silent",
	}

	db, err := database.NewDatabase(cfg)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	return db
}

func TestNewServer(t *testing.T) {
	cfg := createTestConfig()
	db := createTestDB(t)
	defer func() {
		_ = db.Close()
	}()

	srv := NewServer(cfg, db)

	assert.NotNil(t, srv)
	assert.Equal(t, cfg, srv.config)
	assert.Equal(t, db, srv.db)
	assert.NotNil(t, srv.router)
}

func TestServerSetupMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := createTestConfig()
	db := createTestDB(t)
	defer func() {
		_ = db.Close()
	}()

	srv := NewServer(cfg, db)

	// Verify that middleware is set up by checking router has middleware
	assert.NotNil(t, srv.router)
	// In test mode, Gin should have some middleware
	assert.True(t, len(srv.router.Handlers) > 0)
}

func TestServerSetupRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := createTestConfig()
	db := createTestDB(t)
	defer func() {
		_ = db.Close()
	}()

	srv := NewServer(cfg, db)

	// Test that health endpoint exists
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	srv.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Test alternative health endpoint
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/health", nil)
	srv.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestServerGetRouter(t *testing.T) {
	cfg := createTestConfig()
	db := createTestDB(t)
	defer func() {
		_ = db.Close()
	}()

	srv := NewServer(cfg, db)
	router := srv.GetRouter()

	assert.NotNil(t, router)
	assert.IsType(t, &gin.Engine{}, router)
}

func TestServerGetConfig(t *testing.T) {
	cfg := createTestConfig()
	db := createTestDB(t)
	defer func() {
		_ = db.Close()
	}()

	srv := NewServer(cfg, db)
	retrievedConfig := srv.GetConfig()

	assert.Equal(t, cfg, retrievedConfig)
}

func TestServerGetDB(t *testing.T) {
	cfg := createTestConfig()
	db := createTestDB(t)
	defer func() {
		_ = db.Close()
	}()

	srv := NewServer(cfg, db)
	retrievedDB := srv.GetDB()

	assert.Equal(t, db, retrievedDB)
}

func TestServerRequestIDMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := createTestConfig()
	db := createTestDB(t)
	defer func() {
		_ = db.Close()
	}()

	srv := NewServer(cfg, db)

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)

	// Manually trigger the request ID middleware
	srv.requestIDMiddleware()(c)

	// Check that request ID is set
	requestID, exists := c.Get("request_id")
	assert.True(t, exists)
	assert.NotEmpty(t, requestID)
	assert.True(t, len(requestID.(string)) > 10) // Request ID should be reasonable length
}

func TestServerErrorHandling(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := createTestConfig()
	db := createTestDB(t)
	defer func() {
		_ = db.Close()
	}()

	srv := NewServer(cfg, db)

	// Test 404 handler
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/non-existent", nil)
	srv.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestServerCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := createTestConfig()
	cfg.CORS = config.CORSConfig{
		AllowedOrigins: []string{"http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST"},
		ExposedHeaders: []string{"Content-Type"},
	}

	db := createTestDB(t)
	defer func() {
		_ = db.Close()
	}()

	srv := NewServer(cfg, db)

	// Test CORS preflight
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/api/v1/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")

	srv.router.ServeHTTP(w, req)

	// Should handle OPTIONS request
	assert.True(t, w.Code == http.StatusNoContent || w.Code == http.StatusOK)
}

func TestServerConfiguration(t *testing.T) {
	cfg := &config.Config{
		Environment: "production",
		Server: config.ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Database: config.DatabaseConfig{
			Type:          "sqlite",
			ConnectionURL: "file::memory:?cache=shared",
		},
		Logger: config.LoggerConfig{
			Level:  "warn",
			Format: "json",
		},
	}

	db := createTestDB(t)
	defer func() {
		_ = db.Close()
	}()

	srv := NewServer(cfg, db)

	assert.Equal(t, cfg, srv.config)
	assert.False(t, srv.config.IsDevelopment())
	assert.True(t, srv.config.IsProduction())
}

func TestServerGracefulShutdown(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := createTestConfig()
	db := createTestDB(t)
	defer func() {
		_ = db.Close()
	}()

	srv := NewServer(cfg, db)

	// Test graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := srv.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestServerHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := createTestConfig()
	db := createTestDB(t)
	defer func() {
		_ = db.Close()
	}()

	srv := NewServer(cfg, db)

	// Test health check endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	srv.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"ok"`)
}

func TestServerVersionInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := createTestConfig()
	db := createTestDB(t)
	defer func() {
		_ = db.Close()
	}()

	srv := NewServer(cfg, db)

	// Test version info endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/version", nil)
	srv.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "version")
}
