#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-4-1-01
# name: Boho hardfork chain config verification
# category: regression/h-hardfork
# tags: [hardfork, boho, config, chain]
# estimated_seconds: 15
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/h-hardfork/h-01-chain-config-boho
# TC-4-1-01 — Verify chain starts with bohoBlock=0 config
#   1) Nodes are running (block number > 0 and increasing)
#   2) Block period ~1 second (timestamp diff between consecutive blocks)
#   3) istanbul_getWbftExtraInfo returns epoch info
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-01-chain-config-boho"
check_env || { test_result; exit 1; }

# --- 1) Verify nodes are running and producing blocks ---
block1_hex=$(rpc "1" "eth_blockNumber" "[]" | json_get - result)
block1_dec=$(hex_to_dec "$block1_hex")
observe "block_number_first" "$block1_dec"
assert_gt "$block1_dec" "0" "node 1 block number > 0 (chain is running)"

# Wait a short time for a new block
sleep 2

block2_hex=$(rpc "1" "eth_blockNumber" "[]" | json_get - result)
block2_dec=$(hex_to_dec "$block2_hex")
observe "block_number_second" "$block2_dec"
assert_gt "$block2_dec" "$block1_dec" "block number is increasing (chain is producing blocks)"

# --- 2) Check block period ~1 second ---
# Fetch two consecutive blocks and compare their timestamps
blk_a_resp=$(rpc "1" "eth_getBlockByNumber" "[\"${block1_hex}\", false]")
blk_a_ts_hex=$(json_get "$blk_a_resp" "result.timestamp")
blk_a_ts=$(hex_to_dec "$blk_a_ts_hex")

# Next block after block1
next_num=$(( block1_dec + 1 ))
next_hex=$(printf '0x%x' "$next_num")
blk_b_resp=$(rpc "1" "eth_getBlockByNumber" "[\"${next_hex}\", false]")
blk_b_ts_hex=$(json_get "$blk_b_resp" "result.timestamp")
blk_b_ts=$(hex_to_dec "$blk_b_ts_hex")

ts_diff=$(( blk_b_ts - blk_a_ts ))
observe "block_period_seconds" "$ts_diff"
printf '[INFO]  block %d ts=%s, block %d ts=%s, diff=%ds\n' \
  "$block1_dec" "$blk_a_ts" "$next_num" "$blk_b_ts" "$ts_diff" >&2

# Block period should be between 0 and 3 seconds (target ~1s, allow jitter)
assert_ge "$ts_diff" "0" "block period >= 0s"
assert_ge "3" "$ts_diff" "block period <= 3s (expected ~1s)"

# --- 3) Check istanbul_getWbftExtraInfo returns epoch info ---
extra_resp=$(get_wbft_extra_json "1")
extra_result=$(json_get "$extra_resp" "result")
assert_not_empty "$extra_result" "istanbul_getWbftExtraInfo returns result"

# Verify epoch field exists in the response
epoch=$(json_get "$extra_resp" "result.epoch")
observe "wbft_epoch" "$epoch"
printf '[INFO]  WbftExtraInfo epoch = %s\n' "$epoch" >&2
assert_not_empty "$epoch" "epoch field exists in WbftExtraInfo"

test_result
