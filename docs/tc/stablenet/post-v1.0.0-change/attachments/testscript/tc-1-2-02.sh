#!/bin/bash
# TC-1-2-02: Boho 이전에는 secp256r1 프리컴파일 미존재
# 네트워크: Both (Privatenet: bohoBlock=1, Testnet: 환경변수로 override)
# 선행조건: 없음

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-1-2-02"

# ── Test Data ──
# P256_INPUT, BOHO_BLOCK: common.sh에서 상속
check_not_tbd "P256_INPUT" "$P256_INPUT"

# ── Setup ──
echo -e "  ${BOLD}[Setup]${NC} 블록 $((BOHO_BLOCK + 1)) 대기"
wait_for_block $((BOHO_BLOCK + 1))

# ── Verification ──
echo -e "\n  ${BOLD}[Verification]${NC}"

# 1. 블록 99 호출 (Boho 미활성) → 빈 결과
RESULT_99=$(cast call "$P256" "$P256_INPUT" --block $((BOHO_BLOCK - 1)) --rpc-url "$RPC" 2>&1)
assert_empty 1 "블록 $((BOHO_BLOCK-1)) 호출 = 빈 결과" "$RESULT_99"

# 2. 블록 99 코드 없음
CODE_99=$(cast code "$P256" --block $((BOHO_BLOCK - 1)) --rpc-url "$RPC" 2>/dev/null)
assert_eq 2 "블록 $((BOHO_BLOCK-1)) 코드 없음" "$CODE_99" "0x"

# 3. 블록 100 호출 (Boho 활성) → 결과 반환
RESULT_100=$(cast call "$P256" "$P256_INPUT" --block "$BOHO_BLOCK" --rpc-url "$RPC" 2>&1)
assert_non_empty 3 "블록 ${BOHO_BLOCK} 호출 = 결과 반환" "$RESULT_100"

tc_end
