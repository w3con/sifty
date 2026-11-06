#!/usr/bin/env bash
# Step 4.2 — form inspection runs through the TypeScript Playwright worker invoked by the Go core:
# the worker returns the questionnaire fields for a fixture form, and no browser-automation code or
# dependency lives in Go.
set -euo pipefail
ROOT="$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)"
cd "$ROOT/sifty"

echo "== Playwright worker present =="
test -f workers/playwright/index.mjs || { echo "FAIL: workers/playwright/index.mjs missing"; exit 1; }
echo "PASS: workers/playwright/index.mjs present"

echo "== no browser-automation dependency or code in Go =="
BAD=0
# No Go browser-automation library in the module graph.
for mod in 'playwright-community/playwright-go' 'chromedp' 'go-rod/rod' 'tebeka/selenium'; do
  if grep -n "$mod" go.mod go.sum 2>/dev/null; then echo "FOUND forbidden module: $mod"; BAD=1; fi
done
# No import of a browser-automation package from any Go file.
if grep -rn --include='*.go' -E '"[^"]*(playwright-go|chromedp|go-rod/rod|/rod"|tebeka/selenium)[^"]*"' cmd internal; then
  echo "FOUND forbidden import"; BAD=1
fi
[ "$BAD" -eq 0 ] && echo "PASS: no browser-automation dependency or import in Go"
[ "$BAD" -eq 0 ] || { echo "FAIL: browser-automation present in Go"; exit 1; }

echo "== form fields returned via the worker =="
go test ./internal/inspect/ -v -run TestInspectFormViaWorker
