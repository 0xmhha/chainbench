#!/usr/bin/env bash
# tests/unit/tests/assert-observe.sh - Test observe() API and result JSON observed field
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/assert.sh"

CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
export CHAINBENCH_DIR

TMPDIR_ROOT="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_ROOT"' EXIT

REGRESSION_ASSERT="${CHAINBENCH_DIR}/tests/lib/assert.sh"

# Helper: run a test scenario in isolated subshell, return result JSON
_run_scenario() {
  local scenario_name="$1"
  shift
  local tmpdir="${TMPDIR_ROOT}/${scenario_name}"
  mkdir -p "${tmpdir}/state/results"
  CHAINBENCH_DIR="${tmpdir}" bash -c "
    source '${REGRESSION_ASSERT}'
    $*
  " 2>/dev/null
  cat "${tmpdir}"/state/results/*.json 2>/dev/null
}

# ---- Test 1: observe adds key-value to result JSON --------------------------
describe "observe: adds observed field to result JSON"
result_json="$(_run_scenario "basic" '
  test_start "test/observe-basic"
  observe "block_number" "1234"
  observe "tx_hash" "0xabc"
  _assert_pass "dummy pass"
  test_result
')"
has_observed="$(echo "$result_json" | python3 -c "import sys,json; d=json.load(sys.stdin); print('yes' if 'observed' in d else 'no')")"
assert_eq "$has_observed" "yes" "result JSON contains observed field"

block_val="$(echo "$result_json" | python3 -c "import sys,json; print(json.load(sys.stdin)['observed']['block_number'])")"
assert_eq "$block_val" "1234" "observed.block_number == 1234"

tx_val="$(echo "$result_json" | python3 -c "import sys,json; print(json.load(sys.stdin)['observed']['tx_hash'])")"
assert_eq "$tx_val" "0xabc" "observed.tx_hash == 0xabc"

# ---- Test 2: no observe calls -> observed is empty dict ---------------------
describe "observe: no calls yields empty observed"
result_json2="$(_run_scenario "empty" '
  test_start "test/observe-empty"
  _assert_pass "dummy"
  test_result
')"
observed_keys="$(echo "$result_json2" | python3 -c "import sys,json; print(len(json.load(sys.stdin).get('observed',{})))")"
assert_eq "$observed_keys" "0" "observed is empty dict when no observe calls"

# ---- Test 3: observe with special characters in value -----------------------
describe "observe: special characters in value"
result_json3="$(_run_scenario "special" '
  test_start "test/observe-special"
  observe "msg" "hello world & friends"
  _assert_pass "dummy"
  test_result
')"
msg_val="$(echo "$result_json3" | python3 -c "import sys,json; print(json.load(sys.stdin)['observed']['msg'])")"
assert_contains "$msg_val" "hello" "special chars: contains hello"
assert_contains "$msg_val" "friends" "special chars: contains friends"

unit_summary
