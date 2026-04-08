#!/usr/bin/env bash
# Test: regression/a-ethereum/a4-02-eth-get-balance
# RT-A-4-02 — eth_getBalance 정상 조회
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a4-02-eth-get-balance"

# Validator 계정은 genesis alloc에 1e27 wei
bal=$(rpc "1" "eth_getBalance" "[\"${VALIDATOR_1_ADDR}\", \"latest\"]" | json_get - result)
assert_contains "$bal" "0x" "balance returned as hex"
bal_dec=$(hex_to_dec "$bal")
assert_gt "$bal_dec" "0" "validator1 balance > 0"

# TEST_ACC_A alloc: 1e27 wei
test_bal=$(rpc "1" "eth_getBalance" "[\"${TEST_ACC_A_ADDR}\", \"latest\"]" | json_get - result)
test_bal_dec=$(hex_to_dec "$test_bal")
assert_gt "$test_bal_dec" "0" "TEST_ACC_A balance > 0"

# 존재하지 않는 계정 (0x00…01)은 0
zero_bal=$(rpc "1" "eth_getBalance" "[\"0x0000000000000000000000000000000000000001\", \"latest\"]" | json_get - result)
zero_bal_dec=$(hex_to_dec "$zero_bal")
assert_eq "$zero_bal_dec" "0" "non-existent address balance == 0"

test_result
