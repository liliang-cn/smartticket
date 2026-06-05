package macro

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
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Macro{}))
	svc := NewService(db)
	return NewHandlers(svc), svc
}

// authMiddleware injects a fake authenticated user into the gin context.
func authMiddleware(userID uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("user_role", "engineer")
		c.Next()
	}
}

func TestHandlers_CreateAndList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newTestHandlers(t)

	router := gin.New()
	router.Use(authMiddleware(1))
	router.POST("/macros", h.Create)
	router.GET("/macros", h.List)

	// Create a macro.
	body := `{"title":"Quick reply","body":"Hi {{customer.name}}","shared":true}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/macros", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.True(t, resp["success"].(bool))
	data := resp["data"].(map[string]interface{})
	require.Equal(t, "Quick reply", data["title"])

	// List should return the created macro.
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/macros", nil)
	router.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)

	var listResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &listResp))
	items := listResp["data"].([]interface{})
	require.Len(t, items, 1)
}

func TestHandlers_Get(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, svc := newTestHandlers(t)

	m, err := svc.Create(1, CreateRequest{
		Title:  "Test",
		Body:   "Body text",
		Shared: boolPtr(true),
	})
	require.NoError(t, err)

	router := gin.New()
	router.Use(authMiddleware(1))
	router.GET("/macros/:id", h.Get)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/macros/%d", m.ID), nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestHandlers_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, svc := newTestHandlers(t)

	m, err := svc.Create(1, CreateRequest{
		Title:  "To delete",
		Body:   "bye",
		Shared: boolPtr(true),
	})
	require.NoError(t, err)

	router := gin.New()
	router.Use(authMiddleware(1))
	router.DELETE("/macros/:id", h.Delete)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/macros/%d", m.ID), nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}
