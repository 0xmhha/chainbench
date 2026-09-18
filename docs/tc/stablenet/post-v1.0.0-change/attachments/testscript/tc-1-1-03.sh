#!/bin/bash
# TC-1-1-03: 소각 제안 → 만료(expiry) → 환불 전체 흐름 검증
# 네트워크: Privatenet (bohoBlock=1)
# 선행조건: genesis expiry=3 (3초 만료) 으로 설정한 체인
# 차이점: TC-1-1-01은 cancel, TC-1-1-03은 expiry 시간 경과 후 expireProposal 호출
#
# 만료 흐름:
#   proposeBurn → (expiry 시간 경과) → expireProposal(proposalId)
#   → _finalizeProposal(Expired) → _onProposalFinalized()
#   → _cleanupBurnDeposit(): burnBalance→refundableBalance 이동
#
# Step 1.  MEMBER_A native balance 확인
# Step 2.  GovMinter 컨트랙트 native balance 확인
# Step 3.  proposeBurn 호출
# Step 4.  GovMinter balance 증가 검증 (+BURN_AMOUNT)
# Step 5.  MEMBER_A balance 감소 검증 (-BURN_AMOUNT - gas)
# Step 6.  expiry 대기 + expireProposal 호출
# Step 7.  proposal 상태 검증 — Expired(5) + refundableBalance 검증
# Step 8.  GovMinter native balance 표기 (expire 후, claim 전)
# Step 9.  claimBurnRefund 호출
# Step 10. refundableBalance 에서 BURN_AMOUNT 제거 검증
# Step 11. MEMBER_A native balance 변화 확인
# Step 12. GovMinter native balance 감소 검증
# Step 13. claim receipt 이벤트 디코딩

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-1-1-03"

# ── Test Data ──
BURN_AMOUNT=${BURN_AMOUNT:-100000000000000000000}  # 100 ether
# genesis expiry=3 기준, 여유분 포함 대기 시간 (초)
WAIT_SECONDS=${WAIT_SECONDS:-5}

# ── 필수 변수 확인 ──
check_not_tbd "MEMBER_A" "$MEMBER_A"
check_not_tbd "MEMBER_A_KEY" "$MEMBER_A_KEY"

# ── Setup: 멤버 확인 + expiry 확인 ──
echo -e "  ${BOLD}[Setup]${NC} GovMinter v2 멤버 및 expiry 확인"
IS_MEMBER=$(gmi_is_member "$MEMBER_A")
if [ "$IS_MEMBER" != "true" ]; then
  tc_blocked "MEMBER_A($MEMBER_A)가 GovMinter 멤버가 아님"
fi

# 체인의 실제 proposalExpiry 조회
CHAIN_EXPIRY=$(cast call "$GMI" 'proposalExpiry()(uint256)' --rpc-url "$RPC" 2>/dev/null | strip_cast)
echo "  MEMBER_A: $MEMBER_A (isMember=$IS_MEMBER)"
echo "  BURN_AMOUNT: $BURN_AMOUNT wei ($(cast from-wei "$BURN_AMOUNT" 2>/dev/null || echo "N/A") ether)"
echo "  proposalExpiry (체인): ${CHAIN_EXPIRY} 초"
echo "  대기 시간: ${WAIT_SECONDS} 초"

if [ "${CHAIN_EXPIRY:-0}" -gt 60 ] 2>/dev/null; then
  echo -e "  ${YELLOW}경고: expiry(${CHAIN_EXPIRY}초)가 60초 초과 — 테스트에 오래 걸릴 수 있음${NC}"
  echo -e "  ${YELLOW}genesis expiry=3 으로 설정한 체인에서 실행 권장${NC}"
fi

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 1. MEMBER_A native balance 확인
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 1]${NC} MEMBER_A native balance (before proposeBurn)"
MEMBER_BAL_BEFORE=$(cast balance "$MEMBER_A" --rpc-url "$RPC" | strip_cast)
echo "  MEMBER_A balance: ${MEMBER_BAL_BEFORE} wei ($(cast from-wei "${MEMBER_BAL_BEFORE:-0}" 2>/dev/null || echo "N/A") ether)"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 2. GovMinter 컨트랙트 native balance 확인
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 2]${NC} GovMinter native balance (before proposeBurn)"
GMI_BAL_BEFORE=$(cast balance "$GMI" --rpc-url "$RPC" | strip_cast)
echo "  GMI balance: ${GMI_BAL_BEFORE} wei ($(cast from-wei "${GMI_BAL_BEFORE:-0}" 2>/dev/null || echo "N/A") ether)"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 3. proposeBurn 호출
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 3]${NC} proposeBurn 호출 (BURN_AMOUNT=$(cast from-wei "$BURN_AMOUNT" 2>/dev/null) ether)"

PROOF_DATA=$(build_burn_proof "$MEMBER_A" "$BURN_AMOUNT" "WD-1-1-03-001" "REF-1-1-03-001" "TC-1-1-03 소각 제안 만료 검증")

TX_PROPOSE=$(cast_send "$GMI" 'proposeBurn(bytes)' "$PROOF_DATA" \
  --value "$BURN_AMOUNT" \
  --private-key "$MEMBER_A_KEY" \
  --rpc-url "$RPC" --json | jq -r .transactionHash)
print_tx_info "$TX_PROPOSE"

PROPOSAL_ID=$(cast call "$GMI" 'currentProposalId()(uint256)' --rpc-url "$RPC" 2>/dev/null | strip_cast)
echo "  proposalId: $PROPOSAL_ID"
echo "  burnBalance(MEMBER_A): $(gmi_burn_balance "$MEMBER_A")"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 4. GovMinter balance 증가 검증
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 4]${NC} GovMinter balance 검증 (after proposeBurn)"
GMI_BAL_AFTER_PROPOSE=$(cast balance "$GMI" --rpc-url "$RPC" | strip_cast)
echo "  GMI balance: ${GMI_BAL_AFTER_PROPOSE} wei ($(cast from-wei "${GMI_BAL_AFTER_PROPOSE:-0}" 2>/dev/null || echo "N/A") ether)"

GMI_INCREASE=$(big_sub "$GMI_BAL_AFTER_PROPOSE" "$GMI_BAL_BEFORE")
echo "  GMI 증가분: ${GMI_INCREASE} wei (기대: ${BURN_AMOUNT})"
assert_eq 1 "GMI balance 증가 == BURN_AMOUNT" "$GMI_INCREASE" "$BURN_AMOUNT"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 5. MEMBER_A balance 감소 검증
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 5]${NC} MEMBER_A balance 검증 (after proposeBurn)"
MEMBER_BAL_AFTER_PROPOSE=$(cast balance "$MEMBER_A" --rpc-url "$RPC" | strip_cast)
echo "  MEMBER_A balance: ${MEMBER_BAL_AFTER_PROPOSE} wei ($(cast from-wei "${MEMBER_BAL_AFTER_PROPOSE:-0}" 2>/dev/null || echo "N/A") ether)"

MEMBER_DECREASE=$(big_sub "$MEMBER_BAL_BEFORE" "$MEMBER_BAL_AFTER_PROPOSE")
echo "  감소분: ${MEMBER_DECREASE} wei (BURN_AMOUNT=${BURN_AMOUNT} + gas)"
assert_gte 2 "MEMBER_A 감소분 >= BURN_AMOUNT" "$MEMBER_DECREASE" "$BURN_AMOUNT"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 6. expiry 대기 + expireProposal 호출
# expireProposal(proposalId):
#   - Voting 또는 Approved 상태에서만 호출 가능
#   - block.timestamp > createdAt + proposalExpiry 이면 → Expired 로 전환
#   - _finalizeProposal(Expired) → _onProposalFinalized() → _cleanupBurnDeposit()
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 6]${NC} expiry 대기 (${WAIT_SECONDS}초) + expireProposal 호출"

# 만료 전 상태 확인
IS_VOTING_BEFORE=$(gmi_is_voting "$PROPOSAL_ID")
echo "  isProposalInVoting (만료 전): ${IS_VOTING_BEFORE}"

echo "  ${WAIT_SECONDS}초 대기 중..."
sleep "$WAIT_SECONDS"

echo "  expireProposal 호출 (proposalId=$PROPOSAL_ID)..."
TX_EXPIRE=$(cast_send "$GMI" 'expireProposal(uint256)' "$PROPOSAL_ID" \
  --private-key "$MEMBER_A_KEY" \
  --rpc-url "$RPC" --json | jq -r .transactionHash)
print_tx_info "$TX_EXPIRE"
assert_eq 3 "expireProposal TX 성공" "$(get_tx_status "$TX_EXPIRE")" "0x1"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 7. proposal 상태 검증 — Expired(5) + refundableBalance 검증
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 7]${NC} proposal 상태 및 refundableBalance 검증 (after expire)"

IS_VOTING=$(gmi_is_voting "$PROPOSAL_ID")
echo "  isProposalInVoting: ${IS_VOTING}"
assert_eq 4 "투표 종료 (isProposalInVoting=false)" "$IS_VOTING" "false"

# proposals(proposalId) status 필드 조회
PROPOSAL_RAW=$(cast call "$GMI" 'proposals(uint256)(bytes32,uint256,uint256,uint256,uint256,address,uint32,uint32,uint32,uint8,bytes)' \
  "$PROPOSAL_ID" --rpc-url "$RPC" 2>/dev/null)
PROPOSAL_STATUS=$(echo "$PROPOSAL_RAW" | sed -n '10p' | strip_cast)
echo "  proposal status: ${PROPOSAL_STATUS} (5=Expired)"
assert_eq 5 "proposal status == Expired(5)" "$PROPOSAL_STATUS" "5"

# refundableBalance 검증
REFUND_BAL=$(gmi_refundable_balance "$MEMBER_A")
echo "  refundableBalance(MEMBER_A): $REFUND_BAL"
assert_eq 6 "refundableBalance == BURN_AMOUNT" "$REFUND_BAL" "$BURN_AMOUNT"

BURN_BAL_AFTER=$(gmi_burn_balance "$MEMBER_A")
echo "  burnBalance(MEMBER_A): $BURN_BAL_AFTER"
assert_eq 7 "burnBalance == 0 (refundable로 이동)" "$BURN_BAL_AFTER" "0"

# BurnDepositRefunded 이벤트 디코딩 검증
# BurnDepositRefunded(uint256 indexed proposalId, address indexed requester, uint256 amount)
#   topics[0] = event sig, topics[1] = proposalId, topics[2] = requester, data = amount
EXPIRE_RECEIPT=$(get_tx_receipt "$TX_EXPIRE")
BDR_SIG=$(get_burn_deposit_refunded_sig)
HAS_EVENT=$(has_event_in_receipt "$EXPIRE_RECEIPT" "$BDR_SIG")
assert_eq 8 "BurnDepositRefunded 이벤트 발생" "$HAS_EVENT" "true"

BDR_PROPOSAL=$(echo "$EXPIRE_RECEIPT" | jq -r ".logs[] | select(.topics[0]==\"${BDR_SIG}\") | .topics[1]" 2>/dev/null | head -1)
PROPOSAL_ID_PADDED="0x$(printf '%064s' "$PROPOSAL_ID" | tr ' ' '0')"
echo "  BDR proposalId: $BDR_PROPOSAL (expected: $PROPOSAL_ID_PADDED)"
assert_eq 9 "BDR proposalId 일치" \
  "$(echo "$BDR_PROPOSAL" | tr '[:upper:]' '[:lower:]')" \
  "$(echo "$PROPOSAL_ID_PADDED" | tr '[:upper:]' '[:lower:]')"

BDR_REQUESTER=$(echo "$EXPIRE_RECEIPT" | jq -r ".logs[] | select(.topics[0]==\"${BDR_SIG}\") | .topics[2]" 2>/dev/null | head -1)
MEMBER_A_PADDED_BDR="0x$(printf '%064s' "${MEMBER_A#0x}" | tr ' ' '0')"
echo "  BDR requester: $BDR_REQUESTER (expected: $MEMBER_A_PADDED_BDR)"
assert_eq 10 "BDR requester == MEMBER_A" \
  "$(echo "$BDR_REQUESTER" | tr '[:upper:]' '[:lower:]')" \
  "$(echo "$MEMBER_A_PADDED_BDR" | tr '[:upper:]' '[:lower:]')"

BDR_AMOUNT=$(echo "$EXPIRE_RECEIPT" | jq -r ".logs[] | select(.topics[0]==\"${BDR_SIG}\") | .data" 2>/dev/null | head -1)
BURN_HEX_RAW_BDR=$(cast to-hex "$BURN_AMOUNT" 2>/dev/null | sed 's/^0x//')
BURN_AMOUNT_HEX_BDR="0x$(printf '%064s' "$BURN_HEX_RAW_BDR" | tr ' ' '0')"
echo "  BDR amount: $BDR_AMOUNT (expected: $BURN_AMOUNT_HEX_BDR)"
assert_eq 11 "BDR amount == BURN_AMOUNT" "$BDR_AMOUNT" "$BURN_AMOUNT_HEX_BDR"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 8. GovMinter native balance 표기 (expire 후, claim 전)
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 8]${NC} GovMinter native balance (after expire, before claim)"
GMI_BAL_BEFORE_CLAIM=$(cast balance "$GMI" --rpc-url "$RPC" | strip_cast)
echo "  GMI balance: ${GMI_BAL_BEFORE_CLAIM} wei ($(cast from-wei "${GMI_BAL_BEFORE_CLAIM:-0}" 2>/dev/null || echo "N/A") ether)"
assert_eq 12 "GMI balance 유지 (expire은 자금 미반환)" "$GMI_BAL_BEFORE_CLAIM" "$GMI_BAL_AFTER_PROPOSE"

MEMBER_BAL_BEFORE_CLAIM=$(cast balance "$MEMBER_A" --rpc-url "$RPC" | strip_cast)
echo "  MEMBER_A balance (before claim): ${MEMBER_BAL_BEFORE_CLAIM} wei"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 9. claimBurnRefund 호출
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 9]${NC} claimBurnRefund 호출 (MEMBER_A)"
TX_CLAIM=$(cast_send "$GMI" 'claimBurnRefund()' \
  --private-key "$MEMBER_A_KEY" \
  --rpc-url "$RPC" --json | jq -r .transactionHash)
print_tx_info "$TX_CLAIM"
assert_eq 13 "claimBurnRefund 성공" "$(get_tx_status "$TX_CLAIM")" "0x1"

CLAIM_RECEIPT=$(get_tx_receipt "$TX_CLAIM")
BRC_SIG=$(get_burn_refund_claimed_sig)
HAS_CLAIM_EVENT=$(has_event_in_receipt "$CLAIM_RECEIPT" "$BRC_SIG")
assert_eq 14 "BurnRefundClaimed 이벤트 발생" "$HAS_CLAIM_EVENT" "true"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 10. refundableBalance 에서 BURN_AMOUNT 제거 검증
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 10]${NC} refundableBalance 검증 (after claim)"
REFUND_BAL_AFTER_CLAIM=$(gmi_refundable_balance "$MEMBER_A")
echo "  refundableBalance(MEMBER_A): $REFUND_BAL_AFTER_CLAIM"
assert_eq 15 "refundableBalance == 0 (claim 완료)" "$REFUND_BAL_AFTER_CLAIM" "0"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 11. MEMBER_A native balance 변화 확인 (출력만, assert 없음)
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 11]${NC} MEMBER_A native balance 변화 확인 (after claim)"
MEMBER_BAL_AFTER_CLAIM=$(cast balance "$MEMBER_A" --rpc-url "$RPC" | strip_cast)

echo "  MEMBER_A balance (before claim): ${MEMBER_BAL_BEFORE_CLAIM} wei ($(cast from-wei "${MEMBER_BAL_BEFORE_CLAIM:-0}" 2>/dev/null || echo "N/A") ether)"
echo "  MEMBER_A balance (after claim):  ${MEMBER_BAL_AFTER_CLAIM} wei ($(cast from-wei "${MEMBER_BAL_AFTER_CLAIM:-0}" 2>/dev/null || echo "N/A") ether)"

MEMBER_CLAIM_INCREASE=$(big_sub "$MEMBER_BAL_AFTER_CLAIM" "$MEMBER_BAL_BEFORE_CLAIM")
echo "  증가분:                          ${MEMBER_CLAIM_INCREASE} wei ($(cast from-wei "${MEMBER_CLAIM_INCREASE:-0}" 2>/dev/null || echo "N/A") ether)"
echo "  refundableBalance 였던 값:       ${BURN_AMOUNT} wei ($(cast from-wei "${BURN_AMOUNT:-0}" 2>/dev/null || echo "N/A") ether)"
echo "  차이 (gas cost):                 $(big_sub "$BURN_AMOUNT" "$MEMBER_CLAIM_INCREASE") wei"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 12. GovMinter native balance 감소 검증
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 12]${NC} GovMinter native balance 검증 (after claim)"
GMI_BAL_AFTER_CLAIM=$(cast balance "$GMI" --rpc-url "$RPC" | strip_cast)
echo "  GMI balance: ${GMI_BAL_AFTER_CLAIM} wei ($(cast from-wei "${GMI_BAL_AFTER_CLAIM:-0}" 2>/dev/null || echo "N/A") ether)"

GMI_CLAIM_DECREASE=$(big_sub "$GMI_BAL_BEFORE_CLAIM" "$GMI_BAL_AFTER_CLAIM")
echo "  GMI 감소분: ${GMI_CLAIM_DECREASE} wei (기대: ${BURN_AMOUNT})"
assert_eq 16 "GMI balance 감소 == BURN_AMOUNT" "$GMI_CLAIM_DECREASE" "$BURN_AMOUNT"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 13. claim receipt 이벤트 디코딩
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 13]${NC} claim receipt 이벤트 디코딩 (BurnRefundClaimed)"
echo "  TX: $TX_CLAIM"

BRC_SIG=$(get_burn_refund_claimed_sig)

EVENT_REQUESTER=$(echo "$CLAIM_RECEIPT" | jq -r ".logs[] | select(.topics[0]==\"${BRC_SIG}\") | .topics[1]" 2>/dev/null | head -1)
MEMBER_A_PADDED="0x$(printf '%064s' "${MEMBER_A#0x}" | tr ' ' '0')"
echo "  event requester: $EVENT_REQUESTER"
assert_eq 17 "이벤트 requester == MEMBER_A" \
  "$(echo "$EVENT_REQUESTER" | tr '[:upper:]' '[:lower:]')" \
  "$(echo "$MEMBER_A_PADDED" | tr '[:upper:]' '[:lower:]')"

EVENT_DATA=$(echo "$CLAIM_RECEIPT" | jq -r ".logs[] | select(.topics[0]==\"${BRC_SIG}\") | .data" 2>/dev/null | head -1)
BURN_HEX_RAW=$(cast to-hex "$BURN_AMOUNT" 2>/dev/null | sed 's/^0x//')
BURN_AMOUNT_HEX="0x$(printf '%064s' "$BURN_HEX_RAW" | tr ' ' '0')"
echo "  event data (amount): $EVENT_DATA"
echo "  expected (hex):      $BURN_AMOUNT_HEX"
assert_eq 18 "이벤트 amount == BURN_AMOUNT" "$EVENT_DATA" "$BURN_AMOUNT_HEX"

EVENT_ADDR=$(echo "$CLAIM_RECEIPT" | jq -r ".logs[] | select(.topics[0]==\"${BRC_SIG}\") | .address" 2>/dev/null | head -1)
assert_eq 19 "이벤트 발생 컨트랙트 == GMI" \
  "$(echo "$EVENT_ADDR" | tr '[:upper:]' '[:lower:]')" \
  "$(echo "$GMI" | tr '[:upper:]' '[:lower:]')"

# ── 후속 TC용 저장 ──
echo "$PROPOSAL_ID" > "${RESULT_DIR}/TC-1-1-03.proposal_id"
echo "$TX_EXPIRE" > "${RESULT_DIR}/TC-1-1-03.tx_expire"
echo "$TX_CLAIM" > "${RESULT_DIR}/TC-1-1-03.tx_claim"

tc_end
