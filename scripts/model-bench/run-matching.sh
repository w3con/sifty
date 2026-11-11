#!/usr/bin/env bash
# Re-run the Sifty matching pass of ONE model over every vacancy that already has a verdict, and save a
# copy of the store with the result.
#
#   scripts/model-bench/run-matching.sh --data-dir DIR --provider NAME --out after.sqlite [--allow-paid] [--timeout-min 120]
#
# DIR       a COPY of your Sifty data directory (it is never modified; the script works on a temporary copy)
# NAME      a provider defined in DIR/config/providers.yaml. The matching pass sends the profile facts closest to
#           each requirement to that provider; which provider that is, is your choice.
# --allow-paid  needed for a provider that is not local and not marked `free: true` (a paid API or a
#           subscription): without it Sifty refuses to run a bulk pass on it so that a backlog is never billed by surprise.
# --out     where to write the copy of the store. Use it with `report.py matching`.
#
# How it works: the temporary copy gets `stages.matching: NAME`, web collection is switched off, and a
# temporary Sifty server runs on a free port. The server's background worker matches again every vacancy
# whose verdicts were made by another model (the model name is part of the matcher version) and the script
# waits until all of them are done, then stops the server and saves the store.
# Run it once per model; run it twice for the same model to measure repeatability (the second run needs a
# fresh copy of DIR, otherwise there is nothing left to match).
set -euo pipefail

data_dir="" provider="" out="" timeout_min=240 allow_paid=0
while [ $# -gt 0 ]; do
  case "$1" in
    --data-dir) data_dir="$2"; shift 2 ;;
    --provider) provider="$2"; shift 2 ;;
    --out) out="$2"; shift 2 ;;
    --allow-paid) allow_paid=1; shift ;;
    --timeout-min) timeout_min="$2"; shift 2 ;;
    -h|--help) sed -n 2,20p "$0"; exit 0 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done
for v in data_dir provider out; do
  [ -n "${!v}" ] || { echo "missing --${v//_/-}" >&2; exit 2; }
done
command -v sqlite3 >/dev/null || { echo "sqlite3 is required" >&2; exit 2; }
[ -f "$data_dir/store.sqlite" ] && [ -d "$data_dir/config" ] || { echo "$data_dir is not a Sifty data directory" >&2; exit 2; }
grep -q "^  $provider:" "$data_dir/config/providers.yaml" || { echo "provider '$provider' is not defined in $data_dir/config/providers.yaml" >&2; exit 2; }

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
work="$(mktemp -d)"
server_pid=""
cleanup() { [ -n "$server_pid" ] && kill "$server_pid" 2>/dev/null || true; rm -rf "$work"; }
trap cleanup EXIT

# Copy everything except the store, then take a consistent copy of the store.
for entry in "$data_dir"/*; do
  [ "$(basename "$entry")" = "store.sqlite" ] || cp -R "$entry" "$work/"
done
sqlite3 "$data_dir/store.sqlite" ".backup $work/store.sqlite"

python3 - "$work/config" "$provider" "$allow_paid" <<'PY'
import re, sys
cfg, name, allow_paid = sys.argv[1:4]
p = cfg + "/providers.yaml"
t = open(p).read()
t = re.sub(r"(?m)^  matching:.*\n", "", t)
t, n = re.subn(r"(?m)^(  extraction:.*)$", lambda m: m.group(1) + "\n  matching: " + name, t, count=1)
if n != 1:
    sys.exit("providers.yaml has no stages.extraction line")
open(p, "w").write(t)
r = cfg + "/runtime.yaml"
t = open(r).read()
t = re.sub(r"(?m)^collection_disabled:.*\n", "", t)
t = re.sub(r"(?m)^(version:.*)$", lambda m: m.group(1) + "\ncollection_disabled: true", t, count=1)
t = re.sub(r"(?m)^(\s*)daily_limit:.*\n", "", t)
t = re.sub(r"(?m)^(\s*)paused:.*$", lambda m: m.group(0) + "\n" + m.group(1) + "daily_limit: 1", t, count=1)
if allow_paid == "1":
    t = re.sub(r"(?m)^\s*allow_paid_bulk_provider:.*\n", "", t)
    t = re.sub(r"(?m)^(\s*)paused:.*$", lambda m: m.group(0) + "\n" + m.group(1) + "allow_paid_bulk_provider: true", t, count=1)
open(r, "w").write(t)
PY

total="$(sqlite3 "$work/store.sqlite" 'select count(distinct job_id) from match_requirement_matches')"
old_versions="$(sqlite3 "$work/store.sqlite" "select quote(matcher_version) from match_requirement_matches group by 1" | paste -sd, -)"
[ "$total" -gt 0 ] || { echo "the store has no matched vacancies to re-match" >&2; exit 2; }
echo "vacancies to match: $total"

(cd "$root/backend" && go build -o "$work/sifty" ./cmd/sifty)
port="$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()')"
dist_args=()
[ -d "$root/frontend/dist" ] && dist_args=(--dist "$root/frontend/dist")
"$work/sifty" serve --data-dir "$work" "${dist_args[@]}" --workers-dir "$root/backend/workers" --port "$port" >"$work/server.log" 2>&1 &
server_pid=$!

deadline=$(( $(date +%s) + timeout_min * 60 ))
while :; do
  done_count="$(sqlite3 "$work/store.sqlite" "select count(distinct job_id) from match_requirement_matches where matcher_version not in ($old_versions)" 2>/dev/null || echo 0)"
  printf '\r%s of %s vacancies matched' "$done_count" "$total"
  [ "$done_count" -ge "$total" ] && break
  kill -0 "$server_pid" 2>/dev/null || { echo; echo "the server stopped; see its log:"; tail -20 "$work/server.log"; exit 1; }
  [ "$(date +%s)" -lt "$deadline" ] || { echo; echo "timed out after $timeout_min minutes" >&2; exit 1; }
  sleep 10
done
echo
sleep 5   # let the worker store the last score
kill "$server_pid"; wait "$server_pid" 2>/dev/null || true; server_pid=""
mkdir -p "$(dirname "$out")"
sqlite3 "$work/store.sqlite" ".backup $out"
echo "saved $out"
