#!/bin/bash
# TC-1-3-01 ~ TC-1-3-06: 최소 가스비 하한선 적용
# 네트워크: Both
# 선행조건: 없음

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-1-3-GAS"

# ── Test Data ──
UNDER_FEE=$((MIN_FEE - 1))
OVER_FEE=$((MIN_FEE + 1))

# ── 필수 변수 확인 ──
check_not_tbd "TEST_ACCOUNT" "$TEST_ACCOUNT"
check_not_tbd "TEST_PRIVKEY" "$TEST_PRIVKEY"

echo "  MIN_FEE: $MIN_FEE"
echo "  UNDER_FEE: $UNDER_FEE"
echo "  OVER_FEE: $OVER_FEE"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# TC-1-3-01: gasFeeCap < MIN_FEE → 거부
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}── TC-1-3-01: gasFeeCap < MIN_FEE → 거부 ──${NC}"
TX_RESULT_01=$(cast send "$TEST_ACCOUNT" --value 1 \
  --gas-price "$UNDER_FEE" \
  --private-key "$TEST_PRIVKEY" \
  --rpc-url "$RPC" 2>&1 || true)
assert_contains 1 "TC-1-3-01: underpriced 거부" "$TX_RESULT_01" "underpriced"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# TC-1-3-02: gasFeeCap == MIN_FEE → 허용
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}── TC-1-3-02: gasFeeCap == MIN_FEE → 허용 ──${NC}"
TX_HASH_02=$(cast send "$TEST_ACCOUNT" --value 1 \
  --gas-price "$MIN_FEE" \
  --private-key "$TEST_PRIVKEY" \
  --rpc-url "$RPC" --json 2>/dev/null | jq -r .transactionHash)
STATUS_02=$(cast receipt "$TX_HASH_02" --rpc-url "$RPC" --json | jq -r .status 2>/dev/null)
assert_eq 2 "TC-1-3-02: MIN_FEE 전송 성공" "$STATUS_02" "0x1"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# TC-1-3-03: gasFeeCap > MIN_FEE → 허용
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}── TC-1-3-03: gasFeeCap > MIN_FEE → 허용 ──${NC}"
TX_HASH_03=$(cast send "$TEST_ACCOUNT" --value 1 \
  --gas-price "$OVER_FEE" \
  --private-key "$TEST_PRIVKEY" \
  --rpc-url "$RPC" --json 2>/dev/null | jq -r .transactionHash)
STATUS_03=$(cast receipt "$TX_HASH_03" --rpc-url "$RPC" --json | jq -r .status 2>/dev/null)
assert_eq 3 "TC-1-3-03: OVER_FEE 전송 성공" "$STATUS_03" "0x1"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# TC-1-3-04: LegacyTx
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}── TC-1-3-04: LegacyTx ──${NC}"

# 거부
TX_LEGACY_UNDER=$(cast send "$TEST_ACCOUNT" --value 1 --gas-price "$UNDER_FEE" --legacy \
  --private-key "$TEST_PRIVKEY" --rpc-url "$RPC" 2>&1 || true)
assert_contains 4 "TC-1-3-04: Legacy underpriced 거부" "$TX_LEGACY_UNDER" "underpriced"

# 허용
TX_LEGACY_OK=$(cast send "$TEST_ACCOUNT" --value 1 --gas-price "$MIN_FEE" --legacy \
  --private-key "$TEST_PRIVKEY" --rpc-url "$RPC" --json 2>/dev/null | jq -r .transactionHash)
STATUS_L=$(cast receipt "$TX_LEGACY_OK" --rpc-url "$RPC" --json | jq -r .status 2>/dev/null)
assert_eq 5 "TC-1-3-04: Legacy MIN_FEE 허용" "$STATUS_L" "0x1"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# TC-1-3-05: AccessListTx
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}── TC-1-3-05: AccessListTx ──${NC}"

TX_AL_UNDER=$(cast send "$TEST_ACCOUNT" --value 1 --gas-price "$UNDER_FEE" \
  --private-key "$TEST_PRIVKEY" --rpc-url "$RPC" 2>&1 || true)
assert_contains 6 "TC-1-3-05: AccessList underpriced 거부" "$TX_AL_UNDER" "underpriced"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# TC-1-3-06: DynamicFeeTx 2단계 검증
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}── TC-1-3-06: DynamicFeeTx ──${NC}"

# GasFeeCap 미달
TX_DYN_FEE=$(cast send "$TEST_ACCOUNT" --value 1 \
  --priority-gas-price "$INITIAL_GAS_TIP" \
  --gas-price "$UNDER_FEE" \
  --private-key "$TEST_PRIVKEY" --rpc-url "$RPC" 2>&1 || true)
assert_contains 7 "TC-1-3-06: DynFee GasFeeCap 미달" "$TX_DYN_FEE" "underpriced"

# GasTipCap 미달
UNDER_TIP=$((INITIAL_GAS_TIP - 1))
TX_DYN_TIP=$(cast send "$TEST_ACCOUNT" --value 1 \
  --priority-gas-price "$UNDER_TIP" \
  --gas-price "$MIN_FEE" \
  --private-key "$TEST_PRIVKEY" --rpc-url "$RPC" 2>&1 || true)
assert_contains 8 "TC-1-3-06: DynFee GasTipCap 미달" "$TX_DYN_TIP" "underpriced"

tc_end
