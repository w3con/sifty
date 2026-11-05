#!/usr/bin/env bash
# Step 2.1 — the Go module builds to a single runnable binary.
set -euo pipefail
ROOT="$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)"
cd "$ROOT/sifty"
OUT="$(mktemp -d)/sifty"
trap 'rm -rf "$(dirname "$OUT")"' EXIT
go build -o "$OUT" ./cmd/sifty
echo "built: $OUT"
echo "== runs: version =="
"$OUT" version
echo "== runs: help =="
"$OUT" help >/dev/null && echo "help ok"
