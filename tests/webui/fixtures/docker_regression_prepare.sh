#!/usr/bin/env bash
# Resolve and verify the same configured server1 used by the gated Go tests.
# The helper refuses conflicting existing fixtures instead of overwriting them.
set -euo pipefail
root=$(cd "$(dirname "$0")/../../.." && pwd)
cd "$root"
exec go run ./tests/webui/fixtures/dockerprepare
