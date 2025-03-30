package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// responseRecorder captures status code and bytes written for logging.
type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{ResponseWriter: w, status: http.StatusOK}
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.bytes += int64(n)
	return n, err
}

// AccessLogger emits a structured JSON log line for every request.
func AccessLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := newResponseRecorder(w)
		next.ServeHTTP(rec, r)
		latency := time.Since(start)

		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"bytes", rec.bytes,
			"latency_ms", latency.Milliseconds(),
			"client_ip", clientIP(r),
			"user_agent", r.UserAgent(),
		)
	})
}
