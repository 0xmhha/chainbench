#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-G-1-03
# name: eth_getTransactionByHash
# category: regression/g-api
# tags: [rpc]
# estimated_seconds: 35
# preconditions:
#   chain_running: true
# depends_on: []
# ---end-meta---
# RT-G-1-03 — eth_getTransactionByHash
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/g-api/g1-03-get-tx-by-hash"
check_env || { test_result; exit 1; }
ensure_nodes_running

# tx 발행
tx_hash=$(tx_send_as 1 "$(acct_addr 2)" "1" "" "21000" "dynamic")
wait_receipt "$(node 1)" "$tx_hash" 30 >/dev/null

resp=$(rpc "$(node 1)" eth_getTransactionByHash "[\"${tx_hash}\"]")
assert_not_empty "$(printf '%s' "$resp" | json_get - 'result.blockNumber')" "tx has blockNumber"
assert_not_empty "$(printf '%s' "$resp" | json_get - 'result.from')" "tx has from"
assert_not_empty "$(printf '%s' "$resp" | json_get - 'result.to')" "tx has to"
assert_not_empty "$(printf '%s' "$resp" | json_get - 'result.value')" "tx has value"

test_result
