package balancer

import (
	"hash/fnv"
	"net"
	"net/http"

	"github.com/ninijhawan/gobalancer/internal/backend"
)

// IPHash maps each client IP to a backend deterministically using FNV-1a.
// The same IP always hits the same backend (sticky, stateless).
// Caveat: adding/removing a backend remaps ~50% of clients — use
// ConsistentHash when minimal disruption on scale-out matters.
type IPHash struct{}

func NewIPHash() *IPHash {
	return &IPHash{}
}

func (h *IPHash) Next(r *http.Request, backends []*backend.Backend) (*backend.Backend, error) {
	if len(backends) == 0 {
		return nil, ErrNoBackends
	}

	ip := clientIP(r)
	hash := fnv32a(ip)
	idx := int(hash) % len(backends)
	return backends[idx], nil
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the leftmost (original) IP.
		ip, _, _ := net.SplitHostPort(xff)
		if ip != "" {
			return ip
		}
		return xff
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func fnv32a(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}
