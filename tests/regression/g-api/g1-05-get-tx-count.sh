#!/usr/bin/env bash
# RT-G-1-05 — eth_getTransactionCount (nonce 조회)
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/g-api/g1-05-get-tx-count"
check_env || { test_result; exit 1; }

before=$(hex_to_dec "$(rpc 1 eth_getTransactionCount "[\"${TEST_ACC_A_ADDR}\", \"latest\"]" | json_get - result)")

# tx 발행
tx_hash=$(send_raw_tx "1" "$TEST_ACC_A_PK" "$TEST_ACC_B_ADDR" "1" "" "21000" "dynamic")
wait_receipt "1" "$tx_hash" 30 >/dev/null

after=$(hex_to_dec "$(rpc 1 eth_getTransactionCount "[\"${TEST_ACC_A_ADDR}\", \"latest\"]" | json_get - result)")
assert_eq "$((after - before))" "1" "nonce incremented by 1 after tx"

test_result
