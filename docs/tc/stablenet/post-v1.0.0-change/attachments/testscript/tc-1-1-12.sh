#!/bin/bash
# TC-1-1-12: v2 업그레이드 후 기존 v1 상태 보존 확인
# 네트워크: Both (Privatenet: bohoBlock=1, Testnet: 환경변수로 override)
# 선행조건: 없음

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-1-1-12"

# ── Test Data ──
# common.sh에서 BOHO_BLOCK 상속 (Privatenet=1)

# ── Setup & Action ──
echo -e "  ${BOLD}[Setup]${NC} 블록 $((BOHO_BLOCK - 1)) 대기"
wait_for_block $((BOHO_BLOCK - 1))

echo -e "\n  ${BOLD}[Action]${NC} 블록 $((BOHO_BLOCK - 1)) 시점의 v1 상태 기록"
QUORUM_99=$(cast call "$GV" 'quorum()(uint256)' --block $((BOHO_BLOCK - 1)) --rpc-url "$RPC" 2>/dev/null)
GASTIP_99=$(cast call "$GV" 'gasTip()(uint256)' --block $((BOHO_BLOCK - 1)) --rpc-url "$RPC" 2>/dev/null)
echo "  quorum@$((BOHO_BLOCK-1)): $QUORUM_99"
echo "  gasTip@$((BOHO_BLOCK-1)): $GASTIP_99"

echo -e "  블록 ${BOHO_BLOCK} 대기"
wait_for_block $((BOHO_BLOCK + 1))

# ── Verification ──
echo -e "\n  ${BOLD}[Verification]${NC}"

# 1. quorum 보존
QUORUM_100=$(cast call "$GV" 'quorum()(uint256)' --block "$BOHO_BLOCK" --rpc-url "$RPC" 2>/dev/null)
assert_eq 1 "quorum 보존" "$QUORUM_100" "$QUORUM_99"

# 2. gasTip 보존
GASTIP_100=$(cast call "$GV" 'gasTip()(uint256)' --block "$BOHO_BLOCK" --rpc-url "$RPC" 2>/dev/null)
assert_eq 2 "gasTip 보존" "$GASTIP_100" "$GASTIP_99"

# 3. v2 신규 함수 호출 가능 (빈 매핑 = 0)
V2_CALL=$(cast call "$GMI" 'refundableBalance(address)(uint256)' "${TEST_ACCOUNT}" \
  --block "$BOHO_BLOCK" --rpc-url "$RPC" 2>&1)
if echo "$V2_CALL" | grep -qi "error\|revert"; then
  assert_eq 3 "v2 신규 함수 호출 가능" "error" "no_error"
else
  assert_eq 3 "v2 신규 함수 호출 가능" "ok" "ok"
fi

tc_end
