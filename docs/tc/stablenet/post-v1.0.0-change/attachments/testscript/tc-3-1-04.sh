#!/bin/bash
# TC-3-1-04: Full Sync 검증
# 네트워크: Privatenet
# 선행조건: 체인 가동 중, Full Sync EN 노드 가동 중
# Full Sync EN: 172.21.132.11~14

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-3-1-04"

# ── Test Data ──
# 기존 Full Sync EN 노드 사용 (로컬 노드 기동 불필요)
SYNC_RPC=${SYNC_RPC:-"$EN_FULL1_RPC"}

# ── Action ──
echo -e "\n  ${BOLD}[Action]${NC} Full Sync EN 노드 동기화 상태 확인"
echo "  BP1_RPC: $BP1_RPC"
echo "  SYNC_RPC: $SYNC_RPC"

echo -e "  ${BOLD}[Action]${NC} 동기화 완료 대기..."
TIMEOUT=${SYNC_TIMEOUT:-600}  # 10분 타임아웃
ELAPSED=0
while true; do
  SYNC_STATUS=$(cast rpc eth_syncing --rpc-url "$SYNC_RPC" 2>&1 || echo "connecting")
  if [ "$SYNC_STATUS" = "false" ]; then
    echo -e "  ${GREEN}동기화 완료${NC}"
    break
  fi
  if [ "$ELAPSED" -ge "$TIMEOUT" ]; then
    echo -e "  ${YELLOW}타임아웃 ($TIMEOUT초)${NC}"
    break
  fi
  sleep 10
  ELAPSED=$((ELAPSED + 10))
  echo -e "  ${CYAN}대기 중... (${ELAPSED}s) sync=$SYNC_STATUS${NC}"
done

# ── Verification ──
echo -e "\n  ${BOLD}[Verification]${NC}"

# 1. 동기화 완료
FINAL_SYNC=$(cast rpc eth_syncing --rpc-url "$SYNC_RPC" 2>&1)
assert_eq 1 "동기화 완료" "$FINAL_SYNC" "false"

# 2. 블록 번호 일치 (±2)
BLOCK_BP=$(cast block-number --rpc-url "$BP1_RPC" 2>/dev/null || echo "0")
BLOCK_SYNC=$(cast block-number --rpc-url "$SYNC_RPC" 2>/dev/null || echo "0")
DIFF=$((BLOCK_BP - BLOCK_SYNC))
if [ "$DIFF" -lt 0 ]; then DIFF=$((-DIFF)); fi
echo "  BP block: $BLOCK_BP, Sync block: $BLOCK_SYNC, diff: $DIFF"
assert_lte 2 "블록 번호 차이 <= 2" "$DIFF" 2

tc_end
