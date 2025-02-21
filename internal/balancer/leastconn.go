package balancer

import (
	"math"
	"net/http"

	"github.com/ninijhawan/gobalancer/internal/backend"
)

// LeastConnections routes each request to the backend with the fewest
// active connections. O(n) scan — suitable for small backend counts (<100).
// All reads are lock-free via atomic.LoadInt64.
type LeastConnections struct{}

func NewLeastConnections() *LeastConnections {
	return &LeastConnections{}
}

func (lc *LeastConnections) Next(_ *http.Request, backends []*backend.Backend) (*backend.Backend, error) {
	if len(backends) == 0 {
		return nil, ErrNoBackends
	}

	var best *backend.Backend
	var bestConns int64 = math.MaxInt64

	for _, b := range backends {
		conns := b.ActiveConns()
		if conns < bestConns {
			bestConns = conns
			best = b
		}
	}
	return best, nil
}
