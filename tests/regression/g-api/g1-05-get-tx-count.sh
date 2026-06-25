#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-G-1-05
# name: eth_getTransactionCount (nonce 조회)
# category: regression/g-api
# tags: [rpc]
# estimated_seconds: 35
# preconditions:
#   chain_running: true
# depends_on: []
# ---end-meta---
# RT-G-1-05 — eth_getTransactionCount (nonce 조회)
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/g-api/g1-05-get-tx-count"
check_env || { test_result; exit 1; }
ensure_nodes_running

before=$(hex_to_dec "$(rpc "$(node 1)" eth_getTransactionCount "[\"$(acct_addr 1)\", \"latest\"]" | json_get - result)")

# tx 발행
tx_hash=$(tx_send_as 1 "$(acct_addr 2)" "1" "" "21000" "dynamic")
wait_receipt "$(node 1)" "$tx_hash" 30 >/dev/null

after=$(hex_to_dec "$(rpc "$(node 1)" eth_getTransactionCount "[\"$(acct_addr 1)\", \"latest\"]" | json_get - result)")
assert_eq "$((after - before))" "1" "nonce incremented by 1 after tx"

test_result
