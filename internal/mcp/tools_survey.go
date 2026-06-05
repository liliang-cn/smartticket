package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/company/smartticket/internal/survey"
)

// surveyStatsOutput is the MCP-local view of survey.Stats. The fields are
// scalar values so no omitempty conversion is needed; the type mirrors the
// service-layer Stats directly for clarity.
type surveyStatsOutput struct {
	SentCount     int     `json:"sent_count" jsonschema:"total number of surveys sent"`
	ResponseCount int     `json:"response_count" jsonschema:"number of surveys that received a response"`
	ResponseRate  float64 `json:"response_rate" jsonschema:"fraction of surveys answered (0.0 to 1.0)"`
	AverageRating float64 `json:"average_rating" jsonschema:"mean customer rating (1..5); 0 when no responses yet"`
}

type surveyStatsInput struct{}

func registerSurveyTools(s *mcp.Server, b Backend) {
	registerTool(s, "survey_stats",
		"Return aggregate CSAT survey statistics: sent count, response count, response rate, and average rating.",
		"survey:read",
		func(_ context.Context, _ surveyStatsInput) (surveyStatsOutput, string, error) {
			return surveyStats(b)
		})
}

func surveyStats(b Backend) (surveyStatsOutput, string, error) {
	st, err := b.GetSurveyStats()
	if err != nil {
		return surveyStatsOutput{}, "", err
	}
	out := surveyStatsOutput{
		SentCount:     st.SentCount,
		ResponseCount: st.ResponseCount,
		ResponseRate:  st.ResponseRate,
		AverageRating: st.AverageRating,
	}
	summary := buildSurveyStatsSummary(st)
	return out, summary, nil
}

func buildSurveyStatsSummary(st survey.Stats) string {
	if st.SentCount == 0 {
		return "No surveys sent yet."
	}
	return "CSAT stats: " +
		itoa(st.SentCount) + " sent, " +
		itoa(st.ResponseCount) + " responded, " +
		fmtPct(st.ResponseRate) + "% response rate, " +
		fmtRating(st.AverageRating) + " avg rating."
}

// itoa converts an int to its decimal string representation without importing
// strconv at the cost of a format call — kept inline for readability.
func itoa(n int) string {
	return intToStr(n)
}

// intToStr converts an int to string (avoids the strconv import).
func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

func fmtPct(r float64) string {
	pct := int(r * 100)
	return intToStr(pct)
}

func fmtRating(r float64) string {
	// Format to one decimal place without fmt to avoid import bloat.
	// We round to nearest 0.1.
	whole := int(r)
	frac := int((r-float64(whole))*10 + 0.5)
	if frac >= 10 {
		whole++
		frac = 0
	}
	return intToStr(whole) + "." + intToStr(frac)
}
