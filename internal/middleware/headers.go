package middleware

import (
	"net/http"
)

// HeaderRewriter adds, removes, or overrides response headers from a config map.
// Key format: "add:<header>", "remove:<header>", "set:<header>".
// All rules are applied before the handler writes headers.
type HeaderRewriter struct {
	adds    map[string]string // header → value
	removes []string
	sets    map[string]string
}

func NewHeaderRewriter() *HeaderRewriter {
	return &HeaderRewriter{
		adds: make(map[string]string),
		sets: make(map[string]string),
	}
}

func (h *HeaderRewriter) Add(name, value string) { h.adds[name] = value }
func (h *HeaderRewriter) Remove(name string)      { h.removes = append(h.removes, name) }
func (h *HeaderRewriter) Set(name, value string)  { h.sets[name] = value }

func (h *HeaderRewriter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		for k, v := range h.adds {
			if w.Header().Get(k) == "" {
				w.Header().Set(k, v)
			}
		}
		for _, k := range h.removes {
			w.Header().Del(k)
		}
		for k, v := range h.sets {
			w.Header().Set(k, v)
		}
	})
}

// SecurityHeaders adds common security headers to every response.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	})
}
