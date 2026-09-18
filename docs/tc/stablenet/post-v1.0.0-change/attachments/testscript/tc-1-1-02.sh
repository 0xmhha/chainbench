#!/bin/bash
# TC-1-1-02: 소각 제안 → 거부(reject) → 환불 전체 흐름 검증
# 네트워크: Privatenet (bohoBlock=1, 멤버 7명, quorum=5)
# 선행조건: 없음
# 차이점: TC-1-1-01은 제안자가 cancel, TC-1-1-02는 node2+node3이 NO 투표로 reject
#
# Step 1.  MEMBER_A (node1) native balance 확인
# Step 2.  GovMinter 컨트랙트 native balance 확인
# Step 3.  node1 로 proposeBurn 호출
# Step 4.  GovMinter balance 증가 검증 (+BURN_AMOUNT)
# Step 5.  MEMBER_A balance 감소 검증 (-BURN_AMOUNT - gas)
# Step 6.  MEMBER_B (node2) + MEMBER_C (node3) disapprove → reject
# Step 7.  refundableBalance 에 BURN_AMOUNT 추가 검증
# Step 8.  GovMinter native balance 표기
# Step 9.  claimBurnRefund 호출
# Step 10. refundableBalance 에서 BURN_AMOUNT 제거 검증
# Step 11. MEMBER_A native balance 변화 확인
# Step 12. GovMinter native balance 감소 검증
# Step 13. claim receipt 이벤트 디코딩

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-1-1-02"

# ── Test Data ──
BURN_AMOUNT=${BURN_AMOUNT:-100000000000000000000}  # 100 ether

# ── 필수 변수 확인 ──
check_not_tbd "MEMBER_A" "$MEMBER_A"
check_not_tbd "MEMBER_A_KEY" "$MEMBER_A_KEY"
check_not_tbd "MEMBER_B" "$MEMBER_B"
check_not_tbd "MEMBER_B_KEY" "$MEMBER_B_KEY"
check_not_tbd "MEMBER_C" "$MEMBER_C"
check_not_tbd "MEMBER_C_KEY" "$MEMBER_C_KEY"

# ── Setup: 멤버 확인 ──
echo -e "  ${BOLD}[Setup]${NC} GovMinter v2 멤버 확인"
for ADDR_VAR in MEMBER_A MEMBER_B MEMBER_C; do
  ADDR="${!ADDR_VAR}"
  IS_M=$(gmi_is_member "$ADDR")
  echo "  ${ADDR_VAR}: ${ADDR} (isMember=${IS_M})"
  if [ "$IS_M" != "true" ]; then
    tc_blocked "${ADDR_VAR}(${ADDR})가 GovMinter 멤버가 아님"
  fi
done
echo "  BURN_AMOUNT: $BURN_AMOUNT wei ($(cast from-wei "$BURN_AMOUNT" 2>/dev/null || echo "N/A") ether)"

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
# Step 3. proposeBurn 호출 (node1)
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 3]${NC} proposeBurn 호출 (BURN_AMOUNT=$(cast from-wei "$BURN_AMOUNT" 2>/dev/null) ether)"

PROOF_DATA=$(build_burn_proof "$MEMBER_A" "$BURN_AMOUNT" "WD-1-1-02-001" "REF-1-1-02-001" "TC-1-1-02 소각 제안 거부 검증")
echo "  withdrawalId=WD-1-1-02-001  referenceId=REF-1-1-02-001"

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
# Step 6. MEMBER_B (node2) + MEMBER_C (node3) disapprove → reject
# quorum=2: NO 투표 2표 → 자동 reject
# → node2, node3 이 disapprove (2표 = quorum 달성 → reject)
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 6]${NC} disapproveProposal — node2, node3 NO 투표 (proposalId=$PROPOSAL_ID)"

echo "  disapprove by MEMBER_B (node2: ${MEMBER_B})..."
TX_REJECT_B=$(cast_send "$GMI" 'disapproveProposal(uint256)' "$PROPOSAL_ID" \
  --private-key "$MEMBER_B_KEY" --rpc-url "$RPC" --json | jq -r .transactionHash)
print_tx_info "$TX_REJECT_B"
assert_eq 3 "MEMBER_B disapprove 성공" "$(get_tx_status "$TX_REJECT_B")" "0x1"

echo ""
echo "  disapprove by MEMBER_C (node3: ${MEMBER_C})..."
TX_REJECT_C=$(cast_send "$GMI" 'disapproveProposal(uint256)' "$PROPOSAL_ID" \
  --private-key "$MEMBER_C_KEY" --rpc-url "$RPC" --json | jq -r .transactionHash)
print_tx_info "$TX_REJECT_C"
assert_eq 4 "MEMBER_C disapprove 성공" "$(get_tx_status "$TX_REJECT_C")" "0x1"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 6-1. proposal 상태 검증 — Rejected(7) 확인
# disapprove quorum 달성 시:
#   _vote() → _finalizeProposal(Rejected) → _onProposalFinalized()
#   → _cleanupBurnDeposit(): burnBalance→refundableBalance 이동 + BurnDepositRefunded 이벤트
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo ""
echo -e "  ${BOLD}[Step 6-1]${NC} proposal 상태 검증"

IS_VOTING=$(gmi_is_voting "$PROPOSAL_ID")
echo "  isProposalInVoting: ${IS_VOTING}"
assert_eq 5 "투표 종료 (isProposalInVoting=false)" "$IS_VOTING" "false"

# proposals(proposalId) 에서 status 필드 조회 (10번째 반환값 = uint8 status)
# ProposalStatus: 0=None, 1=Voting, 2=Approved, 3=Executed, 4=Cancelled, 5=Expired, 6=Failed, 7=Rejected
PROPOSAL_RAW=$(cast call "$GMI" 'proposals(uint256)(bytes32,uint256,uint256,uint256,uint256,address,uint32,uint32,uint32,uint8,bytes)' \
  "$PROPOSAL_ID" --rpc-url "$RPC" 2>/dev/null)
PROPOSAL_STATUS=$(echo "$PROPOSAL_RAW" | sed -n '10p' | strip_cast)
echo "  proposal status: ${PROPOSAL_STATUS} (7=Rejected)"
assert_eq 6 "proposal status == Rejected(7)" "$PROPOSAL_STATUS" "7"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 7. refundableBalance 에 BURN_AMOUNT 추가 검증
# _cleanupBurnDeposit() 에서 자동 처리됨:
#   burnBalance[requester] -= amount
#   refundableBalance[requester] += amount
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 7]${NC} refundableBalance 검증 (after reject)"
REFUND_BAL=$(gmi_refundable_balance "$MEMBER_A")
echo "  refundableBalance(MEMBER_A): $REFUND_BAL"
assert_eq 7 "refundableBalance == BURN_AMOUNT" "$REFUND_BAL" "$BURN_AMOUNT"

BURN_BAL_AFTER=$(gmi_burn_balance "$MEMBER_A")
echo "  burnBalance(MEMBER_A): $BURN_BAL_AFTER"
assert_eq 8 "burnBalance == 0 (refundable로 이동)" "$BURN_BAL_AFTER" "0"

# BurnDepositRefunded 이벤트 디코딩 검증 (마지막 reject TX에서 발생)
# BurnDepositRefunded(uint256 indexed proposalId, address indexed requester, uint256 amount)
#   topics[0] = event sig, topics[1] = proposalId, topics[2] = requester, data = amount
REJECT_RECEIPT=$(get_tx_receipt "$TX_REJECT_C")
BDR_SIG=$(get_burn_deposit_refunded_sig)
HAS_EVENT=$(has_event_in_receipt "$REJECT_RECEIPT" "$BDR_SIG")
assert_eq 9 "BurnDepositRefunded 이벤트 발생" "$HAS_EVENT" "true"

BDR_PROPOSAL=$(echo "$REJECT_RECEIPT" | jq -r ".logs[] | select(.topics[0]==\"${BDR_SIG}\") | .topics[1]" 2>/dev/null | head -1)
PROPOSAL_ID_PADDED="0x$(printf '%064s' "$PROPOSAL_ID" | tr ' ' '0')"
echo "  BDR proposalId: $BDR_PROPOSAL (expected: $PROPOSAL_ID_PADDED)"
assert_eq 10 "BDR proposalId 일치" \
  "$(echo "$BDR_PROPOSAL" | tr '[:upper:]' '[:lower:]')" \
  "$(echo "$PROPOSAL_ID_PADDED" | tr '[:upper:]' '[:lower:]')"

BDR_REQUESTER=$(echo "$REJECT_RECEIPT" | jq -r ".logs[] | select(.topics[0]==\"${BDR_SIG}\") | .topics[2]" 2>/dev/null | head -1)
MEMBER_A_PADDED_BDR="0x$(printf '%064s' "${MEMBER_A#0x}" | tr ' ' '0')"
echo "  BDR requester: $BDR_REQUESTER (expected: $MEMBER_A_PADDED_BDR)"
assert_eq 11 "BDR requester == MEMBER_A" \
  "$(echo "$BDR_REQUESTER" | tr '[:upper:]' '[:lower:]')" \
  "$(echo "$MEMBER_A_PADDED_BDR" | tr '[:upper:]' '[:lower:]')"

BDR_AMOUNT=$(echo "$REJECT_RECEIPT" | jq -r ".logs[] | select(.topics[0]==\"${BDR_SIG}\") | .data" 2>/dev/null | head -1)
BURN_HEX_RAW_BDR=$(cast to-hex "$BURN_AMOUNT" 2>/dev/null | sed 's/^0x//')
BURN_AMOUNT_HEX_BDR="0x$(printf '%064s' "$BURN_HEX_RAW_BDR" | tr ' ' '0')"
echo "  BDR amount: $BDR_AMOUNT (expected: $BURN_AMOUNT_HEX_BDR)"
assert_eq 12 "BDR amount == BURN_AMOUNT" "$BDR_AMOUNT" "$BURN_AMOUNT_HEX_BDR"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 8. GovMinter native balance 표기 (reject 후, claim 전)
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 8]${NC} GovMinter native balance (after reject, before claim)"
GMI_BAL_BEFORE_CLAIM=$(cast balance "$GMI" --rpc-url "$RPC" | strip_cast)
echo "  GMI balance: ${GMI_BAL_BEFORE_CLAIM} wei ($(cast from-wei "${GMI_BAL_BEFORE_CLAIM:-0}" 2>/dev/null || echo "N/A") ether)"
assert_eq 13 "GMI balance 유지 (reject은 자금 미반환)" "$GMI_BAL_BEFORE_CLAIM" "$GMI_BAL_AFTER_PROPOSE"

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
assert_eq 14 "claimBurnRefund 성공" "$(get_tx_status "$TX_CLAIM")" "0x1"

CLAIM_RECEIPT=$(get_tx_receipt "$TX_CLAIM")
BRC_SIG=$(get_burn_refund_claimed_sig)
HAS_CLAIM_EVENT=$(has_event_in_receipt "$CLAIM_RECEIPT" "$BRC_SIG")
assert_eq 15 "BurnRefundClaimed 이벤트 발생" "$HAS_CLAIM_EVENT" "true"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 10. refundableBalance 에서 BURN_AMOUNT 제거 검증
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 10]${NC} refundableBalance 검증 (after claim)"
REFUND_BAL_AFTER_CLAIM=$(gmi_refundable_balance "$MEMBER_A")
echo "  refundableBalance(MEMBER_A): $REFUND_BAL_AFTER_CLAIM"
assert_eq 16 "refundableBalance == 0 (claim 완료)" "$REFUND_BAL_AFTER_CLAIM" "0"

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
assert_eq 17 "GMI balance 감소 == BURN_AMOUNT" "$GMI_CLAIM_DECREASE" "$BURN_AMOUNT"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 13. claim receipt 이벤트 디코딩 — BurnRefundClaimed 수량 검증
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 13]${NC} claim receipt 이벤트 디코딩 (BurnRefundClaimed)"
echo "  TX: $TX_CLAIM"

BRC_SIG=$(get_burn_refund_claimed_sig)

# requester == MEMBER_A
EVENT_REQUESTER=$(echo "$CLAIM_RECEIPT" | jq -r ".logs[] | select(.topics[0]==\"${BRC_SIG}\") | .topics[1]" 2>/dev/null | head -1)
MEMBER_A_PADDED="0x$(printf '%064s' "${MEMBER_A#0x}" | tr ' ' '0')"
echo "  event requester: $EVENT_REQUESTER"
echo "  expected (padded): $MEMBER_A_PADDED"
assert_eq 18 "이벤트 requester == MEMBER_A" \
  "$(echo "$EVENT_REQUESTER" | tr '[:upper:]' '[:lower:]')" \
  "$(echo "$MEMBER_A_PADDED" | tr '[:upper:]' '[:lower:]')"

# amount == BURN_AMOUNT
EVENT_DATA=$(echo "$CLAIM_RECEIPT" | jq -r ".logs[] | select(.topics[0]==\"${BRC_SIG}\") | .data" 2>/dev/null | head -1)
BURN_HEX_RAW=$(cast to-hex "$BURN_AMOUNT" 2>/dev/null | sed 's/^0x//')
BURN_AMOUNT_HEX="0x$(printf '%064s' "$BURN_HEX_RAW" | tr ' ' '0')"
echo "  event data (amount): $EVENT_DATA"
echo "  expected (hex):      $BURN_AMOUNT_HEX"
assert_eq 19 "이벤트 amount == BURN_AMOUNT" "$EVENT_DATA" "$BURN_AMOUNT_HEX"

# 이벤트 주소 == GMI
EVENT_ADDR=$(echo "$CLAIM_RECEIPT" | jq -r ".logs[] | select(.topics[0]==\"${BRC_SIG}\") | .address" 2>/dev/null | head -1)
echo "  event address: $EVENT_ADDR"
assert_eq 20 "이벤트 발생 컨트랙트 == GMI" \
  "$(echo "$EVENT_ADDR" | tr '[:upper:]' '[:lower:]')" \
  "$(echo "$GMI" | tr '[:upper:]' '[:lower:]')"

# ── 후속 TC용 저장 ──
echo "$PROPOSAL_ID" > "${RESULT_DIR}/TC-1-1-02.proposal_id"
echo "$TX_REJECT_C" > "${RESULT_DIR}/TC-1-1-02.tx_reject"
echo "$TX_CLAIM" > "${RESULT_DIR}/TC-1-1-02.tx_claim"

tc_end
