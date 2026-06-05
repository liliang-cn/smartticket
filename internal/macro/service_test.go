package macro

import (
	"fmt"
	"testing"

	sqlite "github.com/company/smartticket/internal/database/moderncsqlite"
	"github.com/company/smartticket/internal/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Macro{}))
	return NewService(db)
}

// TestList_SharedVisibleToAll checks that a shared macro is returned for any userID.
func TestList_SharedVisibleToAll(t *testing.T) {
	svc := newTestService(t)

	// Create a shared macro owned by user 1.
	_, err := svc.Create(1, CreateRequest{
		Title:  "Welcome",
		Body:   "Hello {{customer.name}}",
		Shared: boolPtr(true),
	})
	require.NoError(t, err)

	// User 2 (different user) should still see the shared macro.
	macros, err := svc.List(2)
	require.NoError(t, err)
	require.Len(t, macros, 1)
	require.Equal(t, "Welcome", macros[0].Title)
}

// TestList_PrivateHiddenFromOtherUser checks that a private macro is invisible to non-owners.
func TestList_PrivateHiddenFromOtherUser(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Create(1, CreateRequest{
		Title:  "My private macro",
		Body:   "Secret text",
		Shared: boolPtr(false),
	})
	require.NoError(t, err)

	// User 2 should NOT see user 1's private macro.
	macros, err := svc.List(2)
	require.NoError(t, err)
	require.Empty(t, macros)
}

// TestList_PrivateVisibleToOwner checks that the owner sees their own private macro.
func TestList_PrivateVisibleToOwner(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Create(1, CreateRequest{
		Title:  "My private macro",
		Body:   "Secret text",
		Shared: boolPtr(false),
	})
	require.NoError(t, err)

	macros, err := svc.List(1)
	require.NoError(t, err)
	require.Len(t, macros, 1)
}

// TestCreate_SetsOwnerID checks OwnerID is populated from the acting userID.
func TestCreate_SetsOwnerID(t *testing.T) {
	svc := newTestService(t)

	m, err := svc.Create(42, CreateRequest{
		Title:  "Test",
		Body:   "Body",
		Shared: boolPtr(true),
	})
	require.NoError(t, err)
	require.Equal(t, uint(42), m.OwnerID)
}

// TestDelete_PrivateMacro_OwnerCanDelete checks owner can delete their private macro.
func TestDelete_PrivateMacro_OwnerCanDelete(t *testing.T) {
	svc := newTestService(t)

	m, err := svc.Create(1, CreateRequest{
		Title:  "Private",
		Body:   "text",
		Shared: boolPtr(false),
	})
	require.NoError(t, err)

	err = svc.Delete(1, m.ID)
	require.NoError(t, err)
}

// TestDelete_PrivateMacro_NonOwnerForbidden checks non-owner cannot delete a private macro.
func TestDelete_PrivateMacro_NonOwnerForbidden(t *testing.T) {
	svc := newTestService(t)

	m, err := svc.Create(1, CreateRequest{
		Title:  "Private",
		Body:   "text",
		Shared: boolPtr(false),
	})
	require.NoError(t, err)

	err = svc.Delete(2, m.ID)
	require.Error(t, err) // must be forbidden
}

// TestDelete_SharedMacro_AnyUserCanDelete checks any user can delete a shared macro.
func TestDelete_SharedMacro_AnyUserCanDelete(t *testing.T) {
	svc := newTestService(t)

	m, err := svc.Create(1, CreateRequest{
		Title:  "Shared",
		Body:   "text",
		Shared: boolPtr(true),
	})
	require.NoError(t, err)

	// User 2 (not owner) should be allowed to delete a shared macro.
	err = svc.Delete(2, m.ID)
	require.NoError(t, err)
}

// TestApply_RendersAndIncrementsUsageCount verifies Apply substitutes variables
// and increments UsageCount on each call.
func TestApply_RendersAndIncrementsUsageCount(t *testing.T) {
	svc := newTestService(t)

	m, err := svc.Create(1, CreateRequest{
		Title:  "Greeting",
		Body:   "Hi {{customer.name}}, ticket {{ticket.id}}",
		Shared: boolPtr(true),
	})
	require.NoError(t, err)
	require.Equal(t, 0, m.UsageCount)

	rctx := RenderContext{CustomerName: "Alice", TicketID: "99"}
	rendered, _, err := svc.Apply(m.ID, 1, rctx)
	require.NoError(t, err)
	require.Equal(t, "Hi Alice, ticket 99", rendered)

	// Reload to check persisted counter.
	reloaded, err := svc.Get(1, m.ID)
	require.NoError(t, err)
	require.Equal(t, 1, reloaded.UsageCount)

	// Second apply increments again.
	_, _, err = svc.Apply(m.ID, 1, rctx)
	require.NoError(t, err)
	reloaded2, err := svc.Get(1, m.ID)
	require.NoError(t, err)
	require.Equal(t, 2, reloaded2.UsageCount)
}

// TestApply_PrivateMacro_NonOwnerForbidden checks non-owner cannot apply a private macro.
func TestApply_PrivateMacro_NonOwnerForbidden(t *testing.T) {
	svc := newTestService(t)

	m, err := svc.Create(1, CreateRequest{
		Title:  "Private",
		Body:   "text",
		Shared: boolPtr(false),
	})
	require.NoError(t, err)

	_, _, err = svc.Apply(m.ID, 2, RenderContext{})
	require.Error(t, err)
}

func boolPtr(b bool) *bool { return &b }
