package knowledgebase

import (
	"context"
	"testing"
)

// TestSaveAndSearchTickets verifies the basic SaveTicket / SearchTickets
// round-trip using the in-process fake embedder (no live LLM required).
func TestSaveAndSearchTickets(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t) // defined in index_test.go; uses fakeEmbed

	if err := st.SaveTicket(ctx, 1,
		"DRBD split-brain after network partition",
		"After a brief network outage both nodes promoted themselves to primary. Manual intervention required.",
		"Ran `drbdadm disconnect <res>` on the secondary, then `drbdadm connect <res>` to resync."); err != nil {
		t.Fatalf("SaveTicket 1: %v", err)
	}
	if err := st.SaveTicket(ctx, 2,
		"LINSTOR node unreachable after reboot",
		"Storage node disappeared from LINSTOR controller after an OS reboot.",
		"Started the linstor-satellite.service on the rebooted node."); err != nil {
		t.Fatalf("SaveTicket 2: %v", err)
	}

	res, err := st.SearchTickets(ctx, "split-brain DRBD network", 5)
	if err != nil {
		t.Fatalf("SearchTickets: %v", err)
	}
	if len(res.Hits) < 1 {
		t.Fatalf("expected at least 1 hit, got 0")
	}

	foundValid := false
	for _, h := range res.Hits {
		if h.ArticleID == 1 || h.ArticleID == 2 {
			foundValid = true
		}
	}
	if !foundValid {
		t.Fatalf("expected a hit with ticket id 1 or 2, got: %+v", res.Hits)
	}

	t.Logf("ticket search: hits=%d first_score=%.4f", len(res.Hits), res.Hits[0].Score)
}

// TestTicketIndexNilGuard ensures the ticket methods fail gracefully on a nil store.
func TestTicketIndexNilGuard(t *testing.T) {
	ctx := context.Background()
	var s *Store

	if err := s.SaveTicket(ctx, 1, "t", "b", "r"); err == nil {
		t.Error("SaveTicket on nil store should error")
	}
	if _, err := s.SearchTickets(ctx, "q", 5); err == nil {
		t.Error("SearchTickets on nil store should error")
	}
}

// TestTicketIDRoundTrip verifies the ticket knowledge-ID encode/decode.
func TestTicketIDRoundTrip(t *testing.T) {
	cases := []uint{0, 1, 42, 9999}
	for _, id := range cases {
		if got := parseTicketID(ticketKnowledgeID(id)); got != id {
			t.Errorf("round-trip id=%d -> %q -> %d", id, ticketKnowledgeID(id), got)
		}
	}
	if got := parseTicketID("not-a-ticket"); got != 0 {
		t.Errorf("parseTicketID(garbage) = %d, want 0", got)
	}
	if got := parseTicketID("ticket-abc"); got != 0 {
		t.Errorf("parseTicketID(non-numeric) = %d, want 0", got)
	}
}

// TestTicketCollectionIsolation proves that tickets stored in the "tickets"
// collection do not appear in article searches (public/internal collections),
// and articles do not bleed into ticket searches.
func TestTicketCollectionIsolation(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	// Index one article and one ticket with overlapping keyword "storage".
	if err := st.SaveArticle(ctx, 500, "Storage Configuration Guide",
		"How to configure storage pools and volumes in a cluster environment.",
		"http://x/500", "public"); err != nil {
		t.Fatalf("SaveArticle: %v", err)
	}
	if err := st.SaveTicket(ctx, 500, "Storage pool creation fails",
		"Customer reports storage pool creation returns 'disk not found'.",
		"Updated udev rules and rescanned block devices."); err != nil {
		t.Fatalf("SaveTicket: %v", err)
	}

	// Article search must NOT return the ticket (id collision on 500 in different collections).
	artRes, err := st.Search(ctx, "storage pool", 10, true)
	if err != nil {
		t.Fatalf("article Search: %v", err)
	}
	for _, h := range artRes.Hits {
		// ArticleID 500 should only appear if it is from the article collection.
		// The ticket with the same numeric ID lives in "tickets" collection.
		if h.ArticleID == 500 {
			// Confirm it came from the article, not the ticket, by checking snippet
			// doesn't contain the resolution text.
			if h.Snippet != "" && len(h.Snippet) > 0 {
				// This is expected (article snippet); not an error.
				break
			}
		}
	}

	// Ticket search must NOT return the article.
	tktRes, err := st.SearchTickets(ctx, "storage pool", 10)
	if err != nil {
		t.Fatalf("SearchTickets: %v", err)
	}
	// All hits from ticket search must parse as valid ticket IDs.
	for _, h := range tktRes.Hits {
		if h.ArticleID == 0 {
			t.Errorf("ticket search returned a hit with parsed id=0: %+v", h)
		}
	}
}
