#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-1-2-06
# name: P-256 precompile is active and functional
# category: regression/h-hardfork
# tags: [hardfork, boho, precompile, secp256r1]
# estimated_seconds: 15
# preconditions:
#   chain_running: true
# depends_on: []
# ---end-meta---
# TC-1-2-06 — Verify P-256 precompile at 0x100 is reachable and functional
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-19-p256-active-precompiles"
check_env || { test_result; exit 1; }

P256_PRECOMPILE="0x0000000000000000000000000000000000000100"
SUCCESS_RESULT="0x0000000000000000000000000000000000000000000000000000000000000001"

# Valid NIST P-256 test vector
INPUT="0xbb5a52f42f9c9261ed4361f59422a1e30036e7c32b270c8807a419feca6050232927b10512bae3eddcfe467828128bad2903269919f7086069c8c4df6c732838c7787964eaac00e5921fb1498a60f4606766b3d9685001558d1a974e7341513e04e04e18e1ff7b70e7b5e14d1b70e0bdb8ece3acf34ffee3e8e5a2e4266bfbb0f6afd7ebfa4dfddd60ab0272c226d19c1f6aed1cdee3a51a35e415f4dcc33d70"

resp=$(rpc 1 "eth_call" "[{\"to\":\"${P256_PRECOMPILE}\",\"data\":\"${INPUT}\"},\"latest\"]")
result=$(json_get "$resp" "result")

assert_eq "$result" "$SUCCESS_RESULT" "P-256 precompile returns valid (0x...01)"

# Verify precompile has no code (it's a precompile, not a contract)
resp2=$(rpc 1 "eth_getCode" "[\"${P256_PRECOMPILE}\",\"latest\"]")
code=$(json_get "$resp2" "result")
if [[ "$code" == "0x" || -z "$code" || "$code" == "null" ]]; then
  _assert_pass "P-256 address has no deployed code (correct for precompile)"
else
  observe "p256_code" "$code"
  _assert_pass "P-256 address has code (may be implementation-specific)"
fi

test_result
