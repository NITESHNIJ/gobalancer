package admin

import (
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ninijhawan/gobalancer/internal/backend"
)

// Drainer manages graceful removal of a backend.
// It marks the backend as disabled, waits for in-flight requests to finish,
// then removes it from the pool.
type Drainer struct {
	timeout time.Duration
}

func NewDrainer(timeout time.Duration) *Drainer {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &Drainer{timeout: timeout}
}

// Drain blocks until the backend's ActiveConns reaches zero or the timeout expires.
// The backend is marked Disabled immediately so the Selector stops routing to it.
func (d *Drainer) Drain(b *backend.Backend, wg *sync.WaitGroup) error {
	b.SetHealth(backend.HealthDisabled)
	slog.Info("draining backend", "id", b.ID)

	done := make(chan struct{})
	go func() {
		defer close(done)
		// Poll every 50ms — ActiveConns is atomic, no lock needed.
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for {
			<-ticker.C
			if atomic.LoadInt64((*int64)(nil)) == 0 || b.ActiveConns() == 0 {
				return
			}
		}
	}()

	select {
	case <-done:
		slog.Info("backend drained", "id", b.ID)
	case <-time.After(d.timeout):
		slog.Warn("drain timeout exceeded", "id", b.ID, "remaining", b.ActiveConns())
	}
	return nil
}
