# gobalancer

A production-grade L4/L7 TCP & HTTP load balancer built from scratch in Go —
featuring pluggable routing algorithms, consistent hashing, TLS termination,
circuit breaking, and a live admin API.

[![CI](https://github.com/ninijhawan/gobalancer/actions/workflows/ci.yml/badge.svg)](https://github.com/ninijhawan/gobalancer/actions)

## Architecture

```
                     ┌─────────────────────────────────┐
  Clients            │           gobalancer             │           Backends
                     │                                 │
  HTTP/HTTPS ───────>│  :8080  Listener (TLS optional) │──────────> Backend A :8081
  TCP        ───────>│    ↓                            │──────────> Backend B :8082
                     │  L7 Router                      │──────────> Backend C :8083
                     │    path / host / header rules   │
                     │    ↓                            │
                     │  Middleware pipeline             │
                     │    rate limiter · logger · gzip │
                     │    ↓                            │
                     │  Selector (algorithm)           │
                     │    round-robin / WRR / LC       │
                     │    consistent-hash / P2C        │
                     │    ↓                            │
                     │  Circuit Breaker (per backend)  │
                     │    closed → open → half-open    │
                     │    ↓                            │
                     │  Upstream pool                  │
                     │                                 │
                     │  Admin API  :9001               │
                     │  Metrics    :9001/metrics        │
                     └─────────────────────────────────┘
```

## Features

| Category | Features |
|----------|----------|
| **Algorithms** | Round Robin, Weighted RR, Least Connections, IP Hash, Consistent Hash, P2C |
| **Health checks** | Active HTTP/TCP/gRPC, passive outlier detection, exponential backoff |
| **Circuit breaker** | Per-backend FSM: Closed → Open → Half-Open, single-permit probe |
| **TLS** | Termination, hot cert reload, SNI passthrough, mTLS |
| **L7 routing** | Path prefix trie, host header, header-based, weighted canary, request shadowing |
| **Middleware** | Rate limiter (token bucket), access logger, request ID, gzip, header rewrite |
| **Admin API** | CRUD backends, live weight update, hot config reload |
| **Observability** | Prometheus metrics on every hot path, structured JSON logging |

## Quickstart

```bash
# 1. Clone and build
git clone https://github.com/ninijhawan/gobalancer
cd gobalancer
make build

# 2. Start 3 echo backends (requires Docker)
docker compose up -d backend1 backend2 backend3

# 3. Run the load balancer
./bin/gobalancer -config config/config.yaml

# 4. Test it
curl http://localhost:8080/
curl http://localhost:9001/admin/backends  # admin API
curl http://localhost:9001/metrics         # Prometheus metrics
```

## Configuration

```yaml
server:
  addr: ":8080"
  read_timeout: 30s
  write_timeout: 30s

pools:
  - name: default
    algorithm: least_connections   # round_robin | weighted_round_robin |
                                   # least_connections | ip_hash |
                                   # consistent_hash | p2c
    backends:
      - id: b1
        addr: "http://localhost:8081"
        weight: 1
    health:
      interval: 10s
      timeout: 2s
      path: /health
      failure_threshold: 3
      success_threshold: 2
      passive_error_rate: 0.3

admin:
  addr: ":9001"
```

## Admin API

| Method | Path | Description |
|--------|------|-------------|
| GET | /admin/backends | List all backends with health/conns/circuit state |
| POST | /admin/backends | Add a backend at runtime |
| DELETE | /admin/backends/:id | Drain and remove a backend |
| PUT | /admin/backends/:id/weight | Update weight live |
| POST | /admin/reload | Hot-reload config from file |

## Benchmarks

See [BENCHMARKS.md](BENCHMARKS.md) for full results.

**TL;DR** at 10k RPS with skewed backends:

| Algorithm | p99 latency |
|-----------|------------|
| Round Robin | 54 ms (routes to slow backend) |
| Least Connections | 1.2 ms |
| P2C | 1.2 ms |

## Development

```bash
make test    # run tests
make race    # run with race detector
make bench   # run microbenchmarks
make lint    # golangci-lint
make docker  # build Docker image
```

## Design

See [DESIGN.md](DESIGN.md) for architecture decisions and algorithm analysis.
