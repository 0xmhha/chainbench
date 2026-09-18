#!/bin/bash
# TC-5-2-06: 미지원 시스템 컨트랙트 버전 init 실패 확인
# 네트워크: Privatenet
# 선행조건: gstable 바이너리, version="v99" genesis.json

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-5-2-06"

# ── Test Data ──
DATADIR=${DATADIR:-"/tmp/gstable-test"}
GENESIS_BAD_VER=${GENESIS_BAD_VER:-"genesis_bad_version.json"}

if [ ! -f "$GENESIS_BAD_VER" ]; then
  tc_blocked "Genesis 파일 없음: $GENESIS_BAD_VER (\"version\": \"v99\" 설정 필요)"
fi

# ── Action ──
echo -e "\n  ${BOLD}[Action]${NC} 미지원 버전(v99) genesis init"
rm -rf "${DATADIR}/tc5206"
mkdir -p "${DATADIR}/tc5206"
RESULT=$(gstable --datadir "${DATADIR}/tc5206" init "$GENESIS_BAD_VER" 2>&1 || true)
EXIT_CODE=$?
echo "  Result: ${RESULT:0:200}"
echo "  Exit code: $EXIT_CODE"

# ── Verification ──
echo -e "\n  ${BOLD}[Verification]${NC}"

# 1. 에러 메시지 확인 (unknown version)
assert_error 1 "미지원 버전 에러" "$RESULT"

# 2. 종료 코드 != 0
assert_neq 2 "종료 코드 != 0" "$EXIT_CODE" "0"

tc_end
