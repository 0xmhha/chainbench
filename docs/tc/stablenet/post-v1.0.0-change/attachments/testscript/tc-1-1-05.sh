#!/bin/bash
# TC-1-1-05: claimBurnRefund 정상 출금
# 네트워크: Privatenet
# 선행조건: TC-1-1-01 완료 (refundableBalance(MEMBER_A) == BURN_AMOUNT)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-1-1-05"

# ── Test Data ──
BURN_AMOUNT=${BURN_AMOUNT:-1000000000000000000}

# ── 필수 변수 확인 ──
check_not_tbd "MEMBER_A" "$MEMBER_A"
check_not_tbd "MEMBER_A_KEY" "$MEMBER_A_KEY"

# ── Setup ──
echo -e "  ${BOLD}[Setup]${NC} refundableBalance 확인 (TC-1-1-01 완료 상태 필요)"
REFUND_BAL=$(cast call "$GMI" 'refundableBalance(address)(uint256)' "$MEMBER_A" --rpc-url "$RPC" 2>/dev/null)
if [ "$REFUND_BAL" != "$BURN_AMOUNT" ]; then
  tc_blocked "refundableBalance($REFUND_BAL) != BURN_AMOUNT($BURN_AMOUNT) — TC-1-1-01 먼저 실행 필요"
fi

BALANCE_BEFORE=$(cast balance "$MEMBER_A" --rpc-url "$RPC" 2>/dev/null)
echo "  Balance before: $BALANCE_BEFORE"

# ── Action ──
echo -e "\n  ${BOLD}[Action]${NC} claimBurnRefund 호출"
TX_CLAIM=$(cast_send "$GMI" 'claimBurnRefund()' \
  --private-key "$MEMBER_A_KEY" \
  --rpc-url "$RPC" --json | jq -r .transactionHash)
echo "  TX claimBurnRefund: $TX_CLAIM"

# ── Verification ──
echo -e "\n  ${BOLD}[Verification]${NC}"

# 1. claimBurnRefund 성공
CLAIM_STATUS=$(cast receipt "$TX_CLAIM" --rpc-url "$RPC" --json | jq -r .status)
assert_eq 1 "claimBurnRefund 성공" "$CLAIM_STATUS" "0x1"

# 2. refundableBalance == 0
REFUND_BAL_AFTER=$(cast call "$GMI" 'refundableBalance(address)(uint256)' "$MEMBER_A" --rpc-url "$RPC" 2>/dev/null)
assert_eq 2 "refundableBalance == 0" "$REFUND_BAL_AFTER" "0"

# 3. 잔액 증가
BALANCE_AFTER=$(cast balance "$MEMBER_A" --rpc-url "$RPC" 2>/dev/null)
assert_gte 3 "잔액 >= 이전 잔액" "$BALANCE_AFTER" "$BALANCE_BEFORE"

# 4. BurnRefundClaimed 이벤트
CLAIM_RECEIPT=$(cast receipt "$TX_CLAIM" --rpc-url "$RPC" --json)
BRC_SIG=$(get_burn_refund_claimed_sig)
HAS_EVENT=$(has_event_in_receipt "$CLAIM_RECEIPT" "$BRC_SIG")
assert_eq 4 "BurnRefundClaimed 이벤트 발생" "$HAS_EVENT" "true"

# 후속 TC용 TX 저장
echo "$TX_CLAIM" > "${RESULT_DIR}/TC-1-1-05.tx_claim"

tc_end
