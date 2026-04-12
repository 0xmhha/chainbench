#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-1-2-05
# name: P-256 precompile handles short input gracefully
# category: regression/h-hardfork
# tags: [hardfork, boho, precompile, secp256r1]
# estimated_seconds: 15
# preconditions:
#   chain_running: true
# depends_on: []
# ---end-meta---
# TC-1-2-05 — Input < 160 bytes should not return success
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-18-p256-short-input"
check_env || { test_result; exit 1; }

P256_PRECOMPILE="0x0000000000000000000000000000000000000100"
SUCCESS_RESULT="0x0000000000000000000000000000000000000000000000000000000000000001"

# Only 64 bytes (hash + r), missing s, x, y
SHORT_INPUT="0xbb5a52f42f9c9261ed4361f59422a1e30036e7c32b270c8807a419feca6050232927b10512bae3eddcfe467828128bad2903269919f7086069c8c4df6c732838"

resp=$(rpc 1 "eth_call" "[{\"to\":\"${P256_PRECOMPILE}\",\"data\":\"${SHORT_INPUT}\"},\"latest\"]")
result=$(json_get "$resp" "result")
error=$(json_get "$resp" "error.message")

observe "p256_short_result" "$result"

if [[ -n "$error" && "$error" != "null" && "$error" != "" ]]; then
  _assert_pass "P-256 returned error for short input: $error"
elif [[ -z "$result" || "$result" == "0x" || "$result" == "null" ]]; then
  _assert_pass "P-256 returned empty for short input"
elif [[ "$result" == "$SUCCESS_RESULT" ]]; then
  _assert_fail "P-256 returned success for 64-byte input (needs 160)"
else
  _assert_pass "P-256 returned non-success for short input"
fi

test_result
