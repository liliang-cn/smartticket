package webhook

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/company/smartticket/internal/models"
	"github.com/stretchr/testify/require"
)

func TestWorkerDeliversPendingWithSignature(t *testing.T) {
	db := newTestDB(t)
	var hits int32
	var gotSig string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		gotSig = r.Header.Get("X-SmartTicket-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	svc := NewService(db)
	wh, _ := svc.Create(CreateInput{Name: "a", URL: srv.URL, Events: []string{"ticket.created"}}, 1)
	require.NoError(t, svc.Enqueue("ticket.created", `{"id":1}`))

	w := NewWorker(db, WorkerOptions{BlockPrivateIPs: false})
	w.processOnce()

	require.Equal(t, int32(1), atomic.LoadInt32(&hits))
	require.Contains(t, gotSig, "sha256=")

	var d models.WebhookDelivery
	require.NoError(t, db.Where("webhook_id = ?", wh.ID).First(&d).Error)
	require.Equal(t, "success", d.Status)
	require.Equal(t, http.StatusOK, d.StatusCode)
}

func TestWorkerMarksFailedAfterMaxAttempts(t *testing.T) {
	db := newTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	svc := NewService(db)
	svc.Create(CreateInput{Name: "a", URL: srv.URL, Events: []string{"e"}}, 1)
	require.NoError(t, svc.Enqueue("e", `{}`))

	w := NewWorker(db, WorkerOptions{BlockPrivateIPs: false})
	for i := 0; i < maxAttempts+1; i++ {
		w.processOnce()
		time.Sleep(time.Millisecond)
	}
	var d models.WebhookDelivery
	require.NoError(t, db.First(&d).Error)
	require.Equal(t, "failed", d.Status)
	require.Equal(t, maxAttempts, d.Attempts)
}

// A transport-level failure (connection refused) must increment Attempts
// exactly once per pass — not twice — so the retry cap is honored precisely.
func TestWorkerTransportFailureCountsOncePerPass(t *testing.T) {
	db := newTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	deadURL := srv.URL
	srv.Close() // now connections are refused

	svc := NewService(db)
	svc.Create(CreateInput{Name: "a", URL: deadURL, Events: []string{"e"}}, 1)
	require.NoError(t, svc.Enqueue("e", `{}`))

	w := NewWorker(db, WorkerOptions{BlockPrivateIPs: false})
	for i := 0; i < maxAttempts+1; i++ {
		w.processOnce()
	}
	var d models.WebhookDelivery
	require.NoError(t, db.First(&d).Error)
	require.Equal(t, "failed", d.Status)
	require.Equal(t, maxAttempts, d.Attempts) // exactly maxAttempts, proving single increment
}
