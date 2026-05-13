// Package services provides the background service manager and all long-running
// goroutine-based services that replace the legacy PHP daemon scripts.
package services

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Service is implemented by every background worker.
type Service interface {
	// Name returns a human-readable identifier used in logs and health checks.
	Name() string
	// Run starts the service loop. It must respect context cancellation and
	// return promptly when ctx is cancelled.
	Run(ctx context.Context) error
}

// Manager starts, supervises, and gracefully shuts down all services.
type Manager struct {
	services []Service
	wg       sync.WaitGroup
}

// New creates a Manager with the provided services.
func New(svcs ...Service) *Manager {
	return &Manager{
		services: svcs,
	}
}

// Run starts all services and blocks until ctx is cancelled or a service
// returns a fatal error. Panicking services are caught and restarted.
func (m *Manager) Run(ctx context.Context) {
	for _, svc := range m.services {
		svc := svc
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			m.runWithRestart(ctx, svc)
		}()
	}
	m.wg.Wait()
}

// runWithRestart restarts the service after any non-context error, with an
// exponential back-off capped at 60 seconds.
func (m *Manager) runWithRestart(ctx context.Context, svc Service) {
	backoff := 5 * time.Second
	for {
		slog.Info("service starting", "service", svc.Name())
		err := svc.Run(ctx)

		if ctx.Err() != nil {
			// Graceful shutdown — do not restart.
			slog.Info("service stopped", "service", svc.Name())
			return
		}

		slog.Error("service exited with error — restarting",
			"service", svc.Name(), "error", err, "backoff", backoff)

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}

		if backoff < 60*time.Second {
			backoff *= 2
		}
	}
}
