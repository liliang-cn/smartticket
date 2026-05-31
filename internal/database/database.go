package database

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	sqlite "github.com/company/smartticket/internal/database/moderncsqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/company/smartticket/internal/config"
)

// Database wraps the GORM database connection.
type Database struct {
	*gorm.DB
	config *config.DatabaseConfig
}

// NewDatabase creates a new database connection.
func NewDatabase(cfg *config.DatabaseConfig) (*Database, error) {
	// Validate database configuration
	if err := validateDatabaseConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid database configuration: %w", err)
	}

	// Ensure database directory exists
	dbPath := cfg.ConnectionURL
	if cfg.Type == "sqlite" && dbPath != ":memory:" {
		dbDir := filepath.Dir(dbPath)
		if err := config.ValidateDirectory(dbDir, "database"); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	// Configure GORM logger
	var gormLogger logger.Interface
	switch cfg.LogLevel {
	case "silent":
		gormLogger = logger.Default.LogMode(logger.Silent)
	case "error":
		gormLogger = logger.Default.LogMode(logger.Error)
	case "warn":
		gormLogger = logger.Default.LogMode(logger.Warn)
	case "info":
		gormLogger = logger.Default.LogMode(logger.Info)
	case "debug":
		gormLogger = logger.Default.LogMode(logger.Info) // GORM doesn't have debug level
	default:
		gormLogger = logger.Default.LogMode(logger.Error)
	}

	// Open database connection
	var db *gorm.DB
	var err error

	switch cfg.Type {
	case "sqlite":
		db, err = openSQLite(dbPath, gormLogger)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxConnections)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)

	// Test the connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{
		DB:     db,
		config: cfg,
	}

	return database, nil
}

// openSQLite creates a new SQLite database connection (pure-Go modernc driver).
func openSQLite(dsn string, gormLogger logger.Interface) (*gorm.DB, error) {
	// modernc uses _pragma=NAME(value) syntax. Foreign keys are disabled during
	// migration (re-enabled by EnableForeignKeys) to avoid constraint churn.
	// Only wrap a bare filesystem path: leave :memory:, existing file: URIs and
	// any DSN that already carries query params untouched.
	if dsn != ":memory:" && !strings.HasPrefix(dsn, "file:") && !strings.Contains(dsn, "?") {
		dsn = fmt.Sprintf(
			"file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(0)&_pragma=busy_timeout(5000)",
			dsn,
		)
	}

	return gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:      gormLogger,
		PrepareStmt: true,
	})
}

// Close closes the database connection.
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	return sqlDB.Close()
}

// GetConfig returns the database configuration.
func (d *Database) GetConfig() *config.DatabaseConfig {
	return d.config
}

// EnableForeignKeys enables foreign key constraints (should be called after migrations).
func (d *Database) EnableForeignKeys() error {
	if d.config.Type != "sqlite" {
		return nil // Only relevant for SQLite
	}

	return d.DB.Exec("PRAGMA foreign_keys = ON").Error
}

// DisableForeignKeys disables foreign key constraints (useful for migrations).
func (d *Database) DisableForeignKeys() error {
	if d.config.Type != "sqlite" {
		return nil // Only relevant for SQLite
	}

	return d.DB.Exec("PRAGMA foreign_keys = OFF").Error
}

// GetDB returns the underlying GORM database instance.
func (d *Database) GetDB() *gorm.DB {
	return d.DB
}

// IsHealthy checks if the database connection is healthy.
func (d *Database) IsHealthy() bool {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return false
	}

	if err := sqlDB.Ping(); err != nil {
		return false
	}

	return true
}

// Stats returns database connection statistics.
func (d *Database) Stats() map[string]interface{} {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	stats := sqlDB.Stats()
	return map[string]interface{}{
		"max_open_connections": stats.MaxOpenConnections,
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"wait_count":           stats.WaitCount,
		"wait_duration":        stats.WaitDuration,
		"max_idle_closed":      stats.MaxIdleClosed,
		"max_idle_time_closed": stats.MaxIdleTimeClosed,
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
	}
}

// validateDatabaseConfig validates the database configuration.
func validateDatabaseConfig(cfg *config.DatabaseConfig) error {
	if cfg == nil {
		return fmt.Errorf("database configuration is required")
	}

	// Validate database type
	if cfg.Type == "" {
		return fmt.Errorf("database type is required")
	}

	// Validate connection URL
	if cfg.ConnectionURL == "" {
		return fmt.Errorf("database connection URL is required")
	}

	// Validate connection limits
	if cfg.MaxConnections <= 0 {
		return fmt.Errorf("max connections must be greater than 0")
	}

	if cfg.MaxIdleConns < 0 {
		return fmt.Errorf("max idle connections cannot be negative")
	}

	if cfg.MaxIdleConns > cfg.MaxConnections {
		return fmt.Errorf("max idle connections cannot exceed max connections")
	}

	// Validate connection lifetime
	if cfg.ConnMaxLifetime <= 0 {
		return fmt.Errorf("connection max lifetime must be greater than 0")
	}

	return nil
}

// Backup creates a backup of the SQLite database.
func (d *Database) Backup(backupPath string) error {
	if d.config.Type != "sqlite" {
		return fmt.Errorf("backup is only supported for SQLite databases")
	}

	// Ensure backup directory exists
	backupDir := filepath.Dir(backupPath)
	if err := config.ValidateDirectory(backupDir, "backup"); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Get database connection info
	_, err := d.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Use SQLite backup API
	sourceDB := d.config.ConnectionURL
	if sourceDB == ":memory:" {
		return fmt.Errorf("backup not supported for in-memory databases")
	}

	// Copy the database file
	sourceFile, err := os.Open(sourceDB)
	if err != nil {
		return fmt.Errorf("failed to open source database file: %w", err)
	}
	defer func() {
		_ = sourceFile.Close()
	}()

	backupFile, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer func() {
		_ = backupFile.Close()
	}()

	buf := make([]byte, 64*1024) // 64KB buffer
	for {
		n, err := sourceFile.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read from source database: %w", err)
		}
		if n == 0 {
			break
		}
		if _, err := backupFile.Write(buf[:n]); err != nil {
			return fmt.Errorf("failed to write to backup file: %w", err)
		}
	}

	// Sync the backup file to ensure data is written to disk
	if err := backupFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync backup file: %w", err)
	}

	return nil
}
