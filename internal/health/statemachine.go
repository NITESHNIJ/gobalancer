package health

import (
	"log/slog"
	"sync"
	"time"

	"github.com/ninijhawan/gobalancer/internal/backend"
)

// probeState tracks per-backend consecutive failure/success counters
// and the time the backend entered Unhealthy state (for backoff).
type probeState struct {
	mu           sync.Mutex
	failures     int
	successes    int
	unhealthySince time.Time
	backoff      time.Duration
}

func newProbeState() *probeState {
	return &probeState{backoff: 5 * time.Second}
}

// record updates counters and returns (isHealthy, isRecovering).
func (ps *probeState) record(ok bool) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ok {
		ps.failures = 0
		ps.successes++
	} else {
		ps.successes = 0
		ps.failures++
		if ps.unhealthySince.IsZero() {
			ps.unhealthySince = time.Now()
		}
	}
}

// StateMachine applies configurable thresholds to drive backend health state.
type StateMachine struct {
	cfg    Config
	states map[string]*probeState
	mu     sync.Mutex
}

func NewStateMachine(cfg Config) *StateMachine {
	return &StateMachine{
		cfg:    cfg,
		states: make(map[string]*probeState),
	}
}

func (sm *StateMachine) state(id string) *probeState {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.states[id] == nil {
		sm.states[id] = newProbeState()
	}
	return sm.states[id]
}

// Observe records a probe result and drives the backend FSM.
func (sm *StateMachine) Observe(b *backend.Backend, ok bool) {
	ps := sm.state(b.ID)
	ps.record(ok)

	ps.mu.Lock()
	failures := ps.failures
	successes := ps.successes
	ps.mu.Unlock()

	current := b.GetHealth()

	switch current {
	case backend.HealthHealthy:
		if failures >= sm.cfg.FailureThreshold {
			slog.Warn("→ Unhealthy", "backend", b.ID, "failures", failures)
			b.SetHealth(backend.HealthUnhealthy)
		}

	case backend.HealthUnhealthy:
		// Only attempt recovery after backoff expires.
		ps.mu.Lock()
		elapsed := time.Since(ps.unhealthySince)
		backoff := ps.backoff
		ps.mu.Unlock()

		if elapsed >= backoff && successes >= sm.cfg.SuccessThreshold {
			slog.Info("→ Probation", "backend", b.ID)
			b.SetHealth(backend.HealthProbation)
			// Exponential backoff for next failure window.
			ps.mu.Lock()
			ps.backoff = min(ps.backoff*2, 5*time.Minute)
			ps.unhealthySince = time.Time{}
			ps.mu.Unlock()
		}

	case backend.HealthProbation:
		if failures > 0 {
			slog.Warn("→ Unhealthy (probation failed)", "backend", b.ID)
			b.SetHealth(backend.HealthUnhealthy)
			ps.mu.Lock()
			ps.unhealthySince = time.Now()
			ps.mu.Unlock()
		} else if successes >= sm.cfg.SuccessThreshold {
			slog.Info("→ Healthy", "backend", b.ID)
			b.SetHealth(backend.HealthHealthy)
			ps.mu.Lock()
			ps.backoff = 5 * time.Second // reset on full recovery
			ps.mu.Unlock()
		}
	}
}

func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
