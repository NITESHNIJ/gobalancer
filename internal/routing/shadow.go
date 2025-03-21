package routing

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// ShadowTimeout is the hard deadline for shadow requests.
// Shadow responses are discarded; this prevents runaway goroutines.
const ShadowTimeout = 5 * time.Second

// Shadow clones r and fires it at targetURL asynchronously.
// The response is discarded — shadowing is purely for traffic mirroring
// (e.g. testing a new backend with production-shaped load).
// Errors are logged at DEBUG level only; the caller's response is unaffected.
func Shadow(r *http.Request, targetURL string, client *http.Client) {
	// Clone the body so the original request is not consumed.
	var bodyBytes []byte
	if r.Body != nil {
		bodyBytes, _ = io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), ShadowTimeout)
		defer cancel()

		clone, err := http.NewRequestWithContext(ctx, r.Method, targetURL+r.URL.RequestURI(), bytes.NewReader(bodyBytes))
		if err != nil {
			slog.Debug("shadow: clone request failed", "err", err)
			return
		}
		for k, vs := range r.Header {
			clone.Header[k] = vs
		}

		resp, err := client.Do(clone)
		if err != nil {
			slog.Debug("shadow: upstream error", "url", targetURL, "err", err)
			return
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		slog.Debug("shadow: response", "status", resp.StatusCode, "url", targetURL)
	}()
}

// ShadowMiddleware wraps a handler to mirror every request to shadowURL.
func ShadowMiddleware(shadowURL string, next http.Handler) http.Handler {
	client := &http.Client{Timeout: ShadowTimeout}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Shadow(r, shadowURL, client)
		next.ServeHTTP(w, r)
	})
}
