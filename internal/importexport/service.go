package importexport

import (
	"encoding/json"
	"errors"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	apperrors "github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/models"
	"gorm.io/gorm"
)

// Service provides import/export operations.
type Service struct {
	db *gorm.DB
}

// NewService creates a new import/export service.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// JobStatus represents the status of an import/export job.
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)

// FileType represents supported file types.
type FileType string

const (
	FileTypeCSV      FileType = "csv"
	FileTypeJSON     FileType = "json"
	FileTypeXML      FileType = "xml"
	FileTypeMarkdown FileType = "markdown"
	FileTypeSQLite   FileType = "sqlite"
)

// ExportType represents export data types.
type ExportType string

const (
	ExportTypeTickets           ExportType = "tickets"
	ExportTypeKnowledgeArticles ExportType = "knowledge_articles"
	ExportTypeUsers             ExportType = "users"
	ExportTypeProducts          ExportType = "products"
	ExportTypeServices          ExportType = "services"
	ExportTypeComplete          ExportType = "complete"
)

// ThirdPartySource represents third-party system sources.
type ThirdPartySource string

const (
	SourceZendesk   ThirdPartySource = "zendesk"
	SourceJira      ThirdPartySource = "jira"
	SourceFreshdesk ThirdPartySource = "freshdesk"
	SourceCustom    ThirdPartySource = "custom"
)

// ImportRequest represents the request to create an import job.
type ImportRequest struct {
	Type         ExportType       `json:"type" binding:"required,oneof=tickets knowledge_articles users products services"`
	SourceType   ThirdPartySource `json:"source_type" binding:"required,oneof=zendesk jira freshdesk custom"`
	SourceFormat FileType         `json:"source_format" binding:"required,oneof=csv json xml"`
	Mapping      string           `json:"mapping"` // JSON mapping configuration
	Options      string           `json:"options"` // JSON options
}

// ExportRequest represents the request to create an export job.
type ExportRequest struct {
	Type         ExportType `json:"type" binding:"required,oneof=tickets knowledge_articles users products services complete"`
	TargetFormat FileType   `json:"target_format" binding:"required,oneof=csv json xml markdown sqlite"`
	Filters      string     `json:"filters"` // JSON filters
	Options      string     `json:"options"` // JSON options
}

// JobResponse represents the response for a job.
type JobResponse struct {
	ID               uint         `json:"id"`
	Type             string       `json:"type"`
	Status           string       `json:"status"`
	Progress         int          `json:"progress"`
	TotalRecords     int          `json:"total_records"`
	ProcessedRecords int          `json:"processed_records"`
	FailedRecords    int          `json:"failed_records"`
	SourceFormat     string       `json:"source_format"`
	TargetFormat     string       `json:"target_format"`
	FilePath         string       `json:"file_path"`
	Configuration    string       `json:"configuration"`
	Error            string       `json:"error,omitempty"`
	StartedAt        *time.Time   `json:"started_at,omitempty"`
	CompletedAt      *time.Time   `json:"completed_at,omitempty"`
	CreatedAt        time.Time    `json:"created_at"`
	StartedByUser    *models.User `json:"started_by_user,omitempty"`
}

// JobListResponse represents a paginated job list.
type JobListResponse struct {
	Data       []JobResponse `json:"data"`
	Total      int64         `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	TotalPages int           `json:"total_pages"`
}

// CreateImportJob creates a new import job.
func (s *Service) CreateImportJob(userID uint, file *multipart.FileHeader, req *ImportRequest) (*JobResponse, error) {
	// Validate file size (max 100MB)
	if file.Size > 100*1024*1024 {
		return nil, apperrors.NewValidationError("file size exceeds 100MB limit")
	}

	// Create import job
	job := &models.ImportExportJob{
		
		Type:             string(req.Type),
		Status:           string(JobStatusPending),
		Progress:         0,
		TotalRecords:     0,
		ProcessedRecords: 0,
		FailedRecords:    0,
		SourceFormat:     string(req.SourceFormat),
		TargetFormat:     "",
		FilePath:         file.Filename,
		Configuration:    s.buildJobConfig(req),
		StartedBy:        userID,
	}

	if err := s.db.Create(job).Error; err != nil {
		return nil, apperrors.NewInternalError("failed to create import job: %w", err)
	}

	// TODO: In a real implementation, you would:
	// 1. Save the uploaded file to a secure location
	// 2. Start a background goroutine to process the job
	// 3. Update job status and progress asynchronously

	return s.getJobResponse(job)
}

// CreateExportJob creates a new export job.
func (s *Service) CreateExportJob(userID uint, req *ExportRequest) (*JobResponse, error) {
	// Create export job
	job := &models.ImportExportJob{
		
		Type:             string(req.Type),
		Status:           string(JobStatusPending),
		Progress:         0,
		TotalRecords:     0,
		ProcessedRecords: 0,
		FailedRecords:    0,
		SourceFormat:     "",
		TargetFormat:     string(req.TargetFormat),
		FilePath:         "",
		Configuration:    s.buildExportConfig(req),
		StartedBy:        userID,
	}

	if err := s.db.Create(job).Error; err != nil {
		return nil, apperrors.NewInternalError("failed to create export job: %w", err)
	}

	// TODO: In a real implementation, you would:
	// 1. Start a background goroutine to process the export
	// 2. Query data based on export type and filters
	// 3. Generate file in requested format
	// 4. Update job status and provide download URL

	return s.getJobResponse(job)
}

// GetJob retrieves a job by ID.
func (s *Service) GetJob(jobID uint) (*JobResponse, error) {
	var job models.ImportExportJob
	if err := s.db.Where("id = ?", jobID).
		Preload("StartedByUser").
		First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFoundError("import/export job not found")
		}
		return nil, apperrors.NewInternalError("failed to retrieve job: %w", err)
	}

	return s.getJobResponse(&job)
}

// ListJobs retrieves import/export jobs with pagination.
func (s *Service) ListJobs(page, pageSize int, filters map[string]interface{}) (*JobListResponse, error) {
	offset := (page - 1) * pageSize

	// Build query
	query := s.db.Model(&models.ImportExportJob{}).
		Preload("StartedByUser")

	// Apply filters
	if jobType, ok := filters["type"]; ok {
		query = query.Where("type = ?", jobType)
	}
	if status, ok := filters["status"]; ok {
		query = query.Where("status = ?", status)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, apperrors.NewInternalError("failed to count jobs: %w", err)
	}

	// Get jobs
	var jobs []models.ImportExportJob
	if err := query.Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&jobs).Error; err != nil {
		return nil, apperrors.NewInternalError("failed to retrieve jobs: %w", err)
	}

	// Convert to response
	responses := make([]JobResponse, len(jobs))
	for i, job := range jobs {
		response, err := s.getJobResponse(&job)
		if err != nil {
			return nil, err
		}
		responses[i] = *response
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &JobListResponse{
		Data:       responses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// CancelJob cancels a running job.
func (s *Service) CancelJob(jobID uint, userID uint) error {
	var job models.ImportExportJob
	if err := s.db.Where("id = ?", jobID).
		First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.NewNotFoundError("import/export job not found")
		}
		return apperrors.NewInternalError("failed to retrieve job: %w", err)
	}

	// Can only cancel pending or running jobs
	if job.Status != string(JobStatusPending) && job.Status != string(JobStatusRunning) {
		return apperrors.NewValidationError("job cannot be cancelled in current status")
	}

	// Update job status
	if err := s.db.Model(&job).Updates(map[string]interface{}{
		"status":       string(JobStatusFailed),
		"error":        "Job cancelled by user",
		"completed_at": time.Now(),
	}).Error; err != nil {
		return apperrors.NewInternalError("failed to cancel job: %w", err)
	}

	return nil
}

// DeleteJob soft deletes a job.
func (s *Service) DeleteJob(jobID uint, userID uint) error {
	// Get user email for audit
	var user models.User
	if err := s.db.Where("id = ? AND is_active = ?", userID, true).First(&user).Error; err != nil {
		return apperrors.NewInternalError("failed to find user: %w", err)
	}

	result := s.db.Model(&models.ImportExportJob{}).
		Where("id = ?", jobID).
		Updates(map[string]interface{}{
			"deleted_at": gorm.Expr("CURRENT_TIMESTAMP"),
			"updated_by": user.Email,
		})

	if result.Error != nil {
		return apperrors.NewInternalError("failed to delete job: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return apperrors.NewNotFoundError("import/export job not found")
	}

	return nil
}

// Helper functions

func (s *Service) getJobResponse(job *models.ImportExportJob) (*JobResponse, error) {
	response := &JobResponse{
		ID:               job.ID,
		Type:             job.Type,
		Status:           job.Status,
		Progress:         job.Progress,
		TotalRecords:     job.TotalRecords,
		ProcessedRecords: job.ProcessedRecords,
		FailedRecords:    job.FailedRecords,
		SourceFormat:     job.SourceFormat,
		TargetFormat:     job.TargetFormat,
		FilePath:         job.FilePath,
		Configuration:    job.Configuration,
		Error:            job.Error,
		StartedAt:        job.StartedAt,
		CompletedAt:      job.CompletedAt,
		CreatedAt:        job.CreatedAt,
		StartedByUser:    job.StartedByUser,
	}

	return response, nil
}

func (s *Service) buildJobConfig(req *ImportRequest) string {
	config := map[string]interface{}{
		"source_type":   req.SourceType,
		"source_format": req.SourceFormat,
	}

	if req.Mapping != "" {
		config["mapping"] = req.Mapping
	}

	if req.Options != "" {
		config["options"] = req.Options
	}

	// Convert to JSON string
	configBytes, _ := json.Marshal(config)
	return string(configBytes)
}

func (s *Service) buildExportConfig(req *ExportRequest) string {
	config := map[string]interface{}{
		"target_format": req.TargetFormat,
	}

	if req.Filters != "" {
		config["filters"] = req.Filters
	}

	if req.Options != "" {
		config["options"] = req.Options
	}

	// Convert to JSON string
	configBytes, _ := json.Marshal(config)
	return string(configBytes)
}

// FormatValidation validates file format based on extension.
func (s *Service) ValidateFileFormat(filename string, expectedType FileType) error {
	ext := strings.ToLower(filepath.Ext(filename))

	switch expectedType {
	case FileTypeCSV:
		if ext != ".csv" {
			return apperrors.NewValidationError("CSV file expected")
		}
	case FileTypeJSON:
		if ext != ".json" {
			return apperrors.NewValidationError("JSON file expected")
		}
	case FileTypeXML:
		if ext != ".xml" {
			return apperrors.NewValidationError("XML file expected")
		}
	case FileTypeMarkdown:
		if ext != ".md" && ext != ".markdown" {
			return apperrors.NewValidationError("Markdown file expected")
		}
	case FileTypeSQLite:
		if ext != ".db" && ext != ".sqlite" && ext != ".sqlite3" {
			return apperrors.NewValidationError("SQLite file expected")
		}
	}

	return nil
}

// GetJobStats returns import/export job statistics.
func (s *Service) GetJobStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get job counts by status
	var statusCounts []struct {
		Status string
		Count  int64
	}
	if err := s.db.Model(&models.ImportExportJob{}).
		Select("status, count(*) as count").
		Group("status").
		Scan(&statusCounts).Error; err != nil {
		return nil, apperrors.NewInternalError("failed to get status breakdown: %w", err)
	}

	statusBreakdown := make(map[string]int64)
	for _, sc := range statusCounts {
		statusBreakdown[sc.Status] = sc.Count
	}
	stats["status_breakdown"] = statusBreakdown

	// Get job counts by type
	var typeCounts []struct {
		Type  string
		Count int64
	}
	if err := s.db.Model(&models.ImportExportJob{}).
		Select("type, count(*) as count").
		Group("type").
		Scan(&typeCounts).Error; err != nil {
		return nil, apperrors.NewInternalError("failed to get type breakdown: %w", err)
	}

	typeBreakdown := make(map[string]int64)
	for _, tc := range typeCounts {
		typeBreakdown[tc.Type] = tc.Count
	}
	stats["type_breakdown"] = typeBreakdown

	// Get total counts
	var totalJobs int64
	if err := s.db.Model(&models.ImportExportJob{}).
		Count(&totalJobs).Error; err != nil {
		return nil, apperrors.NewInternalError("failed to count total jobs: %w", err)
	}
	stats["total_jobs"] = totalJobs

	// Get recent activity (last 10 jobs)
	var recentJobs []models.ImportExportJob
	if err := s.db.Model(&models.ImportExportJob{}).
		Order("created_at DESC").
		Limit(10).
		Select("id, type, status, created_at, completed_at").
		Find(&recentJobs).Error; err != nil {
		return nil, apperrors.NewInternalError("failed to get recent activity: %w", err)
	}

	stats["recent_activity"] = recentJobs

	return stats, nil
}
