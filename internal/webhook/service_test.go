package webhook

import (
	"fmt"
	"testing"

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
	require.NoError(t, db.AutoMigrate(&models.Webhook{}, &models.WebhookDelivery{}))
	return db
}

func TestEnqueueOnlyMatchingSubscribers(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db)

	_, err := svc.Create(CreateInput{Name: "a", URL: "http://x", Events: []string{"ticket.created"}}, 1)
	require.NoError(t, err)
	_, err = svc.Create(CreateInput{Name: "b", URL: "http://y", Events: []string{"ticket.resolved"}}, 1)
	require.NoError(t, err)

	require.NoError(t, svc.Enqueue("ticket.created", `{"id":1}`))

	var deliveries []models.WebhookDelivery
	require.NoError(t, db.Find(&deliveries).Error)
	require.Len(t, deliveries, 1)
	require.Equal(t, "pending", deliveries[0].Status)
	require.Equal(t, "ticket.created", deliveries[0].EventType)
}

func TestInactiveWebhookNotEnqueued(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db)
	wh, _ := svc.Create(CreateInput{Name: "a", URL: "http://x", Events: []string{"ticket.created"}}, 1)
	require.NoError(t, svc.SetActive(wh.ID, false))
	require.NoError(t, svc.Enqueue("ticket.created", `{}`))
	var n int64
	db.Model(&models.WebhookDelivery{}).Count(&n)
	require.Equal(t, int64(0), n)
}
