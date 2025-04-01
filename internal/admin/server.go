package admin

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ninijhawan/gobalancer/internal/backend"
)

// Server is the admin HTTP API running on a separate port (:9001).
// It provides runtime introspection and mutation of the backend pool
// without touching production traffic.
type Server struct {
	pool *backend.Pool
	srv  *http.Server
}

func New(addr string, pool *backend.Pool) *Server {
	s := &Server{pool: pool}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /admin/backends", s.handleListBackends)
	mux.HandleFunc("POST /admin/backends", s.handleAddBackend)
	mux.HandleFunc("DELETE /admin/backends/", s.handleRemoveBackend)
	mux.HandleFunc("PUT /admin/backends/", s.handleUpdateWeight)
	mux.HandleFunc("POST /admin/reload", s.handleReload)

	s.srv = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	return s
}

func (s *Server) Start(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		s.srv.Shutdown(context.Background())
	}()
	slog.Info("admin server listening", "addr", s.srv.Addr)
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) handleListBackends(w http.ResponseWriter, r *http.Request) {
	snaps := s.pool.Snapshots()
	writeJSON(w, http.StatusOK, map[string]interface{}{"backends": snaps})
}

func (s *Server) handleAddBackend(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID     string `json:"id"`
		Addr   string `json:"addr"`
		Weight int    `json:"weight"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	b, err := backend.New(req.ID, req.Addr, req.Weight)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.pool.Add(b)
	writeJSON(w, http.StatusCreated, b.Snapshot())
}

func (s *Server) handleRemoveBackend(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/admin/backends/")
	if err := s.pool.Remove(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleUpdateWeight(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/admin/backends/")
	id = strings.TrimSuffix(id, "/weight")

	var req struct {
		Weight int `json:"weight"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	b, ok := s.pool.Get(id)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	b.Weight = req.Weight
	writeJSON(w, http.StatusOK, b.Snapshot())
}

func (s *Server) handleReload(w http.ResponseWriter, _ *http.Request) {
	// Config reload signal — actual reload wired in cmd/gobalancer/main.go.
	slog.Info("hot reload requested via admin API")
	w.WriteHeader(http.StatusAccepted)
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
