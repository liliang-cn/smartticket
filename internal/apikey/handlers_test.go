package apikey

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCreateHandlerReturnsPlaintextOnce(t *testing.T) {
	db := newTestDB(t)
	u := seedUser(t, db)
	h := NewHandlers(NewService(db))

	r := gin.New()
	r.POST("/admin/api-keys", func(c *gin.Context) { c.Set("user_id", uint(1)); h.Create(c) })

	body := `{"name":"Zapier","user_id":` + strconv.FormatUint(uint64(u.ID), 10) + `}`
	req := httptest.NewRequest(http.MethodPost, "/admin/api-keys", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	plaintext := resp["key"].(string)
	require.Contains(t, plaintext, "stk_live_")

	// The list shows the safe 12-char KeyPrefix but MUST NOT leak the full
	// plaintext secret. The short prefix substring is fine; the whole key is not.
	r.GET("/admin/api-keys", h.List)
	req2 := httptest.NewRequest(http.MethodGet, "/admin/api-keys", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.NotContains(t, w2.Body.String(), plaintext)
	require.Contains(t, w2.Body.String(), `"key_prefix":"stk_live_`) // safe display hint present
}
