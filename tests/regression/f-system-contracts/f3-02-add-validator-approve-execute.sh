#!/usr/bin/env bash
# Test: regression/f-system-contracts/f3-02-add-validator-approve-execute
# RT-F-3-02 — 검증자 추가 proposal quorum 승인 → execute → 다음 에폭 반영
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/f-system-contracts/f3-02-add-validator-approve-execute"
check_env || { test_result; exit 1; }

unlock_all_validators

proposal_id=$(cat /tmp/chainbench-regression/f3_01_proposal.txt 2>/dev/null || echo "")
if [[ -z "$proposal_id" ]]; then
  _assert_fail "run f3-01 first"; test_result; exit 1
fi

# validator2가 approve → quorum 달성
gov_approve "1" "$GOV_VALIDATOR" "$proposal_id" "$VALIDATOR_2_ADDR" >/dev/null
sleep 2

# execute
exec_tx=$(gov_execute "1" "$GOV_VALIDATOR" "$proposal_id" "$VALIDATOR_1_ADDR")
exec_receipt=$(wait_tx_receipt_full "1" "$exec_tx" 30)
exec_status=$(printf '%s' "$exec_receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', ''))")
assert_eq "$exec_status" "0x1" "execute status == 0x1"

# proposal 상태 == Executed (enum 3)
status=$(gov_proposal_status "1" "$GOV_VALIDATOR" "$proposal_id" 2>/dev/null || echo "")
if [[ -n "$status" ]]; then
  assert_eq "$status" "3" "proposal state == Executed (3)"
fi

# 새 member가 isActiveMember == true
is_sel=$(selector "isActiveMember(address)")
target_padded=$(pad_address "$TEST_ACC_E_ADDR" | sed 's/^0x//')
is_member=$(hex_to_dec "$(eth_call_raw 1 "$GOV_VALIDATOR" "${is_sel}${target_padded}")")
assert_eq "$is_member" "1" "TEST_ACC_E is now GovValidator member"

# 참고: 실제 validator list (istanbul_getValidators)는 configureValidator 호출 전에는 변하지 않음
# 다음 에폭에 반영되려면 새 member가 configureValidator + BLS key 등록 후 에폭 경계 도달해야 함

test_result
