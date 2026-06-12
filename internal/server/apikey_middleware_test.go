package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/company/smartticket/internal/apikey"
	"github.com/company/smartticket/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func buildKeyTestServer(t *testing.T) (*Server, *gin.Engine) {
	t.Helper()
	db := createTestDB(t)
	require.NoError(t, db.DB.AutoMigrate(&models.User{}, &models.APIKey{}))
	cfg := createTestConfig()
	cfg.Environment = "production" // prevent X-Skip-Auth dev shortcut from interfering
	s := &Server{
		config:        cfg,
		db:            db,
		apiKeyService: apikey.NewService(db.DB),
	}
	r := gin.New()
	r.GET("/whoami", s.authMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user_id": c.GetUint("user_id"), "role": c.GetString("user_role")})
	})
	return s, r
}

func issueKey(t *testing.T, s *Server, role string) string {
	t.Helper()
	u := models.User{Email: role + "@svc.local", Username: "svc_" + role, PasswordHash: "-", Role: role, IsActive: true}
	require.NoError(t, s.db.DB.Create(&u).Error)
	pt, _, err := s.apiKeyService.Create("test", u.ID, nil, 1)
	require.NoError(t, err)
	return pt
}

func TestAPIKeyAuthResolvesUser(t *testing.T) {
	s, r := buildKeyTestServer(t)
	key := issueKey(t, s, "admin")

	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	req.Header.Set("Authorization", "Bearer "+key)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"role":"admin"`)
}

func TestInvalidAPIKeyRejected(t *testing.T) {
	_, r := buildKeyTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	req.Header.Set("Authorization", "Bearer stk_live_bogus")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}
