package ticket

import (
	"testing"

	"github.com/company/smartticket/internal/authz"
	"github.com/company/smartticket/internal/database"
	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/sla"
	"github.com/company/smartticket/tests/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- helpers ---

func newMergeService(t *testing.T, db *database.Database) *Service {
	t.Helper()
	return NewService(db.DB, sla.NewCalculator(db.DB))
}

func mkTicket(t *testing.T, db *database.Database, status string) *models.Ticket {
	t.Helper()
	tkt := &models.Ticket{
		TicketNumber:   generateTicketNumber(),
		Title:          "Ticket " + status,
		Description:    "desc",
		Status:         status,
		Priority:       "medium",
		Severity:       "minor",
		RequesterName:  "Req",
		RequesterEmail: "req@example.com",
	}
	require.NoError(t, db.DB.Create(tkt).Error)
	return tkt
}

func mkMessage(t *testing.T, db *database.Database, ticketID uint) *models.Message {
	t.Helper()
	m := &models.Message{TicketID: ticketID, Content: "msg", ContentType: "text"}
	require.NoError(t, db.DB.Create(m).Error)
	return m
}

func mkAttachment(t *testing.T, db *database.Database, ticketID uint) *models.Attachment {
	t.Helper()
	a := &models.Attachment{
		TicketID:     ticketID,
		FileName:     "file.txt",
		OriginalName: "file.txt",
		FilePath:     "/tmp/file.txt",
		FileSize:     100,
		ContentType:  "text/plain",
	}
	require.NoError(t, db.DB.Create(a).Error)
	return a
}

var adminActor = authz.Actor{Role: authz.RoleAdmin, UserID: 1}
var customerActor = authz.Actor{Role: authz.RoleCustomer, CustomerID: func() *uint { v := uint(1); return &v }()}

// --- Merge tests ---

func TestMerge_MovesMessagesAndAttachmentsToTarget(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		svc := newMergeService(t, db)

		src := mkTicket(t, db, "open")
		tgt := mkTicket(t, db, "open")
		m1 := mkMessage(t, db, src.ID)
		m2 := mkMessage(t, db, src.ID)
		att := mkAttachment(t, db, src.ID)

		err := svc.Merge(adminActor, src.ID, tgt.ID)
		require.NoError(t, err)

		// Messages reassigned to target
		var msg1, msg2 models.Message
		require.NoError(t, db.DB.First(&msg1, m1.ID).Error)
		assert.Equal(t, tgt.ID, msg1.TicketID)

		require.NoError(t, db.DB.First(&msg2, m2.ID).Error)
		assert.Equal(t, tgt.ID, msg2.TicketID)

		// Attachment reassigned
		var a models.Attachment
		require.NoError(t, db.DB.First(&a, att.ID).Error)
		assert.Equal(t, tgt.ID, a.TicketID)

		// Source: status=merged, MergedIntoID=target
		var src2 models.Ticket
		require.NoError(t, db.DB.First(&src2, src.ID).Error)
		assert.Equal(t, "merged", src2.Status)
		require.NotNil(t, src2.MergedIntoID)
		assert.Equal(t, tgt.ID, *src2.MergedIntoID)

		// Events recorded on both tickets
		var srcEvents []models.TicketEvent
		require.NoError(t, db.DB.Where("ticket_id = ?", src.ID).Find(&srcEvents).Error)
		assert.NotEmpty(t, srcEvents)

		var tgtEvents []models.TicketEvent
		require.NoError(t, db.DB.Where("ticket_id = ?", tgt.ID).Find(&tgtEvents).Error)
		assert.NotEmpty(t, tgtEvents)
	})
}

func TestMerge_SelfMergeRejected(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		svc := newMergeService(t, db)
		tkt := mkTicket(t, db, "open")

		err := svc.Merge(adminActor, tkt.ID, tkt.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "self")
	})
}

func TestMerge_AlreadyMergedSourceRejected(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		svc := newMergeService(t, db)
		src := mkTicket(t, db, "merged")
		tgt := mkTicket(t, db, "open")

		err := svc.Merge(adminActor, src.ID, tgt.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already merged")
	})
}

func TestMerge_TargetAlreadyMergedRejected(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		svc := newMergeService(t, db)
		src := mkTicket(t, db, "open")
		tgt := mkTicket(t, db, "merged")

		err := svc.Merge(adminActor, src.ID, tgt.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already merged")
	})
}

func TestMerge_CustomerActorRejected(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		svc := newMergeService(t, db)
		src := mkTicket(t, db, "open")
		tgt := mkTicket(t, db, "open")

		err := svc.Merge(customerActor, src.ID, tgt.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "only team")
	})
}

// --- LinkTickets tests ---

func TestLinkTickets_CreateAndList(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		svc := newMergeService(t, db)
		a := mkTicket(t, db, "open")
		b := mkTicket(t, db, "open")

		link, err := svc.LinkTickets(adminActor, a.ID, b.ID, "related")
		require.NoError(t, err)
		require.NotNil(t, link)
		assert.Equal(t, a.ID, link.SourceID)
		assert.Equal(t, b.ID, link.TargetID)
		assert.Equal(t, "related", link.Type)

		// List links for ticket a
		links, err := svc.ListLinks(adminActor, a.ID)
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, link.ID, links[0].ID)

		// List links for ticket b (should also appear, it's the target)
		links2, err := svc.ListLinks(adminActor, b.ID)
		require.NoError(t, err)
		require.Len(t, links2, 1)
		assert.Equal(t, link.ID, links2[0].ID)
	})
}

func TestLinkTickets_SelfLinkRejected(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		svc := newMergeService(t, db)
		a := mkTicket(t, db, "open")

		_, err := svc.LinkTickets(adminActor, a.ID, a.ID, "related")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "self")
	})
}

func TestLinkTickets_InvalidTypeRejected(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		svc := newMergeService(t, db)
		a := mkTicket(t, db, "open")
		b := mkTicket(t, db, "open")

		_, err := svc.LinkTickets(adminActor, a.ID, b.ID, "bogus")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid link type")
	})
}

func TestLinkTickets_DuplicateLinkGraceful(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		svc := newMergeService(t, db)
		a := mkTicket(t, db, "open")
		b := mkTicket(t, db, "open")

		link1, err := svc.LinkTickets(adminActor, a.ID, b.ID, "duplicate")
		require.NoError(t, err)

		// Second call with same (source, target, type) should return the existing link (no error)
		link2, err := svc.LinkTickets(adminActor, a.ID, b.ID, "duplicate")
		require.NoError(t, err)
		require.NotNil(t, link2)
		assert.Equal(t, link1.ID, link2.ID, "idempotent: must return the existing link")
	})
}

func TestLinkTickets_CustomerActorRejected(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		svc := newMergeService(t, db)
		a := mkTicket(t, db, "open")
		b := mkTicket(t, db, "open")

		_, err := svc.LinkTickets(customerActor, a.ID, b.ID, "related")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "only team")
	})
}

// --- Unlink tests ---

func TestUnlink(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		svc := newMergeService(t, db)
		a := mkTicket(t, db, "open")
		b := mkTicket(t, db, "open")

		link, err := svc.LinkTickets(adminActor, a.ID, b.ID, "blocks")
		require.NoError(t, err)

		err = svc.Unlink(adminActor, a.ID, link.ID)
		require.NoError(t, err)

		// Link should be gone
		links, err := svc.ListLinks(adminActor, a.ID)
		require.NoError(t, err)
		assert.Empty(t, links)
	})
}

func TestUnlink_CustomerRejected(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		svc := newMergeService(t, db)
		a := mkTicket(t, db, "open")
		b := mkTicket(t, db, "open")

		link, err := svc.LinkTickets(adminActor, a.ID, b.ID, "blocks")
		require.NoError(t, err)

		err = svc.Unlink(customerActor, a.ID, link.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "only team")
	})
}

func TestUnlink_NotFound(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		svc := newMergeService(t, db)
		a := mkTicket(t, db, "open")

		err := svc.Unlink(adminActor, a.ID, 99999)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestUnlink_WrongTicket verifies that passing a ticketID that is not part of
// the link returns NotFound and does NOT delete the link.
func TestUnlink_WrongTicket(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		svc := newMergeService(t, db)
		a := mkTicket(t, db, "open")
		b := mkTicket(t, db, "open")
		c := mkTicket(t, db, "open") // unrelated ticket

		link, err := svc.LinkTickets(adminActor, a.ID, b.ID, "related")
		require.NoError(t, err)

		// Attempt to unlink using c's ID — c is not part of the link.
		err = svc.Unlink(adminActor, c.ID, link.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		// The link must still exist.
		links, err := svc.ListLinks(adminActor, a.ID)
		require.NoError(t, err)
		require.Len(t, links, 1, "link must not have been deleted")
	})
}
