package attachment

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/company/smartticket/internal/authz"
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
	require.NoError(t, db.AutoMigrate(&models.Ticket{}, &models.Attachment{}, &models.Customer{}, &models.User{}))
	return db
}

func newTicket(t *testing.T, db *gorm.DB, customerID *uint) *models.Ticket {
	t.Helper()
	tkt := &models.Ticket{
		Title:      "test",
		CustomerID: customerID,
	}
	require.NoError(t, db.Create(tkt).Error)
	// Unique ticket number per row (avoid the unique index collision).
	tkt.TicketNumber = fmt.Sprintf("T-%d", tkt.ID)
	require.NoError(t, db.Save(tkt).Error)
	return tkt
}

func teamActor() authz.Actor {
	return authz.Actor{UserID: 1, Role: authz.RoleAdmin}
}

func customerActor(customerID uint) authz.Actor {
	return authz.Actor{UserID: 2, Role: authz.RoleCustomer, CustomerID: &customerID}
}

func TestUpload_TeamHappyPath(t *testing.T) {
	db := newTestDB(t)
	tmp := t.TempDir()
	svc := NewService(db, tmp, 1024*1024, nil)

	cid := uint(7)
	tkt := newTicket(t, db, &cid)

	content := "hello attachment world"
	att, err := svc.Upload(teamActor(), tkt.ID, 1, "notes.txt", "text/plain", strings.NewReader(content), int64(len(content)))
	require.NoError(t, err)
	require.NotZero(t, att.ID)
	require.Equal(t, int64(len(content)), att.FileSize)
	require.NotEmpty(t, att.Hash)
	require.Equal(t, "notes.txt", att.OriginalName)

	// File written under tempdir.
	require.True(t, strings.HasPrefix(att.FilePath, tmp), "file path should be under tempdir")
	data, readErr := os.ReadFile(att.FilePath)
	require.NoError(t, readErr)
	require.Equal(t, content, string(data))

	// Expected dir layout.
	expectedDir := filepath.Join(tmp, "attachments", fmt.Sprintf("ticket-%d", tkt.ID))
	require.Equal(t, expectedDir, filepath.Dir(att.FilePath))

	// List returns it.
	list, err := svc.List(teamActor(), tkt.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, att.ID, list[0].ID)

	// Get returns it.
	got, err := svc.Get(teamActor(), att.ID)
	require.NoError(t, err)
	require.Equal(t, att.ID, got.ID)
}

func TestUpload_CustomerIsolation(t *testing.T) {
	db := newTestDB(t)
	tmp := t.TempDir()
	svc := NewService(db, tmp, 1024*1024, nil)

	owner := uint(10)
	other := uint(20)
	tkt := newTicket(t, db, &owner)

	// Customer from a different org cannot upload.
	_, err := svc.Upload(customerActor(other), tkt.ID, 2, "x.txt", "text/plain", strings.NewReader("data"), 4)
	require.Error(t, err)

	// Nothing written.
	var count int64
	require.NoError(t, db.Model(&models.Attachment{}).Count(&count).Error)
	require.Equal(t, int64(0), count)

	// Different-org customer cannot list either.
	_, err = svc.List(customerActor(other), tkt.ID)
	require.Error(t, err)

	// Owning customer can.
	att, err := svc.Upload(customerActor(owner), tkt.ID, 2, "x.txt", "text/plain", strings.NewReader("data"), 4)
	require.NoError(t, err)

	// Other customer cannot Get it.
	_, err = svc.Get(customerActor(other), att.ID)
	require.Error(t, err)
}

func TestUpload_Oversized(t *testing.T) {
	db := newTestDB(t)
	tmp := t.TempDir()
	svc := NewService(db, tmp, 8, nil) // 8 byte limit

	cid := uint(3)
	tkt := newTicket(t, db, &cid)

	content := "this is definitely larger than eight bytes"
	_, err := svc.Upload(teamActor(), tkt.ID, 1, "big.txt", "text/plain", strings.NewReader(content), int64(len(content)))
	require.Error(t, err)
	require.Contains(t, err.Error(), "maximum size")

	// No row.
	var count int64
	require.NoError(t, db.Model(&models.Attachment{}).Count(&count).Error)
	require.Equal(t, int64(0), count)

	// No leftover file.
	dir := filepath.Join(tmp, "attachments", fmt.Sprintf("ticket-%d", tkt.ID))
	entries, _ := os.ReadDir(dir)
	require.Empty(t, entries)
}

func TestUpload_DisallowedExtension(t *testing.T) {
	db := newTestDB(t)
	tmp := t.TempDir()
	svc := NewService(db, tmp, 1024*1024, []string{".png", ".jpg"})

	cid := uint(5)
	tkt := newTicket(t, db, &cid)

	_, err := svc.Upload(teamActor(), tkt.ID, 1, "script.exe", "application/octet-stream", strings.NewReader("MZ"), 2)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not allowed")

	var count int64
	require.NoError(t, db.Model(&models.Attachment{}).Count(&count).Error)
	require.Equal(t, int64(0), count)

	// Allowed extension works.
	_, err = svc.Upload(teamActor(), tkt.ID, 1, "image.png", "image/png", strings.NewReader("PNGDATA"), 7)
	require.NoError(t, err)
}
