#!/usr/bin/env bash
# Step 1.4 — the CI workflow file is present and the build-and-test entry runs locally.
set -euo pipefail
ROOT="$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)"
SRC="$ROOT/sifty"
echo "== workflow present =="
test -f "$SRC/.github/workflows/ci.yml" && echo "PASS: .github/workflows/ci.yml present"
echo "== test entry runs locally (in a fresh repo root) =="
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
cp -R "$SRC"/. "$WORK"/
cd "$WORK"
git init -q
git config user.email t@e.com
git config user.name t
git remote add origin git@github.com:w3con/sifty.git
bash scripts/test
