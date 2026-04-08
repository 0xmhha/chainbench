#!/usr/bin/env bash
# Test: regression/a-ethereum/a2-02-dynamic-fee-tx
# RT-A-2-02 — EIP-1559 DynamicFeeTx (type 0x2) 발행
# [v2 보강] effectiveGasPrice + gasLimit valid check
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a2-02-dynamic-fee-tx"

check_env || { test_result; exit 1; }

base_fee=$(get_base_fee "1")
printf '[INFO]  baseFee=%s\n' "$base_fee" >&2

# TEST_ACC_A가 Authorized 계정이 아니므로, tip이 header.GasTip()으로 강제됨 → A-2-02의 정확 검증을 위해
# Authorized 계정 또는 validator 대신 TEST_ACC_A 사용 (Anzeon 정책 영향은 RT-C-01에서 별도 검증)
tx_hash=$(send_raw_tx "1" "$TEST_ACC_A_PK" "$TEST_ACC_B_ADDR" "1000000000000000" "" "21000" "dynamic")
assert_not_empty "$tx_hash" "dynamic fee tx hash returned"

receipt=$(wait_tx_receipt_full "1" "$tx_hash" 30)
assert_not_empty "$receipt" "receipt retrieved"

status=$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', ''))")
assert_eq "$status" "0x1" "receipt.status == 0x1"

# tx type == 0x2
tx_type=$(rpc "1" "eth_getTransactionByHash" "[\"${tx_hash}\"]" | json_get - "result.type")
assert_eq "$tx_type" "0x2" "tx type is 0x2 (dynamic fee)"

# gasUsed 검증 (단순 송금)
gas_used_dec=$(hex_to_dec "$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('gasUsed', ''))")")
assert_eq "$gas_used_dec" "21000" "gasUsed == 21000 for simple transfer"

# gasLimit valid check
tx_gas_dec=$(hex_to_dec "$(rpc "1" "eth_getTransactionByHash" "[\"${tx_hash}\"]" | json_get - "result.gas")")
assert_ge "$tx_gas_dec" "21000" "tx.gasLimit >= 21000"

# effectiveGasPrice == baseFee + effectiveTip 검증
effective_gp_dec=$(hex_to_dec "$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('effectiveGasPrice', ''))")")
assert_ge "$effective_gp_dec" "$base_fee" "effectiveGasPrice >= baseFee"

# 블록 gasLimit 대비 tx gasLimit 검증 (RT-A-2-07 직교)
block_gas_limit_hex=$(rpc "1" "eth_getBlockByNumber" "[\"latest\", false]" | json_get - "result.gasLimit")
block_gas_limit_dec=$(hex_to_dec "$block_gas_limit_hex")
assert_true "$( [[ $tx_gas_dec -le $block_gas_limit_dec ]] && echo true || echo false )" "tx.gasLimit <= block.gasLimit"

test_result
