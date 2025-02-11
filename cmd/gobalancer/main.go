package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ninijhawan/gobalancer/config"
	"github.com/ninijhawan/gobalancer/internal/backend"
	"github.com/ninijhawan/gobalancer/internal/balancer"
	"github.com/ninijhawan/gobalancer/internal/proxy"
)

func main() {
	cfgPath := flag.String("config", "config/config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}

	// Build backend pool from first pool config (multi-pool routing added later)
	poolCfg := cfg.Pools[0]
	var backends []*backend.Backend
	for _, bc := range poolCfg.Backends {
		b, err := backend.New(bc.ID, bc.Addr, bc.Weight)
		if err != nil {
			slog.Error("create backend", "id", bc.ID, "err", err)
			os.Exit(1)
		}
		backends = append(backends, b)
	}
	pool := backend.NewPool(backends)

	sel := balancer.NewRoundRobin()
	httpProxy := proxy.NewHTTPProxy(pool, sel)

	srv := proxy.NewHTTPServer(
		cfg.Server.Addr,
		httpProxy,
		cfg.Server.ReadTimeout,
		cfg.Server.WriteTimeout,
		cfg.Server.IdleTimeout,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("gobalancer starting", "addr", cfg.Server.Addr)

	errC := make(chan error, 1)
	go func() { errC <- srv.ListenAndServe() }()

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received, draining...")
		shutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutCtx); err != nil {
			slog.Error("graceful shutdown failed", "err", err)
			os.Exit(1)
		}
		slog.Info("shutdown complete")
	case err := <-errC:
		if err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}
}
