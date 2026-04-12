#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-A-2-03
# name: AccessListTx (type 0x1) 발행
# category: regression/a-ethereum
# tags: [tx, accesslist]
# estimated_seconds: 35
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/a-ethereum/a2-03-access-list-tx
# RT-A-2-03 — AccessListTx (type 0x1) 발행
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a2-03-access-list-tx"

check_env || { test_result; exit 1; }

tx_hash=$(send_raw_tx "1" "$TEST_ACC_A_PK" "$TEST_ACC_B_ADDR" "1" "" "50000" "accesslist")
assert_not_empty "$tx_hash" "access list tx hash returned"

receipt=$(wait_tx_receipt_full "1" "$tx_hash" 30)
assert_not_empty "$receipt" "receipt retrieved"
status=$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', ''))")
assert_eq "$status" "0x1" "receipt.status == 0x1"

# tx type == 0x1
tx_type=$(rpc "1" "eth_getTransactionByHash" "[\"${tx_hash}\"]" | json_get - "result.type")
assert_eq "$tx_type" "0x1" "tx type is 0x1 (access list)"

# accessList 필드 존재 (비어있어도 OK)
has_access=$(rpc "1" "eth_getTransactionByHash" "[\"${tx_hash}\"]" | python3 -c "
import sys, json
r = json.load(sys.stdin).get('result', {})
print('yes' if 'accessList' in r else 'no')
")
assert_eq "$has_access" "yes" "tx has accessList field"

test_result
