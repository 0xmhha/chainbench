#!/bin/bash
# TC-5-1-01 ~ TC-5-1-03: 빌드 검증
# 환경: 빌드 서버
# 선행조건: Go 1.23.12, gstable 소스코드

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-5-1-BUILD"

# ── Test Data ──
GSTABLE_SRC=${GSTABLE_SRC:-""}
if [ -z "$GSTABLE_SRC" ]; then
  tc_blocked "GSTABLE_SRC (gstable 소스코드 경로) 미설정"
fi
cd "$GSTABLE_SRC" || tc_blocked "소스코드 디렉토리 접근 실패: $GSTABLE_SRC"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# TC-5-1-01: 빌드 성공
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}── TC-5-1-01: make all ──${NC}"
make all 2>&1 | tee "${RESULT_DIR}/build.log"
BUILD_EXIT=${PIPESTATUS[0]}
assert_eq 1 "TC-5-1-01: make all 성공" "$BUILD_EXIT" "0"

BINARIES=$(ls build/bin/ 2>/dev/null | wc -l | tr -d ' ')
echo "  바이너리 수: $BINARIES"
assert_eq 2 "TC-5-1-01: 바이너리 수 == 13" "$BINARIES" "13"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# TC-5-1-02: make test 전체 통과
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}── TC-5-1-02: make test ──${NC}"
make test 2>&1 | tee "${RESULT_DIR}/test.log"
FAIL_COUNT=$(grep -c "^FAIL" "${RESULT_DIR}/test.log" 2>/dev/null || echo "0")
assert_eq 3 "TC-5-1-02: FAIL 카운트 == 0" "$FAIL_COUNT" "0"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# TC-5-1-03: Go 버전 확인
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}── TC-5-1-03: Go 버전 ──${NC}"
GO_VERSION=$(build/bin/gstable version 2>&1 | grep "Go Version" || echo "")
echo "  $GO_VERSION"
assert_contains 4 "TC-5-1-03: Go 1.23.12" "$GO_VERSION" "go1.23.12"

tc_end
