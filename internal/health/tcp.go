package health

import (
	"net"
	"time"
)

// probeTCP verifies a backend is reachable at the TCP level without HTTP overhead.
// Used for non-HTTP backends (databases, message queues, raw TCP services).
func probeTCP(addr string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
