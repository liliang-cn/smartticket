package testutils

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/company/smartticket/internal/config"
	"github.com/company/smartticket/internal/database"
	"github.com/company/smartticket/internal/models"
)

// TestDatabase provides a database connection for testing
type TestDatabase struct {
	*database.Database
	tempDir string
}

// NewTestDatabase creates a new test database with isolated storage
func NewTestDatabase(t *testing.T) *TestDatabase {
	// Create temporary directory for test database
	tempDir, err := os.MkdirTemp("", "smartticket_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Create test database configuration
	dbPath := filepath.Join(tempDir, "test.db")
	cfg := &config.DatabaseConfig{
		Type:            "sqlite",
		ConnectionURL:   dbPath,
		MaxConnections:  10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 3600,     // 1 hour
		LogLevel:        "silent", // Keep logs clean during tests
	}

	// Create database connection
	db, err := database.NewDatabase(cfg)
	if err != nil {
		func() { _ = os.RemoveAll(tempDir) }() // Clean up on failure
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Verify database is healthy
	if !db.IsHealthy() {
		func() { _ = db.Close() }()
		func() { _ = os.RemoveAll(tempDir) }() // Clean up on failure
		t.Fatalf("Test database is not healthy")
	}

	return &TestDatabase{
		Database: db,
		tempDir:  tempDir,
	}
}

// Close closes the test database and cleans up temporary files
func (td *TestDatabase) Close() error {
	// Close database connection
	if err := td.Database.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	// Clean up temporary directory
	if err := os.RemoveAll(td.tempDir); err != nil {
		return fmt.Errorf("failed to clean up temp directory: %w", err)
	}

	return nil
}

// CreateTestDatabaseConfig creates a database configuration for testing
func CreateTestDatabaseConfig(dbPath string) *config.DatabaseConfig {
	return &config.DatabaseConfig{
		Type:            "sqlite",
		ConnectionURL:   dbPath,
		MaxConnections:  10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 3600,
		LogLevel:        "silent",
	}
}

// WithTestDatabase is a helper function that runs a test function with a test database
func WithTestDatabase(t *testing.T, testFunc func(*testing.T, *database.Database)) {
	td := NewTestDatabase(t)
	defer func() {
		if err := td.Close(); err != nil {
			t.Errorf("Failed to close test database: %v", err)
		}
	}()

	// Run auto-migration for all models
	migrator := database.NewMigrator(td.DB)

	// Import all models and auto-migrate them
	if err := migrator.AutoMigrate(
		&models.Tenant{},
		&models.User{},
		&models.Ticket{},
		&models.Message{},
		&models.Attachment{},
		&models.KnowledgeArticle{},
		&models.LLMProvider{},
		&models.ImportExportJob{},
		&models.AuditLog{},
		&models.APIKey{},
		&models.SystemSetting{},
		&models.Product{},
		&models.Service{},
		&models.SLATemplate{},
		&models.SLARule{},
		&models.Permission{},
		&models.Role{},
		&models.RolePermission{},
		&models.UserPermission{},
		&models.UserRole{},
	); err != nil {
		t.Fatalf("Failed to auto-migrate database: %v", err)
	}

	testFunc(t, td.Database)
}

// WithTransaction runs a test function within a database transaction
func WithTransaction(t *testing.T, db *database.Database, testFunc func(*testing.T, *database.Database) error) {
	// Begin transaction
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("Failed to begin transaction: %v", tx.Error)
	}

	// Create transaction database wrapper
	txDB := &database.Database{DB: tx}

	// Defer rollback or commit
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r) // Re-panic after rollback
		} else {
			if err := testFunc(t, txDB); err != nil {
				tx.Rollback()
				t.Errorf("Test function failed: %v", err)
			} else {
				if err := tx.Commit().Error; err != nil {
					t.Errorf("Failed to commit transaction: %v", err)
				}
			}
		}
	}()
}
