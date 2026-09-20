#!/bin/bash
# TC-1-2-06: Boho 하드포크 활성 주소 목록에 secp256r1 포함 확인
# 네트워크: Both (Boho 활성)
# 선행조건: 없음

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-1-2-06"

# ── 필수 변수 확인 ──
check_not_tbd "TEST_PRIVKEY" "$TEST_PRIVKEY"
# VALID_P256_INPUT: common.sh에서 상속
check_not_tbd "VALID_P256_INPUT" "$VALID_P256_INPUT"

# ── Verification ──
echo -e "\n  ${BOLD}[Verification]${NC}"

# 1. value 전송 차단 (ErrValueTransferToPrecompile)
VALUE_RESULT=$(cast_send "$P256" --value 1 --private-key "$TEST_PRIVKEY" --rpc-url "$RPC" 2>&1 || true)
assert_error 1 "value 전송 차단" "$VALUE_RESULT"

# 2. value=0 호출은 정상
CALL_RESULT=$(cast call "$P256" "$VALID_P256_INPUT" --rpc-url "$RPC" 2>&1)
if echo "$CALL_RESULT" | grep -qi "error\|revert\|fail"; then
  assert_eq 2 "value=0 호출 정상" "error" "no_error"
else
  assert_eq 2 "value=0 호출 정상" "ok" "ok"
fi

tc_end
