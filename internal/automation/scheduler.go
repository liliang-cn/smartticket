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

// SchedulerConfig controls auto-resolve behaviour.
type SchedulerConfig struct {
	// AutoResolveEnabled closes tickets when a customer has been silent past SilentWindowSeconds.
	AutoResolveEnabled bool
	// SilentWindowSeconds is the inactivity window in seconds. Defaults to 86400 (24 h).
	SilentWindowSeconds int64
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
	overdueIDs, err := s.repo.OverdueTicketIDs(now)
	if err != nil {
		logger.Warn("scheduler: OverdueTicketIDs failed", zap.Error(err))
	}
	for _, id := range overdueIDs {
		ev := Event{
			Type:     EventSLAWarning,
			TicketID: id,
			Source:   "automation", // marks event as system-generated
		}
		s.bus.Publish(ev)
		// Also run schedule-event rules for each overdue ticket.
		// Source:"" so the engine processes it (schedule rules respond to Source:"").
		s.engine.Handle(Event{
			Type:     "schedule",
			TicketID: id,
			Source:   "",
		})
	}

	// 2. Auto-resolve silent tickets when enabled.
	if !s.cfg.AutoResolveEnabled {
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
