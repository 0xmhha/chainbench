#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-Z-05
# name: Layer 2 fee delegation tx
# category: regression/z-layer2-e2e
# tags: [layer2, fee-delegation, custom-tx]
# estimated_seconds: 20
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# ---end-meta---
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"
source "$(dirname "$0")/../../lib/tx_builder.sh"
source "$(dirname "$0")/../../lib/chain_state.sh"

test_start "z-05-fee-delegate"

check_env || { test_result; exit 1; }

# Record fee payer balance before (C pays gas)
bal_c_before=$(cb_get_balance "1" "$TEST_ACC_C_ADDR")
assert_not_empty "$bal_c_before" "got TEST_ACC_C balance before fee delegation tx"

# Record recipient balance before
bal_b_before=$(cb_get_balance "1" "$TEST_ACC_B_ADDR")
assert_not_empty "$bal_b_before" "got TEST_ACC_B balance before fee delegation tx"

# Send Fee Delegation tx: A sends 1 wei to B, C pays gas (type 0x16)
result=$(cb_send_fee_delegate "1" "$TEST_ACC_A_PK" "$TEST_ACC_C_PK" "$TEST_ACC_B_ADDR" "1")
assert_not_empty "$result" "got result JSON from cb_send_fee_delegate"

# Extract tx hash
tx_hash=$(printf '%s' "$result" | python3 -c "import sys,json; print(json.loads(sys.stdin.read()).get('txHash',''))")
assert_not_empty "$tx_hash" "got tx hash from fee delegation result"
assert_contains "$tx_hash" "0x" "tx hash has 0x prefix"

# Wait for receipt
receipt=$(cb_wait_receipt "1" "$tx_hash" 30)
assert_not_empty "$receipt" "got receipt for fee delegation tx"

# Verify tx succeeded
status=$(cb_get_receipt_status "1" "$tx_hash")
assert_eq "$status" "0x1" "fee delegation tx succeeded"

# Verify recipient received the value
bal_b_after=$(cb_get_balance "1" "$TEST_ACC_B_ADDR")
assert_gt "$bal_b_after" "$bal_b_before" "TEST_ACC_B balance increased after fee delegation transfer"

# Verify fee payer (C) paid gas — balance should have decreased
bal_c_after=$(cb_get_balance "1" "$TEST_ACC_C_ADDR")
assert_not_empty "$bal_c_after" "got TEST_ACC_C balance after fee delegation tx"
# C's balance decreases because C paid gas (but didn't send value)
assert_true "$( python3 -c "import sys; sys.exit(0 if int('${bal_c_after}') < int('${bal_c_before}') else 1)" && echo "true" || echo "false" )" "TEST_ACC_C paid gas (balance decreased)"

test_result
