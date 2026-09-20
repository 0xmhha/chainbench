#!/bin/bash
# TC-1-1-07: claimBurnRefund 중복 호출 방어
# 네트워크: Privatenet
# 선행조건: TC-1-1-05 완료 (refundableBalance(MEMBER_A) == 0, 이미 출금 완료)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-1-1-07"

# ── 필수 변수 확인 ──
check_not_tbd "MEMBER_A" "$MEMBER_A"
check_not_tbd "MEMBER_A_KEY" "$MEMBER_A_KEY"

# ── Setup ──
echo -e "  ${BOLD}[Setup]${NC} TC-1-1-05 완료 상태 확인 (refundableBalance == 0)"
REFUND_BAL=$(cast call "$GMI" 'refundableBalance(address)(uint256)' "$MEMBER_A" --rpc-url "$RPC" 2>/dev/null)
echo "  refundableBalance: $REFUND_BAL"

# ── Action ──
echo -e "\n  ${BOLD}[Action]${NC} claimBurnRefund 중복 호출"
TX_CLAIM2=$(cast_send "$GMI" 'claimBurnRefund()' \
  --private-key "$MEMBER_A_KEY" \
  --rpc-url "$RPC" --json 2>&1 | jq -r .transactionHash 2>/dev/null || echo "")

# ── Verification ──
echo -e "\n  ${BOLD}[Verification]${NC}"

# 1. 중복 호출 revert
if [ -n "$TX_CLAIM2" ] && [ "$TX_CLAIM2" != "null" ]; then
  CLAIM_STATUS=$(cast receipt "$TX_CLAIM2" --rpc-url "$RPC" --json | jq -r .status 2>/dev/null)
  assert_eq 1 "중복 호출 revert (status 0x0)" "$CLAIM_STATUS" "0x0"
else
  assert_eq 1 "중복 호출 revert (전송 실패)" "reverted" "reverted"
fi

tc_end
