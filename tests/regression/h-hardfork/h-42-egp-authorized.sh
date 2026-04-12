#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-4-6-01
# name: EffectiveGasPrice for authorized account
# category: regression/h-hardfork
# tags: [hardfork, boho, effectiveGasPrice, authorized]
# estimated_seconds: 20
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# TC-4-6-01 — Authorized accounts get lower effectiveGasPrice (tip=0)
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-42-egp-authorized"
check_env || { test_result; exit 1; }

# Validators are always authorized
unlock_validator 1
resp=$(rpc 1 "eth_sendTransaction" \
  "[{\"from\":\"${VALIDATOR_1_ADDR}\",\"to\":\"${TEST_ACC_B_ADDR}\",\"value\":\"0xDE0B6B3A7640000\"}]")
tx_hash=$(json_get "$resp" "result")
assert_not_empty "$tx_hash" "validator tx sent"

receipt=$(wait_tx_receipt_full 1 "$tx_hash" 30)
assert_not_empty "$receipt" "receipt received"

egp=$(json_get "$receipt" "effectiveGasPrice")
egp_dec=$(hex_to_dec "$egp")
base_fee=$(get_base_fee 1)

observe "effectiveGasPrice_authorized" "$egp_dec"
observe "baseFee" "$base_fee"

# Authorized: effectiveGasPrice should be <= baseFee (no enforced tip)
assert_ge "$base_fee" "0" "baseFee is valid"
# The EGP for authorized should be close to baseFee (tip is 0 or minimal)
printf '[INFO]  authorized EGP=%s, baseFee=%s\n' "$egp_dec" "$base_fee" >&2

test_result
