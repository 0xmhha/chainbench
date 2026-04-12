#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-4-6-04
# name: AuthorizedTxExecuted is last log in authorized tx receipt
# category: regression/h-hardfork
# tags: [hardfork, boho, authorized, event]
# estimated_seconds: 20
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# TC-4-6-04 — AuthorizedTxExecuted event must be the last log entry
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-45-auth-tx-event-last"
check_env || { test_result; exit 1; }

# Send from authorized account (validator)
unlock_validator 1
resp=$(rpc 1 "eth_sendTransaction" \
  "[{\"from\":\"${VALIDATOR_1_ADDR}\",\"to\":\"${TEST_ACC_B_ADDR}\",\"value\":\"0xDE0B6B3A7640000\"}]")
tx_hash=$(json_get "$resp" "result")
assert_not_empty "$tx_hash" "authorized tx sent"

receipt=$(wait_tx_receipt_full 1 "$tx_hash" 30)
assert_not_empty "$receipt" "receipt received"

# Check AuthorizedTxExecuted is the last log
last_log_topic=$(echo "$receipt" | python3 -c "
import json, sys
r = json.loads(sys.stdin.read())
logs = r.get('logs', [])
if not logs:
    print('')
else:
    last = logs[-1]
    topics = last.get('topics', [])
    print(topics[0] if topics else '')
" 2>/dev/null)

observe "last_log_topic" "$last_log_topic"
observe "expected_topic" "$AUTHORIZED_TX_EXECUTED_SIG"

assert_eq "$last_log_topic" "$AUTHORIZED_TX_EXECUTED_SIG" \
  "AuthorizedTxExecuted is the last log in receipt"

test_result
