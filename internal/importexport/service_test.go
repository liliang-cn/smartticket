package importexport

import (
	"encoding/json"
	"fmt"
	"mime/multipart"
	"testing"
	"time"

	"github.com/company/smartticket/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Migrate all models needed for import/export testing
	err = db.AutoMigrate(
		&models.Tenant{},
		&models.User{},
		&models.Ticket{},
		&models.Message{},
		&models.Attachment{},
		&models.KnowledgeArticle{},
		&models.LLMProvider{},
		&models.ImportExportJob{},
		&models.Product{},
		&models.Service{},
	)
	require.NoError(t, err)

	return db
}

// createTestTenant creates a test tenant
func createTestTenant(t *testing.T, db *gorm.DB) *models.Tenant {
	tenant := &models.Tenant{
		Name:     "Test Corporation",
		Slug:     fmt.Sprintf("test-corp-%d", time.Now().UnixNano()),
		Domain:   "test.example.com",
		Plan:     "basic",
		MaxUsers: 100,
		IsActive: true,
	}
	err := db.Create(tenant).Error
	require.NoError(t, err)
	return tenant
}

// createTestUser creates a test user
func createTestUser(t *testing.T, db *gorm.DB, tenantID uint) *models.User {
	user := &models.User{
		TenantID:     tenantID,
		Email:        fmt.Sprintf("test-%d@example.com", time.Now().UnixNano()),
		Username:     fmt.Sprintf("testuser-%d", time.Now().UnixNano()),
		FirstName:    "Test",
		LastName:     "User",
		Role:         "admin",
		PasswordHash: "hashed_password",
		IsActive:     true,
	}
	err := db.Create(user).Error
	require.NoError(t, err)
	return user
}

// createMockFileHeader creates a mock multipart file header
func createMockFileHeader(filename string, size int64) *multipart.FileHeader {
	return &multipart.FileHeader{
		Filename: filename,
		Size:     size,
	}
}

func TestNewService(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	assert.NotNil(t, service)
	assert.Equal(t, db, service.db)
}

func TestService_CreateImportJob(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	// Setup test data
	tenant := createTestTenant(t, db)
	user := createTestUser(t, db, tenant.ID)

	tests := []struct {
		name          string
		file          *multipart.FileHeader
		request       *ImportRequest
		expectedError string
		expectSuccess bool
	}{
		{
			name: "Valid import job creation",
			file: createMockFileHeader("tickets.csv", 1024),
			request: &ImportRequest{
				Type:         ExportTypeTickets,
				SourceType:   SourceZendesk,
				SourceFormat: FileTypeCSV,
				Mapping:      `{"ticket_id": "id", "title": "subject"}`,
				Options:      `{"skip_header": true}`,
			},
			expectSuccess: true,
		},
		{
			name: "File too large",
			file: createMockFileHeader("large_file.csv", 150*1024*1024), // 150MB
			request: &ImportRequest{
				Type:         ExportTypeTickets,
				SourceType:   SourceZendesk,
				SourceFormat: FileTypeCSV,
			},
			expectedError: "file size exceeds 100MB limit",
		},
		{
			name: "JSON format import",
			file: createMockFileHeader("tickets.json", 2048),
			request: &ImportRequest{
				Type:         ExportTypeUsers,
				SourceType:   SourceJira,
				SourceFormat: FileTypeJSON,
				Mapping:      `{"username": "name", "email": "email_address"}`,
			},
			expectSuccess: true,
		},
		{
			name: "XML format import",
			file: createMockFileHeader("products.xml", 3072),
			request: &ImportRequest{
				Type:         ExportTypeProducts,
				SourceType:   SourceCustom,
				SourceFormat: FileTypeXML,
			},
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job, err := service.CreateImportJob(tenant.ID, user.ID, tt.file, tt.request)

			if tt.expectSuccess {
				assert.NoError(t, err)
				assert.NotNil(t, job)
				assert.Equal(t, string(tt.request.Type), job.Type)
				assert.Equal(t, string(JobStatusPending), job.Status)
				assert.Equal(t, string(tt.request.SourceFormat), job.SourceFormat)
				assert.Equal(t, tt.file.Filename, job.FilePath)
				if job.StartedByUser != nil {
					assert.Equal(t, user.ID, job.StartedByUser.ID)
				}
				assert.NotZero(t, job.ID)

				// Verify configuration is properly built
				var config map[string]interface{}
				err = json.Unmarshal([]byte(job.Configuration), &config)
				require.NoError(t, err)
				assert.Equal(t, string(tt.request.SourceType), config["source_type"])
				assert.Equal(t, string(tt.request.SourceFormat), config["source_format"])

				// Verify mapping is included if provided
				if tt.request.Mapping != "" {
					assert.Equal(t, tt.request.Mapping, config["mapping"])
				}

				// Verify options is included if provided
				if tt.request.Options != "" {
					assert.Equal(t, tt.request.Options, config["options"])
				}
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, job)
			}
		})
	}
}

func TestService_CreateExportJob(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	// Setup test data
	tenant := createTestTenant(t, db)
	user := createTestUser(t, db, tenant.ID)

	tests := []struct {
		name          string
		request       *ExportRequest
		expectedError string
		expectSuccess bool
	}{
		{
			name: "Valid CSV export job",
			request: &ExportRequest{
				Type:         ExportTypeTickets,
				TargetFormat: FileTypeCSV,
				Filters:      `{"status": "open", "priority": "high"}`,
				Options:      `{"include_attachments": false}`,
			},
			expectSuccess: true,
		},
		{
			name: "JSON export job",
			request: &ExportRequest{
				Type:         ExportTypeUsers,
				TargetFormat: FileTypeJSON,
				Filters:      `{"role": "customer"}`,
			},
			expectSuccess: true,
		},
		{
			name: "Markdown export job",
			request: &ExportRequest{
				Type:         ExportTypeKnowledgeArticles,
				TargetFormat: FileTypeMarkdown,
				Options:      `{"include_metadata": true}`,
			},
			expectSuccess: true,
		},
		{
			name: "SQLite export job",
			request: &ExportRequest{
				Type:         ExportTypeComplete,
				TargetFormat: FileTypeSQLite,
				Options:      `{"compress": true}`,
			},
			expectSuccess: true,
		},
		{
			name: "XML export job",
			request: &ExportRequest{
				Type:         ExportTypeProducts,
				TargetFormat: FileTypeXML,
			},
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job, err := service.CreateExportJob(tenant.ID, user.ID, tt.request)

			if tt.expectSuccess {
				assert.NoError(t, err)
				assert.NotNil(t, job)
				assert.Equal(t, string(tt.request.Type), job.Type)
				assert.Equal(t, string(JobStatusPending), job.Status)
				assert.Equal(t, string(tt.request.TargetFormat), job.TargetFormat)
				if job.StartedByUser != nil {
					assert.Equal(t, user.ID, job.StartedByUser.ID)
				}
				assert.NotZero(t, job.ID)

				// Verify configuration is properly built
				var config map[string]interface{}
				err = json.Unmarshal([]byte(job.Configuration), &config)
				require.NoError(t, err)
				assert.Equal(t, string(tt.request.TargetFormat), config["target_format"])

				// Verify filters are included if provided
				if tt.request.Filters != "" {
					assert.Equal(t, tt.request.Filters, config["filters"])
				}

				// Verify options are included if provided
				if tt.request.Options != "" {
					assert.Equal(t, tt.request.Options, config["options"])
				}
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, job)
			}
		})
	}
}

func TestService_GetJob(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	// Setup test data
	tenant := createTestTenant(t, db)
	user := createTestUser(t, db, tenant.ID)

	// Create a test job
	job := &models.ImportExportJob{
		TenantID:         tenant.ID,
		Type:             string(ExportTypeTickets),
		Status:           string(JobStatusCompleted),
		Progress:         100,
		TotalRecords:     50,
		ProcessedRecords: 48,
		FailedRecords:    2,
		SourceFormat:     "csv",
		TargetFormat:     "",
		FilePath:         "test.csv",
		StartedBy:        user.ID,
	}
	err := db.Create(job).Error
	require.NoError(t, err)

	tests := []struct {
		name          string
		tenantID      uint
		jobID         uint
		expectedError string
		expectSuccess bool
	}{
		{
			name:          "Valid job retrieval",
			tenantID:      tenant.ID,
			jobID:         job.ID,
			expectSuccess: true,
		},
		{
			name:          "Job not found",
			tenantID:      tenant.ID,
			jobID:         99999,
			expectedError: "import/export job not found",
		},
		{
			name:          "Wrong tenant",
			tenantID:      99999,
			jobID:         job.ID,
			expectedError: "import/export job not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := service.GetJob(tt.tenantID, tt.jobID)

			if tt.expectSuccess {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.Equal(t, job.ID, response.ID)
				assert.Equal(t, job.Type, response.Type)
				assert.Equal(t, job.Status, response.Status)
				assert.Equal(t, job.Progress, response.Progress)
				assert.Equal(t, job.SourceFormat, response.SourceFormat)
				assert.Equal(t, job.FilePath, response.FilePath)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, response)
			}
		})
	}
}

func TestService_ListJobs(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	// Setup test data
	tenant1 := createTestTenant(t, db)
	tenant2 := createTestTenant(t, db)
	user1 := createTestUser(t, db, tenant1.ID)
	user2 := createTestUser(t, db, tenant2.ID)

	// Create test jobs for tenant 1
	for i := 0; i < 5; i++ {
		job := &models.ImportExportJob{
			TenantID:     tenant1.ID,
			Type:         string(ExportTypeTickets),
			Status:       string(JobStatusCompleted),
			Progress:     100,
			StartedBy:    user1.ID,
			SourceFormat: "csv",
		}
		err := db.Create(job).Error
		require.NoError(t, err)
	}

	// Create test jobs for tenant 2
	for i := 0; i < 3; i++ {
		job := &models.ImportExportJob{
			TenantID:     tenant2.ID,
			Type:         string(ExportTypeUsers),
			Status:       string(JobStatusPending),
			Progress:     0,
			StartedBy:    user2.ID,
			SourceFormat: "json",
		}
		err := db.Create(job).Error
		require.NoError(t, err)
	}

	tests := []struct {
		name          string
		tenantID      uint
		page          int
		pageSize      int
		filters       map[string]interface{}
		expectedCount int
		expectedTotal int64
	}{
		{
			name:          "List all jobs for tenant 1",
			tenantID:      tenant1.ID,
			page:          1,
			pageSize:      10,
			filters:       map[string]interface{}{},
			expectedCount: 5,
			expectedTotal: 5,
		},
		{
			name:          "List all jobs for tenant 2",
			tenantID:      tenant2.ID,
			page:          1,
			pageSize:      10,
			filters:       map[string]interface{}{},
			expectedCount: 3,
			expectedTotal: 3,
		},
		{
			name:          "Paginated list",
			tenantID:      tenant1.ID,
			page:          1,
			pageSize:      2,
			filters:       map[string]interface{}{},
			expectedCount: 2,
			expectedTotal: 5,
		},
		{
			name:     "Filter by type",
			tenantID: tenant1.ID,
			page:     1,
			pageSize: 10,
			filters: map[string]interface{}{
				"type": string(ExportTypeTickets),
			},
			expectedCount: 5,
			expectedTotal: 5,
		},
		{
			name:     "Filter by status",
			tenantID: tenant1.ID,
			page:     1,
			pageSize: 10,
			filters: map[string]interface{}{
				"status": string(JobStatusCompleted),
			},
			expectedCount: 5,
			expectedTotal: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := service.ListJobs(tt.tenantID, tt.page, tt.pageSize, tt.filters)

			assert.NoError(t, err)
			assert.NotNil(t, response)
			assert.Equal(t, tt.expectedCount, len(response.Data))
			assert.Equal(t, tt.expectedTotal, response.Total)
			assert.Equal(t, tt.page, response.Page)
			assert.Equal(t, tt.pageSize, response.PageSize)
		})
	}
}

func TestService_CancelJob(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	// Setup test data
	tenant := createTestTenant(t, db)
	user := createTestUser(t, db, tenant.ID)

	// Create test jobs with different statuses
	pendingJob := &models.ImportExportJob{
		TenantID:     tenant.ID,
		Type:         string(ExportTypeTickets),
		Status:       string(JobStatusPending),
		Progress:     0,
		StartedBy:    user.ID,
		SourceFormat: "csv",
	}
	err := db.Create(pendingJob).Error
	require.NoError(t, err)

	runningJob := &models.ImportExportJob{
		TenantID:     tenant.ID,
		Type:         string(ExportTypeUsers),
		Status:       string(JobStatusRunning),
		Progress:     50,
		StartedBy:    user.ID,
		SourceFormat: "json",
	}
	err = db.Create(runningJob).Error
	require.NoError(t, err)

	completedJob := &models.ImportExportJob{
		TenantID:     tenant.ID,
		Type:         string(ExportTypeProducts),
		Status:       string(JobStatusCompleted),
		Progress:     100,
		StartedBy:    user.ID,
		SourceFormat: "xml",
	}
	err = db.Create(completedJob).Error
	require.NoError(t, err)

	tests := []struct {
		name          string
		jobID         uint
		expectedError string
		expectSuccess bool
	}{
		{
			name:          "Cancel pending job",
			jobID:         pendingJob.ID,
			expectSuccess: true,
		},
		{
			name:          "Cancel running job",
			jobID:         runningJob.ID,
			expectSuccess: true,
		},
		{
			name:          "Cannot cancel completed job",
			jobID:         completedJob.ID,
			expectedError: "job cannot be cancelled in current status",
		},
		{
			name:          "Job not found",
			jobID:         99999,
			expectedError: "import/export job not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.CancelJob(tenant.ID, tt.jobID, user.ID)

			if tt.expectSuccess {
				assert.NoError(t, err)

				// Verify job status was updated
				var job models.ImportExportJob
				err = db.First(&job, tt.jobID).Error
				require.NoError(t, err)
				assert.Equal(t, string(JobStatusFailed), job.Status)
				assert.Equal(t, "Job cancelled by user", job.Error)
				assert.NotNil(t, job.CompletedAt)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

func TestService_DeleteJob(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	// Setup test data
	tenant := createTestTenant(t, db)
	user := createTestUser(t, db, tenant.ID)

	// Create a test job
	job := &models.ImportExportJob{
		TenantID:     tenant.ID,
		Type:         string(ExportTypeTickets),
		Status:       string(JobStatusCompleted),
		StartedBy:    user.ID,
		SourceFormat: "csv",
	}
	err := db.Create(job).Error
	require.NoError(t, err)

	tests := []struct {
		name          string
		tenantID      uint
		jobID         uint
		userID        uint
		expectedError string
		expectSuccess bool
	}{
		{
			name:          "Valid job deletion",
			tenantID:      tenant.ID,
			jobID:         job.ID,
			userID:        user.ID,
			expectSuccess: true,
		},
		{
			name:          "Job not found",
			tenantID:      tenant.ID,
			jobID:         99999,
			userID:        user.ID,
			expectedError: "import/export job not found",
		},
		{
			name:          "Wrong tenant",
			tenantID:      99999,
			jobID:         job.ID,
			userID:        user.ID,
			expectedError: "import/export job not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.DeleteJob(tt.tenantID, tt.jobID, tt.userID)

			if tt.expectSuccess {
				assert.NoError(t, err)

				// Verify job was soft deleted
				var job models.ImportExportJob
				err = db.Unscoped().First(&job, tt.jobID).Error
				require.NoError(t, err)
				assert.NotNil(t, job.DeletedAt)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

func TestService_ValidateFileFormat(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	tests := []struct {
		name          string
		filename      string
		expectedType  FileType
		expectError   bool
		expectedError string
	}{
		{
			name:         "Valid CSV file",
			filename:     "tickets.csv",
			expectedType: FileTypeCSV,
			expectError:  false,
		},
		{
			name:         "Valid JSON file",
			filename:     "users.json",
			expectedType: FileTypeJSON,
			expectError:  false,
		},
		{
			name:         "Valid XML file",
			filename:     "products.xml",
			expectedType: FileTypeXML,
			expectError:  false,
		},
		{
			name:         "Valid Markdown file (.md)",
			filename:     "knowledge.md",
			expectedType: FileTypeMarkdown,
			expectError:  false,
		},
		{
			name:         "Valid Markdown file (.markdown)",
			filename:     "article.markdown",
			expectedType: FileTypeMarkdown,
			expectError:  false,
		},
		{
			name:         "Valid SQLite file (.db)",
			filename:     "backup.db",
			expectedType: FileTypeSQLite,
			expectError:  false,
		},
		{
			name:         "Valid SQLite file (.sqlite)",
			filename:     "data.sqlite",
			expectedType: FileTypeSQLite,
			expectError:  false,
		},
		{
			name:         "Valid SQLite file (.sqlite3)",
			filename:     "complete.sqlite3",
			expectedType: FileTypeSQLite,
			expectError:  false,
		},
		{
			name:          "Invalid CSV file",
			filename:      "tickets.txt",
			expectedType:  FileTypeCSV,
			expectError:   true,
			expectedError: "CSV file expected",
		},
		{
			name:          "Invalid JSON file",
			filename:      "users.xml",
			expectedType:  FileTypeJSON,
			expectError:   true,
			expectedError: "JSON file expected",
		},
		{
			name:          "Invalid XML file",
			filename:      "products.csv",
			expectedType:  FileTypeXML,
			expectError:   true,
			expectedError: "XML file expected",
		},
		{
			name:          "Invalid Markdown file",
			filename:      "knowledge.txt",
			expectedType:  FileTypeMarkdown,
			expectError:   true,
			expectedError: "Markdown file expected",
		},
		{
			name:          "Invalid SQLite file",
			filename:      "backup.txt",
			expectedType:  FileTypeSQLite,
			expectError:   true,
			expectedError: "SQLite file expected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateFileFormat(tt.filename, tt.expectedType)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_GetJobStats(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	// Setup test data
	tenant := createTestTenant(t, db)
	user := createTestUser(t, db, tenant.ID)

	// Create test jobs with different statuses
	jobStatuses := []string{
		string(JobStatusPending),
		string(JobStatusRunning),
		string(JobStatusCompleted),
		string(JobStatusFailed),
	}

	jobTypes := []string{
		string(ExportTypeTickets),
		string(ExportTypeUsers),
		string(ExportTypeProducts),
	}

	// Create multiple jobs for variety
	for _, status := range jobStatuses {
		for _, jobType := range jobTypes {
			job := &models.ImportExportJob{
				TenantID:     tenant.ID,
				Type:         jobType,
				Status:       status,
				Progress:     50,
				StartedBy:    user.ID,
				SourceFormat: "csv",
			}
			err := db.Create(job).Error
			require.NoError(t, err)

			// Add some delay to differentiate timestamps
			time.Sleep(time.Millisecond)
		}
	}

	stats, err := service.GetJobStats(tenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, stats)

	// Verify status breakdown
	statusBreakdown, ok := stats["status_breakdown"].(map[string]int64)
	require.True(t, ok)
	assert.Equal(t, int64(3), statusBreakdown[string(JobStatusPending)])
	assert.Equal(t, int64(3), statusBreakdown[string(JobStatusRunning)])
	assert.Equal(t, int64(3), statusBreakdown[string(JobStatusCompleted)])
	assert.Equal(t, int64(3), statusBreakdown[string(JobStatusFailed)])

	// Verify type breakdown
	typeBreakdown, ok := stats["type_breakdown"].(map[string]int64)
	require.True(t, ok)
	assert.Equal(t, int64(4), typeBreakdown[string(ExportTypeTickets)])
	assert.Equal(t, int64(4), typeBreakdown[string(ExportTypeUsers)])
	assert.Equal(t, int64(4), typeBreakdown[string(ExportTypeProducts)])

	// Verify total count
	totalJobs, ok := stats["total_jobs"].(int64)
	require.True(t, ok)
	assert.Equal(t, int64(12), totalJobs)

	// Verify recent activity
	recentActivity, ok := stats["recent_activity"].([]models.ImportExportJob)
	require.True(t, ok)
	assert.Len(t, recentActivity, 10) // Should return max 10 recent jobs
}

func TestService_ConfigurationBuilding(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	t.Run("Build import job configuration", func(t *testing.T) {
		req := &ImportRequest{
			Type:         ExportTypeTickets,
			SourceType:   SourceZendesk,
			SourceFormat: FileTypeCSV,
			Mapping:      `{"ticket_id": "id", "title": "subject"}`,
			Options:      `{"skip_header": true}`,
		}

		config := service.buildJobConfig(req)

		var configMap map[string]interface{}
		err := json.Unmarshal([]byte(config), &configMap)
		require.NoError(t, err)

		assert.Equal(t, string(SourceZendesk), configMap["source_type"])
		assert.Equal(t, string(FileTypeCSV), configMap["source_format"])
		assert.Equal(t, `{"ticket_id": "id", "title": "subject"}`, configMap["mapping"])
		assert.Equal(t, `{"skip_header": true}`, configMap["options"])
	})

	t.Run("Build export job configuration", func(t *testing.T) {
		req := &ExportRequest{
			Type:         ExportTypeTickets,
			TargetFormat: FileTypeJSON,
			Filters:      `{"status": "open"}`,
			Options:      `{"include_attachments": false}`,
		}

		config := service.buildExportConfig(req)

		var configMap map[string]interface{}
		err := json.Unmarshal([]byte(config), &configMap)
		require.NoError(t, err)

		assert.Equal(t, string(FileTypeJSON), configMap["target_format"])
		assert.Equal(t, `{"status": "open"}`, configMap["filters"])
		assert.Equal(t, `{"include_attachments": false}`, configMap["options"])
	})

	t.Run("Build configuration with minimal data", func(t *testing.T) {
		importReq := &ImportRequest{
			Type:         ExportTypeUsers,
			SourceType:   SourceJira,
			SourceFormat: FileTypeCSV,
		}

		config := service.buildJobConfig(importReq)

		var configMap map[string]interface{}
		err := json.Unmarshal([]byte(config), &configMap)
		require.NoError(t, err)

		assert.Equal(t, string(SourceJira), configMap["source_type"])
		assert.Equal(t, string(FileTypeCSV), configMap["source_format"])
		assert.NotContains(t, configMap, "mapping")
		assert.NotContains(t, configMap, "options")

		exportReq := &ExportRequest{
			Type:         ExportTypeProducts,
			TargetFormat: FileTypeXML,
		}

		config = service.buildExportConfig(exportReq)

		err = json.Unmarshal([]byte(config), &configMap)
		require.NoError(t, err)

		assert.Equal(t, string(FileTypeXML), configMap["target_format"])
		assert.NotContains(t, configMap, "filters")
		assert.NotContains(t, configMap, "options")
	})
}

func TestService_ErrorHandling(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	tenant := createTestTenant(t, db)
	user := createTestUser(t, db, tenant.ID)

	t.Run("Database errors", func(t *testing.T) {
		// Close database to simulate errors
		sqlDB, err := db.DB()
		require.NoError(t, err)
		err = sqlDB.Close()
		require.NoError(t, err)

		file := createMockFileHeader("test.csv", 1024)
		req := &ImportRequest{
			Type:         ExportTypeTickets,
			SourceType:   SourceZendesk,
			SourceFormat: FileTypeCSV,
		}

		// This should fail due to closed database
		_, err = service.CreateImportJob(tenant.ID, user.ID, file, req)
		assert.Error(t, err)

		// Test other operations
		_, err = service.GetJob(tenant.ID, 1)
		assert.Error(t, err)

		_, err = service.ListJobs(tenant.ID, 1, 10, map[string]interface{}{})
		assert.Error(t, err)

		err = service.CancelJob(tenant.ID, 1, user.ID)
		assert.Error(t, err)

		err = service.DeleteJob(tenant.ID, 1, user.ID)
		assert.Error(t, err)

		_, err = service.GetJobStats(tenant.ID)
		assert.Error(t, err)
	})
}

// Test concurrent operations
func TestService_ConcurrentOperations(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	tenant := createTestTenant(t, db)
	user := createTestUser(t, db, tenant.ID)

	// Test concurrent job creation
	t.Run("Concurrent import job creation", func(t *testing.T) {
		const numJobs = 10
		jobs := make([]*JobResponse, numJobs)
		errs := make([]error, numJobs)

		// Use channels to coordinate concurrent operations
		done := make(chan int, numJobs)

		for i := 0; i < numJobs; i++ {
			go func(index int) {
				defer func() { done <- index }()

				file := createMockFileHeader(fmt.Sprintf("file_%d.csv", index), 1024)
				req := &ImportRequest{
					Type:         ExportTypeTickets,
					SourceType:   SourceZendesk,
					SourceFormat: FileTypeCSV,
				}

				job, err := service.CreateImportJob(tenant.ID, user.ID, file, req)
				jobs[index] = job
				errs[index] = err
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numJobs; i++ {
			<-done
		}

		// Verify all jobs were created successfully
		successCount := 0
		for i := 0; i < numJobs; i++ {
			if errs[i] == nil && jobs[i] != nil {
				successCount++
			}
		}
		assert.Equal(t, numJobs, successCount)

		// Verify all jobs have unique IDs
		ids := make(map[uint]bool)
		for _, job := range jobs {
			if job != nil {
				assert.False(t, ids[job.ID], "Duplicate job ID found")
				ids[job.ID] = true
			}
		}
	})
}

// Benchmark operations
func BenchmarkService_CreateImportJob(b *testing.B) {
	db := setupTestDB(&testing.T{})
	service := NewService(db)

	tenant := createTestTenant(&testing.T{}, db)
	user := createTestUser(&testing.T{}, db, tenant.ID)

	file := createMockFileHeader("benchmark.csv", 1024)
	req := &ImportRequest{
		Type:         ExportTypeTickets,
		SourceType:   SourceZendesk,
		SourceFormat: FileTypeCSV,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.CreateImportJob(tenant.ID, user.ID, file, req)
		if err != nil {
			b.Fatalf("Failed to create import job: %v", err)
		}
	}
}

func BenchmarkService_ListJobs(b *testing.B) {
	db := setupTestDB(&testing.T{})
	service := NewService(db)

	tenant := createTestTenant(&testing.T{}, db)
	user := createTestUser(&testing.T{}, db, tenant.ID)

	// Create some test data
	for i := 0; i < 100; i++ {
		job := &models.ImportExportJob{
			TenantID:     tenant.ID,
			Type:         string(ExportTypeTickets),
			Status:       string(JobStatusCompleted),
			StartedBy:    user.ID,
			SourceFormat: "csv",
		}
		err := db.Create(job).Error
		if err != nil {
			b.Fatalf("Failed to create test job: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.ListJobs(tenant.ID, 1, 10, map[string]interface{}{})
		if err != nil {
			b.Fatalf("Failed to list jobs: %v", err)
		}
	}
}
