#!/usr/bin/env bash
# Test: regression/a-ethereum/a4-01-eth-block-number
# RT-A-4-01 — eth_blockNumber 단조 증가
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a4-01-eth-block-number"

b1=$(block_number "1")
sleep 2
b2=$(block_number "1")
sleep 2
b3=$(block_number "1")

printf '[INFO]  block samples: %s, %s, %s\n' "$b1" "$b2" "$b3" >&2

assert_gt "$b1" "0" "initial block number > 0"
assert_ge "$b2" "$b1" "b2 >= b1 (monotonic)"
assert_ge "$b3" "$b2" "b3 >= b2 (monotonic)"
assert_gt "$b3" "$b1" "b3 > b1 (blocks progressing)"

test_result
