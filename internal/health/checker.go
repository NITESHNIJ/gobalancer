package health

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/ninijhawan/gobalancer/internal/backend"
)

// Config drives all health-check behaviour for a pool.
type Config struct {
	Interval         time.Duration
	Timeout          time.Duration
	Path             string
	FailureThreshold int
	SuccessThreshold int
	Mode             string // "http" | "tcp" | "grpc"
	PassiveWindow    time.Duration
	PassiveErrorRate float64
}

func DefaultConfig() Config {
	return Config{
		Interval:         10 * time.Second,
		Timeout:          2 * time.Second,
		Path:             "/health",
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Mode:             "http",
		PassiveWindow:    60 * time.Second,
		PassiveErrorRate: 0.3,
	}
}

// Checker runs periodic active health checks against every backend in a pool.
type Checker struct {
	pool    *backend.Pool
	cfg     Config
	client  *http.Client
	// per-backend probe state
	failures map[string]int
	successes map[string]int
}

func NewChecker(pool *backend.Pool, cfg Config) *Checker {
	return &Checker{
		pool: pool,
		cfg:  cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		failures:  make(map[string]int),
		successes: make(map[string]int),
	}
}

// Run starts health-check loops for all backends. Blocks until ctx is cancelled.
func (c *Checker) Run(ctx context.Context) {
	ticker := time.NewTicker(c.cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, b := range c.pool.Backends() {
				go c.probe(b)
			}
		}
	}
}

func (c *Checker) probe(b *backend.Backend) {
	var ok bool
	switch c.cfg.Mode {
	case "tcp":
		ok = probeTCP(b.URL.Host, c.cfg.Timeout)
	default:
		ok = c.probeHTTP(b)
	}

	if ok {
		c.failures[b.ID] = 0
		c.successes[b.ID]++
	} else {
		c.successes[b.ID] = 0
		c.failures[b.ID]++
	}

	c.transition(b)
}

func (c *Checker) probeHTTP(b *backend.Backend) bool {
	url := b.URL.String() + c.cfg.Path
	resp, err := c.client.Get(url)
	if err != nil {
		slog.Debug("health check failed", "backend", b.ID, "err", err)
		return false
	}
	resp.Body.Close()
	return resp.StatusCode < 500
}

func (c *Checker) transition(b *backend.Backend) {
	current := b.GetHealth()
	switch current {
	case backend.HealthHealthy:
		if c.failures[b.ID] >= c.cfg.FailureThreshold {
			slog.Warn("backend marked unhealthy", "id", b.ID, "failures", c.failures[b.ID])
			b.SetHealth(backend.HealthUnhealthy)
		}
	case backend.HealthUnhealthy:
		if c.successes[b.ID] >= c.cfg.SuccessThreshold {
			slog.Info("backend recovered to probation", "id", b.ID)
			b.SetHealth(backend.HealthProbation)
		}
	case backend.HealthProbation:
		if c.failures[b.ID] > 0 {
			b.SetHealth(backend.HealthUnhealthy)
		} else if c.successes[b.ID] >= c.cfg.SuccessThreshold {
			slog.Info("backend fully recovered", "id", b.ID)
			b.SetHealth(backend.HealthHealthy)
		}
	}
}
