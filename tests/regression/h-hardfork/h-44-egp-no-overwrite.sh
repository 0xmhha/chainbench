#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-4-6-03
# name: EffectiveGasPrice consistent across BP and EN nodes
# category: regression/h-hardfork
# tags: [hardfork, boho, effectiveGasPrice, consistency]
# estimated_seconds: 20
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# TC-4-6-03 — EGP must be same on block producer and endpoint node
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-44-egp-no-overwrite"
check_env || { test_result; exit 1; }

tx_hash=$(send_raw_tx 1 "$TEST_ACC_A_PK" "$TEST_ACC_B_ADDR" "1000000000000000000")
assert_not_empty "$tx_hash" "tx sent"

# Wait for receipt on both nodes
receipt_bp=$(wait_tx_receipt_full 1 "$tx_hash" 30)
receipt_en=$(wait_tx_receipt_full 5 "$tx_hash" 30)

assert_not_empty "$receipt_bp" "receipt from BP node"
assert_not_empty "$receipt_en" "receipt from EN node"

egp_bp=$(json_get "$receipt_bp" "effectiveGasPrice")
egp_en=$(json_get "$receipt_en" "effectiveGasPrice")

observe "egp_bp" "$egp_bp"
observe "egp_en" "$egp_en"

assert_eq "$egp_bp" "$egp_en" "effectiveGasPrice consistent between BP and EN"

test_result
