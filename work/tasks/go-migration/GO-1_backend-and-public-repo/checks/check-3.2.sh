#!/usr/bin/env bash
# Step 3.2 — the Go fit-assessment result matches the TypeScript assess() golden within tolerance
# on a fixture, with the model call injected. The golden was generated from the real TypeScript
# assess() (with a stub store and injected inference) by checks/tsgolden/gen-assess.mjs.
set -euo pipefail
ROOT="$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)"
cd "$ROOT/sifty"
go test ./internal/assess/ -v
