# gobalancer — Benchmark Results

## Environment

| | |
|---|---|
| CPU | Apple M2 Pro, 10 cores |
| RAM | 16 GB |
| OS | macOS 14 Sonoma |
| Go | 1.22.5 |
| wrk2 | v4.0.0 |
| Test backends | 3 × httptest echo servers (in-process) |

## Methodology

Each algorithm was tested under two traffic patterns:

1. **Uniform**: all three backends respond in <1 ms (no skew)
2. **Skewed**: backend C has 50 ms artificial delay (simulating a slow node)

Test command:
```
wrk2 -t4 -c100 -d60s -R10000 --latency -s benchmarks/load_test.lua http://localhost:8080
```

## Results — Uniform Backends (10k RPS, 100 connections)

| Algorithm | Throughput (RPS) | p50 (μs) | p95 (μs) | p99 (μs) | Max (μs) |
|-----------|-----------------|-----------|-----------|-----------|----------|
| Round Robin | 10,018 | 412 | 891 | 1,243 | 4,102 |
| Weighted RR | 9,994 | 418 | 903 | 1,271 | 4,287 |
| Least Conn | 10,021 | 407 | 876 | 1,218 | 3,891 |
| Consistent Hash | 9,987 | 421 | 912 | 1,289 | 4,441 |
| P2C | 10,015 | 409 | 882 | 1,231 | 3,974 |

**Observation**: Under uniform load, all algorithms perform similarly. Consistent Hash
has ~2% higher p99 due to binary search overhead on ring lookup.

## Results — Skewed Backends (backend C at +50 ms)

| Algorithm | Throughput (RPS) | p50 (μs) | p95 (μs) | p99 (μs) | Max (μs) |
|-----------|-----------------|-----------|-----------|-----------|----------|
| Round Robin | 9,987 | 17,201 | 51,334 | 54,221 | 61,102 |
| Weighted RR | 9,994 | 16,987 | 50,887 | 53,991 | 60,441 |
| Least Conn | 10,009 | 389 | 842 | 1,189 | 3,712 |
| Consistent Hash | 9,991 | 16,102 | 50,443 | 53,771 | 59,987 |
| P2C | 10,017 | 401 | 869 | 1,204 | 3,887 |

**Observation**: Round Robin and Consistent Hash distribute evenly to the slow backend,
driving p99 to ~54 ms. **Least Connections** and **P2C** naturally avoid the slow
backend as it accumulates active connections — p99 stays under 1.3 ms.

## Algorithm Microbenchmarks (ns/op)

Measured with `go test -bench=. -benchmem ./internal/balancer/`

| Benchmark | 1k backends (ns/op) | 10k backends (ns/op) | Allocs/op |
|-----------|---------------------|----------------------|-----------|
| RoundRobin | 8.2 | 8.3 | 0 |
| LeastConn | 2,341 | 23,182 | 0 |
| IPHash | 112 | 113 | 32 |
| ConsistentHash | 287 | 334 | 32 |
| P2C | 14.1 | 14.3 | 0 |

**Observation**: Round Robin and P2C are O(1) and allocation-free. Least Connections
is O(n) — viable up to ~1k backends, after which P2C is preferred for similar load
quality with constant time.

## Memory at Load

At 50k concurrent connections (vegeta spike test):
- Heap: 48 MB
- Goroutines: 50,012 (one per conn + background workers)
- GC pause p99: 1.2 ms

GOMAXPROCS tuning: setting `GOMAXPROCS=runtime.NumCPU()` (default) is optimal.
Pinning to fewer cores reduces throughput by ~15% per core removed.
