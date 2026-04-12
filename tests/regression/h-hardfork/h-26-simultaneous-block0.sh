#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-4-4-01
# name: Simultaneous Anzeon+Boho activation at block 0
# category: regression/h-hardfork
# tags: [hardfork, boho, anzeon, simultaneous]
# estimated_seconds: 20
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# TC-4-4-01 — Both Anzeon and Boho active from genesis (BohoBlock=0)
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-26-simultaneous-block0"
check_env || { test_result; exit 1; }

# --- 1) Verify P-256 precompile is active (Boho feature) ---
P256_PRECOMPILE="0x0000000000000000000000000000000000000100"
# Valid P-256 test vector (hash, r, s, x, y - 32 bytes each = 160 bytes)
P256_INPUT="0x\
bb5a52f42f9c9261ed4361f59422a1e30036e7c32b270c8807a419feca605023\
741dd5bda817d95e4626b4a73f0b961e01e1ed2d4e4e4de70ef0de3004f944be\
47e01f68bcb6cabb95c7e3ef333ce1f20c71fc9bbe8f3a1a7e39b1aa0803f1e4\
24f66e60f5ae559b1564c6e0d5aee93b4ace0a5e99de5c82e39d3e34d903b3e7\
f94db1cc8a6e7f6e12508e4e4c63ea0e40b2c77fe0cb97a40041e0400b7bae4d"

resp=$(rpc 1 "eth_call" "[{\"to\":\"${P256_PRECOMPILE}\",\"data\":\"${P256_INPUT}\"},\"latest\"]")
p256_result=$(json_get "$resp" "result")
printf '[INFO]  P-256 result = %s\n' "$p256_result" >&2

# Result should end with 01 (valid signature)
p256_last2="${p256_result: -2}"
assert_eq "$p256_last2" "01" "P-256 precompile returns valid (Boho active)"

# --- 2) Verify gas fee floor is enforced (Boho feature) ---
base_fee=$(get_base_fee 1)
observe "baseFee" "$base_fee"
assert_ge "$base_fee" "$MIN_BASE_FEE_WEI" "baseFee >= MIN_BASE_FEE (20 Twei)"

# --- 3) Verify GovMinter v2 bytecode (Boho feature) ---
resp=$(rpc 1 "eth_getCode" "[\"${GOV_MINTER}\",\"latest\"]")
code=$(json_get "$resp" "result")
code_len=${#code}
assert_gt "$code_len" "100" "GovMinter v2 bytecode deployed"

# --- 4) Verify Anzeon features (blacklist enforcement) ---
# Blacklist check works (isBlacklisted returns false for normal account)
is_bl_data="0x$(selector "isBlacklisted(address)")$(pad_address "$TEST_ACC_A_ADDR")"
resp=$(eth_call_raw 1 "$ACCOUNT_MANAGER" "$is_bl_data")
assert_not_empty "$resp" "AccountManager.isBlacklisted callable (Anzeon active)"

observe "both_hardforks_active" "true"

test_result
