package backend

import (
	"fmt"
	"sync"
)

type Pool struct {
	mu       sync.RWMutex
	backends []*Backend
}

func NewPool(backends []*Backend) *Pool {
	return &Pool{backends: backends}
}

func (p *Pool) Backends() []*Backend {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]*Backend, len(p.backends))
	copy(out, p.backends)
	return out
}

func (p *Pool) Healthy() []*Backend {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var out []*Backend
	for _, b := range p.backends {
		if b.IsHealthy() {
			out = append(out, b)
		}
	}
	return out
}

func (p *Pool) Add(b *Backend) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.backends = append(p.backends, b)
}

func (p *Pool) Remove(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, b := range p.backends {
		if b.ID == id {
			p.backends = append(p.backends[:i], p.backends[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("backend %q not found", id)
}

func (p *Pool) Get(id string) (*Backend, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, b := range p.backends {
		if b.ID == id {
			return b, true
		}
	}
	return nil, false
}

func (p *Pool) Snapshots() []Snapshot {
	p.mu.RLock()
	defer p.mu.RUnlock()
	snaps := make([]Snapshot, len(p.backends))
	for i, b := range p.backends {
		snaps[i] = b.Snapshot()
	}
	return snaps
}
