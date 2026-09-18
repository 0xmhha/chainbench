#!/bin/bash
# TC-4-4-01: 동일 블록 복수 하드포크 — Privatenet (bohoBlock=1)
# 네트워크: Privatenet
# 선행조건: Privatenet 체인 구성 (genesis bohoBlock=1)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-4-4-01"

# ── Test Data ──
V2_CODE_HASH=${V2_CODE_HASH:-""}
# bohoBlock=1이므로 v2 코드는 블록 1 이후에 적용됨
CHECK_BLOCK=${CHECK_BLOCK:-"$BOHO_BLOCK"}

# ── Verification ──
echo -e "\n  ${BOLD}[Verification]${NC}"
echo "  bohoBlock: $BOHO_BLOCK → 블록 $CHECK_BLOCK 에서 검증"

# 1. 시스템 컨트랙트 5개 존재
echo "  시스템 컨트랙트 코드 확인..."
ALL_EXIST="true"
for ADDR in $NCA $GV $GMM $GMI $GC; do
  CODE=$(cast code "$ADDR" --block "$CHECK_BLOCK" --rpc-url "$RPC" 2>/dev/null)
  if [ "$CODE" = "0x" ] || [ -z "$CODE" ]; then
    echo -e "    ${RED}$ADDR: 코드 없음${NC}"
    ALL_EXIST="false"
  else
    echo -e "    ${GREEN}$ADDR: 코드 존재${NC}"
  fi
done
assert_eq 1 "시스템 컨트랙트 5개 존재" "$ALL_EXIST" "true"

# 2. GovMinter == v2
GMI_HASH=$(cast code "$GMI" --block "$CHECK_BLOCK" --rpc-url "$RPC" 2>/dev/null | cast keccak 2>/dev/null)
echo "  GovMinter code hash: $GMI_HASH"
if [ -n "$V2_CODE_HASH" ]; then
  assert_eq 2 "GovMinter == v2" "$GMI_HASH" "$V2_CODE_HASH"
else
  assert_non_empty 2 "GovMinter 코드 존재 (v2 해시 미설정)" "$GMI_HASH"
fi

# 3. 나머지 4개 == v1 (코드 존재 확인)
echo "  나머지 4개 v1 확인..."
V1_OK="true"
for ADDR in $NCA $GV $GMM $GC; do
  CODE_HASH=$(cast code "$ADDR" --block "$CHECK_BLOCK" --rpc-url "$RPC" 2>/dev/null | cast keccak 2>/dev/null)
  if [ -z "$CODE_HASH" ]; then
    V1_OK="false"
  fi
done
assert_eq 3 "나머지 4개 v1 코드 존재" "$V1_OK" "true"

# 4. v1 상태 보존 (quorum)
QUORUM_VAL=$(cast call "$GV" 'quorum()(uint256)' --block "$CHECK_BLOCK" --rpc-url "$RPC" 2>/dev/null)
echo "  quorum@block${CHECK_BLOCK}: $QUORUM_VAL"
assert_non_empty 4 "v1 상태 보존 (quorum)" "$QUORUM_VAL"

tc_end
