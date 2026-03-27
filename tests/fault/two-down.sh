#!/usr/bin/env bash
# Test: fault/two-down
# Description: Stop 2/4 validators - consensus should halt, recover when 1 returns
set -euo pipefail

source "$(dirname "$0")/../lib/rpc.sh"
source "$(dirname "$0")/../lib/assert.sh"
source "$(dirname "$0")/../lib/wait.sh"

test_start "fault/two-down"

# Record current block on node 1 before stopping
block_before=$(block_number "1")
printf '[INFO]  Block before two-down: %s\n' "$block_before" >&2

# Stop node 3 and node 4
printf '[INFO]  Stopping nodes 3 and 4...\n' >&2
"${CHAINBENCH_DIR}/chainbench.sh" node stop 3 --quiet
"${CHAINBENCH_DIR}/chainbench.sh" node stop 4 --quiet
printf '[INFO]  Nodes 3 and 4 stopped\n' >&2

# Wait 10 seconds for any in-flight consensus to settle
wait_seconds 10 "Waiting 10s to confirm consensus halt with 2/4 nodes down"

# Check block number has NOT advanced (consensus should be halted with only 2/4)
block_after_wait=$(block_number "1" 2>/dev/null || echo "$block_before")
block_advance=$(( block_after_wait - block_before ))

printf '[INFO]  Block before: %s  Block after 10s: %s  Advance: %s\n' \
  "$block_before" "$block_after_wait" "$block_advance" >&2

# With BFT requiring 3/4 (ceil(2/3 * 4)+1 = 3), 2 validators cannot produce blocks.
# Allow up to 1 block in-flight at the moment of stopping.
assert_ge "1" "$block_advance" "consensus halted with 2/4 validators down (advance=$block_advance)"

# Restart node 3 (now 3/4 validators active — BFT threshold met)
printf '[INFO]  Restarting node 3 to restore 3/4 validators...\n' >&2
"${CHAINBENCH_DIR}/chainbench.sh" node start 3 --quiet

# Wait for new blocks (consensus should resume)
block_at_restart=$(block_number "1" 2>/dev/null || echo "$block_after_wait")
target_block=$(( block_at_restart + 5 ))

printf '[INFO]  Waiting for block %d to confirm consensus resumed...\n' "$target_block" >&2
reached_block=$(wait_for_block "1" "$target_block" 60)

if [[ "$reached_block" == "timeout" ]]; then
  _assert_fail "two-down: consensus did not resume after restarting node 3"
  # Attempt to restore node 4
  "${CHAINBENCH_DIR}/chainbench.sh" node start 4 --quiet 2>/dev/null || true
  test_result
  exit 1
fi

assert_ge "$reached_block" "$target_block" "consensus resumed after node 3 restart (reached $reached_block)"

# Confirm block advanced relative to the halted period
final_block=$(block_number "1")
total_advance=$(( final_block - block_before ))
assert_gt "$total_advance" "0" "overall block height advanced during test ($total_advance new blocks)"

# Restart node 4 to restore full cluster
printf '[INFO]  Restarting node 4 to restore full cluster...\n' >&2
"${CHAINBENCH_DIR}/chainbench.sh" node start 4 --quiet 2>/dev/null || true

test_result
