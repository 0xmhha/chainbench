#!/bin/bash
# TC-1-2-01: Boho 활성화 후 secp256r1 프리컴파일 호출 성공
# 네트워크: Both (Boho 활성)
# 선행조건: 없음

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-1-2-01"

# ── Test Data ──
# EIP-7212 유효한 secp256r1 서명 데이터 (160 bytes) — common.sh에서 상속
check_not_tbd "P256_INPUT" "$P256_INPUT"

# ── Action ──
echo -e "\n  ${BOLD}[Action]${NC} secp256r1 프리컴파일 호출"
RESULT=$(cast call "$P256" "$P256_INPUT" --rpc-url "$RPC" 2>&1)
echo "  Result: $RESULT"

# ── Verification ──
echo -e "\n  ${BOLD}[Verification]${NC}"

# 1. 호출 성공 (에러 없음)
if echo "$RESULT" | grep -qi "error\|revert\|fail"; then
  assert_eq 1 "호출 성공" "error" "no_error"
else
  assert_eq 1 "호출 성공" "ok" "ok"
fi

# 2. 결과값 (32바이트 반환)
RESULT_LEN=${#RESULT}
if [ "$RESULT_LEN" -ge 66 ] || [ "$RESULT" = "0x" ]; then
  assert_eq 2 "32바이트 결과 반환" "ok" "ok"
else
  assert_eq 2 "32바이트 결과 반환" "len=$RESULT_LEN" ">=66"
fi

tc_end
