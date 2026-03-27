#!/usr/bin/env bash
# Test: stress/block-time
# Description: Measure block production time statistics over last 100 blocks
set -euo pipefail

source "$(dirname "$0")/../lib/rpc.sh"
source "$(dirname "$0")/../lib/assert.sh"

test_start "stress/block-time"

# Get current block number on node 1
current_block=$(block_number "1")
assert_gt "$current_block" "0" "chain has produced at least 1 block"

# Use the available range (up to 100 blocks)
sample_size=100
if [[ "$current_block" -lt "$sample_size" ]]; then
  sample_size="$current_block"
fi

if [[ "$sample_size" -lt 2 ]]; then
  _assert_fail "block-time: not enough blocks to measure (only $current_block available)"
  test_result
  exit 1
fi

printf '[INFO]  Sampling %d blocks ending at block %d...\n' \
  "$sample_size" "$current_block" >&2

start_block=$(( current_block - sample_size + 1 ))

# Fetch timestamps for all sampled blocks in a single Python call
stats_json=$(python3 -c "
import json, urllib.request, sys

with open('${CHAINBENCH_DIR}/state/pids.json') as f:
    state = json.load(f)

port = None
for nid, info in state.get('nodes', {}).items():
    if info.get('status') == 'running':
        port = info['http_port']
        break

if port is None:
    print(json.dumps({'error': 'no_running_nodes'}))
    sys.exit(0)

start_block  = int(sys.argv[1])
end_block    = int(sys.argv[2])

timestamps = []
for blk in range(start_block, end_block + 1):
    hex_blk = hex(blk)
    payload = json.dumps({
        'jsonrpc': '2.0',
        'method': 'eth_getBlockByNumber',
        'params': [hex_blk, False],
        'id': 1,
    }).encode()
    req = urllib.request.Request(
        f'http://127.0.0.1:{port}',
        data=payload,
        headers={'Content-Type': 'application/json'},
    )
    try:
        resp = urllib.request.urlopen(req, timeout=5)
        data = json.loads(resp.read())
        result = data.get('result') or {}
        ts_hex = result.get('timestamp', '')
        if ts_hex:
            timestamps.append(int(ts_hex, 16))
    except Exception:
        pass

if len(timestamps) < 2:
    print(json.dumps({'error': 'insufficient_data', 'count': len(timestamps)}))
    sys.exit(0)

timestamps.sort()
diffs = [timestamps[i+1] - timestamps[i] for i in range(len(timestamps) - 1)]
diffs = [d for d in diffs if d > 0]  # filter duplicates / out-of-order

if not diffs:
    print(json.dumps({'error': 'no_positive_diffs'}))
    sys.exit(0)

diffs_sorted = sorted(diffs)
n = len(diffs_sorted)
avg  = sum(diffs_sorted) / n
mn   = diffs_sorted[0]
mx   = diffs_sorted[-1]
p95_idx = int(n * 0.95)
p95  = diffs_sorted[min(p95_idx, n - 1)]

print(json.dumps({
    'blocks_sampled': end_block - start_block + 1,
    'diffs_counted':  n,
    'avg_s':  round(avg, 3),
    'min_s':  mn,
    'max_s':  mx,
    'p95_s':  p95,
}))
" "$start_block" "$current_block")

printf '[INFO]  Block time stats: %s\n' "$stats_json" >&2

# Parse error
has_error=$(python3 -c "
import json, sys
d = json.loads(sys.argv[1])
print('true' if 'error' in d else 'false')
" "$stats_json")

if [[ "$has_error" == "true" ]]; then
  error_msg=$(python3 -c "
import json, sys
print(json.loads(sys.argv[1]).get('error', 'unknown'))
" "$stats_json")
  _assert_fail "block-time: failed to collect stats — $error_msg"
  test_result
  exit 1
fi

avg_s=$(python3 -c "
import json, sys
print(json.loads(sys.argv[1])['avg_s'])
" "$stats_json")

min_s=$(python3 -c "
import json, sys
print(json.loads(sys.argv[1])['min_s'])
" "$stats_json")

max_s=$(python3 -c "
import json, sys
print(json.loads(sys.argv[1])['max_s'])
" "$stats_json")

p95_s=$(python3 -c "
import json, sys
print(json.loads(sys.argv[1])['p95_s'])
" "$stats_json")

diffs_counted=$(python3 -c "
import json, sys
print(json.loads(sys.argv[1])['diffs_counted'])
" "$stats_json")

printf '[INFO]  Block time over %s intervals:\n' "$diffs_counted" >&2
printf '[INFO]    avg=%ss  min=%ss  max=%ss  p95=%ss\n' \
  "$avg_s" "$min_s" "$max_s" "$p95_s" >&2

# Sanity assertions:
# avg block time should be > 0 and < 60 seconds for a healthy chain
assert_gt "$diffs_counted" "0" "at least 1 block interval measured"

avg_int=$(python3 -c "print(int(float('$avg_s')))" 2>/dev/null || echo "0")
assert_gt "$avg_int" "0" "average block time > 0s"
assert_ge "60" "$avg_int" "average block time <= 60s (got ${avg_s}s)"

# p95 should be < 120 seconds
p95_int=$(python3 -c "print(int(float('$p95_s')))" 2>/dev/null || echo "0")
assert_ge "120" "$p95_int" "p95 block time <= 120s (got ${p95_s}s)"

test_result
