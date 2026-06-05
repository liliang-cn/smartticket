package ticket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/company/smartticket/internal/authz"
	"github.com/company/smartticket/internal/database"
	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/sla"
	"github.com/company/smartticket/tests/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupMergeRouter builds a minimal Gin engine with the merge+link routes wired
// and an actor injected via a fake auth middleware.
func setupMergeRouter(t *testing.T, db *database.Database, actor authz.Actor) (*gin.Engine, *Handlers) {
	t.Helper()
	svc := NewService(db.DB, sla.NewCalculator(db.DB))
	h := NewHandlers(svc)

	r := gin.New()

	// Inject actor via middleware (mirrors how the real auth middleware works).
	r.Use(func(c *gin.Context) {
		c.Set("user_id", actor.UserID)
		c.Set("user_role", actor.Role)
		if actor.CustomerID != nil {
			c.Set("user_customer_id", *actor.CustomerID)
		}
		c.Next()
	})

	r.POST("/tickets/:id/merge", h.MergeTicket)
	r.POST("/tickets/:id/links", h.CreateTicketLink)
	r.GET("/tickets/:id/links", h.ListTicketLinks)
	r.DELETE("/tickets/:id/links/:linkId", h.UnlinkTicket)

	return r, h
}

func TestHandlerMerge_Success(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		r, _ := setupMergeRouter(t, db, authz.Actor{Role: authz.RoleAdmin, UserID: 1})

		src := mkTicket(t, db, "open")
		tgt := mkTicket(t, db, "open")

		body, _ := json.Marshal(map[string]uint{"into": tgt.ID})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost,
			fmt.Sprintf("/tickets/%d/merge", src.ID),
			bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.True(t, resp["success"].(bool))

		// Verify source is merged in DB
		var srcTicket models.Ticket
		require.NoError(t, db.DB.First(&srcTicket, src.ID).Error)
		assert.Equal(t, "merged", srcTicket.Status)
	})
}

func TestHandlerMerge_SelfMerge_BadRequest(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		r, _ := setupMergeRouter(t, db, authz.Actor{Role: authz.RoleAdmin, UserID: 1})

		tkt := mkTicket(t, db, "open")

		body, _ := json.Marshal(map[string]uint{"into": tkt.ID})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost,
			fmt.Sprintf("/tickets/%d/merge", tkt.ID),
			bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		// self-merge is a business rule → 422 (UnprocessableEntity)
		assert.True(t, w.Code == http.StatusUnprocessableEntity || w.Code == http.StatusBadRequest,
			"expected 422 or 400, got %d", w.Code)
	})
}

func TestHandlerCreateLink_And_ListLinks(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		r, _ := setupMergeRouter(t, db, authz.Actor{Role: authz.RoleAdmin, UserID: 1})

		a := mkTicket(t, db, "open")
		b := mkTicket(t, db, "open")

		body, _ := json.Marshal(map[string]interface{}{
			"target_id": b.ID,
			"type":      "related",
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost,
			fmt.Sprintf("/tickets/%d/links", a.ID),
			bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		// List
		w2 := httptest.NewRecorder()
		req2, _ := http.NewRequest(http.MethodGet,
			fmt.Sprintf("/tickets/%d/links", a.ID), nil)
		r.ServeHTTP(w2, req2)
		require.Equal(t, http.StatusOK, w2.Code)

		var list map[string]interface{}
		require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &list))
		assert.True(t, list["success"].(bool))
		data := list["data"].([]interface{})
		assert.Len(t, data, 1)
	})
}
