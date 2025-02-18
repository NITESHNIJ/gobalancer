package balancer

import (
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/ninijhawan/gobalancer/internal/backend"
)

// WeightedRoundRobin expands each backend into N virtual slots proportional
// to its weight, then cycles through them atomically.
// Rebuilding the slot slice is O(sum_of_weights) and happens only when
// the caller explicitly calls Rebuild (e.g. after a weight change).
type WeightedRoundRobin struct {
	mu      sync.RWMutex
	slots   []*backend.Backend
	counter int64
}

func NewWeightedRoundRobin() *WeightedRoundRobin {
	return &WeightedRoundRobin{}
}

// Rebuild regenerates the virtual slot table from the current backend weights.
// Must be called after any weight change. Safe for concurrent use.
func (w *WeightedRoundRobin) Rebuild(backends []*backend.Backend) {
	var slots []*backend.Backend
	for _, b := range backends {
		weight := b.Weight
		if weight <= 0 {
			weight = 1
		}
		for i := 0; i < weight; i++ {
			slots = append(slots, b)
		}
	}
	w.mu.Lock()
	w.slots = slots
	w.mu.Unlock()
}

func (w *WeightedRoundRobin) Next(_ *http.Request, backends []*backend.Backend) (*backend.Backend, error) {
	w.mu.RLock()
	slots := w.slots
	w.mu.RUnlock()

	// If slots haven't been built yet, build on first call.
	if len(slots) == 0 {
		w.Rebuild(backends)
		w.mu.RLock()
		slots = w.slots
		w.mu.RUnlock()
	}
	if len(slots) == 0 {
		return nil, ErrNoBackends
	}

	idx := int(atomic.AddInt64(&w.counter, 1)-1) % len(slots)
	b := slots[idx]

	// Verify the selected backend is still healthy (slots may be stale).
	if !b.IsHealthy() {
		// Fall back to first healthy backend from passed-in list.
		for _, fb := range backends {
			if fb.IsHealthy() {
				return fb, nil
			}
		}
		return nil, ErrNoBackends
	}
	return b, nil
}
