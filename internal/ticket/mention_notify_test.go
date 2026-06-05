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

// TestMentionNotify_InternalNoteWithMention verifies that an internal message
// containing "@alice" causes alice to receive a ticket_mention notification.
func TestMentionNotify_InternalNoteWithMention(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		svc := NewService(db.DB, sla.NewCalculator(db.DB))
		fake := &fakeNotifier{}
		svc.SetNotifier(fake)

		alice := &models.User{
			Email: "alice@test.io", Username: "alice",
			PasswordHash: "x", Role: authz.RoleEngineer, IsActive: true,
		}
		require.NoError(t, db.DB.Create(alice).Error)

		tk := &models.Ticket{
			TicketNumber: generateTicketNumber(),
			Title: "T", Status: "open", Priority: "medium", Severity: "minor",
		}
		require.NoError(t, db.DB.Create(tk).Error)

		teamActor := authz.Actor{UserID: 99, Role: authz.RoleAdmin}
		_, err := svc.CreateMessage(teamActor, tk.ID, 99, &CreateMessageRequest{
			Content:    "@alice look at this",
			IsInternal: true,
		})
		require.NoError(t, err)

		// Find the mention notification among all captured calls.
		var mentionCalls []capturedNotify
		for _, c := range fake.calls {
			if c.ntype == "ticket_mention" {
				mentionCalls = append(mentionCalls, c)
			}
		}
		require.Len(t, mentionCalls, 1, "alice should receive exactly one mention notification")
		assert.Contains(t, mentionCalls[0].userIDs, alice.ID)
		assert.Equal(t, tk.ID, mentionCalls[0].refID)
		assert.Equal(t, "ticket", mentionCalls[0].refType)
	})
}

// TestMentionNotify_UnknownHandleIgnored verifies that a handle that does not
// match any user does not produce a notification or an error.
func TestMentionNotify_UnknownHandleIgnored(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		svc := NewService(db.DB, sla.NewCalculator(db.DB))
		fake := &fakeNotifier{}
		svc.SetNotifier(fake)

		tk := &models.Ticket{
			TicketNumber: generateTicketNumber(),
			Title: "T", Status: "open", Priority: "medium", Severity: "minor",
		}
		require.NoError(t, db.DB.Create(tk).Error)

		teamActor := authz.Actor{UserID: 1, Role: authz.RoleAdmin}
		_, err := svc.CreateMessage(teamActor, tk.ID, 1, &CreateMessageRequest{
			Content:    "@nobody_exists please check",
			IsInternal: true,
		})
		require.NoError(t, err)

		for _, c := range fake.calls {
			assert.NotEqual(t, "ticket_mention", c.ntype,
				"no mention notification expected for unknown handle")
		}
	})
}

// TestMentionNotify_PublicMessageNoMention verifies that @mentions in a PUBLIC
// message do NOT produce mention notifications.
func TestMentionNotify_PublicMessageNoMention(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		svc := NewService(db.DB, sla.NewCalculator(db.DB))
		fake := &fakeNotifier{}
		svc.SetNotifier(fake)

		alice := &models.User{
			Email: "alice2@test.io", Username: "alice2",
			PasswordHash: "x", Role: authz.RoleEngineer, IsActive: true,
		}
		require.NoError(t, db.DB.Create(alice).Error)

		code := "PM1"
		cust := &models.Customer{Name: "C", Code: &code, IsActive: true}
		require.NoError(t, db.DB.Create(cust).Error)
		cid := cust.ID

		tk := &models.Ticket{
			TicketNumber: generateTicketNumber(),
			Title: "T", Status: "open", Priority: "medium", Severity: "minor",
			CustomerID: &cid,
		}
		require.NoError(t, db.DB.Create(tk).Error)

		teamActor := authz.Actor{UserID: 99, Role: authz.RoleAdmin}
		_, err := svc.CreateMessage(teamActor, tk.ID, 99, &CreateMessageRequest{
			Content:    "@alice2 public reply",
			IsInternal: false, // PUBLIC message
		})
		require.NoError(t, err)

		for _, c := range fake.calls {
			assert.NotEqual(t, "ticket_mention", c.ntype,
				"public messages must NOT produce mention notifications")
		}
	})
}

// TestMentionNotify_AuthorNotNotified verifies that an author who mentions
// themselves does not receive a self-notification.
func TestMentionNotify_AuthorNotNotified(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		svc := NewService(db.DB, sla.NewCalculator(db.DB))
		fake := &fakeNotifier{}
		svc.SetNotifier(fake)

		bob := &models.User{
			Email: "bob@test.io", Username: "bob",
			PasswordHash: "x", Role: authz.RoleEngineer, IsActive: true,
		}
		require.NoError(t, db.DB.Create(bob).Error)

		tk := &models.Ticket{
			TicketNumber: generateTicketNumber(),
			Title: "T", Status: "open", Priority: "medium", Severity: "minor",
		}
		require.NoError(t, db.DB.Create(tk).Error)

		// Bob (author) mentions himself in an internal note.
		bobActor := authz.Actor{UserID: bob.ID, Role: authz.RoleEngineer}
		_, err := svc.CreateMessage(bobActor, tk.ID, bob.ID, &CreateMessageRequest{
			Content:    "@bob self-mention",
			IsInternal: true,
		})
		require.NoError(t, err)

		for _, c := range fake.calls {
			if c.ntype == "ticket_mention" {
				for _, uid := range c.userIDs {
					assert.NotEqual(t, bob.ID, uid, "author must not receive self-mention notification")
				}
			}
		}
	})
}

// TestMentionNotify_CaseInsensitiveHandleMatch verifies that @ALICE matches a
// user with username "alice" (case-insensitive lookup).
func TestMentionNotify_CaseInsensitiveHandleMatch(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		svc := NewService(db.DB, sla.NewCalculator(db.DB))
		fake := &fakeNotifier{}
		svc.SetNotifier(fake)

		carol := &models.User{
			Email: "carol@test.io", Username: "carol",
			PasswordHash: "x", Role: authz.RoleEngineer, IsActive: true,
		}
		require.NoError(t, db.DB.Create(carol).Error)

		tk := &models.Ticket{
			TicketNumber: generateTicketNumber(),
			Title: "T", Status: "open", Priority: "medium", Severity: "minor",
		}
		require.NoError(t, db.DB.Create(tk).Error)

		teamActor := authz.Actor{UserID: 99, Role: authz.RoleAdmin}
		_, err := svc.CreateMessage(teamActor, tk.ID, 99, &CreateMessageRequest{
			Content:    "@CAROL please review",
			IsInternal: true,
		})
		require.NoError(t, err)

		var mentionCalls []capturedNotify
		for _, c := range fake.calls {
			if c.ntype == "ticket_mention" {
				mentionCalls = append(mentionCalls, c)
			}
		}
		require.Len(t, mentionCalls, 1)
		assert.Contains(t, mentionCalls[0].userIDs, carol.ID)
	})
}
