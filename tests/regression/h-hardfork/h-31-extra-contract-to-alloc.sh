#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-4-5-03,TC-4-5-04
# name: GovCouncil params synced into AccountManager (contract → alloc direction)
# category: regression/h-hardfork
# tags: [hardfork, boho, account-extra]
# estimated_seconds: 20
# preconditions:
#   chain_running: true
#   profile: hardfork-account-extra
# depends_on: []
# ---end-meta---
# TC-4-5-03 — GovCouncil.authorizedAddresses includes TEST_ACC_D → AccountManager reflects it
# TC-4-5-04 — GovCouncil.blacklistedAddresses includes TEST_ACC_E → AccountManager reflects it
#
# The profile sets GovCouncil params with TEST_ACC_D in authorizedAddresses and TEST_ACC_E
# in blacklistedAddresses.  Genesis init must sync those contract-side entries into
# AccountManager so eth_call returns true even if alloc.Extra alone was absent.
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-31-extra-contract-to-alloc"
check_env || { test_result; exit 1; }
ensure_nodes_running

# --- TC-4-5-03: GovCouncil authorizedAddresses → isAuthorized(TEST_ACC_D) == true ---
is_auth_sel=$(selector "isAuthorized(address)")
td_padded=$(pad_address "$(acct_addr 4)" | sed 's/^0x//')
td_result_raw=$(eth_call_raw "$(node 1)" "$ACCOUNT_MANAGER" "${is_auth_sel}${td_padded}")
td_result=$(hex_to_dec "$td_result_raw")
observe "isAuthorized_TEST_ACC_D_raw" "$td_result_raw"
assert_eq "$td_result" "1" \
  "TC-4-5-03: GovCouncil.authorizedAddresses[TEST_ACC_D] synced → isAuthorized == true"

# --- TC-4-5-04: GovCouncil blacklistedAddresses → isBlacklisted(TEST_ACC_E) == true ---
is_bl_sel=$(selector "isBlacklisted(address)")
te_padded=$(pad_address "$(acct_addr 5)" | sed 's/^0x//')
te_result_raw=$(eth_call_raw "$(node 1)" "$ACCOUNT_MANAGER" "${is_bl_sel}${te_padded}")
te_result=$(hex_to_dec "$te_result_raw")
observe "isBlacklisted_TEST_ACC_E_raw" "$te_result_raw"
assert_eq "$te_result" "1" \
  "TC-4-5-04: GovCouncil.blacklistedAddresses[TEST_ACC_E] synced → isBlacklisted == true"

test_result
