#!/usr/bin/env bash
# Test: regression/f-system-contracts/f3-06-proposal-expiry
# RT-F-3-06 — proposal expiry 초과 → Expired 상태 전환 → execute 불가
#
# regression profile에서 expiry=60초로 설정되어 있음
# 시나리오: propose → 60초 대기 → expireProposal() → 상태 Expired → execute revert
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/f-system-contracts/f3-06-proposal-expiry"
check_env || { test_result; exit 1; }

unlock_all_validators

# proposeGasTip(unique_value) — 고유한 값으로 중복 회피
unique_tip=$(( 28000000000000 + (RANDOM % 1000) ))
propose_sel=$(selector "proposeGasTip(uint256)")
amt_padded=$(pad_uint256 "$unique_tip" | sed 's/^0x//')
propose_data="${propose_sel}${amt_padded}"

tx_hash=$(gov_call "1" "$GOV_VALIDATOR" "$propose_data" "$VALIDATOR_1_ADDR" 800000)
wait_receipt "1" "$tx_hash" 30 >/dev/null
proposal_id=$(extract_proposal_id_from_receipt "1" "$tx_hash")
[[ -z "$proposal_id" ]] && { _assert_fail "propose failed"; test_result; exit 1; }

printf '[INFO]  created proposal %s, waiting 65s for expiry...\n' "$proposal_id" >&2

# expiry (60초) 초과 대기 + 여유 5초
sleep 65

# expireProposal(uint256) 호출
expire_sel=$(selector "expireProposal(uint256)")
pid_padded=$(pad_uint256 "$proposal_id" | sed 's/^0x//')
expire_data="${expire_sel}${pid_padded}"

exp_tx=$(gov_call "1" "$GOV_VALIDATOR" "$expire_data" "$VALIDATOR_1_ADDR" 500000)
exp_receipt=$(wait_tx_receipt_full "1" "$exp_tx" 30)
exp_status=$(printf '%s' "$exp_receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', ''))")
assert_eq "$exp_status" "0x1" "expireProposal executed (proposal is now expired)"

# proposalStatus == Expired (enum 5)
status=$(gov_proposal_status "1" "$GOV_VALIDATOR" "$proposal_id" 2>/dev/null || echo "")
if [[ -n "$status" ]]; then
  assert_eq "$status" "5" "proposal state == Expired (5)"
fi

# executeProposal 시도 → revert
execute_sel=$(selector "executeProposal(uint256)")
exec_data="${execute_sel}${pid_padded}"
exec_tx=$(gov_call "1" "$GOV_VALIDATOR" "$exec_data" "$VALIDATOR_1_ADDR" 500000 2>/dev/null || echo "")
if [[ -n "$exec_tx" && "$exec_tx" != "null" ]]; then
  exec_receipt=$(wait_tx_receipt_full "1" "$exec_tx" 30)
  exec_status=$(printf '%s' "$exec_receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', ''))")
  assert_eq "$exec_status" "0x0" "executeProposal on expired proposal reverted"
fi

test_result
