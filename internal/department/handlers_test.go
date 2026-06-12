package department

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func strconvFmt(u uint) string { return strconv.FormatUint(uint64(u), 10) }

func TestCreateAndListHandlers(t *testing.T) {
	db := newTestDB(t)
	h := NewHandlers(NewService(db))
	r := gin.New()
	r.POST("/admin/departments", h.Create)
	r.GET("/admin/departments", h.List)

	req := httptest.NewRequest(http.MethodPost, "/admin/departments", strings.NewReader(`{"name":"Support"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	req2 := httptest.NewRequest(http.MethodGet, "/admin/departments", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)
	require.Contains(t, w2.Body.String(), "Support")
}

func TestUpdateCycleReturns400(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db)
	h := NewHandlers(svc)
	root, _ := svc.Create(CreateInput{Name: "root"})
	child, _ := svc.Create(CreateInput{Name: "child", ParentID: &root.ID})

	r := gin.New()
	r.PUT("/admin/departments/:id", h.Update)
	body := `{"parent_id":` + strconvFmt(child.ID) + `}`
	req := httptest.NewRequest(http.MethodPut, "/admin/departments/"+strconvFmt(root.ID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteHandler(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db)
	h := NewHandlers(svc)
	dept, _ := svc.Create(CreateInput{Name: "ToDelete"})

	r := gin.New()
	r.DELETE("/admin/departments/:id", h.Delete)
	req := httptest.NewRequest(http.MethodDelete, "/admin/departments/"+strconvFmt(dept.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "true")
}
