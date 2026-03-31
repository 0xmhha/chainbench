#!/usr/bin/env bash
# Test: fault/p2p-topology
# Description: Test consensus and TX propagation under restricted hub-spoke P2P topology
set -euo pipefail

source "$(dirname "$0")/../lib/rpc.sh"
source "$(dirname "$0")/../lib/assert.sh"
source "$(dirname "$0")/../lib/wait.sh"

test_start "fault/p2p-topology"

# ---------------------------------------------------------------------------
# Pre-check
# ---------------------------------------------------------------------------

sync_result=$(wait_for_sync 60)
assert_eq "$sync_result" "synced" "nodes synced before topology test"

running_nodes=$(get_running_node_ids)
node_count=$(echo "$running_nodes" | wc -w | tr -d ' ')

if [[ "$node_count" -lt 4 ]]; then
  _assert_fail "need 4 running nodes (found $node_count)"
  test_result
  exit 1
fi

# Collect enode URLs
declare -A enode_urls=()
for nid in $running_nodes; do
  enode=$(admin_enode "$nid" 2>/dev/null || echo "")
  [[ -n "$enode" ]] && enode_urls[$nid]="$enode"
done

# ---------------------------------------------------------------------------
# Helper: restore full-mesh topology (used in cleanup and at end)
# ---------------------------------------------------------------------------
_restore_full_mesh() {
  printf '[INFO]  Restoring full mesh topology...\n' >&2
  for src in $running_nodes; do
    for dst in $running_nodes; do
      [[ "$src" == "$dst" ]] && continue
      [[ -n "${enode_urls[$dst]:-}" ]] && \
        admin_add_peer "$src" "${enode_urls[$dst]}" > /dev/null 2>&1
    done
  done
  sleep 3
}

# Ensure cleanup on exit
trap '_restore_full_mesh' EXIT

# ---------------------------------------------------------------------------
# Phase 1: Create hub-spoke topology (node 1 = hub)
#
# Full mesh:  1-2, 1-3, 1-4, 2-3, 2-4, 3-4
# Hub-spoke:  1-2, 1-3, 1-4  (remove: 2-3, 2-4, 3-4)
#
# All consensus messages must relay through node 1.
# ---------------------------------------------------------------------------

printf '[INFO]  Creating hub-spoke topology (hub=node1)...\n' >&2

for src in 2 3 4; do
  for dst in 2 3 4; do
    [[ "$src" -ge "$dst" ]] && continue
    [[ -n "${enode_urls[$dst]:-}" ]] && \
      admin_remove_peer "$src" "${enode_urls[$dst]}" > /dev/null 2>&1
    [[ -n "${enode_urls[$src]:-}" ]] && \
      admin_remove_peer "$dst" "${enode_urls[$src]}" > /dev/null 2>&1
  done
done

sleep 3

# Enforce removal again to counter auto-reconnect
for _ in 1 2; do
  for src in 2 3 4; do
    for dst in 2 3 4; do
      [[ "$src" -ge "$dst" ]] && continue
      [[ -n "${enode_urls[$dst]:-}" ]] && \
        admin_remove_peer "$src" "${enode_urls[$dst]}" > /dev/null 2>&1
      [[ -n "${enode_urls[$src]:-}" ]] && \
        admin_remove_peer "$dst" "${enode_urls[$src]}" > /dev/null 2>&1
    done
  done
  sleep 2
done

# Verify topology
hub_peers=$(peer_count "1" 2>/dev/null || echo "0")
printf '[INFO]  Hub (node1) peers: %s\n' "$hub_peers" >&2
assert_ge "$hub_peers" "2" "hub has >= 2 peers (got $hub_peers)"

for nid in 2 3 4; do
  spoke_peers=$(peer_count "$nid" 2>/dev/null || echo "0")
  printf '[INFO]  Spoke (node%s) peers: %s\n' "$nid" "$spoke_peers" >&2
done

# ---------------------------------------------------------------------------
# Phase 2: Verify consensus works under hub-spoke
# ---------------------------------------------------------------------------

block_before=$(block_number "1")
target_block=$(( block_before + 10 ))
printf '[INFO]  Verifying consensus: waiting for block %d...\n' "$target_block" >&2

reached=$(wait_for_block "1" "$target_block" 60)
if [[ "$reached" == "timeout" ]]; then
  _assert_fail "consensus stalled under hub-spoke topology"
  test_result
  exit 1
fi

assert_ge "$reached" "$target_block" \
  "consensus works under hub-spoke (reached $reached)"

# ---------------------------------------------------------------------------
# Phase 3: TX propagation through hub
# ---------------------------------------------------------------------------

printf '[INFO]  Testing TX propagation through hub...\n' >&2

coinbase=$(get_coinbase "1")
unlock_account "1" "$coinbase" "1" 600
recipient=$(get_coinbase "2" 2>/dev/null || echo "$coinbase")

tx_hash=$(send_tx "1" "$coinbase" "$recipient" "0x1" 2>/dev/null || echo "")

if [[ -n "$tx_hash" && "$tx_hash" != TX_ERROR:* ]]; then
  printf '[INFO]  Sent TX via hub: %s\n' "$tx_hash" >&2
  receipt=$(wait_receipt "1" "$tx_hash" 30 2>/dev/null || echo "timeout")
  assert_eq "$receipt" "success" "TX via hub-spoke confirmed ($receipt)"
else
  printf '[WARN]  TX send failed: %s\n' "$tx_hash" >&2
  _assert_fail "failed to send TX through hub"
fi

# Verify all spokes synced through hub
sync_hub=$(wait_for_sync 60)
assert_eq "$sync_hub" "synced" "all spokes synced through hub"

# ---------------------------------------------------------------------------
# Phase 4: Block propagation timing under hub-spoke
# ---------------------------------------------------------------------------

printf '[INFO]  Measuring block propagation delay through hub...\n' >&2

# Sample: compare block arrival at hub (node 1) vs farthest spoke (node 4)
delays=()
for _ in $(seq 1 5); do
  b1=$(block_number "1" 2>/dev/null || echo "0")
  b4=$(block_number "4" 2>/dev/null || echo "0")
  lag=$(( b1 - b4 ))
  [[ "$lag" -lt 0 ]] && lag=$(( -lag ))
  delays+=("$lag")
  sleep 2
done

max_lag=0
for d in "${delays[@]}"; do
  [[ "$d" -gt "$max_lag" ]] && max_lag="$d"
done

printf '[INFO]  Max block lag hub<->spoke: %d blocks\n' "$max_lag" >&2
assert_ge "3" "$max_lag" "hub-spoke block lag <= 3 (got $max_lag)"

# ---------------------------------------------------------------------------
# Phase 5: Restore full mesh (trap handles this, but also assert sync)
# ---------------------------------------------------------------------------

_restore_full_mesh
sync_restored=$(wait_for_sync 60)
assert_eq "$sync_restored" "synced" "all nodes synced after full mesh restore"

# Deactivate trap since we already restored
trap - EXIT

test_result
