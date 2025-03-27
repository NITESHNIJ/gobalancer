package middleware

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// bucket is a per-IP token bucket.
type bucket struct {
	tokens    int64 // atomic
	lastRefill int64 // atomic unix nano
}

// RateLimiter implements a per-IP token-bucket limiter with lock-free hot path.
// Tokens are refilled lazily on each request rather than by a background ticker,
// keeping the code simple and avoiding goroutine-per-IP overhead.
type RateLimiter struct {
	rate      float64 // tokens per second
	burst     int64   // max tokens (bucket capacity)
	mu        sync.Mutex
	buckets   map[string]*bucket
}

func NewRateLimiter(ratePerSec float64, burst int) *RateLimiter {
	return &RateLimiter{
		rate:    ratePerSec,
		burst:   int64(burst),
		buckets: make(map[string]*bucket),
	}
}

func (rl *RateLimiter) bucket(ip string) *bucket {
	rl.mu.Lock()
	b, ok := rl.buckets[ip]
	if !ok {
		b = &bucket{
			tokens:    rl.burst,
			lastRefill: time.Now().UnixNano(),
		}
		rl.buckets[ip] = b
	}
	rl.mu.Unlock()
	return b
}

// Allow returns true if the request may proceed, false if it should be rate-limited.
func (rl *RateLimiter) Allow(ip string) bool {
	b := rl.bucket(ip)

	now := time.Now().UnixNano()
	last := atomic.SwapInt64(&b.lastRefill, now)
	elapsed := float64(now-last) / float64(time.Second)
	refill := int64(elapsed * rl.rate)

	if refill > 0 {
		cur := atomic.LoadInt64(&b.tokens)
		newVal := cur + refill
		if newVal > rl.burst {
			newVal = rl.burst
		}
		atomic.StoreInt64(&b.tokens, newVal)
	}

	for {
		cur := atomic.LoadInt64(&b.tokens)
		if cur <= 0 {
			return false
		}
		if atomic.CompareAndSwapInt64(&b.tokens, cur, cur-1) {
			return true
		}
	}
}

// Middleware returns an http.Handler that enforces the rate limit.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !rl.Allow(ip) {
			w.Header().Set("Retry-After", "1")
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// StartCleanup periodically evicts stale buckets to prevent unbounded memory growth.
func (rl *RateLimiter) StartCleanup(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cutoff := time.Now().Add(-interval).UnixNano()
				rl.mu.Lock()
				for ip, b := range rl.buckets {
					if atomic.LoadInt64(&b.lastRefill) < cutoff {
						delete(rl.buckets, ip)
					}
				}
				rl.mu.Unlock()
			}
		}
	}()
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	ip := r.RemoteAddr
	for i := len(ip) - 1; i >= 0; i-- {
		if ip[i] == ':' {
			return ip[:i]
		}
	}
	return ip
}
