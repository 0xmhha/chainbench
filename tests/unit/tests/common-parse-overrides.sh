#!/usr/bin/env bash
# tests/unit/tests/common-parse-overrides.sh - Test _cb_parse_runtime_overrides
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/assert.sh"

CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
export CHAINBENCH_DIR
export CHAINBENCH_QUIET=1
source "${CHAINBENCH_DIR}/lib/common.sh"

# Cleanup env before each test
_reset_env() {
  unset CHAINBENCH_BINARY_PATH 2>/dev/null || true
  unset CHAINBENCH_LOGROT_PATH 2>/dev/null || true
}

# ---- Test 1: --binary-path /x init -> env + remaining ---------------------
describe "parse_overrides: --binary-path space-separated"
_reset_env
_CB_REMAINING=()
_cb_parse_runtime_overrides _CB_REMAINING --binary-path /opt/gstable init
assert_eq "${CHAINBENCH_BINARY_PATH}" "/opt/gstable" "BINARY_PATH set"
assert_eq "${#_CB_REMAINING[@]}" "1" "one remaining arg"
assert_eq "${_CB_REMAINING[0]}" "init" "remaining arg is init"

# ---- Test 2: --binary-path=/x init -> same --------------------------------
describe "parse_overrides: --binary-path= equals syntax"
_reset_env
_CB_REMAINING=()
_cb_parse_runtime_overrides _CB_REMAINING --binary-path=/opt/gstable init
assert_eq "${CHAINBENCH_BINARY_PATH}" "/opt/gstable" "BINARY_PATH set via ="

# ---- Test 3: init --binary-path /x -> same --------------------------------
describe "parse_overrides: flag after subcommand"
_reset_env
_CB_REMAINING=()
_cb_parse_runtime_overrides _CB_REMAINING init --binary-path /opt/gstable
assert_eq "${CHAINBENCH_BINARY_PATH}" "/opt/gstable" "BINARY_PATH set after subcmd"
assert_eq "${_CB_REMAINING[0]}" "init" "remaining starts with init"

# ---- Test 4: --logrot-path /y --binary-path /x -> both exported -----------
describe "parse_overrides: both logrot and binary paths"
_reset_env
_CB_REMAINING=()
_cb_parse_runtime_overrides _CB_REMAINING --logrot-path /opt/logrot --binary-path /opt/gstable
assert_eq "${CHAINBENCH_BINARY_PATH}" "/opt/gstable" "BINARY_PATH set"
assert_eq "${CHAINBENCH_LOGROT_PATH}" "/opt/logrot" "LOGROT_PATH set"

# ---- Test 5: Unknown flags pass through -----------------------------------
describe "parse_overrides: unknown flags pass through"
_reset_env
_CB_REMAINING=()
_cb_parse_runtime_overrides _CB_REMAINING --unknown-flag value init
assert_eq "${#_CB_REMAINING[@]}" "3" "three remaining args"
assert_eq "${_CB_REMAINING[0]}" "--unknown-flag" "unknown flag preserved"

# ---- Test 6: Missing value -> exit 2 --------------------------------------
describe "parse_overrides: missing value exits with error"
_reset_env
rc=0
( _CB_REMAINING=(); _cb_parse_runtime_overrides _CB_REMAINING --binary-path 2>/dev/null ) || rc=$?
assert_eq "$rc" "2" "exits with code 2 on missing value"

unit_summary
