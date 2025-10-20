package importexport

import (
	"net/http"
	"strconv"

	"github.com/company/smartticket/internal/errors"
	"github.com/gin-gonic/gin"
)

// Handlers handles import/export-related HTTP requests
type Handlers struct {
	service *Service
}

// NewHandlers creates new import/export handlers
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// parseJobID extracts and validates job ID from request parameters
func (h *Handlers) parseJobID(c *gin.Context) (uint, error) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErr := errors.NewInvalidInputError("job_id", c.Param("id"))
		errors.ErrorHandler(c, appErr)
		return 0, err
	}
	return uint(jobID), nil
}

// logSecurityEvent logs a security event with target resource
func (h *Handlers) logSecurityEvent(c *gin.Context, event, target string) {
	c.Set("security_event", event)
	c.Set("target_resource", target)
}

// CreateImportJob creates a new import job
func (h *Handlers) CreateImportJob(c *gin.Context) {
	// Get tenant ID and user ID from context
	tenantID := c.GetUint("tenant_id")
	userID := c.GetUint("user_id")

	// Parse multipart form (max 100MB)
	if err := c.Request.ParseMultipartForm(100 << 20); err != nil {
		appErr := errors.NewValidationError("failed to parse form: " + err.Error())
		errors.ErrorHandler(c, appErr)
		return
	}

	// Get uploaded file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		appErr := errors.NewInvalidInputError("file", "file upload required")
		errors.ErrorHandler(c, appErr)
		return
	}
	defer func() {
		_ = file.Close()
	}()

	// Parse import configuration
	var req ImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := errors.NewInvalidInputError("request_body", err.Error())
		errors.ErrorHandler(c, appErr)
		return
	}

	// Validate file format
	if err := h.service.ValidateFileFormat(header.Filename, req.SourceFormat); err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	// Log import job creation attempt
	c.Set("security_event", "import_job_creation_attempt")
	c.Set("target_resource", header.Filename)

	// Create import job
	job, err := h.service.CreateImportJob(tenantID, userID, header, &req)
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	// Log successful creation
	c.Set("security_event", "import_job_created")
	c.Set("target_resource", job.Type)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    job,
	})
}

// CreateExportJob creates a new export job
func (h *Handlers) CreateExportJob(c *gin.Context) {
	// Get tenant ID and user ID from context
	tenantID := c.GetUint("tenant_id")
	userID := c.GetUint("user_id")

	// Parse request
	var req ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := errors.NewInvalidInputError("request_body", err.Error())
		errors.ErrorHandler(c, appErr)
		return
	}

	// Log export job creation attempt
	c.Set("security_event", "export_job_creation_attempt")
	c.Set("target_resource", req.Type)

	// Create export job
	job, err := h.service.CreateExportJob(tenantID, userID, &req)
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	// Log successful creation
	c.Set("security_event", "export_job_created")
	c.Set("target_resource", job.Type)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    job,
	})
}

// GetImportExportJob retrieves an import/export job by ID
func (h *Handlers) GetImportExportJob(c *gin.Context) {
	// Parse job ID
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErr := errors.NewInvalidInputError("job_id", c.Param("id"))
		errors.ErrorHandler(c, appErr)
		return
	}

	// Get tenant ID from context
	tenantID := c.GetUint("tenant_id")

	// Get job
	job, err := h.service.GetJob(tenantID, uint(jobID))
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    job,
	})
}

// ListImportExportJobs retrieves import/export jobs with pagination and filtering
func (h *Handlers) ListImportExportJobs(c *gin.Context) {
	// Get tenant ID from context
	tenantID := c.GetUint("tenant_id")

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Parse filters
	filters := make(map[string]interface{})
	if jobType := c.Query("type"); jobType != "" {
		filters["type"] = jobType
	}
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}

	// Get jobs
	result, err := h.service.ListJobs(tenantID, page, pageSize, filters)
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result.Data,
		"meta": gin.H{
			"total":       result.Total,
			"page":        result.Page,
			"page_size":   result.PageSize,
			"total_pages": result.TotalPages,
		},
	})
}

// CancelImportExportJob cancels a running import/export job
func (h *Handlers) CancelImportExportJob(c *gin.Context) {
	// Parse job ID
	jobID, err := h.parseJobID(c)
	if err != nil {
		return
	}

	// Get tenant ID and user ID from context
	tenantID := c.GetUint("tenant_id")
	userID := c.GetUint("user_id")

	// Log job cancellation attempt
	h.logSecurityEvent(c, "import_export_job_cancellation_attempt", strconv.FormatUint(uint64(jobID), 10))

	// Cancel job
	err = h.service.CancelJob(tenantID, jobID, userID)
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	// Log successful cancellation
	h.logSecurityEvent(c, "import_export_job_cancelled", strconv.FormatUint(uint64(jobID), 10))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Import/export job cancelled successfully",
	})
}

// DeleteImportExportJob soft deletes an import/export job
func (h *Handlers) DeleteImportExportJob(c *gin.Context) {
	// Parse job ID
	jobID, err := h.parseJobID(c)
	if err != nil {
		return
	}

	// Get tenant ID and user ID from context
	tenantID := c.GetUint("tenant_id")
	userID := c.GetUint("user_id")

	// Log job deletion attempt
	h.logSecurityEvent(c, "import_export_job_deletion_attempt", strconv.FormatUint(uint64(jobID), 10))

	// Delete job
	err = h.service.DeleteJob(tenantID, jobID, userID)
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	// Log successful deletion
	h.logSecurityEvent(c, "import_export_job_deleted", strconv.FormatUint(uint64(jobID), 10))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Import/export job deleted successfully",
	})
}

// GetImportExportStats retrieves import/export job statistics
func (h *Handlers) GetImportExportStats(c *gin.Context) {
	// Get tenant ID from context
	tenantID := c.GetUint("tenant_id")

	// Get statistics
	stats, err := h.service.GetJobStats(tenantID)
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// DownloadExportFile downloads an exported file
func (h *Handlers) DownloadExportFile(c *gin.Context) {
	// Parse job ID
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErr := errors.NewInvalidInputError("job_id", c.Param("id"))
		errors.ErrorHandler(c, appErr)
		return
	}

	// Get tenant ID from context
	tenantID := c.GetUint("tenant_id")

	// Get job
	job, err := h.service.GetJob(tenantID, uint(jobID))
	if err != nil {
		errors.ErrorHandler(c, err)
		return
	}

	// Check if job is completed and has a file
	if job.Status != "completed" {
		appErr := errors.NewValidationError("export job is not completed")
		errors.ErrorHandler(c, appErr)
		return
	}

	if job.FilePath == "" {
		appErr := errors.NewNotFoundError("export file not found")
		errors.ErrorHandler(c, appErr)
		return
	}

	// TODO: In a real implementation, you would:
	// 1. Serve the actual file from secure storage
	// 2. Set appropriate Content-Type header
	// 3. Set Content-Disposition for download
	// 4. Log the download for audit

	// For now, return a placeholder response
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "File download functionality not yet implemented",
		"file_path": job.FilePath,
	})
}

// GetImportTemplate returns an import template for the specified type
func (h *Handlers) GetImportTemplate(c *gin.Context) {
	// Get import type from query parameter
	importType := c.Query("type")
	sourceFormat := c.DefaultQuery("format", "csv")

	if importType == "" {
		appErr := errors.NewInvalidInputError("type", "import type is required")
		errors.ErrorHandler(c, appErr)
		return
	}

	// Validate import type
	validTypes := map[string]bool{
		"tickets":            true,
		"knowledge_articles": true,
		"users":              true,
		"products":           true,
		"services":           true,
	}
	if !validTypes[importType] {
		appErr := errors.NewValidationError("invalid import type")
		errors.ErrorHandler(c, appErr)
		return
	}

	// TODO: In a real implementation, you would:
	// 1. Generate appropriate template based on type and format
	// 2. Return the template file for download
	// 3. Include field mappings and example data

	// For now, return a placeholder response
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Import template functionality not yet implemented",
		"type":    importType,
		"format":  sourceFormat,
	})
}
