package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Config represents the application configuration.
type Config struct {
	Environment string          `mapstructure:"environment" validate:"required,oneof=development test production"`
	Server      ServerConfig    `mapstructure:"server" validate:"required"`
	Database    DatabaseConfig  `mapstructure:"database" validate:"required"`
	JWT         JWTConfig       `mapstructure:"jwt" validate:"required"`
	CORS        CORSConfig      `mapstructure:"cors" validate:"required"`
	Logger      LoggerConfig    `mapstructure:"logger" validate:"required"`
	RateLimit   RateLimitConfig `mapstructure:"rate_limit" validate:"required"`
	LLM         LLMConfig       `mapstructure:"llm"`
}

// ServerConfig contains HTTP server configuration.
type ServerConfig struct {
	Host           string `mapstructure:"host" validate:"required"`
	Port           int    `mapstructure:"port" validate:"required,min=1,max=65535"`
	ReadTimeout    int    `mapstructure:"read_timeout" validate:"required,min=1,max=300"`
	WriteTimeout   int    `mapstructure:"write_timeout" validate:"required,min=1,max=300"`
	IdleTimeout    int    `mapstructure:"idle_timeout" validate:"required,min=1,max=600"`
	MaxHeaderBytes int    `mapstructure:"max_header_bytes" validate:"required,min=1024,max=1048576"`
}

// DatabaseConfig contains database configuration.
type DatabaseConfig struct {
	Type            string `mapstructure:"type" validate:"required,oneof=sqlite"`
	ConnectionURL   string `mapstructure:"connection_url" validate:"required"`
	MaxConnections  int    `mapstructure:"max_connections" validate:"min=1,max=100"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns" validate:"min=1,max=50"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime" validate:"min=60,max=7200"`
	LogLevel        string `mapstructure:"log_level" validate:"required,oneof=silent error warn info debug"`
}

// JWTConfig contains JWT authentication configuration.
type JWTConfig struct {
	Secret               string        `mapstructure:"secret" validate:"required,min=32"`
	ExpirationTime       time.Duration `mapstructure:"expiration_time" validate:"required,min=1m,max=24h"`
	RefreshTime          time.Duration `mapstructure:"refresh_time" validate:"required,min=1h,max=168h"`
	Issuer               string        `mapstructure:"issuer" validate:"required"`
	AccessTokenDuration  time.Duration `mapstructure:"access_token_duration"`
	RefreshTokenDuration time.Duration `mapstructure:"refresh_token_duration"`
}

// CORSConfig contains CORS configuration.
type CORSConfig struct {
	AllowedOrigins   []string `mapstructure:"allowed_origins" validate:"required,min=1"`
	AllowedMethods   []string `mapstructure:"allowed_methods" validate:"required,min=1"`
	ExposedHeaders   []string `mapstructure:"exposed_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
	MaxAge           int      `mapstructure:"max_age" validate:"min=1,max=86400"`
}

// LoggerConfig contains logging configuration.
type LoggerConfig struct {
	Level      string `mapstructure:"level" validate:"required,oneof=debug info warn error"`
	Format     string `mapstructure:"format" validate:"required,oneof=json text"`
	Output     string `mapstructure:"output" validate:"required,oneof=stdout file"`
	FilePath   string `mapstructure:"file_path"`
	MaxSizeMB  int    `mapstructure:"max_size_mb" validate:"min=1,max=1024"`
	MaxBackups int    `mapstructure:"max_backups" validate:"min=1,max=30"`
	MaxAgeDays int    `mapstructure:"max_age_days" validate:"min=1,max=365"`
}

// RateLimitConfig contains rate limiting configuration.
type RateLimitConfig struct {
	RequestsPerSecond int `mapstructure:"requests_per_second" validate:"required,min=1,max=1000"`
	Burst             int `mapstructure:"burst" validate:"required,min=1,max=1000"`
}

// LLMConfig contains LLM provider configuration.
type LLMConfig struct {
	DefaultProvider string                 `mapstructure:"default_provider"`
	Providers       map[string]LLMProvider `mapstructure:"providers"`
	TaskMapping     map[string]string      `mapstructure:"task_mapping"`
	RateLimit       LLMRateLimitConfig     `mapstructure:"rate_limit"`
}

type LLMProvider struct {
	Name         string   `mapstructure:"name" validate:"required"`
	ProviderType string   `mapstructure:"provider_type" validate:"required,oneof=openai azure anthropic deepseek ollama local"`
	APIEndpoint  string   `mapstructure:"api_endpoint"`
	APIKey       string   `mapstructure:"api_key"`
	Model        string   `mapstructure:"model"`
	MaxTokens    int      `mapstructure:"max_tokens" validate:"min=1,max=32000"`
	Temperature  float64  `mapstructure:"temperature" validate:"min=0,max=2"`
	TaskTypes    []string `mapstructure:"task_types"`
	IsDefault    bool     `mapstructure:"is_default"`
	IsEnabled    bool     `mapstructure:"is_enabled"`
	QuotaLimit   int      `mapstructure:"quota_limit"`
	QuotaUsed    int      `mapstructure:"quota_used"`
}

type LLMRateLimitConfig struct {
	RequestsPerMinute int `mapstructure:"requests_per_minute" validate:"min=1,max=1000"`
	TokensPerMinute   int `mapstructure:"tokens_per_minute" validate:"min=1000,max=100000"`
}

// Load loads configuration from the default search paths and environment
// variables.
func Load() (*Config, error) {
	return loadFrom("")
}

// loadFrom loads configuration. When configFile is non-empty it is read
// directly; otherwise the default search paths are used. Environment variables
// always override file values.
func loadFrom(configFile string) (*Config, error) {
	v := viper.New()

	// Set environment variable prefix
	v.SetEnvPrefix("SMARTTICKET")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if configFile != "" {
		// Use the explicitly provided config file.
		v.SetConfigFile(configFile)
	} else {
		// Set config file search paths
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath("$HOME/.smartticket")
		v.AddConfigPath("/etc/smartticket")
	}

	// Try to read config file
	if err := v.ReadInConfig(); err != nil {
		var configNotFoundErr viper.ConfigFileNotFoundError
		if errors.As(err, &configNotFoundErr) {
			// Config file not found, use defaults and environment variables
			fmt.Println("Config file not found, using environment variables and defaults")
		} else {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Set default values
	setDefaults(v)

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Override specific settings based on environment
	applyEnvironmentOverrides(&config)

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}

// LoadFromFlags loads configuration with command line flag overrides.
func LoadFromFlags(cmd *cobra.Command) (*Config, error) {
	configPath, _ := cmd.Flags().GetString("config")
	return loadFrom(configPath)
}

// setDefaults sets default configuration values.
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.host", "localhost")
	v.SetDefault("server.port", 6533)
	v.SetDefault("server.read_timeout", 30)
	v.SetDefault("server.write_timeout", 30)
	v.SetDefault("server.idle_timeout", 120)
	v.SetDefault("server.max_header_bytes", 1048576)

	// Database defaults
	v.SetDefault("database.type", "sqlite")
	v.SetDefault("database.connection_url", "./data/smartticket.db")
	v.SetDefault("database.max_connections", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", 3600)
	v.SetDefault("database.log_level", "warn")

	// JWT defaults
	v.SetDefault("jwt.secret", "default-secret-key-32-characters-long-for-validation")
	v.SetDefault("jwt.expiration_time", "24h")
	v.SetDefault("jwt.refresh_time", "168h")
	v.SetDefault("jwt.access_token_duration", "24h")
	v.SetDefault("jwt.refresh_token_duration", "168h")
	v.SetDefault("jwt.issuer", "smartticket")

	// CORS defaults
	v.SetDefault("cors.allowed_origins", []string{"http://localhost:3000", "http://localhost:7218"})
	v.SetDefault("cors.allowed_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	v.SetDefault("cors.exposed_headers", []string{"X-Total-Count", "X-Request-ID"})
	v.SetDefault("cors.allow_credentials", true)
	v.SetDefault("cors.max_age", 86400)

	// Logger defaults
	v.SetDefault("logger.level", "info")
	v.SetDefault("logger.format", "json")
	v.SetDefault("logger.output", "stdout")
	v.SetDefault("logger.max_size_mb", 100)
	v.SetDefault("logger.max_backups", 5)
	v.SetDefault("logger.max_age_days", 30)

	// Rate limiting defaults
	v.SetDefault("rate_limit.requests_per_second", 100)
	v.SetDefault("rate_limit.burst", 200)

	// LLM defaults
	v.SetDefault("llm.default_provider", "openai")
	v.SetDefault("llm.rate_limit.requests_per_minute", 100)
	v.SetDefault("llm.rate_limit.tokens_per_minute", 10000)
}

// applyEnvironmentOverrides applies environment-specific overrides.
func applyEnvironmentOverrides(config *Config) {
	switch config.Environment {
	case "development":
		if config.Logger.Level == "" {
			config.Logger.Level = "debug"
		}
		if config.Database.LogLevel == "" {
			config.Database.LogLevel = "debug"
		}
	case "production":
		if config.Logger.Level == "" {
			config.Logger.Level = "info"
		}
		if config.Database.LogLevel == "" {
			config.Database.LogLevel = "error"
		}
		// Production-specific security settings
		if config.JWT.Secret == "default-secret-change-in-production" {
			// This should be overridden by environment variable
			fmt.Println("WARNING: Using default JWT secret in production environment!")
		}
	case "test":
		if config.Logger.Level == "" {
			config.Logger.Level = "debug"
		}
		if config.Database.LogLevel == "" {
			config.Database.LogLevel = "silent"
		}
		// Use in-memory database for testing
		if config.Database.ConnectionURL == "./data/smartticket.db" {
			config.Database.ConnectionURL = ":memory:"
		}
	}
}

// validateConfig validates the configuration.
func validateConfig(config *Config) error {
	// Validate server configuration
	if config.Server.Port < 1 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	// Validate database configuration
	if config.Database.Type != "sqlite" {
		return fmt.Errorf("unsupported database type: %s", config.Database.Type)
	}

	// Validate JWT configuration
	if len(config.JWT.Secret) < 32 {
		return fmt.Errorf("JWT secret must be at least 32 characters")
	}

	// Validate file paths for output configurations
	if config.Logger.Output == "file" && config.Logger.FilePath == "" {
		return fmt.Errorf("file path is required when output is 'file'")
	}

	return nil
}

// IsDevelopment checks if running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction checks if running in production mode.
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// IsTest checks if running in test mode.
func (c *Config) IsTest() bool {
	return c.Environment == "test"
}

// GetDatabasePath returns the database file path.
func (c *Config) GetDatabasePath() string {
	return c.Database.ConnectionURL
}

// GetServerAddress returns the server address.
func (c *Config) GetServerAddress() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}
