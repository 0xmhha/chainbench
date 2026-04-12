#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-1-2-01
# name: secp256r1 (P-256) precompile verification
# category: regression/h-hardfork
# tags: [hardfork, boho, precompile, secp256r1, p256]
# estimated_seconds: 15
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/h-hardfork/h-02-secp256r1-precompile
# TC-1-2-01/03 — Verify secp256r1 precompile at 0x0000...0100
#   eth_call with a valid NIST P-256 test vector should return 1 (valid signature)
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-02-secp256r1-precompile"
check_env || { test_result; exit 1; }

# secp256r1 precompile address (RIP-7212)
readonly P256_PRECOMPILE="0x0000000000000000000000000000000000000100"

# NIST P-256 test vector
# Input: hash || r || s || x || y  (each 32 bytes, total 160 bytes)
readonly HASH="bb5a52f42f9c9261ed4361f59422a1e30036e7c32b270c8807a419feca605023"
readonly R_VAL="2927b10512bae3eddcfe467828128bad2903269919f7086069c8c4df6c732838"
readonly S_VAL="c7787964eaac00e5921fb1498a60f4606766b3d9685001558d1a974e7341513e"
readonly X_VAL="04e04e18e1ff7b70e7b5e14d1b70e0bdb8ece3acf34ffee3e8e5a2e4266bfbb0"
readonly Y_VAL="f6afd7ebfa4dfddd60ab0272c226d19c1f6aed1cdee3a51a35e415f4dcc33d70"

INPUT_DATA="0x${HASH}${R_VAL}${S_VAL}${X_VAL}${Y_VAL}"
observe "input_length_bytes" "160"

# Expected: 0x0000000000000000000000000000000000000000000000000000000000000001
readonly EXPECTED_RESULT="0x0000000000000000000000000000000000000000000000000000000000000001"

# --- Test 1: eth_call to P-256 precompile with valid signature ---
printf '[INFO]  calling P-256 precompile at %s\n' "$P256_PRECOMPILE" >&2
resp=$(rpc "1" "eth_call" "[{\"to\":\"${P256_PRECOMPILE}\",\"data\":\"${INPUT_DATA}\"}, \"latest\"]")
result=$(json_get "$resp" "result")
error=$(json_get "$resp" "error.message")

observe "p256_result" "$result"
observe "p256_error" "$error"

assert_not_empty "$result" "P-256 precompile returned a result"
assert_eq "$result" "$EXPECTED_RESULT" "P-256 signature verification returned 1 (valid)"

# --- Test 2: Verify precompile has code (exists as precompile) ---
# Precompiles may or may not report code via eth_getCode depending on implementation,
# but eth_call should not return an error
if [[ -n "$error" && "$error" != "" ]]; then
  _assert_fail "P-256 eth_call returned error: $error"
else
  _assert_pass "P-256 eth_call completed without error"
fi

test_result
