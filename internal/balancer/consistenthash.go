package balancer

import (
	"fmt"
	"net/http"
	"sort"
	"sync"

	"github.com/ninijhawan/gobalancer/internal/backend"
)

const defaultVirtualNodes = 150

// ConsistentHash places virtualNodes replicas of each backend on a 2^32 ring.
// Requests are routed clockwise to the first virtual node ≥ hash(clientIP).
// Adding one backend remaps only ~1/N of existing keys.
type ConsistentHash struct {
	mu           sync.RWMutex
	ring         []uint32
	nodeMap      map[uint32]*backend.Backend
	virtualNodes int
}

func NewConsistentHash(virtualNodes int) *ConsistentHash {
	if virtualNodes <= 0 {
		virtualNodes = defaultVirtualNodes
	}
	return &ConsistentHash{
		nodeMap:      make(map[uint32]*backend.Backend),
		virtualNodes: virtualNodes,
	}
}

// Build (re)constructs the ring from a backend slice. Call whenever the
// backend set changes.
func (ch *ConsistentHash) Build(backends []*backend.Backend) {
	ring := make([]uint32, 0, len(backends)*ch.virtualNodes)
	nodeMap := make(map[uint32]*backend.Backend, len(backends)*ch.virtualNodes)

	for _, b := range backends {
		for i := 0; i < ch.virtualNodes; i++ {
			key := fnv32a(fmt.Sprintf("%s#%d", b.ID, i))
			ring = append(ring, key)
			nodeMap[key] = b
		}
	}
	sort.Slice(ring, func(i, j int) bool { return ring[i] < ring[j] })

	ch.mu.Lock()
	ch.ring = ring
	ch.nodeMap = nodeMap
	ch.mu.Unlock()
}

func (ch *ConsistentHash) Next(r *http.Request, backends []*backend.Backend) (*backend.Backend, error) {
	ch.mu.RLock()
	ring := ch.ring
	nodeMap := ch.nodeMap
	ch.mu.RUnlock()

	if len(ring) == 0 {
		ch.Build(backends)
		ch.mu.RLock()
		ring = ch.ring
		nodeMap = ch.nodeMap
		ch.mu.RUnlock()
	}
	if len(ring) == 0 {
		return nil, ErrNoBackends
	}

	hash := fnv32a(clientIP(r))
	// Binary search for the first ring position ≥ hash (clockwise walk).
	idx := sort.Search(len(ring), func(i int) bool { return ring[i] >= hash })
	if idx == len(ring) {
		idx = 0 // wrap around
	}
	return nodeMap[ring[idx]], nil
}
