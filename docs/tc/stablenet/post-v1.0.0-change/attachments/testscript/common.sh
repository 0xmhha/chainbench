#!/bin/bash
# common.sh - 하드포크 테스트 공통 환경변수 & 유틸리티 함수
# 모든 TC 스크립트에서 source 하여 사용

set -euo pipefail

# ===================== 색상 =====================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# ===================== 환경 변수 =====================
export RPC=${RPC:-http://172.21.132.1:8601}
# Validator 노드 RPC (172.21.132.1~7, HTTPPort=8601)
export BP1_RPC=${BP1_RPC:-http://172.21.132.1:8601}
export BP2_RPC=${BP2_RPC:-http://172.21.132.2:8601}
export BP3_RPC=${BP3_RPC:-http://172.21.132.3:8601}
# EN 노드 - Snap Sync (172.21.132.8~10)
export EN_SNAP1_RPC=${EN_SNAP1_RPC:-http://172.21.132.8:8601}
export EN_SNAP2_RPC=${EN_SNAP2_RPC:-http://172.21.132.9:8601}
export EN_SNAP3_RPC=${EN_SNAP3_RPC:-http://172.21.132.10:8601}
# EN 노드 - Full Sync (172.21.132.11~14)
export EN_FULL1_RPC=${EN_FULL1_RPC:-http://172.21.132.11:8601}
export EN_FULL2_RPC=${EN_FULL2_RPC:-http://172.21.132.12:8601}
export EN_FULL3_RPC=${EN_FULL3_RPC:-http://172.21.132.13:8601}
export EN_FULL4_RPC=${EN_FULL4_RPC:-http://172.21.132.14:8601}
# PN (EN + Bootnode, 172.21.132.15)
export PN_RPC=${PN_RPC:-http://172.21.132.15:8601}
# 호환 alias
export EN1_RPC=${EN1_RPC:-$EN_SNAP1_RPC}

# 시스템 컨트랙트 주소 (20 bytes = 0x + 40 hex)
export NCA="0x0000000000000000000000000000000000001000"  # NativeCoinAdapter
export GV="0x0000000000000000000000000000000000001001"   # GovValidator
export GMM="0x0000000000000000000000000000000000001002"  # GovMasterMinter
export GMI="0x0000000000000000000000000000000000001003"  # GovMinter
export GC="0x0000000000000000000000000000000000001004"   # GovCouncil

# Native manager / 프리컴파일 주소
export AM="0x0000000000000000000000000000000000b00003"   # AccountManager
export P256="0x0000000000000000000000000000000000000100" # secp256r1 precompile (Boho)

# 가스 상수 (decimal)
export MIN_BASE_FEE=20000000000000           # 20 Twei
export INITIAL_GAS_TIP=27600000000000        # 27.6 Twei
export MIN_FEE=$((MIN_BASE_FEE + INITIAL_GAS_TIP))  # 47.6 Twei

# 체인 설정 (genesis.json 기준)
export CHAIN_ID=${CHAIN_ID:-14761}
export BOHO_BLOCK=${BOHO_BLOCK:-1}        # genesis "bohoBlock": 1
export QUORUM=${QUORUM:-2}                 # GovMinter/GovValidator quorum
export EXPIRY=${EXPIRY:-3}                  # 제안 만료 시간 (초, genesis 설정에 맞춤)

# ── Validator 계정 (genesis.json validators + nodekey) ──
# BP 노드 1 (172.21.132.1)
export VAL1_ADDR="0x518b3Efa7dB538F29615Cb9d76f4ac234EBE5893"
export VAL1_KEY="<REDACTED_PRIVATE_KEY>"
# BP 노드 2 (172.21.132.2)
export VAL2_ADDR="0xD76975b29BDE03F4C644851393D988CcEa9d1471"
export VAL2_KEY="<REDACTED_PRIVATE_KEY>"
# BP 노드 3 (172.21.132.3)
export VAL3_ADDR="0x0f51a9c1E728CAfD808F570773b4DcA201E0892D"
export VAL3_KEY="<REDACTED_PRIVATE_KEY>"
# BP 노드 4~7 (키 미제공 — 투표 TC 실행 시 환경변수로 주입)
export VAL4_ADDR="0x5b3267f014De8044FFD4d45A575Bdfe1e6395060"
export VAL4_KEY=${VAL4_KEY:-""}
export VAL5_ADDR="0xd997f9A99FB33a63E07220F42143F3eE0DEEDfc0"
export VAL5_KEY=${VAL5_KEY:-""}
export VAL6_ADDR="0x7379433952e44Ee51D313A4581F0bB79aa6C4ff2"
export VAL6_KEY=${VAL6_KEY:-""}
export VAL7_ADDR="0xb0a24793C36c8C6489B20f213bD735B9B6749447"
export VAL7_KEY=${VAL7_KEY:-""}

# ── 기본 테스트 계정 (Validator 1 기본) ──
export TEST_ACCOUNT=${TEST_ACCOUNT:-"$VAL1_ADDR"}
export TEST_PRIVKEY=${TEST_PRIVKEY:-"$VAL1_KEY"}
export MEMBER_ACCOUNT=${MEMBER_ACCOUNT:-"$VAL1_ADDR"}
export MEMBER_PRIVKEY=${MEMBER_PRIVKEY:-"$VAL1_KEY"}

# ── 멤버 계정 (TC별로 override 가능) ──
export MEMBER_A=${MEMBER_A:-"$VAL1_ADDR"}
export MEMBER_A_KEY=${MEMBER_A_KEY:-"$VAL1_KEY"}
export MEMBER_B=${MEMBER_B:-"$VAL2_ADDR"}
export MEMBER_B_KEY=${MEMBER_B_KEY:-"$VAL2_KEY"}
export MEMBER_C=${MEMBER_C:-"$VAL3_ADDR"}
export MEMBER_C_KEY=${MEMBER_C_KEY:-"$VAL3_KEY"}
# MEMBER_D/E: quorum=5 투표 시 필요 (VAL4_KEY, VAL5_KEY 환경변수 주입)
export MEMBER_D=${MEMBER_D:-"$VAL4_ADDR"}
export MEMBER_D_KEY=${MEMBER_D_KEY:-"$VAL4_KEY"}
export MEMBER_E=${MEMBER_E:-"$VAL5_ADDR"}
export MEMBER_E_KEY=${MEMBER_E_KEY:-"$VAL5_KEY"}

# ── 인증 계정 (TC-4-5, TC-4-6 용) ──
export AUTH_ACCOUNT=${AUTH_ACCOUNT:-"$VAL1_ADDR"}
export AUTH_PRIVKEY=${AUTH_PRIVKEY:-"$VAL1_KEY"}

# ── secp256r1 (P-256) 테스트 벡터 — RFC 6979 A.2.5 ──
# hash(32) || r(32) || s(32) || x(32) || y(32) = 160 bytes
export VALID_P256_INPUT=${VALID_P256_INPUT:-"0xaf2bdbe1aa9b6ec1e2ade1d694f41fc71a831d0268e9891562113d8a62add1bfefd48b2aacb6a8fd1140dd9cd45e81d69d2c877b56aaf991c34d0ea84eaf3716f7cb1c942d657c41d436c7a1b6e29f65f3e900dbb9aff4064dc4ab2f843acda860fed4ba255a9d31c961eb74c6356d68c049b8923b61fa6ce669622e60f29fb67903fe1008b8bc99a41ae9e95628bc64f2f1b20c2d7e9f5177a3c294d4462299"}
export P256_INPUT=${P256_INPUT:-"$VALID_P256_INPUT"}
# r 값 1바이트 변조 (마지막 6→7) → 검증 실패 기대
export INVALID_P256_INPUT=${INVALID_P256_INPUT:-"0xaf2bdbe1aa9b6ec1e2ade1d694f41fc71a831d0268e9891562113d8a62add1bfefd48b2aacb6a8fd1140dd9cd45e81d69d2c877b56aaf991c34d0ea84eaf3717f7cb1c942d657c41d436c7a1b6e29f65f3e900dbb9aff4064dc4ab2f843acda860fed4ba255a9d31c961eb74c6356d68c049b8923b61fa6ce669622e60f29fb67903fe1008b8bc99a41ae9e95628bc64f2f1b20c2d7e9f5177a3c294d4462299"}

# 이벤트 시그니처 (lazy 계산 — cast 필요)
BURN_REFUND_CLAIMED_SIG=""
BURN_DEPOSIT_REFUNDED_SIG=""

# ===================== 결과 디렉토리 =====================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESULT_DIR="${SCRIPT_DIR}/results"
mkdir -p "$RESULT_DIR"

# ===================== 테스트 상태 =====================
TC_NAME=""
TC_PASS_COUNT=0
TC_FAIL_COUNT=0
TC_TOTAL=0

# ===================== 유틸리티 함수 =====================

tc_start() {
  TC_NAME="$1"
  TC_PASS_COUNT=0
  TC_FAIL_COUNT=0
  TC_TOTAL=0
  echo ""
  echo -e "${BOLD}${CYAN}========================================${NC}"
  echo -e "${BOLD}${CYAN}  TC: ${TC_NAME}${NC}"
  echo -e "${BOLD}${CYAN}========================================${NC}"
  echo -e "  시작 시각: $(date '+%Y-%m-%d %H:%M:%S')"
  echo ""
}

tc_end() {
  echo ""
  echo -e "${BOLD}  ── 결과 요약 ──${NC}"
  echo -e "  전체: ${TC_TOTAL}  |  Pass: ${GREEN}${TC_PASS_COUNT}${NC}  |  Fail: ${RED}${TC_FAIL_COUNT}${NC}"

  local status
  if [ "$TC_FAIL_COUNT" -eq 0 ] && [ "$TC_TOTAL" -gt 0 ]; then
    status="PASS"
    echo -e "  ${GREEN}${BOLD}>>> PASS <<<${NC}"
  else
    status="FAIL"
    echo -e "  ${RED}${BOLD}>>> FAIL <<<${NC}"
  fi

  # 결과 파일 기록
  local result_file="${RESULT_DIR}/${TC_NAME}.result"
  cat > "$result_file" <<EOF
tc=${TC_NAME}
status=${status}
total=${TC_TOTAL}
pass=${TC_PASS_COUNT}
fail=${TC_FAIL_COUNT}
timestamp=$(date '+%Y-%m-%d %H:%M:%S')
EOF
  echo -e "  종료 시각: $(date '+%Y-%m-%d %H:%M:%S')"
  echo -e "${CYAN}========================================${NC}"
  echo ""

  if [ "$status" = "FAIL" ]; then
    return 1
  fi
  return 0
}

tc_blocked() {
  local reason="$1"
  echo -e "  ${YELLOW}BLOCKED${NC}: ${reason}"
  local result_file="${RESULT_DIR}/${TC_NAME}.result"
  cat > "$result_file" <<EOF
tc=${TC_NAME}
status=BLOCKED
reason=${reason}
timestamp=$(date '+%Y-%m-%d %H:%M:%S')
EOF
  echo -e "${CYAN}========================================${NC}"
  exit 2
}

# assert_eq: 정확한 일치 검증
# 사용: assert_eq <번호> <설명> <실제값> <기대값>
assert_eq() {
  local num="$1"
  local desc="$2"
  local actual="$3"
  local expected="$4"

  TC_TOTAL=$((TC_TOTAL + 1))
  if [ "$actual" = "$expected" ]; then
    TC_PASS_COUNT=$((TC_PASS_COUNT + 1))
    echo -e "  [${num}] ${GREEN}PASS${NC} - ${desc}"
    echo -e "       expected: ${expected}"
  else
    TC_FAIL_COUNT=$((TC_FAIL_COUNT + 1))
    echo -e "  [${num}] ${RED}FAIL${NC} - ${desc}"
    echo -e "       expected: ${expected}"
    echo -e "       actual:   ${actual}"
  fi
}

# assert_neq: 불일치 검증
assert_neq() {
  local num="$1"
  local desc="$2"
  local actual="$3"
  local unexpected="$4"

  TC_TOTAL=$((TC_TOTAL + 1))
  if [ "$actual" != "$unexpected" ]; then
    TC_PASS_COUNT=$((TC_PASS_COUNT + 1))
    echo -e "  [${num}] ${GREEN}PASS${NC} - ${desc}"
    echo -e "       actual: ${actual} (!= ${unexpected})"
  else
    TC_FAIL_COUNT=$((TC_FAIL_COUNT + 1))
    echo -e "  [${num}] ${RED}FAIL${NC} - ${desc}"
    echo -e "       should not be: ${unexpected}"
    echo -e "       actual:        ${actual}"
  fi
}

# assert_contains: 문자열 포함 검증
assert_contains() {
  local num="$1"
  local desc="$2"
  local haystack="$3"
  local needle="$4"

  TC_TOTAL=$((TC_TOTAL + 1))
  if echo "$haystack" | grep -q "$needle"; then
    TC_PASS_COUNT=$((TC_PASS_COUNT + 1))
    echo -e "  [${num}] ${GREEN}PASS${NC} - ${desc}"
    echo -e "       contains: ${needle}"
  else
    TC_FAIL_COUNT=$((TC_FAIL_COUNT + 1))
    echo -e "  [${num}] ${RED}FAIL${NC} - ${desc}"
    echo -e "       expected to contain: ${needle}"
    echo -e "       actual: ${haystack:0:200}"
  fi
}

# assert_not_contains: 문자열 미포함 검증
assert_not_contains() {
  local num="$1"
  local desc="$2"
  local haystack="$3"
  local needle="$4"

  TC_TOTAL=$((TC_TOTAL + 1))
  if ! echo "$haystack" | grep -q "$needle"; then
    TC_PASS_COUNT=$((TC_PASS_COUNT + 1))
    echo -e "  [${num}] ${GREEN}PASS${NC} - ${desc}"
    echo -e "       does not contain: ${needle}"
  else
    TC_FAIL_COUNT=$((TC_FAIL_COUNT + 1))
    echo -e "  [${num}] ${RED}FAIL${NC} - ${desc}"
    echo -e "       should not contain: ${needle}"
    echo -e "       actual: ${haystack:0:200}"
  fi
}

# assert_non_empty: 비어있지 않은지 검증
assert_non_empty() {
  local num="$1"
  local desc="$2"
  local value="$3"

  TC_TOTAL=$((TC_TOTAL + 1))
  if [ -n "$value" ] && [ "$value" != "0x" ] && [ "$value" != "null" ]; then
    TC_PASS_COUNT=$((TC_PASS_COUNT + 1))
    echo -e "  [${num}] ${GREEN}PASS${NC} - ${desc}"
    echo -e "       value: ${value:0:80}"
  else
    TC_FAIL_COUNT=$((TC_FAIL_COUNT + 1))
    echo -e "  [${num}] ${RED}FAIL${NC} - ${desc}"
    echo -e "       expected: non-empty"
    echo -e "       actual: '${value}'"
  fi
}

# assert_empty: 비어있는지 검증 (0x 또는 빈값)
assert_empty() {
  local num="$1"
  local desc="$2"
  local value="$3"

  TC_TOTAL=$((TC_TOTAL + 1))
  if [ -z "$value" ] || [ "$value" = "0x" ] || [ "$value" = "null" ] || [ "$value" = "" ]; then
    TC_PASS_COUNT=$((TC_PASS_COUNT + 1))
    echo -e "  [${num}] ${GREEN}PASS${NC} - ${desc}"
    echo -e "       value: '${value}' (empty)"
  else
    TC_FAIL_COUNT=$((TC_FAIL_COUNT + 1))
    echo -e "  [${num}] ${RED}FAIL${NC} - ${desc}"
    echo -e "       expected: empty/0x/null"
    echo -e "       actual: '${value:0:80}'"
  fi
}

# assert_gte: 크거나 같은지 검증 (bc 사용 — big number safe)
assert_gte() {
  local num="$1"
  local desc="$2"
  local actual="$3"
  local threshold="$4"

  TC_TOTAL=$((TC_TOTAL + 1))
  if [ "$(echo "${actual:-0} >= ${threshold:-0}" | bc 2>/dev/null)" = "1" ]; then
    TC_PASS_COUNT=$((TC_PASS_COUNT + 1))
    echo -e "  [${num}] ${GREEN}PASS${NC} - ${desc}"
    echo -e "       actual: ${actual} (>= ${threshold})"
  else
    TC_FAIL_COUNT=$((TC_FAIL_COUNT + 1))
    echo -e "  [${num}] ${RED}FAIL${NC} - ${desc}"
    echo -e "       expected: >= ${threshold}"
    echo -e "       actual: ${actual}"
  fi
}

# assert_lte: 작거나 같은지 검증 (bc 사용 — big number safe)
assert_lte() {
  local num="$1"
  local desc="$2"
  local actual="$3"
  local threshold="$4"

  TC_TOTAL=$((TC_TOTAL + 1))
  if [ "$(echo "${actual:-0} <= ${threshold:-0}" | bc 2>/dev/null)" = "1" ]; then
    TC_PASS_COUNT=$((TC_PASS_COUNT + 1))
    echo -e "  [${num}] ${GREEN}PASS${NC} - ${desc}"
    echo -e "       actual: ${actual} (<= ${threshold})"
  else
    TC_FAIL_COUNT=$((TC_FAIL_COUNT + 1))
    echo -e "  [${num}] ${RED}FAIL${NC} - ${desc}"
    echo -e "       expected: <= ${threshold}"
    echo -e "       actual: ${actual}"
  fi
}

# assert_error: 에러 발생 검증
assert_error() {
  local num="$1"
  local desc="$2"
  local output="$3"

  TC_TOTAL=$((TC_TOTAL + 1))
  if echo "$output" | grep -qi "error\|revert\|fail\|exception"; then
    TC_PASS_COUNT=$((TC_PASS_COUNT + 1))
    echo -e "  [${num}] ${GREEN}PASS${NC} - ${desc}"
    echo -e "       error detected"
  else
    TC_FAIL_COUNT=$((TC_FAIL_COUNT + 1))
    echo -e "  [${num}] ${RED}FAIL${NC} - ${desc}"
    echo -e "       expected error, got: ${output:0:200}"
  fi
}

# ===================== 헬퍼 함수 =====================

# Big number 산술 (bash $((...)) 는 64-bit 오버플로우, bc 사용)
big_add() { echo "$1 + $2" | bc; }
big_sub() { echo "$1 - $2" | bc; }
big_mul() { echo "$1 * $2" | bc; }
big_div() { echo "$1 / $2" | bc; }
big_gte() { [ "$(echo "$1 >= $2" | bc)" = "1" ]; }

# cast call 반환값에서 annotation 제거: "1000000000 [1e9]" → "1000000000"
strip_cast() { awk '{print $1}'; }

# RPC에서 baseFee/gasTip을 조회하고 프로토콜 최소값과 비교하여 가스 파라미터 결정
# gov_burn_proposal.sh 의 resolve_gas_params() 와 동일 로직
resolve_gas_params() {
  local rpc="${1:-$RPC}"

  # 현재 블록의 baseFeePerGas 조회
  local base_fee_hex
  base_fee_hex=$(cast rpc eth_getBlockByNumber "latest" "false" --rpc-url "$rpc" \
    | jq -r '.baseFeePerGas' 2>/dev/null)

  local base_fee
  if [ -n "$base_fee_hex" ] && [ "$base_fee_hex" != "null" ]; then
    base_fee=$(cast to-dec "$base_fee_hex" 2>/dev/null || echo "0")
  else
    base_fee=0
  fi

  # baseFee가 프로토콜 최소값보다 낮으면 최소값 사용
  if [ "$base_fee" -lt "$MIN_BASE_FEE" ] 2>/dev/null; then
    base_fee="$MIN_BASE_FEE"
  fi

  # 현재 네트워크의 maxPriorityFeePerGas 조회
  local gas_tip_hex
  gas_tip_hex=$(cast rpc eth_maxPriorityFeePerGas --rpc-url "$rpc" 2>/dev/null | jq -r '.' 2>/dev/null)

  local gas_tip
  if [ -n "$gas_tip_hex" ] && [ "$gas_tip_hex" != "null" ]; then
    gas_tip=$(cast to-dec "$gas_tip_hex" 2>/dev/null || echo "0")
  else
    gas_tip=0
  fi

  # gasTip이 프로토콜 최소값보다 낮으면 최소값 사용
  if [ "$gas_tip" -lt "$INITIAL_GAS_TIP" ] 2>/dev/null; then
    gas_tip="$INITIAL_GAS_TIP"
  fi

  # feeCap = baseFee + gasTip
  GAS_TIP="$gas_tip"
  GAS_PRICE=$((base_fee + gas_tip))

  echo -e "  ${CYAN}[gas]${NC} baseFee=${base_fee} tip=${GAS_TIP} feeCap=${GAS_PRICE}"
}

# cast send 래퍼 — TX 전송 직전에 최신 가스 파라미터 조회 후 사용
# testnet 에서 baseFee 가 블록마다 변동하므로, 매 TX 마다 재조회 필요
# 가스비를 직접 지정하는 TC (tc-1-3-gas.sh 등)는 cast send 를 직접 사용
cast_send() {
  resolve_gas_params "$RPC" > /dev/null 2>&1
  cast send \
    --gas-price "$GAS_PRICE" \
    --priority-gas-price "$GAS_TIP" \
    "$@"
}

# 이벤트 시그니처 계산 (lazy init)
get_burn_refund_claimed_sig() {
  if [ -z "$BURN_REFUND_CLAIMED_SIG" ]; then
    BURN_REFUND_CLAIMED_SIG=$(cast sig-event 'BurnRefundClaimed(address,uint256)' 2>/dev/null || echo "")
  fi
  echo "$BURN_REFUND_CLAIMED_SIG"
}

get_burn_deposit_refunded_sig() {
  if [ -z "$BURN_DEPOSIT_REFUNDED_SIG" ]; then
    BURN_DEPOSIT_REFUNDED_SIG=$(cast sig-event 'BurnDepositRefunded(uint256,address,uint256)' 2>/dev/null || echo "")
  fi
  echo "$BURN_DEPOSIT_REFUNDED_SIG"
}

# 블록 대기
wait_for_block() {
  local target_block="$1"
  local rpc="${2:-$RPC}"
  echo -e "  ${CYAN}블록 ${target_block} 대기 중...${NC}"
  while true; do
    local current
    current=$(cast block-number --rpc-url "$rpc" 2>/dev/null || echo "0")
    if [ "$current" -ge "$target_block" ] 2>/dev/null; then
      echo -e "  ${GREEN}블록 ${current} 도달${NC}"
      break
    fi
    sleep 2
  done
}

# BurnProof ABI 인코딩 (proposeBurn 의 bytes 파라미터 생성)
# BurnProof: (address from, uint256 amount, uint256 timestamp,
#             string withdrawalId, string referenceId, string memo)
# 사용: PROOF_DATA=$(build_burn_proof "$FROM_ADDR" "$AMOUNT_WEI" "$WID_PREFIX" "$RID_PREFIX" "$MEMO")
# withdrawalId/referenceId 에 timestamp 자동 추가 → 재실행 시 WithdrawalIdInUse 방지
build_burn_proof() {
  local from_addr="$1"
  local amount_wei="$2"
  local timestamp
  timestamp=$(date +%s)
  local ts_suffix="${timestamp}"
  local withdrawal_id="${3:-WD}-${ts_suffix}"
  local reference_id="${4:-REF}-${ts_suffix}"
  local memo="${5:-}"

  cast abi-encode \
    "f(address,uint256,uint256,string,string,string)" \
    "$from_addr" \
    "$amount_wei" \
    "$timestamp" \
    "$withdrawal_id" \
    "$reference_id" \
    "$memo"
}

# receipt에서 로그 추출
get_log_topic() {
  local receipt_json="$1"
  local topic_index="$2"
  local contract_addr="$3"
  echo "$receipt_json" | jq -r ".logs[] | select(.address==\"${contract_addr}\") | .topics[${topic_index}]" 2>/dev/null | head -1
}

# receipt에서 특정 이벤트 시그니처 존재 확인
has_event_in_receipt() {
  local receipt_json="$1"
  local event_sig="$2"
  local found
  found=$(echo "$receipt_json" | jq -r ".logs[] | select(.topics[0]==\"${event_sig}\") | .topics[0]" 2>/dev/null | head -1)
  [ -n "$found" ] && echo "true" || echo "false"
}

# TBD 변수 확인
check_not_tbd() {
  local var_name="$1"
  local var_value="$2"
  if [ "$var_value" = "<TBD>" ] || [ -z "$var_value" ]; then
    tc_blocked "${var_name} 미설정 (환경변수로 주입 필요)"
  fi
}

# ===================== 블록 조회 =====================

get_block_number() {
  local rpc="${1:-$RPC}"
  cast block-number --rpc-url "$rpc" 2>/dev/null || echo "0"
}

get_block_json() {
  local block="${1:-latest}"
  local rpc="${2:-$RPC}"
  cast block "$block" --rpc-url "$rpc" --json 2>/dev/null
}

get_block_timestamp() {
  local block="${1:-latest}"
  local rpc="${2:-$RPC}"
  local ts_hex
  ts_hex=$(cast block "$block" --rpc-url "$rpc" --json 2>/dev/null | jq -r .timestamp)
  if [ -n "$ts_hex" ] && [ "$ts_hex" != "null" ]; then
    printf '%d' "$ts_hex" 2>/dev/null || echo "0"
  else
    echo "0"
  fi
}

print_block_info() {
  local block="${1:-latest}"
  local rpc="${2:-$RPC}"
  local json
  json=$(get_block_json "$block" "$rpc")
  if [ -z "$json" ]; then
    echo -e "  ${RED}블록 ${block} 조회 실패${NC}"
    return 1
  fi
  local num ts tx_count miner base_fee
  num=$(echo "$json" | jq -r .number)
  ts=$(echo "$json" | jq -r .timestamp)
  tx_count=$(echo "$json" | jq '.transactions | length')
  miner=$(echo "$json" | jq -r .miner)
  base_fee=$(echo "$json" | jq -r '.baseFeePerGas // "N/A"')
  echo -e "  블록 ${num}: ts=${ts} txs=${tx_count} miner=${miner:0:18}... baseFee=${base_fee}"
}

# ===================== TX 조회 =====================

get_tx_receipt() {
  local tx_hash="$1"
  local rpc="${2:-$RPC}"
  cast receipt "$tx_hash" --rpc-url "$rpc" --json 2>/dev/null
}

get_tx_status() {
  local tx_hash="$1"
  local rpc="${2:-$RPC}"
  cast receipt "$tx_hash" --rpc-url "$rpc" --json 2>/dev/null | jq -r .status
}

get_tx_gas_used() {
  local tx_hash="$1"
  local rpc="${2:-$RPC}"
  cast receipt "$tx_hash" --rpc-url "$rpc" --json 2>/dev/null | jq -r .gasUsed
}

get_tx_effective_gas_price() {
  local tx_hash="$1"
  local rpc="${2:-$RPC}"
  cast receipt "$tx_hash" --rpc-url "$rpc" --json 2>/dev/null | jq -r .effectiveGasPrice
}

get_tx_block_number() {
  local tx_hash="$1"
  local rpc="${2:-$RPC}"
  cast receipt "$tx_hash" --rpc-url "$rpc" --json 2>/dev/null | jq -r .blockNumber
}

get_tx_logs_count() {
  local tx_hash="$1"
  local rpc="${2:-$RPC}"
  cast receipt "$tx_hash" --rpc-url "$rpc" --json 2>/dev/null | jq '.logs | length'
}

print_tx_info() {
  local tx_hash="$1"
  local rpc="${2:-$RPC}"
  local receipt
  receipt=$(get_tx_receipt "$tx_hash" "$rpc")
  if [ -z "$receipt" ] || [ "$receipt" = "null" ]; then
    echo -e "  ${RED}TX ${tx_hash:0:18}... receipt 없음${NC}"
    return 1
  fi
  local status block gas_used egp logs_n
  status=$(echo "$receipt" | jq -r .status)
  block=$(echo "$receipt" | jq -r .blockNumber)
  gas_used=$(echo "$receipt" | jq -r .gasUsed)
  egp=$(echo "$receipt" | jq -r .effectiveGasPrice)
  logs_n=$(echo "$receipt" | jq '.logs | length')
  local label
  if [ "$status" = "0x1" ]; then label="${GREEN}SUCCESS${NC}"; else label="${RED}REVERTED${NC}"; fi
  echo -e "  TX ${tx_hash:0:18}... ${label}  block=${block} gasUsed=${gas_used} effectiveGasPrice=${egp} logs=${logs_n}"
}

# ===================== GovMinter (GMI, 0x1003) 조회 =====================
# - cast call 반환값을 xargs로 trim
# - || echo "" 로 set -e 환경에서 cast call 실패 시 스크립트 중단 방지
# - isMember, getQuorum 등 GovBase 함수는 memberVersion 파라미터 필요

# GovMinter — memberVersion (현재 멤버 버전)
gmi_member_version() {
  local rpc="${1:-$RPC}"
  cast call "$GMI" 'memberVersion()(uint256)' --rpc-url "$rpc" 2>/dev/null | strip_cast || echo "1"
}

# GovMinter — isMember(address, memberVersion)
gmi_is_member() {
  local addr="$1"; local rpc="${2:-$RPC}"
  local ver
  ver=$(gmi_member_version "$rpc")
  cast call "$GMI" 'isMember(address,uint256)(bool)' "$addr" "$ver" --rpc-url "$rpc" 2>/dev/null | xargs || echo ""
}

gmi_burn_balance() {
  local addr="$1"; local rpc="${2:-$RPC}"
  cast call "$GMI" 'burnBalance(address)(uint256)' "$addr" --rpc-url "$rpc" 2>/dev/null | strip_cast || echo "0"
}

gmi_refundable_balance() {
  local addr="$1"; local rpc="${2:-$RPC}"
  cast call "$GMI" 'refundableBalance(address)(uint256)' "$addr" --rpc-url "$rpc" 2>/dev/null | strip_cast || echo "0"
}

gmi_is_paused() {
  local rpc="${1:-$RPC}"
  cast call "$GMI" 'emergencyPaused()(bool)' --rpc-url "$rpc" 2>/dev/null | xargs || echo ""
}

gmi_reserved_mint() {
  local rpc="${1:-$RPC}"
  cast call "$GMI" 'reservedMintAmount()(uint256)' --rpc-url "$rpc" 2>/dev/null | strip_cast || echo "0"
}

gmi_is_voting() {
  local proposal_id="$1"; local rpc="${2:-$RPC}"
  cast call "$GMI" 'isProposalInVoting(uint256)(bool)' "$proposal_id" --rpc-url "$rpc" 2>/dev/null | xargs || echo ""
}

gmi_is_executable() {
  local proposal_id="$1"; local rpc="${2:-$RPC}"
  cast call "$GMI" 'isProposalExecutable(uint256)(bool)' "$proposal_id" --rpc-url "$rpc" 2>/dev/null | xargs || echo ""
}

gmi_has_approved() {
  local addr="$1"; local proposal_id="$2"; local rpc="${3:-$RPC}"
  cast call "$GMI" 'hasApproved(address,uint256)(bool)' "$addr" "$proposal_id" --rpc-url "$rpc" 2>/dev/null | xargs || echo ""
}

gmi_can_create_proposal() {
  local addr="$1"; local rpc="${2:-$RPC}"
  cast call "$GMI" 'canCreateProposal(address)(bool)' "$addr" --rpc-url "$rpc" 2>/dev/null | xargs || echo ""
}

gmi_used_proof_hash() {
  local hash="$1"; local rpc="${2:-$RPC}"
  cast call "$GMI" 'usedProofHashes(bytes32)(bool)' "$hash" --rpc-url "$rpc" 2>/dev/null | xargs || echo ""
}

print_gmi_member_status() {
  local addr="$1"; local label="${2:-}"; local rpc="${3:-$RPC}"
  local member burn refund can_propose
  member=$(gmi_is_member "$addr" "$rpc")
  burn=$(gmi_burn_balance "$addr" "$rpc")
  refund=$(gmi_refundable_balance "$addr" "$rpc")
  can_propose=$(gmi_can_create_proposal "$addr" "$rpc")
  echo -e "  ${label}${addr:0:18}... isMember=${member} burnBal=${burn} refundBal=${refund} canPropose=${can_propose}"
}

print_gmi_proposal_status() {
  local proposal_id="$1"; local rpc="${2:-$RPC}"
  local voting executable
  voting=$(gmi_is_voting "$proposal_id" "$rpc")
  executable=$(gmi_is_executable "$proposal_id" "$rpc")
  echo -e "  Proposal #${proposal_id}: inVoting=${voting} executable=${executable}"
}

# ===================== GovValidator (GV, 0x1001) 조회 =====================

gv_quorum() {
  local rpc="${1:-$RPC}"
  cast call "$GV" 'quorum()(uint256)' --rpc-url "$rpc" 2>/dev/null | strip_cast || echo "0"
}

gv_gas_tip() {
  local rpc="${1:-$RPC}"
  cast call "$GV" 'gasTip()(uint256)' --rpc-url "$rpc" 2>/dev/null | strip_cast || echo "0"
}

gv_is_validator() {
  local addr="$1"; local rpc="${2:-$RPC}"
  cast call "$GV" 'isValidator(address)(bool)' "$addr" --rpc-url "$rpc" 2>/dev/null | xargs || echo ""
}

gv_validator_count() {
  local rpc="${1:-$RPC}"
  cast call "$GV" 'validatorCount()(uint256)' --rpc-url "$rpc" 2>/dev/null | strip_cast || echo "0"
}

# ===================== GovCouncil (GC, 0x1004) 조회 =====================

gc_is_authorized() {
  local addr="$1"; local rpc="${2:-$RPC}"
  cast call "$GC" 'isAuthorizedAccount(address)(bool)' "$addr" --rpc-url "$rpc" 2>/dev/null | xargs || echo ""
}

gc_is_blacklisted() {
  local addr="$1"; local rpc="${2:-$RPC}"
  cast call "$GC" 'isBlacklisted(address)(bool)' "$addr" --rpc-url "$rpc" 2>/dev/null | xargs || echo ""
}

gc_blacklist_count() {
  local rpc="${1:-$RPC}"
  cast call "$GC" 'getBlacklistCount()(uint256)' --rpc-url "$rpc" 2>/dev/null | strip_cast || echo "0"
}

gc_authorized_count() {
  local rpc="${1:-$RPC}"
  cast call "$GC" 'getAuthorizedAccountCount()(uint256)' --rpc-url "$rpc" 2>/dev/null | strip_cast || echo "0"
}

# ===================== GovMasterMinter (GMM, 0x1002) 조회 =====================

gmm_is_minter() {
  local addr="$1"; local rpc="${2:-$RPC}"
  cast call "$GMM" 'getIsMinter(address)(bool)' "$addr" --rpc-url "$rpc" 2>/dev/null | xargs || echo ""
}

gmm_minter_allowance() {
  local addr="$1"; local rpc="${2:-$RPC}"
  cast call "$GMM" 'getMinterAllowance(address)(uint256)' "$addr" --rpc-url "$rpc" 2>/dev/null | strip_cast || echo "0"
}

gmm_max_minter_allowance() {
  local rpc="${1:-$RPC}"
  cast call "$GMM" 'maxMinterAllowance()(uint256)' --rpc-url "$rpc" 2>/dev/null | strip_cast || echo "0"
}

# ===================== NativeCoinAdapter (NCA, 0x1000) 조회 =====================

nca_balance_of() {
  local addr="$1"; local rpc="${2:-$RPC}"
  cast call "$NCA" 'balanceOf(address)(uint256)' "$addr" --rpc-url "$rpc" 2>/dev/null | strip_cast || echo "0"
}

nca_total_supply() {
  local rpc="${1:-$RPC}"
  cast call "$NCA" 'totalSupply()(uint256)' --rpc-url "$rpc" 2>/dev/null | strip_cast || echo "0"
}

nca_name() {
  local rpc="${1:-$RPC}"
  cast call "$NCA" 'name()(string)' --rpc-url "$rpc" 2>/dev/null | xargs || echo ""
}

nca_symbol() {
  local rpc="${1:-$RPC}"
  cast call "$NCA" 'symbol()(string)' --rpc-url "$rpc" 2>/dev/null | xargs || echo ""
}

# ===================== 초기화 =====================

# RPC 연결 후 가스 파라미터 자동 결정
GAS_TIP="$INITIAL_GAS_TIP"
GAS_PRICE="$MIN_FEE"
resolve_gas_params "$RPC"

echo -e "${CYAN}[common.sh] 하드포크 테스트 공통 환경 로드 완료${NC}"
echo -e "  RPC: ${RPC}"
echo -e "  BP1: ${BP1_RPC}  |  EN1: ${EN1_RPC}"
echo -e "  ChainID: ${CHAIN_ID}  |  BohoBlock: ${BOHO_BLOCK}  |  Quorum: ${QUORUM}"
echo -e "  GMI: ${GMI}"
echo -e "  MEMBER_A: ${MEMBER_A}"
echo -e "  결과 디렉토리: ${RESULT_DIR}"
