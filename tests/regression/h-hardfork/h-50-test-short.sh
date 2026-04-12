#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-5-1-02
# name: Unit tests pass (make test-short)
# category: regression/h-hardfork
# tags: [hardfork, build, test]
# estimated_seconds: 300
# slow: true
# preconditions:
#   chain_running: false
# depends_on: []
# ---end-meta---
# TC-5-1-02 — make test-short passes
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-50-test-short"

GOSTABLENET_DIR="${GOSTABLENET_DIR:-}"
if [[ -z "$GOSTABLENET_DIR" ]]; then
  GOSTABLENET_DIR="$(cd "$CHAINBENCH_DIR/.." && pwd)/go-stablenet"
fi

if [[ ! -d "$GOSTABLENET_DIR" ]]; then
  printf '[SKIP]  go-stablenet not found\n' >&2
  observe "skip_reason" "directory not found"
  test_result
  exit 0
fi

printf '[INFO]  Running make test-short in %s\n' "$GOSTABLENET_DIR" >&2
if make -C "$GOSTABLENET_DIR" test-short 2>&1 | tail -10; then
  _assert_pass "make test-short passed"
else
  _assert_fail "make test-short failed"
fi

test_result
