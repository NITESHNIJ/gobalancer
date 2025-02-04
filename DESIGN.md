# gobalancer — Design Overview

## Architecture

gobalancer is a production-grade L4/L7 TCP & HTTP load balancer built from scratch in Go.
It operates at both the raw TCP layer (L4) and the HTTP/HTTPS application layer (L7),
making routing decisions based on configurable algorithms and rules.

```
                        ┌──────────────────────────────────┐
   Clients              │           gobalancer              │           Backends
                        │                                  │
  HTTP/HTTPS ──────────>│  Listener (TLS termination)      │──────────> Backend A
  TCP         ──────────>│  ↓                               │──────────> Backend B
                        │  Router (path / host / header)   │──────────> Backend C
                        │  ↓                               │
                        │  Middleware pipeline             │
                        │  ↓                               │
                        │  Selector (balancing algorithm)  │
                        │  ↓                               │
                        │  Health-aware proxy              │
                        │  ↓                               │
                        │  Circuit Breaker                 │
                        │  ↓                               │
                        │  Upstream connection pool        │
                        │                                  │
                        │  Admin API (:9001)               │
                        │  Metrics  (:9001/metrics)        │
                        └──────────────────────────────────┘
```

## Balancing Algorithms

All algorithms implement the `Selector` interface:

```go
type Selector interface {
    Next(r *http.Request, backends []*Backend) (*Backend, error)
}
```

| Algorithm | Best For | Complexity |
|-----------|----------|------------|
| Round Robin | Uniform backends | O(1) |
| Weighted Round Robin | Heterogeneous capacity | O(1) amortised |
| Least Connections | Long-lived or variable requests | O(n) |
| IP Hash | Sticky sessions without shared state | O(1) |
| Consistent Hash | Session affinity + minimal disruption on scale | O(log n) |
| Power of Two Choices | Near-optimal load with low overhead | O(1) |

### Consistent Hashing vs IP Hash

IP Hash uses `clientIP % len(backends)`. Adding or removing one backend
remaps ~50% of clients. Consistent hashing places 150 virtual nodes per
backend on a 2^32 ring; adding one backend remaps only ~1/N clients.
Use consistent hashing when session affinity matters at scale.

### Power of Two Choices

Pick two backends uniformly at random; route to the one with fewer active
connections. Reduces worst-case load from O(log n / log log n) toward
O(log log n) vs pure random — Mitzenmacher (2001).

## Health Checking

### Active checks

A background goroutine per backend probes health on a configurable interval.
Three probe types: HTTP GET (status code + optional body pattern), TCP dial,
gRPC Health/Check RPC.

### Passive outlier detection

Every proxied response is classified. A sliding-window counter tracks
5xx errors and connection timeouts per backend. When the error rate crosses
a threshold (e.g. 30% in 60 s), the backend is automatically ejected
with exponential backoff before re-admission.

### State machine

```
         N consecutive failures
Healthy ─────────────────────────> Unhealthy
    ^                                  │
    │         M consecutive successes  │ backoff expires + probe succeeds
    └──── Probation <──────────────────┘
```

## Circuit Breaker

Per-backend FSM that fails fast instead of waiting on dead upstreams:

```
Closed ──(failure threshold)──> Open ──(timeout)──> Half-Open
  ^                                                      │
  └────────────────(probe succeeds)─────────────────────┘
                        │
                 (probe fails) ──> Open
```

In Open state requests return `ErrCircuitOpen` in microseconds rather than
waiting for a 30-second TCP timeout. Half-Open uses a single-permit semaphore
to prevent thundering herd on recovery.

## Connection Draining

On SIGTERM:
1. Stop accepting new connections.
2. Set an atomic drain flag — Selector excludes draining backends.
3. Wait for `sync.WaitGroup` tracking in-flight requests to reach zero.
4. Configured drain timeout prevents indefinite hang on stuck connections.

## Concurrency Model

- One goroutine per accepted connection (TCP mode).
- `httputil.ReverseProxy` manages goroutine lifecycle for HTTP mode.
- Backend state (health, active connections, weight) uses `sync/atomic` for
  hot-path reads and `sync.RWMutex` for infrequent writes.
- All algorithms tested with `go test -race`.

## Admin API

Runs on a separate port (`:9001`) to avoid polluting production metrics.
All mutations are atomic and idempotent. No authentication in v1
(assumes internal network); mTLS planned for v2.

## Observability

Every hot path emits Prometheus metrics:

| Metric | Type | Labels |
|--------|------|--------|
| `gobalancer_requests_total` | counter | backend, status_class |
| `gobalancer_request_duration_seconds` | histogram | backend, route |
| `gobalancer_active_connections` | gauge | backend |
| `gobalancer_backend_health` | gauge | backend |
| `gobalancer_circuit_breaker_state` | gauge | backend |
| `gobalancer_upstream_errors_total` | counter | backend, type |
