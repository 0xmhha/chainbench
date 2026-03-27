#!/usr/bin/env bash
# Test: basic/peers
# Description: Verify all nodes have proper peer connectivity
set -euo pipefail

source "$(dirname "$0")/../lib/rpc.sh"
source "$(dirname "$0")/../lib/assert.sh"

test_start "basic/peers"

# Collect running node IDs from pids.json
running_nodes=$(python3 -c "
import json, sys
with open('${CHAINBENCH_DIR}/state/pids.json') as f:
    state = json.load(f)
ids = [nid for nid, info in state.get('nodes', {}).items()
       if info.get('status') == 'running']
print(' '.join(sorted(ids, key=int)))
")

if [[ -z "$running_nodes" ]]; then
  _assert_fail "peers: no running nodes found in pids.json"
  test_result
  exit 1
fi

total_connections=0
node_count=0

for node_id in $running_nodes; do
  count=$(peer_count "$node_id" 2>/dev/null || echo "0")
  printf '[INFO]  Node %s peer count: %s\n' "$node_id" "$count" >&2

  # Assert each node has at least 1 peer
  assert_ge "$count" "1" "node $node_id has >= 1 peer (got $count)"

  total_connections=$(( total_connections + count ))
  node_count=$(( node_count + 1 ))
done

# Assert total connections are reasonable:
# Each peer connection is counted twice (once per side), so
# total should be >= node_count - 1 (minimum spanning connectivity).
min_expected=$(( node_count - 1 ))
assert_ge "$total_connections" "$min_expected" \
  "total peer connections ($total_connections) >= minimum expected ($min_expected)"

printf '[INFO]  %d nodes checked, %d total peer connections\n' \
  "$node_count" "$total_connections" >&2

test_result
