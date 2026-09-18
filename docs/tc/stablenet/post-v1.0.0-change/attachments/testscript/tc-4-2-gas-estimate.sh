#!/bin/bash
# TC-4-2-01 ~ TC-4-2-03: 가스 추정 오류 수정 (AuthorizationList)
# 네트워크: Both
# 선행조건: 없음

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-4-2-GAS-ESTIMATE"

# ── 필수 변수 확인 ──
check_not_tbd "TEST_ACCOUNT" "$TEST_ACCOUNT"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# TC-4-2-01: AuthorizationList 포함/미포함 가스 차이
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}── TC-4-2-01: AuthorizationList 가스 차이 ──${NC}"

# AuthorizationList 포함
GAS_WITH=$(curl -s "$RPC" -d '{
  "jsonrpc":"2.0","id":1,"method":"eth_estimateGas","params":[{
    "from":"'"$TEST_ACCOUNT"'","to":"'"$TEST_ACCOUNT"'",
    "authorizationList":[{"chainId":"0x39a9","address":"'"$TEST_ACCOUNT"'","nonce":"0x0","yParity":"0x0","r":"0x0","s":"0x0"}]
  }]
}' | jq -r .result)

# AuthorizationList 미포함
GAS_WITHOUT=$(curl -s "$RPC" -d '{
  "jsonrpc":"2.0","id":1,"method":"eth_estimateGas","params":[{
    "from":"'"$TEST_ACCOUNT"'","to":"'"$TEST_ACCOUNT"'"
  }]
}' | jq -r .result)

echo "  GAS_WITH: $GAS_WITH"
echo "  GAS_WITHOUT: $GAS_WITHOUT"

if [ -n "$GAS_WITH" ] && [ "$GAS_WITH" != "null" ] && [ -n "$GAS_WITHOUT" ] && [ "$GAS_WITHOUT" != "null" ]; then
  GAS_WITH_DEC=$((16#${GAS_WITH#0x}))
  GAS_WITHOUT_DEC=$((16#${GAS_WITHOUT#0x}))
  GAS_DIFF=$((GAS_WITH_DEC - GAS_WITHOUT_DEC))
  echo "  차이: $GAS_DIFF (최소 12500 = TxAuthTupleGas)"
  assert_gte 1 "TC-4-2-01: 가스 차이 >= 12500" "$GAS_DIFF" 12500
else
  assert_eq 1 "TC-4-2-01: 가스 추정 응답" "null" "valid_hex"
fi

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# TC-4-2-02: 기본 전송 가스 == 21000
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}── TC-4-2-02: 기본 전송 가스 ──${NC}"
if [ -n "$GAS_WITHOUT" ] && [ "$GAS_WITHOUT" != "null" ]; then
  assert_eq 2 "TC-4-2-02: 기본 전송 가스 == 21000" "$GAS_WITHOUT_DEC" "21000"
else
  assert_eq 2 "TC-4-2-02: 가스 추정 응답" "null" "21000"
fi

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# TC-4-2-03: AuthorizationList 3개 → 차이 >= 37500
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo -e "\n  ${BOLD}── TC-4-2-03: AuthorizationList x3 ──${NC}"
GAS_WITH_3=$(curl -s "$RPC" -d '{
  "jsonrpc":"2.0","id":1,"method":"eth_estimateGas","params":[{
    "from":"'"$TEST_ACCOUNT"'","to":"'"$TEST_ACCOUNT"'",
    "authorizationList":[
      {"chainId":"0x39a9","address":"'"$TEST_ACCOUNT"'","nonce":"0x0","yParity":"0x0","r":"0x0","s":"0x0"},
      {"chainId":"0x39a9","address":"'"$TEST_ACCOUNT"'","nonce":"0x1","yParity":"0x0","r":"0x0","s":"0x0"},
      {"chainId":"0x39a9","address":"'"$TEST_ACCOUNT"'","nonce":"0x2","yParity":"0x0","r":"0x0","s":"0x0"}
    ]
  }]
}' | jq -r .result)

if [ -n "$GAS_WITH_3" ] && [ "$GAS_WITH_3" != "null" ]; then
  GAS_WITH_3_DEC=$((16#${GAS_WITH_3#0x}))
  GAS_DIFF_3=$((GAS_WITH_3_DEC - GAS_WITHOUT_DEC))
  echo "  GAS_WITH_3: $GAS_WITH_3 ($GAS_WITH_3_DEC)"
  echo "  차이: $GAS_DIFF_3 (최소 37500 = 12500 x 3)"
  assert_gte 3 "TC-4-2-03: 가스 차이 >= 37500" "$GAS_DIFF_3" 37500
else
  assert_eq 3 "TC-4-2-03: 가스 추정 응답" "null" "valid_hex"
fi

tc_end
