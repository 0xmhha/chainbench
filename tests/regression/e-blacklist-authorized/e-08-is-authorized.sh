#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-E-08
# name: AccountManager.isAuthorized 조회
# category: regression/e-blacklist-authorized
# tags: [blacklist, access-control]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
# depends_on: []
# ---end-meta---
# Test: regression/e-blacklist-authorized/e-08-is-authorized
# RT-E-08 — AccountManager.isAuthorized 조회
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/e-blacklist-authorized/e-08-is-authorized"
check_env || { test_result; exit 1; }
ensure_nodes_running

unlock_all_validators

is_auth_sel=$(selector "isAuthorized(address)")

# 일반 계정 (non-authorized)
ta_padded=$(pad_address "$(acct_addr 1)" | sed 's/^0x//')
ta_result=$(hex_to_dec "$(eth_call_raw "$(node 1)" "$ACCOUNT_MANAGER" "${is_auth_sel}${ta_padded}")")
assert_eq "$ta_result" "0" "TEST_ACC_A is not authorized initially"

# TEST_ACC_D를 authorize 등록
propose_sel=$(selector "proposeAddAuthorizedAccount(address)")
td_padded=$(pad_address "$(acct_addr 4)" | sed 's/^0x//')

td_before=$(hex_to_dec "$(eth_call_raw "$(node 1)" "$ACCOUNT_MANAGER" "${is_auth_sel}${td_padded}")")
if [[ "$td_before" != "1" ]]; then
  propose_data="${propose_sel}${td_padded}"
  gov_full_flow "$GOV_COUNCIL" "$propose_data" "$(validator_addr 1)" "$(validator_addr 2)" >/dev/null || true
fi

td_result=$(hex_to_dec "$(eth_call_raw "$(node 1)" "$ACCOUNT_MANAGER" "${is_auth_sel}${td_padded}")")
assert_eq "$td_result" "1" "TEST_ACC_D is authorized"

test_result
