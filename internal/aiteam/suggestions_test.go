package aiteam

import (
	"fmt"
	"testing"

	sqlite "github.com/company/smartticket/internal/database/moderncsqlite"
	"github.com/company/smartticket/internal/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newSuggDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.AISuggestion{}))
	return db
}

func TestUpsertReplacesPerTicketAgent(t *testing.T) {
	db := newSuggDB(t)
	st := NewSuggestionStore(db)

	s1, err := st.Upsert(42, "Triage", "done", 0.8, `{"priority":"high"}`)
	require.NoError(t, err)
	require.Equal(t, "done", s1.Status)

	// Same (ticket, agent) updates the SAME row, not a new one.
	s2, err := st.Upsert(42, "Triage", "done", 0.6, `{"priority":"medium"}`)
	require.NoError(t, err)
	require.Equal(t, s1.ID, s2.ID)

	list, err := st.List(42)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.InDelta(t, 0.6, list[0].Confidence, 0.001)
}

func TestAdoptAndDismiss(t *testing.T) {
	db := newSuggDB(t)
	st := NewSuggestionStore(db)
	s, _ := st.Upsert(1, "Reviewer", "done", 0.9, "{}")

	require.NoError(t, st.Adopt(s.ID, 7))
	got, _ := st.Get(s.ID)
	require.Equal(t, "adopted", got.Status)
	require.NotNil(t, got.AdoptedBy)
	require.Equal(t, uint(7), *got.AdoptedBy)
	require.NotNil(t, got.ResolvedAt)

	s2, _ := st.Upsert(1, "Drafter", "done", 0.5, "{}")
	require.NoError(t, st.Dismiss(s2.ID))
	got2, _ := st.Get(s2.ID)
	require.Equal(t, "dismissed", got2.Status)
}
