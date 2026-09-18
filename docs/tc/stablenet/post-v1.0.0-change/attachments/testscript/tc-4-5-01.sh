#!/bin/bash
# TC-4-5-01: Genesis 인증 계정 상태 — alloc.Extra Authorized 비트 검증
# 네트워크: Privatenet
# 선행조건: genesis.json에 Extra=0x4000000000000000 설정

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-4-5-01"

# ── Test Data ──
# AUTH_ACCOUNT: common.sh에서 상속 (VAL1)
check_not_tbd "AUTH_ACCOUNT" "$AUTH_ACCOUNT"

# ── Verification ──
echo -e "\n  ${BOLD}[Verification]${NC}"

# 1. GovCouncil authorized 등록
IS_AUTH_GC=$(cast call "$GC" 'isAuthorized(address)(bool)' "$AUTH_ACCOUNT" --rpc-url "$RPC" 2>/dev/null)
assert_eq 1 "GovCouncil isAuthorized" "$IS_AUTH_GC" "true"

# 2. AccountManager 확인
IS_AUTH_AM=$(cast call "$AM" 'isAuthorized(address)(bool)' "$AUTH_ACCOUNT" --rpc-url "$RPC" 2>/dev/null)
assert_eq 2 "AccountManager isAuthorized" "$IS_AUTH_AM" "true"

tc_end
