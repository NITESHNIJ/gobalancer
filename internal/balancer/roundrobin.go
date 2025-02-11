package balancer

import (
	"net/http"
	"sync/atomic"

	"github.com/ninijhawan/gobalancer/internal/backend"
)

// RoundRobin cycles through backends sequentially using a lock-free atomic counter.
type RoundRobin struct {
	counter int64
}

func NewRoundRobin() *RoundRobin {
	return &RoundRobin{}
}

func (rr *RoundRobin) Next(_ *http.Request, backends []*backend.Backend) (*backend.Backend, error) {
	n := len(backends)
	if n == 0 {
		return nil, ErrNoBackends
	}
	// Atomically increment, wrap with mod. Works correctly even when backends
	// slice length changes between calls because we take a snapshot above.
	idx := int(atomic.AddInt64(&rr.counter, 1)-1) % n
	return backends[idx], nil
}
