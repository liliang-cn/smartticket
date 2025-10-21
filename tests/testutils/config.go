package testutils

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/company/smartticket/internal/config"
)

// TestConfig provides configuration for testing.
type TestConfig struct {
	*config.Config
	tempDir string
}

// NewTestConfig creates a new test configuration.
func NewTestConfig(t *testing.T) *TestConfig {
	// Create temporary directory for test configuration
	tempDir, err := os.MkdirTemp("", "smartticket_config_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp config directory: %v", err)
	}

	// Create test configuration
	cfg := &config.Config{
		Environment: "test",
		Server: config.ServerConfig{
			Host:           "localhost",
			Port:           0, // Random port for testing
			ReadTimeout:    30,
			WriteTimeout:   30,
			IdleTimeout:    60,
			MaxHeaderBytes: 1048576,
		},
		Database: config.DatabaseConfig{
			Type:            "sqlite",
			ConnectionURL:   filepath.Join(tempDir, "test.db"),
			MaxConnections:  10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 3600,
			LogLevel:        "silent",
		},
		Logger: config.LoggerConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			MaxSizeMB:  100,
			MaxBackups: 3,
			MaxAgeDays: 28,
		},
		JWT: config.JWTConfig{
			Secret:         "test-secret-key-for-testing-only-32-chars",
			ExpirationTime: 1 * time.Hour,
			RefreshTime:    24 * time.Hour,
			Issuer:         "smartticket-test",
		},
		CORS: config.CORSConfig{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			ExposedHeaders:   []string{"X-Total-Count", "X-Request-ID"},
			AllowCredentials: true,
			MaxAge:           86400,
		},
		RateLimit: config.RateLimitConfig{
			RequestsPerSecond: 1000,
			Burst:             2000,
		},
	}

	return &TestConfig{
		Config:  cfg,
		tempDir: tempDir,
	}
}

// Close cleans up test configuration.
func (tc *TestConfig) Close() error {
	return os.RemoveAll(tc.tempDir)
}

// CreateTestConfigFile creates a test configuration file.
func CreateTestConfigFile(t *testing.T, cfg *config.Config) string {
	tempDir, err := os.MkdirTemp("", "smartticket_config_file_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp config file directory: %v", err)
	}

	configPath := filepath.Join(tempDir, "config.test.yaml")

	// For testing purposes, we'll use the in-memory config
	// In a real implementation, you would marshal the config to YAML
	t.Cleanup(func() {
		_ = os.RemoveAll(tempDir)
	})

	return configPath
}

// WithTestConfig is a helper function that runs a test function with test configuration.
func WithTestConfig(t *testing.T, testFunc func(*testing.T, *config.Config)) {
	tc := NewTestConfig(t)
	defer func() {
		if err := tc.Close(); err != nil {
			t.Errorf("Failed to close test config: %v", err)
		}
	}()

	testFunc(t, tc.Config)
}
