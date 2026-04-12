#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-4-5-01,TC-4-5-02
# name: Account Extra alloc bits reflected in AccountManager (authorized + blacklisted)
# category: regression/h-hardfork
# tags: [hardfork, boho, account-extra]
# estimated_seconds: 20
# preconditions:
#   chain_running: true
#   profile: hardfork-account-extra
# depends_on: []
# ---end-meta---
# TC-4-5-01 — TEST_ACC_D (bit 62: authorized) → AccountManager.isAuthorized == true
# TC-4-5-02 — TEST_ACC_E (bit 63: blacklisted) → AccountManager.isBlacklisted == true
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-30-extra-alloc-to-contract"
check_env || { test_result; exit 1; }

# --- TC-4-5-01: alloc.Extra bit 62 (authorized) → isAuthorized(TEST_ACC_D) == true ---
is_auth_sel=$(selector "isAuthorized(address)")
td_padded=$(pad_address "$TEST_ACC_D_ADDR" | sed 's/^0x//')
td_result_raw=$(eth_call_raw 1 "$ACCOUNT_MANAGER" "${is_auth_sel}${td_padded}")
td_result=$(hex_to_dec "$td_result_raw")
observe "isAuthorized_TEST_ACC_D_raw" "$td_result_raw"
assert_eq "$td_result" "1" "TC-4-5-01: TEST_ACC_D (alloc.Extra bit 62) isAuthorized == true"

# --- TC-4-5-02: alloc.Extra bit 63 (blacklisted) → isBlacklisted(TEST_ACC_E) == true ---
is_bl_sel=$(selector "isBlacklisted(address)")
te_padded=$(pad_address "$TEST_ACC_E_ADDR" | sed 's/^0x//')
te_result_raw=$(eth_call_raw 1 "$ACCOUNT_MANAGER" "${is_bl_sel}${te_padded}")
te_result=$(hex_to_dec "$te_result_raw")
observe "isBlacklisted_TEST_ACC_E_raw" "$te_result_raw"
assert_eq "$te_result" "1" "TC-4-5-02: TEST_ACC_E (alloc.Extra bit 63) isBlacklisted == true"

test_result
