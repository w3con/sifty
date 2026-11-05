#!/usr/bin/env bash
# Step 2.2 — the binary resolves its data directory from config or environment and writes answers
# and feedback there, with no FCTO naming, no git-root resolution, no kb/ writes, no ai/shared.
set -euo pipefail
ROOT="$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)"
cd "$ROOT/sifty"
BIN="$(mktemp -d)/sifty"
go build -o "$BIN" ./cmd/sifty

echo "== prohibitions absent from Go code =="
BAD=0
for pat in 'FCTO_STATE_DIR' 'FCTO-Catalogue' 'FCTO_RECORD' 'rev-parse' 'ai/shared'; do
  if grep -rn --include='*.go' "$pat" cmd internal; then echo "FOUND forbidden: $pat"; BAD=1; fi
done
# No writes into a kb/ path from Go code.
if grep -rn --include='*.go' '"kb/' cmd internal; then echo "FOUND kb/ path"; BAD=1; fi
[ "$BAD" -eq 0 ] && echo "PASS: none of the forbidden forms present"
[ "$BAD" -eq 0 ] || { echo "FAIL: forbidden form present"; exit 1; }

echo "== environment-based data directory =="
D1="$(mktemp -d)"
SIFTY_DATA_DIR="$D1" "$BIN" selfcheck
test -f "$D1"/answers/*.md && test -f "$D1"/feedback/*.md && echo "PASS-env: answer and feedback written under \$SIFTY_DATA_DIR"

echo "== flag-based data directory =="
D2="$(mktemp -d)"
"$BIN" selfcheck --data-dir "$D2"
test -f "$D2"/answers/*.md && test -f "$D2"/feedback/*.md && echo "PASS-flag: answer and feedback written under --data-dir"

echo "== config read from the data directory =="
D3="$(mktemp -d)"
mkdir -p "$D3/config"
cat > "$D3/config/search.yaml" <<'YML'
version: 1
sites:
  - name: Example Board
    url: https://example.com/jobs
    enabled: true
search:
  titles: ["CTO"]
  keywords: ["golang"]
preferences: {}
YML
cat > "$D3/config/runtime.yaml" <<'YML'
version: 1
port: 4317
max_pages: 4
max_jobs_per_source: 100
request_timeout_ms: 20000
max_assessments_per_run: 5
daily_hour: 9
daily_minute: 0
YML
cat > "$D3/config/providers.yaml" <<'YML'
version: 1
stages:
  assessment: local
  extraction: local
providers:
  local:
    type: compatible
    model: test-model
    endpoint: http://127.0.0.1:1234/v1
    timeout_ms: 120000
    context_chars: 24000
YML
cat > "$D3/config/candidate.yaml" <<'YML'
version: 1
files: ["cv.pdf"]
profiles: []
cache_hours: 168
source_roles: {}
review_notes: []
YML
"$BIN" selfcheck --data-dir "$D3" | grep -q "config loaded: 1 site(s)" && echo "PASS-config: config read from data directory"
echo "ALL PASS"
