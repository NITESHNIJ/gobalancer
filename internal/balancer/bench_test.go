package balancer

import (
	"net/http"
	"testing"
)

func benchmarkSelector(b *testing.B, sel Selector, n int) {
	b.Helper()
	bs := makeBackends(n)
	r := &http.Request{}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = sel.Next(r, bs)
	}
}

func BenchmarkRoundRobin_1k(b *testing.B)  { benchmarkSelector(b, NewRoundRobin(), 1000) }
func BenchmarkRoundRobin_10k(b *testing.B) { benchmarkSelector(b, NewRoundRobin(), 10000) }

func BenchmarkLeastConn_1k(b *testing.B)  { benchmarkSelector(b, NewLeastConnections(), 1000) }
func BenchmarkLeastConn_10k(b *testing.B) { benchmarkSelector(b, NewLeastConnections(), 10000) }

func BenchmarkIPHash_1k(b *testing.B) {
	sel := NewIPHash()
	bs := makeBackends(1000)
	r := newRequest("192.168.1.1:1234")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = sel.Next(r, bs)
	}
}

func BenchmarkConsistentHash_1k(b *testing.B) {
	sel := NewConsistentHash(150)
	bs := makeBackends(1000)
	sel.Build(bs)
	r := newRequest("192.168.1.1:1234")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = sel.Next(r, bs)
	}
}

func BenchmarkP2C_1k(b *testing.B)  { benchmarkSelector(b, NewP2C(), 1000) }
func BenchmarkP2C_10k(b *testing.B) { benchmarkSelector(b, NewP2C(), 10000) }
