package aiteam

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"

	"github.com/liliang-cn/agent-go/v2/pkg/agent"
	taskpkg "github.com/liliang-cn/agent-go/v2/pkg/task"
)

// aiTaskConcurrency bounds how many advisory-agent tasks run at once. agent-go's
// task queue spawns a goroutine per submit with no cap, so a burst of tickets or
// messages could otherwise fan out into unbounded concurrent LLM + embedding
// calls (provider rate limits / memory). Kept small (single-tenant workload).
func aiTaskConcurrency() int {
	n := runtime.NumCPU() - 2
	if n < 2 {
		return 2
	}
	if n > 4 {
		return 4
	}
	return n
}

// aiTaskSlots is the package-wide semaphore enforcing aiTaskConcurrency().
var aiTaskSlots = make(chan struct{}, aiTaskConcurrency())

// specFromSchema builds an agent-go StructuredOutputSpec from one of our existing
// JSON-schema maps. The schema is reused verbatim so behaviour matches the old
// prompt-based path; Strict is left false so the runtime returns best-effort
// output once the lint retry budget is exhausted (graceful degradation) rather
// than blocking the task.
func specFromSchema(name, description string, schema map[string]interface{}) *agent.StructuredOutputSpec {
	raw, err := json.Marshal(schema)
	if err != nil {
		// Should never happen for our static schema literals.
		raw = json.RawMessage(`{"type":"object"}`)
	}
	return &agent.StructuredOutputSpec{
		Name:        name,
		Schema:      json.RawMessage(raw),
		Description: description,
		Strict:      false,
	}
}

// Per-agent output specs, built once from the schema maps in agents.go.
var (
	triageSpec    = specFromSchema("triage", "Ticket triage verdict.", triageSchema)
	sentinelSpec  = specFromSchema("sentinel", "Escalation-risk assessment.", sentinelSchema)
	researcherSpec = specFromSchema("researcher", "Proposed resolution from provided context.", researcherSchema)
	reviewerSpec  = specFromSchema("reviewer", "Draft-reply review verdict.", reviewerSchema)
	drafterSpec   = specFromSchema("drafter", "Drafted customer reply.", drafterSchema)
)

// structured submits a schema-validated background Task to the named specialist
// via the real TeamManager task queue, awaits its terminal state, and returns
// the parsed output map.
//
// The task runs through agent-go's StructuredOutput machinery (native
// response_format where the provider supports it, plus a schema-validating lint
// with bounded retry). It is persisted to agent-go's SQLite store, so it
// survives a restart and can be reconciled on startup.
//
// valid is false when the model produced no schema-conforming output — callers
// fall back to a zero-value result with Confidence 0 (graceful degradation),
// mirroring the previous GenerateStructured behaviour.
func (t *Team) structured(ctx context.Context, agentName, prompt string, spec *agent.StructuredOutputSpec) (map[string]interface{}, bool, error) {
	// Backpressure: wait for a concurrency slot (or ctx cancellation).
	select {
	case aiTaskSlots <- struct{}{}:
		defer func() { <-aiTaskSlots }()
	case <-ctx.Done():
		return nil, false, ctx.Err()
	}

	submitted, err := t.mgr.Tasks().Submit(ctx, agent.TaskSubmitOptions{
		SessionID:    "aiteam-" + strings.ToLower(agentName),
		AgentName:    agentName,
		Input:        prompt,
		OutputSchema: spec,
	})
	if err != nil {
		return nil, false, fmt.Errorf("aiteam: submit %s task: %w", agentName, err)
	}

	done, err := t.mgr.Tasks().Await(ctx, submitted.ID)
	if err != nil {
		return nil, false, fmt.Errorf("aiteam: await %s task: %w", agentName, err)
	}

	switch done.Status {
	case taskpkg.StatusFailed:
		// A structured-output lint rejection means the model produced no
		// schema-conforming JSON after the retry budget — degrade gracefully
		// (valid=false), matching the old GenerateStructured Valid:false path.
		// Any other failure (e.g. the LLM call itself errored) propagates.
		if isLintRejection(done.Error) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("aiteam: %s task failed: %s", agentName, done.Error)
	case taskpkg.StatusCancelled:
		return nil, false, fmt.Errorf("aiteam: %s task cancelled", agentName)
	}

	out := strings.TrimSpace(done.Output)
	if out == "" {
		return nil, false, nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		// Best-effort text that isn't clean JSON — degrade gracefully.
		return nil, false, nil
	}
	return m, true, nil
}

// isLintRejection reports whether a task-failure message is the structured-output
// lint rejecting non-conforming model output (vs. a genuine execution error).
func isLintRejection(msg string) bool {
	return strings.Contains(msg, "structured_output") || strings.Contains(msg, "output lint")
}
