package proxy

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/ninijhawan/gobalancer/internal/backend"
)

type TCPProxy struct {
	listener net.Listener
	pool     *backend.Pool
	dialTimeout time.Duration
	wg       sync.WaitGroup
}

func NewTCPProxy(addr string, pool *backend.Pool, dialTimeout time.Duration) (*TCPProxy, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("tcp listen %s: %w", addr, err)
	}
	if dialTimeout == 0 {
		dialTimeout = 10 * time.Second
	}
	return &TCPProxy{
		listener:    l,
		pool:        pool,
		dialTimeout: dialTimeout,
	}, nil
}

func (p *TCPProxy) Serve(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		p.listener.Close()
	}()

	for {
		conn, err := p.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				p.wg.Wait()
				return nil
			default:
				return fmt.Errorf("accept: %w", err)
			}
		}
		p.wg.Add(1)
		go func(c net.Conn) {
			defer p.wg.Done()
			p.handleConn(c)
		}(conn)
	}
}

func (p *TCPProxy) handleConn(client net.Conn) {
	defer client.Close()

	backends := p.pool.Healthy()
	if len(backends) == 0 {
		slog.Warn("no healthy backends available")
		return
	}

	// Simple round-robin for TCP — full selector wired in HTTP proxy
	b := backends[0]
	b.IncConns()
	defer b.DecConns()

	upstream, err := net.DialTimeout("tcp", b.URL.Host, p.dialTimeout)
	if err != nil {
		slog.Error("dial upstream", "backend", b.ID, "err", err)
		return
	}
	defer upstream.Close()

	tunnel(client, upstream)
}

func tunnel(a, b net.Conn) {
	done := make(chan struct{}, 2)
	copy := func(dst, src net.Conn) {
		io.Copy(dst, src)
		dst.(*net.TCPConn).CloseWrite()
		done <- struct{}{}
	}
	go copy(a, b)
	go copy(b, a)
	<-done
	<-done
}
