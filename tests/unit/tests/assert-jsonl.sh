#!/usr/bin/env bash
# tests/unit/tests/assert-jsonl.sh - Test JSONL event stream output mode
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/assert.sh"

CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
export CHAINBENCH_DIR

TMPDIR_ROOT="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_ROOT"' EXIT

REGRESSION_ASSERT="${CHAINBENCH_DIR}/tests/lib/assert.sh"

# Helper: run scenario, capture stdout (JSONL) separately from stderr
_run_jsonl_scenario() {
  local name="$1"; shift
  local tmpdir="${TMPDIR_ROOT}/${name}"
  mkdir -p "${tmpdir}/state/results"
  CB_FORMAT=jsonl CHAINBENCH_DIR="${tmpdir}" bash -c "
    source '${REGRESSION_ASSERT}'
    $*
  " 2>/dev/null
}

# ---- Test 1: test_start emits JSONL event -----------------------------------
describe "jsonl: test_start emits event"
output="$(_run_jsonl_scenario "start" '
  test_start "test/jsonl-start"
  _assert_pass "dummy"
  test_result
')"
first_line="$(echo "$output" | head -1)"
has_test_start="$(echo "$first_line" | python3 -c "import sys,json; d=json.load(sys.stdin); print('yes' if d.get('event')=='test_start' else 'no')" 2>/dev/null || echo "no")"
assert_eq "$has_test_start" "yes" "first line is test_start event"

# ---- Test 2: assert_pass emits JSONL event ----------------------------------
describe "jsonl: assert events emitted"
output2="$(_run_jsonl_scenario "assert" '
  test_start "test/jsonl-assert"
  assert_eq "a" "a" "eq check"
  test_result
')"
has_pass="$(echo "$output2" | python3 -c "
import sys, json
for line in sys.stdin:
    line = line.strip()
    if not line: continue
    d = json.loads(line)
    if d.get('event') == 'assert_pass':
        print('yes')
        break
else:
    print('no')
" 2>/dev/null || echo "no")"
assert_eq "$has_pass" "yes" "assert_pass event found"

# ---- Test 3: observe emits JSONL event --------------------------------------
describe "jsonl: observe emits event"
output3="$(_run_jsonl_scenario "observe" '
  test_start "test/jsonl-observe"
  observe "key1" "val1"
  _assert_pass "dummy"
  test_result
')"
has_observe="$(echo "$output3" | python3 -c "
import sys, json
for line in sys.stdin:
    line = line.strip()
    if not line: continue
    d = json.loads(line)
    if d.get('event') == 'observe' and d.get('key') == 'key1':
        print('yes')
        break
else:
    print('no')
" 2>/dev/null || echo "no")"
assert_eq "$has_observe" "yes" "observe event found"

# ---- Test 4: test_end emits JSONL event -------------------------------------
describe "jsonl: test_end emitted at result"
last_line="$(echo "$output3" | tail -1)"
has_end="$(echo "$last_line" | python3 -c "import sys,json; d=json.load(sys.stdin); print('yes' if d.get('event')=='test_end' else 'no')" 2>/dev/null || echo "no")"
assert_eq "$has_end" "yes" "last line is test_end event"

# ---- Test 5: text mode produces no stdout -----------------------------------
describe "jsonl: text mode produces no stdout"
output_text="$(
  tmpdir="${TMPDIR_ROOT}/textmode"
  mkdir -p "${tmpdir}/state/results"
  CB_FORMAT=text CHAINBENCH_DIR="${tmpdir}" bash -c "
    source '${REGRESSION_ASSERT}'
    test_start 'test/text-mode'
    observe 'k' 'v'
    _assert_pass 'dummy'
    test_result
  " 2>/dev/null
)"
assert_empty "$output_text" "text mode stdout is empty"

unit_summary
