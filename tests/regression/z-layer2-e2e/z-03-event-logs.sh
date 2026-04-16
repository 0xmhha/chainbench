#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-Z-03
# name: Layer 2 event log parsing
# category: regression/z-layer2-e2e
# tags: [layer2, event, cast]
# estimated_seconds: 15
# preconditions:
#   chain_running: true
# ---end-meta---
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"
source "$(dirname "$0")/../../lib/tx_builder.sh"
source "$(dirname "$0")/../../lib/event.sh"

test_start "z-03-event-logs"

check_env || { test_result; exit 1; }

# Test: cb_topic_hash for Transfer event matches known value
transfer_topic=$(cb_topic_hash "Transfer(address,address,uint256)")
assert_eq "$transfer_topic" "$EVENT_TRANSFER" "Transfer event topic hash correct"

# Send a Transfer on NativeCoinAdapter (emits Transfer event)
# transfer(address,uint256) — send 1000000000000000 wei (0.001 ether) to TEST_ACC_B
transfer_data=$(cb_abi_encode "transfer(address,uint256)" "$TEST_ACC_B_ADDR" "1000000000000000")
tx_hash=$(cb_send_tx "1" "$TEST_ACC_A_PK" "$SC_NATIVE_COIN_ADAPTER" "0" "$transfer_data" "100000")
assert_not_empty "$tx_hash" "got tx hash for transfer call"

receipt=$(cb_wait_receipt "1" "$tx_hash" 30)
assert_not_empty "$receipt" "got receipt after transfer"

status=$(printf '%s' "$receipt" | python3 -c "import sys,json; print(json.loads(sys.stdin.read()).get('status',''))")
assert_eq "$status" "0x1" "transfer tx succeeded"

# Get logs from receipt
logs=$(cb_get_receipt_logs "1" "$tx_hash")
assert_not_empty "$logs" "receipt has logs"

log_count=$(printf '%s' "$logs" | python3 -c "import sys,json; print(len(json.loads(sys.stdin.read())))")
assert_gt "$log_count" "0" "receipt has at least one log"

# Find Transfer event log in receipt
transfer_log=$(cb_find_log "$receipt" "$SC_NATIVE_COIN_ADAPTER" "$EVENT_TRANSFER")
assert_not_empty "$transfer_log" "found Transfer event log in receipt"

# Verify the log has expected fields
log_topics=$(printf '%s' "$transfer_log" | python3 -c "import sys,json; d=json.loads(sys.stdin.read()); print(len(d.get('topics',[])))")
assert_gt "$log_topics" "0" "Transfer log has topics"

# Test: cb_get_logs — query Transfer events from block 0
from_block_logs=$(cb_get_logs "1" "$SC_NATIVE_COIN_ADAPTER" "$EVENT_TRANSFER" "0x0" "latest")
assert_not_empty "$from_block_logs" "cb_get_logs returned results"

test_result
