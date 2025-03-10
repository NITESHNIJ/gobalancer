package circuitbreaker

import "sync"

// Registry manages one Breaker per backend ID.
type Registry struct {
	mu      sync.RWMutex
	cfg     Config
	breakers map[string]*Breaker
}

func NewRegistry(cfg Config) *Registry {
	return &Registry{
		cfg:      cfg,
		breakers: make(map[string]*Breaker),
	}
}

// Get returns the Breaker for a backend, creating one if it doesn't exist.
func (r *Registry) Get(backendID string) *Breaker {
	r.mu.RLock()
	b, ok := r.breakers[backendID]
	r.mu.RUnlock()
	if ok {
		return b
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if b, ok = r.breakers[backendID]; ok {
		return b
	}
	b = New(r.cfg)
	r.breakers[backendID] = b
	return b
}

// States returns a snapshot of all breaker states keyed by backend ID.
func (r *Registry) States() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]string, len(r.breakers))
	for id, b := range r.breakers {
		out[id] = b.State().String()
	}
	return out
}
