package balancer

import (
	"fmt"
	"testing"
)

func TestIPHash_SameIPSameBackend(t *testing.T) {
	bs := makeBackends(5)
	h := NewIPHash()

	for _, ip := range []string{"192.168.1.1", "10.0.0.1", "172.16.0.50"} {
		r := newRequest(ip + ":1234")
		first, err := h.Next(r, bs)
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 50; i++ {
			b, _ := h.Next(r, bs)
			if b.ID != first.ID {
				t.Fatalf("IP %s: non-deterministic — got %s, want %s", ip, b.ID, first.ID)
			}
		}
	}
}

func TestIPHash_Distribution(t *testing.T) {
	bs := makeBackends(5)
	h := NewIPHash()
	counts := make(map[string]int)

	for i := 0; i < 5000; i++ {
		ip := fmt.Sprintf("10.%d.%d.%d", i/65536, (i/256)%256, i%256)
		r := newRequest(ip + ":0")
		b, _ := h.Next(r, bs)
		counts[b.ID]++
	}

	// Each backend should get roughly 20% of traffic (±10% tolerance).
	for _, b := range bs {
		got := float64(counts[b.ID]) / 5000.0
		if got < 0.10 || got > 0.30 {
			t.Errorf("backend %s: %.1f%% of traffic (want 10-30%%)", b.ID, got*100)
		}
	}
}
