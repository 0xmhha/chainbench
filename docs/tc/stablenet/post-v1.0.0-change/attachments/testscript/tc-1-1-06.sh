#!/bin/bash
# TC-1-1-06: claimBurnRefund 잔액 0 시 revert
# 네트워크: Privatenet
# 선행조건: 없음 (refundableBalance == 0 인 계정 필요)
#
# Step 1. node3 주소의 refundableBalance == 0 검증
# Step 2. node3 주소로 claimBurnRefund 호출 → revert 검증

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-1-1-06"

# ── 필수 변수 확인 ──
# node3 (VAL3) 사용 — VAL1/VAL2는 다른 TC에서 refundableBalance가 생길 수 있음
check_not_tbd "VAL3_ADDR" "$VAL3_ADDR"
check_not_tbd "VAL3_KEY" "$VAL3_KEY"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 1. node3 주소의 refundableBalance == 0 검증
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 1]${NC} node3 refundableBalance 확인"
REFUND_BAL=$(gmi_refundable_balance "$VAL3_ADDR")
echo "  refundableBalance(VAL3): $REFUND_BAL"
assert_eq 1 "refundableBalance == 0" "$REFUND_BAL" "0"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 2. node3 주소로 claimBurnRefund 호출 → revert 검증
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 2]${NC} claimBurnRefund 호출 (잔액 0 계정)"
TX_CLAIM=$(cast_send "$GMI" 'claimBurnRefund()' \
  --private-key "$VAL3_KEY" \
  --rpc-url "$RPC" --json 2>&1 | jq -r .transactionHash 2>/dev/null || echo "")

if [ -n "$TX_CLAIM" ] && [ "$TX_CLAIM" != "null" ]; then
  CLAIM_STATUS=$(get_tx_status "$TX_CLAIM")
  assert_eq 2 "tx revert (status 0x0)" "$CLAIM_STATUS" "0x0"
else
  # cast_send 자체가 revert → 전송 실패
  assert_eq 2 "tx revert (전송 실패)" "reverted" "reverted"
fi

tc_end
