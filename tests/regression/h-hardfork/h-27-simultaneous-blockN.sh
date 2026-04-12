#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-4-4-02
# name: Boho activates at block N while Anzeon already active
# category: regression/h-hardfork
# tags: [hardfork, boho, anzeon, delayed]
# estimated_seconds: 30
# preconditions:
#   chain_running: true
#   profile: hardfork-boho-delayed
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# TC-4-4-02 — Anzeon active from genesis, Boho activates at block 10
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-27-simultaneous-blockN"
check_env || { test_result; exit 1; }

BOHO_BLOCK=10
P256_PRECOMPILE="0x0000000000000000000000000000000000000100"
P256_INPUT="0xbb5a52f42f9c9261ed4361f59422a1e30036e7c32b270c8807a419feca605023741dd5bda817d95e4626b4a73f0b961e01e1ed2d4e4e4de70ef0de3004f944be47e01f68bcb6cabb95c7e3ef333ce1f20c71fc9bbe8f3a1a7e39b1aa0803f1e424f66e60f5ae559b1564c6e0d5aee93b4ace0a5e99de5c82e39d3e34d903b3e7f94db1cc8a6e7f6e12508e4e4c63ea0e40b2c77fe0cb97a40041e0400b7bae4d"

# --- 1) Before BohoBlock: Anzeon active, Boho not yet ---
current=$(block_number 1)
if (( current < BOHO_BLOCK )); then
  # Anzeon feature check: AccountManager callable
  is_bl_data="0x$(selector "isBlacklisted(address)")$(pad_address "$TEST_ACC_A_ADDR")"
  resp=$(eth_call_raw 1 "$ACCOUNT_MANAGER" "$is_bl_data")
  assert_not_empty "$resp" "Anzeon active: AccountManager callable before BohoBlock"
  observe "anzeon_active_pre_boho" "true"
fi

# --- 2) Wait for Boho activation ---
wait_for_block 1 "$BOHO_BLOCK" 60

# --- 3) After BohoBlock: Both active ---
resp=$(rpc 1 "eth_call" "[{\"to\":\"${P256_PRECOMPILE}\",\"data\":\"${P256_INPUT}\"},\"latest\"]")
p256_result=$(json_get "$resp" "result")
p256_last2="${p256_result: -2}"
assert_eq "$p256_last2" "01" "P-256 precompile active after BohoBlock"

# GovMinter v2 bytecode present
resp=$(rpc 1 "eth_getCode" "[\"${GOV_MINTER}\",\"latest\"]")
code=$(json_get "$resp" "result")
assert_gt "${#code}" "100" "GovMinter v2 bytecode after BohoBlock"

observe "boho_activated_at_block" "$BOHO_BLOCK"

test_result
