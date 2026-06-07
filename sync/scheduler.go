package sync

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"
)

// Scheduler manages background sync operations.
type Scheduler struct {
	syncer   *Syncer
	interval time.Duration
	product  string
	logger   *slog.Logger

	running  atomic.Bool
	stopCh   chan struct{}
	doneCh   chan struct{}
	lastSync time.Time
	lastErr  error
}

// SchedulerConfig configures the sync scheduler.
type SchedulerConfig struct {
	// Interval between sync runs. Default: 15 minutes.
	Interval time.Duration

	// Product ID to sync.
	Product string

	// Logger for sync events. Default: slog.Default().
	Logger *slog.Logger

	// Entities to sync. Default: all entities.
	Entities []string
}

// DefaultSchedulerConfig returns default scheduler configuration.
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		Interval: 15 * time.Minute,
		Entities: []string{"features", "ideas", "releases", "initiatives", "goals", "epics"},
	}
}

// NewScheduler creates a new sync scheduler.
func NewScheduler(syncer *Syncer, cfg SchedulerConfig) *Scheduler {
	if cfg.Interval == 0 {
		cfg.Interval = 15 * time.Minute
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return &Scheduler{
		syncer:   syncer,
		interval: cfg.Interval,
		product:  cfg.Product,
		logger:   cfg.Logger,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

// Start begins background sync. Non-blocking.
func (s *Scheduler) Start(ctx context.Context) error {
	if !s.running.CompareAndSwap(false, true) {
		return fmt.Errorf("scheduler already running")
	}

	s.logger.Info("starting background sync scheduler",
		"interval", s.interval,
		"product", s.product)

	go s.run(ctx)
	return nil
}

// Stop stops the background sync gracefully.
func (s *Scheduler) Stop() {
	if !s.running.Load() {
		return
	}

	s.logger.Info("stopping background sync scheduler")
	close(s.stopCh)
	<-s.doneCh
	s.running.Store(false)
}

// IsRunning returns true if the scheduler is running.
func (s *Scheduler) IsRunning() bool {
	return s.running.Load()
}

// LastSync returns the time of the last successful sync.
func (s *Scheduler) LastSync() time.Time {
	return s.lastSync
}

// LastError returns the last sync error, if any.
func (s *Scheduler) LastError() error {
	return s.lastErr
}

// SyncNow triggers an immediate sync, regardless of schedule.
func (s *Scheduler) SyncNow(ctx context.Context) ([]SyncResult, error) {
	return s.doSync(ctx)
}

func (s *Scheduler) run(ctx context.Context) {
	defer close(s.doneCh)

	// Do initial sync
	if _, err := s.doSync(ctx); err != nil {
		s.logger.Error("initial sync failed", "error", err)
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("scheduler stopped: context cancelled")
			return
		case <-s.stopCh:
			s.logger.Info("scheduler stopped: stop requested")
			return
		case <-ticker.C:
			if _, err := s.doSync(ctx); err != nil {
				s.logger.Error("scheduled sync failed", "error", err)
			}
		}
	}
}

func (s *Scheduler) doSync(ctx context.Context) ([]SyncResult, error) {
	s.logger.Info("starting sync", "product", s.product)
	start := time.Now()

	results, err := s.syncer.SyncAll(ctx, SyncOptions{
		Product:     s.product,
		Incremental: true,
	})

	duration := time.Since(start)
	s.lastErr = err

	if err != nil {
		s.logger.Error("sync failed",
			"duration", duration,
			"error", err)
		return results, err
	}

	s.lastSync = time.Now()

	// Log results
	var totalRecords int
	for _, r := range results {
		if r.Error != nil {
			s.logger.Warn("entity sync failed",
				"entity", r.Entity,
				"error", r.Error)
		} else {
			s.logger.Debug("entity synced",
				"entity", r.Entity,
				"records", r.RecordCount,
				"duration", r.Duration)
			totalRecords += r.RecordCount
		}
	}

	s.logger.Info("sync completed",
		"duration", duration,
		"total_records", totalRecords)

	return results, nil
}

// Status returns the current scheduler status.
func (s *Scheduler) Status() SchedulerStatus {
	return SchedulerStatus{
		Running:  s.running.Load(),
		Interval: s.interval,
		Product:  s.product,
		LastSync: s.lastSync,
		LastErr:  s.lastErr,
	}
}

// SchedulerStatus contains scheduler status information.
type SchedulerStatus struct {
	Running  bool
	Interval time.Duration
	Product  string
	LastSync time.Time
	LastErr  error
}
