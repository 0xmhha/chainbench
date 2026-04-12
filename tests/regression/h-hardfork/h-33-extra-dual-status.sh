#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-4-5-07
# name: TEST_ACC_F dual status: authorized AND blacklisted simultaneously
# category: regression/h-hardfork
# tags: [hardfork, boho, account-extra]
# estimated_seconds: 20
# preconditions:
#   chain_running: true
#   profile: hardfork-account-extra
# depends_on: []
# ---end-meta---
# TC-4-5-07 — TEST_ACC_F has alloc.Extra=0xc000000000000000 (bits 62+63 set).
#              Both isAuthorized and isBlacklisted must return true independently.
#              Address: 0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-33-extra-dual-status"
check_env || { test_result; exit 1; }

readonly TEST_ACC_F_ADDR="0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc"

# --- isAuthorized(TEST_ACC_F) == true ---
is_auth_sel=$(selector "isAuthorized(address)")
tf_padded=$(pad_address "$TEST_ACC_F_ADDR" | sed 's/^0x//')
auth_result_raw=$(eth_call_raw 1 "$ACCOUNT_MANAGER" "${is_auth_sel}${tf_padded}")
auth_result=$(hex_to_dec "$auth_result_raw")
observe "isAuthorized_TEST_ACC_F_raw" "$auth_result_raw"
assert_eq "$auth_result" "1" \
  "TC-4-5-07a: TEST_ACC_F (extra=0xc000...) isAuthorized == true (bit 62 set)"

# --- isBlacklisted(TEST_ACC_F) == true ---
is_bl_sel=$(selector "isBlacklisted(address)")
bl_result_raw=$(eth_call_raw 1 "$ACCOUNT_MANAGER" "${is_bl_sel}${tf_padded}")
bl_result=$(hex_to_dec "$bl_result_raw")
observe "isBlacklisted_TEST_ACC_F_raw" "$bl_result_raw"
assert_eq "$bl_result" "1" \
  "TC-4-5-07b: TEST_ACC_F (extra=0xc000...) isBlacklisted == true (bit 63 set)"

test_result
