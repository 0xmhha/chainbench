#!/usr/bin/env bash
# tests/unit/tests/cmd-report-format.sh - Pin the report --format contract (D1)
#
# C4 loop-back parsing depends on `chainbench report --format json` emitting
# JSON. The MCP report tool used to pass a bare `--json` flag, which the
# report arg parser silently ignores (unknown option) — leaving the format at
# "text" and breaking the agent's `summary.failed` parse. These tests lock the
# CLI contract so the regression can't reappear: `--format json` yields JSON,
# while a bare `--json` does NOT (it is ignored and falls back to the text
# report). Run against whatever results state exists — the differentiator is
# the output *format*, not the counts, so the test is state-independent.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/assert.sh"

CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
export CHAINBENCH_DIR
export CHAINBENCH_QUIET=1

CB="${CHAINBENCH_DIR}/chainbench.sh"

# ---- Test 1: --format json emits parseable JSON with summary.failed -------
describe "report --format json emits JSON with summary.failed"
json_out="$("${CB}" report --format json 2>/dev/null || true)"
echo "${json_out}" | python3 -c "import json,sys; json.load(sys.stdin)['summary']['failed']" >/dev/null 2>&1
assert_eq "$?" "0" "--format json parses and exposes summary.failed"

# ---- Test 2: bare --json is ignored -> NOT JSON (the D1 trap) -------------
describe "bare --json is ignored (falls back to text, not JSON)"
bare_out="$("${CB}" report --json 2>/dev/null || true)"
rc=0
echo "${bare_out}" | python3 -c "import json,sys; json.load(sys.stdin)" >/dev/null 2>&1 || rc=$?
assert_neq "$rc" "0" "bare --json does not produce JSON"

unit_summary
