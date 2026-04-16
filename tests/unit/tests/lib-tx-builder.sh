#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/assert.sh"
export CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
source "${CHAINBENCH_DIR}/tests/lib/tx_builder.sh"

describe "tx_builder: tier 1 functions defined (cast)"
assert_exit_code 0 type cb_send_tx
assert_exit_code 0 type cb_send_legacy_tx
assert_exit_code 0 type cb_sign_tx
assert_exit_code 0 type cb_send_raw
assert_exit_code 0 type cb_wait_receipt

describe "tx_builder: tier 2 functions defined (custom tx)"
assert_exit_code 0 type cb_send_fee_delegate
assert_exit_code 0 type cb_sign_fee_delegate

describe "tx_builder: tier 3 functions defined (invalid tx)"
assert_exit_code 0 type cb_send_invalid_fee_delegate
assert_exit_code 0 type cb_send_invalid_tx

describe "tx_builder: fee_delegate.py found"
fd_path=$(_cb_fee_delegate_py)
assert_file_exists "$fd_path" "fee_delegate.py exists"

describe "tx_builder: default gas tip matches Anzeon policy"
assert_eq "$CB_DEFAULT_GAS_TIP" "27600000000000" "27.6 Gwei"

unit_summary
