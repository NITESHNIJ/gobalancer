#!/usr/bin/env bash
# Run all load tests and write results to benchmarks/results/
set -euo pipefail

RESULTS_DIR="benchmarks/results"
mkdir -p "$RESULTS_DIR"
TARGET="${TARGET:-http://localhost:8080}"
DURATION="${DURATION:-60s}"
RPS="${RPS:-10000}"

algorithms=(round_robin weighted_round_robin least_connections consistent_hash p2c)

for algo in "${algorithms[@]}"; do
  echo "=== Testing algorithm: $algo ==="
  outfile="$RESULTS_DIR/${algo}_$(date +%Y%m%d_%H%M%S).txt"

  # Start gobalancer with the given algorithm.
  GOBALANCER_ALGORITHM="$algo" ./bin/gobalancer -config config/config.yaml &
  pid=$!
  sleep 1  # wait for listener to be ready

  wrk2 -t4 -c100 -d"$DURATION" -R"$RPS" --latency \
    -s benchmarks/load_test.lua "$TARGET" 2>&1 | tee "$outfile"

  kill "$pid"
  wait "$pid" 2>/dev/null || true
  echo "Results written to $outfile"
  echo
done

echo "All benchmarks complete. Results in $RESULTS_DIR"
