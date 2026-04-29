#!/usr/bin/env bash
# Test: fault/node-crash
# Description: Stop 1 validator and verify consensus continues with 3/4
# ---chainbench-meta---
# description: Stop 1 validator and verify consensus continues with 3/4
# requires_capabilities: [process]
# chain_compat: [stablenet, wbft]
# ---end-meta---
set -euo pipefail

source "$(dirname "$0")/../lib/rpc.sh"
source "$(dirname "$0")/../lib/assert.sh"
source "$(dirname "$0")/../lib/wait.sh"

test_start "fault/node-crash"

# Verify sync before stopping node 3
sync_before=$(check_sync)
synced_before=$(python3 -c "
import json, sys
d = json.loads(sys.argv[1])
print('true' if d.get('synced') else 'false')
" "$sync_before")
assert_true "$synced_before" "nodes are synced before test"

# Record current block on node 1
block_before=$(block_number "1")
printf '[INFO]  Block before node-crash: %s\n' "$block_before" >&2

# Stop node 3
printf '[INFO]  Stopping node 3...\n' >&2
"${CHAINBENCH_DIR}/chainbench.sh" node stop 3 --quiet
printf '[INFO]  Node 3 stopped\n' >&2

# Wait for 10 new blocks on node 1 (proves 3/4 consensus holds)
target_block=$(( block_before + 10 ))
printf '[INFO]  Waiting for block %d to confirm 3/4 consensus...\n' "$target_block" >&2

reached_block=$(wait_for_block "1" "$target_block" 60)

if [[ "$reached_block" == "timeout" ]]; then
  _assert_fail "node-crash: consensus stalled — 3/4 validators could not produce 10 blocks"
  # Attempt recovery before exiting
  "${CHAINBENCH_DIR}/chainbench.sh" node start 3 --quiet 2>/dev/null || true
  test_result
  exit 1
fi

assert_ge "$reached_block" "$target_block" "3/4 validators produced 10 new blocks (reached $reached_block)"

# Restart node 3
printf '[INFO]  Restarting node 3...\n' >&2
"${CHAINBENCH_DIR}/chainbench.sh" node start 3 --quiet

# Wait for sync (node 3 catches up)
printf '[INFO]  Waiting for all nodes to sync...\n' >&2
sync_result=$(wait_for_sync 90)

assert_eq "$sync_result" "synced" "node 3 caught up after restart"

# Final sync check
sync_after=$(check_sync)
synced_after=$(python3 -c "
import json, sys
d = json.loads(sys.argv[1])
print('true' if d.get('synced') else 'false')
" "$sync_after")
diff_after=$(python3 -c "
import json, sys
d = json.loads(sys.argv[1])
print(d.get('diff', 999))
" "$sync_after")

assert_true "$synced_after" "all nodes synced after node 3 recovery"
assert_ge "2" "$diff_after" "block diff <= 2 after recovery (got $diff_after)"

test_result
