package mcp

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/company/smartticket/internal/errors"
	servicemgmt "github.com/company/smartticket/internal/service"
)

func TestSvcCreate(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("service:write"))

	in := svcCreateInput{
		ProductID:    7,
		Name:         "Email Support",
		Code:         "EMAIL-SUP",
		Type:         "support",
		Availability: "24x7",
	}
	expectedReq := &servicemgmt.CreateServiceRequest{
		ProductID:    7,
		Name:         "Email Support",
		Code:         "EMAIL-SUP",
		Type:         "support",
		Availability: "24x7",
	}
	resp := &servicemgmt.ServiceResponse{ID: 42, Name: "Email Support", Code: "EMAIL-SUP"}
	mb.On("CreateService", expectedReq).Return(resp, nil)

	out, summary, err := svcCreate(ctx, mb, in)
	require.NoError(t, err)
	assert.Equal(t, svcResponseFrom(resp), out)
	assert.Equal(t, "Created service #42 (Email Support).", summary)
	mb.AssertExpectations(t)
}

func TestSvcCreateError(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("service:write"))

	in := svcCreateInput{ProductID: 7, Name: "X", Code: "X", Type: "support"}
	wantErr := apperrors.NewConflictError("Service code already exists")
	mb.On("CreateService", &servicemgmt.CreateServiceRequest{
		ProductID: 7, Name: "X", Code: "X", Type: "support",
	}).Return(nil, wantErr)

	out, summary, err := svcCreate(ctx, mb, in)
	require.Error(t, err)
	assert.Equal(t, wantErr, err)
	assert.Zero(t, out)
	assert.Empty(t, summary)
	mb.AssertExpectations(t)
}

func TestSvcGet(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("service:read"))

	resp := &servicemgmt.ServiceResponse{ID: 9, Name: "DB Hosting"}
	mb.On("GetService", uint(9)).Return(resp, nil)

	out, summary, err := svcGet(ctx, mb, svcGetInput{ID: 9})
	require.NoError(t, err)
	assert.Equal(t, svcResponseFrom(resp), out)
	assert.Equal(t, "Service #9 (DB Hosting).", summary)
	mb.AssertExpectations(t)
}

func TestSvcGetNotFound(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("service:read"))

	wantErr := apperrors.NewNotFoundError("Service")
	mb.On("GetService", uint(404)).Return(nil, wantErr)

	out, summary, err := svcGet(ctx, mb, svcGetInput{ID: 404})
	require.Error(t, err)
	assert.Equal(t, wantErr, err)
	assert.Zero(t, out)
	assert.Empty(t, summary)
	mb.AssertExpectations(t)
}

func TestSvcList(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("service:read"))

	in := svcListInput{Page: 2, PageSize: 10, Search: "host", ProductID: 3, Type: "infrastructure", Status: "active", SortBy: "name", SortOrder: "asc"}
	expectedReq := &servicemgmt.ListServicesRequest{
		Page: 2, PageSize: 10, Search: "host", ProductID: 3, Type: "infrastructure", Status: "active", SortBy: "name", SortOrder: "asc",
	}
	services := []servicemgmt.ServiceResponse{
		{ID: 1, Name: "A"},
		{ID: 2, Name: "B"},
	}
	mb.On("ListServices", expectedReq).Return(services, 5, nil)

	out, summary, err := svcList(ctx, mb, in)
	require.NoError(t, err)
	assert.Equal(t, svcResponsesFrom(services), out.Services)
	assert.Equal(t, int64(5), out.Total)
	assert.Equal(t, "Listed 2 of 5 service(s).", summary)
	mb.AssertExpectations(t)
}

func TestSvcListError(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("service:read"))

	wantErr := errors.New("db down")
	mb.On("ListServices", &servicemgmt.ListServicesRequest{}).Return(nil, 0, wantErr)

	out, summary, err := svcList(ctx, mb, svcListInput{})
	require.Error(t, err)
	assert.Equal(t, wantErr, err)
	assert.Empty(t, out.Services)
	assert.Zero(t, out.Total)
	assert.Empty(t, summary)
	mb.AssertExpectations(t)
}

func TestSvcUpdate(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("service:write"))

	pid := uint(11)
	in := svcUpdateInput{ID: 5, ProductID: &pid, Name: "Renamed", Status: "maintenance"}
	expectedReq := &servicemgmt.UpdateServiceRequest{ProductID: &pid, Name: "Renamed", Status: "maintenance"}
	resp := &servicemgmt.ServiceResponse{ID: 5, Name: "Renamed"}
	mb.On("UpdateService", uint(5), expectedReq).Return(resp, nil)

	out, summary, err := svcUpdate(ctx, mb, in)
	require.NoError(t, err)
	assert.Equal(t, svcResponseFrom(resp), out)
	assert.Equal(t, "Updated service #5 (Renamed).", summary)
	mb.AssertExpectations(t)
}

func TestSvcUpdateError(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("service:write"))

	wantErr := apperrors.NewInvalidInputError("type", "Invalid service type")
	mb.On("UpdateService", uint(5), &servicemgmt.UpdateServiceRequest{Type: "bogus"}).Return(nil, wantErr)

	out, summary, err := svcUpdate(ctx, mb, svcUpdateInput{ID: 5, Type: "bogus"})
	require.Error(t, err)
	assert.Equal(t, wantErr, err)
	assert.Zero(t, out)
	assert.Empty(t, summary)
	mb.AssertExpectations(t)
}

func TestSvcDelete(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("service:write"))

	mb.On("DeleteService", uint(8)).Return(nil)

	out, summary, err := svcDelete(ctx, mb, svcDeleteInput{ID: 8})
	require.NoError(t, err)
	assert.Equal(t, uint(8), out.ID)
	assert.Equal(t, "Deleted service #8.", summary)
	mb.AssertExpectations(t)
}

func TestSvcDeleteError(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("service:write"))

	wantErr := apperrors.NewBusinessRuleError("Cannot delete service", "tickets attached")
	mb.On("DeleteService", uint(8)).Return(wantErr)

	out, summary, err := svcDelete(ctx, mb, svcDeleteInput{ID: 8})
	require.Error(t, err)
	assert.Equal(t, wantErr, err)
	assert.Zero(t, out.ID)
	assert.Empty(t, summary)
	mb.AssertExpectations(t)
}

func TestSvcActivate(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("service:write"))

	mb.On("ActivateService", uint(3)).Return(nil)

	out, summary, err := svcActivate(ctx, mb, svcActivateInput{ID: 3})
	require.NoError(t, err)
	assert.Equal(t, uint(3), out.ID)
	assert.Equal(t, "Activated service #3.", summary)
	mb.AssertExpectations(t)
}

func TestSvcActivateError(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("service:write"))

	wantErr := apperrors.NewNotFoundError("Service")
	mb.On("ActivateService", uint(3)).Return(wantErr)

	out, summary, err := svcActivate(ctx, mb, svcActivateInput{ID: 3})
	require.Error(t, err)
	assert.Equal(t, wantErr, err)
	assert.Zero(t, out.ID)
	assert.Empty(t, summary)
	mb.AssertExpectations(t)
}

func TestSvcDeactivate(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("service:write"))

	mb.On("DeactivateService", uint(4)).Return(nil)

	out, summary, err := svcDeactivate(ctx, mb, svcDeactivateInput{ID: 4})
	require.NoError(t, err)
	assert.Equal(t, uint(4), out.ID)
	assert.Equal(t, "Deactivated service #4.", summary)
	mb.AssertExpectations(t)
}

func TestSvcDeactivateError(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("service:write"))

	wantErr := apperrors.NewNotFoundError("Service")
	mb.On("DeactivateService", uint(4)).Return(wantErr)

	out, summary, err := svcDeactivate(ctx, mb, svcDeactivateInput{ID: 4})
	require.Error(t, err)
	assert.Equal(t, wantErr, err)
	assert.Zero(t, out.ID)
	assert.Empty(t, summary)
	mb.AssertExpectations(t)
}
