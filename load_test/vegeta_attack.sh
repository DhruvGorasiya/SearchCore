#!/usr/bin/env bash
# SearchCore load test using vegeta.
#
# Prerequisites:
#   brew install vegeta   (or: go install github.com/tsenart/vegeta@latest)
#
# The server must be running with at least some documents indexed.
# If using the gRPC-only server, run this via ghz instead (see comments at bottom).
#
# Usage:
#   bash load_test/vegeta_attack.sh [RATE] [DURATION]
#
# Defaults: 500 RPS for 30 seconds.

set -euo pipefail

RATE="${1:-500}"
DURATION="${2:-30s}"
TARGET_URL="http://localhost:8080/search"    # adjust if using grpc-gateway
OUTPUT_DIR="load_test/results"

mkdir -p "$OUTPUT_DIR"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
REPORT_FILE="$OUTPUT_DIR/report_${RATE}rps_${TIMESTAMP}.txt"
BIN_FILE="$OUTPUT_DIR/attack_${RATE}rps_${TIMESTAMP}.bin"

QUERIES=(
  "search engine inverted index"
  "bm25 ranking relevance score"
  "golang performance benchmark"
  "distributed database cluster"
  "token frequency document retrieval"
  "grpc protobuf service"
  "concurrent goroutine latency"
  "corpus statistics avg length"
)

# Pick a random query from the list.
QUERY="${QUERIES[$RANDOM % ${#QUERIES[@]}]}"

echo "SearchCore Load Test"
echo "  Target:   $TARGET_URL"
echo "  Rate:     ${RATE} RPS"
echo "  Duration: $DURATION"
echo "  Query:    \"$QUERY\""
echo ""

# Run the attack.
echo "POST $TARGET_URL
Content-Type: application/json
@load_test/body.json" | vegeta attack \
  -rate="$RATE" \
  -duration="$DURATION" \
  -timeout=5s \
  | tee "$BIN_FILE" \
  | vegeta report | tee "$REPORT_FILE"

echo ""
echo "Full report saved to: $REPORT_FILE"
echo ""
vegeta report -type=hist[0,1ms,5ms,10ms,20ms,50ms] < "$BIN_FILE"

# --- ghz alternative for pure gRPC (no HTTP gateway) ---
#
# ghz --insecure \
#     --proto proto/searchcore.proto \
#     --call searchcore.SearchService.Search \
#     --data '{"query":"search engine","top_k":10}' \
#     --rps 500 \
#     --duration 30s \
#     localhost:50051
