#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-4-6-02
# name: EffectiveGasPrice for regular (non-authorized) account
# category: regression/h-hardfork
# tags: [hardfork, boho, effectiveGasPrice, regular]
# estimated_seconds: 20
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# TC-4-6-02 — Regular accounts pay enforced gasTip
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-43-egp-regular"
check_env || { test_result; exit 1; }

# Send from TEST_ACC_A (regular, not authorized)
tx_hash=$(send_raw_tx 1 "$TEST_ACC_A_PK" "$TEST_ACC_B_ADDR" "1000000000000000000")
assert_not_empty "$tx_hash" "regular tx sent"

receipt=$(wait_tx_receipt_full 1 "$tx_hash" 30)
assert_not_empty "$receipt" "receipt received"

egp=$(json_get "$receipt" "effectiveGasPrice")
egp_dec=$(hex_to_dec "$egp")
base_fee=$(get_base_fee 1)
gas_tip=$(get_header_gas_tip 1)

observe "effectiveGasPrice_regular" "$egp_dec"
observe "baseFee" "$base_fee"
observe "gasTip" "$gas_tip"

# Regular: effectiveGasPrice >= baseFee + gasTip
min_egp=$(( base_fee + gas_tip ))
assert_ge "$egp_dec" "$min_egp" "EGP >= baseFee + gasTip for regular account"

test_result
