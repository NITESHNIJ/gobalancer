package balancer

import (
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/ninijhawan/gobalancer/internal/backend"
)

func makeBackends(n int) []*backend.Backend {
	bs := make([]*backend.Backend, n)
	for i := range bs {
		b, _ := backend.New(
			fmt.Sprintf("b%d", i),
			fmt.Sprintf("http://localhost:%d", 8080+i),
			1,
		)
		bs[i] = b
	}
	return bs
}

func TestRoundRobin_ZeroBackends(t *testing.T) {
	rr := NewRoundRobin()
	_, err := rr.Next(&http.Request{}, nil)
	if err == nil {
		t.Fatal("expected error for zero backends")
	}
}

func TestRoundRobin_SingleBackend(t *testing.T) {
	rr := NewRoundRobin()
	bs := makeBackends(1)
	for i := 0; i < 10; i++ {
		b, err := rr.Next(&http.Request{}, bs)
		if err != nil {
			t.Fatal(err)
		}
		if b != bs[0] {
			t.Fatalf("expected bs[0], got %v", b.ID)
		}
	}
}

func TestRoundRobin_EvenDistribution(t *testing.T) {
	const N = 3
	const requests = 300
	rr := NewRoundRobin()
	bs := makeBackends(N)
	counts := make(map[string]int, N)

	for i := 0; i < requests; i++ {
		b, err := rr.Next(&http.Request{}, bs)
		if err != nil {
			t.Fatal(err)
		}
		counts[b.ID]++
	}

	expected := requests / N
	for id, c := range counts {
		if c != expected {
			t.Errorf("backend %s got %d requests, want %d", id, c, expected)
		}
	}
}

func TestRoundRobin_Concurrent(t *testing.T) {
	rr := NewRoundRobin()
	bs := makeBackends(5)
	const goroutines = 100
	const each = 100

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < each; j++ {
				_, err := rr.Next(&http.Request{}, bs)
				if err != nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()
}
