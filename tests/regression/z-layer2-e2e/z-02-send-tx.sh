#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-Z-02
# name: Layer 2 send tx via cast
# category: regression/z-layer2-e2e
# tags: [layer2, tx, cast]
# estimated_seconds: 15
# preconditions:
#   chain_running: true
# ---end-meta---
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"
source "$(dirname "$0")/../../lib/tx_builder.sh"
source "$(dirname "$0")/../../lib/chain_state.sh"

test_start "z-02-send-tx"

check_env || { test_result; exit 1; }

# Get initial balance of receiver (decimal wei)
bal_before=$(cb_get_balance "1" "$TEST_ACC_B_ADDR")
assert_not_empty "$bal_before" "got initial balance of TEST_ACC_B"

# Send 1 wei via cb_send_tx (EIP-1559)
tx_hash=$(cb_send_tx "1" "$TEST_ACC_A_PK" "$TEST_ACC_B_ADDR" "1")
assert_not_empty "$tx_hash" "got tx hash from cb_send_tx"
assert_contains "$tx_hash" "0x" "tx hash has 0x prefix"

# Wait for receipt
receipt=$(cb_wait_receipt "1" "$tx_hash" 30)
assert_not_empty "$receipt" "got receipt from cb_wait_receipt"

# Check receipt status
status=$(cb_get_receipt_status "1" "$tx_hash")
assert_eq "$status" "0x1" "tx succeeded (status == 0x1)"

# Verify balance changed (B now has at least 1 wei more)
bal_after=$(cb_get_balance "1" "$TEST_ACC_B_ADDR")
assert_not_empty "$bal_after" "got balance after transfer"
assert_gt "$bal_after" "$bal_before" "TEST_ACC_B balance increased after transfer"

# Test: cb_send_tx with data field (send to self, no-op data)
noop_data="0xdeadbeef"
tx_hash2=$(cb_send_tx "1" "$TEST_ACC_A_PK" "$TEST_ACC_A_ADDR" "0" "$noop_data")
assert_not_empty "$tx_hash2" "got tx hash for tx with data"
status2=$(cb_get_receipt_status "1" "$tx_hash2")
assert_eq "$status2" "0x1" "data tx succeeded"

test_result
