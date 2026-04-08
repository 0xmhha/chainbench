#!/usr/bin/env bash
# RT-G-1-03 — eth_getTransactionByHash
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/g-api/g1-03-get-tx-by-hash"
check_env || { test_result; exit 1; }

# tx 발행
tx_hash=$(send_raw_tx "1" "$TEST_ACC_A_PK" "$TEST_ACC_B_ADDR" "1" "" "21000" "dynamic")
wait_receipt "1" "$tx_hash" 30 >/dev/null

resp=$(rpc 1 eth_getTransactionByHash "[\"${tx_hash}\"]")
assert_not_empty "$(printf '%s' "$resp" | json_get - 'result.blockNumber')" "tx has blockNumber"
assert_not_empty "$(printf '%s' "$resp" | json_get - 'result.from')" "tx has from"
assert_not_empty "$(printf '%s' "$resp" | json_get - 'result.to')" "tx has to"
assert_not_empty "$(printf '%s' "$resp" | json_get - 'result.value')" "tx has value"

test_result
