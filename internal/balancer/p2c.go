package balancer

import (
	"math/rand"
	"net/http"

	"github.com/ninijhawan/gobalancer/internal/backend"
)

// P2C implements the Power of Two Choices algorithm.
// Pick two backends uniformly at random, route to the less-loaded one.
// Reduces worst-case load from O(log n / log log n) to O(log log n)
// compared to pure random — Mitzenmacher (2001).
type P2C struct {
	rng *rand.Rand
}

func NewP2C() *P2C {
	return &P2C{
		rng: rand.New(rand.NewSource(42)),
	}
}

func (p *P2C) Next(_ *http.Request, backends []*backend.Backend) (*backend.Backend, error) {
	n := len(backends)
	switch n {
	case 0:
		return nil, ErrNoBackends
	case 1:
		return backends[0], nil
	}

	// Pick two distinct random indices.
	i := p.rng.Intn(n)
	j := p.rng.Intn(n - 1)
	if j >= i {
		j++
	}

	a, b := backends[i], backends[j]
	if a.ActiveConns() <= b.ActiveConns() {
		return a, nil
	}
	return b, nil
}
