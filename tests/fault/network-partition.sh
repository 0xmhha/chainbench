#!/usr/bin/env bash
# Test: fault/network-partition
# Description: Simulate network partition via admin_removePeer - verify consensus halts and recovers after heal
set -euo pipefail

source "$(dirname "$0")/../lib/rpc.sh"
source "$(dirname "$0")/../lib/assert.sh"
source "$(dirname "$0")/../lib/wait.sh"

test_start "fault/network-partition"

# ---------------------------------------------------------------------------
# Pre-check
# ---------------------------------------------------------------------------

sync_result=$(wait_for_sync 60)
assert_eq "$sync_result" "synced" "nodes synced before partition test"

running_nodes=$(get_running_node_ids)
node_count=$(echo "$running_nodes" | wc -w | tr -d ' ')
printf '[INFO]  Running nodes: %s (count: %s)\n' "$running_nodes" "$node_count" >&2

if [[ "$node_count" -lt 4 ]]; then
  _assert_fail "need 4 running nodes for partition test (found $node_count)"
  test_result
  exit 1
fi

# ---------------------------------------------------------------------------
# Collect enode URLs
# ---------------------------------------------------------------------------

declare -A enode_urls=()
for nid in $running_nodes; do
  enode=$(admin_enode "$nid" 2>/dev/null || echo "")
  if [[ -n "$enode" ]]; then
    enode_urls[$nid]="$enode"
    printf '[INFO]  Node %s enode: %s\n' "$nid" "${enode:0:60}..." >&2
  fi
done

collected=${#enode_urls[@]}
assert_ge "$collected" "4" "collected enode URLs for all nodes ($collected)"

block_before=$(block_number "1")
printf '[INFO]  Block before partition: %s\n' "$block_before" >&2

# ---------------------------------------------------------------------------
# Create partition: {1,2} vs {3,4}
# Neither side meets BFT threshold (needs 3/4), so consensus should halt.
# ---------------------------------------------------------------------------

printf '[INFO]  Creating partition: {1,2} vs {3,4}...\n' >&2

# Cross-partition peer removal: disconnect all links between the two groups
for src in 1 2; do
  for dst in 3 4; do
    [[ -n "${enode_urls[$dst]:-}" ]] && \
      admin_remove_peer "$src" "${enode_urls[$dst]}" > /dev/null 2>&1
    [[ -n "${enode_urls[$src]:-}" ]] && \
      admin_remove_peer "$dst" "${enode_urls[$src]}" > /dev/null 2>&1
  done
done

# Brief wait for disconnection to settle
sleep 5

# Repeatedly enforce partition in case static node reconnection is attempted
for _ in 1 2 3; do
  for src in 1 2; do
    for dst in 3 4; do
      [[ -n "${enode_urls[$dst]:-}" ]] && \
        admin_remove_peer "$src" "${enode_urls[$dst]}" > /dev/null 2>&1
      [[ -n "${enode_urls[$src]:-}" ]] && \
        admin_remove_peer "$dst" "${enode_urls[$src]}" > /dev/null 2>&1
    done
  done
  sleep 2
done

# Verify peer counts reflect partition
p1_peers=$(peer_count "1" 2>/dev/null || echo "0")
p3_peers=$(peer_count "3" 2>/dev/null || echo "0")
printf '[INFO]  Partition A (node1) peers: %s\n' "$p1_peers" >&2
printf '[INFO]  Partition B (node3) peers: %s\n' "$p3_peers" >&2

# Each node should see at most its partition partner
assert_ge "2" "$p1_peers" "node1 peers <= 2 during partition (got $p1_peers)"
assert_ge "2" "$p3_peers" "node3 peers <= 2 during partition (got $p3_peers)"

# ---------------------------------------------------------------------------
# Verify consensus halt: no significant block advance during partition
# ---------------------------------------------------------------------------

wait_seconds 15 "Waiting 15s to confirm consensus halt under 2-2 partition"

block_during=$(block_number "1" 2>/dev/null || echo "$block_before")
advance=$(( block_during - block_before ))
printf '[INFO]  Block advance during partition: %d\n' "$advance" >&2

# Allow up to 2 in-flight blocks at the moment of partition
assert_ge "2" "$advance" \
  "consensus halted during 2-2 partition (advance=$advance)"

# ---------------------------------------------------------------------------
# Heal partition: reconnect all cross-partition peers
# ---------------------------------------------------------------------------

printf '[INFO]  Healing partition...\n' >&2

for src in 1 2; do
  for dst in 3 4; do
    [[ -n "${enode_urls[$dst]:-}" ]] && \
      admin_add_peer "$src" "${enode_urls[$dst]}" > /dev/null 2>&1
    [[ -n "${enode_urls[$src]:-}" ]] && \
      admin_add_peer "$dst" "${enode_urls[$src]}" > /dev/null 2>&1
  done
done

# Wait for peer reconnection
sleep 5

# Verify consensus resumes
block_after_heal=$(block_number "1" 2>/dev/null || echo "$block_during")
target_resume=$(( block_after_heal + 5 ))
printf '[INFO]  Waiting for block %d to confirm consensus resumed...\n' "$target_resume" >&2

reached=$(wait_for_block "1" "$target_resume" 60)
if [[ "$reached" == "timeout" ]]; then
  _assert_fail "consensus did not resume after partition heal"
  test_result
  exit 1
fi

assert_ge "$reached" "$target_resume" \
  "consensus resumed after partition heal (reached $reached)"

# Full sync check
sync_after=$(wait_for_sync 90)
assert_eq "$sync_after" "synced" "all nodes synced after partition recovery"

test_result
