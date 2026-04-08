#!/usr/bin/env bash
# Test: regression/a-ethereum/a2-01-legacy-tx
# RT-A-2-01 — Legacy Tx (type 0x0) 발행
# [v2 보강] effectiveGasPrice + gasLimit valid check (21000)
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a2-01-legacy-tx"

check_env || { test_result; exit 1; }

# baseFee 조회
base_fee=$(get_base_fee "1")
printf '[INFO]  current baseFee = %s wei\n' "$base_fee" >&2

# TEST_ACC_A → TEST_ACC_B 1 wei Legacy tx 송금
tx_hash=$(send_raw_tx "1" "$TEST_ACC_A_PK" "$TEST_ACC_B_ADDR" "1" "" "21000" "legacy")
assert_not_empty "$tx_hash" "legacy tx hash returned"
assert_contains "$tx_hash" "0x" "tx hash is hex"
printf '[INFO]  tx hash = %s\n' "$tx_hash" >&2

# Receipt 대기
receipt=$(wait_tx_receipt_full "1" "$tx_hash" 30)
assert_not_empty "$receipt" "receipt retrieved"

# status == 0x1
status=$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', ''))")
assert_eq "$status" "0x1" "receipt.status == 0x1"

# tx type == 0x0 (legacy)
tx_type=$(rpc "1" "eth_getTransactionByHash" "[\"${tx_hash}\"]" | json_get - "result.type")
assert_eq "$tx_type" "0x0" "tx type is 0x0 (legacy)"

# gasUsed == 21000 (단순 송금)
gas_used_hex=$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('gasUsed', ''))")
gas_used_dec=$(hex_to_dec "$gas_used_hex")
assert_eq "$gas_used_dec" "21000" "gasUsed == 21000 for simple transfer"

# gasLimit valid check: tx.gas >= 21000
tx_gas_hex=$(rpc "1" "eth_getTransactionByHash" "[\"${tx_hash}\"]" | json_get - "result.gas")
tx_gas_dec=$(hex_to_dec "$tx_gas_hex")
assert_ge "$tx_gas_dec" "21000" "tx.gasLimit >= 21000"

# effectiveGasPrice != null, 올바른 값
effective_gp_hex=$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('effectiveGasPrice', ''))")
assert_not_empty "$effective_gp_hex" "effectiveGasPrice is present (PR #70 fix)"
effective_gp_dec=$(hex_to_dec "$effective_gp_hex")
assert_ge "$effective_gp_dec" "$base_fee" "effectiveGasPrice >= baseFee"

test_result
