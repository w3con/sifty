#!/usr/bin/env bash
# setup.sh — install the Playwright worker's Node.js dependencies and the Chromium browser
# binary. Run once after a fresh clone, or after updating playwright in package.json.
#
# Usage:
#   sifty/workers/playwright/setup.sh
#
# The script changes into its own directory so npm install and npx run from the
# correct location regardless of where the script is invoked from.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

echo "Installing Playwright npm dependencies..."
npm install

echo "Installing Chromium browser binary..."
npx playwright install chromium

echo "Playwright worker ready."
