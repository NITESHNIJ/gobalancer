package metrics

import (
	"net/http"
	"strconv"
	"time"
)

// InstrumentHandler wraps h with Prometheus instrumentation.
// backend and route labels are extracted from the request context.
func (c *Collector) InstrumentHandler(backendID, route string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		start := time.Now()

		c.ActiveConns.WithLabelValues(backendID).Inc()
		defer c.ActiveConns.WithLabelValues(backendID).Dec()

		h.ServeHTTP(rec, r)

		duration := time.Since(start).Seconds()
		class := StatusClass(rec.status)

		c.RequestsTotal.WithLabelValues(backendID, class).Inc()
		c.RequestDuration.WithLabelValues(backendID, route).Observe(duration)

		if rec.status >= 500 {
			c.UpstreamErrors.WithLabelValues(backendID, strconv.Itoa(rec.status)).Inc()
		}
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
