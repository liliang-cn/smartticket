package mcp

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	apperrors "github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/importexport"
	"github.com/company/smartticket/internal/models"
)

// --- import_create ---

func TestIEImportCreateNotSupported(t *testing.T) {
	b := &MockBackend{}
	ctx := ctxWithSession(newTestSession("importexport:write"))

	_, _, err := ieImportCreate(ctx, b, ieImportCreateInput{
		Type:         "tickets",
		SourceType:   "zendesk",
		SourceFormat: "csv",
	})

	require.Error(t, err)
	appErr, ok := apperrors.IsAppError(err)
	require.True(t, ok)
	assert.Equal(t, apperrors.ErrCodeForbidden, appErr.Code)
	// CreateImportJob must never be called over MCP.
	b.AssertNotCalled(t, "CreateImportJob", mock.Anything, mock.Anything, mock.Anything)
	b.AssertExpectations(t)
}

func TestIEImportCreateUnauthenticated(t *testing.T) {
	b := &MockBackend{}
	ctx := ctxWithSession(nil)

	_, _, err := ieImportCreate(ctx, b, ieImportCreateInput{})

	assert.ErrorIs(t, err, ErrUnauthenticated)
	b.AssertNotCalled(t, "CreateImportJob", mock.Anything, mock.Anything, mock.Anything)
	b.AssertExpectations(t)
}

// --- export_create ---

func TestIEExportCreateSuccess(t *testing.T) {
	b := &MockBackend{}
	session := newTestSession("importexport:write")
	ctx := ctxWithSession(session)

	in := ieExportCreateInput{
		Type:         "tickets",
		TargetFormat: "csv",
		Filters:      `{"status":"open"}`,
		Options:      `{"include_closed":false}`,
	}

	expectedReq := &importexport.ExportRequest{
		Type:         importexport.ExportTypeTickets,
		TargetFormat: importexport.FileTypeCSV,
		Filters:      in.Filters,
		Options:      in.Options,
	}

	resp := &importexport.JobResponse{ID: 42, Type: "tickets", TargetFormat: "csv", Status: "pending"}
	b.On("CreateExportJob", session.UserID, expectedReq).Return(resp, nil)

	out, summary, err := ieExportCreate(ctx, b, in)

	require.NoError(t, err)
	assert.Equal(t, ieJobFrom(resp), out)
	assert.Equal(t, uint(42), out.ID)
	assert.Contains(t, summary, "#42")
	b.AssertExpectations(t)
}

func TestIEExportCreateUnauthenticated(t *testing.T) {
	b := &MockBackend{}
	ctx := ctxWithSession(nil)

	_, _, err := ieExportCreate(ctx, b, ieExportCreateInput{Type: "tickets", TargetFormat: "csv"})

	assert.ErrorIs(t, err, ErrUnauthenticated)
	b.AssertNotCalled(t, "CreateExportJob", mock.Anything, mock.Anything)
	b.AssertExpectations(t)
}

func TestIEExportCreateBackendError(t *testing.T) {
	b := &MockBackend{}
	session := newTestSession("importexport:write")
	ctx := ctxWithSession(session)

	wantErr := errors.New("boom")
	b.On("CreateExportJob", session.UserID, mock.AnythingOfType("*importexport.ExportRequest")).
		Return(nil, wantErr)

	_, _, err := ieExportCreate(ctx, b, ieExportCreateInput{Type: "users", TargetFormat: "json"})

	assert.ErrorIs(t, err, wantErr)
	b.AssertExpectations(t)
}

// --- importexport_job_get ---

func TestIEJobGetSuccess(t *testing.T) {
	b := &MockBackend{}
	resp := &importexport.JobResponse{ID: 7, Type: "tickets", Status: "completed"}
	b.On("GetJob", uint(7)).Return(resp, nil)

	out, summary, err := ieJobGet(ctxWithSession(newTestSession("importexport:read")), b, ieJobGetInput{ID: 7})

	require.NoError(t, err)
	assert.Equal(t, ieJobFrom(resp), out)
	assert.Equal(t, uint(7), out.ID)
	assert.Contains(t, summary, "#7")
	b.AssertExpectations(t)
}

func TestIEJobGetError(t *testing.T) {
	b := &MockBackend{}
	wantErr := apperrors.NewNotFoundError("import/export job not found")
	b.On("GetJob", uint(99)).Return(nil, wantErr)

	_, _, err := ieJobGet(ctxWithSession(newTestSession("importexport:read")), b, ieJobGetInput{ID: 99})

	assert.ErrorIs(t, err, wantErr)
	b.AssertExpectations(t)
}

// --- importexport_job_list ---

func TestIEJobListSuccess(t *testing.T) {
	b := &MockBackend{}
	expectedFilters := map[string]interface{}{"type": "tickets", "status": "running"}
	resp := &importexport.JobListResponse{
		Data: []importexport.JobResponse{
			{ID: 1, StartedByUser: &models.User{BaseModel: models.BaseModel{ID: 9}}},
			{ID: 2},
		},
		Total:      2,
		Page:       1,
		PageSize:   20,
		TotalPages: 1,
	}
	b.On("ListJobs", 1, 20, expectedFilters).Return(resp, nil)

	out, summary, err := ieJobList(ctxWithSession(newTestSession("importexport:read")), b, ieJobListInput{
		Page:     1,
		PageSize: 20,
		Type:     "tickets",
		Status:   "running",
	})

	require.NoError(t, err)
	require.Len(t, out.Data, 2)
	assert.Equal(t, uint(1), out.Data[0].ID)
	assert.Equal(t, uint(9), out.Data[0].StartedByUserID) // embedded user flattened to ID
	assert.Equal(t, uint(2), out.Data[1].ID)
	assert.Zero(t, out.Data[1].StartedByUserID)
	assert.Equal(t, int64(2), out.Total)
	assert.Equal(t, 1, out.Page)
	assert.Equal(t, 20, out.PageSize)
	assert.Contains(t, summary, "2")
	b.AssertExpectations(t)
}

func TestIEJobListNoFilters(t *testing.T) {
	b := &MockBackend{}
	resp := &importexport.JobListResponse{Data: nil, Total: 0, Page: 1}
	b.On("ListJobs", 0, 0, map[string]interface{}{}).Return(resp, nil)

	_, _, err := ieJobList(ctxWithSession(newTestSession("importexport:read")), b, ieJobListInput{})

	require.NoError(t, err)
	b.AssertExpectations(t)
}

// --- importexport_job_cancel ---

func TestIEJobCancelSuccess(t *testing.T) {
	b := &MockBackend{}
	session := newTestSession("importexport:write")
	b.On("CancelJob", uint(5), session.UserID).Return(nil)

	out, summary, err := ieJobCancel(ctxWithSession(session), b, ieJobCancelInput{ID: 5})

	require.NoError(t, err)
	assert.Equal(t, ieJobCancelOutput{ID: 5, Cancelled: true}, out)
	assert.Contains(t, summary, "#5")
	b.AssertExpectations(t)
}

func TestIEJobCancelUnauthenticated(t *testing.T) {
	b := &MockBackend{}

	_, _, err := ieJobCancel(ctxWithSession(nil), b, ieJobCancelInput{ID: 5})

	assert.ErrorIs(t, err, ErrUnauthenticated)
	b.AssertNotCalled(t, "CancelJob", mock.Anything, mock.Anything)
	b.AssertExpectations(t)
}

func TestIEJobCancelError(t *testing.T) {
	b := &MockBackend{}
	session := newTestSession("importexport:write")
	wantErr := apperrors.NewValidationError("job cannot be cancelled in current status")
	b.On("CancelJob", uint(5), session.UserID).Return(wantErr)

	_, _, err := ieJobCancel(ctxWithSession(session), b, ieJobCancelInput{ID: 5})

	assert.ErrorIs(t, err, wantErr)
	b.AssertExpectations(t)
}

// --- importexport_job_delete ---

func TestIEJobDeleteSuccess(t *testing.T) {
	b := &MockBackend{}
	session := newTestSession("importexport:write")
	b.On("DeleteJob", uint(8), session.UserID).Return(nil)

	out, summary, err := ieJobDelete(ctxWithSession(session), b, ieJobDeleteInput{ID: 8})

	require.NoError(t, err)
	assert.Equal(t, ieJobDeleteOutput{ID: 8, Deleted: true}, out)
	assert.Contains(t, summary, "#8")
	b.AssertExpectations(t)
}

func TestIEJobDeleteUnauthenticated(t *testing.T) {
	b := &MockBackend{}

	_, _, err := ieJobDelete(ctxWithSession(nil), b, ieJobDeleteInput{ID: 8})

	assert.ErrorIs(t, err, ErrUnauthenticated)
	b.AssertNotCalled(t, "DeleteJob", mock.Anything, mock.Anything)
	b.AssertExpectations(t)
}

func TestIEJobDeleteError(t *testing.T) {
	b := &MockBackend{}
	session := newTestSession("importexport:write")
	wantErr := apperrors.NewNotFoundError("import/export job not found")
	b.On("DeleteJob", uint(8), session.UserID).Return(wantErr)

	_, _, err := ieJobDelete(ctxWithSession(session), b, ieJobDeleteInput{ID: 8})

	assert.ErrorIs(t, err, wantErr)
	b.AssertExpectations(t)
}

// --- importexport_job_stats ---

func TestIEJobStatsSuccess(t *testing.T) {
	b := &MockBackend{}
	stats := map[string]interface{}{
		"total_jobs":       int64(3),
		"status_breakdown": map[string]int64{"completed": 2, "failed": 1},
	}
	b.On("GetJobStats").Return(stats, nil)

	out, summary, err := ieJobStats(ctxWithSession(newTestSession("importexport:read")), b, ieJobStatsInput{})

	require.NoError(t, err)
	assert.Equal(t, stats, out.Stats)
	assert.NotEmpty(t, summary)
	b.AssertExpectations(t)
}

func TestIEJobStatsError(t *testing.T) {
	b := &MockBackend{}
	wantErr := errors.New("db down")
	b.On("GetJobStats").Return(nil, wantErr)

	_, _, err := ieJobStats(ctxWithSession(newTestSession("importexport:read")), b, ieJobStatsInput{})

	assert.ErrorIs(t, err, wantErr)
	b.AssertExpectations(t)
}
