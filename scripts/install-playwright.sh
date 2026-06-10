#!/usr/bin/env bash
# Install Chromium for playwright-go (used by E2E browser tests).
set -euo pipefail

WITH_DEPS="${PLAYWRIGHT_WITH_DEPS:-0}"
args=(install chromium)

if [[ "$WITH_DEPS" == "1" ]]; then
  args=(install --with-deps chromium)
fi

go run github.com/playwright-community/playwright-go/cmd/playwright@latest "${args[@]}"
