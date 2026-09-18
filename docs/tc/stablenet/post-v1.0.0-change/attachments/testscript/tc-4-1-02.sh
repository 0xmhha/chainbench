#!/bin/bash
# TC-4-1-02: CollectUpgrades → SetConfigFromChainConfig 경로 확인
# 네트워크: Both (Privatenet: bohoBlock=1, Testnet: 환경변수로 override)
# 선행조건: 체인 가동 중

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-4-1-02"

# ── Test Data ──
# common.sh에서 BOHO_BLOCK 상속 (Privatenet=1)
V1_CODE_HASH=${V1_CODE_HASH:-""}
V2_CODE_HASH=${V2_CODE_HASH:-""}

# TC-1-1-11 결과 재활용
if [ -z "$V1_CODE_HASH" ] && [ -f "${RESULT_DIR}/TC-1-1-11.v1_hash" ]; then
  V1_CODE_HASH=$(cat "${RESULT_DIR}/TC-1-1-11.v1_hash")
fi
if [ -z "$V2_CODE_HASH" ] && [ -f "${RESULT_DIR}/TC-1-1-11.v2_hash" ]; then
  V2_CODE_HASH=$(cat "${RESULT_DIR}/TC-1-1-11.v2_hash")
fi

# ── Setup ──
wait_for_block $((BOHO_BLOCK + 1))

# ── Verification ──
echo -e "\n  ${BOLD}[Verification]${NC}"

# 1. 블록 99 GovMinter == v1
HASH_99=$(cast code "$GMI" --block $((BOHO_BLOCK - 1)) --rpc-url "$RPC" 2>/dev/null | cast keccak 2>/dev/null)
if [ -n "$V1_CODE_HASH" ]; then
  assert_eq 1 "블록 $((BOHO_BLOCK-1)) GovMinter == v1" "$HASH_99" "$V1_CODE_HASH"
else
  assert_non_empty 1 "블록 $((BOHO_BLOCK-1)) GovMinter 코드 존재" "$HASH_99"
fi

# 2. 블록 100 GovMinter == v2
HASH_100=$(cast code "$GMI" --block "$BOHO_BLOCK" --rpc-url "$RPC" 2>/dev/null | cast keccak 2>/dev/null)
if [ -n "$V2_CODE_HASH" ]; then
  assert_eq 2 "블록 ${BOHO_BLOCK} GovMinter == v2" "$HASH_100" "$V2_CODE_HASH"
else
  assert_non_empty 2 "블록 ${BOHO_BLOCK} GovMinter 코드 존재" "$HASH_100"
  assert_neq 2 "v1 != v2" "$HASH_99" "$HASH_100"
fi

tc_end
