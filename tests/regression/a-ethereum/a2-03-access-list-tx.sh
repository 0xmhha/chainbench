#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-A-2-03
# name: AccessListTx (type 0x1) 발행
# category: regression/a-ethereum
# tags: [tx, accesslist]
# estimated_seconds: 35
# preconditions:
#   chain_running: true
# depends_on: []
# ---end-meta---
# Test: regression/a-ethereum/a2-03-access-list-tx
# RT-A-2-03 — AccessListTx (type 0x1) 발행
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a2-03-access-list-tx"

check_env || { test_result; exit 1; }
ensure_nodes_running

tx_hash=$(tx_send_as 1 "$(acct_addr 2)" "1" "" "50000" "accesslist")
assert_not_empty "$tx_hash" "access list tx hash returned"

receipt=$(wait_tx_receipt_full "$(node 1)" "$tx_hash" "$TIMEOUT_TX_RECEIPT")
assert_not_empty "$receipt" "receipt retrieved"
status=$(printf '%s' "$receipt" | jq -r '.status // empty')
assert_eq "$status" "0x1" "receipt.status == 0x1"

# tx type == 0x1
tx_type=$(rpc "$(node 1)" "eth_getTransactionByHash" "[\"${tx_hash}\"]" | json_get - "result.type")
assert_eq "$tx_type" "0x1" "tx type is 0x1 (access list)"

# accessList 필드 존재 (비어있어도 OK)
has_access=$(rpc "$(node 1)" "eth_getTransactionByHash" "[\"${tx_hash}\"]" \
  | jq -r 'if (.result | has("accessList")) then "yes" else "no" end')
assert_eq "$has_access" "yes" "tx has accessList field"

test_result
