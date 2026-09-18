#!/bin/bash
# TC-4-1-01: 체인 설정 초기화 오류 수정 — 노드 정상 기동
# 네트워크: Privatenet
# 선행조건: Privatenet 체인 구성 완료

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-4-1-01"

# ── Verification ──
echo -e "\n  ${BOLD}[Verification]${NC}"

# 1. 노드 정상 기동 (블록 증가)
BLOCK1=$(cast block-number --rpc-url "$RPC" 2>/dev/null || echo "0")
sleep 3
BLOCK2=$(cast block-number --rpc-url "$RPC" 2>/dev/null || echo "0")
echo "  block1: $BLOCK1, block2: $BLOCK2 (3초 후)"

if [ "$BLOCK2" -gt "$BLOCK1" ] 2>/dev/null; then
  assert_eq 1 "노드 정상 기동 (블록 증가)" "increasing" "increasing"
else
  assert_eq 1 "노드 정상 기동 (블록 증가)" "stalled($BLOCK1->$BLOCK2)" "increasing"
fi

# 2. 블록 간격 ~1초
T1=$(cast block "$BLOCK1" --rpc-url "$RPC" --json 2>/dev/null | jq -r .timestamp)
T2=$(cast block "$BLOCK2" --rpc-url "$RPC" --json 2>/dev/null | jq -r .timestamp)
if [ -n "$T1" ] && [ -n "$T2" ]; then
  T1_DEC=$((16#${T1#0x}))
  T2_DEC=$((16#${T2#0x}))
  TIME_DIFF=$((T2_DEC - T1_DEC))
  EXPECTED_DIFF=$((BLOCK2 - BLOCK1))
  TOLERANCE=$((EXPECTED_DIFF + 1))
  echo "  timestamp diff: $TIME_DIFF, expected ~$EXPECTED_DIFF"
  assert_lte 2 "블록 간격 ±1초" "$TIME_DIFF" "$TOLERANCE"
else
  assert_eq 2 "블록 간격 확인" "timestamp_unavailable" "~1s"
fi

# 3. 에폭 전환 (10블록)
EPOCH_INFO=$(curl -s "$RPC" -d '{"jsonrpc":"2.0","id":1,"method":"istanbul_getWbftExtraInfo","params":["0x9"]}' | jq -r '.result.epochInfo' 2>/dev/null)
assert_non_empty 3 "에폭 전환 (블록 9)" "$EPOCH_INFO"

tc_end
