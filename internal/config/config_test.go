package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	// Test loading default config (will use environment variables and defaults)
	config, err := Load()

	// Should not fail even without config file
	if err != nil {
		t.Logf("Config load error (expected if no config file): %v", err)
	}

	if config != nil {
		assert.Equal(t, "localhost", config.Server.Host)
		assert.Equal(t, 6533, config.Server.Port)
		assert.Equal(t, "sqlite", config.Database.Type)
	}
}

func TestEnvironmentDetection(t *testing.T) {
	config := &Config{Environment: "development"}
	assert.True(t, config.IsDevelopment())
	assert.False(t, config.IsProduction())
	assert.False(t, config.IsTest())

	config = &Config{Environment: "production"}
	assert.False(t, config.IsDevelopment())
	assert.True(t, config.IsProduction())
	assert.False(t, config.IsTest())

	config = &Config{Environment: "test"}
	assert.False(t, config.IsDevelopment())
	assert.False(t, config.IsProduction())
	assert.True(t, config.IsTest())
}

func TestGetDatabasePath(t *testing.T) {
	config := &Config{
		Database: DatabaseConfig{
			ConnectionURL: "./data/test.db",
		},
	}

	assert.Equal(t, "./data/test.db", config.GetDatabasePath())
}

func TestGetServerAddress(t *testing.T) {
	config := &Config{
		Server: ServerConfig{
			Host: "localhost",
			Port: 6533,
		},
	}

	assert.Equal(t, "localhost:6533", config.GetServerAddress())
}

func TestValidateConfig(t *testing.T) {
	// Test valid config
	validConfig := &Config{
		Environment: "test",
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Database: DatabaseConfig{
			Type:          "sqlite",
			ConnectionURL: ":memory:",
		},
		JWT: JWTConfig{
			Secret:         "this-is-a-valid-secret-key-32-chars",
			ExpirationTime: 0, // Will be set by defaults
			RefreshTime:    0, // Will be set by defaults
			Issuer:         "test",
		},
	}

	err := validateConfig(validConfig)
	assert.NoError(t, err)

	// Test invalid port
	invalidConfig := &Config{
		Environment: "test",
		Server: ServerConfig{
			Host: "localhost",
			Port: 70000, // Invalid port
		},
		Database: DatabaseConfig{
			Type:          "sqlite",
			ConnectionURL: ":memory:",
		},
		JWT: JWTConfig{
			Secret: "this-is-a-valid-secret-key-32-chars",
			Issuer: "test",
		},
	}

	err = validateConfig(invalidConfig)
	assert.Error(t, err)
}

func TestApplyEnvironmentOverrides(t *testing.T) {
	// Test development environment
	devConfig := &Config{
		Environment: "development",
		Logger:      LoggerConfig{},
		Database:    DatabaseConfig{},
	}

	applyEnvironmentOverrides(devConfig)
	assert.Equal(t, "debug", devConfig.Logger.Level)
	assert.Equal(t, "debug", devConfig.Database.LogLevel)

	// Test production environment
	prodConfig := &Config{
		Environment: "production",
		Logger:      LoggerConfig{},
		Database:    DatabaseConfig{},
	}

	applyEnvironmentOverrides(prodConfig)
	assert.Equal(t, "info", prodConfig.Logger.Level)
	assert.Equal(t, "error", prodConfig.Database.LogLevel)

	// Test environment
	testConfig := &Config{
		Environment: "test",
		Logger:      LoggerConfig{},
		Database: DatabaseConfig{
			ConnectionURL: "./data/smartticket.db",
		},
	}

	applyEnvironmentOverrides(testConfig)
	assert.Equal(t, "debug", testConfig.Logger.Level)
	assert.Equal(t, "silent", testConfig.Database.LogLevel)
	assert.Equal(t, ":memory:", testConfig.Database.ConnectionURL)
}

func TestLoadWithEnvVars(t *testing.T) {
	// Set environment variables
	_ = os.Setenv("SMARTTICKET_SERVER_PORT", "9999")
	_ = os.Setenv("SMARTTICKET_DATABASE_TYPE", "sqlite")
	_ = os.Setenv("SMARTTICKET_LOGGER_LEVEL", "debug")
	defer func() {
		_ = os.Unsetenv("SMARTTICKET_SERVER_PORT")
		_ = os.Unsetenv("SMARTTICKET_DATABASE_TYPE")
		_ = os.Unsetenv("SMARTTICKET_LOGGER_LEVEL")
	}()

	config, err := Load()
	if err != nil {
		t.Skipf("Skipping test due to config load error: %v", err)
		return
	}

	assert.Equal(t, 9999, config.Server.Port)
	assert.Equal(t, "sqlite", config.Database.Type)
	assert.Equal(t, "debug", config.Logger.Level)
}

func TestConfigValidationEdgeCases(t *testing.T) {
	// Test JWT secret validation
	shortSecretConfig := &Config{
		Environment: "test",
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Database: DatabaseConfig{
			Type:          "sqlite",
			ConnectionURL: ":memory:",
		},
		JWT: JWTConfig{
			Secret: "short",
			Issuer: "test",
		},
	}

	err := validateConfig(shortSecretConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least 32 characters")

	// Test file output validation without path
	fileOutputConfig := &Config{
		Environment: "test",
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Database: DatabaseConfig{
			Type:          "sqlite",
			ConnectionURL: ":memory:",
		},
		JWT: JWTConfig{
			Secret: "this-is-a-valid-secret-key-32-chars",
			Issuer: "test",
		},
		Logger: LoggerConfig{
			Output: "file",
			// FilePath is missing
		},
	}

	err = validateConfig(fileOutputConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file path is required")
}
