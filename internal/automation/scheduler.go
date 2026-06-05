package automation

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/company/smartticket/internal/logger"
)

// TicketRepo is the minimal persistence interface the Scheduler needs.
// Implemented by the automation Service (which wraps GORM) in service.go.
type TicketRepo interface {
	// OverdueTicketIDs returns IDs of open tickets whose SLA due_date is in the past.
	OverdueTicketIDs(now time.Time) ([]uint, error)
	// SilentCustomerTicketIDs returns IDs of in-progress tickets where the last
	// customer message is older than windowSeconds.
	SilentCustomerTicketIDs(now time.Time, windowSeconds int64) ([]uint, error)
}

// AutoResolveSettingsReader is a narrow interface the Scheduler uses to read the
// live AutoResolveEnabled flag each tick. Satisfied by *aiassist.SettingsStore
// (which is NOT imported here to avoid a cycle — aiassist already imports automation).
type AutoResolveSettingsReader interface {
	// AutoResolveEnabled returns whether the auto-resolve feature is enabled.
	// Returns false on any error so the scheduler is safe to call with no DB.
	AutoResolveEnabled() bool
}

// SchedulerConfig controls auto-resolve behaviour.
type SchedulerConfig struct {
	// SilentWindowSeconds is the inactivity window in seconds. Defaults to 86400 (24 h).
	SilentWindowSeconds int64
	// Settings is an optional live reader for AutoResolveEnabled. When nil, auto-resolve
	// is disabled. Pass the aiassist.SettingsStore (wrapped with SettingsStoreAdapter)
	// so that toggling the setting in /settings takes effect without a restart.
	Settings AutoResolveSettingsReader
}

// Scheduler emits SLA warning events and optionally auto-closes silent tickets.
type Scheduler struct {
	bus    *Bus
	repo   TicketRepo
	engine *Engine
	cfg    SchedulerConfig
}

// NewScheduler constructs a Scheduler. engine is used to run "schedule" event rules.
func NewScheduler(bus *Bus, repo TicketRepo, engine *Engine, cfg SchedulerConfig) *Scheduler {
	if cfg.SilentWindowSeconds <= 0 {
		cfg.SilentWindowSeconds = 86400
	}
	return &Scheduler{bus: bus, repo: repo, engine: engine, cfg: cfg}
}

// Tick is the pure, clock-injectable heart of the scheduler. Call it from tests
// directly; Run calls it every 60 seconds with time.Now().
func (s *Scheduler) Tick(now time.Time) {
	// 1. Emit SLA warning events for overdue tickets.
	//
	// Source is "" (human/legitimate-trigger) so the engine's recursion guard does NOT
	// skip this event and admin rules with event="ticket.sla_warning" fire normally.
	//
	// Loop-safety: an sla_warning rule whose action mutates the ticket causes the action
	// effector to publish ticket.updated with Source:"automation" → the recursion guard
	// in Engine.Handle skips it, breaking the chain. SLA warning itself is not a ticket
	// mutation, so there is no path from sla_warning → sla_warning.
	overdueIDs, err := s.repo.OverdueTicketIDs(now)
	if err != nil {
		logger.Warn("scheduler: OverdueTicketIDs failed", zap.Error(err))
	}
	for _, id := range overdueIDs {
		ev := Event{
			Type:     EventSLAWarning,
			TicketID: id,
			Source:   "", // legitimate trigger — engine must process it
		}
		s.bus.Publish(ev)
		// Also run internal "schedule" event rules for each overdue ticket.
		// Source:"" so the engine processes it too.
		s.engine.Handle(Event{
			Type:     "schedule",
			TicketID: id,
			Source:   "",
		})
	}

	// 2. Auto-resolve silent tickets when enabled.
	//
	// Read the live setting each tick so toggling it in /settings takes effect
	// without a server restart. nil settings reader → treat as disabled.
	autoResolve := s.cfg.Settings != nil && s.cfg.Settings.AutoResolveEnabled()
	if !autoResolve {
		return
	}
	silentIDs, err := s.repo.SilentCustomerTicketIDs(now, s.cfg.SilentWindowSeconds)
	if err != nil {
		logger.Warn("scheduler: SilentCustomerTicketIDs failed", zap.Error(err))
		return
	}
	for _, id := range silentIDs {
		if err := s.engine.CloseTicket(id); err != nil {
			logger.Warn("scheduler: auto-close failed",
				zap.Uint("ticket_id", id), zap.Error(err))
		}
	}
}

// Run starts the 60-second ticker. Call as a goroutine; stops when ctx is done.
func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			s.Tick(t)
		}
	}
}
