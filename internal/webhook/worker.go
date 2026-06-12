package webhook

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/company/smartticket/internal/models"
	"gorm.io/gorm"
)

const maxAttempts = 3

// WorkerOptions controls runtime behaviour of the delivery worker.
type WorkerOptions struct {
	BlockPrivateIPs bool
	Interval        time.Duration // default 5s
}

// Worker polls the database for pending/retryable deliveries and dispatches them.
type Worker struct {
	db   *gorm.DB
	opts WorkerOptions
	cli  *http.Client
}

// NewWorker creates a Worker. Call Run to start the polling loop.
func NewWorker(db *gorm.DB, opts WorkerOptions) *Worker {
	if opts.Interval == 0 {
		opts.Interval = 5 * time.Second
	}
	return &Worker{db: db, opts: opts, cli: &http.Client{Timeout: 10 * time.Second}}
}

// Run loops until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	t := time.NewTicker(w.opts.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			w.processOnce()
		}
	}
}

// processOnce attempts every deliverable row once (pending, or failed-but-retryable).
func (w *Worker) processOnce() {
	var rows []models.WebhookDelivery
	w.db.Where("status = ? OR (status = ? AND attempts < ?)", "pending", "failed", maxAttempts).
		Order("created_at ASC").Limit(50).Find(&rows)
	for i := range rows {
		w.deliver(&rows[i])
	}
}

func (w *Worker) deliver(d *models.WebhookDelivery) {
	var wh models.Webhook
	if err := w.db.First(&wh, d.WebhookID).Error; err != nil {
		w.fail(d, 0, "webhook gone")
		return
	}
	if w.opts.BlockPrivateIPs {
		if err := guardSSRF(wh.URL); err != nil {
			w.fail(d, 0, err.Error())
			return
		}
	}
	body := []byte(d.Payload)
	req, err := http.NewRequest(http.MethodPost, wh.URL, bytes.NewReader(body))
	if err != nil {
		w.fail(d, 0, err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-SmartTicket-Event", d.EventType)
	req.Header.Set("X-SmartTicket-Delivery", fmt.Sprintf("%d", d.ID))
	req.Header.Set("X-SmartTicket-Signature", Sign(body, wh.Secret))

	resp, err := w.cli.Do(req)
	now := time.Now()
	d.Attempts++
	d.LastAttemptAt = &now
	if err != nil {
		w.fail(d, 0, err.Error())
		return
	}
	defer resp.Body.Close()
	d.StatusCode = resp.StatusCode
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		d.Status = "success"
		d.Error = ""
	} else {
		d.Status = "failed"
		d.Error = fmt.Sprintf("non-2xx: %d", resp.StatusCode)
	}
	w.db.Save(d)
}

func (w *Worker) fail(d *models.WebhookDelivery, code int, msg string) {
	now := time.Now()
	d.Attempts++
	d.LastAttemptAt = &now
	d.StatusCode = code
	d.Status = "failed"
	d.Error = msg
	w.db.Save(d)
}

// guardSSRF rejects URLs that resolve to private / loopback IP ranges.
func guardSSRF(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return errors.New("bad url")
	}
	host := u.Hostname()
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("dns: %w", err)
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
			return errors.New("destination resolves to a private address")
		}
	}
	return nil
}
