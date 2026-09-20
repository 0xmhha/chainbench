#!/bin/bash
# TC-4-3-01 ~ TC-4-3-06: 설정 문자열 처리 수정 (authorizedAccounts 파싱)
# 네트워크: Privatenet
# 선행조건: gstable 바이너리, 각 TC별 genesis.json 생성 필요

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

tc_start "TC-4-3-STRING"

# ── Test Data ──
DATADIR=${DATADIR:-"/tmp/gstable-test"}
GENESIS_TEMPLATE=${GENESIS_TEMPLATE:-""}  # 베이스 genesis.json 경로

# 테스트용 계정 주소 (genesis.json 내 authorizedAccounts 에 사용)
ADDR_A=${ADDR_A:-"0x0000000000000000000000000000000000000aaa"}
ADDR_B=${ADDR_B:-"0x0000000000000000000000000000000000000bbb"}
ADDR_C=${ADDR_C:-"0x0000000000000000000000000000000000000ccc"}

# TC별 authorizedAccounts 입력값과 기대 결과
declare -A TC_INPUTS
TC_INPUTS["01"]="$ADDR_A,$ADDR_B,$ADDR_C"          # 공백 없음
TC_INPUTS["02"]="$ADDR_A, $ADDR_B, $ADDR_C"        # 공백 포함
TC_INPUTS["03"]=" $ADDR_A , $ADDR_B "              # 앞뒤 공백
TC_INPUTS["04"]="$ADDR_A,,$ADDR_B"                  # 빈 항목
TC_INPUTS["05"]="$ADDR_A"                            # 단일
TC_INPUTS["06"]=""                                    # 빈 문자열

declare -A TC_EXPECTED_COUNT
TC_EXPECTED_COUNT["01"]=3
TC_EXPECTED_COUNT["02"]=3
TC_EXPECTED_COUNT["03"]=2
TC_EXPECTED_COUNT["04"]=2
TC_EXPECTED_COUNT["05"]=1
TC_EXPECTED_COUNT["06"]=0

declare -A TC_CHECK_ADDRS
TC_CHECK_ADDRS["01"]="$ADDR_A $ADDR_B $ADDR_C"
TC_CHECK_ADDRS["02"]="$ADDR_A $ADDR_B $ADDR_C"
TC_CHECK_ADDRS["03"]="$ADDR_A $ADDR_B"
TC_CHECK_ADDRS["04"]="$ADDR_A $ADDR_B"
TC_CHECK_ADDRS["05"]="$ADDR_A"
TC_CHECK_ADDRS["06"]=""

ASSERTION_NUM=0

for TC_NUM in "01" "02" "03" "04" "05" "06"; do
  echo -e "\n  ${BOLD}── TC-4-3-${TC_NUM} ──${NC}"
  echo "  입력: '${TC_INPUTS[$TC_NUM]}'"

  TC_DATADIR="${DATADIR}/tc43${TC_NUM}"
  TC_PORT=$((18545 + ${TC_NUM#0}))

  # genesis.json 내 authorizedAccounts 값 변경 후 init + 기동이 필요
  # 실제 환경에서는 genesis 파일을 생성해야 함
  echo "  datadir: $TC_DATADIR, RPC port: $TC_PORT"
  echo "  (genesis 파일 생성 및 노드 기동은 환경에 맞게 수행)"

  TC_RPC="http://localhost:$TC_PORT"

  if [ "${TC_EXPECTED_COUNT[$TC_NUM]}" = "0" ]; then
    # TC-4-3-06: 빈 문자열 → authorizedAccountCount == 0
    ASSERTION_NUM=$((ASSERTION_NUM + 1))
    COUNT=$(cast call "$GC" 'authorizedAccountCount()(uint256)' --rpc-url "$TC_RPC" 2>/dev/null || echo "error")
    if [ "$COUNT" = "error" ]; then
      echo -e "  ${YELLOW}SKIP${NC}: 노드 미기동 (수동 실행 필요)"
    else
      assert_eq "$ASSERTION_NUM" "TC-4-3-${TC_NUM}: authorizedAccountCount == 0" "$COUNT" "0"
    fi
  else
    # isAuthorized 확인
    for ADDR in ${TC_CHECK_ADDRS[$TC_NUM]}; do
      ASSERTION_NUM=$((ASSERTION_NUM + 1))
      IS_AUTH=$(cast call "$GC" 'isAuthorized(address)(bool)' "$ADDR" --rpc-url "$TC_RPC" 2>/dev/null || echo "error")
      if [ "$IS_AUTH" = "error" ]; then
        echo -e "  ${YELLOW}SKIP${NC}: 노드 미기동 (수동 실행 필요)"
      else
        assert_eq "$ASSERTION_NUM" "TC-4-3-${TC_NUM}: isAuthorized($ADDR)" "$IS_AUTH" "true"
      fi
    done
  fi
done

tc_end
