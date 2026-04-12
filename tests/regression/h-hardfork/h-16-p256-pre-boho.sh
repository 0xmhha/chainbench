#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-1-2-02
# name: P-256 precompile unavailable before Boho activation
# category: regression/h-hardfork
# tags: [hardfork, boho, precompile, secp256r1, p256]
# estimated_seconds: 15
# preconditions:
#   chain_running: true
#   profile: hardfork-boho-delayed
# depends_on: []
# ---end-meta---
# TC-1-2-02 — P-256 precompile should NOT work before BohoBlock
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-16-p256-pre-boho"
check_env || { test_result; exit 1; }

P256_PRECOMPILE="0x0000000000000000000000000000000000000100"
P256_INPUT="0xbb5a52f42f9c9261ed4361f59422a1e30036e7c32b270c8807a419feca6050232927b10512bae3eddcfe467828128bad2903269919f7086069c8c4df6c732838c7787964eaac00e5921fb1498a60f4606766b3d9685001558d1a974e7341513e04e04e18e1ff7b70e7b5e14d1b70e0bdb8ece3acf34ffee3e8e5a2e4266bfbb0f6afd7ebfa4dfddd60ab0272c226d19c1f6aed1cdee3a51a35e415f4dcc33d70"

SUCCESS_RESULT="0x0000000000000000000000000000000000000000000000000000000000000001"

# Call at block 5 (before BohoBlock=10)
resp=$(rpc 1 "eth_call" "[{\"to\":\"${P256_PRECOMPILE}\",\"data\":\"${P256_INPUT}\"},\"0x5\"]")
result=$(json_get "$resp" "result")
error=$(json_get "$resp" "error.message")

observe "p256_result_pre_boho" "$result"

if [[ -n "$error" && "$error" != "null" && "$error" != "" ]]; then
  _assert_pass "P-256 returned error before Boho (expected): $error"
elif [[ -z "$result" || "$result" == "0x" || "$result" == "null" ]]; then
  _assert_pass "P-256 returned empty before Boho (expected)"
elif [[ "$result" == "$SUCCESS_RESULT" ]]; then
  _assert_fail "P-256 returned success BEFORE Boho — precompile should not be active"
else
  _assert_pass "P-256 returned non-success before Boho: $result"
fi

test_result
