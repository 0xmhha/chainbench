#!/bin/bash
# TC-4-5-09: 미정의 Extra 비트 설정 시 init 실패 검증
# 네트워크: Privatenet
# 선행조건: gstable 바이너리, 잘못된 Extra 비트를 가진 genesis.json

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-4-5-09"

# ── Test Data ──
DATADIR=${DATADIR:-"/tmp/gstable-test"}
GENESIS_INVALID=${GENESIS_INVALID:-"genesis_invalid_extra.json"}

if [ ! -f "$GENESIS_INVALID" ]; then
  tc_blocked "Genesis 파일 없음: $GENESIS_INVALID (bit 61 = 0x2000000000000000 설정 필요)"
fi

# ── Action ──
echo -e "\n  ${BOLD}[Action]${NC} 미정의 Extra 비트로 init"
rm -rf "${DATADIR}/tc4509"
mkdir -p "${DATADIR}/tc4509"
RESULT=$(gstable --datadir "${DATADIR}/tc4509" init "$GENESIS_INVALID" 2>&1 || true)
EXIT_CODE=$?
echo "  Result: ${RESULT:0:200}"
echo "  Exit code: $EXIT_CODE"

# ── Verification ──
echo -e "\n  ${BOLD}[Verification]${NC}"

# 1. 초기화 실패 메시지
assert_contains 1 "unknown bits 에러" "$RESULT" "unknown bits set in account extra"

# 2. 종료 코드 != 0
assert_neq 2 "종료 코드 != 0" "$EXIT_CODE" "0"

tc_end
