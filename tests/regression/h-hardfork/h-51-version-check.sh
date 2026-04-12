#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-5-1-03
# name: gstable version output is valid
# category: regression/h-hardfork
# tags: [hardfork, build, version]
# estimated_seconds: 10
# preconditions:
#   chain_running: false
# depends_on: []
# ---end-meta---
# TC-5-1-03 — gstable version returns non-empty output
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-51-version-check"

# Find gstable binary
GSTABLE_BIN="${GOSTABLENET_DIR:-}/build/bin/gstable"
if [[ ! -f "$GSTABLE_BIN" ]]; then
  GSTABLE_BIN="$(cd "$CHAINBENCH_DIR/.." && pwd)/go-stablenet/build/bin/gstable"
fi
if [[ ! -f "$GSTABLE_BIN" ]]; then
  GSTABLE_BIN="$(which gstable 2>/dev/null || true)"
fi

if [[ -z "$GSTABLE_BIN" || ! -f "$GSTABLE_BIN" ]]; then
  printf '[SKIP]  gstable binary not found\n' >&2
  observe "skip_reason" "binary not found"
  test_result
  exit 0
fi

version_output=$("$GSTABLE_BIN" version 2>&1 || true)
observe "version_output" "$version_output"

assert_not_empty "$version_output" "gstable version output is non-empty"

test_result
