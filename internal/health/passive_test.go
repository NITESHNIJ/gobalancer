package health

import (
	"testing"
	"time"

	"github.com/ninijhawan/gobalancer/internal/backend"
)

func TestPassiveChecker_EjectOnHighErrorRate(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PassiveErrorRate = 0.5
	cfg.PassiveWindow = 10 * time.Second

	b, _ := backend.New("b0", "http://localhost:8080", 1)
	pc := NewPassiveChecker(cfg)

	// 6 errors out of 10 = 60% > 50% threshold
	for i := 0; i < 4; i++ {
		pc.Observe(b, false)
	}
	if b.GetHealth() != backend.HealthHealthy {
		t.Fatal("should still be healthy after 4/10 errors")
	}
	for i := 0; i < 6; i++ {
		pc.Observe(b, true)
	}

	if b.GetHealth() != backend.HealthUnhealthy {
		t.Errorf("expected Unhealthy after 60%% error rate, got %v", b.GetHealth())
	}
}

func TestPassiveChecker_NoEjectBelowThreshold(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PassiveErrorRate = 0.5
	cfg.PassiveWindow = 10 * time.Second

	b, _ := backend.New("b0", "http://localhost:8080", 1)
	pc := NewPassiveChecker(cfg)

	for i := 0; i < 10; i++ {
		pc.Observe(b, i < 4) // 40% errors
	}

	if b.GetHealth() != backend.HealthHealthy {
		t.Error("should remain healthy below threshold")
	}
}

func TestPassiveChecker_ReadmitAfterBackoff(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PassiveErrorRate = 0.3
	cfg.PassiveWindow = 10 * time.Second

	b, _ := backend.New("b0", "http://localhost:8080", 1)
	pc := NewPassiveChecker(cfg)

	// Force ejection.
	for i := 0; i < 10; i++ {
		pc.Observe(b, true)
	}
	if b.GetHealth() != backend.HealthUnhealthy {
		t.Fatal("expected ejection")
	}

	// Manually set ejectedAt far in the past to simulate backoff expiry.
	pc.mu.Lock()
	pc.ejected[b.ID] = time.Now().Add(-10 * time.Minute)
	pc.mu.Unlock()

	pc.TryReadmit(b)
	if b.GetHealth() != backend.HealthProbation {
		t.Errorf("expected Probation after backoff, got %v", b.GetHealth())
	}
}
