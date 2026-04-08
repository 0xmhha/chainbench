#!/usr/bin/env bash
# Test: regression/e-blacklist-authorized/e-04-unblacklist
# RT-E-04 — 블랙리스트 해제 후 tx 정상 처리
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/e-blacklist-authorized/e-04-unblacklist"
check_env || { test_result; exit 1; }

unlock_all_validators

target_padded=$(pad_address "$TEST_ACC_E_ADDR" | sed 's/^0x//')
is_bl_sel=$(selector "isBlacklisted(address)")

# 현재 blacklist 상태인지 확인
before=$(hex_to_dec "$(eth_call_raw 1 "$ACCOUNT_MANAGER" "${is_bl_sel}${target_padded}")")
if [[ "$before" != "1" ]]; then
  _assert_fail "TEST_ACC_E should be blacklisted (run e-01 first)"
  test_result
  exit 1
fi

# GovCouncil proposeRemoveBlacklist 실행
propose_sel=$(selector "proposeRemoveBlacklist(address)")
propose_data="${propose_sel}${target_padded}"

receipt=$(gov_full_flow "$GOV_COUNCIL" "$propose_data" "$VALIDATOR_1_ADDR" "$VALIDATOR_2_ADDR") || {
  _assert_fail "gov flow failed"; test_result; exit 1;
}
exec_status=$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', ''))")
assert_eq "$exec_status" "0x1" "unBlacklist executed"

# 해제 확인
after=$(hex_to_dec "$(eth_call_raw 1 "$ACCOUNT_MANAGER" "${is_bl_sel}${target_padded}")")
assert_eq "$after" "0" "TEST_ACC_E is no longer blacklisted"

# 정상 tx 전송
tx_hash=$(send_raw_tx "1" "$TEST_ACC_E_PK" "$TEST_ACC_B_ADDR" "1" "" "21000" "dynamic")
assert_contains "$tx_hash" "0x" "tx submitted after unblacklist"
status=$(wait_receipt "1" "$tx_hash" 30)
assert_eq "$status" "success" "tx processed normally"

test_result
