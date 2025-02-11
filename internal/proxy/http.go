package proxy

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/ninijhawan/gobalancer/internal/backend"
)

const ctxKeyBackend = ctxKey("backend")

type ctxKey string

// HTTPProxy is an HTTP/HTTPS reverse proxy that delegates backend selection
// to a pluggable Selector and handles X-Forwarded-For injection.
type HTTPProxy struct {
	pool     *backend.Pool
	selector Selector
	transport http.RoundTripper
}

// Selector picks the next backend for an incoming request.
type Selector interface {
	Next(r *http.Request, backends []*backend.Backend) (*backend.Backend, error)
}

func NewHTTPProxy(pool *backend.Pool, sel Selector) *HTTPProxy {
	return &HTTPProxy{
		pool:     pool,
		selector: sel,
		transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          200,
			MaxIdleConnsPerHost:   20,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
		},
	}
}

func (h *HTTPProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	backends := h.pool.Healthy()
	if len(backends) == 0 {
		http.Error(w, "no healthy backends", http.StatusBadGateway)
		return
	}

	b, err := h.selector.Next(r, backends)
	if err != nil {
		slog.Error("selector error", "err", err)
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}

	b.IncConns()
	defer b.DecConns()

	rp := &httputil.ReverseProxy{
		Director:  h.director(b.URL),
		Transport: h.transport,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			slog.Error("upstream error", "backend", b.ID, "err", err)
			http.Error(w, "bad gateway", http.StatusBadGateway)
		},
	}
	rp.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKeyBackend, b)))
}

func (h *HTTPProxy) director(target *url.URL) func(*http.Request) {
	return func(r *http.Request) {
		r.URL.Scheme = target.Scheme
		r.URL.Host = target.Host
		r.URL.Path = singleJoiningSlash(target.Path, r.URL.Path)
		if target.RawQuery == "" || r.URL.RawQuery == "" {
			r.URL.RawQuery = target.RawQuery + r.URL.RawQuery
		} else {
			r.URL.RawQuery = target.RawQuery + "&" + r.URL.RawQuery
		}
		r.Host = target.Host

		clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			clientIP = r.RemoteAddr
		}
		if prior := r.Header.Get("X-Forwarded-For"); prior != "" {
			clientIP = prior + ", " + clientIP
		}
		r.Header.Set("X-Forwarded-For", clientIP)
		r.Header.Set("X-Forwarded-Proto", scheme(r))
	}
}

func singleJoiningSlash(a, b string) string {
	aslash := len(a) > 0 && a[len(a)-1] == '/'
	bslash := len(b) > 0 && b[0] == '/'
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		if b == "" {
			return a
		}
		return a + "/" + b
	}
	return a + b
}

func scheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

// BackendFromContext retrieves the selected backend stored by HTTPProxy.
func BackendFromContext(ctx context.Context) (*backend.Backend, bool) {
	b, ok := ctx.Value(ctxKeyBackend).(*backend.Backend)
	return b, ok
}

func NewHTTPServer(addr string, handler http.Handler, readTimeout, writeTimeout, idleTimeout time.Duration) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}
}

func ListenAndServe(ctx context.Context, srv *http.Server) error {
	errC := make(chan error, 1)
	go func() { errC <- srv.ListenAndServe() }()
	select {
	case <-ctx.Done():
		return srv.Shutdown(context.Background())
	case err := <-errC:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	}
}
