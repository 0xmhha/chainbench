#!/usr/bin/env bash
# tests/unit/tests/cmd-config.sh - Test chainbench config subcommand
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/assert.sh"

REAL_CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

TMPDIR_ROOT="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_ROOT"' EXIT

# Create isolated chainbench directory
FAKE_CB="${TMPDIR_ROOT}/cb"
mkdir -p "${FAKE_CB}/state" "${FAKE_CB}/lib"
cp "${REAL_CHAINBENCH_DIR}/lib/cmd_config.sh" "${FAKE_CB}/lib/"
cp "${REAL_CHAINBENCH_DIR}/lib/common.sh" "${FAKE_CB}/lib/"

OVERLAY="${FAKE_CB}/state/local-config.yaml"

# Helper: run config via chainbench-like dispatch
run_config() {
  CHAINBENCH_DIR="${FAKE_CB}" CHAINBENCH_QUIET=1 \
    bash -c 'export CHAINBENCH_DIR CHAINBENCH_QUIET; source "$CHAINBENCH_DIR/lib/common.sh"; unset _CB_CMD_CONFIG_SH_LOADED; source "$CHAINBENCH_DIR/lib/cmd_config.sh"' \
    _ "$@" 2>/dev/null
}

# ---- Test 1: set creates file -----------------------------------------------
describe "config: set creates file"
run_config set chain.binary_path /opt/gstable
assert_file_exists "$OVERLAY" "overlay file created"

# ---- Test 2: get returns the value ------------------------------------------
describe "config: get returns the value"
result="$(run_config get chain.binary_path)"
assert_eq "$result" "/opt/gstable" "get returns set value"

# ---- Test 3: second set preserves earlier fields ----------------------------
describe "config: second set preserves earlier fields"
run_config set chain.logrot_path /opt/logrot
result1="$(run_config get chain.binary_path)"
result2="$(run_config get chain.logrot_path)"
assert_eq "$result1" "/opt/gstable" "first field preserved"
assert_eq "$result2" "/opt/logrot" "second field set"

# ---- Test 4: unset removes the targeted field --------------------------------
describe "config: unset removes the field"
run_config unset chain.logrot_path
rc=0
run_config get chain.logrot_path || rc=$?
assert_eq "$rc" "1" "get of unset field exits 1"

# ---- Test 5: list prints the whole overlay -----------------------------------
describe "config: list prints the overlay"
result="$(run_config list)"
assert_contains "$result" "binary_path" "list output includes field"

# ---- Test 6: invalid field path -> exit 1 ------------------------------------
describe "config: invalid field path rejected"
rc=0
run_config set "..bad" "value" || rc=$?
assert_eq "$rc" "1" "double-dot field rejected"

# ---- Test 7: get of missing field -> exit 1 ----------------------------------
describe "config: get missing field exits 1"
rc=0
run_config get nonexistent.field || rc=$?
assert_eq "$rc" "1" "missing field exits 1"

# ---- Test 8: JSON array value stored correctly --------------------------------
describe "config: JSON array value stored correctly"
run_config set nodes.extra_flags '["-flag1", "-flag2"]'
result="$(run_config get nodes.extra_flags)"
assert_contains "$result" "flag1" "array value stored"

unit_summary
