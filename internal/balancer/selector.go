package balancer

import (
	"errors"
	"net/http"

	"github.com/ninijhawan/gobalancer/internal/backend"
)

// ErrNoBackends is returned when no healthy backend is available.
var ErrNoBackends = errors.New("no healthy backends available")

// Selector picks the next backend for a given request.
// All implementations must be safe for concurrent use.
type Selector interface {
	Next(r *http.Request, backends []*backend.Backend) (*backend.Backend, error)
}
