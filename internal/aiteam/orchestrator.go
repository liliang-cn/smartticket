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
