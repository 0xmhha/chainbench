#!/bin/bash
# TC-1-1-09: BurnRefundClaimed 이벤트 발생 확인
# 네트워크: Privatenet
# 선행조건: TC-1-1-05의 receipt 활용

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-1-1-09"

# ── 필수 변수 확인 ──
check_not_tbd "MEMBER_A" "$MEMBER_A"

# ── Setup ──
echo -e "  ${BOLD}[Setup]${NC} TC-1-1-05의 TX 로드"
TX_FILE="${RESULT_DIR}/TC-1-1-05.tx_claim"
if [ -f "$TX_FILE" ]; then
  TX_CLAIM=$(cat "$TX_FILE")
else
  TX_CLAIM="${TX_CLAIM:-}"
  if [ -z "$TX_CLAIM" ]; then
    tc_blocked "TC-1-1-05 TX hash 없음 — TC-1-1-05 먼저 실행 필요"
  fi
fi
echo "  TX claimBurnRefund: $TX_CLAIM"

CLAIM_RECEIPT=$(cast receipt "$TX_CLAIM" --rpc-url "$RPC" --json)
BRC_SIG=$(get_burn_refund_claimed_sig)
BURN_AMOUNT=${BURN_AMOUNT:-1000000000000000000}

# ── Verification ──
echo -e "\n  ${BOLD}[Verification]${NC}"

# 1. 이벤트 주소 == GMI
EVENT_ADDR=$(echo "$CLAIM_RECEIPT" | jq -r ".logs[] | select(.topics[0]==\"${BRC_SIG}\") | .address" 2>/dev/null | head -1)
assert_eq 1 "이벤트 주소 == GMI" "$(echo "$EVENT_ADDR" | tr '[:upper:]' '[:lower:]')" "$(echo "$GMI" | tr '[:upper:]' '[:lower:]')"

# 2. topics[1] (requester) == MEMBER_A (zero-padded)
EVENT_REQUESTER=$(echo "$CLAIM_RECEIPT" | jq -r ".logs[] | select(.topics[0]==\"${BRC_SIG}\") | .topics[1]" 2>/dev/null | head -1)
MEMBER_A_PADDED="0x$(printf '%064s' "${MEMBER_A#0x}" | tr ' ' '0')"
assert_eq 2 "topics[1] requester == MEMBER_A" "$(echo "$EVENT_REQUESTER" | tr '[:upper:]' '[:lower:]')" "$(echo "$MEMBER_A_PADDED" | tr '[:upper:]' '[:lower:]')"

# 3. data (amount)
EVENT_DATA=$(echo "$CLAIM_RECEIPT" | jq -r ".logs[] | select(.topics[0]==\"${BRC_SIG}\") | .data" 2>/dev/null | head -1)
BURN_AMOUNT_HEX="0x$(printf '%064x' "$BURN_AMOUNT")"
assert_eq 3 "data amount == BURN_AMOUNT" "$EVENT_DATA" "$BURN_AMOUNT_HEX"

# 4. eth_getLogs 필터
LOG_COUNT=$(cast logs --address "$GMI" --topic-0 "$BRC_SIG" --rpc-url "$RPC" --json 2>/dev/null | jq 'length')
assert_gte 4 "eth_getLogs BurnRefundClaimed >= 1" "${LOG_COUNT:-0}" 1

tc_end
