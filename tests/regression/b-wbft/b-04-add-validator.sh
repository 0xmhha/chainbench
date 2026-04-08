#!/usr/bin/env bash
# Test: regression/b-wbft/b-04-add-validator
# RT-B-04 — 검증자 member 추가 (GovBase proposeAddMember + voting)
#
# 시나리오:
#   1. 현재 4 validator (member), quorum=2
#   2. validator1이 proposeAddMember(TEST_ACC_D, newQuorum=3) 호출
#   3. validator2가 approveProposal → quorum(2) 달성 → 상태 Approved
#   4. validator1이 executeProposal 호출 → TEST_ACC_D가 GovValidator member로 추가
#   5. validator 목록 자체는 변하지 않음 (configureValidator 미호출)
#      실제 validator로 활동하려면 새 member가 configureValidator + BLS key 등록 필요
#
# 참고: 이 TC는 "member 추가"까지만 검증. 실제 validator 추가는 BLS 키 프로비저닝이 필요
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/b-wbft/b-04-add-validator"
check_env || { test_result; exit 1; }

# 모든 validator keystore unlock
unlock_all_validators

# 초기 member 수 확인 (memberList 조회) — GovValidator 주소
# versionedMemberList(uint32) — memberVersion=1
sel=$(selector "memberList()")
before_data=$(eth_call_raw "1" "$GOV_VALIDATOR" "$sel" 2>/dev/null || echo "0x")
printf '[INFO]  memberList result (hex): %s\n' "${before_data:0:100}" >&2

# proposeAddMember(TEST_ACC_D, newQuorum=3)
propose_sel=$(selector "proposeAddMember(address,uint32)")
new_member_padded=$(pad_address "$TEST_ACC_D_ADDR" | sed 's/^0x//')
new_quorum_padded=$(pad_uint256 "3" | sed 's/^0x//')
propose_data="${propose_sel}${new_member_padded}${new_quorum_padded}"

receipt=$(gov_full_flow "$GOV_VALIDATOR" "$propose_data" "$VALIDATOR_1_ADDR" "$VALIDATOR_2_ADDR") || {
  _assert_fail "gov_full_flow failed"
  test_result
  exit 1
}

# execute tx status 확인
exec_status=$(printf '%s' "$receipt" | python3 -c "import sys, json; r = json.load(sys.stdin); print(r.get('status', ''))")
assert_eq "$exec_status" "0x1" "executeProposal receipt status == 0x1"

# isActiveMember(TEST_ACC_D) 확인
is_member_sel=$(selector "isActiveMember(address)")
target_padded=$(pad_address "$TEST_ACC_D_ADDR" | sed 's/^0x//')
is_member_result=$(eth_call_raw "1" "$GOV_VALIDATOR" "${is_member_sel}${target_padded}")
is_member=$(hex_to_dec "$is_member_result")
assert_eq "$is_member" "1" "TEST_ACC_D is now an active member of GovValidator"

# validator list는 변하지 않음 (configureValidator 미호출)
val_list=$(rpc "1" "istanbul_getValidators" '["latest"]' | python3 -c "
import sys, json
print(len(json.load(sys.stdin).get('result', [])))
")
assert_eq "$val_list" "4" "validator count unchanged (4) — configureValidator not called"

# 상태 저장 (b-05가 이어서 제거 테스트)
echo "$TEST_ACC_D_ADDR" > /tmp/chainbench-regression/added_member.txt

test_result
