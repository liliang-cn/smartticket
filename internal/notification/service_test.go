package notification

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	sqlite "github.com/company/smartticket/internal/database/moderncsqlite"
	"github.com/company/smartticket/internal/models"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Notification{}))
	return db
}

func TestNotify_CreatesRowsDedupeAndSkipZero(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db)

	// 1 and 2 are valid, 0 is skipped, duplicate 1 is deduped.
	svc.Notify(context.Background(), []uint{1, 0, 2, 1}, "ticket_reply", "title", "body", "ticket", 42)

	var rows []models.Notification
	require.NoError(t, db.Order("user_id").Find(&rows).Error)
	require.Len(t, rows, 2)
	assert.Equal(t, uint(1), rows[0].UserID)
	assert.Equal(t, uint(2), rows[1].UserID)
	assert.Equal(t, "ticket_reply", rows[0].Type)
	assert.Equal(t, "ticket", rows[0].RefType)
	assert.Equal(t, uint(42), rows[0].RefID)
	assert.False(t, rows[0].IsRead)

	// Empty / all-zero recipients create nothing.
	svc.Notify(context.Background(), []uint{0, 0}, "x", "t", "b", "ticket", 1)
	var count int64
	require.NoError(t, db.Model(&models.Notification{}).Count(&count).Error)
	assert.Equal(t, int64(2), count)
}

func TestList_NewestFirstAndUnreadOnly(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db)

	// Three notifications for user 1, oldest first.
	for i := 1; i <= 3; i++ {
		require.NoError(t, db.Create(&models.Notification{UserID: 1, Type: "ticket_reply", Title: fmt.Sprintf("n%d", i)}).Error)
	}
	// One for a different user (must not leak).
	require.NoError(t, db.Create(&models.Notification{UserID: 2, Type: "ticket_reply", Title: "other"}).Error)

	items, total, err := svc.List(1, false, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	require.Len(t, items, 3)
	// Newest first: highest id first.
	assert.True(t, items[0].ID > items[1].ID && items[1].ID > items[2].ID)

	// Mark one read, then unreadOnly should drop it.
	require.NoError(t, svc.MarkRead(1, items[0].ID))
	unread, unreadTotal, err := svc.List(1, true, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(2), unreadTotal)
	assert.Len(t, unread, 2)
}

func TestUnreadCount(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db)

	require.NoError(t, db.Create(&models.Notification{UserID: 1, Type: "x"}).Error)
	require.NoError(t, db.Create(&models.Notification{UserID: 1, Type: "x"}).Error)
	require.NoError(t, db.Create(&models.Notification{UserID: 1, Type: "x", IsRead: true}).Error)
	require.NoError(t, db.Create(&models.Notification{UserID: 2, Type: "x"}).Error)

	count, err := svc.UnreadCount(1)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestMarkRead_OnlyOwnersRow(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db)

	mine := &models.Notification{UserID: 1, Type: "x"}
	theirs := &models.Notification{UserID: 2, Type: "x"}
	require.NoError(t, db.Create(mine).Error)
	require.NoError(t, db.Create(theirs).Error)

	// User 1 tries to mark user 2's row read -> no effect.
	require.NoError(t, svc.MarkRead(1, theirs.ID))
	var theirsReloaded models.Notification
	require.NoError(t, db.First(&theirsReloaded, theirs.ID).Error)
	assert.False(t, theirsReloaded.IsRead, "another user's notification must be untouched")

	// User 1 marks their own row read -> effective.
	require.NoError(t, svc.MarkRead(1, mine.ID))
	var mineReloaded models.Notification
	require.NoError(t, db.First(&mineReloaded, mine.ID).Error)
	assert.True(t, mineReloaded.IsRead)
}

func TestMarkAllRead(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db)

	require.NoError(t, db.Create(&models.Notification{UserID: 1, Type: "x"}).Error)
	require.NoError(t, db.Create(&models.Notification{UserID: 1, Type: "x"}).Error)
	require.NoError(t, db.Create(&models.Notification{UserID: 2, Type: "x"}).Error)

	require.NoError(t, svc.MarkAllRead(1))

	c1, err := svc.UnreadCount(1)
	require.NoError(t, err)
	assert.Equal(t, int64(0), c1)

	// User 2 untouched.
	c2, err := svc.UnreadCount(2)
	require.NoError(t, err)
	assert.Equal(t, int64(1), c2)
}
