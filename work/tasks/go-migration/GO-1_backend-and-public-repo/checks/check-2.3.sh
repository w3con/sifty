#!/usr/bin/env bash
# Step 2.3 — a first answer/feedback write produces files matching the embedded templates.
set -euo pipefail
ROOT="$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)"
cd "$ROOT/sifty"
go test ./internal/records/ -run 'MatchesTemplate|Exclusive' -v
