package testutils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/company/smartticket/internal/config"
	"github.com/company/smartticket/internal/database"
	"github.com/company/smartticket/internal/models"
)

// WithTestDatabase creates a temporary test database and executes the test function.
func WithTestDatabase(t *testing.T, testFunc func(t *testing.T, db *database.Database)) {
	t.Helper()

	// Create temporary database file
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Create database config
	cfg := &config.DatabaseConfig{
		Type:            "sqlite",
		ConnectionURL:   dbPath,
		MaxConnections:  10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 3600,
		LogLevel:        "error",
	}

	// Initialize database
	db, err := database.NewDatabase(cfg)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("Failed to close test database: %v", err)
		}
		// Clean up database file
		_ = os.Remove(dbPath)
	}()

	// Run GORM auto-migration for all models
	if err := db.DB.AutoMigrate(
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
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Execute test function
	testFunc(t, db)
}
