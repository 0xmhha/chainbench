#!/bin/bash
# TC-1-1-10: BurnDepositRefunded 이벤트 발생 확인
# 네트워크: Privatenet
# 선행조건: TC-1-1-01의 receipt 활용

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-1-1-10"

# ── 필수 변수 확인 ──
check_not_tbd "MEMBER_A" "$MEMBER_A"

# ── Setup ──
echo -e "  ${BOLD}[Setup]${NC} TC-1-1-01의 TX 로드"
TX_CANCEL_FILE="${RESULT_DIR}/TC-1-1-01.tx_cancel"
PROPOSAL_ID_FILE="${RESULT_DIR}/TC-1-1-01.proposal_id"

if [ -f "$TX_CANCEL_FILE" ]; then
  TX_CANCEL=$(cat "$TX_CANCEL_FILE")
else
  TX_CANCEL="${TX_CANCEL:-}"
  if [ -z "$TX_CANCEL" ]; then
    tc_blocked "TC-1-1-01 TX hash 없음 — TC-1-1-01 먼저 실행 필요"
  fi
fi

if [ -f "$PROPOSAL_ID_FILE" ]; then
  PROPOSAL_ID=$(cat "$PROPOSAL_ID_FILE")
else
  PROPOSAL_ID="${PROPOSAL_ID:-}"
fi

echo "  TX cancelProposal: $TX_CANCEL"
echo "  proposalId: $PROPOSAL_ID"

CANCEL_RECEIPT=$(cast receipt "$TX_CANCEL" --rpc-url "$RPC" --json)
BDR_SIG=$(get_burn_deposit_refunded_sig)
BURN_AMOUNT=${BURN_AMOUNT:-1000000000000000000}

# ── Verification ──
echo -e "\n  ${BOLD}[Verification]${NC}"

# 1. topics[0] == BurnDepositRefunded sig
EVENT_TOPIC0=$(echo "$CANCEL_RECEIPT" | jq -r ".logs[] | select(.topics[0]==\"${BDR_SIG}\") | .topics[0]" 2>/dev/null | head -1)
assert_eq 1 "topics[0] == BurnDepositRefunded sig" "$EVENT_TOPIC0" "$BDR_SIG"

# 2. topics[1] (proposalId)
EVENT_PROPOSAL=$(echo "$CANCEL_RECEIPT" | jq -r ".logs[] | select(.topics[0]==\"${BDR_SIG}\") | .topics[1]" 2>/dev/null | head -1)
assert_eq 2 "topics[1] == proposalId" "$EVENT_PROPOSAL" "$PROPOSAL_ID"

# 3. topics[2] (requester) == MEMBER_A
EVENT_REQUESTER=$(echo "$CANCEL_RECEIPT" | jq -r ".logs[] | select(.topics[0]==\"${BDR_SIG}\") | .topics[2]" 2>/dev/null | head -1)
MEMBER_A_PADDED="0x$(printf '%064s' "${MEMBER_A#0x}" | tr ' ' '0')"
assert_eq 3 "topics[2] requester == MEMBER_A" "$(echo "$EVENT_REQUESTER" | tr '[:upper:]' '[:lower:]')" "$(echo "$MEMBER_A_PADDED" | tr '[:upper:]' '[:lower:]')"

# 4. data (amount)
EVENT_DATA=$(echo "$CANCEL_RECEIPT" | jq -r ".logs[] | select(.topics[0]==\"${BDR_SIG}\") | .data" 2>/dev/null | head -1)
BURN_AMOUNT_HEX="0x$(printf '%064x' "$BURN_AMOUNT")"
assert_eq 4 "data amount == BURN_AMOUNT" "$EVENT_DATA" "$BURN_AMOUNT_HEX"

tc_end
