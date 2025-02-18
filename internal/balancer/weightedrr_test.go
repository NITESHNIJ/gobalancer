package balancer

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/ninijhawan/gobalancer/internal/backend"
)

func makeWeightedBackends(weights []int) []*backend.Backend {
	bs := make([]*backend.Backend, len(weights))
	for i, w := range weights {
		b, _ := backend.New(fmt.Sprintf("b%d", i), fmt.Sprintf("http://localhost:%d", 8080+i), w)
		bs[i] = b
	}
	return bs
}

func TestWeightedRR_Distribution(t *testing.T) {
	weights := []int{1, 2, 3}
	total := 0
	for _, w := range weights {
		total += w
	}
	const requests = 6000

	bs := makeWeightedBackends(weights)
	wrr := NewWeightedRoundRobin()
	wrr.Rebuild(bs)

	counts := make(map[string]int)
	for i := 0; i < requests; i++ {
		b, err := wrr.Next(&http.Request{}, bs)
		if err != nil {
			t.Fatal(err)
		}
		counts[b.ID]++
	}

	for i, b := range bs {
		expectedFraction := float64(weights[i]) / float64(total)
		got := float64(counts[b.ID]) / float64(requests)
		deviation := (got - expectedFraction) / expectedFraction * 100
		if deviation < 0 {
			deviation = -deviation
		}
		if deviation > 2.0 {
			t.Errorf("backend %s: got %.2f%%, expected %.2f%% (%.1f%% deviation, want ≤2%%)",
				b.ID, got*100, expectedFraction*100, deviation)
		}
	}
}

func TestWeightedRR_ZeroWeightFallsBackToOne(t *testing.T) {
	bs := makeWeightedBackends([]int{0, 0})
	wrr := NewWeightedRoundRobin()
	wrr.Rebuild(bs)

	for i := 0; i < 10; i++ {
		_, err := wrr.Next(&http.Request{}, bs)
		if err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
	}
}

func TestWeightedRR_RebuildAfterWeightChange(t *testing.T) {
	bs := makeWeightedBackends([]int{1, 1})
	wrr := NewWeightedRoundRobin()
	wrr.Rebuild(bs)

	// Change weight of b0 to 3.
	bs[0].Weight = 3
	wrr.Rebuild(bs)

	counts := make(map[string]int)
	for i := 0; i < 400; i++ {
		b, _ := wrr.Next(&http.Request{}, bs)
		counts[b.ID]++
	}

	// b0 should get ~75%, b1 ~25%
	frac0 := float64(counts["b0"]) / 400.0
	if frac0 < 0.70 || frac0 > 0.80 {
		t.Errorf("after rebuild b0 got %.2f%%, want ~75%%", frac0*100)
	}
}
