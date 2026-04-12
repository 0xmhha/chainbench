#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-E-07
# name: AccountManager.isBlacklisted 조회
# category: regression/e-blacklist-authorized
# tags: [blacklist, access-control]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/e-blacklist-authorized/e-07-is-blacklisted
# RT-E-07 — AccountManager.isBlacklisted 조회
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/e-blacklist-authorized/e-07-is-blacklisted"
check_env || { test_result; exit 1; }

unlock_all_validators

# 일반 계정 (non-blacklisted) 먼저
is_bl_sel=$(selector "isBlacklisted(address)")
ta_padded=$(pad_address "$TEST_ACC_A_ADDR" | sed 's/^0x//')
ta_result=$(hex_to_dec "$(eth_call_raw 1 "$ACCOUNT_MANAGER" "${is_bl_sel}${ta_padded}")")
assert_eq "$ta_result" "0" "TEST_ACC_A is not blacklisted"

# TEST_ACC_E를 blacklist 등록 후 확인
propose_sel=$(selector "proposeAddBlacklist(address)")
te_padded=$(pad_address "$TEST_ACC_E_ADDR" | sed 's/^0x//')

# 이미 blacklist면 skip
te_before=$(hex_to_dec "$(eth_call_raw 1 "$ACCOUNT_MANAGER" "${is_bl_sel}${te_padded}")")
if [[ "$te_before" != "1" ]]; then
  propose_data="${propose_sel}${te_padded}"
  gov_full_flow "$GOV_COUNCIL" "$propose_data" "$VALIDATOR_1_ADDR" "$VALIDATOR_2_ADDR" >/dev/null || true
fi

# 다시 확인
te_result=$(hex_to_dec "$(eth_call_raw 1 "$ACCOUNT_MANAGER" "${is_bl_sel}${te_padded}")")
assert_eq "$te_result" "1" "TEST_ACC_E is blacklisted after proposal"

test_result
