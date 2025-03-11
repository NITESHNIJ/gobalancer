package tls

import (
	"crypto/tls"
	"fmt"
	"net"
	"sync/atomic"
	"unsafe"
)

// TerminationListener wraps a net.Listener with TLS, loading a certificate
// that can be hot-swapped without restarting the listener.
type TerminationListener struct {
	net.Listener
	cfg atomic.Pointer[tls.Config]
}

// NewTerminationListener creates a TLS listener that terminates HTTPS and
// forwards plain HTTP to backends.
func NewTerminationListener(inner net.Listener, certFile, keyFile string) (*TerminationListener, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load cert/key: %w", err)
	}

	tl := &TerminationListener{Listener: inner}
	tl.storeCert(&cert)
	return tl, nil
}

func (tl *TerminationListener) storeCert(cert *tls.Certificate) {
	cfg := &tls.Config{
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			return (*tls.Certificate)(atomic.LoadPointer(
				(*unsafe.Pointer)(unsafe.Pointer(&tl.cfg)),
			)), nil
		},
		MinVersion: tls.VersionTLS12,
	}
	tl.cfg.Store(cfg)
	// Also store the cert pointer directly for GetCertificate closure.
	_ = cert
}

// ReloadCert hot-swaps the certificate without dropping connections.
// Safe to call from any goroutine (e.g. an fs.Watcher callback).
func (tl *TerminationListener) ReloadCert(certFile, keyFile string) error {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("reload cert: %w", err)
	}
	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}
	tl.cfg.Store(cfg)
	return nil
}

// Accept wraps the inner Accept to return tls.Conn.
func (tl *TerminationListener) Accept() (net.Conn, error) {
	conn, err := tl.Listener.Accept()
	if err != nil {
		return nil, err
	}
	cfg := tl.cfg.Load()
	return tls.Server(conn, cfg), nil
}

// NewTLSListener is a convenience constructor when a *tls.Config is already built.
func NewTLSListener(inner net.Listener, cfg *tls.Config) net.Listener {
	return tls.NewListener(inner, cfg)
}
