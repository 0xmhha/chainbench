#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-5-1-01
# name: Full binary compilation succeeds
# category: regression/h-hardfork
# tags: [hardfork, build, compilation]
# estimated_seconds: 120
# preconditions:
#   chain_running: false
# depends_on: []
# ---end-meta---
# TC-5-1-01 — make all succeeds and produces gstable binary
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-49-build-all"

# Find go-stablenet directory
GOSTABLENET_DIR="${GOSTABLENET_DIR:-}"
if [[ -z "$GOSTABLENET_DIR" ]]; then
  GOSTABLENET_DIR="$(cd "$CHAINBENCH_DIR/.." && pwd)/go-stablenet"
fi

if [[ ! -d "$GOSTABLENET_DIR" ]]; then
  printf '[SKIP]  go-stablenet directory not found at %s\n' "$GOSTABLENET_DIR" >&2
  printf '[SKIP]  Set GOSTABLENET_DIR env var to the correct path\n' >&2
  observe "skip_reason" "go-stablenet directory not found"
  test_result
  exit 0
fi

observe "gostablenet_dir" "$GOSTABLENET_DIR"

# Build
printf '[INFO]  Running make all in %s\n' "$GOSTABLENET_DIR" >&2
if make -C "$GOSTABLENET_DIR" all 2>&1 | tail -5; then
  _assert_pass "make all succeeded"
else
  _assert_fail "make all failed"
fi

# Verify binary exists
if [[ -f "$GOSTABLENET_DIR/build/bin/gstable" ]]; then
  _assert_pass "gstable binary exists"
  observe "binary_path" "$GOSTABLENET_DIR/build/bin/gstable"
else
  _assert_fail "gstable binary not found at build/bin/gstable"
fi

test_result
