package aiteam

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sqlite "github.com/company/smartticket/internal/database/moderncsqlite"
	"github.com/company/smartticket/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func init() { gin.SetMode(gin.TestMode) }

func newHandlerDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.AISuggestion{}))
	return db
}

func setupHandlers(db *gorm.DB) (*Handlers, *gin.Engine) {
	store := NewSuggestionStore(db)
	h := NewHandlers(nil, store, func(id uint) TicketContext { return TicketContext{TicketID: id} })
	r := gin.New()
	r.GET("/tickets/:id/ai/suggestions", h.List)
	r.POST("/tickets/:id/ai/suggestions/:sid/adopt", func(c *gin.Context) {
		c.Set("user_id", uint(5)) // inject user_id as if authMiddleware ran
		h.Adopt(c)
	})
	r.POST("/tickets/:id/ai/suggestions/:sid/dismiss", h.Dismiss)
	r.POST("/tickets/:id/ai/research", h.Research)
	r.POST("/tickets/:id/ai/review", h.Review)
	r.POST("/tickets/:id/ai/draft", h.Draft)
	return h, r
}

// Adopting a suggestion that belongs to a DIFFERENT ticket than the path must
// be rejected (IDOR guard), even though the caller is authorized for the path
// ticket by the route middleware.
func TestAdoptRejectsCrossTicketSuggestion(t *testing.T) {
	db := newHandlerDB(t)
	_, r := setupHandlers(db)
	store := NewSuggestionStore(db)
	// Suggestion belongs to ticket 7.
	sug, err := store.Upsert(7, "Triage", "done", 0.8, "{}")
	require.NoError(t, err)

	// Attempt to adopt it under ticket 99 (a ticket the caller can access).
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/tickets/99/ai/suggestions/%d/adopt", sug.ID), nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)

	// The suggestion must remain un-adopted.
	got, _ := store.Get(sug.ID)
	require.Equal(t, "done", got.Status)

	// Adopting under the CORRECT ticket succeeds.
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/tickets/7/ai/suggestions/%d/adopt", sug.ID), nil)
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)
}

// TestListEmpty ensures List returns {"suggestions":[]} when no rows exist.
func TestHandlerListEmpty(t *testing.T) {
	db := newHandlerDB(t)
	_, r := setupHandlers(db)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tickets/99/ai/suggestions", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	sugs := resp["suggestions"].([]interface{})
	require.Empty(t, sugs)
}

// TestListWithRows seeds a suggestion and verifies List returns it.
func TestHandlerListWithRows(t *testing.T) {
	db := newHandlerDB(t)
	_, r := setupHandlers(db)

	store := NewSuggestionStore(db)
	_, err := store.Upsert(10, "Triage", "done", 0.9, `{"priority":"high"}`)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tickets/10/ai/suggestions", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	sugs := resp["suggestions"].([]interface{})
	require.Len(t, sugs, 1)
}

// TestAdoptSetsStatus verifies Adopt transitions status to "adopted".
func TestHandlerAdopt(t *testing.T) {
	db := newHandlerDB(t)
	_, r := setupHandlers(db)

	store := NewSuggestionStore(db)
	sug, err := store.Upsert(20, "Drafter", "done", 0.8, "{}")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/tickets/20/ai/suggestions/%d/adopt", sug.ID), nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, true, resp["adopted"])

	got, err := store.Get(sug.ID)
	require.NoError(t, err)
	require.Equal(t, "adopted", got.Status)
	require.NotNil(t, got.AdoptedBy)
	require.Equal(t, uint(5), *got.AdoptedBy)
}

// TestDismissSetsStatus verifies Dismiss transitions status to "dismissed".
func TestHandlerDismiss(t *testing.T) {
	db := newHandlerDB(t)
	_, r := setupHandlers(db)

	store := NewSuggestionStore(db)
	sug, err := store.Upsert(30, "Sentinel", "done", 0.7, "{}")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/tickets/30/ai/suggestions/%d/dismiss", sug.ID), nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, true, resp["dismissed"])

	got, err := store.Get(sug.ID)
	require.NoError(t, err)
	require.Equal(t, "dismissed", got.Status)
}

// TestRunEndpoints503WhenOrchNil verifies Research/Review/Draft return 503
// when no orchestrator is configured (orch = nil).
func TestHandlerRunEndpoints503WhenOrchNil(t *testing.T) {
	db := newHandlerDB(t)
	_, r := setupHandlers(db)

	for _, path := range []string{
		"/tickets/1/ai/research",
		"/tickets/1/ai/review",
		"/tickets/1/ai/draft",
	} {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)
			require.Equal(t, http.StatusServiceUnavailable, w.Code)
			var resp map[string]string
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			require.Contains(t, resp["error"], "AI not configured")
		})
	}
}

// TestBadTicketID verifies a non-numeric ticket id returns 400.
func TestHandlerBadTicketID(t *testing.T) {
	db := newHandlerDB(t)
	_, r := setupHandlers(db)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tickets/abc/ai/suggestions", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}
