#!/bin/bash
# TC-1-2-04: 잘못된 secp256r1 서명 검증 실패
# 네트워크: Both (Boho 활성)
# 선행조건: 없음

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-1-2-04"

# ── Test Data ──
# VALID_P256_INPUT 에서 r 값 1바이트 변조 — common.sh에서 상속
check_not_tbd "INVALID_P256_INPUT" "$INVALID_P256_INPUT"

# ── Verification ──
echo -e "\n  ${BOLD}[Verification]${NC}"

RESULT=$(cast call "$P256" "$INVALID_P256_INPUT" --rpc-url "$RPC" 2>&1)
echo "  Result: '$RESULT'"

# 1. 검증 실패 (빈 결과, 에러 아님)
assert_empty 1 "검증 실패 (빈 결과)" "$RESULT"

tc_end
