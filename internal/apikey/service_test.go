package apikey

import (
	"fmt"
	"testing"
	"time"

	sqlite "github.com/company/smartticket/internal/database/moderncsqlite"
	"github.com/company/smartticket/internal/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.APIKey{}))
	return db
}

func seedUser(t *testing.T, db *gorm.DB) models.User {
	t.Helper()
	u := models.User{Email: "svc@x.local", Username: "svc", PasswordHash: "-", Role: "engineer", IsActive: true}
	require.NoError(t, db.Create(&u).Error)
	return u
}

func TestCreateAndAuthenticateRoundTrip(t *testing.T) {
	db := newTestDB(t)
	u := seedUser(t, db)
	svc := NewService(db)

	plaintext, key, err := svc.Create("Zapier", u.ID, nil, 99)
	require.NoError(t, err)
	require.True(t, len(plaintext) > 20)
	require.Contains(t, plaintext, "stk_live_")
	require.Equal(t, plaintext[:12], key.KeyPrefix)

	got, err := svc.Authenticate(plaintext)
	require.NoError(t, err)
	require.Equal(t, u.ID, got.ID)
}

func TestAuthenticateRejectsUnknownRevokedExpired(t *testing.T) {
	db := newTestDB(t)
	u := seedUser(t, db)
	svc := NewService(db)

	_, err := svc.Authenticate("stk_live_doesnotexist")
	require.Error(t, err)

	pt, key, _ := svc.Create("k", u.ID, nil, 1)
	require.NoError(t, svc.Revoke(key.ID))
	_, err = svc.Authenticate(pt)
	require.ErrorIs(t, err, ErrRevoked)

	past := time.Now().Add(-time.Hour)
	pt2, _, _ := svc.Create("k2", u.ID, &past, 1)
	_, err = svc.Authenticate(pt2)
	require.ErrorIs(t, err, ErrExpired)
}

// A key bound to a deactivated user must stop authenticating, mirroring the
// JWT path which rejects inactive accounts on every request.
func TestAuthenticateRejectsDeactivatedUser(t *testing.T) {
	db := newTestDB(t)
	u := seedUser(t, db)
	svc := NewService(db)

	pt, _, err := svc.Create("k", u.ID, nil, 1)
	require.NoError(t, err)

	// Works while the user is active.
	_, err = svc.Authenticate(pt)
	require.NoError(t, err)

	// Deactivate the bound service account.
	require.NoError(t, db.Model(&models.User{}).Where("id = ?", u.ID).Update("is_active", false).Error)

	_, err = svc.Authenticate(pt)
	require.ErrorIs(t, err, ErrInvalid)
}
