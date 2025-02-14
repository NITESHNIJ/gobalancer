package proxy

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

// TransportConfig holds all tunable parameters for the upstream HTTP transport.
type TransportConfig struct {
	DialTimeout           time.Duration
	KeepAlive             time.Duration
	MaxIdleConns          int
	MaxIdleConnsPerHost   int
	IdleConnTimeout       time.Duration
	TLSHandshakeTimeout   time.Duration
	ResponseHeaderTimeout time.Duration
	InsecureSkipVerify    bool
}

// DefaultTransportConfig returns sensible production defaults.
func DefaultTransportConfig() TransportConfig {
	return TransportConfig{
		DialTimeout:           10 * time.Second,
		KeepAlive:             30 * time.Second,
		MaxIdleConns:          200,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}
}

// NewTransport creates an *http.Transport from config.
func NewTransport(cfg TransportConfig) *http.Transport {
	return &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   cfg.DialTimeout,
			KeepAlive: cfg.KeepAlive,
		}).DialContext,
		MaxIdleConns:          cfg.MaxIdleConns,
		MaxIdleConnsPerHost:   cfg.MaxIdleConnsPerHost,
		IdleConnTimeout:       cfg.IdleConnTimeout,
		TLSHandshakeTimeout:   cfg.TLSHandshakeTimeout,
		ResponseHeaderTimeout: cfg.ResponseHeaderTimeout,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.InsecureSkipVerify,
		},
	}
}

// deadlineConn wraps a net.Conn and sets read/write deadlines on every operation.
type deadlineConn struct {
	net.Conn
	readDeadline  time.Duration
	writeDeadline time.Duration
}

func (d *deadlineConn) Read(b []byte) (int, error) {
	if d.readDeadline > 0 {
		d.Conn.SetReadDeadline(time.Now().Add(d.readDeadline))
	}
	return d.Conn.Read(b)
}

func (d *deadlineConn) Write(b []byte) (int, error) {
	if d.writeDeadline > 0 {
		d.Conn.SetWriteDeadline(time.Now().Add(d.writeDeadline))
	}
	return d.Conn.Write(b)
}
