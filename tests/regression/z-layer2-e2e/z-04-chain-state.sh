#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-Z-04
# name: Layer 2 chain state queries
# category: regression/z-layer2-e2e
# tags: [layer2, state, cast]
# estimated_seconds: 10
# preconditions:
#   chain_running: true
# ---end-meta---
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"
source "$(dirname "$0")/../../lib/chain_state.sh"

test_start "z-04-chain-state"

check_env || { test_result; exit 1; }

# Test: cb_get_chain_id
chain_id=$(cb_get_chain_id "1")
assert_not_empty "$chain_id" "got chain ID"
assert_eq "$chain_id" "8283" "chain ID is 8283"

# Test: cb_get_balance — validator should have non-zero balance
balance=$(cb_get_balance "1" "$VALIDATOR_1_ADDR")
assert_not_empty "$balance" "got validator balance"
assert_gt "$balance" "0" "validator balance is non-zero"

# Test: cb_get_balance — test account A should have non-zero balance
balance_a=$(cb_get_balance "1" "$TEST_ACC_A_ADDR")
assert_not_empty "$balance_a" "got TEST_ACC_A balance"
assert_gt "$balance_a" "0" "TEST_ACC_A balance is non-zero"

# Test: cb_get_nonce
nonce=$(cb_get_nonce "1" "$VALIDATOR_1_ADDR")
assert_not_empty "$nonce" "got nonce for validator"

# Test: cb_get_base_fee
base_fee=$(cb_get_base_fee "1")
assert_not_empty "$base_fee" "got base fee"
assert_gt "$base_fee" "0" "base fee is non-zero"

# Test: cb_get_block (latest)
block_json=$(cb_get_block "1" "latest")
assert_not_empty "$block_json" "got latest block JSON"

block_number=$(printf '%s' "$block_json" | python3 -c "import sys,json; d=json.loads(sys.stdin.read()); print(d.get('number',''))")
assert_not_empty "$block_number" "block JSON has number field"

# Test: cb_get_block_field
gas_limit=$(cb_get_block_field "1" "gasLimit" "latest")
assert_not_empty "$gas_limit" "got gasLimit from block"

# Test: cb_is_authorized — test account A should not be authorized by default
is_auth=$(cb_is_authorized "1" "$TEST_ACC_A_ADDR")
assert_not_empty "$is_auth" "got is_authorized result"

# Test: cb_is_blacklisted — test account A should not be blacklisted
is_bl=$(cb_is_blacklisted "1" "$TEST_ACC_A_ADDR")
assert_eq "$is_bl" "false" "TEST_ACC_A is not blacklisted"

test_result
