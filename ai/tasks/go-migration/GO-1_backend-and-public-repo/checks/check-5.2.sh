#!/usr/bin/env bash
# Step 5.2 — full-tree leak sweep of the public repository, the manual sweep the three change-shaped
# hooks cannot be (it reads every tracked file, not an added line). The public repository is the
# sifty/ subtree, which is published as a fresh git repo whose origin is github.com/w3con/sifty.
# It is not yet its own git repo, so this sweep reproduces the publishable tree in a throwaway repo:
# copy sifty/ honouring its own .gitignore, git init, set the public origin so untidy forms are
# checked, stage, and run the vendored tree-check. Exit 0 when clean, non-zero when any form is
# found. This script performs NO push.
set -euo pipefail
ROOT="$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)"
SRC="$ROOT/sifty"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

rsync -a \
  --exclude 'node_modules/' \
  --exclude '.git/' \
  --exclude 'data/' \
  --exclude '.local/' \
  --exclude '*.pdf' \
  --exclude 'dist/' --exclude 'build/' \
  "$SRC"/ "$TMP"/

cd "$TMP"
git init -q
git remote add origin github.com/w3con/sifty
git add -A

echo "== publishable tree =="
echo "tracked files: $(git ls-files | wc -l | tr -d ' ')"
echo "node_modules tracked: $(git ls-files | grep -c node_modules || true)"
echo "origin: $(git config --get remote.origin.url)"
echo ""
echo "== full-tree leak sweep =="
python3 "$TMP/scripts/leakcheck/tree-check" "$TMP"
