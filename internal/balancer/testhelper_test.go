package balancer

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// assertDistribution checks that no bucket deviates from expected by more than tolPct percent.
func assertDistribution(t *testing.T, counts map[string]int, total int, n int, tolPct float64) {
	t.Helper()
	expected := float64(total) / float64(n)
	for id, c := range counts {
		deviation := (float64(c) - expected) / expected * 100
		if deviation < 0 {
			deviation = -deviation
		}
		if deviation > tolPct {
			t.Errorf("backend %s: got %d (%.1f%% deviation from expected %.0f, tolerance %.1f%%)",
				id, c, deviation, expected, tolPct)
		}
	}
}

// newRequest creates a minimal *http.Request for testing selectors.
func newRequest(remoteAddr string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = remoteAddr
	return r
}
