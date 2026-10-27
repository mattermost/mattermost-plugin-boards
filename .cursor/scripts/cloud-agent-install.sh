#!/usr/bin/env bash
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$ROOT"

require() {
    command -v "$1" >/dev/null 2>&1 || {
        echo "Missing required tool: $1" >&2
        exit 1
    }
}

require go
require node
require npm
require docker
require skopeo

if [[ "${CLOUD_AGENT_SKIP_GO_MOD:-}" != "1" ]]; then
    go mod download
fi

if [[ "${CLOUD_AGENT_SKIP_BUILD_TOOLS:-}" != "1" ]]; then
    make apply
fi

if [[ "${CLOUD_AGENT_SKIP_WEBAPP_DEPS:-}" != "1" && -f webapp/package-lock.json ]]; then
    npm ci --prefix webapp
fi

echo "Cloud agent install complete."
