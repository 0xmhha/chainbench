#!/usr/bin/env bash
# Test: basic/rpc-health
# Description: Verify all node RPC endpoints are responding
set -euo pipefail

source "$(dirname "$0")/../lib/rpc.sh"
source "$(dirname "$0")/../lib/assert.sh"

test_start "basic/rpc-health"

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
  _assert_fail "rpc-health: no running nodes found in pids.json"
  test_result
  exit 1
fi

for node_id in $running_nodes; do
  # Call eth_blockNumber and capture result
  raw_response=$(rpc "$node_id" "eth_blockNumber" "[]" 2>/dev/null || echo "")

  if [[ -z "$raw_response" ]]; then
    _assert_fail "node $node_id: RPC endpoint did not respond"
    continue
  fi

  # Validate that result is a valid hex number
  block_hex=$(python3 -c "
import json, sys
try:
    d = json.loads(sys.argv[1])
    result = d.get('result', '')
    if result and result.startswith('0x'):
        print(result)
    else:
        print('')
except Exception:
    print('')
" "$raw_response")

  assert_not_empty "$block_hex" "node $node_id eth_blockNumber returns a valid hex result"

  if [[ -n "$block_hex" ]]; then
    block_dec=$(python3 -c "print(int('$block_hex', 16))" 2>/dev/null || echo "-1")
    assert_ge "$block_dec" "0" "node $node_id block number is non-negative ($block_dec)"
    printf '[INFO]  Node %s is healthy at block %s\n' "$node_id" "$block_dec" >&2
  fi
done

test_result
