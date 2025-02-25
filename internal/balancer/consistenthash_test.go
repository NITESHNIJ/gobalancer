package balancer

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/ninijhawan/gobalancer/internal/backend"
)

func TestConsistentHash_Determinism(t *testing.T) {
	bs := makeBackends(3)
	ch := NewConsistentHash(150)
	ch.Build(bs)

	// Same IP must always land on the same backend.
	r := newRequest("192.168.1.1:1234")
	first, err := ch.Next(r, bs)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		b, _ := ch.Next(r, bs)
		if b.ID != first.ID {
			t.Fatalf("non-deterministic: got %s, want %s on iteration %d", b.ID, first.ID, i)
		}
	}
}

func TestConsistentHash_MinimalRemapping(t *testing.T) {
	const sampleIPs = 10000
	const N = 5

	bs := makeBackends(N)
	ch := NewConsistentHash(150)
	ch.Build(bs)

	// Record which backend each IP hashes to with N backends.
	before := make(map[string]string, sampleIPs)
	for i := 0; i < sampleIPs; i++ {
		ip := fmt.Sprintf("10.0.%d.%d", i/256, i%256)
		r := newRequest(ip + ":0")
		b, _ := ch.Next(r, bs)
		before[ip] = b.ID
	}

	// Add one backend.
	extra, _ := backend.New("extra", "http://localhost:9090", 1)
	extended := append(bs, extra)
	ch.Build(extended)

	remapped := 0
	for i := 0; i < sampleIPs; i++ {
		ip := fmt.Sprintf("10.0.%d.%d", i/256, i%256)
		r := newRequest(ip + ":0")
		b, _ := ch.Next(r, extended)
		if b.ID != before[ip] {
			remapped++
		}
	}

	pct := float64(remapped) / float64(sampleIPs) * 100
	// Theoretical minimum: 1/(N+1) ≈ 16.7%. Allow up to 2× for small N.
	threshold := 2.0 / float64(N+1) * 100
	if pct > threshold {
		t.Errorf("too many keys remapped: %.1f%% (threshold %.1f%%)", pct, threshold)
	}
	t.Logf("%.1f%% keys remapped after adding 1 of %d backends (theory: ~%.1f%%)",
		pct, N, 100.0/float64(N+1))
}

func TestConsistentHash_ZeroBackends(t *testing.T) {
	ch := NewConsistentHash(150)
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "1.2.3.4:0"
	_, err := ch.Next(r, nil)
	if err == nil {
		t.Fatal("expected error for zero backends")
	}
}
