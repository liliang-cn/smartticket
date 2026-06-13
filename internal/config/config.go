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
	Storage     StorageConfig   `mapstructure:"storage"`
	Email       EmailConfig     `mapstructure:"email"`
	Webhook     WebhookConfig   `mapstructure:"webhook"`
	// SecretKeyRaw is the raw encryption key (hex/base64) used for at-rest
	// secrets such as LLM provider API keys. Bound from SMARTTICKET_SECRET_KEY.
	SecretKeyRaw string `mapstructure:"secret_key"`
	// WidgetJSPath is the filesystem path to the compiled widget JS bundle
	// (web-widget/dist/widget.js). Defaults to "./web-widget/dist/widget.js".
	// Override via SMARTTICKET_WIDGET_JS_PATH environment variable or config file.
	WidgetJSPath string `mapstructure:"widget_js_path"`

	// App holds application-level settings.
	App AppConfig `mapstructure:"app"`
}

// AppConfig holds application-level settings.
type AppConfig struct {
	// BaseURL is the externally reachable root URL of this deployment
	// (e.g. "https://support.example.com"). It is used to construct links
	// in outbound emails (e.g. CSAT survey links). Defaults to
	// "http://localhost:6533". Set SMARTTICKET_APP_BASE_URL to override.
	BaseURL string `mapstructure:"base_url"`
}

// StorageConfig contains file (attachment) storage configuration.
type StorageConfig struct {
	DataPath          string   `mapstructure:"data_path"`
	MaxFileSize       int64    `mapstructure:"max_file_size"`
	AllowedExtensions []string `mapstructure:"allowed_extensions"`
}

// EmailConfig configures bidirectional email: outbound ticket replies and
// inbound email-to-ticket. Outbound goes through Resend (HTTP API) or any SMTP
// server; inbound arrives on a signed webhook (e.g. Resend Inbound, or any MTA
// that can POST). Disabled by default — the whole feature is opt-in.
type EmailConfig struct {
	Enabled bool `mapstructure:"enabled"`
	// Provider selects the outbound transport: "resend" (default) or "smtp".
	Provider    string        `mapstructure:"provider"`
	FromName    string        `mapstructure:"from_name"`
	FromAddress string        `mapstructure:"from_address"`
	Resend      ResendConfig  `mapstructure:"resend"`
	SMTP        SMTPConfig    `mapstructure:"smtp"`
	Inbound     InboundConfig `mapstructure:"inbound"`
	IMAP        IMAPConfig    `mapstructure:"imap"`
}

// ResendConfig holds the Resend API credentials (https://resend.com).
type ResendConfig struct {
	APIKey string `mapstructure:"api_key"`
}

// SMTPConfig is an alternative outbound transport for self-hosted mail servers.
type SMTPConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	// TLS mode: "starttls" (default, port 587), "tls" (implicit, port 465) or "none".
	TLS string `mapstructure:"tls"`
}

// InboundConfig guards the email→ticket webhook. The webhook is public (no JWT)
// so it is authenticated by a shared secret presented in the request.
type InboundConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Secret  string `mapstructure:"secret"`
}

// WebhookConfig guards outbound webhook delivery behavior.
type WebhookConfig struct {
	// BlockPrivateIPs rejects deliveries whose URL resolves to a private/loopback
	// address (SSRF guard). Default false — self-hosted setups often target intranet.
	BlockPrivateIPs bool `mapstructure:"block_private_ips"`
}

// IMAPConfig polls a mailbox over IMAP and turns new mail into tickets — the
// fully self-hosted inbound path (no webhook, no DNS, no third party).
type IMAPConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	Host        string `mapstructure:"host"`
	Port        int    `mapstructure:"port"`
	Username    string `mapstructure:"username"`
	Password    string `mapstructure:"password"`
	Mailbox     string `mapstructure:"mailbox"`
	TLS         bool   `mapstructure:"tls"`
	PollSeconds int    `mapstructure:"poll_seconds"`
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
	v.BindEnv("secret_key", "SMARTTICKET_SECRET_KEY")
	_ = v.BindEnv("app.base_url", "SMARTTICKET_APP_BASE_URL")
	// Email is most often configured via environment (secrets stay out of the
	// config file). Bind each key explicitly — viper's AutomaticEnv does not
	// reliably resolve nested keys that lack a config-file entry.
	for envKey, path := range map[string]string{
		"SMARTTICKET_EMAIL_ENABLED":           "email.enabled",
		"SMARTTICKET_EMAIL_PROVIDER":          "email.provider",
		"SMARTTICKET_EMAIL_FROM_NAME":         "email.from_name",
		"SMARTTICKET_EMAIL_FROM_ADDRESS":      "email.from_address",
		"SMARTTICKET_EMAIL_SMTP_HOST":         "email.smtp.host",
		"SMARTTICKET_EMAIL_SMTP_PORT":         "email.smtp.port",
		"SMARTTICKET_EMAIL_SMTP_USERNAME":     "email.smtp.username",
		"SMARTTICKET_EMAIL_SMTP_PASSWORD":     "email.smtp.password",
		"SMARTTICKET_EMAIL_SMTP_TLS":          "email.smtp.tls",
		"SMARTTICKET_EMAIL_INBOUND_ENABLED":   "email.inbound.enabled",
		"SMARTTICKET_EMAIL_INBOUND_SECRET":    "email.inbound.secret",
		"SMARTTICKET_EMAIL_IMAP_ENABLED":      "email.imap.enabled",
		"SMARTTICKET_EMAIL_IMAP_HOST":         "email.imap.host",
		"SMARTTICKET_EMAIL_IMAP_PORT":         "email.imap.port",
		"SMARTTICKET_EMAIL_IMAP_USERNAME":     "email.imap.username",
		"SMARTTICKET_EMAIL_IMAP_PASSWORD":     "email.imap.password",
		"SMARTTICKET_EMAIL_IMAP_MAILBOX":      "email.imap.mailbox",
		"SMARTTICKET_EMAIL_IMAP_TLS":          "email.imap.tls",
		"SMARTTICKET_EMAIL_IMAP_POLL_SECONDS": "email.imap.poll_seconds",
	} {
		_ = v.BindEnv(path, envKey)
	}
	// Resend key accepts the project-prefixed name or a bare RESEND_API_KEY.
	_ = v.BindEnv("email.resend.api_key", "SMARTTICKET_EMAIL_RESEND_API_KEY", "RESEND_API_KEY")

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

	// Email (bidirectional) defaults — feature off until configured.
	v.SetDefault("email.enabled", false)
	v.SetDefault("email.provider", "resend")
	v.SetDefault("email.from_name", "Support")
	v.SetDefault("email.smtp.port", 587)
	v.SetDefault("email.smtp.tls", "starttls")
	v.SetDefault("email.inbound.enabled", false)
	v.SetDefault("email.imap.enabled", false)
	v.SetDefault("email.imap.port", 993)
	v.SetDefault("email.imap.mailbox", "INBOX")
	v.SetDefault("email.imap.tls", true)
	v.SetDefault("email.imap.poll_seconds", 60)

	// Application defaults
	v.SetDefault("app.base_url", "http://localhost:6533")

	// Webhook delivery defaults
	v.SetDefault("webhook.block_private_ips", false)

	// Storage (attachments) defaults
	v.SetDefault("storage.data_path", "./data")
	v.SetDefault("storage.max_file_size", 20971520) // 20MB
	v.SetDefault("storage.allowed_extensions", []string{})
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
