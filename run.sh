#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

LOG_FILE="$ROOT_DIR/experiment.log"

echo "Running experiment..."
go test -race -v -count=1 -run '^TestExperiment$' ./internal/kademlia 2>&1 | tee "$LOG_FILE"

echo "Creating plots..."
go run ./tools/plot.go

echo "Created:"
printf '%s\n' scalability-hops.svg scalability-probes.svg reliability.svg