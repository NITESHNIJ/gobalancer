package proxy

import (
	"io"
	"log/slog"
	"net"
)

// pipe copies data bidirectionally between two net.Conn until both sides close.
// It uses TCP half-close so each direction signals EOF independently.
func pipe(a, b net.Conn) {
	errc := make(chan error, 2)

	cp := func(dst, src net.Conn) {
		_, err := io.Copy(dst, src)
		// Signal write-side EOF so the peer knows we're done sending.
		if tc, ok := dst.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
		errc <- err
	}

	go cp(a, b)
	go cp(b, a)

	for i := 0; i < 2; i++ {
		if err := <-errc; err != nil && !isConnClosed(err) {
			slog.Debug("tunnel copy error", "err", err)
		}
	}
}

func isConnClosed(err error) bool {
	if err == nil || err == io.EOF {
		return true
	}
	if ne, ok := err.(net.Error); ok && !ne.Timeout() {
		return true
	}
	return false
}
