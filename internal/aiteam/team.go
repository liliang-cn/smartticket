// Package aiteam wires the 5-member AI advisory team used by SmartTicket's
// ticket-assist features. Each member is a named agent-go specialist that runs
// structured inference via the operator's BYO-LLM.
package aiteam

import (
	"context"

	"github.com/company/smartticket/internal/aiassist"
	"github.com/liliang-cn/agent-go/v2/pkg/agent"
	"github.com/liliang-cn/agent-go/v2/pkg/domain"
	"gorm.io/gorm"
)

const teamName = "support-advisory"

// memberDefs holds the ordered definitions for the 5 advisory specialists.
// The map iteration order is non-deterministic; we keep a slice to ensure
// idempotent registration works across runs.
var memberDefs = []struct {
	name        string
	description string
	instructions string
}{
	{
		name:        "Triage",
		description: "Triage advisory agent",
		instructions: `You triage a new support ticket. Judge priority, severity, a short category, and (if obvious) a suggested team. Be conservative; never invent facts.`,
	},
	{
		name:        "Sentinel",
		description: "Sentinel advisory agent",
		instructions: `You assess escalation risk on a support ticket conversation. Judge customer sentiment, churn risk, SLA-breach risk, and whether to escalate to a manager. Be conservative.`,
	},
	{
		name:        "Researcher",
		description: "Researcher advisory agent",
		instructions: `You help an agent resolve a ticket: find relevant knowledge-base snippets and similar past tickets, and propose a resolution. Use only provided context; never invent.`,
	},
	{
		name:        "Reviewer",
		description: "Reviewer advisory agent",
		instructions: `You review an agent's draft reply before it is sent: flag tone, accuracy, policy and missing-info issues, and optionally provide a revised draft.`,
	},
	{
		name:        "Drafter",
		description: "Drafter advisory agent",
		instructions: `You draft the agent's next reply to the customer: clear, friendly, professional. Never invent facts or commitments.`,
	},
}

// Team wraps an agent-go TeamManager (member roster) plus the BYO-LLM generator
// and KB tool used for structured inference by the agent run methods.
type Team struct {
	mgr      *agent.TeamManager
	gen      domain.Generator
	kb       aiassist.KBSearcher
	settings *aiassist.SettingsStore
	db       *gorm.DB
}

// NewTeam builds the agent-go team and registers the 5 specialists (idempotent).
// dbPath is agent-go's own SQLite store (e.g. "./data/agentgo-team.db").
func NewTeam(dbPath string, gen domain.Generator, kb aiassist.KBSearcher, settings *aiassist.SettingsStore, db *gorm.DB) (*Team, error) {
	store, err := agent.NewStore(dbPath)
	if err != nil {
		return nil, err
	}
	mgr := agent.NewTeamManager(store)
	mgr.SetLLM(gen)
	mgr.SetDisableMemory(true)
	t := &Team{mgr: mgr, gen: gen, kb: kb, settings: settings, db: db}
	if err := t.ensureMembers(context.Background()); err != nil {
		return nil, err
	}
	return t, nil
}

// ensureMembers creates the team and 5 specialist members if they do not
// already exist. Safe to call on every startup (idempotent).
func (t *Team) ensureMembers(ctx context.Context) error {
	team, err := t.mgr.GetTeamByName(teamName)
	if err != nil || team == nil {
		// Team does not exist yet — create it. CreateTeam also creates an
		// orchestrator agent automatically; that's fine.
		team, err = t.mgr.CreateTeam(ctx, &agent.Team{
			Name:        teamName,
			Description: "SmartTicket AI advisory team",
		})
		if err != nil {
			return err
		}
	}

	for _, def := range memberDefs {
		m, merr := t.mgr.GetMemberByName(def.name)
		if merr == nil && m != nil {
			// Already registered.
			continue
		}
		if _, err := t.mgr.AddSpecialist(ctx, team.ID, def.name, def.description, def.instructions); err != nil {
			return err
		}
	}
	return nil
}

// Members returns all registered team members (specialists + orchestrator).
func (t *Team) Members() ([]*agent.AgentModel, error) { return t.mgr.ListMembers() }
