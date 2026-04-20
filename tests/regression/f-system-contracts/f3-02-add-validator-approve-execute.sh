#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-F-3-02
# name: 검증자 추가 proposal quorum 승인 → execute → 다음 에폭 반영
# category: regression/f-system-contracts
# tags: [governance, validator]
# estimated_seconds: 37
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
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

# validator2가 approve → quorum 달성 (GovBase는 quorum 도달 시 approve 내에서 자동 실행)
approve_tx=$(gov_approve "1" "$GOV_VALIDATOR" "$proposal_id" "$VALIDATOR_2_ADDR")
sleep 2

# approve 후 proposal 상태 확인 — auto-execute 되었으면 별도 execute 불필요
prop_status=$(gov_proposal_status "1" "$GOV_VALIDATOR" "$proposal_id")
if [[ "$prop_status" == "3" ]]; then
  approve_node=$(addr_to_node "$VALIDATOR_2_ADDR")
  exec_receipt=$(wait_tx_receipt_full "$approve_node" "$approve_tx" 30)
else
  exec_tx=$(gov_execute "1" "$GOV_VALIDATOR" "$proposal_id" "$VALIDATOR_1_ADDR")
  exec_receipt=$(wait_tx_receipt_full "1" "$exec_tx" 30)
fi
exec_status=$(printf '%s' "$exec_receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', ''))")
assert_eq "$exec_status" "0x1" "execute status == 0x1"

# proposal 상태 == Executed (enum 3)
status=$(gov_proposal_status "1" "$GOV_VALIDATOR" "$proposal_id" 2>/dev/null || echo "")
if [[ -n "$status" ]]; then
  assert_eq "$status" "3" "proposal state == Executed (3)"
fi

# 새 member가 members(address) → (bool isActive, uint32 joinedAt) 확인
# GovBase에 isActiveMember() public view는 없음. members mapping의 automatic getter 사용.
members_sel=$(selector "members(address)")
target_padded=$(pad_address "$TEST_ACC_E_ADDR" | sed 's/^0x//')
members_result=$(eth_call_raw 1 "$GOV_VALIDATOR" "${members_sel}${target_padded}")
is_member=$(python3 -c "
raw = '${members_result}'.removeprefix('0x')
print(int(raw[:64], 16) if len(raw) >= 64 else 0)
")
assert_eq "$is_member" "1" "TEST_ACC_E is now GovValidator member"

# 참고: 실제 validator list (istanbul_getValidators)는 configureValidator 호출 전에는 변하지 않음
# 다음 에폭에 반영되려면 새 member가 configureValidator + BLS key 등록 후 에폭 경계 도달해야 함

test_result
