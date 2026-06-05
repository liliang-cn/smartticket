package automation_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	sqlite "github.com/company/smartticket/internal/database/moderncsqlite"
	"github.com/company/smartticket/internal/automation"
	"github.com/company/smartticket/internal/models"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// Use a unique in-memory DB per test to avoid shared state.
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=private", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.AutomationRule{}))
	return db
}

func setupRouter(svc *automation.Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := automation.NewHandlers(svc)
	r.GET("/automations", h.ListRules)
	r.POST("/automations", h.CreateRule)
	r.POST("/automations/reorder", h.ReorderRules)
	r.GET("/automations/:id", h.GetRule)
	r.PUT("/automations/:id", h.UpdateRule)
	r.DELETE("/automations/:id", h.DeleteRule)
	return r
}

func TestHandlers_CreateAndList(t *testing.T) {
	db := setupTestDB(t)
	svc := automation.NewService(db)
	r := setupRouter(svc)

	body := map[string]any{
		"name":    "Test Rule",
		"event":   "ticket.created",
		"enabled": true,
		"match":   "all",
	}
	bs, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/automations", bytes.NewReader(bs))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var createResp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))
	assert.True(t, createResp["success"].(bool))

	// List
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/automations", nil)
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	var listResp map[string]any
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &listResp))
	data := listResp["data"].([]any)
	assert.Len(t, data, 1)
}

func TestHandlers_CreateMissingName(t *testing.T) {
	db := setupTestDB(t)
	svc := automation.NewService(db)
	r := setupRouter(svc)

	body := map[string]any{"event": "ticket.created"}
	bs, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/automations", bytes.NewReader(bs))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// Should fail validation (name required)
	assert.NotEqual(t, http.StatusCreated, w.Code)
}
