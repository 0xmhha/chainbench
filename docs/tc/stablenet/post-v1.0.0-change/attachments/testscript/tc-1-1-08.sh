#!/bin/bash
# TC-1-1-08: _cleanupBurnDeposit 멱등성 확인
# 네트워크: Privatenet
# 선행조건: TC-1-1-01 완료 (proposalId가 Cancelled 상태)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-1-1-08"

# ── 필수 변수 확인 ──
check_not_tbd "MEMBER_A" "$MEMBER_A"
check_not_tbd "MEMBER_A_KEY" "$MEMBER_A_KEY"

# ── Setup ──
echo -e "  ${BOLD}[Setup]${NC} TC-1-1-01의 proposalId 로드"
PROPOSAL_ID_FILE="${RESULT_DIR}/TC-1-1-01.proposal_id"
if [ -f "$PROPOSAL_ID_FILE" ]; then
  PROPOSAL_ID=$(cat "$PROPOSAL_ID_FILE")
else
  PROPOSAL_ID="${PROPOSAL_ID:-<TBD>}"
  check_not_tbd "PROPOSAL_ID" "$PROPOSAL_ID"
fi
echo "  proposalId: $PROPOSAL_ID"

REFUND_BAL_BEFORE=$(cast call "$GMI" 'refundableBalance(address)(uint256)' "$MEMBER_A" --rpc-url "$RPC" 2>/dev/null)
echo "  refundableBalance before re-cancel: $REFUND_BAL_BEFORE"

# ── Action ──
echo -e "\n  ${BOLD}[Action]${NC} 이미 Cancelled된 제안에 cancelProposal 재호출"
TX_RECANCEL=$(cast_send "$GMI" 'cancelProposal(uint256)' "$PROPOSAL_ID" \
  --private-key "$MEMBER_A_KEY" \
  --rpc-url "$RPC" --json 2>&1 | jq -r .transactionHash 2>/dev/null || echo "")

# ── Verification ──
echo -e "\n  ${BOLD}[Verification]${NC}"

# 1. 재호출 revert
if [ -n "$TX_RECANCEL" ] && [ "$TX_RECANCEL" != "null" ]; then
  RECANCEL_STATUS=$(cast receipt "$TX_RECANCEL" --rpc-url "$RPC" --json | jq -r .status 2>/dev/null)
  assert_eq 1 "재호출 revert (status 0x0)" "$RECANCEL_STATUS" "0x0"
else
  assert_eq 1 "재호출 revert (전송 실패)" "reverted" "reverted"
fi

# 2. refundableBalance 불변
REFUND_BAL_AFTER=$(cast call "$GMI" 'refundableBalance(address)(uint256)' "$MEMBER_A" --rpc-url "$RPC" 2>/dev/null)
assert_eq 2 "refundableBalance 불변" "$REFUND_BAL_AFTER" "$REFUND_BAL_BEFORE"

tc_end
