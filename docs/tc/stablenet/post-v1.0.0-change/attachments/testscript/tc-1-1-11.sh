#!/bin/bash
# TC-1-1-11: Boho 하드포크 전후 GovMinter 바이트코드 변경 확인
# 네트워크: Both (Privatenet: bohoBlock=1, Testnet: 환경변수로 override)
# 선행조건: 없음

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-1-1-11"

# ── Test Data ──
# common.sh에서 BOHO_BLOCK 상속 (Privatenet=1)
V1_CODE_HASH=${V1_CODE_HASH:-""}
V2_CODE_HASH=${V2_CODE_HASH:-""}

# ── Setup ──
echo -e "  ${BOLD}[Setup]${NC} v1/v2 바이트코드 해시 사전 계산"
if [ -n "$V1_CODE_HASH" ] && [ -n "$V2_CODE_HASH" ]; then
  echo "  V1_CODE_HASH: $V1_CODE_HASH (사전 설정)"
  echo "  V2_CODE_HASH: $V2_CODE_HASH (사전 설정)"
else
  echo "  V1/V2 해시 미설정 — 체인에서 직접 비교"
fi

# ── Action ──
echo -e "\n  ${BOLD}[Action]${NC} 블록 $((BOHO_BLOCK + 1)) 대기"
wait_for_block $((BOHO_BLOCK + 1))

# ── Verification ──
echo -e "\n  ${BOLD}[Verification]${NC}"

# 블록에서 직접 코드 해시 계산
HASH_BEFORE=$(cast code "$GMI" --block $((BOHO_BLOCK - 1)) --rpc-url "$RPC" 2>/dev/null | cast keccak 2>/dev/null)
HASH_AFTER=$(cast code "$GMI" --block "$BOHO_BLOCK" --rpc-url "$RPC" 2>/dev/null | cast keccak 2>/dev/null)
echo "  블록 $((BOHO_BLOCK - 1)) code hash: $HASH_BEFORE"
echo "  블록 ${BOHO_BLOCK} code hash: $HASH_AFTER"

# 1. 블록 99 코드 == v1
if [ -n "$V1_CODE_HASH" ]; then
  assert_eq 1 "블록 $((BOHO_BLOCK-1)) 코드 == v1" "$HASH_BEFORE" "$V1_CODE_HASH"
else
  assert_non_empty 1 "블록 $((BOHO_BLOCK-1)) 코드 존재" "$HASH_BEFORE"
fi

# 2. 블록 100 코드 == v2
if [ -n "$V2_CODE_HASH" ]; then
  assert_eq 2 "블록 ${BOHO_BLOCK} 코드 == v2" "$HASH_AFTER" "$V2_CODE_HASH"
else
  assert_non_empty 2 "블록 ${BOHO_BLOCK} 코드 존재" "$HASH_AFTER"
fi

# 3. v1 != v2
assert_neq 3 "v1 != v2 (바이트코드 변경됨)" "$HASH_BEFORE" "$HASH_AFTER"

# 4. v2 함수 호출 가능
V2_CALL=$(cast call "$GMI" 'refundableBalance(address)(uint256)' 0x0000000000000000000000000000000000000000 \
  --block "$BOHO_BLOCK" --rpc-url "$RPC" 2>&1)
if echo "$V2_CALL" | grep -qi "error\|revert"; then
  assert_eq 4 "v2 함수 호출 가능" "error" "no_error"
else
  assert_eq 4 "v2 함수 호출 가능" "ok" "ok"
fi

# 후속 TC용 해시 저장
echo "$HASH_BEFORE" > "${RESULT_DIR}/TC-1-1-11.v1_hash"
echo "$HASH_AFTER" > "${RESULT_DIR}/TC-1-1-11.v2_hash"

tc_end
