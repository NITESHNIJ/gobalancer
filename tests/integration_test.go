package tests

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/ninijhawan/gobalancer/internal/backend"
	"github.com/ninijhawan/gobalancer/internal/balancer"
	"github.com/ninijhawan/gobalancer/internal/proxy"
)

// echoServer returns an httptest.Server that responds with its own ID.
func echoServer(id string, counter *int64) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(counter, 1)
		fmt.Fprintf(w, "backend:%s", id)
	}))
}

func TestIntegration_RoundRobin_ThreeBackends(t *testing.T) {
	const numBackends = 3
	const requests = 300

	counters := make([]int64, numBackends)
	servers := make([]*httptest.Server, numBackends)
	backends := make([]*backend.Backend, numBackends)

	for i := 0; i < numBackends; i++ {
		i := i
		servers[i] = echoServer(fmt.Sprintf("b%d", i), &counters[i])
		defer servers[i].Close()

		b, err := backend.New(fmt.Sprintf("b%d", i), servers[i].URL, 1)
		if err != nil {
			t.Fatal(err)
		}
		backends[i] = b
	}

	pool := backend.NewPool(backends)
	sel := balancer.NewRoundRobin()
	h := proxy.NewHTTPProxy(pool, sel)

	ts := httptest.NewServer(h)
	defer ts.Close()

	client := ts.Client()
	for i := 0; i < requests; i++ {
		resp, err := client.Get(ts.URL)
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
		io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("request %d: status %d", i, resp.StatusCode)
		}
	}

	expected := requests / numBackends
	for i, c := range counters {
		got := atomic.LoadInt64(&c)
		if got != int64(expected) {
			t.Errorf("backend b%d: got %d requests, want %d", i, got, expected)
		}
	}
}

func TestIntegration_UnhealthyBackendSkipped(t *testing.T) {
	var counter0, counter1 int64
	s0 := echoServer("b0", &counter0)
	s1 := echoServer("b1", &counter1)
	defer s0.Close()
	defer s1.Close()

	b0, _ := backend.New("b0", s0.URL, 1)
	b1, _ := backend.New("b1", s1.URL, 1)
	b1.SetHealth(backend.HealthUnhealthy)

	pool := backend.NewPool([]*backend.Backend{b0, b1})
	sel := balancer.NewRoundRobin()
	h := proxy.NewHTTPProxy(pool, sel)

	ts := httptest.NewServer(h)
	defer ts.Close()

	client := ts.Client()
	for i := 0; i < 10; i++ {
		resp, err := client.Get(ts.URL)
		if err != nil {
			t.Fatal(err)
		}
		io.ReadAll(resp.Body)
		resp.Body.Close()
	}

	if atomic.LoadInt64(&counter1) > 0 {
		t.Errorf("unhealthy backend b1 received %d requests, want 0", counter1)
	}
	if atomic.LoadInt64(&counter0) != 10 {
		t.Errorf("healthy backend b0 received %d requests, want 10", counter0)
	}
}
