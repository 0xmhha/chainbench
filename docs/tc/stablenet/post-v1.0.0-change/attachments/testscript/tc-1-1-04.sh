#!/bin/bash
# TC-1-1-04: 소각 제안 → 승인(approve) → 자동 실행(auto-execute) → burn 성공 검증
# 네트워크: Privatenet (bohoBlock=1, quorum=3)
# 선행조건: 없음
# 차이점: TC-1-1-01은 cancel→refundable, TC-1-1-04는 approve→auto-execute→burn(자산 영구 삭제)
#
# 핵심: approveProposal 은 autoExecute=true 로 _vote 호출
#   → quorum 달성 시 같은 TX 에서 자동 실행 (별도 executeProposal 불필요)
#
# Step 1.  MEMBER_A native balance 확인
# Step 2.  GovMinter 컨트랙트 native balance 확인
# Step 3.  NCA totalSupply 확인 (burn 전)
# Step 4.  proposeBurn 호출
# Step 5.  GovMinter balance 증가 검증 (+BURN_AMOUNT)
# Step 6.  MEMBER_A balance 감소 검증 (-BURN_AMOUNT - gas)
# Step 7.  MEMBER_B + MEMBER_C YES 투표 → quorum=3 달성 → 자동 실행
# Step 8.  proposal 상태 검증 — Executed(3)
# Step 9.  refundableBalance == 0 (burn 성공이므로 환불 없음)
# Step 10. GovMinter native balance 감소 검증 (burn으로 자산 영구 삭제)
# Step 11. NCA totalSupply 감소 검증 (burn 만큼 전체 공급량 감소)
# Step 12. MEMBER_A native balance 변화 확인 (burn 된 자산은 돌아오지 않음)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-1-1-04"

# ── Test Data ──
BURN_AMOUNT=${BURN_AMOUNT:-100000000000000000000}  # 100 ether

# ── 필수 변수 확인 (quorum=3 → 제안자 자동승인 + node2, node3 승인) ──
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

# 체인의 실제 quorum 조회
CHAIN_QUORUM=$(cast call "$GMI" 'quorum()(uint256)' --rpc-url "$RPC" 2>/dev/null | strip_cast || echo "?")
echo "  BURN_AMOUNT: $BURN_AMOUNT wei ($(cast from-wei "$BURN_AMOUNT" 2>/dev/null || echo "N/A") ether)"
echo "  quorum (체인): ${CHAIN_QUORUM}"

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
# Step 3. NCA totalSupply 확인 (burn 전)
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 3]${NC} NCA totalSupply (before burn)"
TOTAL_SUPPLY_BEFORE=$(nca_total_supply)
echo "  NCA totalSupply: ${TOTAL_SUPPLY_BEFORE} wei ($(cast from-wei "${TOTAL_SUPPLY_BEFORE:-0}" 2>/dev/null || echo "N/A") ether)"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 4. proposeBurn 호출
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 4]${NC} proposeBurn 호출 (BURN_AMOUNT=$(cast from-wei "$BURN_AMOUNT" 2>/dev/null) ether)"

PROOF_DATA=$(build_burn_proof "$MEMBER_A" "$BURN_AMOUNT" "WD-1-1-04-001" "REF-1-1-04-001" "TC-1-1-04 소각 실행 성공 검증")

TX_PROPOSE=$(cast_send "$GMI" 'proposeBurn(bytes)' "$PROOF_DATA" \
  --value "$BURN_AMOUNT" \
  --private-key "$MEMBER_A_KEY" \
  --rpc-url "$RPC" --json | jq -r .transactionHash)
print_tx_info "$TX_PROPOSE"

PROPOSAL_ID=$(cast call "$GMI" 'currentProposalId()(uint256)' --rpc-url "$RPC" 2>/dev/null | strip_cast)
echo "  proposalId: $PROPOSAL_ID"
echo "  burnBalance(MEMBER_A): $(gmi_burn_balance "$MEMBER_A")"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 5. GovMinter balance 증가 검증
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 5]${NC} GovMinter balance 검증 (after proposeBurn)"
GMI_BAL_AFTER_PROPOSE=$(cast balance "$GMI" --rpc-url "$RPC" | strip_cast)
echo "  GMI balance: ${GMI_BAL_AFTER_PROPOSE} wei ($(cast from-wei "${GMI_BAL_AFTER_PROPOSE:-0}" 2>/dev/null || echo "N/A") ether)"

GMI_INCREASE=$(big_sub "$GMI_BAL_AFTER_PROPOSE" "$GMI_BAL_BEFORE")
echo "  GMI 증가분: ${GMI_INCREASE} wei (기대: ${BURN_AMOUNT})"
assert_eq 1 "GMI balance 증가 == BURN_AMOUNT" "$GMI_INCREASE" "$BURN_AMOUNT"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 6. MEMBER_A balance 감소 검증
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 6]${NC} MEMBER_A balance 검증 (after proposeBurn)"
MEMBER_BAL_AFTER_PROPOSE=$(cast balance "$MEMBER_A" --rpc-url "$RPC" | strip_cast)
echo "  MEMBER_A balance: ${MEMBER_BAL_AFTER_PROPOSE} wei ($(cast from-wei "${MEMBER_BAL_AFTER_PROPOSE:-0}" 2>/dev/null || echo "N/A") ether)"

MEMBER_DECREASE=$(big_sub "$MEMBER_BAL_BEFORE" "$MEMBER_BAL_AFTER_PROPOSE")
echo "  감소분: ${MEMBER_DECREASE} wei (BURN_AMOUNT=${BURN_AMOUNT} + gas)"
assert_gte 2 "MEMBER_A 감소분 >= BURN_AMOUNT" "$MEMBER_DECREASE" "$BURN_AMOUNT"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 7. MEMBER_B + MEMBER_C YES 투표 → quorum 달성 → 자동 실행
# approveProposal 은 _vote(proposalId, true, true) — autoExecute=true
# → quorum 달성 시 같은 TX 에서 _executeProposal 자동 호출
# → 별도 executeProposal 호출 불필요
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 7]${NC} approveProposal — node2, node3 YES 투표 (proposalId=$PROPOSAL_ID)"
echo "  quorum=${CHAIN_QUORUM}: 제안자(1) + node2(1) = 2, 아직 미달"

echo "  approve by MEMBER_B (node2: ${MEMBER_B})..."
TX_APPROVE_B=$(cast_send "$GMI" 'approveProposal(uint256)' "$PROPOSAL_ID" \
  --private-key "$MEMBER_B_KEY" --rpc-url "$RPC" --json | jq -r .transactionHash)
print_tx_info "$TX_APPROVE_B"
assert_eq 3 "MEMBER_B approve 성공" "$(get_tx_status "$TX_APPROVE_B")" "0x1"

echo ""
echo "  quorum=${CHAIN_QUORUM}: 제안자(1) + node2(1) + node3(1) = 3 → quorum 달성 → 자동 실행"
echo "  approve by MEMBER_C (node3: ${MEMBER_C})..."
TX_APPROVE_C=$(cast_send "$GMI" 'approveProposal(uint256)' "$PROPOSAL_ID" \
  --private-key "$MEMBER_C_KEY" --rpc-url "$RPC" --json | jq -r .transactionHash)
print_tx_info "$TX_APPROVE_C"
assert_eq 4 "MEMBER_C approve 성공 (= 자동 실행 TX)" "$(get_tx_status "$TX_APPROVE_C")" "0x1"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 8. proposal 상태 검증 — Executed(3)
# approveProposal 의 autoExecute=true 로 인해 Approved(2)를 거쳐
# 같은 TX 안에서 바로 Executed(3) 로 전환됨
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 8]${NC} proposal 상태 검증 (after approve → auto-execute)"

PROPOSAL_RAW=$(cast call "$GMI" 'proposals(uint256)(bytes32,uint256,uint256,uint256,uint256,address,uint32,uint32,uint32,uint8,bytes)' \
  "$PROPOSAL_ID" --rpc-url "$RPC" 2>/dev/null)
PROPOSAL_STATUS=$(echo "$PROPOSAL_RAW" | sed -n '10p' | strip_cast)
echo "  proposal status: ${PROPOSAL_STATUS} (3=Executed)"
assert_eq 5 "proposal status == Executed(3)" "$PROPOSAL_STATUS" "3"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 9. refundableBalance == 0 (burn 성공 시 환불 없음)
# _cleanupBurnDeposit: status == Executed → refundableBalance 이동하지 않음
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 9]${NC} refundableBalance / burnBalance 검증 (burn 성공 → 환불 없음)"

REFUND_BAL=$(gmi_refundable_balance "$MEMBER_A")
echo "  refundableBalance(MEMBER_A): $REFUND_BAL"
assert_eq 6 "refundableBalance == 0 (burn 성공, 환불 없음)" "$REFUND_BAL" "0"

BURN_BAL=$(gmi_burn_balance "$MEMBER_A")
echo "  burnBalance(MEMBER_A): $BURN_BAL"
assert_eq 7 "burnBalance == 0" "$BURN_BAL" "0"

# BurnDepositRefunded 미발생 (성공이므로 환불 이벤트 없음)
# 자동 실행 TX = 마지막 approve TX (TX_APPROVE_C)
EXEC_RECEIPT=$(get_tx_receipt "$TX_APPROVE_C")
BDR_SIG=$(get_burn_deposit_refunded_sig)
HAS_REFUND_EVENT=$(has_event_in_receipt "$EXEC_RECEIPT" "$BDR_SIG")
assert_eq 8 "BurnDepositRefunded 미발생 (성공이므로)" "$HAS_REFUND_EVENT" "false"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 10. GovMinter native balance 감소 검증 (burn 으로 자산 영구 삭제)
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 10]${NC} GovMinter native balance 검증 (after burn)"
GMI_BAL_AFTER_BURN=$(cast balance "$GMI" --rpc-url "$RPC" | strip_cast)
echo "  GMI balance (before burn): ${GMI_BAL_AFTER_PROPOSE} wei ($(cast from-wei "${GMI_BAL_AFTER_PROPOSE:-0}" 2>/dev/null || echo "N/A") ether)"
echo "  GMI balance (after burn):  ${GMI_BAL_AFTER_BURN} wei ($(cast from-wei "${GMI_BAL_AFTER_BURN:-0}" 2>/dev/null || echo "N/A") ether)"

GMI_BURN_DECREASE=$(big_sub "$GMI_BAL_AFTER_PROPOSE" "$GMI_BAL_AFTER_BURN")
echo "  GMI 감소분: ${GMI_BURN_DECREASE} wei (기대: ${BURN_AMOUNT})"
assert_eq 9 "GMI balance 감소 == BURN_AMOUNT (burn 영구 삭제)" "$GMI_BURN_DECREASE" "$BURN_AMOUNT"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 11. NCA totalSupply 감소 검증 (burn 만큼 전체 공급량 감소)
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 11]${NC} NCA totalSupply 검증 (after burn)"
TOTAL_SUPPLY_AFTER=$(nca_total_supply)
echo "  NCA totalSupply (before burn): ${TOTAL_SUPPLY_BEFORE} wei ($(cast from-wei "${TOTAL_SUPPLY_BEFORE:-0}" 2>/dev/null || echo "N/A") ether)"
echo "  NCA totalSupply (after burn):  ${TOTAL_SUPPLY_AFTER} wei ($(cast from-wei "${TOTAL_SUPPLY_AFTER:-0}" 2>/dev/null || echo "N/A") ether)"

SUPPLY_DECREASE=$(big_sub "$TOTAL_SUPPLY_BEFORE" "$TOTAL_SUPPLY_AFTER")
echo "  totalSupply 감소분: ${SUPPLY_DECREASE} wei (기대: ${BURN_AMOUNT})"
assert_eq 10 "NCA totalSupply 감소 == BURN_AMOUNT" "$SUPPLY_DECREASE" "$BURN_AMOUNT"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 12. MEMBER_A native balance 변화 확인 (burn 된 자산은 돌아오지 않음)
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}[Step 12]${NC} MEMBER_A native balance 변화 확인 (burn 완료)"
MEMBER_BAL_FINAL=$(cast balance "$MEMBER_A" --rpc-url "$RPC" | strip_cast)

echo "  MEMBER_A balance (최초):     ${MEMBER_BAL_BEFORE} wei ($(cast from-wei "${MEMBER_BAL_BEFORE:-0}" 2>/dev/null || echo "N/A") ether)"
echo "  MEMBER_A balance (burn 후):  ${MEMBER_BAL_FINAL} wei ($(cast from-wei "${MEMBER_BAL_FINAL:-0}" 2>/dev/null || echo "N/A") ether)"

TOTAL_DECREASE=$(big_sub "$MEMBER_BAL_BEFORE" "$MEMBER_BAL_FINAL")
echo "  총 감소분: ${TOTAL_DECREASE} wei (BURN_AMOUNT=${BURN_AMOUNT} + gas)"
echo "  (burn 성공 → 자산 영구 삭제, claimBurnRefund 불가)"

# ── 후속 TC용 저장 ──
echo "$PROPOSAL_ID" > "${RESULT_DIR}/TC-1-1-04.proposal_id"
echo "$TX_APPROVE_C" > "${RESULT_DIR}/TC-1-1-04.tx_execute"

tc_end
