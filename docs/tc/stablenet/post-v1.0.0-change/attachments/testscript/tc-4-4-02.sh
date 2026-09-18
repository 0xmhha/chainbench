#!/bin/bash
# TC-4-4-02: 동일 블록 복수 하드포크 — Both
# 네트워크: Both (Privatenet: bohoBlock=1, Testnet: 환경변수로 override)
# 선행조건: 체인 가동, TC-1-1-11과 동일 검증

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-4-4-02"

# common.sh에서 BOHO_BLOCK 상속 (Privatenet=1)

# ── Setup ──
wait_for_block $((BOHO_BLOCK + 1))

# ── Verification ──
echo -e "\n  ${BOLD}[Verification]${NC}"

HASH_BEFORE=$(cast code "$GMI" --block $((BOHO_BLOCK - 1)) --rpc-url "$RPC" 2>/dev/null | cast keccak 2>/dev/null)
HASH_AFTER=$(cast code "$GMI" --block "$BOHO_BLOCK" --rpc-url "$RPC" 2>/dev/null | cast keccak 2>/dev/null)

echo "  블록 $((BOHO_BLOCK-1)) code hash: $HASH_BEFORE"
echo "  블록 ${BOHO_BLOCK} code hash: $HASH_AFTER"

# 1. v1 -> v2 전환 확인
assert_neq 1 "v1 != v2 바이트코드 변경" "$HASH_BEFORE" "$HASH_AFTER"

# 2. v2 함수 호출 가능
V2_CALL=$(cast call "$GMI" 'refundableBalance(address)(uint256)' 0x0000000000000000000000000000000000000000 \
  --block "$BOHO_BLOCK" --rpc-url "$RPC" 2>&1)
if echo "$V2_CALL" | grep -qi "error\|revert"; then
  assert_eq 2 "v2 함수 호출 가능" "error" "no_error"
else
  assert_eq 2 "v2 함수 호출 가능" "ok" "ok"
fi

tc_end
