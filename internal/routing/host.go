package routing

import (
	"net/http"
	"sync"
)

// HostRouter selects a pool based on the HTTP Host header.
// Exact-match lookup is O(1). Used for multi-tenant virtual hosting.
type HostRouter struct {
	mu    sync.RWMutex
	rules map[string]string // host → pool name
}

func NewHostRouter() *HostRouter {
	return &HostRouter{rules: make(map[string]string)}
}

// Add registers a Host → pool mapping. Host should be the bare hostname
// (no scheme), optionally including port (e.g. "api.example.com:8080").
func (h *HostRouter) Add(host, pool string) {
	h.mu.Lock()
	h.rules[host] = pool
	h.mu.Unlock()
}

// Match returns the pool name for the request's Host header.
func (h *HostRouter) Match(r *http.Request) (string, bool) {
	host := r.Host
	// Strip port if present.
	if i := len(host) - 1; i >= 0 && host[i] != ']' {
		for j := len(host) - 1; j >= 0; j-- {
			if host[j] == ':' {
				host = host[:j]
				break
			}
		}
	}
	h.mu.RLock()
	pool, ok := h.rules[host]
	h.mu.RUnlock()
	return pool, ok
}
