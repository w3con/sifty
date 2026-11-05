#!/usr/bin/env bash
# Step 1.3 — leak-check hooks via relative core.hooksPath catch a seeded secret at commit time,
# pass a clean commit, and catch a --no-verify'd secret at push time.
set -uo pipefail
ROOT="$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)"
SRC="$ROOT/sifty"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
cp -R "$SRC"/. "$WORK"/
cd "$WORK"
git init -q
git config user.email t@example.com
git config user.name tester
git remote add origin git@github.com:w3con/sifty.git
bash scripts/setup
echo "hooksPath=$(git config --get core.hooksPath)"
SECRET="AKIA$(printf 'ABCDEFGHIJKLMNOP')"

echo "== Test A: commit carrying a seeded secret is refused =="
printf 'id = "%s"\n' "$SECRET" > leaked.txt
git add leaked.txt
if git commit -q -m "feat: add config value" 2>A.err; then
  echo "FAIL-A: secret commit was accepted"; cat A.err; exit 1
fi
grep -q "cloud access key" A.err && echo "PASS-A: refused (cloud access key named)" || { echo "FAIL-A: refusal did not name the form"; cat A.err; exit 1; }
git reset -q

echo "== Test B: a clean commit passes =="
rm -f leaked.txt
echo "clean source line" > ok.txt
git add ok.txt LICENSE README.md .gitignore scripts .githooks
if git commit -q -m "feat: add clean file"; then
  echo "PASS-B: clean commit accepted"
else
  echo "FAIL-B: clean commit refused"; exit 1
fi

echo "== Test C: a --no-verify secret commit is caught by the pre-push re-scan =="
printf 'token = "%s"\n' "$SECRET" > sneaked.txt
git add sneaked.txt
git commit -q --no-verify -m "feat: sneak value past hooks"
echo "committed with --no-verify: $(git rev-parse --short HEAD)"
LOCAL_SHA="$(git rev-parse HEAD)"
ZERO=$(printf '0%.0s' {1..40})
# Simulate git's pre-push protocol on stdin for a brand-new ref.
if printf 'refs/heads/main %s refs/heads/main %s\n' "$LOCAL_SHA" "$ZERO" | .githooks/pre-push origin git@github.com:w3con/sifty.git 2>C.err; then
  echo "FAIL-C: pre-push accepted a commit carrying a secret"; cat C.err; exit 1
fi
grep -q "cloud access key" C.err && echo "PASS-C: pre-push caught the --no-verify'd secret" || { echo "FAIL-C: pre-push refusal did not name the form"; cat C.err; exit 1; }

echo "ALL PASS"
