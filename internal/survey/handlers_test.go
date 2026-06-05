package survey

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	sqlite "github.com/company/smartticket/internal/database/moderncsqlite"
	"github.com/company/smartticket/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTestHandlers(t *testing.T) (*Handlers, *Service) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	dsn := fmt.Sprintf("file:%s_h?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.SatisfactionSurvey{}))
	svc := NewService(db)
	return NewHandlers(svc), svc
}

func setupRouter(h *Handlers) *gin.Engine {
	r := gin.New()
	r.GET("/api/v1/survey/:token", h.GetSurvey)
	r.POST("/api/v1/survey/:token", h.SubmitSurvey)
	r.GET("/api/v1/survey/stats", h.GetStats)
	return r
}

func TestGetSurvey_ValidToken(t *testing.T) {
	h, svc := newTestHandlers(t)
	s, err := svc.CreateForTicket(1)
	require.NoError(t, err)

	r := setupRouter(h)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/survey/"+s.Token, nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.True(t, body["success"].(bool))
	data := body["data"].(map[string]interface{})
	require.Equal(t, float64(1), data["ticket_id"])
	require.False(t, data["responded"].(bool))
}

func TestGetSurvey_BadToken(t *testing.T) {
	h, _ := newTestHandlers(t)
	r := setupRouter(h)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/survey/doesnotexist", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestSubmitSurvey_ValidSubmit(t *testing.T) {
	h, svc := newTestHandlers(t)
	s, err := svc.CreateForTicket(2)
	require.NoError(t, err)

	body, _ := json.Marshal(map[string]interface{}{"rating": 4, "comment": "helpful"})
	r := setupRouter(h)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/survey/"+s.Token, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	// Subsequent GET should show responded=true
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/api/v1/survey/"+s.Token, nil)
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp))
	data := resp["data"].(map[string]interface{})
	require.True(t, data["responded"].(bool))
}

func TestSubmitSurvey_RatingSix_Returns400(t *testing.T) {
	h, svc := newTestHandlers(t)
	s, err := svc.CreateForTicket(3)
	require.NoError(t, err)

	body, _ := json.Marshal(map[string]interface{}{"rating": 6})
	r := setupRouter(h)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/survey/"+s.Token, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSubmitSurvey_BadToken_Returns404(t *testing.T) {
	h, _ := newTestHandlers(t)

	body, _ := json.Marshal(map[string]interface{}{"rating": 3})
	r := setupRouter(h)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/survey/badtoken", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestSubmitSurvey_DoubleSubmit_Returns409(t *testing.T) {
	h, svc := newTestHandlers(t)
	s, err := svc.CreateForTicket(4)
	require.NoError(t, err)

	b1, _ := json.Marshal(map[string]interface{}{"rating": 5})
	r := setupRouter(h)
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/survey/"+s.Token, bytes.NewReader(b1))
	req1.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w1, req1)
	require.Equal(t, http.StatusOK, w1.Code)

	b2, _ := json.Marshal(map[string]interface{}{"rating": 3})
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/api/v1/survey/"+s.Token, bytes.NewReader(b2))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusConflict, w2.Code)
}
