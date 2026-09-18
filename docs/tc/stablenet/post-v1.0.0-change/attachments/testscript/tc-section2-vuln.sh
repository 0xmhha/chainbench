#!/bin/bash
# Section 2: 취약점 패치 단위 테스트
# 환경: 빌드 서버 (go test)
# 선행조건: Go 환경, gstable 소스코드 위치

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-SECTION2-VULN"

# ── Test Data ──
GSTABLE_SRC=${GSTABLE_SRC:-""}
if [ -z "$GSTABLE_SRC" ]; then
  tc_blocked "GSTABLE_SRC (gstable 소스코드 경로) 미설정"
fi

cd "$GSTABLE_SRC" || tc_blocked "소스코드 디렉토리 접근 실패: $GSTABLE_SRC"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 2-1: KZG
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}── 2-1: KZG fetcher ──${NC}"
KZG_RESULT=$(go test -v ./eth/fetcher/... 2>&1)
KZG_EXIT=$?
if [ $KZG_EXIT -eq 0 ]; then
  assert_eq 1 "2-1 KZG fetcher 테스트 통과" "pass" "pass"
else
  assert_eq 1 "2-1 KZG fetcher 테스트 통과" "fail(exit=$KZG_EXIT)" "pass"
  echo "  $KZG_RESULT" | tail -20
fi

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 2-2: ECIES
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}── 2-2: ECIES ──${NC}"
ECIES_RESULT=$(go test -v ./crypto/ecies/... ./p2p/rlpx/... 2>&1)
ECIES_EXIT=$?
if [ $ECIES_EXIT -eq 0 ]; then
  assert_eq 2 "2-2 ECIES 테스트 통과" "pass" "pass"
else
  assert_eq 2 "2-2 ECIES 테스트 통과" "fail(exit=$ECIES_EXIT)" "pass"
  echo "  $ECIES_RESULT" | tail -20
fi

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 2-3: secp256k1
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}── 2-3: secp256k1 ──${NC}"
SECP_RESULT=$(go test -v ./crypto/secp256k1/... 2>&1)
SECP_EXIT=$?
if [ $SECP_EXIT -eq 0 ]; then
  assert_eq 3 "2-3 secp256k1 테스트 통과" "pass" "pass"
else
  assert_eq 3 "2-3 secp256k1 테스트 통과" "fail(exit=$SECP_EXIT)" "pass"
  echo "  $SECP_RESULT" | tail -20
fi

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 2-4: P2P DoS
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}── 2-4: P2P DoS ──${NC}"
P2P_RESULT=$(go test -v ./p2p/tracker/... ./rlp/... ./eth/protocols/... 2>&1)
P2P_EXIT=$?
if [ $P2P_EXIT -eq 0 ]; then
  assert_eq 4 "2-4 P2P DoS 테스트 통과" "pass" "pass"
else
  assert_eq 4 "2-4 P2P DoS 테스트 통과" "fail(exit=$P2P_EXIT)" "pass"
  echo "  $P2P_RESULT" | tail -20
fi

tc_end
