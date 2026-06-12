package webhook

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCreateAndListHandlers(t *testing.T) {
	db := newTestDB(t)
	h := NewHandlers(NewService(db))
	r := gin.New()
	r.POST("/admin/webhooks", func(c *gin.Context) { c.Set("user_id", uint(1)); h.Create(c) })
	r.GET("/admin/webhooks", h.List)

	body := `{"name":"a","url":"http://x","events":["ticket.created"]}`
	req := httptest.NewRequest(http.MethodPost, "/admin/webhooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	require.Contains(t, w.Body.String(), `"secret"`)

	req2 := httptest.NewRequest(http.MethodGet, "/admin/webhooks", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.NotContains(t, w2.Body.String(), `"secret"`)
}
