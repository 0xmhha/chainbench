#!/bin/bash
# TC-1-2-03: 유효한 secp256r1 서명 검증 성공
# 네트워크: Both (Boho 활성)
# 선행조건: 없음

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-1-2-03"

# ── Test Data ──
# P-256 유효 서명 테스트 벡터 (160 bytes) — common.sh에서 상속
check_not_tbd "VALID_P256_INPUT" "$VALID_P256_INPUT"

# ── Verification ──
echo -e "\n  ${BOLD}[Verification]${NC}"

RESULT=$(cast call "$P256" "$VALID_P256_INPUT" --rpc-url "$RPC" 2>&1)
echo "  Result: $RESULT"

# 1. 검증 성공 (반환값 == 1)
EXPECTED="0x0000000000000000000000000000000000000000000000000000000000000001"
assert_eq 1 "secp256r1 검증 성공 (return 1)" "$RESULT" "$EXPECTED"

tc_end
