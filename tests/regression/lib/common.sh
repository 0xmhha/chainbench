#!/usr/bin/env bash
# tests/regression/lib/common.sh — go-stablenet v2 회귀 테스트 공통 헬퍼
#
# 사용법:
#   source "$(dirname "$0")/../lib/common.sh"
#
# 제공 기능:
#   - 환경 프로파일(tests/env) 로드 + 환경-중립 접근자(accessors.sh) 배선
#   - 공용 상수(constants.sh)
#   - eth_call / raw tx / receipt 조회 헬퍼 (cast/jq 기반)
#   - Governance 프로포절 생성/투표/실행 헬퍼
#   - Keccak-256 해시, ABI selector 계산 (cast)
#   - Log/event 검색 헬퍼 (jq)
#   - cast(Foundry) / jq 기반 서명·인코딩·파싱 유틸 (python 의존 제거)

[[ -n "${_CB_REGRESSION_COMMON_LOADED:-}" ]] && return 0
readonly _CB_REGRESSION_COMMON_LOADED=1

CHAINBENCH_DIR="${CHAINBENCH_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)}"
source "${CHAINBENCH_DIR}/tests/lib/rpc.sh"
source "${CHAINBENCH_DIR}/tests/lib/assert.sh"
source "${CHAINBENCH_DIR}/tests/lib/wait.sh"

# 환경 프로파일(토폴로지/배선) + 접근자 + 공용 상수
source "${CHAINBENCH_DIR}/tests/env/profile.sh"
source "${CHAINBENCH_DIR}/tests/regression/lib/accessors.sh"
source "${CHAINBENCH_DIR}/tests/regression/lib/constants.sh"
source "${CHAINBENCH_DIR}/tests/regression/lib/sign_backends.sh"
# 노드제어 백엔드(local|closednet). 없으면 ensure_nodes_running이 no-op.
source "${CHAINBENCH_DIR}/tests/regression/lib/node_ctrl/${CB_NODECTRL_BACKEND}.sh" 2>/dev/null || true

# ============================================================================
# 시스템 컨트랙트 / 상수 (환경 불변)
# ============================================================================

readonly NATIVE_COIN_ADAPTER="0x0000000000000000000000000000000000001000"
readonly GOV_VALIDATOR="0x0000000000000000000000000000000000001001"
readonly GOV_MASTER_MINTER="0x0000000000000000000000000000000000001002"
readonly GOV_MINTER="0x0000000000000000000000000000000000001003"
readonly GOV_COUNCIL="0x0000000000000000000000000000000000001004"
readonly ACCOUNT_MANAGER="0x0000000000000000000000000000000000B00003"
readonly NATIVE_COIN_MANAGER="0x0000000000000000000000000000000000B00002"
readonly BLS_POP="0x0000000000000000000000000000000000B00001"

readonly ZERO_ADDRESS="0x0000000000000000000000000000000000000000"
readonly MIN_BASE_FEE_WEI="20000000000000"  # 20 Gwei

# 이벤트 시그니처 해시
readonly TRANSFER_EVENT_SIG="0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
readonly AUTHORIZED_TX_EXECUTED_SIG="0x40e728a89c7f5b192cf1c1b747fb64d51d81c7a2b3ed4607b94d3a1e6a3e0373"
# GovBase ProposalCreated 이벤트 sig. proposeBurn 등 payable 함수는 Transfer 이벤트가
# ProposalCreated보다 먼저 emit되므로 sig 기반 필터링 필수(아래 extract_proposal_id_from_receipt).
readonly PROPOSAL_CREATED_SIG="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"

# ============================================================================
# 전환용 backward-compat shim (W9에서 본문을 접근자로 바꾸면 제거)
# 본문이 아직 TEST_ACC_*/VALIDATOR_* 리터럴을 참조하므로 프로파일 값으로 채운다.
# ★ 신규 코드는 acct_addr()/validator_addr()/tx_send_as()를 쓸 것.
# ============================================================================

readonly TEST_ACC_A_ADDR="$(acct_addr 1)"
readonly TEST_ACC_B_ADDR="$(acct_addr 2)"
readonly TEST_ACC_C_ADDR="$(acct_addr 3)"
readonly TEST_ACC_D_ADDR="$(acct_addr 4)"
readonly TEST_ACC_E_ADDR="$(acct_addr 5)"

# 공개 테스트 private key(로컬, 비밀 아님). closednet(방식 B)에서는 CB_ACCT_PK 미정의 → 빈 값.
# (closednet 본문은 이 변수를 쓰지 않고 tx_send_as로 노드측 서명한다.)
readonly TEST_ACC_A_PK="${CB_ACCT_PK[1]:-}"
readonly TEST_ACC_B_PK="${CB_ACCT_PK[2]:-}"
readonly TEST_ACC_C_PK="${CB_ACCT_PK[3]:-}"
readonly TEST_ACC_D_PK="${CB_ACCT_PK[4]:-}"
readonly TEST_ACC_E_PK="${CB_ACCT_PK[5]:-}"

readonly VALIDATOR_1_ADDR="$(validator_addr 1)"
readonly VALIDATOR_2_ADDR="$(validator_addr 2)"
readonly VALIDATOR_3_ADDR="$(validator_addr 3)"
readonly VALIDATOR_4_ADDR="$(validator_addr 4)"
readonly VALIDATOR_KEYSTORE_PASSWORD="${CB_VALIDATOR_KEYSTORE_PASSWORD-}"

# ============================================================================
# 기본 유틸리티
# ============================================================================

# hex_to_dec <hex_with_0x>  "0x10" → "16"
# cast to-dec로 2^63 초과(예: 10^27 wei) 큰 정수도 처리.
hex_to_dec() {
  local hex="${1#0x}"
  [[ -z "$hex" || "$hex" == "null" ]] && { echo "0"; return; }
  cast to-dec "0x${hex}"
}

# to_checksum <address> — EIP-55 체크섬 주소로 정규화.
to_checksum() {
  local addr="$1"
  [[ -z "$addr" || "$addr" == "null" ]] && { echo ""; return; }
  cast to-checksum "$addr"
}

# get_header_gas_tip <target>
# latest 블록 WBFTExtra.GasTip을 decimal string으로 반환.
get_header_gas_tip() {
  local target="${1:-1}"
  local latest_hex
  latest_hex=$(rpc "$target" "eth_blockNumber" "[]" | json_get - result)
  local resp gt
  resp=$(rpc "$target" "istanbul_getWbftExtraInfo" "[\"${latest_hex}\"]")
  gt=$(printf '%s' "$resp" | jq -r '.result.gasTip // "0"')
  if [[ "$gt" == 0x* ]]; then
    cast to-dec "$gt"
  else
    echo "${gt:-0}"
  fi
}

# get_wbft_extra_json <target> [block_hex]
get_wbft_extra_json() {
  local target="${1:-1}" block_hex="${2:-}"
  if [[ -z "$block_hex" ]]; then
    block_hex=$(rpc "$target" "eth_blockNumber" "[]" | json_get - result)
  fi
  rpc "$target" "istanbul_getWbftExtraInfo" "[\"${block_hex}\"]"
}

# dec_to_hex <decimal>  "255" → "0xff"
dec_to_hex() {
  printf '0x%x\n' "$1"
}

# json_get <json_or_dash> <key_path>  — jq로 필드 추출. 첫 인자 "-"면 stdin.
# dot-path "result.0" → ".result[0]" 로 변환.
json_get() {
  local json="$1" path="$2"
  if [[ "$json" == "-" ]]; then
    json="$(cat)"
  fi
  if [[ -z "$json" ]]; then
    return 0
  fi
  local jq_path=".$path"
  jq_path=$(printf '%s' "$jq_path" | sed 's/\.\([0-9][0-9]*\)/[\1]/g')
  printf '%s' "$json" | jq -r "
    try (${jq_path}) catch null
    | if . == null then \"\"
      elif type == \"object\" or type == \"array\" then tojson
      else tostring
      end
  " 2>/dev/null || true
}

# ============================================================================
# 노드/계정 관리
# ============================================================================

# get_node_url <target> — 노드/remote alias의 HTTP RPC URL
get_node_url() {
  local target="$1"
  if [[ "$target" == @* ]]; then
    _cb_rpc_ensure_remote_state
    _cb_rpc_resolve_remote_url "${target#@}"
  else
    local port
    port=$(pids_get_field "$target" "http_port") || return 1
    echo "http://127.0.0.1:${port}"
  fi
}

# get_node_ws_url <target> — 노드/remote alias의 WebSocket RPC URL
get_node_ws_url() {
  local target="$1"
  if [[ "$target" == @* ]]; then
    _cb_rpc_ensure_remote_state
    _cb_rpc_resolve_remote_ws_url "${target#@}"
  else
    local ws_port
    ws_port=$(pids_get_field "$target" "ws_port") || return 1
    echo "ws://127.0.0.1:${ws_port}"
  fi
}

# unlock_validator <target>
# 해당 노드의 coinbase(validator) 계정을 keystore 비밀번호로 잠금 해제.
unlock_validator() {
  local node="${1:-1}"
  local addr
  addr=$(get_coinbase "$node")
  [[ -z "$addr" || "$addr" == "null" ]] && return 1
  unlock_account "$node" "$addr" "$VALIDATOR_KEYSTORE_PASSWORD" 3600 || return 1
  echo "$addr"
}

# get_nonce <target> <address>
get_nonce() {
  local target="${1:-1}" addr="$2"
  local response result
  response=$(rpc "$target" "eth_getTransactionCount" "[\"${addr}\", \"pending\"]") || return 1
  result=$(json_get "$response" "result")
  hex_to_dec "$result"
}

# get_base_fee <target> — 현재 블록 baseFee를 decimal로 반환
get_base_fee() {
  local target="${1:-1}"
  local response result
  response=$(rpc "$target" "eth_getBlockByNumber" "[\"latest\", false]") || return 1
  result=$(json_get "$response" "result.baseFeePerGas")
  hex_to_dec "$result"
}

# get_priority_fee <target> — eth_maxPriorityFeePerGas를 decimal로 반환
get_priority_fee() {
  local target="${1:-1}"
  local response result
  response=$(rpc "$target" "eth_maxPriorityFeePerGas" "[]") || return 1
  result=$(json_get "$response" "result")
  hex_to_dec "$result"
}

# wait_for_block <target> <block_number_decimal> [timeout_secs]
wait_for_block() {
  local target="${1:?wait_for_block: target required}"
  local target_block="${2:?wait_for_block: block number required}"
  local timeout="${3:-120}"

  local elapsed=0
  local current
  while (( elapsed < timeout )); do
    current=$(block_number "$target" 2>/dev/null || echo "0")
    if (( current >= target_block )); then
      echo >&2 "  Block $current reached (target: $target_block)"
      return 0
    fi
    if (( elapsed % 5 == 0 )); then
      echo >&2 "  Waiting for block $target_block (current: $current, ${elapsed}s/${timeout}s)"
    fi
    sleep 1
    elapsed=$(( elapsed + 1 ))
  done

  echo >&2 "  TIMEOUT waiting for block $target_block (current: $current after ${timeout}s)"
  return 1
}

# ============================================================================
# Raw Transaction (cast 클라이언트 서명)
# ============================================================================

# send_raw_tx <target> <private_key> <to> <value_wei> [data] [gas_limit] [tx_type] [tip_cap] [fee_cap]
# tx_type: legacy, dynamic (default), accesslist
# value/gas_limit/tip_cap/fee_cap: decimal string. data: hex(0x 유무 무관, 기본 "").
# tip_cap 생략 시 노드 eth_maxPriorityFeePerGas, fee_cap 생략 시 base_fee+tip_cap.
# 반환: txHash (0x...)
#
# NOTE: 이 함수는 client_cast 서명 경로다(로컬/임시계정용). 폐쇄망 본문은 직접 호출하지 말고
#       tx_send_as 를 통해 활성 서명 백엔드(node_keystore)로 위임할 것.
send_raw_tx() {
  local target="$1" pk="$2" to="$3" value="$4"
  local data="${5:-}" gas_limit="${6:-21000}" tx_type="${7:-dynamic}"
  local tip_cap="${8:-}" fee_cap="${9:-}"

  local url
  url=$(get_node_url "$target") || return 1

  local cast_args=(--rpc-url "$url" --private-key "$pk" --gas-limit "$gas_limit" --async)

  # baseFee/priorityFee를 노드에서 조회해 명시. cast 기본값이 네트워크 최소 priorityFee보다
  # 낮아 "transaction underpriced"가 날 수 있어 직접 계산한다.
  local base_fee priority_fee
  base_fee=$(get_base_fee "$target")
  priority_fee=$(get_priority_fee "$target")
  [[ -n "$tip_cap" ]] && priority_fee="$tip_cap"
  [[ -n "$fee_cap" ]] || fee_cap="$(( base_fee + priority_fee ))"

  case "$tx_type" in
    legacy)
      cast_args+=(--legacy --gas-price "$fee_cap")
      ;;
    accesslist)
      cast_args+=(--access-list --priority-gas-price "$priority_fee" --gas-price "$fee_cap")
      ;;
    dynamic|*)
      # EIP-1559: --gas-price = maxFeePerGas, --priority-gas-price = maxPriorityFeePerGas
      cast_args+=(--priority-gas-price "$priority_fee" --gas-price "$fee_cap")
      ;;
  esac

  local result
  if [[ -n "$data" ]]; then
    [[ "$data" != 0x* ]] && data="0x$data"
    if [[ -z "$to" ]]; then
      result=$(cast send "${cast_args[@]}" --value "$value" --create "$data" 2>&1) || {
        printf 'TX_ERROR:%s\n' "$result" >&2; return 1; }
    else
      result=$(cast send "${cast_args[@]}" --value "$value" "$to" "$data" 2>&1) || {
        printf 'TX_ERROR:%s\n' "$result" >&2; return 1; }
    fi
  else
    result=$(cast send "${cast_args[@]}" --value "$value" "$to" 2>&1) || {
      printf 'TX_ERROR:%s\n' "$result" >&2; return 1; }
  fi
  local hash
  hash=$(printf '%s\n' "$result" | grep -oE '0x[0-9a-fA-F]{64}' | head -1)
  printf '[INFO]  txHash=%s\n' "${hash:-$result}" >&2
  printf '%s\n' "${hash:-$result}"
}

# ============================================================================
# Receipt / Log 조회
# ============================================================================

# get_receipt <target> <tx_hash> — 전체 receipt JSON 객체
get_receipt() {
  local target="${1:-1}" tx_hash="$2"
  local response
  response=$(rpc "$target" "eth_getTransactionReceipt" "[\"${tx_hash}\"]") || return 1
  json_get "$response" "result"
}

# get_receipt_field <target> <tx_hash> <field>
get_receipt_field() {
  local target="$1" tx_hash="$2" field="$3"
  local receipt
  receipt=$(get_receipt "$target" "$tx_hash") || return 1
  printf '%s' "$receipt" | jq -r ".${field} // empty"
}

# wait_tx_receipt_full <target> <tx_hash> [timeout]
wait_tx_receipt_full() {
  local target="${1:-1}" tx_hash="$2" timeout="${3:-30}"
  local i receipt
  for i in $(seq 1 "$timeout"); do
    receipt=$(get_receipt "$target" "$tx_hash")
    if [[ -n "$receipt" && "$receipt" != "null" ]]; then
      printf '%s' "$receipt"
      return 0
    fi
    sleep 1
  done
  return 1
}

# find_log_by_topic <receipt_json> <address> <topic0>
find_log_by_topic() {
  local receipt="$1" address="$2" topic0="$3"
  local addr_lower topic_lower
  addr_lower=$(printf '%s' "$address" | tr '[:upper:]' '[:lower:]')
  topic_lower=$(printf '%s' "$topic0" | tr '[:upper:]' '[:lower:]')
  printf '%s' "$receipt" | jq -r --arg addr "$addr_lower" --arg t0 "$topic_lower" \
    'first(.logs[] | select((.address | ascii_downcase) == $addr and (.topics[0] | ascii_downcase) == $t0)) | tojson' \
    2>/dev/null || true
}

# count_logs_by_address <receipt_json> <address>
count_logs_by_address() {
  local receipt="$1" address="$2"
  local addr_lower
  addr_lower=$(printf '%s' "$address" | tr '[:upper:]' '[:lower:]')
  printf '%s' "$receipt" | jq -r --arg addr "$addr_lower" \
    '[.logs[] | select((.address | ascii_downcase) == $addr)] | length' \
    2>/dev/null || echo "0"
}

# ============================================================================
# eth_call (view 함수 호출)
# ============================================================================

# eth_call_raw <target> <to> <data_hex> — 원시 eth_call 결과(0x...)
eth_call_raw() {
  local target="${1:-1}" to="$2" data="$3"
  [[ "$data" != 0x* ]] && data="0x$data"
  local response result
  response=$(rpc "$target" "eth_call" \
    "[{\"to\":\"${to}\",\"data\":\"${data}\"}, \"latest\"]") || return 1
  result=$(json_get "$response" "result")
  printf '%s' "$result"
}

# selector <function_signature>  "transfer(address,uint256)" → "0xa9059cbb"
selector() {
  local sig="$1"
  cast sig "$sig"
}

# pad_address <addr> — 20-byte 주소 → 32-byte padded
pad_address() {
  local addr="${1#0x}"
  printf '0x%064s\n' "$addr" | tr ' ' '0'
}

# pad_uint256 <decimal_or_hex>
pad_uint256() {
  local val="$1" hex
  if [[ "$val" == 0x* ]]; then
    hex="${val#0x}"
  else
    hex=$(cast to-hex "$val" 2>/dev/null | sed 's/^0x//')
  fi
  printf '0x%064s\n' "$hex" | tr ' ' '0'
}

# ============================================================================
# Governance 헬퍼 (proposal lifecycle)
# ============================================================================

# gov_call <target> <contract> <data> <from_addr> [gas]
# governance tx 전송 (validator coinbase unlock 전제). EIP-1559(type 0x2)로 명시 전송.
gov_call() {
  local target="${1:-1}" contract="$2" data="$3" from="$4" gas="${5:-800000}"
  local tip_hex base_fee_hex base_fee_dec tip_dec max_fee_hex response
  tip_hex=$(rpc "$target" "eth_maxPriorityFeePerGas" "[]" | json_get - "result")
  base_fee_hex=$(rpc "$target" "eth_getBlockByNumber" '["latest",false]' | \
    jq -r '.result.baseFeePerGas // "0x0"')
  base_fee_dec=$(hex_to_dec "$base_fee_hex")
  tip_dec=$(hex_to_dec "$tip_hex")
  max_fee_hex=$(dec_to_hex "$(( base_fee_dec + tip_dec + 1 ))")
  response=$(rpc "$target" "eth_sendTransaction" \
    "[{\"from\":\"${from}\",\"to\":\"${contract}\",\"data\":\"${data}\",\"gas\":\"$(dec_to_hex "$gas")\",\"type\":\"0x2\",\"maxFeePerGas\":\"${max_fee_hex}\",\"maxPriorityFeePerGas\":\"${tip_hex}\"}]") || return 1
  printf '[GOV-DEBUG] gov_call raw response: %s\n' "$response" >&2
  json_get "$response" "result"
}

# unlock_all_validators — 모든 BP validator의 coinbase를 unlock.
unlock_all_validators() {
  local i
  for (( i=1; i<=CB_NODE_COUNT; i++ )); do
    local tgt addr
    tgt=$(node "$i")
    addr=$(unlock_validator "$tgt" 2>&1) || \
      printf '[WARN]  unlock_validator failed for %s: %s\n' "$tgt" "$addr" >&2
  done
}

# addr_to_node <validator_address>
# Validator 주소를 그 검증자의 홈 노드 타깃으로 매핑. 노드별 keystore isolation 때문에
# eth_sendTransaction은 반드시 그 validator의 홈 노드에서 호출해야 한다.
addr_to_node() {
  local addr i vaddr
  addr=$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]')
  for (( i=1; i<=CB_NODE_COUNT; i++ )); do
    vaddr=$(printf '%s' "${CB_VALIDATOR_ADDR[$i]:-}" | tr '[:upper:]' '[:lower:]')
    if [[ -n "$vaddr" && "$addr" == "$vaddr" ]]; then
      node "$i"; return 0
    fi
  done
  node 1   # fallback (non-validator)
}

# extract_proposal_id_from_receipt <target> <tx_hash>
# ProposalCreated 이벤트 topics[1]에서 proposalId(decimal) 추출.
# PROPOSAL_CREATED_SIG로 정확히 필터링한다(단순 첫 log 선택 금지):
#   - proposeBurn(payable): Transfer가 ProposalCreated보다 먼저 emit → 첫 log 오추출.
extract_proposal_id_from_receipt() {
  local target="${1:-1}" tx_hash="$2"
  local receipt
  receipt=$(wait_tx_receipt_full "$target" "$tx_hash" "$TIMEOUT_TX_RECEIPT") || return 1
  local receipt_status log_count
  receipt_status=$(printf '%s' "$receipt" | jq -r '.status // "?"')
  log_count=$(printf '%s' "$receipt" | jq '.logs | length' 2>/dev/null || echo "0")
  printf '[GOV-DEBUG] extract_proposal_id: status=%s logs=%s\n' "$receipt_status" "$log_count" >&2

  local topic1
  topic1=$(printf '%s' "$receipt" | jq -r --arg sig "$PROPOSAL_CREATED_SIG" \
    'first(.logs[] | select(.topics[0] == $sig and (.topics | length >= 2))) | .topics[1]' 2>/dev/null)

  if [[ -z "$topic1" || "$topic1" == "null" ]]; then
    printf '[GOV-DEBUG] extract_proposal_id: ProposalCreated (sig=%s) NOT FOUND in %s logs\n' \
      "$PROPOSAL_CREATED_SIG" "$log_count" >&2
    echo ""; return 1
  fi
  cast to-dec "$topic1"
}

# gov_propose <target> <contract> <propose_data> <from_addr> → proposalId (decimal)
# <target>은 호환용; 실제 tx는 from_addr의 홈 노드에서 전송.
gov_propose() {
  local _target_unused="$1" contract="$2" data="$3" from="$4"
  local node
  node=$(addr_to_node "$from")
  local tx_hash
  tx_hash=$(gov_call "$node" "$contract" "$data" "$from" 800000) || {
    printf '[GOV-DEBUG] gov_propose: gov_call FAILED (node=%s from=%s contract=%s)\n' "$node" "$from" "$contract" >&2
    return 1
  }
  if [[ -z "$tx_hash" || "$tx_hash" == "null" ]]; then
    printf '[GOV-DEBUG] gov_propose: empty tx_hash (node=%s from=%s)\n' "$node" "$from" >&2
    return 1
  fi
  printf '[GOV-DEBUG] gov_propose: tx_hash=%s\n' "$tx_hash" >&2
  extract_proposal_id_from_receipt "$node" "$tx_hash"
}

# gov_approve <target> <contract> <proposal_id> <from_addr> [gas]
# approveProposal(uint256). 반드시 from의 홈 노드에서 전송. auto-execute가 무거운
# 컨트랙트는 더 높은 gas 지정(예: 2000000).
gov_approve() {
  local _target_unused="${1:-1}" contract="$2" proposal_id="$3" from="$4" gas="${5:-500000}"
  local sel padded node
  sel=$(selector "approveProposal(uint256)")
  padded=$(pad_uint256 "$proposal_id" | sed 's/^0x//')
  local data="${sel}${padded}"
  node=$(addr_to_node "$from")
  gov_call "$node" "$contract" "$data" "$from" "$gas"
}

# gov_execute <target> <contract> <proposal_id> <from_addr>
gov_execute() {
  local _target_unused="${1:-1}" contract="$2" proposal_id="$3" from="$4"
  local sel padded node
  sel=$(selector "executeProposal(uint256)")
  padded=$(pad_uint256 "$proposal_id" | sed 's/^0x//')
  local data="${sel}${padded}"
  node=$(addr_to_node "$from")
  gov_call "$node" "$contract" "$data" "$from" 1500000
}

# gov_proposal_status <target> <contract> <proposal_id>
# None=0 Voting=1 Approved=2 Executed=3 Cancelled=4 Expired=5 Failed=6 Rejected=7
# proposals(uint256) 자동 getter의 10-field tuple을 decode, status=index 9(uint8).
gov_proposal_status() {
  local target="${1:-1}" contract="$2" proposal_id="$3"
  local sel padded call_result
  sel=$(selector "proposals(uint256)")
  padded=$(pad_uint256 "$proposal_id" | sed 's/^0x//')
  call_result=$(eth_call_raw "$target" "$contract" "${sel}${padded}")
  [[ -z "$call_result" || "$call_result" == "0x" ]] && { echo "0"; return; }
  cast abi-decode \
    "proposals()(bytes32,uint256,uint256,uint256,uint256,address,uint32,uint32,uint32,uint8)" \
    "$call_result" 2>/dev/null | tail -1 || echo "0"
}

# gov_full_flow <contract> <propose_data> <proposer_addr> <approver1> [approver2] ...
# 전체 lifecycle: propose → approve(N) → execute → 실행 receipt 반환.
gov_full_flow() {
  local contract="$1" propose_data="$2" proposer="$3"
  shift 3
  local approvers=("$@")

  # 1) propose
  local proposal_id
  proposal_id=$(gov_propose "$(node 1)" "$contract" "$propose_data" "$proposer") || return 1
  [[ -z "$proposal_id" ]] && return 1
  local status_after_propose
  status_after_propose=$(gov_proposal_status "$(node 1)" "$contract" "$proposal_id")
  printf '[GOV]  proposalId=%s (status after propose=%s)\n' "$proposal_id" "$status_after_propose" >&2

  # 2) approve (각 approver) — receipt status 검증. quorum 도달 시 approve tx 안에서 auto-execute.
  local last_approve_receipt=""
  for ap in "${approvers[@]}"; do
    local tx
    tx=$(gov_approve "$(node 1)" "$contract" "$proposal_id" "$ap") || true
    if [[ -z "$tx" || "$tx" == "null" ]]; then
      printf '[GOV]  approve by %s: NO TX HASH (submission failed)\n' "$ap" >&2
      return 1
    fi
    local receipt_json approve_status
    receipt_json=$(wait_tx_receipt_full "$(node 1)" "$tx" "$TIMEOUT_TX_APPROVE" 2>/dev/null || echo "")
    approve_status=$(printf '%s' "$receipt_json" | jq -r '.status // empty' 2>/dev/null)
    local prop_status
    prop_status=$(gov_proposal_status "$(node 1)" "$contract" "$proposal_id")
    printf '[GOV]  approved by %s (tx=%s, receipt.status=%s, proposal.status=%s)\n' \
      "$ap" "$tx" "${approve_status:-none}" "$prop_status" >&2
    if [[ "$approve_status" != "0x1" ]]; then
      printf '[GOV]  ERROR: approve tx reverted/missing — aborting before execute\n' >&2
      return 1
    fi
    last_approve_receipt="$receipt_json"
    if [[ "$prop_status" == "3" ]]; then
      printf '[GOV]  auto-executed during approve by %s (tx=%s)\n' "$ap" "$tx" >&2
      printf '%s' "$last_approve_receipt"
      return 0
    fi
  done

  # 3) execute (수동 execute 필요한 경우)
  local exec_tx
  exec_tx=$(gov_execute "$(node 1)" "$contract" "$proposal_id" "$proposer") || return 1
  [[ -z "$exec_tx" || "$exec_tx" == "null" ]] && return 1
  printf '[GOV]  executeTx=%s\n' "$exec_tx" >&2

  # 4) receipt 반환
  wait_tx_receipt_full "$(node 1)" "$exec_tx" "$TIMEOUT_TX_RECEIPT"
}

# ============================================================================
# GovMinter v2 (Boho) helpers
# ============================================================================

# Function selectors (computed lazily on first use)
PROPOSE_BURN_SIG=""
CLAIM_BURN_REFUND_SIG=""
CANCEL_PROPOSAL_SIG=""
DISAPPROVE_PROPOSAL_SIG=""
BURN_BALANCE_SIG=""
REFUNDABLE_BALANCE_SIG=""

_init_govminter_v2_sigs() {
  [[ -n "$PROPOSE_BURN_SIG" ]] && return 0
  PROPOSE_BURN_SIG=$(selector "proposeBurn(bytes)")
  CLAIM_BURN_REFUND_SIG=$(selector "claimBurnRefund()")
  CANCEL_PROPOSAL_SIG=$(selector "cancelProposal(uint256)")
  DISAPPROVE_PROPOSAL_SIG=$(selector "disapproveProposal(uint256)")
  BURN_BALANCE_SIG=$(selector "burnBalance(address)")
  REFUNDABLE_BALANCE_SIG=$(selector "refundableBalance(address)")
}

# Event signature constants (pre-computed keccak256)
readonly BURN_REFUND_CLAIMED_SIG="0x9543fa265d2616af3e7021d8b5a7d1271eb7bba960908675ce3bddaf60c1af24"
readonly BURN_DEPOSIT_REFUNDED_SIG="0x334fe3eaa506b12e7e46ba469c310822737a959f2553b3cb38dff68085291aed"

# propose_burn <from_addr> <proof_data_hex> <value_wei> — GovMinter.proposeBurn(bytes), msg.value 포함.
propose_burn() {
  _init_govminter_v2_sigs
  local from_addr="${1:?propose_burn: from_addr required}"
  local proof_data="${2:?propose_burn: proof_data required}"
  local value_wei="${3:?propose_burn: value_wei required}"

  local node_num
  node_num=$(addr_to_node "$from_addr")
  unlock_validator "$node_num"

  proof_data="${proof_data#0x}"
  local data_len=$(( ${#proof_data} / 2 ))
  local padded_len
  padded_len=$(printf '%064x' "$data_len")
  local padded_data="$proof_data"
  local remainder=$(( ${#proof_data} % 64 ))
  if (( remainder > 0 )); then
    local pad_zeros=$(( 64 - remainder ))
    padded_data="${padded_data}$(printf '%0*d' "$pad_zeros" 0)"
  fi
  local calldata="${PROPOSE_BURN_SIG}0000000000000000000000000000000000000000000000000000000000000020${padded_len}${padded_data}"

  local value_hex
  value_hex=$(dec_to_hex "$value_wei")

  local response
  response=$(rpc "$node_num" "eth_sendTransaction" \
    "[{\"from\":\"${from_addr}\",\"to\":\"${GOV_MINTER}\",\"data\":\"0x${calldata}\",\"value\":\"${value_hex}\",\"gas\":\"0x1e8480\"}]") || return 1
  json_get "$response" "result"
}

# claim_burn_refund <from_addr> — claimBurnRefund() on GOV_MINTER.
claim_burn_refund() {
  _init_govminter_v2_sigs
  local from_addr="${1:?claim_burn_refund: from_addr required}"
  local node_num
  node_num=$(addr_to_node "$from_addr")
  unlock_validator "$node_num"

  local response
  response=$(rpc "$node_num" "eth_sendTransaction" \
    "[{\"from\":\"${from_addr}\",\"to\":\"${GOV_MINTER}\",\"data\":\"0x${CLAIM_BURN_REFUND_SIG}\",\"gas\":\"0x1e8480\"}]") || return 1
  json_get "$response" "result"
}

# get_burn_balance <target> <address> — burnBalance, decimal wei.
get_burn_balance() {
  _init_govminter_v2_sigs
  local target="${1:?get_burn_balance: target required}"
  local address="${2:?get_burn_balance: address required}"
  local data="0x${BURN_BALANCE_SIG}$(pad_address "$address")"
  local result
  result=$(eth_call_raw "$target" "$GOV_MINTER" "$data") || return 1
  hex_to_dec "$result"
}

# get_refundable_balance <target> <address> — refundableBalance, decimal wei.
get_refundable_balance() {
  _init_govminter_v2_sigs
  local target="${1:?get_refundable_balance: target required}"
  local address="${2:?get_refundable_balance: address required}"
  local data="0x${REFUNDABLE_BALANCE_SIG}$(pad_address "$address")"
  local result
  result=$(eth_call_raw "$target" "$GOV_MINTER" "$data") || return 1
  hex_to_dec "$result"
}

# disapprove_proposal <contract> <proposal_id> <from_addr>
disapprove_proposal() {
  _init_govminter_v2_sigs
  local contract="${1:?disapprove_proposal: contract required}"
  local proposal_id="${2:?disapprove_proposal: proposal_id required}"
  local from_addr="${3:?disapprove_proposal: from_addr required}"
  local node_num
  node_num=$(addr_to_node "$from_addr")
  unlock_validator "$node_num"
  local data="0x${DISAPPROVE_PROPOSAL_SIG}$(pad_uint256 "$proposal_id")"
  gov_call "$node_num" "$contract" "$data" "$from_addr" "0x1e8480"
}

# cancel_proposal <contract> <proposal_id> <from_addr>
cancel_proposal() {
  _init_govminter_v2_sigs
  local contract="${1:?cancel_proposal: contract required}"
  local proposal_id="${2:?cancel_proposal: proposal_id required}"
  local from_addr="${3:?cancel_proposal: from_addr required}"
  local node_num
  node_num=$(addr_to_node "$from_addr")
  unlock_validator "$node_num"
  local data="0x${CANCEL_PROPOSAL_SIG}$(pad_uint256 "$proposal_id")"
  gov_call "$node_num" "$contract" "$data" "$from_addr" "0x1e8480"
}

# ============================================================================
# 검증 헬퍼
# ============================================================================

# assert_receipt_status <target> <tx_hash> <expected_status>  ("0x1"|"0x0")
assert_receipt_status() {
  local target="$1" tx_hash="$2" expected="$3" msg="${4:-receipt status}"
  local status
  status=$(wait_receipt "$target" "$tx_hash" 30 || echo "timeout")
  case "$expected" in
    "0x1") [[ "$status" == "success" ]] && _assert_pass "$msg: success" || _assert_fail "$msg: expected success, got $status" ;;
    "0x0") [[ "$status" == "failed" ]] && _assert_pass "$msg: failed (expected)" || _assert_fail "$msg: expected failed, got $status" ;;
    *)     _assert_fail "$msg: invalid expected status $expected" ;;
  esac
}

# assert_error_contains <error_output> <substring> [msg]
assert_error_contains() {
  local err="$1" substr="$2" msg="${3:-error message}"
  if [[ "$err" == *"$substr"* ]]; then
    _assert_pass "$msg: contains '$substr'"
  else
    _assert_fail "$msg: missing '$substr' in '$err'"
  fi
}

# ============================================================================
# 환경 확인 / 노드 상태
# ============================================================================

# check_env — 회귀 테스트 전제 도구 확인 (cast, jq)
check_env() {
  local ok=true
  if ! command -v cast >/dev/null 2>&1; then
    printf '[ERROR] cast (Foundry) not found. Install: https://getfoundry.sh\n' >&2
    ok=false
  fi
  if ! command -v jq >/dev/null 2>&1; then
    printf '[ERROR] jq not found. Install: brew install jq\n' >&2
    ok=false
  fi
  [[ "$ok" == "true" ]]
}

# ensure_nodes_running — 활성 노드제어 백엔드로 위임.
# 백엔드(tests/regression/lib/node_ctrl/${CB_NODECTRL_BACKEND}.sh)가
# cb_nodectrl_<backend>_ensure 를 정의한다(W8). 미구현이면 no-op(기동된 노드 가정).
ensure_nodes_running() {
  local fn="cb_nodectrl_${CB_NODECTRL_BACKEND}_ensure"
  if declare -F "$fn" >/dev/null 2>&1; then
    "$fn"
  fi
}
