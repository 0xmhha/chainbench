#!/usr/bin/env bash
# Test: stress/tx-flood
# Description: Send N transactions rapidly and measure throughput
set -euo pipefail

source "$(dirname "$0")/../lib/rpc.sh"
source "$(dirname "$0")/../lib/assert.sh"

test_start "stress/tx-flood"

# Default N=100, configurable via $1
tx_count="${1:-100}"

if ! [[ "$tx_count" =~ ^[1-9][0-9]*$ ]]; then
  _assert_fail "tx-flood: invalid tx count '$tx_count' — must be a positive integer"
  test_result
  exit 1
fi

printf '[INFO]  Sending %d transactions...\n' "$tx_count" >&2

# Get coinbase and unlock it
coinbase=$(get_coinbase "1")
assert_not_empty "$coinbase" "node1 coinbase is set"

unlock_account "1" "$coinbase" "1" 3600
printf '[INFO]  Unlocked account: %s\n' "$coinbase" >&2

# Use node 2's coinbase as recipient (or coinbase itself for loopback)
recipient=$(get_coinbase "2" 2>/dev/null || echo "")
if [[ -z "$recipient" || "$recipient" == "null" ]]; then
  recipient=$(rpc "1" "eth_accounts" "[]" \
    | python3 -c "
import sys, json
accounts = json.load(sys.stdin).get('result', [])
print(accounts[1] if len(accounts) > 1 else accounts[0])
" 2>/dev/null || echo "$coinbase")
fi
printf '[INFO]  Recipient: %s\n' "$recipient" >&2

# Record start time
flood_start=$(date +%s)

# Send N transactions sequentially, collecting hashes
declare -a tx_hashes=()
send_errors=0

for i in $(seq 1 "$tx_count"); do
  hash=$(send_tx "1" "$coinbase" "$recipient" "0x1" 2>/dev/null || echo "")

  if [[ -z "$hash" || "$hash" == TX_ERROR:* ]]; then
    send_errors=$(( send_errors + 1 ))
    printf '[WARN]  tx %d failed: %s\n' "$i" "${hash:-empty}" >&2
  else
    tx_hashes+=("$hash")
  fi
done

# Record end time
flood_end=$(date +%s)
flood_elapsed=$(( flood_end - flood_start ))
sent_count=${#tx_hashes[@]}

printf '[INFO]  Sent %d/%d transactions in %ds\n' \
  "$sent_count" "$tx_count" "$flood_elapsed" >&2

assert_gt "$sent_count" "0" "at least 1 transaction sent successfully"

# Calculate approximate send TPS
if [[ "$flood_elapsed" -gt 0 ]]; then
  send_tps=$(( sent_count / flood_elapsed ))
  printf '[INFO]  Send rate: ~%d tx/s\n' "$send_tps" >&2
fi

# Wait for all receipts to arrive (generous timeout based on tx count)
receipt_timeout=$(( tx_count / 5 + 60 ))
printf '[INFO]  Waiting for receipts (timeout %ds)...\n' "$receipt_timeout" >&2

confirmed=0
failed_receipts=0
confirm_start=$(date +%s)

for hash in "${tx_hashes[@]}"; do
  status=$(wait_receipt "1" "$hash" "$receipt_timeout" 2>/dev/null || echo "timeout")
  if [[ "$status" == "success" ]]; then
    confirmed=$(( confirmed + 1 ))
  elif [[ "$status" == "failed" ]]; then
    failed_receipts=$(( failed_receipts + 1 ))
  fi
  # Progressively reduce timeout — earlier txs should have already mined
  receipt_timeout=$(( receipt_timeout > 5 ? receipt_timeout - 1 : 5 ))
done

confirm_end=$(date +%s)
confirm_elapsed=$(( confirm_end - confirm_start ))
total_elapsed=$(( confirm_end - flood_start ))

# Calculate confirmed TPS
if [[ "$total_elapsed" -gt 0 ]]; then
  confirmed_tps=$(( confirmed / total_elapsed ))
  printf '[INFO]  Confirmed %d txs in %ds (TPS: ~%d)\n' \
    "$confirmed" "$total_elapsed" "$confirmed_tps" >&2
fi

printf '[INFO]  Results: sent=%d confirmed=%d failed=%d errors=%d\n' \
  "$sent_count" "$confirmed" "$failed_receipts" "$send_errors" >&2

# At least 80% of sent transactions should be confirmed
min_confirmed=$(( sent_count * 8 / 10 ))
assert_ge "$confirmed" "$min_confirmed" \
  "at least 80% of transactions confirmed ($confirmed/$sent_count)"

test_result
