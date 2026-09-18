#!/bin/bash
# TC-4-6-01: Snap Sync EffectiveGasPrice 일치 검증
# 네트워크: Privatenet
# 선행조건: 체인 가동, Snap Sync EN 노드 동기화 완료
# Snap Sync EN: 172.21.132.8~10

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-4-6-01"

# ── Test Data ──
# AUTH_PRIVKEY: common.sh에서 상속 (VAL1)
# 기존 Snap Sync EN 노드 사용 (로컬 노드 기동 불필요)
SNAP_RPC=${SNAP_RPC:-"$EN_SNAP1_RPC"}

check_not_tbd "AUTH_PRIVKEY" "$AUTH_PRIVKEY"
check_not_tbd "TEST_ACCOUNT" "$TEST_ACCOUNT"

# ── Action ──
echo -e "\n  ${BOLD}[Action]${NC} 인증 계정으로 tx 전송 (maxPriorityFeePerGas=0)"
echo "  BP1_RPC: $BP1_RPC"
echo "  SNAP_RPC: $SNAP_RPC"

TX_AUTH=$(cast send "$TEST_ACCOUNT" --value 1 \
  --priority-gas-price 0 --gas-price "$MIN_FEE" \
  --private-key "$AUTH_PRIVKEY" \
  --rpc-url "$BP1_RPC" --json 2>/dev/null | jq -r .transactionHash)
echo "  TX: $TX_AUTH"

echo "  Snap Sync 노드 동기화 대기 (30초)..."
sleep 30

# ── Verification ──
echo -e "\n  ${BOLD}[Verification]${NC}"

# 1. BP receipt.effectiveGasPrice
EGP_BP=$(cast receipt "$TX_AUTH" --rpc-url "$BP1_RPC" --json 2>/dev/null | jq -r .effectiveGasPrice)
assert_non_empty 1 "BP effectiveGasPrice" "$EGP_BP"

# 2. Snap receipt.effectiveGasPrice == BP
EGP_SNAP=$(cast receipt "$TX_AUTH" --rpc-url "$SNAP_RPC" --json 2>/dev/null | jq -r .effectiveGasPrice)
assert_eq 2 "Snap effectiveGasPrice == BP" "$EGP_SNAP" "$EGP_BP"

tc_end
