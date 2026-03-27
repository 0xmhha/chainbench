#!/usr/bin/env bash
# Test: fault/node-recover
# Description: Stop a node, wait, restart, and measure sync time
set -euo pipefail

source "$(dirname "$0")/../lib/rpc.sh"
source "$(dirname "$0")/../lib/assert.sh"
source "$(dirname "$0")/../lib/wait.sh"

test_start "fault/node-recover"

# Record starting block on node 1
block_at_start=$(block_number "1")
printf '[INFO]  Block at test start: %s\n' "$block_at_start" >&2

# Stop node 3
printf '[INFO]  Stopping node 3...\n' >&2
"${CHAINBENCH_DIR}/chainbench.sh" node stop 3 --quiet
printf '[INFO]  Node 3 stopped\n' >&2

# Wait for 20 blocks on remaining nodes
target_block=$(( block_at_start + 20 ))
printf '[INFO]  Waiting for remaining nodes to reach block %d...\n' "$target_block" >&2

reached=$(wait_for_block "1" "$target_block" 90)

if [[ "$reached" == "timeout" ]]; then
  _assert_fail "node-recover: remaining nodes stalled before restart test"
  "${CHAINBENCH_DIR}/chainbench.sh" node start 3 --quiet 2>/dev/null || true
  test_result
  exit 1
fi

assert_ge "$reached" "$target_block" "remaining nodes reached block $target_block while node 3 was down"

# Record block before restart
block_before_restart=$(block_number "1")
printf '[INFO]  Block just before node 3 restart: %s\n' "$block_before_restart" >&2

# Start time measurement
recovery_start=$(date +%s)

# Restart node 3
printf '[INFO]  Restarting node 3...\n' >&2
"${CHAINBENCH_DIR}/chainbench.sh" node start 3 --quiet

# Poll until node 3 is within 2 blocks of node 1
printf '[INFO]  Polling until node 3 is within 2 blocks of node 1...\n' >&2

max_wait=120
elapsed=0
recovered=0

while [[ "$elapsed" -lt "$max_wait" ]]; do
  node1_block=$(block_number "1" 2>/dev/null || echo "0")
  node3_block=$(block_number "3" 2>/dev/null || echo "0")
  lag=$(( node1_block - node3_block ))

  printf '[INFO]  node1=%s node3=%s lag=%s\n' \
    "$node1_block" "$node3_block" "$lag" >&2

  if [[ "$lag" -le 2 && "$node3_block" -gt 0 ]]; then
    recovered=1
    break
  fi

  sleep 2
  elapsed=$(( elapsed + 2 ))
done

recovery_end=$(date +%s)
recovery_time=$(( recovery_end - recovery_start ))

printf '[INFO]  Node 3 recovery time: %ds\n' "$recovery_time" >&2

assert_eq "$recovered" "1" "node 3 recovered within ${max_wait}s (actual: ${recovery_time}s)"

if [[ "$recovered" -eq 1 ]]; then
  final_node1=$(block_number "1" 2>/dev/null || echo "0")
  final_node3=$(block_number "3" 2>/dev/null || echo "0")
  final_lag=$(( final_node1 - final_node3 ))
  assert_ge "2" "$final_lag" "node 3 lag <= 2 blocks after recovery (lag=$final_lag)"
fi

test_result
