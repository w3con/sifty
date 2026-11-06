#!/usr/bin/env bash
# Step 2.4 — a stub TypeScript worker round-trips a request from the Go core over the protocol.
set -euo pipefail
ROOT="$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)"
cd "$ROOT/sifty"
go test ./internal/worker/ -run 'RoundTrip|HandledError|Timeout' -v
