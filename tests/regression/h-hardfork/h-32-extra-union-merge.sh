#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-4-5-05,TC-4-5-06
# name: Union merge: alloc.Extra AND GovCouncil params produce no duplicates
# category: regression/h-hardfork
# tags: [hardfork, boho, account-extra]
# estimated_seconds: 20
# preconditions:
#   chain_running: true
#   profile: hardfork-account-extra
# depends_on: []
# ---end-meta---
# TC-4-5-05 — TEST_ACC_D appears in both alloc.Extra (bit 62) AND GovCouncil.authorizedAddresses;
#              isAuthorized must return true (union, not double-registration error)
# TC-4-5-06 — No duplicate: GovCouncil authorizedAddresses count equals expected value
#              (union dedup keeps exactly one entry per address)
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-32-extra-union-merge"
check_env || { test_result; exit 1; }

# --- TC-4-5-05: Union merge → isAuthorized(TEST_ACC_D) == true (no revert/double-add) ---
is_auth_sel=$(selector "isAuthorized(address)")
td_padded=$(pad_address "$TEST_ACC_D_ADDR" | sed 's/^0x//')
td_result_raw=$(eth_call_raw 1 "$ACCOUNT_MANAGER" "${is_auth_sel}${td_padded}")
td_result=$(hex_to_dec "$td_result_raw")
observe "isAuthorized_TEST_ACC_D_raw" "$td_result_raw"
assert_eq "$td_result" "1" \
  "TC-4-5-05: union merge of alloc.Extra + GovCouncil → isAuthorized(TEST_ACC_D) == true"

# --- TC-4-5-06: Verify no duplicate in authorized list ---
# GovCouncil.getAuthorizedCount() or equivalent: use authorizedAddresses length via
# a known ABI.  GovCouncil v1 exposes authorizedAddresses(uint256) as an auto-getter
# for the array and does not expose a length() directly, so we probe index 0 and 1:
#   - index 0 should exist (TEST_ACC_D)
#   - index 1 should NOT exist if union dedup worked (only one entry)
# We confirm TEST_ACC_D appears exactly once by checking both indices.
auth_arr_sel=$(selector "authorizedAddresses(uint256)")

# index 0
idx0_padded=$(python3 -c "print('0x' + format(0,'064x'))" | sed 's/^0x//')
idx0_raw=$(eth_call_raw 1 "$GOV_COUNCIL" "${auth_arr_sel}${idx0_padded}" 2>/dev/null || echo "")
observe "govCouncil_authorizedAddresses_0" "$idx0_raw"

# index 0 must be non-empty and non-zero (TEST_ACC_D or the zero-padded address)
# Extract the address portion (last 40 hex chars of the 32-byte return value)
if [[ -n "$idx0_raw" && "$idx0_raw" != "0x" ]]; then
  idx0_addr="0x${idx0_raw: -40}"
  idx0_addr_lc="${idx0_addr,,}"
  td_addr_lc="${TEST_ACC_D_ADDR,,}"
  if [[ "$idx0_addr_lc" == "$td_addr_lc" ]]; then
    _assert_pass "TC-4-5-06a: GovCouncil.authorizedAddresses[0] == TEST_ACC_D (no offset from dup)"
  else
    _assert_fail "TC-4-5-06a: GovCouncil.authorizedAddresses[0] expected TEST_ACC_D, got ${idx0_addr}"
  fi
else
  _assert_fail "TC-4-5-06a: GovCouncil.authorizedAddresses[0] returned empty — list missing entirely"
fi

# index 1 — in this profile only TEST_ACC_D is in authorizedAddresses, so index 1 should
# revert or return zero address (no second entry means no duplicate was added).
idx1_padded=$(python3 -c "print('0x' + format(1,'064x'))" | sed 's/^0x//')
idx1_raw=$(eth_call_raw 1 "$GOV_COUNCIL" "${auth_arr_sel}${idx1_padded}" 2>/dev/null || echo "0x")
observe "govCouncil_authorizedAddresses_1" "$idx1_raw"

# If result is empty, 0x, or all-zeros → no second entry (correct dedup)
idx1_stripped="${idx1_raw#0x}"
# Remove leading zeros
idx1_nonzero=$(echo "$idx1_stripped" | sed 's/^0*//')
if [[ -z "$idx1_nonzero" ]]; then
  _assert_pass "TC-4-5-06b: GovCouncil.authorizedAddresses[1] is zero/empty — no duplicate entry"
else
  _assert_fail "TC-4-5-06b: GovCouncil.authorizedAddresses[1] is non-zero (${idx1_raw}) — possible duplicate"
fi

test_result
