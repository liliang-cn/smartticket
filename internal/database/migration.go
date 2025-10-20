package database

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Migration represents a database migration
type Migration struct {
	ID          uint   `gorm:"primaryKey"`
	Version     string `gorm:"uniqueIndex;not null"`
	Name        string `gorm:"not null"`
	Applied     bool   `gorm:"not null;default:false"`
	AppliedAt   *time.Time
	Description string
	Checksum    string `gorm:"not null"`
}

// Migrator handles database migrations
type Migrator struct {
	db *gorm.DB
}

// NewMigrator creates a new migrator instance
func NewMigrator(db *gorm.DB) *Migrator {
	return &Migrator{db: db}
}

// AutoMigrate runs GORM auto-migration for all models
func (m *Migrator) AutoMigrate(models ...interface{}) error {
	for _, model := range models {
		if err := m.db.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to auto-migrate model %T: %w", model, err)
		}
	}
	return nil
}

// CreateMigrationTable creates the migrations tracking table
func (m *Migrator) CreateMigrationTable() error {
	return m.db.AutoMigrate(&Migration{})
}

// GetAppliedMigrations returns all applied migrations
func (m *Migrator) GetAppliedMigrations() ([]Migration, error) {
	var migrations []Migration
	if err := m.db.Where("applied = ?", true).Order("version").Find(&migrations).Error; err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}
	return migrations, nil
}

// GetPendingMigrations returns migrations that haven't been applied
func (m *Migrator) GetPendingMigrations(migrationsDir string) ([]string, error) {
	// Get migration files from filesystem
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Extract migration versions from filenames
	var migrationFiles []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		if strings.HasSuffix(name, ".sql") {
			version := strings.TrimSuffix(name, ".sql")
			migrationFiles = append(migrationFiles, version)
		}
	}

	// Sort migration files by version
	sort.Strings(migrationFiles)

	// Get applied migrations
	applied, err := m.GetAppliedMigrations()
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Create map of applied versions
	appliedMap := make(map[string]bool)
	for _, migration := range applied {
		appliedMap[migration.Version] = true
	}

	// Filter out applied migrations
	var pending []string
	for _, version := range migrationFiles {
		if !appliedMap[version] {
			pending = append(pending, version)
		}
	}

	return pending, nil
}

// RunMigration executes a single migration
func (m *Migrator) RunMigration(migrationsDir, version string) error {
	migrationFile := filepath.Join(migrationsDir, version+".sql")

	// Read migration SQL
	sqlBytes, err := os.ReadFile(migrationFile)
	if err != nil {
		return fmt.Errorf("failed to read migration file %s: %w", migrationFile, err)
	}
	sql := string(sqlBytes)

	// Calculate checksum
	checksum := calculateChecksum(sql)

	// Start transaction
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Execute migration SQL
	if err := tx.Exec(sql).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to execute migration SQL for %s: %w", version, err)
	}

	// Record migration
	migration := Migration{
		Version:     version,
		Name:        extractMigrationName(sql),
		Applied:     true,
		AppliedAt:   &time.Time{},
		Description: extractMigrationDescription(sql),
		Checksum:    checksum,
	}
	*migration.AppliedAt = time.Now()

	if err := tx.Create(&migration).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to record migration %s: %w", version, err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit migration %s: %w", version, err)
	}

	return nil
}

// RunMigrations executes all pending migrations
func (m *Migrator) RunMigrations(migrationsDir string) error {
	// Ensure migrations table exists
	if err := m.CreateMigrationTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get pending migrations
	pending, err := m.GetPendingMigrations(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to get pending migrations: %w", err)
	}

	if len(pending) == 0 {
		fmt.Println("No pending migrations")
		return nil
	}

	fmt.Printf("Running %d migrations...\n", len(pending))

	// Run each pending migration
	for _, version := range pending {
		fmt.Printf("Running migration: %s\n", version)
		if err := m.RunMigration(migrationsDir, version); err != nil {
			return fmt.Errorf("failed to run migration %s: %w", version, err)
		}
		fmt.Printf("Migration completed: %s\n", version)
	}

	fmt.Println("All migrations completed successfully")
	return nil
}

// RollbackMigration rolls back the last applied migration
func (m *Migrator) RollbackMigration(migrationsDir string) error {
	// Get the last applied migration
	var migration Migration
	if err := m.db.Where("applied = ?", true).Order("version desc").First(&migration).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("no migrations to rollback")
		}
		return fmt.Errorf("failed to get last migration: %w", err)
	}

	// Check if rollback file exists
	rollbackFile := filepath.Join(migrationsDir, migration.Version+"_rollback.sql")
	if _, err := os.Stat(rollbackFile); os.IsNotExist(err) {
		return fmt.Errorf("rollback file not found for migration %s", migration.Version)
	}

	// Read rollback SQL
	sqlBytes, err := os.ReadFile(rollbackFile)
	if err != nil {
		return fmt.Errorf("failed to read rollback file %s: %w", rollbackFile, err)
	}
	sql := string(sqlBytes)

	// Start transaction
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Execute rollback SQL
	if err := tx.Exec(sql).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to execute rollback SQL for %s: %w", migration.Version, err)
	}

	// Mark migration as not applied
	if err := tx.Model(&migration).Update("applied", false).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update migration record %s: %w", migration.Version, err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit rollback %s: %w", migration.Version, err)
	}

	return nil
}

// GetMigrationStatus returns the status of all migrations
func (m *Migrator) GetMigrationStatus(migrationsDir string) ([]MigrationStatus, error) {
	// Get all migration files
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []MigrationStatus{}, nil
		}
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Extract migration versions
	var allVersions []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		if strings.HasSuffix(name, ".sql") && !strings.HasSuffix(name, "_rollback.sql") {
			version := strings.TrimSuffix(name, ".sql")
			allVersions = append(allVersions, version)
		}
	}

	// Sort versions
	sort.Strings(allVersions)

	// Get applied migrations
	applied, err := m.GetAppliedMigrations()
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Create map of applied migrations
	appliedMap := make(map[string]Migration)
	for _, migration := range applied {
		appliedMap[migration.Version] = migration
	}

	// Build status
	var status []MigrationStatus
	for _, version := range allVersions {
		applied, exists := appliedMap[version]
		if exists {
			status = append(status, MigrationStatus{
				Version:     version,
				Name:        applied.Name,
				Applied:     true,
				AppliedAt:   applied.AppliedAt,
				Description: applied.Description,
			})
		} else {
			status = append(status, MigrationStatus{
				Version:     version,
				Name:        extractMigrationNameFromFile(filepath.Join(migrationsDir, version+".sql")),
				Applied:     false,
				AppliedAt:   nil,
				Description: extractMigrationDescriptionFromFile(filepath.Join(migrationsDir, version+".sql")),
			})
		}
	}

	return status, nil
}

// MigrationStatus represents the status of a migration
type MigrationStatus struct {
	Version     string
	Name        string
	Applied     bool
	AppliedAt   *time.Time
	Description string
}

// Helper functions

func calculateChecksum(sql string) string {
	// Simple checksum implementation - in production, use proper hashing
	return fmt.Sprintf("%x", len(sql))
}

func extractMigrationName(sql string) string {
	lines := strings.Split(sql, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "--") {
			return strings.TrimPrefix(line, "--")
		}
	}
	return "Unknown"
}

func extractMigrationDescription(sql string) string {
	lines := strings.Split(sql, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "-- Description:") {
			return strings.TrimPrefix(line, "-- Description:")
		}
	}
	return ""
}

func extractMigrationNameFromFile(filename string) string {
	content, err := os.ReadFile(filename)
	if err != nil {
		return "Unknown"
	}
	return extractMigrationName(string(content))
}

func extractMigrationDescriptionFromFile(filename string) string {
	content, err := os.ReadFile(filename)
	if err != nil {
		return ""
	}
	return extractMigrationDescription(string(content))
}
