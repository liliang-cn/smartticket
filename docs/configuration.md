# Configuration Management

This guide covers SmartTicket's configuration system, including environment-specific settings, validation, and best practices.

## Overview

SmartTicket uses a hierarchical configuration system based on YAML files and environment variables. Configuration is loaded in the following order of precedence (highest to lowest):

1. **Environment Variables** - Runtime overrides
2. **Configuration Files** - Environment-specific YAML files
3. **Default Values** - Built-in application defaults

## Configuration Files

### File Structure

```
configs/
├── schema.yaml              # Configuration schema/validation
├── config.local.yaml        # Local development (gitignored)
├── config.dev.yaml          # Development environment
├── config.test.yaml         # Testing environment
├── config.staging.yaml      # Staging environment
└── config.prod.yaml         # Production environment
```

### Environment-Specific Configurations

#### Development (`config.dev.yaml`)
- Debug logging enabled
- Local database and services
- Permissive CORS settings
- Hot reload support
- Development tools enabled

#### Testing (`config.test.yaml`)
- In-memory database
- Mock external services
- High rate limits
- No external dependencies

#### Staging (`config.staging.yaml`)
- Production-like settings
- Staging databases and services
- Real external APIs
- Monitoring enabled

#### Production (`config.prod.yaml`)
- Optimized for performance
- Security hardening
- Production databases and services
- Full monitoring and logging

#### Local (`config.local.yaml`)
- Personal development overrides
- **Do not commit to version control**
- Custom settings for local development
- Override any development settings

## Configuration Sections

### Core Settings

```yaml
environment: development  # Application environment

server:
  host: "localhost"      # Server bind address
  port: 6533            # Server port (non-standard)
  read_timeout: 30      # Read timeout (seconds)
  write_timeout: 30     # Write timeout (seconds)
  idle_timeout: 120     # Idle timeout (seconds)
```

### Database Configuration

```yaml
database:
  type: sqlite                          # Database type
  connection_url: "./data/smartticket.db"  # Connection URL
  max_connections: 25                   # Max connections
  max_idle_conns: 5                     # Max idle connections
  conn_max_lifetime: 3600               # Connection lifetime (seconds)
  log_level: debug                      # Database log level
  enable_wal_mode: true                 # SQLite WAL mode
  enable_foreign_keys: true             # SQLite foreign keys
```

### Authentication

```yaml
jwt:
  secret: "${JWT_SECRET}"               # JWT signing secret
  expiration_time: 1h                   # Token expiration
  refresh_time: 24h                     # Refresh token time
  issuer: "smartticket"                 # JWT issuer
```

### CORS Settings

```yaml
cors:
  allowed_origins:                      # Allowed origins
    - "http://localhost:3000"
  allowed_methods:                      # Allowed HTTP methods
    - "GET"
    - "POST"
    - "PUT"
    - "DELETE"
  allow_credentials: true               # Allow credentials
  max_age: 86400                        # Max age (seconds)
```

### Security

```yaml
security:
  enable_csrf: true                     # CSRF protection
  enable_content_type_check: true       # Content type validation
  max_request_size: 5242880            # Max request size (5MB)
  allowed_hosts:                        # Allowed hostnames
    - "localhost"
```

### Logging

```yaml
logger:
  level: debug                          # Log level
  format: json                          # Log format (json/text/console)
  output: stdout                        # Output destination
  file_path: "./logs/app.log"           # Log file path
  max_size_mb: 100                      # Max file size (MB)
  max_backups: 10                       # Max backup files
  max_age_days: 30                      # Max file age (days)
```

### Rate Limiting

```yaml
rate_limit:
  requests_per_second: 100              # Requests per second
  burst: 200                            # Burst size
```

### File Storage

```yaml
storage:
  data_path: "./data"                   # Data directory
  backup_path: "./backups"              # Backup directory
  temp_path: "./tmp"                    # Temporary directory
  max_file_size: 52428800               # Max file size (50MB)
  allowed_extensions:                   # Allowed file types
    - ".txt"
    - ".pdf"
    - ".jpg"
```

### Email Configuration

```yaml
email:
  enabled: true                         # Enable email
  smtp_host: "smtp.gmail.com"           # SMTP host
  smtp_port: 587                        # SMTP port
  username: "${SMTP_USERNAME}"          # SMTP username
  password: "${SMTP_PASSWORD}"          # SMTP password
  from_address: "noreply@smartticket.com"  # From address
  use_tls: true                         # Use TLS
```

### LLM Configuration

```yaml
llm:
  default_provider: "openai"            # Default LLM provider
  providers:
    openai:
      name: "OpenAI GPT"
      provider_type: "openai"
      api_endpoint: "https://api.openai.com/v1"
      api_key: "${OPENAI_API_KEY}"
      model: "gpt-4o-mini"
      max_tokens: 4096
      temperature: 0.7
      task_types:                       # Supported task types
        - "chat"
        - "generation"
        - "summarization"
      is_enabled: true
      quota_limit: 10000                # Usage quota
  task_mapping:                         # Task to provider mapping
    chat: "openai"
    generation: "openai"
    summarization: "openai"
```

### Feature Flags

```yaml
features:
  enable_user_registration: true        # User registration
  enable_email_verification: false      # Email verification
  enable_password_reset: true           # Password reset
  enable_ticket_exports: true           # Ticket exports
  enable_knowledge_base: true           # Knowledge base
  enable_ai_features: true              # AI features
  enable_multi_tenancy: true            # Multi-tenancy
  enable_debug_endpoints: false         # Debug endpoints
```

## Environment Variables

Environment variables override configuration file values:

```bash
# Core Application
export PORT=6533
export ENVIRONMENT=development
export LOG_LEVEL=debug

# Database
export DB_URL=./data/smartticket.db
export DB_TYPE=sqlite

# Security
export JWT_SECRET=your-super-secret-key-here

# External Services
export REDIS_URL=redis://localhost:6379
export OPENAI_API_KEY=your-openai-api-key

# Email
export SMTP_HOST=smtp.gmail.com
export SMTP_USERNAME=your-email@gmail.com
export SMTP_PASSWORD=your-app-password
```

### Environment Variable Reference

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PORT` | Server port | `6533` | No |
| `ENVIRONMENT` | Application environment | `development` | No |
| `LOG_LEVEL` | Logging level | `info` | No |
| `JWT_SECRET` | JWT signing secret | - | Yes (prod) |
| `DB_URL` | Database connection URL | - | Yes |
| `REDIS_URL` | Redis connection URL | - | No |
| `OPENAI_API_KEY` | OpenAI API key | - | No |
| `SMTP_HOST` | SMTP server host | - | No |
| `SMTP_USERNAME` | SMTP username | - | No |
| `SMTP_PASSWORD` | SMTP password | - | No |

## Configuration Loading

### Loading Priority

1. **Environment variables** (`SMARTTICKET_` prefix)
2. **Environment-specific config file** (`config.{ENV}.yaml`)
3. **Default config file** (`config.yaml`)
4. **Built-in defaults**

### Loading Process

```go
// Configuration loading logic
config := config.New()

// 1. Load defaults
config.LoadDefaults()

// 2. Load configuration file
configFile := fmt.Sprintf("configs/config.%s.yaml", environment)
config.LoadFromFile(configFile)

// 3. Override with environment variables
config.LoadFromEnv("SMARTTICKET_")

// 4. Validate configuration
if err := config.Validate(); err != nil {
    log.Fatal("Invalid configuration:", err)
}
```

### Environment Variable Naming

Environment variables use the `SMARTTICKET_` prefix and nested structure:

```yaml
# config.yaml
server:
  port: 6533
database:
  connection_url: "./data/app.db"
```

```bash
# Environment variables
export SMARTTICKET_SERVER_PORT=8080
export SMARTTICKET_DATABASE_CONNECTION_URL="./data/prod.db"
```

## Configuration Validation

### Schema Validation

The configuration is validated against the schema defined in `configs/schema.yaml`:

```bash
# Validate configuration
make config-validate

# Or using Go
go run cmd/server/main.go config-validate --config configs/config.dev.yaml
```

### Validation Rules

- **Required fields**: Must be present
- **Type validation**: Correct data types
- **Range validation**: Values within allowed ranges
- **Format validation**: Correct format (email, URL, etc.)
- **Custom validation**: Business logic validation

### Common Validation Errors

```yaml
# Invalid: Missing required field
server:
  # port field is missing

# Invalid: Type mismatch
server:
  port: "8080"  # Should be integer

# Invalid: Range violation
server:
  port: 99999   # Should be 1-65535

# Invalid: Format violation
jwt:
  secret: "short"  # Should be min 32 characters
```

## Best Practices

### Security

1. **Never commit secrets** to version control
2. **Use environment variables** for sensitive data
3. **Strong JWT secrets** (minimum 32 characters)
4. **Restrict CORS origins** in production
5. **Enable security features** in production

### Performance

1. **Use WAL mode** for SQLite
2. **Optimize connection pools**
3. **Enable compression** in production
4. **Configure appropriate timeouts**
5. **Monitor resource usage**

### Development

1. **Use local configuration** for personal settings
2. **Don't commit** `config.local.yaml`
3. **Document configuration changes**
4. **Test configuration changes**
5. **Use feature flags** for new features

### Production

1. **Environment-specific secrets**
2. **Monitoring and logging**
3. **Backup configuration**
4. **Security hardening**
5. **Performance optimization**

## Troubleshooting

### Common Issues

1. **Configuration not loading**
   ```bash
   # Check file exists and is readable
   ls -la configs/config.dev.yaml

   # Check syntax
   yamllint configs/config.dev.yaml
   ```

2. **Environment variables not working**
   ```bash
   # Check variable names
   env | grep SMARTTICKET_

   # Check prefix usage
   export SMARTTICKET_SERVER_PORT=8080
   ```

3. **Validation errors**
   ```bash
   # Validate against schema
   make config-validate

   # Check for missing fields
   go run cmd/server/main.go config-check
   ```

### Debug Mode

Enable debug logging to troubleshoot configuration:

```bash
# Enable debug logging
export LOG_LEVEL=debug

# Run with verbose output
go run cmd/server/main.go serve --verbose

# Check loaded configuration
go run cmd/server/main.go config-dump
```

## Migration Guide

### Upgrading Configuration

1. **Backup current configuration**
2. **Review new configuration options**
3. **Update configuration files**
4. **Test new configuration**
5. **Deploy changes**

### Adding New Configuration

1. **Update schema** (`configs/schema.yaml`)
2. **Add to environment configs**
3. **Update documentation**
4. **Add validation rules**
5. **Test changes**

### Removing Configuration

1. **Deprecate** first, don't remove immediately
2. **Update documentation**
3. **Remove from configs**
4. **Update schema**
5. **Clean up code**