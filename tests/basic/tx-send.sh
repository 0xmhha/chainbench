#!/usr/bin/env bash
# Test: basic/tx-send
# Description: Send a transaction and verify it gets included in a block
set -euo pipefail

source "$(dirname "$0")/../lib/rpc.sh"
source "$(dirname "$0")/../lib/assert.sh"

test_start "basic/tx-send"

# Get coinbase of node 1
coinbase=$(get_coinbase "1")
assert_not_empty "$coinbase" "node1 coinbase address is set"

# Unlock the account (password "1")
unlock_account "1" "$coinbase" "1" 600
printf '[INFO]  Unlocked account: %s\n' "$coinbase" >&2

# Get coinbase of node 2 to use as recipient
recipient=$(get_coinbase "2" 2>/dev/null || echo "")
if [[ -z "$recipient" || "$recipient" == "null" ]]; then
  # Fall back to a deterministic address derived from coinbase via a second call
  recipient=$(rpc "1" "eth_accounts" "[]" \
    | python3 -c "
import sys, json
accounts = json.load(sys.stdin).get('result', [])
print(accounts[1] if len(accounts) > 1 else accounts[0])
" 2>/dev/null || echo "$coinbase")
fi
printf '[INFO]  Recipient address: %s\n' "$recipient" >&2
assert_not_empty "$recipient" "recipient address is available"

# Get initial balance of recipient
initial_balance=$(get_balance "1" "$recipient")
printf '[INFO]  Recipient initial balance: %s wei\n' "$initial_balance" >&2

# Send 1 wei transaction
tx_hash=$(send_tx "1" "$coinbase" "$recipient" "0x1")
assert_not_empty "$tx_hash" "transaction hash returned"
assert_contains "$tx_hash" "0x" "tx hash is a hex string"

if [[ "$tx_hash" == TX_ERROR:* ]]; then
  _assert_fail "send_tx failed: $tx_hash"
  test_result
  exit 1
fi

printf '[INFO]  Tx hash: %s\n' "$tx_hash" >&2

# Wait for receipt (timeout 30s)
receipt_status=$(wait_receipt "1" "$tx_hash" 30)
assert_eq "$receipt_status" "success" "transaction receipt status is success"

test_result
