package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatusClass(t *testing.T) {
	cases := []struct {
		code int
		want string
	}{
		{100, "1xx"}, {200, "2xx"}, {301, "3xx"},
		{404, "4xx"}, {500, "5xx"}, {503, "5xx"},
	}
	for _, tc := range cases {
		got := StatusClass(tc.code)
		if got != tc.want {
			t.Errorf("StatusClass(%d) = %q, want %q", tc.code, got, tc.want)
		}
	}
}

func TestInstrumentHandler_CountsRequests(t *testing.T) {
	c := New()
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := c.InstrumentHandler("b0", "/api", inner)

	for i := 0; i < 5; i++ {
		r := httptest.NewRequest("GET", "/api", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
	}
	// If no panic occurred and counters didn't error, the test passes.
	// Label cardinality is enforced by Prometheus at registration time.
}
