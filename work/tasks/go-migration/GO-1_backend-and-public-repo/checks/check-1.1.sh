#!/usr/bin/env bash
# Step 1.1 — a fresh public repo tracks only public code, never a private path.
set -euo pipefail
ROOT="$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)"
SRC="$ROOT/sifty"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
cp -R "$SRC"/. "$WORK"/
cd "$WORK"
# Seed decoy private paths that must stay untracked.
mkdir -p ai kb bin config prompts data .local
echo "private" > ai/secret.md
echo "private" > kb/note.md
echo "SECRET=1" > .env
: > cv.pdf
: > profile.pdf
echo "store" > data/store.sqlite
git init -q
git add -A
echo "== tracked/staged files =="
git status --porcelain
echo "== assert no private path staged =="
if git status --porcelain | grep -E ' (ai/|kb/|bin/|config/|prompts/|data/|\.local/|\.env|cv\.pdf|profile\.pdf)'; then
  echo "FAIL: a private path is tracked"; exit 1
fi
echo "PASS: no private path is tracked"
echo "== git check-ignore confirms exclusion =="
git check-ignore ai/secret.md kb/note.md .env cv.pdf profile.pdf data/store.sqlite .local
