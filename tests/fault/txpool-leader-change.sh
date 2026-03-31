#!/usr/bin/env bash
# Test: fault/txpool-leader-change
# Description: Verify pending transactions survive leader node failure and get processed by remaining validators
set -euo pipefail

source "$(dirname "$0")/../lib/rpc.sh"
source "$(dirname "$0")/../lib/assert.sh"
source "$(dirname "$0")/../lib/wait.sh"

test_start "fault/txpool-leader-change"

# Pre-check: all nodes synced
sync_result=$(wait_for_sync 60)
assert_eq "$sync_result" "synced" "nodes synced before test"

# Setup
coinbase=$(get_coinbase "1")
assert_not_empty "$coinbase" "node1 coinbase is set"
unlock_account "1" "$coinbase" "1" 3600

recipient=$(get_coinbase "2" 2>/dev/null || echo "$coinbase")

# Record recipient balance before test
initial_balance=$(get_balance "2" "$recipient" 2>/dev/null || echo "0")
printf '[INFO]  Recipient initial balance: %s wei\n' "$initial_balance" >&2

# ---------------------------------------------------------------------------
# Phase 1: Send TXs to node 1 and wait for propagation
# ---------------------------------------------------------------------------

tx_count=10
declare -a tx_hashes=()
for i in $(seq 1 "$tx_count"); do
  h=$(send_tx "1" "$coinbase" "$recipient" "0x1" 2>/dev/null || echo "")
  if [[ -n "$h" && "$h" != TX_ERROR:* ]]; then
    tx_hashes+=("$h")
  fi
done

sent=${#tx_hashes[@]}
printf '[INFO]  Sent %d/%d transactions to node 1\n' "$sent" "$tx_count" >&2
assert_gt "$sent" "0" "at least 1 transaction sent"

# Allow propagation to other nodes
sleep 3

# Snapshot txpool on node 2 before crash
pool_node2_before=$(txpool_pending_count "2" 2>/dev/null || echo "0")
printf '[INFO]  Node 2 txpool pending before crash: %s\n' "$pool_node2_before" >&2

# ---------------------------------------------------------------------------
# Phase 2: Kill node 1 (simulate leader failure)
# ---------------------------------------------------------------------------

printf '[INFO]  Stopping node 1 (leader failure)...\n' >&2
"${CHAINBENCH_DIR}/chainbench.sh" node stop 1 --quiet

# Verify remaining nodes continue producing blocks (3/4 BFT threshold)
block_before=$(block_number "2" 2>/dev/null || echo "0")
target_block=$(( block_before + 10 ))
printf '[INFO]  Waiting for block %d on node 2...\n' "$target_block" >&2

reached=$(wait_for_block "2" "$target_block" 60)
if [[ "$reached" == "timeout" ]]; then
  _assert_fail "consensus stalled after node 1 failure"
  "${CHAINBENCH_DIR}/chainbench.sh" node start 1 --quiet 2>/dev/null || true
  test_result
  exit 1
fi

assert_ge "$reached" "$target_block" \
  "remaining validators produced 10 blocks after leader failure"

# ---------------------------------------------------------------------------
# Phase 3: Verify pending TXs got processed
# ---------------------------------------------------------------------------

final_pending=$(txpool_pending_count "2" 2>/dev/null || echo "0")
printf '[INFO]  Node 2 txpool pending after 10 blocks: %s\n' "$final_pending" >&2
assert_eq "$final_pending" "0" "all pending TXs processed after leader failure"

# Check some TX receipts on node 2 (surviving node)
confirmed=0
for hash in "${tx_hashes[@]}"; do
  status=$(wait_receipt "2" "$hash" 5 2>/dev/null || echo "timeout")
  if [[ "$status" == "success" ]]; then
    confirmed=$(( confirmed + 1 ))
  fi
done

printf '[INFO]  Confirmed %d/%d TXs on node 2\n' "$confirmed" "$sent" >&2

min_confirmed=$(( sent * 8 / 10 ))
assert_ge "$confirmed" "$min_confirmed" \
  ">=80%% of TXs confirmed on surviving node ($confirmed/$sent)"

# ---------------------------------------------------------------------------
# Phase 4: Restore cluster
# ---------------------------------------------------------------------------

printf '[INFO]  Restarting node 1...\n' >&2
"${CHAINBENCH_DIR}/chainbench.sh" node start 1 --quiet

sync_after=$(wait_for_sync 90)
assert_eq "$sync_after" "synced" "all nodes synced after recovery"

test_result
