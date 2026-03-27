#!/usr/bin/env bash
# Test: basic/consensus
# Description: Verify blocks are being produced and all validators participate
set -euo pipefail

source "$(dirname "$0")/../lib/rpc.sh"
source "$(dirname "$0")/../lib/assert.sh"
source "$(dirname "$0")/../lib/wait.sh"

test_start "basic/consensus"

# Get current block number on node 1
start_block=$(block_number "1")
assert_gt "$start_block" "-1" "node1 has a valid block number"

# Wait for 20 new blocks (timeout 60s)
target_block=$(( start_block + 20 ))
printf '[INFO]  Waiting for block %d (current: %d)...\n' "$target_block" "$start_block" >&2

before_wait=$(date +%s)
reached_block=$(wait_for_block "1" "$target_block" 60)

if [[ "$reached_block" == "timeout" ]]; then
  _assert_fail "consensus: timed out waiting for 20 new blocks from block $start_block"
  test_result
  exit 1
fi

after_wait=$(date +%s)
elapsed=$(( after_wait - before_wait ))

# Measure average block time
blocks_produced=$(( reached_block - start_block ))
if [[ "$blocks_produced" -gt 0 && "$elapsed" -gt 0 ]]; then
  avg_block_ms=$(( (elapsed * 1000) / blocks_produced ))
  printf '[INFO]  Produced %d blocks in %ds (avg ~%dms per block)\n' \
    "$blocks_produced" "$elapsed" "$avg_block_ms" >&2
fi

assert_ge "$reached_block" "$target_block" "20 new blocks produced (reached $reached_block)"

# Check all running nodes are synced
sync_json=$(check_sync)
synced=$(python3 -c "
import json, sys
d = json.loads(sys.argv[1])
print('true' if d.get('synced') else 'false')
" "$sync_json")
diff_val=$(python3 -c "
import json, sys
d = json.loads(sys.argv[1])
print(d.get('diff', 999))
" "$sync_json")

assert_true "$synced" "all running nodes are synced"
assert_ge "2" "$diff_val" "block diff <= 2 (got $diff_val)"

# Check miner diversity: at least 2 distinct miners in last 20 blocks
current_block=$(block_number "1")
distinct_miners=$(python3 -c "
import sys
current = int(sys.argv[1])
count   = int(sys.argv[2])
start   = max(0, current - count + 1)

import urllib.request, json

port_data = open('${CHAINBENCH_DIR}/state/pids.json')
state = json.load(port_data)
port_data.close()

port = None
for nid, info in state.get('nodes', {}).items():
    if info.get('status') == 'running':
        port = info['http_port']
        break

if port is None:
    print(0)
    sys.exit(0)

miners = set()
for blk in range(start, current + 1):
    hex_blk = hex(blk)
    req = urllib.request.Request(
        f'http://127.0.0.1:{port}',
        data=json.dumps({
            'jsonrpc': '2.0',
            'method': 'eth_getBlockByNumber',
            'params': [hex_blk, False],
            'id': 1,
        }).encode(),
        headers={'Content-Type': 'application/json'},
    )
    try:
        resp = urllib.request.urlopen(req, timeout=5)
        data = json.loads(resp.read())
        result = data.get('result') or {}
        miner = result.get('miner', '')
        if miner:
            miners.add(miner.lower())
    except Exception:
        pass

print(len(miners))
" "$current_block" "20")

assert_ge "$distinct_miners" "2" "at least 2 distinct miners in last 20 blocks (found $distinct_miners)"

test_result
