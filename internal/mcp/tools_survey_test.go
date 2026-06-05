package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/company/smartticket/internal/survey"
)

func TestSurveyStatsTool(t *testing.T) {
	b := new(MockBackend)

	b.On("GetSurveyStats").Return(survey.Stats{
		SentCount:     50,
		ResponseCount: 30,
		ResponseRate:  0.6,
		AverageRating: 4.2,
	}, nil)

	out, summary, err := surveyStats(b)
	require.NoError(t, err)
	assert.Equal(t, 50, out.SentCount)
	assert.Equal(t, 30, out.ResponseCount)
	assert.InDelta(t, 0.6, out.ResponseRate, 0.001)
	assert.InDelta(t, 4.2, out.AverageRating, 0.001)
	assert.Contains(t, summary, "50")
	assert.Contains(t, summary, "30")
	b.AssertExpectations(t)
}

func TestSurveyStatsEmpty(t *testing.T) {
	b := new(MockBackend)

	b.On("GetSurveyStats").Return(survey.Stats{}, nil)

	out, summary, err := surveyStats(b)
	require.NoError(t, err)
	assert.Equal(t, 0, out.SentCount)
	assert.Equal(t, "No surveys sent yet.", summary)
	b.AssertExpectations(t)
}

func TestSurveyPermissionDenied(t *testing.T) {
	// survey:read is required; a session without it must be denied.
	ctx := ctxWithSession(newTestSession()) // no survey:read
	err := RequirePermission(ctx, "survey:read")
	var permErr *PermissionError
	require.ErrorAs(t, err, &permErr)
	assert.Equal(t, "survey:read", permErr.Code)
}
