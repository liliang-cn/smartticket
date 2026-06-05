package survey

import (
	"fmt"
	"testing"

	sqlite "github.com/company/smartticket/internal/database/moderncsqlite"
	"github.com/company/smartticket/internal/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.SatisfactionSurvey{}))
	return NewService(db)
}

func TestCreateForTicket_CreatesNewSurvey(t *testing.T) {
	svc := newTestService(t)

	survey, err := svc.CreateForTicket(1)
	require.NoError(t, err)
	require.Equal(t, uint(1), survey.TicketID)
	require.NotEmpty(t, survey.Token)
	require.NotNil(t, survey.SentAt)
	require.Equal(t, 0, survey.Rating) // not yet answered
}

func TestCreateForTicket_Idempotent(t *testing.T) {
	svc := newTestService(t)

	s1, err := svc.CreateForTicket(42)
	require.NoError(t, err)

	s2, err := svc.CreateForTicket(42)
	require.NoError(t, err)

	// Must return the same survey (same ID, same token).
	require.Equal(t, s1.ID, s2.ID)
	require.Equal(t, s1.Token, s2.Token)
}

func TestSubmit_RejectsRatingZero(t *testing.T) {
	svc := newTestService(t)
	s, err := svc.CreateForTicket(10)
	require.NoError(t, err)

	err = svc.Submit(s.Token, 0, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "between 1 and 5")
}

func TestSubmit_RejectsRatingSix(t *testing.T) {
	svc := newTestService(t)
	s, err := svc.CreateForTicket(10)
	require.NoError(t, err)

	err = svc.Submit(s.Token, 6, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "between 1 and 5")
}

func TestSubmit_AcceptsValidRating(t *testing.T) {
	svc := newTestService(t)
	s, err := svc.CreateForTicket(20)
	require.NoError(t, err)

	err = svc.Submit(s.Token, 5, "great service")
	require.NoError(t, err)

	updated, err := svc.GetByToken(s.Token)
	require.NoError(t, err)
	require.Equal(t, 5, updated.Rating)
	require.Equal(t, "great service", updated.Comment)
	require.NotNil(t, updated.RespondedAt)
}

func TestSubmit_RejectsDoubleSubmit(t *testing.T) {
	svc := newTestService(t)
	s, err := svc.CreateForTicket(30)
	require.NoError(t, err)

	require.NoError(t, svc.Submit(s.Token, 4, "good"))

	err = svc.Submit(s.Token, 3, "changed mind")
	require.Error(t, err)
	require.Contains(t, err.Error(), "already responded")
}

func TestGetStats_ComputesAverageAndRate(t *testing.T) {
	svc := newTestService(t)

	// 3 surveys; 2 respond
	for i := uint(100); i < 103; i++ {
		s, err := svc.CreateForTicket(i)
		require.NoError(t, err)
		if i < 102 {
			require.NoError(t, svc.Submit(s.Token, int(i-97), "")) // ratings: 3, 4
		}
	}

	stats, err := svc.GetStats()
	require.NoError(t, err)
	require.Equal(t, 3, stats.SentCount)
	require.Equal(t, 2, stats.ResponseCount)
	require.InDelta(t, 2.0/3.0, stats.ResponseRate, 0.001)
	require.InDelta(t, 3.5, stats.AverageRating, 0.001) // (3+4)/2
}

func TestGetStats_NoSurveys(t *testing.T) {
	svc := newTestService(t)
	stats, err := svc.GetStats()
	require.NoError(t, err)
	require.Equal(t, 0, stats.SentCount)
	require.Equal(t, 0, stats.ResponseCount)
	require.Equal(t, 0.0, stats.AverageRating)
	require.Equal(t, 0.0, stats.ResponseRate)
}

func TestGetByToken_UnknownTokenReturnsError(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.GetByToken("nonexistent-token")
	require.Error(t, err)
}

func TestCreateForTicket_NewSurveyAfterResponded(t *testing.T) {
	// After a survey has been responded to, CreateForTicket should create
	// a NEW survey (the old one is answered; there's no unanswered one).
	svc := newTestService(t)

	s1, err := svc.CreateForTicket(50)
	require.NoError(t, err)
	require.NoError(t, svc.Submit(s1.Token, 4, "ok"))

	s2, err := svc.CreateForTicket(50)
	require.NoError(t, err)
	// New survey — different ID and token.
	require.NotEqual(t, s1.ID, s2.ID)
	require.NotEqual(t, s1.Token, s2.Token)
}
