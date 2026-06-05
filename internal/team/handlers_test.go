package team

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
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupRouter(t *testing.T) (*gin.Engine, *Service) {
	t.Helper()
	svc := newTestService(t)
	h := NewHandlers(svc)

	r := gin.New()
	r.GET("/teams", h.ListTeams)
	r.POST("/teams", h.CreateTeam)
	r.GET("/teams/:id", h.GetTeam)
	r.PUT("/teams/:id", h.UpdateTeam)
	r.DELETE("/teams/:id", h.DeleteTeam)
	r.GET("/teams/:id/members", h.ListMembers)
	r.POST("/teams/:id/members", h.AddMember)
	r.DELETE("/teams/:id/members/:userId", h.RemoveMember)
	return r, svc
}

func TestHandlers_CreateAndList(t *testing.T) {
	r, _ := setupRouter(t)

	// Create a team.
	body, _ := json.Marshal(map[string]string{"name": "Backend", "description": "backend team"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/teams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var create map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &create))
	assert.True(t, create["success"].(bool))
	data := create["data"].(map[string]interface{})
	assert.Equal(t, "Backend", data["name"])

	// List.
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/teams", nil)
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)

	var list map[string]interface{}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &list))
	assert.True(t, list["success"].(bool))
	arr := list["data"].([]interface{})
	assert.Len(t, arr, 1)
}

func TestHandlers_GetTeam_NotFound(t *testing.T) {
	r, _ := setupRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/teams/99999", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandlers_UpdateTeam(t *testing.T) {
	r, _ := setupRouter(t)

	// Create.
	body, _ := json.Marshal(map[string]string{"name": "OldName"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/teams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var create map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &create))
	id := uint(create["data"].(map[string]interface{})["id"].(float64))

	// Update.
	upBody, _ := json.Marshal(map[string]string{"name": "NewName"})
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/teams/%d", id), bytes.NewReader(upBody))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)

	var upRes map[string]interface{}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &upRes))
	assert.Equal(t, "NewName", upRes["data"].(map[string]interface{})["name"])
}

func TestHandlers_DeleteTeam(t *testing.T) {
	r, _ := setupRouter(t)

	body, _ := json.Marshal(map[string]string{"name": "DeleteMe"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/teams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	var create map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &create))
	id := uint(create["data"].(map[string]interface{})["id"].(float64))

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/teams/%d", id), nil)
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	// Should be gone.
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/teams/%d", id), nil)
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusNotFound, w3.Code)
}

func TestHandlers_Members(t *testing.T) {
	r, svc := setupRouter(t)

	// Create a team.
	body, _ := json.Marshal(map[string]string{"name": "MemberTest"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/teams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	var create map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &create))
	teamID := uint(create["data"].(map[string]interface{})["id"].(float64))

	// Create a user directly in the DB.
	user := createUser(t, svc)

	// Add member.
	addBody, _ := json.Marshal(map[string]uint{"user_id": user.ID})
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/teams/%d/members", teamID), bytes.NewReader(addBody))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	// List members.
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/teams/%d/members", teamID), nil)
	r.ServeHTTP(w3, req3)
	require.Equal(t, http.StatusOK, w3.Code)
	var listRes map[string]interface{}
	require.NoError(t, json.Unmarshal(w3.Body.Bytes(), &listRes))
	members := listRes["data"].([]interface{})
	require.Len(t, members, 1)

	// Remove member.
	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/teams/%d/members/%d", teamID, user.ID), nil)
	r.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusOK, w4.Code)

	// Members list should be empty.
	w5 := httptest.NewRecorder()
	req5, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/teams/%d/members", teamID), nil)
	r.ServeHTTP(w5, req5)
	require.Equal(t, http.StatusOK, w5.Code)
	var listRes2 map[string]interface{}
	require.NoError(t, json.Unmarshal(w5.Body.Bytes(), &listRes2))
	assert.Empty(t, listRes2["data"].([]interface{}))
}
