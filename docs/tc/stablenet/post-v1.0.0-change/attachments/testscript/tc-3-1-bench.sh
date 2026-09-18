#!/bin/bash
# TC-3-1-01 ~ TC-3-1-03: secp256k1 성능 벤치마크
# 환경: 빌드 서버 (Go 1.23.12)
# 선행조건: Go 환경, gstable 소스코드 위치

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-3-1-BENCH"

# ── Test Data ──
GSTABLE_SRC=${GSTABLE_SRC:-""}
if [ -z "$GSTABLE_SRC" ]; then
  tc_blocked "GSTABLE_SRC (gstable 소스코드 경로) 미설정"
fi
cd "$GSTABLE_SRC" || tc_blocked "소스코드 디렉토리 접근 실패: $GSTABLE_SRC"

BENCH_COUNT=${BENCH_COUNT:-5}

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# TC-3-1-01: 서명 생성 BenchmarkSign
# Pass: <= 25,000 ns/op | Fail: > 35,000 ns/op
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}── TC-3-1-01: BenchmarkSign ──${NC}"
SIGN_OUTPUT=$(go test -bench=BenchmarkSign -benchmem -count="$BENCH_COUNT" ./crypto/secp256k1/... 2>&1)
SIGN_NS=$(echo "$SIGN_OUTPUT" | grep "BenchmarkSign" | awk '{print $3}' | head -1 | tr -d ' ')
echo "  BenchmarkSign: ${SIGN_NS:-N/A} ns/op"
if [ -n "$SIGN_NS" ]; then
  assert_lte 1 "TC-3-1-01: BenchmarkSign <= 25000 ns/op" "${SIGN_NS%.*}" 25000
else
  assert_eq 1 "TC-3-1-01: BenchmarkSign 결과" "N/A" "<= 25000"
fi

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# TC-3-1-02: 서명 복원 BenchmarkRecover
# Pass: <= 38,000 ns/op | Fail: > 47,000 ns/op
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}── TC-3-1-02: BenchmarkRecover ──${NC}"
RECOVER_OUTPUT=$(go test -bench=BenchmarkRecover -benchmem -count="$BENCH_COUNT" ./crypto/secp256k1/... 2>&1)
RECOVER_NS=$(echo "$RECOVER_OUTPUT" | grep "BenchmarkRecover" | awk '{print $3}' | head -1 | tr -d ' ')
echo "  BenchmarkRecover: ${RECOVER_NS:-N/A} ns/op"
if [ -n "$RECOVER_NS" ]; then
  assert_lte 2 "TC-3-1-02: BenchmarkRecover <= 38000 ns/op" "${RECOVER_NS%.*}" 38000
else
  assert_eq 2 "TC-3-1-02: BenchmarkRecover 결과" "N/A" "<= 38000"
fi

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# TC-3-1-03: 서명 검증 BenchmarkVerifySignature
# Pass: <= 33,000 ns/op | Fail: > 41,000 ns/op
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}── TC-3-1-03: BenchmarkVerifySignature ──${NC}"
VERIFY_OUTPUT=$(go test -bench=BenchmarkVerifySignature -benchmem -count="$BENCH_COUNT" ./crypto/... 2>&1)
VERIFY_NS=$(echo "$VERIFY_OUTPUT" | grep "BenchmarkVerifySignature" | awk '{print $3}' | head -1 | tr -d ' ')
echo "  BenchmarkVerifySignature: ${VERIFY_NS:-N/A} ns/op"
if [ -n "$VERIFY_NS" ]; then
  assert_lte 3 "TC-3-1-03: BenchmarkVerifySignature <= 33000 ns/op" "${VERIFY_NS%.*}" 33000
else
  assert_eq 3 "TC-3-1-03: BenchmarkVerifySignature 결과" "N/A" "<= 33000"
fi

# 추가: BenchmarkEcrecoverSignature
ECRECOVER_OUTPUT=$(go test -bench=BenchmarkEcrecoverSignature -benchmem -count="$BENCH_COUNT" ./crypto/... 2>&1)
ECRECOVER_NS=$(echo "$ECRECOVER_OUTPUT" | grep "BenchmarkEcrecover" | awk '{print $3}' | head -1 | tr -d ' ')
echo "  BenchmarkEcrecoverSignature: ${ECRECOVER_NS:-N/A} ns/op"

tc_end
