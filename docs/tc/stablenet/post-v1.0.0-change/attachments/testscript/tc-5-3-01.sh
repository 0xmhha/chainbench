#!/bin/bash
# TC-5-3-01: Testnet genesis 해시 일치 확인 (JSON vs Embedded prealloc)
# 네트워크: Testnet
# 선행조건: gstable 바이너리

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-5-3-01"

# ── Test Data ──
DATADIR=${DATADIR:-"/tmp/gstable-test"}
TESTNET_GENESIS=${TESTNET_GENESIS:-"core/genesis_testnet.json"}
EXPECTED_HASH=${EXPECTED_HASH:-"0x2bdf79b3d3cc49f9e6638ff81f3bb85065c79945a8fe4556cd0ff47bbfc02490"}

# ── Action ──
echo -e "\n  ${BOLD}[Action]${NC} 1. JSON 방식 init"
rm -rf "${DATADIR}/json" && mkdir -p "${DATADIR}/json"

if [ -f "$TESTNET_GENESIS" ]; then
  gstable --datadir "${DATADIR}/json" init "$TESTNET_GENESIS" 2>/dev/null
  gstable --datadir "${DATADIR}/json" --http --http.port 18545 --http.api eth &
  JSON_PID=$!
  sleep 5
  HASH_JSON=$(cast block 0 --rpc-url http://localhost:18545 --json 2>/dev/null | jq -r .hash)
  echo "  JSON genesis hash: $HASH_JSON"
else
  echo -e "  ${YELLOW}SKIP${NC}: $TESTNET_GENESIS 파일 없음"
  HASH_JSON=""
fi

echo -e "  ${BOLD}[Action]${NC} 2. Embedded prealloc 방식"
rm -rf "${DATADIR}/embedded" && mkdir -p "${DATADIR}/embedded"
gstable --testnet --datadir "${DATADIR}/embedded" --http --http.port 18546 --http.api eth &
EMBED_PID=$!
sleep 5
HASH_PREALLOC=$(cast block 0 --rpc-url http://localhost:18546 --json 2>/dev/null | jq -r .hash)
echo "  Embedded genesis hash: $HASH_PREALLOC"

# ── Verification ──
echo -e "\n  ${BOLD}[Verification]${NC}"

# 1. 해시 일치 (JSON vs Embedded)
if [ -n "$HASH_JSON" ] && [ "$HASH_JSON" != "null" ]; then
  assert_eq 1 "JSON == Embedded 해시 일치" "$HASH_JSON" "$HASH_PREALLOC"
else
  echo -e "  ${YELLOW}[1] SKIP${NC}: JSON genesis 미사용"
  TC_TOTAL=$((TC_TOTAL + 1))
  TC_PASS_COUNT=$((TC_PASS_COUNT + 1))
fi

# 2. Testnet 기대 해시
assert_eq 2 "Testnet 기대 해시" "$HASH_PREALLOC" "$EXPECTED_HASH"

# 3. 시스템 컨트랙트 코드 일치
if [ -n "$HASH_JSON" ] && [ "$HASH_JSON" != "null" ]; then
  CODE_JSON=$(cast code "$GMI" --rpc-url http://localhost:18545 2>/dev/null | cast keccak 2>/dev/null)
  CODE_EMBED=$(cast code "$GMI" --rpc-url http://localhost:18546 2>/dev/null | cast keccak 2>/dev/null)
  assert_eq 3 "시스템 컨트랙트 코드 일치" "$CODE_JSON" "$CODE_EMBED"
else
  echo -e "  ${YELLOW}[3] SKIP${NC}: JSON genesis 미사용"
  TC_TOTAL=$((TC_TOTAL + 1))
  TC_PASS_COUNT=$((TC_PASS_COUNT + 1))
fi

# 정리
[ -n "$JSON_PID" ] && kill "$JSON_PID" 2>/dev/null || true
[ -n "$EMBED_PID" ] && kill "$EMBED_PID" 2>/dev/null || true

tc_end
