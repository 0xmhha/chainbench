#!/usr/bin/env bash
# tests/unit/lib/assert.sh - Lightweight assertion library for unit tests
# Zero external dependencies. Each test file sources this.

_UT_PASS=0
_UT_FAIL=0
_UT_TEST_NAME=""
_UT_FAILURES=()

# Colors (respect NO_COLOR)
if [[ -n "${NO_COLOR:-}" ]] || ! [[ -t 2 ]]; then
  _UT_RED="" _UT_GREEN="" _UT_YELLOW="" _UT_CYAN="" _UT_RESET=""
else
  _UT_RED='\033[0;31m'
  _UT_GREEN='\033[0;32m'
  _UT_YELLOW='\033[0;33m'
  _UT_CYAN='\033[0;36m'
  _UT_RESET='\033[0m'
fi

# describe <test_name>
# Labels the current test group for output.
describe() {
  _UT_TEST_NAME="$1"
  printf "${_UT_CYAN}  ▸ %s${_UT_RESET}\n" "$1" >&2
}

# assert_eq <actual> <expected> [msg]
assert_eq() {
  local actual="$1" expected="$2" msg="${3:-assert_eq}"
  if [[ "$actual" == "$expected" ]]; then
    (( _UT_PASS++ ))
    printf "${_UT_GREEN}    ✓ %s${_UT_RESET}\n" "$msg" >&2
  else
    (( _UT_FAIL++ ))
    _UT_FAILURES+=("${_UT_TEST_NAME}: ${msg} — expected '${expected}', got '${actual}'")
    printf "${_UT_RED}    ✗ %s — expected '%s', got '%s'${_UT_RESET}\n" "$msg" "$expected" "$actual" >&2
  fi
}

# assert_neq <actual> <not_expected> [msg]
assert_neq() {
  local actual="$1" not_expected="$2" msg="${3:-assert_neq}"
  if [[ "$actual" != "$not_expected" ]]; then
    (( _UT_PASS++ ))
    printf "${_UT_GREEN}    ✓ %s${_UT_RESET}\n" "$msg" >&2
  else
    (( _UT_FAIL++ ))
    _UT_FAILURES+=("${_UT_TEST_NAME}: ${msg} — got unwanted '${actual}'")
    printf "${_UT_RED}    ✗ %s — got unwanted '%s'${_UT_RESET}\n" "$msg" "$actual" >&2
  fi
}

# assert_empty <value> [msg]
assert_empty() {
  local value="$1" msg="${2:-assert_empty}"
  if [[ -z "$value" ]]; then
    (( _UT_PASS++ ))
    printf "${_UT_GREEN}    ✓ %s${_UT_RESET}\n" "$msg" >&2
  else
    (( _UT_FAIL++ ))
    _UT_FAILURES+=("${_UT_TEST_NAME}: ${msg} — expected empty, got '${value}'")
    printf "${_UT_RED}    ✗ %s — expected empty, got '%s'${_UT_RESET}\n" "$msg" "$value" >&2
  fi
}

# assert_nonempty <value> [msg]
assert_nonempty() {
  local value="$1" msg="${2:-assert_nonempty}"
  if [[ -n "$value" ]]; then
    (( _UT_PASS++ ))
    printf "${_UT_GREEN}    ✓ %s${_UT_RESET}\n" "$msg" >&2
  else
    (( _UT_FAIL++ ))
    _UT_FAILURES+=("${_UT_TEST_NAME}: ${msg} — expected non-empty")
    printf "${_UT_RED}    ✗ %s — expected non-empty${_UT_RESET}\n" "$msg" >&2
  fi
}

# assert_file_exists <path> [msg]
assert_file_exists() {
  local path="$1" msg="${2:-assert_file_exists}"
  if [[ -f "$path" ]]; then
    (( _UT_PASS++ ))
    printf "${_UT_GREEN}    ✓ %s${_UT_RESET}\n" "$msg" >&2
  else
    (( _UT_FAIL++ ))
    _UT_FAILURES+=("${_UT_TEST_NAME}: ${msg} — file not found: ${path}")
    printf "${_UT_RED}    ✗ %s — file not found: %s${_UT_RESET}\n" "$msg" "$path" >&2
  fi
}

# assert_contains <haystack> <needle> [msg]
assert_contains() {
  local haystack="$1" needle="$2" msg="${3:-assert_contains}"
  if [[ "$haystack" == *"$needle"* ]]; then
    (( _UT_PASS++ ))
    printf "${_UT_GREEN}    ✓ %s${_UT_RESET}\n" "$msg" >&2
  else
    (( _UT_FAIL++ ))
    _UT_FAILURES+=("${_UT_TEST_NAME}: ${msg} — '${needle}' not in output")
    printf "${_UT_RED}    ✗ %s — '%s' not found${_UT_RESET}\n" "$msg" "$needle" >&2
  fi
}

# assert_exit_code <expected_code> <command...>
# Runs the command and checks the exit code.
assert_exit_code() {
  local expected="$1"
  shift
  local actual=0
  "$@" >/dev/null 2>&1 || actual=$?
  if [[ "$actual" -eq "$expected" ]]; then
    (( _UT_PASS++ ))
    printf "${_UT_GREEN}    ✓ exit code %s${_UT_RESET}\n" "$expected" >&2
  else
    (( _UT_FAIL++ ))
    _UT_FAILURES+=("${_UT_TEST_NAME}: expected exit $expected, got $actual — $*")
    printf "${_UT_RED}    ✗ expected exit %s, got %s — %s${_UT_RESET}\n" "$expected" "$actual" "$*" >&2
  fi
}

# unit_summary
# Prints totals and returns non-zero if any test failed.
unit_summary() {
  local total=$(( _UT_PASS + _UT_FAIL ))
  echo "" >&2
  if [[ "$_UT_FAIL" -eq 0 ]]; then
    printf "${_UT_GREEN}All %d assertions passed.${_UT_RESET}\n" "$total" >&2
  else
    printf "${_UT_RED}%d of %d assertions failed:${_UT_RESET}\n" "$_UT_FAIL" "$total" >&2
    for f in "${_UT_FAILURES[@]}"; do
      printf "  - %s\n" "$f" >&2
    done
  fi
  [[ "$_UT_FAIL" -eq 0 ]]
}
