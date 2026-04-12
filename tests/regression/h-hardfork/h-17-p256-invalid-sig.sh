#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-1-2-04
# name: P-256 precompile rejects invalid signature
# category: regression/h-hardfork
# tags: [hardfork, boho, precompile, secp256r1]
# estimated_seconds: 15
# preconditions:
#   chain_running: true
# depends_on: []
# ---end-meta---
# TC-1-2-04 — Invalid P-256 signature must NOT return success
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-17-p256-invalid-sig"
check_env || { test_result; exit 1; }

P256_PRECOMPILE="0x0000000000000000000000000000000000000100"
SUCCESS_RESULT="0x0000000000000000000000000000000000000000000000000000000000000001"

# Corrupted r value (first byte 29->ff)
HASH="bb5a52f42f9c9261ed4361f59422a1e30036e7c32b270c8807a419feca605023"
R_BAD="ff27b10512bae3eddcfe467828128bad2903269919f7086069c8c4df6c732838"
S_VAL="c7787964eaac00e5921fb1498a60f4606766b3d9685001558d1a974e7341513e"
X_VAL="04e04e18e1ff7b70e7b5e14d1b70e0bdb8ece3acf34ffee3e8e5a2e4266bfbb0"
Y_VAL="f6afd7ebfa4dfddd60ab0272c226d19c1f6aed1cdee3a51a35e415f4dcc33d70"

INPUT="0x${HASH}${R_BAD}${S_VAL}${X_VAL}${Y_VAL}"

resp=$(rpc 1 "eth_call" "[{\"to\":\"${P256_PRECOMPILE}\",\"data\":\"${INPUT}\"},\"latest\"]")
result=$(json_get "$resp" "result")

observe "p256_invalid_result" "$result"

if [[ "$result" == "$SUCCESS_RESULT" ]]; then
  _assert_fail "P-256 returned success for invalid signature"
else
  _assert_pass "P-256 correctly rejected invalid signature (result: $result)"
fi

test_result
