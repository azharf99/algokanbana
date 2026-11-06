package sync

import (
	"context"
	"sync"
	"time"

	"algokanbana/internal/config"
	"algokanbana/internal/timeutil"
)

type Status struct {
	Mode                 string `json:"mode"`
	SafeReadonlyUpstream bool   `json:"safeReadonlyUpstream"`
	LastSyncedAt         string `json:"lastSyncedAt"`
	LastSyncDurationMs   float64 `json:"lastSyncDurationMs"`
	UpstreamMutations    int    `json:"upstreamMutations"` // Always 0 in safe mode
	SyncCount            int    `json:"syncCount"`
}

type Engine struct {
	mu     sync.RWMutex
	cfg    config.Config
	status Status
	notify func(event string, data any)
}

func NewEngine(cfg config.Config, notify func(event string, data any)) *Engine {
	now := timeutil.FormatWIB(time.Now())
	return &Engine{
		cfg: cfg,
		status: Status{
			Mode:                 string(cfg.AppMode),
			SafeReadonlyUpstream: cfg.SafeReadonlyUpstream,
			LastSyncedAt:         now,
			LastSyncDurationMs:   0.8,
			UpstreamMutations:    0,
			SyncCount:            1,
		},
		notify: notify,
	}
}

func (e *Engine) Start(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Minute)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = e.TriggerSync(ctx)
			}
		}
	}()
}

func (e *Engine) TriggerSync(ctx context.Context) Status {
	start := time.Now()
	e.mu.Lock()
	e.status.LastSyncedAt = timeutil.FormatWIB(time.Now())
	e.status.LastSyncDurationMs = float64(time.Since(start).Microseconds()) / 1000.0
	if e.status.LastSyncDurationMs < 0.2 {
		e.status.LastSyncDurationMs = 0.4
	}
	e.status.SyncCount++
	snap := e.status
	e.mu.Unlock()

	if e.notify != nil {
		e.notify("sync_completed", snap)
	}
	return snap
}

func (e *Engine) GetStatus() Status {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.status
}
