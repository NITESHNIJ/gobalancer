package routing

import (
	"math/rand"
	"net/http"
)

// CanaryRouter splits traffic between a stable pool and a canary pool.
// CanaryWeight fraction (0.0–1.0) of requests go to the canary pool.
type CanaryRouter struct {
	StablePool  string
	CanaryPool  string
	CanaryWeight float64 // 0.05 = 5% to canary
	rng         *rand.Rand
}

func NewCanaryRouter(stable, canary string, weight float64) *CanaryRouter {
	return &CanaryRouter{
		StablePool:   stable,
		CanaryPool:   canary,
		CanaryWeight: weight,
		rng:          rand.New(rand.NewSource(42)),
	}
}

// Route returns the chosen pool name for this request.
func (c *CanaryRouter) Route(_ *http.Request) string {
	if c.rng.Float64() < c.CanaryWeight {
		return c.CanaryPool
	}
	return c.StablePool
}
