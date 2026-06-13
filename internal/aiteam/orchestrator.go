package aiteam

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/company/smartticket/internal/models"
)

// Broadcaster pushes a suggestion update to a ticket's realtime room.
type Broadcaster interface {
	Broadcast(room string, payload []byte)
}

// Orchestrator runs an advisory agent, persists its suggestion, and broadcasts
// the result. It respects the master AI toggle, per-agent toggles, and the
// Sentinel throttle window.
type Orchestrator struct {
	team  *Team
	store *SuggestionStore
	bc    Broadcaster // may be nil (no broadcast)
}

// NewOrchestrator builds an Orchestrator.
func NewOrchestrator(team *Team, store *SuggestionStore, bc Broadcaster) *Orchestrator {
	return &Orchestrator{team: team, store: store, bc: bc}
}

// Run executes agentName for the given TicketContext, persists the suggestion,
// and broadcasts the outcome.  draft is only used by the Reviewer agent.
//
// Returns (nil, nil) in no-op cases (master disabled, per-agent toggle off,
// Sentinel throttle). Returns (*AISuggestion, error) on success or agent
// failure; the caller should log a non-nil error.
func (o *Orchestrator) Run(ctx context.Context, agentName string, tc TicketContext, draft string) (*models.AISuggestion, error) {
	// Gate: master switch + per-agent toggles.
	if o.team.settings != nil {
		set, err := o.team.settings.Get()
		if err != nil {
			return nil, fmt.Errorf("orchestrator: load settings: %w", err)
		}
		if !set.Enabled {
			return nil, nil
		}
		switch agentName {
		case "Triage":
			if !set.TriageEnabled {
				return nil, nil
			}
		case "Sentinel":
			if !set.SentinelEnabled {
				return nil, nil
			}
			// Sentinel throttle: skip if run recently.
			throttleSec := set.SentinelThrottleSec
			if throttleSec <= 0 {
				throttleSec = 60
			}
			existing, err := o.store.GetByTicketAgent(tc.TicketID, "Sentinel")
			if err != nil {
				return nil, fmt.Errorf("orchestrator: sentinel throttle check: %w", err)
			}
			if existing != nil {
				age := time.Since(existing.UpdatedAt)
				if age < time.Duration(throttleSec)*time.Second {
					return nil, nil
				}
			}
		}
		// Researcher, Reviewer, Drafter: always allowed on-demand — no toggle.
	}

	// Persist a pending row so callers can poll immediately.
	sug, err := o.store.Upsert(tc.TicketID, agentName, "pending", 0, "")
	if err != nil {
		return nil, fmt.Errorf("orchestrator: upsert pending: %w", err)
	}

	// Dispatch to the right Run method.
	payload, confidence, runErr := o.dispatch(ctx, agentName, tc, draft)

	if runErr != nil {
		// Mark failed; return the error so callers can log it.
		_, _ = o.store.Upsert(tc.TicketID, agentName, "failed", 0, "")
		return nil, runErr
	}

	// Persist the completed suggestion.
	sug, err = o.store.Upsert(tc.TicketID, agentName, "done", confidence, payload)
	if err != nil {
		return nil, fmt.Errorf("orchestrator: upsert done: %w", err)
	}

	// Broadcast to the ticket realtime room.
	if o.bc != nil {
		notice, _ := json.Marshal(map[string]string{
			"type":   "ai_suggestion",
			"agent":  agentName,
			"status": "done",
		})
		o.bc.Broadcast(fmt.Sprintf("ticket:%d", tc.TicketID), notice)
	}

	return sug, nil
}

// RunAsync starts agentName in the background so the HTTP request returns
// immediately — AI must never block a user action. It synchronously creates a
// "pending" suggestion (returned to the caller so the UI can show "analyzing"),
// then runs the agent in a panic-isolated, timeout-bounded goroutine; the
// completed result is persisted and broadcast over the realtime hub on finish.
// Returns nil when AI is globally disabled (no pending row is created).
func (o *Orchestrator) RunAsync(agentName string, tc TicketContext, draft string) *models.AISuggestion {
	if o.team.settings != nil {
		if set, err := o.team.settings.Get(); err == nil && !set.Enabled {
			return nil
		}
	}
	pending, err := o.store.Upsert(tc.TicketID, agentName, "pending", 0, "")
	if err != nil {
		return nil
	}
	go func() {
		defer func() { _ = recover() }()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		_, _ = o.Run(ctx, agentName, tc, draft)
	}()
	return pending
}

// RecoverPending re-runs advisory suggestions left in "pending" by a crash or HA
// failover, so no ticket is stuck showing "analyzing…" forever. It is meant to
// be called once on startup (in a background goroutine).
//
// agent-go persists each task, but the in-process goroutine that finalizes our
// AISuggestion dies with the process; rather than resume the agent-go task and
// re-thread its id, we simply re-run the agent from the persisted ticket id —
// idempotent (Upsert overwrites the row) and cheap. Reviewer is the exception:
// its draft input isn't persisted, so an orphaned Reviewer run is marked failed
// for the user to re-trigger. Returns the number of suggestions acted on.
func (o *Orchestrator) RecoverPending(buildCtx func(uint) TicketContext) (int, error) {
	pend, err := o.store.ListByStatus("pending")
	if err != nil {
		return 0, err
	}
	n := 0
	for _, sug := range pend {
		if sug.AgentName == "Reviewer" {
			// Can't reconstruct the draft — fail it so the UI stops spinning.
			_, _ = o.store.Upsert(sug.TicketID, "Reviewer", "failed", 0, "")
			n++
			continue
		}
		o.RunAsync(sug.AgentName, buildCtx(sug.TicketID), "")
		n++
	}
	return n, nil
}

// dispatch calls the appropriate Team Run method and returns (jsonPayload,
// confidence, error).
func (o *Orchestrator) dispatch(ctx context.Context, agentName string, tc TicketContext, draft string) (string, float64, error) {
	switch agentName {
	case "Triage":
		res, err := o.team.RunTriage(ctx, tc)
		if err != nil {
			return "", 0, err
		}
		return marshalPayload(res), res.Confidence, nil

	case "Sentinel":
		res, err := o.team.RunSentinel(ctx, tc)
		if err != nil {
			return "", 0, err
		}
		return marshalPayload(res), res.Confidence, nil

	case "Researcher":
		res, err := o.team.RunResearcher(ctx, tc)
		if err != nil {
			return "", 0, err
		}
		return marshalPayload(res), res.Confidence, nil

	case "Reviewer":
		res, err := o.team.RunReviewer(ctx, tc, draft)
		if err != nil {
			return "", 0, err
		}
		return marshalPayload(res), res.Confidence, nil

	case "Drafter":
		res, err := o.team.RunDrafter(ctx, tc)
		if err != nil {
			return "", 0, err
		}
		return marshalPayload(res), res.Confidence, nil

	default:
		return "", 0, fmt.Errorf("orchestrator: unknown agent %q", agentName)
	}
}

// marshalPayload JSON-encodes v; returns "{}" on any error.
func marshalPayload(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}
