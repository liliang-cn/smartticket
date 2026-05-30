package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	apperrors "github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/sla"
)

// --- template tools ---

func TestSLACreateTemplate(t *testing.T) {
	b := &MockBackend{}
	ctx := ctxWithSession(newTestSession("sla:write"))

	in := slaCreateTemplateInput{Name: "Gold", Description: "premium", IsActive: true}
	b.On("CreateSLATemplate", mock.MatchedBy(func(r *sla.CreateSLATemplateRequest) bool {
		return r.Name == "Gold" && r.Description == "premium" && r.IsActive
	})).Return(&sla.SLATemplateResponse{ID: 7, Name: "Gold"}, nil)

	out, summary, err := slaCreateTemplate(ctx, b, in)
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, uint(7), out.ID)
	assert.Contains(t, summary, "#7")
	b.AssertExpectations(t)
}

func TestSLAGetTemplate(t *testing.T) {
	b := &MockBackend{}
	ctx := ctxWithSession(newTestSession("sla:read"))

	b.On("GetSLATemplate", uint(3)).Return(&sla.SLATemplateResponse{ID: 3, Name: "Silver"}, nil)

	out, summary, err := slaGetTemplate(ctx, b, slaGetTemplateInput{TemplateID: 3})
	require.NoError(t, err)
	assert.Equal(t, "Silver", out.Name)
	assert.Contains(t, summary, "Silver")
	b.AssertExpectations(t)
}

func TestSLAGetTemplate_NotFound(t *testing.T) {
	b := &MockBackend{}
	ctx := ctxWithSession(newTestSession("sla:read"))

	b.On("GetSLATemplate", uint(99)).Return(nil, apperrors.NewNotFoundError("SLA template"))

	_, _, err := slaGetTemplate(ctx, b, slaGetTemplateInput{TemplateID: 99})
	require.Error(t, err)
	b.AssertExpectations(t)
}

func TestSLAListTemplates(t *testing.T) {
	b := &MockBackend{}
	ctx := ctxWithSession(newTestSession("sla:read"))

	active := true
	in := slaListTemplatesInput{Page: 2, PageSize: 10, IsActive: &active}
	b.On("ListSLATemplates", mock.MatchedBy(func(r *sla.ListSLATemplatesRequest) bool {
		return r.Page == 2 && r.PageSize == 10 && r.IsActive != nil && *r.IsActive
	})).Return([]sla.SLATemplateResponse{{ID: 1}, {ID: 2}}, 5, nil)

	out, summary, err := slaListTemplates(ctx, b, in)
	require.NoError(t, err)
	assert.Len(t, out.Templates, 2)
	assert.Equal(t, int64(5), out.Total)
	assert.Equal(t, 2, out.Page)
	assert.Contains(t, summary, "2 of 5")
	b.AssertExpectations(t)
}

func TestSLAUpdateTemplate(t *testing.T) {
	b := &MockBackend{}
	ctx := ctxWithSession(newTestSession("sla:write"))

	name := "Platinum"
	in := slaUpdateTemplateInput{TemplateID: 4, Name: &name}
	b.On("UpdateSLATemplate", uint(4), mock.MatchedBy(func(r *sla.UpdateSLATemplateRequest) bool {
		return r.Name != nil && *r.Name == "Platinum"
	})).Return(&sla.SLATemplateResponse{ID: 4, Name: "Platinum"}, nil)

	out, summary, err := slaUpdateTemplate(ctx, b, in)
	require.NoError(t, err)
	assert.Equal(t, "Platinum", out.Name)
	assert.Contains(t, summary, "#4")
	b.AssertExpectations(t)
}

func TestSLADeleteTemplate(t *testing.T) {
	b := &MockBackend{}
	ctx := ctxWithSession(newTestSession("sla:write"))

	b.On("DeleteSLATemplate", uint(8)).Return(nil)

	out, summary, err := slaDeleteTemplate(ctx, b, slaTemplateIDInput{TemplateID: 8})
	require.NoError(t, err)
	assert.Equal(t, uint(8), out.ID)
	assert.Equal(t, "deleted", out.Status)
	assert.Contains(t, summary, "#8")
	b.AssertExpectations(t)
}

func TestSLASetDefaultTemplate(t *testing.T) {
	b := &MockBackend{}
	ctx := ctxWithSession(newTestSession("sla:write"))

	b.On("SetDefaultSLATemplate", uint(2)).Return(nil)

	out, summary, err := slaSetDefaultTemplate(ctx, b, slaTemplateIDInput{TemplateID: 2})
	require.NoError(t, err)
	assert.Equal(t, "default", out.Status)
	assert.Contains(t, summary, "#2")
	b.AssertExpectations(t)
}

func TestSLAActivateTemplate(t *testing.T) {
	b := &MockBackend{}
	ctx := ctxWithSession(newTestSession("sla:write"))

	b.On("ActivateSLATemplate", uint(5)).Return(nil)

	out, _, err := slaActivateTemplate(ctx, b, slaTemplateIDInput{TemplateID: 5})
	require.NoError(t, err)
	assert.Equal(t, "activated", out.Status)
	b.AssertExpectations(t)
}

func TestSLADeactivateTemplate(t *testing.T) {
	b := &MockBackend{}
	ctx := ctxWithSession(newTestSession("sla:write"))

	b.On("DeactivateSLATemplate", uint(6)).Return(nil)

	out, _, err := slaDeactivateTemplate(ctx, b, slaTemplateIDInput{TemplateID: 6})
	require.NoError(t, err)
	assert.Equal(t, "deactivated", out.Status)
	b.AssertExpectations(t)
}

// --- rule tools ---

func TestSLACreateRule(t *testing.T) {
	b := &MockBackend{}
	ctx := ctxWithSession(newTestSession("sla:write"))

	in := slaCreateRuleInput{
		TemplateID:     1,
		Priority:       "high",
		Severity:       "major",
		ResponseTime:   30,
		ResolutionTime: 240,
	}
	b.On("CreateSLARule", mock.MatchedBy(func(r *sla.CreateSLARuleRequest) bool {
		return r.TemplateID == 1 && r.Priority == "high" && r.Severity == "major" &&
			r.ResponseTime == 30 && r.ResolutionTime == 240
	})).Return(&sla.SLARuleResponse{ID: 11, Priority: "high", Severity: "major"}, nil)

	out, summary, err := slaCreateRule(ctx, b, in)
	require.NoError(t, err)
	assert.Equal(t, uint(11), out.ID)
	assert.Contains(t, summary, "high/major")
	b.AssertExpectations(t)
}

func TestSLAGetRule(t *testing.T) {
	b := &MockBackend{}
	ctx := ctxWithSession(newTestSession("sla:read"))

	b.On("GetSLARule", uint(12)).Return(&sla.SLARuleResponse{ID: 12, Priority: "low", Severity: "minor"}, nil)

	out, summary, err := slaGetRule(ctx, b, slaGetRuleInput{RuleID: 12})
	require.NoError(t, err)
	assert.Equal(t, uint(12), out.ID)
	assert.Contains(t, summary, "low/minor")
	b.AssertExpectations(t)
}

func TestSLAListRules(t *testing.T) {
	b := &MockBackend{}
	ctx := ctxWithSession(newTestSession("sla:read"))

	in := slaListRulesInput{Page: 1, PageSize: 20, Priority: "critical"}
	b.On("ListSLARules", mock.MatchedBy(func(r *sla.ListSLARulesRequest) bool {
		return r.Page == 1 && r.PageSize == 20 && r.Priority == "critical"
	})).Return([]sla.SLARuleResponse{{ID: 1}}, 1, nil)

	out, summary, err := slaListRules(ctx, b, in)
	require.NoError(t, err)
	assert.Len(t, out.Rules, 1)
	assert.Equal(t, int64(1), out.Total)
	assert.Contains(t, summary, "1 of 1")
	b.AssertExpectations(t)
}

func TestSLAUpdateRule(t *testing.T) {
	b := &MockBackend{}
	ctx := ctxWithSession(newTestSession("sla:write"))

	rt := 15
	in := slaUpdateRuleInput{RuleID: 13, ResponseTime: &rt}
	b.On("UpdateSLARule", uint(13), mock.MatchedBy(func(r *sla.UpdateSLARuleRequest) bool {
		return r.ResponseTime != nil && *r.ResponseTime == 15
	})).Return(&sla.SLARuleResponse{ID: 13, Priority: "high", Severity: "major"}, nil)

	out, summary, err := slaUpdateRule(ctx, b, in)
	require.NoError(t, err)
	assert.Equal(t, uint(13), out.ID)
	assert.Contains(t, summary, "#13")
	b.AssertExpectations(t)
}

func TestSLADeleteRule(t *testing.T) {
	b := &MockBackend{}
	ctx := ctxWithSession(newTestSession("sla:write"))

	b.On("DeleteSLARule", uint(14)).Return(nil)

	out, summary, err := slaDeleteRule(ctx, b, slaRuleIDInput{RuleID: 14})
	require.NoError(t, err)
	assert.Equal(t, "deleted", out.Status)
	assert.Contains(t, summary, "#14")
	b.AssertExpectations(t)
}

func TestSLAActivateRule(t *testing.T) {
	b := &MockBackend{}
	ctx := ctxWithSession(newTestSession("sla:write"))

	b.On("ActivateSLARule", uint(15)).Return(nil)

	out, _, err := slaActivateRule(ctx, b, slaRuleIDInput{RuleID: 15})
	require.NoError(t, err)
	assert.Equal(t, "activated", out.Status)
	b.AssertExpectations(t)
}

func TestSLADeactivateRule(t *testing.T) {
	b := &MockBackend{}
	ctx := ctxWithSession(newTestSession("sla:write"))

	b.On("DeactivateSLARule", uint(16)).Return(nil)

	out, _, err := slaDeactivateRule(ctx, b, slaRuleIDInput{RuleID: 16})
	require.NoError(t, err)
	assert.Equal(t, "deactivated", out.Status)
	b.AssertExpectations(t)
}

func TestSLADeactivateRule_Error(t *testing.T) {
	b := &MockBackend{}
	ctx := ctxWithSession(newTestSession("sla:write"))

	b.On("DeactivateSLARule", uint(17)).Return(apperrors.NewNotFoundError("SLA rule"))

	_, _, err := slaDeactivateRule(ctx, b, slaRuleIDInput{RuleID: 17})
	require.Error(t, err)
	b.AssertExpectations(t)
}
