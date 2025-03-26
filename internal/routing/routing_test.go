package routing

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPathTrie_BasicRouting(t *testing.T) {
	trie := NewPathTrie()
	trie.Add("/api", "api-pool")
	trie.Add("/static", "cdn-pool")
	trie.Add("/", "default-pool")

	cases := []struct {
		path string
		want string
	}{
		{"/api/users", "api-pool"},
		{"/api/orders/123", "api-pool"},
		{"/static/style.css", "cdn-pool"},
		{"/static/img/logo.png", "cdn-pool"},
		{"/healthz", "default-pool"},
	}

	for _, tc := range cases {
		r := httptest.NewRequest("GET", tc.path, nil)
		pool, ok := trie.Match(r)
		if !ok {
			t.Errorf("path %s: no match, want %q", tc.path, tc.want)
			continue
		}
		if pool != tc.want {
			t.Errorf("path %s: got %q, want %q", tc.path, pool, tc.want)
		}
	}
}

func TestHostRouter_VirtualHosting(t *testing.T) {
	hr := NewHostRouter()
	hr.Add("api.example.com", "api-pool")
	hr.Add("www.example.com", "web-pool")

	cases := []struct {
		host string
		want string
		ok   bool
	}{
		{"api.example.com", "api-pool", true},
		{"www.example.com", "web-pool", true},
		{"unknown.example.com", "", false},
	}

	for _, tc := range cases {
		r := &http.Request{Host: tc.host, Header: make(http.Header)}
		pool, ok := hr.Match(r)
		if ok != tc.ok || pool != tc.want {
			t.Errorf("host %q: got (%q, %v), want (%q, %v)", tc.host, pool, ok, tc.want, tc.ok)
		}
	}
}

func TestHeaderRouter_ExactMatch(t *testing.T) {
	hr := NewHeaderRouter()
	hr.AddExact("X-Tenant-ID", "acme", "acme-pool")
	hr.AddExact("X-Tenant-ID", "beta", "beta-pool")

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Tenant-ID", "acme")
	pool, ok := hr.Match(r)
	if !ok || pool != "acme-pool" {
		t.Errorf("expected acme-pool, got %q ok=%v", pool, ok)
	}
}

func TestHeaderRouter_RegexMatch(t *testing.T) {
	hr := NewHeaderRouter()
	hr.AddRegex("X-Feature-Flag", `^canary-.*`, "canary-pool")

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Feature-Flag", "canary-2024")
	pool, ok := hr.Match(r)
	if !ok || pool != "canary-pool" {
		t.Errorf("expected canary-pool, got %q ok=%v", pool, ok)
	}
}

func TestCanaryRouter_WeightDistribution(t *testing.T) {
	cr := NewCanaryRouter("stable", "canary", 0.1)
	const N = 10000
	canaryCount := 0
	r := &http.Request{}
	for i := 0; i < N; i++ {
		if cr.Route(r) == "canary" {
			canaryCount++
		}
	}
	pct := float64(canaryCount) / N * 100
	if pct < 7 || pct > 13 {
		t.Errorf("canary got %.1f%% of traffic, want ~10%% (±3%%)", pct)
	}
}
