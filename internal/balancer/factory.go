// Package balancer provides pluggable load-balancing algorithms that all
// satisfy the Selector interface. New algorithms can be added without
// touching the proxy core.
package balancer

import "fmt"

// NewSelector constructs a Selector by algorithm name.
// Valid names: round_robin, weighted_round_robin, least_connections,
// ip_hash, consistent_hash, p2c.
func NewSelector(algorithm string) (Selector, error) {
	switch algorithm {
	case "round_robin":
		return NewRoundRobin(), nil
	case "weighted_round_robin":
		return NewWeightedRoundRobin(), nil
	case "least_connections":
		return NewLeastConnections(), nil
	case "ip_hash":
		return NewIPHash(), nil
	case "consistent_hash":
		return NewConsistentHash(defaultVirtualNodes), nil
	case "p2c":
		return NewP2C(), nil
	default:
		return nil, fmt.Errorf("unknown algorithm %q; valid: round_robin, weighted_round_robin, least_connections, ip_hash, consistent_hash, p2c", algorithm)
	}
}
