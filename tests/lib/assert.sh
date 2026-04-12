#!/usr/bin/env bash
# tests/lib/assert.sh - Assertion library for chainbench tests
# Usage: source tests/lib/assert.sh

CHAINBENCH_DIR="${CHAINBENCH_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"

# ---------------------------------------------------------------------------
# Internal state (reset by test_start)
# ---------------------------------------------------------------------------

_ASSERT_TEST_NAME=""
_ASSERT_PASS=0
_ASSERT_FAIL=0
_ASSERT_FAILURES=()   # array of failure messages
_ASSERT_START_TS=0
_OBSERVED_KEYS=()     # observe() keys
_OBSERVED_VALUES=()   # observe() values

# Color codes (consistent with lib/common.sh)
_ASSERT_RED='\033[0;31m'
_ASSERT_GREEN='\033[0;32m'
_ASSERT_YELLOW='\033[0;33m'
_ASSERT_CYAN='\033[0;36m'
_ASSERT_RESET='\033[0m'

# ---------------------------------------------------------------------------
# Test lifecycle
# ---------------------------------------------------------------------------

# test_start <name>
# Initialises counters for a new test run. Call once at the top of each test.
test_start() {
  _ASSERT_TEST_NAME="${1:?test_start: name required}"
  _ASSERT_PASS=0
  _ASSERT_FAIL=0
  _ASSERT_FAILURES=()
  _OBSERVED_KEYS=()
  _OBSERVED_VALUES=()
  _ASSERT_START_TS=$(date +%s)
  printf "${_ASSERT_CYAN}[TEST]${_ASSERT_RESET}  Starting: %s\n" "$_ASSERT_TEST_NAME" >&2
  if [[ "${CB_FORMAT:-text}" == "jsonl" ]]; then
    python3 -c "import json,sys; print(json.dumps({'event':'test_start','name':sys.argv[1],'ts':sys.argv[2]}))" \
      "$_ASSERT_TEST_NAME" "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  fi
}

# _assert_pass <msg>
# Records a passing assertion.
_assert_pass() {
  local msg="${1:-}"
  _ASSERT_PASS=$(( _ASSERT_PASS + 1 ))
  printf "${_ASSERT_GREEN}  [PASS]${_ASSERT_RESET} %s\n" "$msg" >&2
  if [[ "${CB_FORMAT:-text}" == "jsonl" ]]; then
    python3 -c "import json,sys; print(json.dumps({'event':'assert_pass','msg':sys.argv[1]}))" "$msg"
  fi
}

# _assert_fail <msg>
# Records a failing assertion.
_assert_fail() {
  local msg="${1:-}"
  _ASSERT_FAIL=$(( _ASSERT_FAIL + 1 ))
  _ASSERT_FAILURES+=("$msg")
  printf "${_ASSERT_RED}  [FAIL]${_ASSERT_RESET} %s\n" "$msg" >&2
  if [[ "${CB_FORMAT:-text}" == "jsonl" ]]; then
    python3 -c "import json,sys; print(json.dumps({'event':'assert_fail','msg':sys.argv[1]}))" "$msg"
  fi
}

# ---------------------------------------------------------------------------
# Observables
# ---------------------------------------------------------------------------

# observe <key> <value>
# Collect a key-value observation during test execution.
# Observations are serialized into the result JSON "observed" field.
observe() {
  local key="${1:?observe: key required}"
  local value="${2:-}"
  _OBSERVED_KEYS+=("$key")
  _OBSERVED_VALUES+=("$value")
  if [[ "${CB_FORMAT:-text}" == "jsonl" ]]; then
    python3 -c "import json,sys; print(json.dumps({'event':'observe','key':sys.argv[1],'value':sys.argv[2]}))" "$key" "$value"
  fi
}

# ---------------------------------------------------------------------------
# Assertion functions
# ---------------------------------------------------------------------------

# assert_eq <actual> <expected> [msg]
# Passes when actual == expected (string comparison).
assert_eq() {
  local actual="${1}" expected="${2}" msg="${3:-assert_eq}"
  if [[ "$actual" == "$expected" ]]; then
    _assert_pass "${msg}: '${actual}' == '${expected}'"
  else
    _assert_fail "${msg}: expected '${expected}', got '${actual}'"
  fi
}

# assert_gt <actual> <expected> [msg]
# Passes when actual > expected (integer comparison).
# Uses python3 for arbitrary-precision integer comparison. bash -gt is limited
# to int64 and silently produces wrong results on values like 10^27 wei.
assert_gt() {
  local actual="${1}" expected="${2}" msg="${3:-assert_gt}"
  if python3 -c "import sys; sys.exit(0 if int('$actual') > int('$expected') else 1)" 2>/dev/null; then
    _assert_pass "${msg}: ${actual} > ${expected}"
  else
    _assert_fail "${msg}: expected ${actual} > ${expected}"
  fi
}

# assert_ge <actual> <expected> [msg]
# Passes when actual >= expected (integer comparison).
# See assert_gt note about python3 vs bash int64.
assert_ge() {
  local actual="${1}" expected="${2}" msg="${3:-assert_ge}"
  if python3 -c "import sys; sys.exit(0 if int('$actual') >= int('$expected') else 1)" 2>/dev/null; then
    _assert_pass "${msg}: ${actual} >= ${expected}"
  else
    _assert_fail "${msg}: expected ${actual} >= ${expected}"
  fi
}

# assert_true <value> [msg]
# Passes when value is non-empty and not "false", "0", or "no".
assert_true() {
  local value="${1}" msg="${2:-assert_true}"
  case "${value,,}" in
    ""|false|0|no)
      _assert_fail "${msg}: expected truthy, got '${value}'"
      ;;
    *)
      _assert_pass "${msg}: '${value}' is truthy"
      ;;
  esac
}

# assert_contains <haystack> <needle> [msg]
# Passes when needle is a substring of haystack.
assert_contains() {
  local haystack="${1}" needle="${2}" msg="${3:-assert_contains}"
  if [[ "$haystack" == *"$needle"* ]]; then
    _assert_pass "${msg}: found '${needle}'"
  else
    _assert_fail "${msg}: '${needle}' not found in '${haystack}'"
  fi
}

# assert_not_empty <value> [msg]
# Passes when value is a non-empty string.
assert_not_empty() {
  local value="${1}" msg="${2:-assert_not_empty}"
  if [[ -n "$value" ]]; then
    _assert_pass "${msg}: value is non-empty"
  else
    _assert_fail "${msg}: value is empty"
  fi
}

# ---------------------------------------------------------------------------
# Result reporting
# ---------------------------------------------------------------------------

# test_result
# Prints a summary and writes a JSON result file to state/results/.
# Returns 0 if all assertions passed, 1 if any failed.
test_result() {
  local end_ts
  end_ts=$(date +%s)
  local duration=$(( end_ts - _ASSERT_START_TS ))
  local total=$(( _ASSERT_PASS + _ASSERT_FAIL ))
  local status="passed"
  local exit_code=0

  if [[ "$_ASSERT_FAIL" -gt 0 ]]; then
    status="failed"
    exit_code=1
  fi

  # Print summary
  printf "\n${_ASSERT_CYAN}[TEST]${_ASSERT_RESET}  Result: %s — %s\n" \
    "$_ASSERT_TEST_NAME" "$status" >&2
  printf "         Total: %d | Pass: %d | Fail: %d | Duration: %ds\n" \
    "$total" "$_ASSERT_PASS" "$_ASSERT_FAIL" "$duration" >&2

  if [[ "$_ASSERT_FAIL" -gt 0 ]]; then
    printf "${_ASSERT_RED}  Failures:${_ASSERT_RESET}\n" >&2
    local f
    for f in "${_ASSERT_FAILURES[@]}"; do
      printf "    - %s\n" "$f" >&2
    done
  fi

  # Ensure results directory exists
  local results_dir="${CHAINBENCH_DIR}/state/results"
  mkdir -p "$results_dir"

  # Build a safe filename from the test name
  local safe_name
  safe_name=$(printf '%s' "$_ASSERT_TEST_NAME" | tr -cs '[:alnum:]-_' '_' | tr '[:upper:]' '[:lower:]')
  local ts
  ts=$(date +%Y%m%d_%H%M%S)
  local result_file="${results_dir}/${safe_name}_${ts}.json"

  # Serialise failures array to a JSON array string
  local failures_json
  failures_json=$(python3 -c "
import json, sys
failures = sys.argv[1:]
print(json.dumps(failures))
" "${_ASSERT_FAILURES[@]+"${_ASSERT_FAILURES[@]}"}")

  # Serialise observed key-value pairs to a JSON object string
  local observed_json="{}"
  if [[ ${#_OBSERVED_KEYS[@]} -gt 0 ]]; then
    observed_json=$(python3 -c "
import json, sys
keys = sys.argv[1].split('\x1f') if sys.argv[1] else []
vals = sys.argv[2].split('\x1f') if sys.argv[2] else []
obs = {}
for k, v in zip(keys, vals):
    if k:
        obs[k] = v
print(json.dumps(obs))
" "$(printf '%s\x1f' "${_OBSERVED_KEYS[@]}")" "$(printf '%s\x1f' "${_OBSERVED_VALUES[@]}")")
  fi

  python3 -c "
import json, sys

data = {
    'test':      sys.argv[1],
    'status':    sys.argv[2],
    'pass':      int(sys.argv[3]),
    'fail':      int(sys.argv[4]),
    'total':     int(sys.argv[5]),
    'duration':  int(sys.argv[6]),
    'timestamp': sys.argv[7],
    'failures':  json.loads(sys.argv[8]),
    'observed':  json.loads(sys.argv[9]),
}
print(json.dumps(data, indent=2))
" \
    "$_ASSERT_TEST_NAME" \
    "$status" \
    "$_ASSERT_PASS" \
    "$_ASSERT_FAIL" \
    "$total" \
    "$duration" \
    "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    "$failures_json" \
    "$observed_json" \
    > "$result_file"

  printf "         Result file: %s\n" "$result_file" >&2

  # JSONL test_end event
  if [[ "${CB_FORMAT:-text}" == "jsonl" ]]; then
    python3 -c "import json,sys; print(json.dumps({'event':'test_end','name':sys.argv[1],'status':sys.argv[2],'duration':int(sys.argv[3]),'pass':int(sys.argv[4]),'fail':int(sys.argv[5])}))" \
      "$_ASSERT_TEST_NAME" "$status" "$duration" "$_ASSERT_PASS" "$_ASSERT_FAIL"
  fi

  return "$exit_code"
}
