package circuitbreaker

import (
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// ErrCircuitOpen is returned when the circuit is open and requests are shed.
var ErrCircuitOpen = errors.New("circuit breaker open")

// State represents the circuit breaker FSM state.
type State int32

const (
	StateClosed   State = 0 // normal operation
	StateOpen     State = 1 // fast-fail; no upstream calls
	StateHalfOpen State = 2 // single probe allowed
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Config controls circuit breaker thresholds.
type Config struct {
	FailureThreshold int           // consecutive failures before opening
	OpenDuration     time.Duration // how long to stay open before half-open
	HalfOpenProbes   int           // max concurrent probes in half-open
}

func DefaultConfig() Config {
	return Config{
		FailureThreshold: 5,
		OpenDuration:     30 * time.Second,
		HalfOpenProbes:   1,
	}
}

// Breaker is a per-backend circuit breaker FSM.
// All methods are safe for concurrent use.
type Breaker struct {
	cfg      Config
	state    int32 // atomic State
	failures int64 // atomic
	openedAt int64 // atomic unix nano

	// Single-permit semaphore for half-open probes.
	probe chan struct{}
	mu    sync.Mutex // guards state transitions
}

func New(cfg Config) *Breaker {
	return &Breaker{
		cfg:   cfg,
		state: int32(StateClosed),
		probe: make(chan struct{}, cfg.HalfOpenProbes),
	}
}

// Allow returns nil if the request may proceed, or ErrCircuitOpen if it should be shed.
func (b *Breaker) Allow() error {
	switch State(atomic.LoadInt32(&b.state)) {
	case StateClosed:
		return nil

	case StateOpen:
		openedAt := time.Unix(0, atomic.LoadInt64(&b.openedAt))
		if time.Since(openedAt) >= b.cfg.OpenDuration {
			b.mu.Lock()
			// Re-check under lock to prevent thundering herd.
			if State(atomic.LoadInt32(&b.state)) == StateOpen {
				slog.Info("circuit → half-open")
				atomic.StoreInt32(&b.state, int32(StateHalfOpen))
				// Drain the probe channel so it has exactly HalfOpenProbes permits.
				for len(b.probe) > 0 {
					<-b.probe
				}
				for i := 0; i < b.cfg.HalfOpenProbes; i++ {
					b.probe <- struct{}{}
				}
			}
			b.mu.Unlock()
		}
		if State(atomic.LoadInt32(&b.state)) == StateOpen {
			return ErrCircuitOpen
		}
		fallthrough

	case StateHalfOpen:
		select {
		case <-b.probe:
			return nil
		default:
			return ErrCircuitOpen
		}
	}
	return nil
}

// RecordSuccess reports a successful upstream response.
func (b *Breaker) RecordSuccess() {
	atomic.StoreInt64(&b.failures, 0)
	if State(atomic.LoadInt32(&b.state)) == StateHalfOpen {
		b.mu.Lock()
		slog.Info("circuit → closed")
		atomic.StoreInt32(&b.state, int32(StateClosed))
		b.mu.Unlock()
	}
}

// RecordFailure reports a failed upstream response.
func (b *Breaker) RecordFailure() {
	n := atomic.AddInt64(&b.failures, 1)
	if int(n) >= b.cfg.FailureThreshold || State(atomic.LoadInt32(&b.state)) == StateHalfOpen {
		b.mu.Lock()
		if State(atomic.LoadInt32(&b.state)) != StateOpen {
			slog.Warn("circuit → open", "failures", n)
			atomic.StoreInt64(&b.openedAt, time.Now().UnixNano())
			atomic.StoreInt32(&b.state, int32(StateOpen))
			atomic.StoreInt64(&b.failures, 0)
		}
		b.mu.Unlock()
	}
}

// State returns the current circuit state.
func (b *Breaker) State() State {
	return State(atomic.LoadInt32(&b.state))
}
