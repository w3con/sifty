#!/usr/bin/env bash
# Step 4.1 — the Go application-answer preparation (the JOB2 flow) matches the TypeScript
# generateApplication golden on a fixture vacancy, with the model call injected. The golden was
# generated from the real TypeScript generateApplication (with a stub store and injected inference)
# by checks/tsgolden/gen-prepare.mjs.
set -euo pipefail
ROOT="$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)"
cd "$ROOT/sifty"
go test ./internal/prepare/ -v
