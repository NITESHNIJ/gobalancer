package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ninijhawan/gobalancer/internal/backend"
)

func setupServer(t *testing.T) (*Server, *backend.Pool) {
	t.Helper()
	b1, _ := backend.New("b1", "http://localhost:8081", 1)
	b2, _ := backend.New("b2", "http://localhost:8082", 1)
	pool := backend.NewPool([]*backend.Backend{b1, b2})
	return New(":9001", pool), pool
}

func TestAdminServer_ListBackends(t *testing.T) {
	s, _ := setupServer(t)
	r := httptest.NewRequest(http.MethodGet, "/admin/backends", nil)
	w := httptest.NewRecorder()
	s.handleListBackends(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Backends []backend.Snapshot `json:"backends"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Backends) != 2 {
		t.Errorf("expected 2 backends, got %d", len(resp.Backends))
	}
}

func TestAdminServer_AddBackend(t *testing.T) {
	s, pool := setupServer(t)
	body := `{"id":"b3","addr":"http://localhost:8083","weight":2}`
	r := httptest.NewRequest(http.MethodPost, "/admin/backends", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	s.handleAddBackend(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body)
	}
	if _, ok := pool.Get("b3"); !ok {
		t.Error("b3 not found in pool after add")
	}
}

func TestAdminServer_RemoveBackend(t *testing.T) {
	s, pool := setupServer(t)
	r := httptest.NewRequest(http.MethodDelete, "/admin/backends/b1", nil)
	w := httptest.NewRecorder()
	s.handleRemoveBackend(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	if _, ok := pool.Get("b1"); ok {
		t.Error("b1 still in pool after remove")
	}
}

func TestAdminServer_UpdateWeight(t *testing.T) {
	s, pool := setupServer(t)
	body := `{"weight":5}`
	r := httptest.NewRequest(http.MethodPut, "/admin/backends/b2/weight",
		bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	s.handleUpdateWeight(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	b, _ := pool.Get("b2")
	if b.Weight != 5 {
		t.Errorf("expected weight 5, got %d", b.Weight)
	}
}
