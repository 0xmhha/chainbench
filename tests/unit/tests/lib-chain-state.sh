#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/assert.sh"
export CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
source "${CHAINBENCH_DIR}/tests/lib/chain_state.sh"

describe "chain_state: functions are defined"
assert_exit_code 0 type cb_get_balance
assert_exit_code 0 type cb_get_nonce
assert_exit_code 0 type cb_is_blacklisted
assert_exit_code 0 type cb_is_authorized
assert_exit_code 0 type cb_wait_for_block
assert_exit_code 0 type cb_check_hardfork_active

unit_summary
