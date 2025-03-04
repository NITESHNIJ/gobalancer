package health

import (
	"log/slog"
	"sync"
	"time"

	"github.com/ninijhawan/gobalancer/internal/backend"
)

// event records a single request outcome for the sliding window.
type event struct {
	ts  time.Time
	err bool
}

// PassiveChecker tracks 5xx errors and timeouts per backend in a sliding window.
// When the error rate exceeds a threshold, the backend is auto-ejected.
type PassiveChecker struct {
	mu       sync.Mutex
	windows  map[string][]event
	cfg      Config
	backoffs map[string]time.Duration
	ejected  map[string]time.Time
}

func NewPassiveChecker(cfg Config) *PassiveChecker {
	return &PassiveChecker{
		windows:  make(map[string][]event),
		cfg:      cfg,
		backoffs: make(map[string]time.Duration),
		ejected:  make(map[string]time.Time),
	}
}

// Observe records a request result. Call on every proxied response.
func (pc *PassiveChecker) Observe(b *backend.Backend, isError bool) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-pc.cfg.PassiveWindow)

	// Trim events outside the window.
	evts := pc.windows[b.ID]
	start := 0
	for start < len(evts) && evts[start].ts.Before(cutoff) {
		start++
	}
	evts = append(evts[start:], event{ts: now, err: isError})
	pc.windows[b.ID] = evts

	// Count errors.
	total := len(evts)
	if total == 0 {
		return
	}
	errs := 0
	for _, e := range evts {
		if e.err {
			errs++
		}
	}
	rate := float64(errs) / float64(total)

	if rate >= pc.cfg.PassiveErrorRate && b.GetHealth() == backend.HealthHealthy {
		slog.Warn("passive eject: error rate exceeded",
			"backend", b.ID,
			"rate", rate,
			"threshold", pc.cfg.PassiveErrorRate,
		)
		b.SetHealth(backend.HealthUnhealthy)
		if pc.backoffs[b.ID] == 0 {
			pc.backoffs[b.ID] = 5 * time.Second
		}
		pc.ejected[b.ID] = now
	}
}

// TryReadmit re-admits a passively ejected backend after the backoff expires.
// Call periodically (e.g. from the active health checker loop).
func (pc *PassiveChecker) TryReadmit(b *backend.Backend) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if b.GetHealth() != backend.HealthUnhealthy {
		return
	}
	ejectedAt, ok := pc.ejected[b.ID]
	if !ok {
		return
	}
	backoff := pc.backoffs[b.ID]
	if time.Since(ejectedAt) >= backoff {
		slog.Info("passive readmit: backoff expired", "backend", b.ID, "backoff", backoff)
		b.SetHealth(backend.HealthProbation)
		// Exponential backoff for future ejections.
		pc.backoffs[b.ID] = min(backoff*2, 5*time.Minute)
		delete(pc.ejected, b.ID)
		pc.windows[b.ID] = nil // reset window
	}
}
