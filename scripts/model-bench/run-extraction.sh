#!/usr/bin/env bash
# Run the Sifty extraction pass of ONE model over a list of vacancies and save the result as JSON.
#
#   scripts/model-bench/run-extraction.sh --data-dir DIR --provider NAME --jobs 84,2616,2520 --out run.json [--allow-paid]
#
# DIR       a COPY of your Sifty data directory (the vacancies are read from its store.sqlite)
# NAME      a provider defined in DIR/config/providers.yaml (for example one you just added)
# --jobs    comma-separated job ids from the store
# --allow-paid  needed for a provider that is not local and not marked `free: true` (a paid API or a
#               subscription CLI). Vacancy texts are sent to that provider; your profile is NOT used by
#               this pass.
#
# The script never touches DIR: it works on a temporary copy of the configuration and the store, switches
# the `extraction` stage to NAME there and runs the extraction pass (Go, from backend/) once per vacancy.
# Compare several runs with report.py.
set -euo pipefail

data_dir="" provider="" jobs="" out="" allow_paid=0
while [ $# -gt 0 ]; do
  case "$1" in
    --data-dir) data_dir="$2"; shift 2 ;;
    --provider) provider="$2"; shift 2 ;;
    --jobs) jobs="$2"; shift 2 ;;
    --out) out="$2"; shift 2 ;;
    --allow-paid) allow_paid=1; shift ;;
    -h|--help) sed -n 2,17p "$0"; exit 0 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done
for v in data_dir provider jobs out; do
  [ -n "${!v}" ] || { echo "missing --${v//_/-}" >&2; exit 2; }
done
[ -f "$data_dir/store.sqlite" ] && [ -d "$data_dir/config" ] || { echo "$data_dir is not a Sifty data directory" >&2; exit 2; }
grep -q "^  $provider:" "$data_dir/config/providers.yaml" || { echo "provider '$provider' is not defined in $data_dir/config/providers.yaml" >&2; exit 2; }

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
cp -R "$data_dir/config" "$work/config"
if command -v sqlite3 >/dev/null; then sqlite3 "$data_dir/store.sqlite" ".backup $work/store.sqlite"; else cp "$data_dir/store.sqlite" "$work/store.sqlite"; fi
# Select the provider for the extraction stage of the temporary copy only.
python3 - "$work/config/providers.yaml" "$provider" <<'PY'
import re, sys
path, name = sys.argv[1:3]
text = open(path).read()
text, n = re.subn(r"(?m)^(  extraction:).*$", r"\1 " + name, text, count=1)
if n != 1:
    sys.exit("providers.yaml has no stages.extraction line")
open(path, "w").write(text)
PY
if [ "$allow_paid" = 1 ]; then
  python3 - "$work/config/runtime.yaml" <<'PY'
import re, sys
path = sys.argv[1]
text = open(path).read()
text = re.sub(r"(?m)^\s*allow_paid_bulk_provider:.*\n", "", text)
text = re.sub(r"(?m)^(\s*)paused:.*$", lambda m: m.group(0) + "\n" + m.group(1) + "allow_paid_bulk_provider: true", text, count=1)
open(path, "w").write(text)
PY
fi
mkdir -p "$(dirname "$out")"
out_abs="$(cd "$(dirname "$out")" && pwd)/$(basename "$out")"
cd "$root/backend"
SIFTY_PROBE_DATA_DIR="$work" SIFTY_PROBE_JOB_IDS="$jobs" SIFTY_REGRESSION_OUT="$out_abs" \
  go test ./internal/vacancyai -run TestLiveExtractionRegression -count=1 -timeout 60m
echo "saved $out_abs"
