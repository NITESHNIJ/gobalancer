package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

var gzipPool = sync.Pool{
	New: func() interface{} {
		w, _ := gzip.NewWriterLevel(io.Discard, gzip.DefaultCompression)
		return w
	},
}

type gzipResponseWriter struct {
	http.ResponseWriter
	gw *gzip.Writer
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	return g.gw.Write(b)
}

func (g *gzipResponseWriter) WriteHeader(code int) {
	g.ResponseWriter.Header().Del("Content-Length") // length changes after compression
	g.ResponseWriter.WriteHeader(code)
}

// GzipMiddleware compresses responses above minBytes if the client sends Accept-Encoding: gzip.
func GzipMiddleware(minBytes int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				next.ServeHTTP(w, r)
				return
			}

			gw := gzipPool.Get().(*gzip.Writer)
			gw.Reset(w)
			defer func() {
				gw.Close()
				gzipPool.Put(gw)
			}()

			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Del("Content-Length")
			next.ServeHTTP(&gzipResponseWriter{ResponseWriter: w, gw: gw}, r)
		})
	}
}
