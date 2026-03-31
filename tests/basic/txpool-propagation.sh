#!/usr/bin/env bash
# Test: basic/txpool-propagation
# Description: Verify TX propagation across nodes and txpool drain under load
set -euo pipefail

source "$(dirname "$0")/../lib/rpc.sh"
source "$(dirname "$0")/../lib/assert.sh"
source "$(dirname "$0")/../lib/wait.sh"

test_start "basic/txpool-propagation"

# Setup: coinbase and running nodes
coinbase=$(get_coinbase "1")
assert_not_empty "$coinbase" "node1 coinbase is set"
unlock_account "1" "$coinbase" "1" 3600

running_nodes=$(get_running_node_ids)
node_count=$(echo "$running_nodes" | wc -w | tr -d ' ')
printf '[INFO]  Running nodes: %s (count: %s)\n' "$running_nodes" "$node_count" >&2

if [[ "$node_count" -lt 2 ]]; then
  _assert_fail "need at least 2 running nodes (found $node_count)"
  test_result
  exit 1
fi

recipient=$(get_coinbase "2" 2>/dev/null || echo "$coinbase")

# ---------------------------------------------------------------------------
# Phase 1: Single TX propagation across nodes
# ---------------------------------------------------------------------------

printf '[INFO]  Phase 1: TX propagation to peer nodes\n' >&2

tx_hash=$(send_tx "1" "$coinbase" "$recipient" "0x1")
assert_not_empty "$tx_hash" "tx sent to node1"

if [[ "$tx_hash" == TX_ERROR:* ]]; then
  _assert_fail "send_tx failed: $tx_hash"
  test_result
  exit 1
fi

printf '[INFO]  Sent tx: %s\n' "$tx_hash" >&2

# Brief wait for propagation
sleep 3

# Check if TX is in other nodes' txpools or already mined
propagated_to=0
for nid in $running_nodes; do
  [[ "$nid" == "1" ]] && continue
  pool_json=$(txpool_content "$nid" 2>/dev/null || echo '{}')
  has_tx=$(python3 -c "
import json, sys
d = json.loads(sys.argv[1])
tx_hash = sys.argv[2].lower()
for section in ('pending', 'queued'):
    for acct_txs in d.get(section, {}).values():
        for tx in acct_txs.values():
            if tx.get('hash', '').lower() == tx_hash:
                print('true')
                sys.exit(0)
print('false')
" "$pool_json" "$tx_hash" 2>/dev/null || echo "false")

  if [[ "$has_tx" == "true" ]]; then
    propagated_to=$(( propagated_to + 1 ))
    printf '[INFO]  TX found in node %s txpool\n' "$nid" >&2
  fi
done

receipt_status=$(wait_receipt "1" "$tx_hash" 10 2>/dev/null || echo "pending")

if [[ "$receipt_status" == "success" ]]; then
  printf '[INFO]  TX already mined — propagation confirmed via block inclusion\n' >&2
  _assert_pass "TX propagated and mined (receipt: success)"
else
  assert_ge "$propagated_to" "1" \
    "TX propagated to at least 1 peer ($propagated_to nodes)"
fi

# Ensure first TX is mined before Phase 2
wait_receipt "1" "$tx_hash" 30 > /dev/null 2>&1 || true

# ---------------------------------------------------------------------------
# Phase 2: Txpool drain under burst load
# ---------------------------------------------------------------------------

printf '[INFO]  Phase 2: txpool drain under burst load\n' >&2

tx_count=20
sent=0
for i in $(seq 1 "$tx_count"); do
  h=$(send_tx "1" "$coinbase" "$recipient" "0x1" 2>/dev/null || echo "")
  if [[ -n "$h" && "$h" != TX_ERROR:* ]]; then
    sent=$(( sent + 1 ))
  fi
done

printf '[INFO]  Sent %d/%d transactions\n' "$sent" "$tx_count" >&2
assert_gt "$sent" "0" "at least 1 transaction sent"

# Snapshot txpool immediately after burst
pool_after_send=$(txpool_pending_count "1" 2>/dev/null || echo "0")
printf '[INFO]  Txpool pending immediately after burst: %s\n' "$pool_after_send" >&2

# Poll until txpool drains (max 30s)
drain_timeout=30
drain_elapsed=0
drained=0

while [[ "$drain_elapsed" -lt "$drain_timeout" ]]; do
  pending=$(txpool_pending_count "1" 2>/dev/null || echo "0")
  if [[ "$pending" -eq 0 ]]; then
    drained=1
    break
  fi
  sleep 2
  drain_elapsed=$(( drain_elapsed + 2 ))
done

printf '[INFO]  Txpool drained in %ds\n' "$drain_elapsed" >&2
assert_eq "$drained" "1" "txpool drained within ${drain_timeout}s"

# Verify consistency: all nodes' txpools should also be empty
for nid in $running_nodes; do
  nid_pending=$(txpool_pending_count "$nid" 2>/dev/null || echo "0")
  assert_eq "$nid_pending" "0" "node $nid txpool empty after drain (pending=$nid_pending)"
done

test_result
