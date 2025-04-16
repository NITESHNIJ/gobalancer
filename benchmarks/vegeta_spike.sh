#!/usr/bin/env bash
# Vegeta spike test — ramp to 50k concurrent connections
# Requires: vegeta (https://github.com/tsenart/vegeta)
set -euo pipefail

TARGET="${TARGET:-http://localhost:8080/}"
RESULTS_DIR="benchmarks/results"
mkdir -p "$RESULTS_DIR"

echo "GET $TARGET" | vegeta attack \
  -rate=50000 \
  -duration=30s \
  -timeout=5s \
  -workers=200 \
  | tee "$RESULTS_DIR/vegeta_spike.bin" \
  | vegeta report

echo ""
echo "Latency histogram:"
vegeta report -type=hist[0,1ms,5ms,10ms,50ms,100ms] < "$RESULTS_DIR/vegeta_spike.bin"

echo ""
echo "Plot (requires gnuplot):"
vegeta plot < "$RESULTS_DIR/vegeta_spike.bin" > "$RESULTS_DIR/vegeta_spike.html"
echo "Open $RESULTS_DIR/vegeta_spike.html in a browser"
