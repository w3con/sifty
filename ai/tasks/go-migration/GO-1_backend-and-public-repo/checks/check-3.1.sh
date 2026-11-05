#!/usr/bin/env bash
# Step 3.1 — native Go collection parsers match the TypeScript goldens on fixtures, and the
# jobspy scraping worker is integrated as a sidecar. Goldens were generated from the current
# TypeScript parsers by checks/tsgolden/gen-golden.mjs.
set -euo pipefail
ROOT="$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)"
cd "$ROOT/sifty"
go test ./internal/collect/ -v
