#!/usr/bin/env bash
# tests/regression/lib/common.sh — go-stablenet v2 회귀 테스트 공통 헬퍼
#
# 사용법:
#   source "$(dirname "$0")/../lib/common.sh"
#
# 제공 기능:
#   - 테스트 계정 (Hardhat 기본 키 5개) 주소/PrivateKey 매핑
#   - Validator 주소 배열 (node1~node4)
#   - eth_call / raw tx / receipt 조회 헬퍼
#   - Governance 프로포절 생성/투표/실행 헬퍼
#   - Keccak-256 해시, ABI selector 계산
#   - Log/event 검색 헬퍼
#   - Python 기반 서명/인코딩 유틸 (eth-account 사용)

[[ -n "${_CB_REGRESSION_COMMON_LOADED:-}" ]] && return 0
readonly _CB_REGRESSION_COMMON_LOADED=1

CHAINBENCH_DIR="${CHAINBENCH_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)}"
source "${CHAINBENCH_DIR}/tests/lib/rpc.sh"
source "${CHAINBENCH_DIR}/tests/lib/assert.sh"
source "${CHAINBENCH_DIR}/tests/lib/wait.sh"

# ============================================================================
# 테스트 계정 (Hardhat/Anvil 기본 계정 — 공개된 test private key)
# ============================================================================

# 주소
readonly TEST_ACC_A_ADDR="0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
readonly TEST_ACC_B_ADDR="0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
readonly TEST_ACC_C_ADDR="0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC"
readonly TEST_ACC_D_ADDR="0x90F79bf6EB2c4f870365E785982E1f101E93b906"
readonly TEST_ACC_E_ADDR="0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65"

# Private key (0x 접두사 포함)
readonly TEST_ACC_A_PK="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
readonly TEST_ACC_B_PK="0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
readonly TEST_ACC_C_PK="0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a"
readonly TEST_ACC_D_PK="0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6"
readonly TEST_ACC_E_PK="0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a"

# Validator 계정 (preset keystore에 저장됨, unlock 필요)
readonly VALIDATOR_1_ADDR="0xc17d493883eaa3b4cceb0f214b273392d562f9d8"
readonly VALIDATOR_2_ADDR="0x2493a84a8f83cb87fdcbe0bb3b2d313f69a58d3c"
readonly VALIDATOR_3_ADDR="0x8c4a10b9108d49b9d23f764464090831d9c17764"
readonly VALIDATOR_4_ADDR="0x8eb79036bc0f3aba136ef18b3a2fb8c1188939a6"
readonly VALIDATOR_KEYSTORE_PASSWORD="1"

# 시스템 컨트랙트 주소
readonly NATIVE_COIN_ADAPTER="0x0000000000000000000000000000000000001000"
readonly GOV_VALIDATOR="0x0000000000000000000000000000000000001001"
readonly GOV_MASTER_MINTER="0x0000000000000000000000000000000000001002"
readonly GOV_MINTER="0x0000000000000000000000000000000000001003"
readonly GOV_COUNCIL="0x0000000000000000000000000000000000001004"
readonly ACCOUNT_MANAGER="0x00000000000000000000000000000000000B00003"
readonly NATIVE_COIN_MANAGER="0x00000000000000000000000000000000000B00002"
readonly BLS_POP="0x00000000000000000000000000000000000B00001"

# 상수
readonly ZERO_ADDRESS="0x0000000000000000000000000000000000000000"
readonly MIN_BASE_FEE_WEI="20000000000000"  # 20 Gwei

# 이벤트 시그니처 해시
readonly TRANSFER_EVENT_SIG="0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
readonly AUTHORIZED_TX_EXECUTED_SIG="0x40e728a89c7f5b192cf1c1b747fb64d51d81c7a2b3ed4607b94d3a1e6a3e0373"

# ============================================================================
# 기본 유틸리티
# ============================================================================

# hex_to_dec <hex_with_0x>
# "0x10" → "16"
hex_to_dec() {
  local hex="${1#0x}"
  [[ -z "$hex" || "$hex" == "null" ]] && { echo "0"; return; }
  printf '%d\n' "0x${hex}"
}

# dec_to_hex <decimal>
# "255" → "0xff"
dec_to_hex() {
  printf '0x%x\n' "$1"
}

# json_get <json> <key_path>
# Extract a field from a JSON object using python
json_get() {
  local json="$1" path="$2"
  printf '%s' "$json" | python3 -c "
import sys, json
data = json.load(sys.stdin)
path = sys.argv[1].split('.')
for p in path:
    if isinstance(data, list):
        data = data[int(p)]
    elif isinstance(data, dict):
        data = data.get(p)
    if data is None:
        print('')
        sys.exit(0)
print(data if not isinstance(data, (dict, list)) else json.dumps(data))
" "$path"
}

# ============================================================================
# 계정 관리
# ============================================================================

# unlock_validator <node_num>
# node N의 coinbase(validator) 계정을 keystore 비밀번호로 잠금 해제
unlock_validator() {
  local node="${1:-1}"
  local addr
  addr=$(get_coinbase "$node")
  [[ -z "$addr" || "$addr" == "null" ]] && return 1
  unlock_account "$node" "$addr" "$VALIDATOR_KEYSTORE_PASSWORD" 3600 >/dev/null 2>&1
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

# get_base_fee <target>
# 현재 블록의 baseFee를 decimal로 반환
get_base_fee() {
  local target="${1:-1}"
  local response result
  response=$(rpc "$target" "eth_getBlockByNumber" "[\"latest\", false]") || return 1
  result=$(json_get "$response" "result.baseFeePerGas")
  hex_to_dec "$result"
}

# ============================================================================
# Raw Transaction (eth-account 사용)
# ============================================================================

# send_raw_tx <target> <private_key> <to> <value_wei> [data] [gas_limit] [tx_type]
# tx_type: legacy, dynamic (default), accesslist, setcode
# value, gas_limit: decimal string
# data: hex string with or without 0x (default: "")
# 반환: txHash (0x... 형식)
send_raw_tx() {
  local target="$1" pk="$2" to="$3" value="$4"
  local data="${5:-}" gas_limit="${6:-21000}" tx_type="${7:-dynamic}"

  local port
  port=$(pids_get_field "$target" "http_port") || return 1

  local chain_id
  chain_id=$(rpc "$target" "eth_chainId" "[]" | json_get - result)
  chain_id=$(hex_to_dec "$chain_id")

  python3 <<PYEOF
import json, os, sys
from eth_account import Account
from eth_account.messages import encode_defunct
import requests

pk = "${pk}"
to = "${to}"
value = int("${value}")
data = "${data}"
gas_limit = int("${gas_limit}")
tx_type = "${tx_type}"
chain_id = int("${chain_id}")
url = "http://127.0.0.1:${port}"

acct = Account.from_key(pk)
nonce_resp = requests.post(url, json={
    "jsonrpc": "2.0", "method": "eth_getTransactionCount",
    "params": [acct.address, "pending"], "id": 1
}).json()
nonce = int(nonce_resp["result"], 16)

blk_resp = requests.post(url, json={
    "jsonrpc": "2.0", "method": "eth_getBlockByNumber",
    "params": ["latest", False], "id": 1
}).json()
base_fee = int(blk_resp["result"].get("baseFeePerGas", "0x0"), 16)

# Default fees — generous tip to ensure inclusion
max_priority = 27600000000000  # 27.6 Gwei (matches default gasTip)
max_fee = base_fee + max_priority * 2

tx = {
    "nonce": nonce,
    "to": to,
    "value": value,
    "gas": gas_limit,
    "chainId": chain_id,
}
if data:
    if not data.startswith("0x"):
        data = "0x" + data
    tx["data"] = data

if tx_type == "legacy":
    tx["gasPrice"] = base_fee + max_priority
    tx["type"] = 0
elif tx_type == "dynamic":
    tx["maxFeePerGas"] = max_fee
    tx["maxPriorityFeePerGas"] = max_priority
    tx["type"] = 2
elif tx_type == "accesslist":
    tx["gasPrice"] = base_fee + max_priority
    tx["accessList"] = []
    tx["type"] = 1

signed = acct.sign_transaction(tx)
send_resp = requests.post(url, json={
    "jsonrpc": "2.0", "method": "eth_sendRawTransaction",
    "params": [signed.rawTransaction.hex()], "id": 1
}).json()

if "error" in send_resp:
    print(f"TX_ERROR:{send_resp['error'].get('message', 'unknown')}", file=sys.stderr)
    sys.exit(1)
print(send_resp["result"])
PYEOF
}

# ============================================================================
# Receipt / Log 조회
# ============================================================================

# get_receipt <target> <tx_hash>
# 전체 receipt JSON 객체 반환
get_receipt() {
  local target="${1:-1}" tx_hash="$2"
  local response
  response=$(rpc "$target" "eth_getTransactionReceipt" "[\"${tx_hash}\"]") || return 1
  json_get "$response" "result"
}

# get_receipt_field <target> <tx_hash> <field>
# receipt의 특정 필드 반환 (예: status, gasUsed, effectiveGasPrice)
get_receipt_field() {
  local target="$1" tx_hash="$2" field="$3"
  local receipt
  receipt=$(get_receipt "$target" "$tx_hash") || return 1
  printf '%s' "$receipt" | python3 -c "
import sys, json
r = json.load(sys.stdin) if sys.stdin.read(1) != '' else None
sys.stdin.seek(0)
try:
    r = json.load(sys.stdin)
    print(r.get('${field}', ''))
except Exception:
    print('')
"
}

# wait_tx_receipt_full <target> <tx_hash> [timeout]
# receipt 대기 후 전체 JSON 반환 (없으면 빈 문자열)
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
# receipt.logs 배열에서 특정 주소/topic0에 매치되는 첫 log 반환
find_log_by_topic() {
  local receipt="$1" address="$2" topic0="$3"
  printf '%s' "$receipt" | python3 -c "
import sys, json
r = json.load(sys.stdin)
logs = r.get('logs', [])
addr = '${address}'.lower()
topic0 = '${topic0}'.lower()
for log in logs:
    log_addr = log.get('address', '').lower()
    topics = [t.lower() for t in log.get('topics', [])]
    if log_addr == addr and topics and topics[0] == topic0:
        print(json.dumps(log))
        break
"
}

# count_logs_by_address <receipt_json> <address>
count_logs_by_address() {
  local receipt="$1" address="$2"
  printf '%s' "$receipt" | python3 -c "
import sys, json
r = json.load(sys.stdin)
logs = r.get('logs', [])
addr = '${address}'.lower()
count = sum(1 for log in logs if log.get('address', '').lower() == addr)
print(count)
"
}

# ============================================================================
# eth_call (view 함수 호출)
# ============================================================================

# eth_call_raw <target> <to> <data_hex>
# 원시 eth_call 결과(0x...) 반환
eth_call_raw() {
  local target="${1:-1}" to="$2" data="$3"
  [[ "$data" != 0x* ]] && data="0x$data"
  local response result
  response=$(rpc "$target" "eth_call" \
    "[{\"to\":\"${to}\",\"data\":\"${data}\"}, \"latest\"]") || return 1
  result=$(json_get "$response" "result")
  printf '%s' "$result"
}

# selector <function_signature>
# "transfer(address,uint256)" → "0xa9059cbb"
selector() {
  local sig="$1"
  python3 -c "
from eth_utils import keccak
sig = '${sig}'
print('0x' + keccak(text=sig)[:4].hex())
"
}

# pad_address <addr>
# 20-byte 주소 → 32-byte padded (0x + 64 hex)
pad_address() {
  local addr="${1#0x}"
  printf '0x%064s\n' "$addr" | tr ' ' '0'
}

# pad_uint256 <decimal_or_hex>
pad_uint256() {
  local val="$1"
  python3 -c "
v = '${val}'
if v.startswith('0x'):
    n = int(v, 16)
else:
    n = int(v)
print('0x' + format(n, '064x'))
"
}

# ============================================================================
# Governance 헬퍼 (proposal lifecycle)
# ============================================================================

# gov_call <target> <contract> <function_selector_with_args> <from_addr> <gas>
# governance proposal/approve/execute tx 전송 (validator coinbase unlock 전제)
gov_call() {
  local target="${1:-1}" contract="$2" data="$3" from="$4" gas="${5:-800000}"
  local response
  response=$(rpc "$target" "eth_sendTransaction" \
    "[{\"from\":\"${from}\",\"to\":\"${contract}\",\"data\":\"${data}\",\"gas\":\"$(dec_to_hex "$gas")\"}]") || return 1
  json_get "$response" "result"
}

# unlock_all_validators
# 모든 BP validator(node1~4)의 coinbase 계정을 unlock. governance proposal/vote 선행 필수
unlock_all_validators() {
  for node in 1 2 3 4; do
    unlock_validator "$node" >/dev/null 2>&1 || true
  done
}

# extract_proposal_id_from_receipt <target> <tx_hash>
# ProposalCreated 이벤트 topics[1]에서 proposalId(hex) 추출
# ProposalCreated(uint256 indexed proposalId, ...) 시그니처 가정
extract_proposal_id_from_receipt() {
  local target="${1:-1}" tx_hash="$2"
  local receipt
  receipt=$(wait_tx_receipt_full "$target" "$tx_hash" 30) || return 1
  # 첫 log의 topics[1] (proposalId는 보통 indexed)
  printf '%s' "$receipt" | python3 -c "
import sys, json
r = json.load(sys.stdin)
logs = r.get('logs', [])
for log in logs:
    topics = log.get('topics', [])
    if len(topics) >= 2:
        # ProposalCreated 이벤트의 topics[1]이 proposalId
        print(int(topics[1], 16))
        break
else:
    print('')
"
}

# gov_propose <target> <contract> <propose_data> <from_addr>
# proposal 생성 tx 전송 → receipt 대기 → proposalId 추출
# 반환: proposalId (decimal)
gov_propose() {
  local target="$1" contract="$2" data="$3" from="$4"
  local tx_hash
  tx_hash=$(gov_call "$target" "$contract" "$data" "$from" 800000) || return 1
  [[ -z "$tx_hash" || "$tx_hash" == "null" ]] && return 1
  extract_proposal_id_from_receipt "$target" "$tx_hash"
}

# gov_approve <target> <contract> <proposal_id> <from_addr>
# approveProposal(uint256) 호출
gov_approve() {
  local target="${1:-1}" contract="$2" proposal_id="$3" from="$4"
  local sel padded
  sel=$(selector "approveProposal(uint256)")
  padded=$(pad_uint256 "$proposal_id" | sed 's/^0x//')
  local data="${sel}${padded}"
  gov_call "$target" "$contract" "$data" "$from" 500000
}

# gov_execute <target> <contract> <proposal_id> <from_addr>
# executeProposal(uint256) 호출
gov_execute() {
  local target="${1:-1}" contract="$2" proposal_id="$3" from="$4"
  local sel padded
  sel=$(selector "executeProposal(uint256)")
  padded=$(pad_uint256 "$proposal_id" | sed 's/^0x//')
  local data="${sel}${padded}"
  gov_call "$target" "$contract" "$data" "$from" 1500000
}

# gov_proposal_status <target> <contract> <proposal_id>
# 프로포절 상태를 enum 값으로 반환: None=0, Voting=1, Approved=2, Executed=3, Cancelled=4, Expired=5, Failed=6, Rejected=7
gov_proposal_status() {
  local target="${1:-1}" contract="$2" proposal_id="$3"
  local sel padded call_result
  sel=$(selector "proposalStatus(uint256)")
  padded=$(pad_uint256 "$proposal_id" | sed 's/^0x//')
  call_result=$(eth_call_raw "$target" "$contract" "${sel}${padded}")
  hex_to_dec "$call_result"
}

# gov_full_flow <contract> <propose_data> <proposer_addr> <approver1> [approver2] ...
# 전체 lifecycle: propose → approve (N명) → execute → 실행 receipt 반환
# 반환: execute tx의 전체 receipt JSON
gov_full_flow() {
  local contract="$1" propose_data="$2" proposer="$3"
  shift 3
  local approvers=("$@")

  # 1) propose (node1 기준)
  local proposal_id
  proposal_id=$(gov_propose "1" "$contract" "$propose_data" "$proposer") || return 1
  [[ -z "$proposal_id" ]] && return 1
  printf '[GOV]  proposalId=%s\n' "$proposal_id" >&2

  # 2) approve (각 approver)
  for ap in "${approvers[@]}"; do
    local tx
    tx=$(gov_approve "1" "$contract" "$proposal_id" "$ap") || true
    if [[ -n "$tx" && "$tx" != "null" ]]; then
      wait_receipt "1" "$tx" 15 >/dev/null 2>&1 || true
      printf '[GOV]  approved by %s (tx=%s)\n' "$ap" "$tx" >&2
    fi
  done

  # 3) execute
  local exec_tx
  exec_tx=$(gov_execute "1" "$contract" "$proposal_id" "$proposer") || return 1
  [[ -z "$exec_tx" || "$exec_tx" == "null" ]] && return 1
  printf '[GOV]  executeTx=%s\n' "$exec_tx" >&2

  # 4) receipt 반환
  wait_tx_receipt_full "1" "$exec_tx" 30
}

# ============================================================================
# 검증 헬퍼
# ============================================================================

# assert_receipt_status <target> <tx_hash> <expected_status>
# expected_status: "0x1" (success) or "0x0" (failed)
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
# 환경 확인
# ============================================================================

# check_env
# 회귀 테스트 실행 전제 조건 확인 (python3, eth-account 설치 여부 등)
check_env() {
  if ! command -v python3 >/dev/null 2>&1; then
    printf '[ERROR] python3 not found\n' >&2
    return 1
  fi
  python3 -c "import eth_account, requests, eth_utils, eth_abi" 2>/dev/null || {
    printf '[ERROR] Required Python packages missing. Install with:\n' >&2
    printf '        pip3 install eth-account requests eth-utils eth-abi rlp eth-keys websockets\n' >&2
    return 1
  }
  return 0
}
