package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	apperrors "github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/importexport"
)

// ieJob is the MCP-specific output view of an import/export job. It mirrors the
// scalar fields of importexport.JobResponse but deliberately omits the embedded
// *models.User (StartedByUser): that struct transitively references models.Product,
// which the SDK's JSON-schema reflector rejects as a cycle. The acting user is
// surfaced as a numeric ID instead.
type ieJob struct {
	ID               uint       `json:"id" jsonschema:"the job's numeric ID"`
	Type             string     `json:"type" jsonschema:"data type: tickets, knowledge_articles, users, products, services, complete"`
	Status           string     `json:"status" jsonschema:"job status: pending, running, completed, failed"`
	Progress         int        `json:"progress" jsonschema:"completion percentage (0-100)"`
	TotalRecords     int        `json:"total_records" jsonschema:"total number of records to process"`
	ProcessedRecords int        `json:"processed_records" jsonschema:"number of records processed so far"`
	FailedRecords    int        `json:"failed_records" jsonschema:"number of records that failed to process"`
	SourceFormat     string     `json:"source_format,omitempty" jsonschema:"source file format (import jobs)"`
	TargetFormat     string     `json:"target_format,omitempty" jsonschema:"output file format (export jobs)"`
	FilePath         string     `json:"file_path,omitempty" jsonschema:"path or filename associated with the job"`
	Configuration    string     `json:"configuration,omitempty" jsonschema:"job configuration as a JSON string"`
	Error            string     `json:"error,omitempty" jsonschema:"error message if the job failed"`
	StartedAt        *time.Time `json:"started_at,omitempty" jsonschema:"when processing started"`
	CompletedAt      *time.Time `json:"completed_at,omitempty" jsonschema:"when the job completed"`
	CreatedAt        time.Time  `json:"created_at" jsonschema:"when the job was created"`
	StartedByUserID  uint       `json:"started_by_user_id,omitempty" jsonschema:"numeric ID of the user who started the job"`
}

// ieJobFrom converts a service-layer importexport.JobResponse into the MCP view,
// flattening the embedded user to its numeric ID.
func ieJobFrom(r *importexport.JobResponse) ieJob {
	job := ieJob{
		ID:               r.ID,
		Type:             r.Type,
		Status:           r.Status,
		Progress:         r.Progress,
		TotalRecords:     r.TotalRecords,
		ProcessedRecords: r.ProcessedRecords,
		FailedRecords:    r.FailedRecords,
		SourceFormat:     r.SourceFormat,
		TargetFormat:     r.TargetFormat,
		FilePath:         r.FilePath,
		Configuration:    r.Configuration,
		Error:            r.Error,
		StartedAt:        r.StartedAt,
		CompletedAt:      r.CompletedAt,
		CreatedAt:        r.CreatedAt,
	}
	if r.StartedByUser != nil {
		job.StartedByUserID = r.StartedByUser.ID
	}
	return job
}

// registerImportExportTools registers the import/export-domain MCP tools.
// See server.go for the tool implementation conventions and auth_whoami template.
//
// All identifiers in this file are prefixed with "ie" to avoid collisions with
// the other tools_*.go files that share this package.
func registerImportExportTools(s *mcp.Server, b Backend) {
	registerTool(s,
		"import_create",
		"Create a data import job from a third-party source (Zendesk, Jira, Freshdesk, or custom).",
		"importexport:write",
		func(ctx context.Context, in ieImportCreateInput) (ieJob, string, error) {
			return ieImportCreate(ctx, b, in)
		},
	)

	registerTool(s,
		"export_create",
		"Create a data export job (tickets, knowledge articles, users, products, services, or a complete export).",
		"importexport:write",
		func(ctx context.Context, in ieExportCreateInput) (ieJob, string, error) {
			return ieExportCreate(ctx, b, in)
		},
	)

	registerTool(s,
		"importexport_job_get",
		"Fetch a single import/export job by its numeric ID.",
		"importexport:read",
		func(ctx context.Context, in ieJobGetInput) (ieJob, string, error) {
			return ieJobGet(ctx, b, in)
		},
	)

	registerTool(s,
		"importexport_job_list",
		"List import/export jobs with optional filtering and pagination.",
		"importexport:read",
		func(ctx context.Context, in ieJobListInput) (ieJobListOutput, string, error) {
			return ieJobList(ctx, b, in)
		},
	)

	registerTool(s,
		"importexport_job_cancel",
		"Cancel a pending or running import/export job.",
		"importexport:write",
		func(ctx context.Context, in ieJobCancelInput) (ieJobCancelOutput, string, error) {
			return ieJobCancel(ctx, b, in)
		},
	)

	registerTool(s,
		"importexport_job_delete",
		"Soft-delete an import/export job by its numeric ID.",
		"importexport:write",
		func(ctx context.Context, in ieJobDeleteInput) (ieJobDeleteOutput, string, error) {
			return ieJobDelete(ctx, b, in)
		},
	)

	registerTool(s,
		"importexport_job_stats",
		"Return aggregate import/export job statistics (counts by status, type, recent activity).",
		"importexport:read",
		func(ctx context.Context, in ieJobStatsInput) (ieJobStatsOutput, string, error) {
			return ieJobStats(ctx, b, in)
		},
	)
}

// ----------------------------------------------------------------------------
// import_create
// ----------------------------------------------------------------------------

// ieImportCreateInput is the MCP input schema for import_create.
type ieImportCreateInput struct {
	Type         string `json:"type" jsonschema:"data type to import: one of tickets, knowledge_articles, users, products, services"`
	SourceType   string `json:"source_type" jsonschema:"third-party source: one of zendesk, jira, freshdesk, custom"`
	SourceFormat string `json:"source_format" jsonschema:"source file format: one of csv, json, xml"`
	Mapping      string `json:"mapping,omitempty" jsonschema:"optional field-mapping configuration as a JSON object string"`
	Options      string `json:"options,omitempty" jsonschema:"optional import options as a JSON object string"`
}

// ieImportCreate creates an import job on behalf of the acting session user.
//
// The underlying service requires an uploaded *multipart.FileHeader, which
// cannot be synthesized or transmitted cleanly over the MCP transport. Rather
// than mangle the service or fabricate a bogus file handle, this tool returns a
// clear, structured error directing callers to the REST upload endpoint. The
// session is still validated so unauthenticated callers get the standard auth
// error.
func ieImportCreate(ctx context.Context, _ Backend, _ ieImportCreateInput) (ieJob, string, error) {
	session, ok := SessionFromContext(ctx)
	if !ok || session == nil {
		return ieJob{}, "", ErrUnauthenticated
	}

	return ieJob{}, "", apperrors.NewForbiddenError(
		"file import is not supported over MCP; use the REST upload endpoint to create an import job",
	)
}

// ----------------------------------------------------------------------------
// export_create
// ----------------------------------------------------------------------------

// ieExportCreateInput is the MCP input schema for export_create.
type ieExportCreateInput struct {
	Type         string `json:"type" jsonschema:"data type to export: one of tickets, knowledge_articles, users, products, services, complete"`
	TargetFormat string `json:"target_format" jsonschema:"output file format: one of csv, json, xml, markdown, sqlite"`
	Filters      string `json:"filters,omitempty" jsonschema:"optional export filters as a JSON object string"`
	Options      string `json:"options,omitempty" jsonschema:"optional export options as a JSON object string"`
}

// ieExportCreate creates an export job on behalf of the acting session user.
func ieExportCreate(ctx context.Context, b Backend, in ieExportCreateInput) (ieJob, string, error) {
	session, ok := SessionFromContext(ctx)
	if !ok || session == nil {
		return ieJob{}, "", ErrUnauthenticated
	}

	req := &importexport.ExportRequest{
		Type:         importexport.ExportType(in.Type),
		TargetFormat: importexport.FileType(in.TargetFormat),
		Filters:      in.Filters,
		Options:      in.Options,
	}

	resp, err := b.CreateExportJob(session.UserID, req)
	if err != nil {
		return ieJob{}, "", err
	}
	return ieJobFrom(resp), fmt.Sprintf("created export job #%d (type %s, format %s)", resp.ID, resp.Type, resp.TargetFormat), nil
}

// ----------------------------------------------------------------------------
// importexport_job_get
// ----------------------------------------------------------------------------

// ieJobGetInput is the MCP input schema for importexport_job_get.
type ieJobGetInput struct {
	ID uint `json:"id" jsonschema:"the numeric ID of the import/export job to fetch"`
}

// ieJobGet fetches a single import/export job by ID.
func ieJobGet(_ context.Context, b Backend, in ieJobGetInput) (ieJob, string, error) {
	resp, err := b.GetJob(in.ID)
	if err != nil {
		return ieJob{}, "", err
	}
	return ieJobFrom(resp), fmt.Sprintf("fetched job #%d (%s, status %s)", resp.ID, resp.Type, resp.Status), nil
}

// ----------------------------------------------------------------------------
// importexport_job_list
// ----------------------------------------------------------------------------

// ieJobListInput is the MCP input schema for importexport_job_list.
type ieJobListInput struct {
	Page     int    `json:"page,omitempty" jsonschema:"1-based page number (defaults to 1)"`
	PageSize int    `json:"page_size,omitempty" jsonschema:"page size (defaults to 20)"`
	Type     string `json:"type,omitempty" jsonschema:"filter by data type: tickets, knowledge_articles, users, products, services, complete"`
	Status   string `json:"status,omitempty" jsonschema:"filter by status: pending, running, completed, failed"`
}

// ieJobListOutput is the MCP-specific output of importexport_job_list. It mirrors
// importexport.JobListResponse but carries the cycle-safe ieJob view.
type ieJobListOutput struct {
	Data       []ieJob `json:"data" jsonschema:"the page of import/export jobs"`
	Total      int64   `json:"total" jsonschema:"total number of matching jobs"`
	Page       int     `json:"page" jsonschema:"the 1-based page number returned"`
	PageSize   int     `json:"page_size" jsonschema:"the page size used"`
	TotalPages int     `json:"total_pages" jsonschema:"total number of pages available"`
}

// ieJobList lists import/export jobs with optional filters and pagination.
func ieJobList(_ context.Context, b Backend, in ieJobListInput) (ieJobListOutput, string, error) {
	filters := map[string]interface{}{}
	if in.Type != "" {
		filters["type"] = in.Type
	}
	if in.Status != "" {
		filters["status"] = in.Status
	}

	resp, err := b.ListJobs(in.Page, in.PageSize, filters)
	if err != nil {
		return ieJobListOutput{}, "", err
	}

	jobs := make([]ieJob, len(resp.Data))
	for i := range resp.Data {
		jobs[i] = ieJobFrom(&resp.Data[i])
	}
	out := ieJobListOutput{
		Data:       jobs,
		Total:      resp.Total,
		Page:       resp.Page,
		PageSize:   resp.PageSize,
		TotalPages: resp.TotalPages,
	}
	return out, fmt.Sprintf("listed %d of %d job(s) (page %d)", len(resp.Data), resp.Total, resp.Page), nil
}

// ----------------------------------------------------------------------------
// importexport_job_cancel
// ----------------------------------------------------------------------------

// ieJobCancelInput is the MCP input schema for importexport_job_cancel.
type ieJobCancelInput struct {
	ID uint `json:"id" jsonschema:"the numeric ID of the job to cancel"`
}

// ieJobCancelOutput is the structured output of importexport_job_cancel.
type ieJobCancelOutput struct {
	ID        uint `json:"id" jsonschema:"the numeric ID of the cancelled job"`
	Cancelled bool `json:"cancelled" jsonschema:"whether the job was cancelled"`
}

// ieJobCancel cancels a pending or running job on behalf of the acting user.
func ieJobCancel(ctx context.Context, b Backend, in ieJobCancelInput) (ieJobCancelOutput, string, error) {
	session, ok := SessionFromContext(ctx)
	if !ok || session == nil {
		return ieJobCancelOutput{}, "", ErrUnauthenticated
	}

	if err := b.CancelJob(in.ID, session.UserID); err != nil {
		return ieJobCancelOutput{}, "", err
	}
	return ieJobCancelOutput{ID: in.ID, Cancelled: true}, fmt.Sprintf("cancelled job #%d", in.ID), nil
}

// ----------------------------------------------------------------------------
// importexport_job_delete
// ----------------------------------------------------------------------------

// ieJobDeleteInput is the MCP input schema for importexport_job_delete.
type ieJobDeleteInput struct {
	ID uint `json:"id" jsonschema:"the numeric ID of the job to delete"`
}

// ieJobDeleteOutput is the structured output of importexport_job_delete.
type ieJobDeleteOutput struct {
	ID      uint `json:"id" jsonschema:"the numeric ID of the deleted job"`
	Deleted bool `json:"deleted" jsonschema:"whether the job was deleted"`
}

// ieJobDelete soft-deletes a job on behalf of the acting user.
func ieJobDelete(ctx context.Context, b Backend, in ieJobDeleteInput) (ieJobDeleteOutput, string, error) {
	session, ok := SessionFromContext(ctx)
	if !ok || session == nil {
		return ieJobDeleteOutput{}, "", ErrUnauthenticated
	}

	if err := b.DeleteJob(in.ID, session.UserID); err != nil {
		return ieJobDeleteOutput{}, "", err
	}
	return ieJobDeleteOutput{ID: in.ID, Deleted: true}, fmt.Sprintf("deleted job #%d", in.ID), nil
}

// ----------------------------------------------------------------------------
// importexport_job_stats
// ----------------------------------------------------------------------------

// ieJobStatsInput is the MCP input schema for importexport_job_stats. It takes
// no arguments.
type ieJobStatsInput struct{}

// ieJobStatsOutput is the structured output of importexport_job_stats.
// Statistics are returned as a free-form map mirroring the service layer's shape.
type ieJobStatsOutput struct {
	Stats map[string]interface{} `json:"stats" jsonschema:"aggregate import/export job statistics keyed by metric name"`
}

// ieJobStats returns aggregate import/export job statistics.
func ieJobStats(_ context.Context, b Backend, _ ieJobStatsInput) (ieJobStatsOutput, string, error) {
	stats, err := b.GetJobStats()
	if err != nil {
		return ieJobStatsOutput{}, "", err
	}
	return ieJobStatsOutput{Stats: stats}, "fetched import/export job statistics", nil
}
