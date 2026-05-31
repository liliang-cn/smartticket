package importexport

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	sqlite "github.com/company/smartticket/internal/database/moderncsqlite"
	"github.com/company/smartticket/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupExportDB creates a shared in-memory SQLite DB scoped to the test name.
func setupExportDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.Customer{},
		&models.Ticket{},
		&models.KnowledgeArticle{},
		&models.Product{},
		&models.Service{},
		&models.ImportExportJob{},
	))
	return db
}

func TestRunExport_TicketsJSON(t *testing.T) {
	db := setupExportDB(t)
	dataPath := t.TempDir()
	svc := NewService(db, dataPath)

	user := &models.User{Email: "agent@example.com", Username: "agent", PasswordHash: "secret-hash", IsActive: true}
	require.NoError(t, db.Create(user).Error)

	require.NoError(t, db.Create(&models.Ticket{TicketNumber: "T-1", Title: "First", Status: "open", Priority: "high", Severity: "minor"}).Error)
	require.NoError(t, db.Create(&models.Ticket{TicketNumber: "T-2", Title: "Second", Status: "closed", Priority: "low", Severity: "minor"}).Error)

	job, err := svc.CreateExportJob(user.ID, &ExportRequest{Type: ExportTypeTickets, TargetFormat: FileTypeJSON})
	require.NoError(t, err)
	assert.Equal(t, string(JobStatusCompleted), job.Status)
	assert.Equal(t, 2, job.TotalRecords)
	assert.Equal(t, 100, job.Progress)
	require.NotEmpty(t, job.FilePath)

	raw, err := os.ReadFile(job.FilePath)
	require.NoError(t, err)

	var parsed []map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &parsed))
	assert.Len(t, parsed, 2)
	assert.Equal(t, "T-1", parsed[0]["ticket_number"])
	// Secrets must never appear in the export.
	assert.NotContains(t, string(raw), "secret-hash")
}

func TestRunExport_ProductsCSV(t *testing.T) {
	db := setupExportDB(t)
	svc := NewService(db, t.TempDir())

	user := &models.User{Email: "u@example.com", Username: "u", PasswordHash: "h", IsActive: true}
	require.NoError(t, db.Create(user).Error)

	require.NoError(t, db.Create(&models.Product{Name: "Alpha", Code: "A", Status: "active"}).Error)
	require.NoError(t, db.Create(&models.Product{Name: "Beta", Code: "B", Status: "active"}).Error)
	require.NoError(t, db.Create(&models.Product{Name: "Gamma", Code: "G", Status: "inactive"}).Error)

	job, err := svc.CreateExportJob(user.ID, &ExportRequest{Type: ExportTypeProducts, TargetFormat: FileTypeCSV})
	require.NoError(t, err)
	assert.Equal(t, string(JobStatusCompleted), job.Status)
	assert.Equal(t, 3, job.TotalRecords)

	raw, err := os.ReadFile(job.FilePath)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	// 1 header row + 3 data rows.
	assert.Len(t, lines, 4)
	assert.Contains(t, lines[0], "name")
	assert.Contains(t, lines[0], "code")
}

func TestRunExport_UsersExcludePasswordHash(t *testing.T) {
	db := setupExportDB(t)
	svc := NewService(db, t.TempDir())

	user := &models.User{Email: "secret@example.com", Username: "secret", PasswordHash: "TOP-SECRET-HASH", Role: "admin", IsActive: true}
	require.NoError(t, db.Create(user).Error)

	job, err := svc.CreateExportJob(user.ID, &ExportRequest{Type: ExportTypeUsers, TargetFormat: FileTypeJSON})
	require.NoError(t, err)
	raw, err := os.ReadFile(job.FilePath)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "TOP-SECRET-HASH")
	assert.Contains(t, string(raw), "secret@example.com")
}

func TestRunExport_CompleteCSVUnsupported(t *testing.T) {
	db := setupExportDB(t)
	svc := NewService(db, t.TempDir())

	user := &models.User{Email: "x@example.com", Username: "x", PasswordHash: "h", IsActive: true}
	require.NoError(t, db.Create(user).Error)

	job, err := svc.CreateExportJob(user.ID, &ExportRequest{Type: ExportTypeComplete, TargetFormat: FileTypeCSV})
	require.Error(t, err)
	assert.Nil(t, job)
	assert.Contains(t, err.Error(), "complete export only supports json/sqlite")

	// The persisted job row should be marked failed.
	var rows []models.ImportExportJob
	require.NoError(t, db.Order("id DESC").Find(&rows).Error)
	require.NotEmpty(t, rows)
	assert.Equal(t, string(JobStatusFailed), rows[0].Status)
}

func TestRunExport_CompleteJSONBundle(t *testing.T) {
	db := setupExportDB(t)
	svc := NewService(db, t.TempDir())

	user := &models.User{Email: "c@example.com", Username: "c", PasswordHash: "h", IsActive: true}
	require.NoError(t, db.Create(user).Error)
	require.NoError(t, db.Create(&models.Ticket{TicketNumber: "T-9", Title: "Bundle", Status: "open", Priority: "low", Severity: "minor"}).Error)
	require.NoError(t, db.Create(&models.Product{Name: "P", Code: "P", Status: "active"}).Error)

	job, err := svc.CreateExportJob(user.ID, &ExportRequest{Type: ExportTypeComplete, TargetFormat: FileTypeJSON})
	require.NoError(t, err)
	assert.Equal(t, string(JobStatusCompleted), job.Status)

	raw, err := os.ReadFile(job.FilePath)
	require.NoError(t, err)
	var bundle map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &bundle))
	assert.Contains(t, bundle, "tickets")
	assert.Contains(t, bundle, "knowledge_articles")
	assert.Contains(t, bundle, "users")
	assert.Contains(t, bundle, "products")
	assert.Contains(t, bundle, "services")
}
