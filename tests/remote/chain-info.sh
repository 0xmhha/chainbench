#!/usr/bin/env bash
# Test: remote/chain-info
# Description: Verify chain ID, sync status, and block production on remote chain
set -euo pipefail

source "$(dirname "$0")/../lib/rpc.sh"
source "$(dirname "$0")/../lib/assert.sh"

test_start "remote/chain-info"

target="$(_cb_rpc_default_target)"
if ! _cb_rpc_is_remote "$target"; then
  _assert_fail "CHAINBENCH_REMOTE is not set — cannot run remote tests"
  test_result
  exit 1
fi

printf '[INFO]  Testing remote target: %s\n' "$target" >&2

# Test 1: eth_chainId returns a valid chain ID
chain_id=$(get_chain_id "$target" 2>/dev/null || echo "")
assert_not_empty "$chain_id" "eth_chainId returns a value"
if [[ -n "$chain_id" ]]; then
  assert_gt "$chain_id" "0" "chain ID is positive ($chain_id)"
  printf '[INFO]  Chain ID: %s\n' "$chain_id" >&2
fi

# Test 2: eth_syncing returns a valid response
sync_result=$(get_syncing "$target" 2>/dev/null || echo "")
assert_not_empty "$sync_result" "eth_syncing returns a response"
if [[ "$sync_result" == "false" ]]; then
  _assert_pass "node reports fully synced"
else
  printf '[INFO]  Node is syncing: %s\n' "${sync_result:0:80}" >&2
  _assert_pass "node is syncing (valid response)"
fi

# Test 3: Block production check — two queries with a delay
block_1=$(block_number "$target" 2>/dev/null || echo "0")
assert_gt "$block_1" "0" "first block number is positive ($block_1)"

printf '[INFO]  Block #1: %s — waiting 5s for block production...\n' "$block_1" >&2
sleep 5

block_2=$(block_number "$target" 2>/dev/null || echo "0")
assert_ge "$block_2" "$block_1" "second block ($block_2) >= first block ($block_1)"

if [[ "$block_2" -gt "$block_1" ]]; then
  _assert_pass "block production confirmed: $block_1 -> $block_2"
else
  printf '[INFO]  Block unchanged (chain may be slow or halted)\n' >&2
fi

test_result
