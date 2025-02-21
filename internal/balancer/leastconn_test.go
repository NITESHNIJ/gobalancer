package balancer

import (
	"net/http"
	"sync"
	"testing"

	"github.com/ninijhawan/gobalancer/internal/backend"
)

func TestLeastConnections_RoutesToLeast(t *testing.T) {
	bs := makeBackends(3)
	// Simulate existing connections: b0=5, b1=1, b2=3
	for i := 0; i < 5; i++ {
		bs[0].IncConns()
	}
	bs[1].IncConns()
	for i := 0; i < 3; i++ {
		bs[2].IncConns()
	}

	lc := NewLeastConnections()
	b, err := lc.Next(&http.Request{}, bs)
	if err != nil {
		t.Fatal(err)
	}
	if b.ID != "b1" {
		t.Errorf("expected b1 (1 conn), got %s (%d conns)", b.ID, b.ActiveConns())
	}
}

func TestLeastConnections_NoStarvation(t *testing.T) {
	const N = 5
	const goroutines = 50
	const each = 200

	bs := makeBackends(N)
	lc := NewLeastConnections()
	counts := make([]int64, N)

	var wg sync.WaitGroup
	var mu sync.Mutex

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < each; i++ {
				b, err := lc.Next(&http.Request{}, bs)
				if err != nil {
					t.Error(err)
					return
				}
				b.IncConns()
				mu.Lock()
				for idx, backend := range bs {
					if backend.ID == b.ID {
						counts[idx]++
						break
					}
				}
				mu.Unlock()
				b.DecConns()
			}
		}()
	}
	wg.Wait()

	total := goroutines * each
	for i, c := range counts {
		if c == 0 {
			t.Errorf("backend b%d received 0 requests — starvation detected", i)
		}
		_ = total
	}
}

func TestLeastConnections_ZeroBackends(t *testing.T) {
	lc := NewLeastConnections()
	_, err := lc.Next(&http.Request{}, nil)
	if err == nil {
		t.Fatal("expected error for zero backends")
	}
}
