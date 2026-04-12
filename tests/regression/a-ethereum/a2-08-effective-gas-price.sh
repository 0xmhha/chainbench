#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-A-2-08
# name: eth_getTransactionReceipt의 effectiveGasPrice 검증
# category: regression/a-ethereum
# tags: [tx, gas]
# estimated_seconds: 38
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/a-ethereum/a2-08-effective-gas-price
# RT-A-2-08 — eth_getTransactionReceipt의 effectiveGasPrice 검증
# PR #70 회귀 검증: BP와 EN(sync) 양쪽에서 동일한 effectiveGasPrice 반환
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a2-08-effective-gas-price"
check_env || { test_result; exit 1; }

# BP(node1)에 tx 전송
tx_hash=$(send_raw_tx "1" "$TEST_ACC_A_PK" "$TEST_ACC_B_ADDR" "1" "" "21000" "dynamic")
assert_not_empty "$tx_hash" "tx hash returned from BP node1"

# BP에서 receipt 대기
bp_receipt=$(wait_tx_receipt_full "1" "$tx_hash" 30)
assert_not_empty "$bp_receipt" "BP receipt retrieved"

# BP effectiveGasPrice
bp_egp=$(printf '%s' "$bp_receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('effectiveGasPrice', ''))")
assert_not_empty "$bp_egp" "BP receipt.effectiveGasPrice != null"
printf '[INFO]  BP effectiveGasPrice = %s\n' "$bp_egp" >&2

# EN(node5)이 sync 되었을 때 receipt 조회
sleep 3
en_receipt=$(get_receipt "5" "$tx_hash")
assert_not_empty "$en_receipt" "EN receipt retrieved"
en_egp=$(printf '%s' "$en_receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('effectiveGasPrice', ''))")
assert_not_empty "$en_egp" "EN receipt.effectiveGasPrice != null (PR #70 fix)"
printf '[INFO]  EN effectiveGasPrice = %s\n' "$en_egp" >&2

# BP/EN effectiveGasPrice 동일 검증
assert_eq "$en_egp" "$bp_egp" "BP and EN return identical effectiveGasPrice"

# 값 자체가 올바른지 (baseFee + tip 공식)
base_fee_hex=$(rpc "1" "eth_getBlockByNumber" "[\"$(printf '%s' "$bp_receipt" | python3 -c 'import sys, json; print(json.load(sys.stdin).get("blockNumber", ""))')\", false]" | json_get - "result.baseFeePerGas")
base_fee=$(hex_to_dec "$base_fee_hex")
egp_dec=$(hex_to_dec "$bp_egp")
assert_ge "$egp_dec" "$base_fee" "effectiveGasPrice >= baseFee"

test_result
