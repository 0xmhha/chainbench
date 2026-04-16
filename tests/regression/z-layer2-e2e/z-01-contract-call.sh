#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-Z-01
# name: Layer 2 contract call via cast
# category: regression/z-layer2-e2e
# tags: [layer2, contract, cast]
# estimated_seconds: 10
# preconditions:
#   chain_running: true
# ---end-meta---
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"
source "$(dirname "$0")/../../lib/contract.sh"
source "$(dirname "$0")/../../lib/chain_state.sh"

test_start "z-01-contract-call"

check_env || { test_result; exit 1; }

# Test: cb_eth_call to query balanceOf on NativeCoinAdapter
balance=$(cb_eth_call "1" "$SC_NATIVE_COIN_ADAPTER" "balanceOf(address)(uint256)" "$VALIDATOR_1_ADDR")
assert_not_empty "$balance" "balanceOf returned a value"
assert_gt "$balance" "0" "validator has non-zero balance"

# Test: cb_selector matches known value
sel=$(cb_selector "balanceOf(address)")
assert_eq "$sel" "0x70a08231" "balanceOf selector correct"

# Test: cb_abi_encode produces valid calldata
encoded=$(cb_abi_encode "balanceOf(address)" "$VALIDATOR_1_ADDR")
assert_contains "$encoded" "70a08231" "encoded calldata includes selector"

# Test: cb_eth_call for totalSupply
total_supply=$(cb_eth_call "1" "$SC_NATIVE_COIN_ADAPTER" "totalSupply()(uint256)")
assert_not_empty "$total_supply" "totalSupply returned a value"
assert_gt "$total_supply" "0" "totalSupply is non-zero"

# Test: cb_eth_call for name
token_name=$(cb_eth_call "1" "$SC_NATIVE_COIN_ADAPTER" "name()(string)")
assert_not_empty "$token_name" "name() returned a value"

# Test: cb_eth_call for decimals
decimals=$(cb_eth_call "1" "$SC_NATIVE_COIN_ADAPTER" "decimals()(uint8)")
assert_not_empty "$decimals" "decimals() returned a value"

test_result
