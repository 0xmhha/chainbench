#!/bin/bash
# TC-1-2-05: 입력 길이 부족 시 처리
# 네트워크: Both (Boho 활성)
# 선행조건: 없음

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-1-2-05"

# ── Verification ──
echo -e "\n  ${BOLD}[Verification]${NC}"

# 1. 128바이트 입력 → 빈 결과
INPUT_128="0x$(python3 -c "print('00'*128)")"
RESULT_128=$(cast call "$P256" "$INPUT_128" --rpc-url "$RPC" 2>&1)
assert_empty 1 "128바이트 입력 = 빈 결과" "$RESULT_128"

# 2. 192바이트 입력 → 빈 결과
INPUT_192="0x$(python3 -c "print('00'*192)")"
RESULT_192=$(cast call "$P256" "$INPUT_192" --rpc-url "$RPC" 2>&1)
assert_empty 2 "192바이트 입력 = 빈 결과" "$RESULT_192"

# 3. 에러 없음 확인
HAS_ERROR="false"
if echo "$RESULT_128" | grep -qi "error\|revert"; then HAS_ERROR="true"; fi
if echo "$RESULT_192" | grep -qi "error\|revert"; then HAS_ERROR="true"; fi
assert_eq 3 "에러 없음" "$HAS_ERROR" "false"

tc_end
