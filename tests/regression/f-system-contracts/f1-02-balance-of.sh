#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-F-1-02
# name: NativeCoinAdapter.balanceOf == eth_getBalance
# category: regression/f-system-contracts
# tags: [governance]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/f-system-contracts/f1-02-balance-of
# RT-F-1-02 — NativeCoinAdapter.balanceOf == eth_getBalance
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/f-system-contracts/f1-02-balance-of"
check_env || { test_result; exit 1; }

sel=$(selector "balanceOf(address)")
addr_padded=$(pad_address "$TEST_ACC_A_ADDR" | sed 's/^0x//')
call_data="${sel}${addr_padded}"

# NativeCoinAdapter.balanceOf
nca_bal_hex=$(eth_call_raw 1 "$NATIVE_COIN_ADAPTER" "$call_data")
nca_bal=$(hex_to_dec "$nca_bal_hex")

# eth_getBalance
eth_bal=$(hex_to_dec "$(rpc 1 eth_getBalance "[\"${TEST_ACC_A_ADDR}\", \"latest\"]" | json_get - result)")

assert_eq "$nca_bal" "$eth_bal" "NativeCoinAdapter.balanceOf == eth_getBalance ($eth_bal)"

test_result
