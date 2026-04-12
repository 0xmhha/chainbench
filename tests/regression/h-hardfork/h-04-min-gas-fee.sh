#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-1-3-01
# name: Minimum gas fee enforcement verification
# category: regression/h-hardfork
# tags: [hardfork, boho, gas, fee, basefee]
# estimated_seconds: 15
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/h-hardfork/h-04-min-gas-fee
# TC-1-3-01 — Verify minimum gas fee enforcement
#   1) baseFee >= MIN_BASE_FEE (20 Twei = 20000000000000 wei)
#   2) gasTip from header matches expected value
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-04-min-gas-fee"
check_env || { test_result; exit 1; }

# MIN_BASE_FEE_WEI is defined in common.sh as "20000000000000" (20 Twei)

# --- 1) Get baseFee from latest block ---
base_fee=$(get_base_fee "1")
observe "base_fee_wei" "$base_fee"
printf '[INFO]  current baseFee = %s wei\n' "$base_fee" >&2
printf '[INFO]  MIN_BASE_FEE    = %s wei\n' "$MIN_BASE_FEE_WEI" >&2

assert_not_empty "$base_fee" "baseFee is present in block header"
assert_ge "$base_fee" "$MIN_BASE_FEE_WEI" "baseFee >= MIN_BASE_FEE (20 Twei)"

# --- 2) Get gasTip from WbftExtraInfo header ---
gas_tip=$(get_header_gas_tip "1")
observe "header_gas_tip_wei" "$gas_tip"
printf '[INFO]  header gasTip = %s wei\n' "$gas_tip" >&2

assert_not_empty "$gas_tip" "gasTip is present in WbftExtraInfo"
# gasTip should be a non-negative value
assert_ge "$gas_tip" "0" "gasTip >= 0"

# --- 3) Verify via eth_maxPriorityFeePerGas API ---
max_priority_resp=$(rpc "1" "eth_maxPriorityFeePerGas" "[]")
max_priority_hex=$(json_get "$max_priority_resp" "result")
max_priority_dec=$(hex_to_dec "$max_priority_hex")
observe "eth_maxPriorityFeePerGas" "$max_priority_dec"
printf '[INFO]  eth_maxPriorityFeePerGas = %s wei\n' "$max_priority_dec" >&2

assert_not_empty "$max_priority_dec" "eth_maxPriorityFeePerGas returned a value"

# --- 4) Cross-check: header gasTip should match eth_maxPriorityFeePerGas ---
# Both should reflect the same minimum tip from the consensus header.
# Allow them to differ slightly due to timing (block may advance between calls).
assert_eq "$gas_tip" "$max_priority_dec" "header gasTip matches eth_maxPriorityFeePerGas"

test_result
