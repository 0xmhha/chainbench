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
readonly ACCOUNT_MANAGER="0x0000000000000000000000000000000000b00003"
readonly NATIVE_COIN_MANAGER="0x0000000000000000000000000000000000b00002"
readonly BLS_POP="0x0000000000000000000000000000000000b00001"

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
# Uses python3 to handle arbitrarily large integers (e.g. wei balances > 2^63).
# bash printf '%d' is limited to int64 and fails with "Result too large" on
# values like 10^27 wei (typical regression alloc).
hex_to_dec() {
  local hex="${1#0x}"
  [[ -z "$hex" || "$hex" == "null" ]] && { echo "0"; return; }
  python3 -c "print(int('${hex}', 16))"
}

# to_checksum <address>
# Convert an Ethereum address to EIP-55 checksum form.
# Required because eth-account >=0.13 rejects all-lowercase addresses
# when passed as the tx["to"] field. receipt.contractAddress is returned
# in lowercase by nodes, so we normalise before use.
to_checksum() {
  local addr="$1"
  [[ -z "$addr" || "$addr" == "null" ]] && { echo ""; return; }
  python3 -c "from eth_utils import to_checksum_address; print(to_checksum_address('$addr'))"
}

# get_header_gas_tip <target>
# latest 블록의 WBFTExtra.GasTip을 decimal string으로 반환.
# istanbul_getWbftExtraInfo는 "latest" 파라미터를 거부하므로 실제 block number 사용.
# gasTip은 decimal string으로 반환됨 (hex가 아님).
get_header_gas_tip() {
  local target="${1:-1}"
  local latest_hex
  latest_hex=$(rpc "$target" "eth_blockNumber" "[]" | json_get - result)
  rpc "$target" "istanbul_getWbftExtraInfo" "[\"${latest_hex}\"]" | python3 -c "
import sys, json
try:
    r = json.load(sys.stdin).get('result', {})
    gt = r.get('gasTip', '0')
    if isinstance(gt, str):
        print(int(gt, 16) if gt.startswith('0x') else int(gt) if gt else 0)
    else:
        print(int(gt) if gt else 0)
except Exception:
    print(0)
"
}

# get_wbft_extra_json <target> [block_hex]
# istanbul_getWbftExtraInfo 응답의 result를 JSON string으로 반환.
# block_hex 생략 시 latest block 사용.
get_wbft_extra_json() {
  local target="${1:-1}" block_hex="${2:-}"
  if [[ -z "$block_hex" ]]; then
    block_hex=$(rpc "$target" "eth_blockNumber" "[]" | json_get - result)
  fi
  rpc "$target" "istanbul_getWbftExtraInfo" "[\"${block_hex}\"]"
}

# dec_to_hex <decimal>
# "255" → "0xff"
dec_to_hex() {
  printf '0x%x\n' "$1"
}

# json_get <json_or_dash> <key_path>
# Extract a field from a JSON object using python.
# If the first argument is "-", reads JSON from stdin instead.
# This enables both usage patterns:
#   resp=$(rpc ...); json_get "$resp" "result.hash"
#   rpc ... | json_get - result.hash
json_get() {
  local json="$1" path="$2"
  if [[ "$json" == "-" ]]; then
    json="$(cat)"
  fi
  # Guard against empty or non-JSON input — return empty string instead of crashing.
  if [[ -z "$json" ]]; then
    return 0
  fi
  printf '%s' "$json" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
except Exception:
    print('')
    sys.exit(0)
path = sys.argv[1].split('.')
for p in path:
    if isinstance(data, list):
        try:
            data = data[int(p)]
        except (ValueError, IndexError):
            print('')
            sys.exit(0)
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

# wait_for_block <target> <block_number_decimal> [timeout_secs]
# Polls eth_blockNumber until the chain reaches the target block.
# Returns 0 when reached, 1 on timeout.
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
    # NOTE: eth-account >=0.13 rejects explicit type=0. Legacy tx is implied
    # by the presence of gasPrice without any type field.
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
    "params": [signed.raw_transaction.to_0x_hex()], "id": 1
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

# addr_to_node <validator_address>
# Validator 주소를 해당 node index로 매핑한다.
# chainbench는 노드별로 자기 keystore만 가지므로 (node 1은 V1 key만, node 2는 V2 key만),
# eth_sendTransaction은 반드시 그 validator의 홈 노드에서 호출해야 한다.
addr_to_node() {
  local addr="${1,,}"  # lowercase
  case "$addr" in
    "${VALIDATOR_1_ADDR,,}") echo "1" ;;
    "${VALIDATOR_2_ADDR,,}") echo "2" ;;
    "${VALIDATOR_3_ADDR,,}") echo "3" ;;
    "${VALIDATOR_4_ADDR,,}") echo "4" ;;
    *) echo "1" ;;  # fallback (non-validator)
  esac
}

# extract_proposal_id_from_receipt <target> <tx_hash>
# ProposalCreated 이벤트 topics[1]에서 proposalId(hex) 추출
# ProposalCreated(uint256 indexed proposalId, ...) 시그니처 가정
extract_proposal_id_from_receipt() {
  local target="${1:-1}" tx_hash="$2"
  local receipt
  receipt=$(wait_tx_receipt_full "$target" "$tx_hash" 30) || return 1
  # ProposalCreated(uint256 indexed proposalId, address indexed proposer, ...) 이벤트를
  # 시그니처(topics[0])로 정확히 매칭한 뒤 topics[1]에서 proposalId 추출
  printf '%s' "$receipt" | python3 -c "
import sys, json
PROPOSAL_CREATED_SIG = '0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a'
r = json.load(sys.stdin)
logs = r.get('logs', [])
for log in logs:
    topics = log.get('topics', [])
    if len(topics) >= 2 and topics[0].lower() == PROPOSAL_CREATED_SIG:
        print(int(topics[1], 16))
        break
else:
    # Fallback: 첫 log의 topics[1] (하위 호환)
    for log in logs:
        topics = log.get('topics', [])
        if len(topics) >= 2:
            print(int(topics[1], 16))
            break
    else:
        print('')
"
}

# gov_propose <target> <contract> <propose_data> <from_addr>
# proposal 생성 tx 전송 → receipt 대기 → proposalId 추출
# 반환: proposalId (decimal)
# NOTE: <target> 인자는 호환성을 위해 남기지만 실제 tx 전송 노드는 from_addr의 홈 노드
# (chainbench 노드별 keystore isolation 때문에 from의 홈 노드에서만 eth_sendTransaction 가능)
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

# gov_approve <target> <contract> <proposal_id> <from_addr>
# approveProposal(uint256) 호출. 반드시 from의 홈 노드에서 전송해야 함.
gov_approve() {
  local _target_unused="${1:-1}" contract="$2" proposal_id="$3" from="$4"
  local sel padded node
  sel=$(selector "approveProposal(uint256)")
  padded=$(pad_uint256 "$proposal_id" | sed 's/^0x//')
  local data="${sel}${padded}"
  node=$(addr_to_node "$from")
  gov_call "$node" "$contract" "$data" "$from" 500000
}

# gov_execute <target> <contract> <proposal_id> <from_addr>
# executeProposal(uint256) 호출. 반드시 from의 홈 노드에서 전송해야 함.
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
# 프로포절 상태를 enum 값으로 반환: None=0, Voting=1, Approved=2, Executed=3,
# Cancelled=4, Expired=5, Failed=6, Rejected=7
#
# GovBase는 별도의 proposalStatus(uint256) 함수가 없고, `mapping(uint256 => Proposal) public proposals`
# 의 automatic getter만 제공한다. dynamic field(callData)는 자동 getter에서 제외되므로
# 10개의 primitive field를 tuple로 decode한다. status는 10번째 (index 9).
# struct Proposal order:
#   bytes32 actionType; uint256 memberVersion; uint256 votedBitmap;
#   uint256 createdAt; uint256 executedAt; address proposer;
#   uint32 requiredApprovals; uint32 approved; uint32 rejected;
#   ProposalStatus status;  // uint8
gov_proposal_status() {
  local target="${1:-1}" contract="$2" proposal_id="$3"
  local sel padded call_result
  sel=$(selector "proposals(uint256)")
  padded=$(pad_uint256 "$proposal_id" | sed 's/^0x//')
  call_result=$(eth_call_raw "$target" "$contract" "${sel}${padded}")
  python3 -c "
from eth_abi import decode
raw_hex = '${call_result}'
if raw_hex.startswith('0x'):
    raw_hex = raw_hex[2:]
if not raw_hex:
    print(0)
else:
    try:
        raw = bytes.fromhex(raw_hex)
        types = ['bytes32','uint256','uint256','uint256','uint256','address','uint32','uint32','uint32','uint8']
        result = decode(types, raw)
        print(result[9])
    except Exception:
        print(0)
"
}

# gov_full_flow <contract> <propose_data> <proposer_addr> <approver1> [approver2] ...
# 전체 lifecycle: propose → approve (N명) → execute → 실행 receipt 반환
# 반환: execute tx의 전체 receipt JSON
#
# 진단 로깅: propose 후 + 각 approve 후 proposal status 출력하여 실패 지점 파악.
# approve tx가 revert되면 early return (quorum 미달 방지).
gov_full_flow() {
  local contract="$1" propose_data="$2" proposer="$3"
  shift 3
  local approvers=("$@")

  # 1) propose (node1 기준)
  local proposal_id
  proposal_id=$(gov_propose "1" "$contract" "$propose_data" "$proposer") || return 1
  [[ -z "$proposal_id" ]] && return 1
  local status_after_propose
  status_after_propose=$(gov_proposal_status "1" "$contract" "$proposal_id")
  printf '[GOV]  proposalId=%s (status after propose=%s)\n' "$proposal_id" "$status_after_propose" >&2

  # 2) approve (각 approver) — receipt status 검증
  # GovBase는 quorum 도달 시 approve tx 안에서 자동으로 execute 한다.
  # 마지막 approver의 tx가 auto-execute를 포함할 수 있으므로 proposal.status==Executed(3)
  # 이면 별도 executeProposal 호출 없이 해당 approve receipt을 반환한다.
  local last_approve_receipt=""
  for ap in "${approvers[@]}"; do
    local tx
    tx=$(gov_approve "1" "$contract" "$proposal_id" "$ap") || true
    if [[ -z "$tx" || "$tx" == "null" ]]; then
      printf '[GOV]  approve by %s: NO TX HASH (submission failed)\n' "$ap" >&2
      return 1
    fi
    # Receipt 대기 + status 체크 (revert 시 early fail)
    local receipt_json approve_status
    receipt_json=$(wait_tx_receipt_full "1" "$tx" 15 2>/dev/null || echo "")
    approve_status=$(printf '%s' "$receipt_json" | python3 -c "
import sys, json
try:
    d = json.loads(sys.stdin.read() or '{}')
    print(d.get('status', ''))
except Exception:
    print('')
" 2>/dev/null)
    local prop_status
    prop_status=$(gov_proposal_status "1" "$contract" "$proposal_id")
    printf '[GOV]  approved by %s (tx=%s, receipt.status=%s, proposal.status=%s)\n' \
      "$ap" "$tx" "${approve_status:-none}" "$prop_status" >&2
    if [[ "$approve_status" != "0x1" ]]; then
      printf '[GOV]  ERROR: approve tx reverted/missing — aborting before execute\n' >&2
      return 1
    fi
    last_approve_receipt="$receipt_json"
    # Auto-executed by quorum reach in this approve — skip explicit execute
    if [[ "$prop_status" == "3" ]]; then
      printf '[GOV]  auto-executed during approve (status=Executed)\n' >&2
      printf '%s' "$last_approve_receipt"
      return 0
    fi
  done

  # 3) execute (quorum 미달/수동 execute 필요한 경우)
  local exec_tx
  exec_tx=$(gov_execute "1" "$contract" "$proposal_id" "$proposer") || return 1
  [[ -z "$exec_tx" || "$exec_tx" == "null" ]] && return 1
  printf '[GOV]  executeTx=%s\n' "$exec_tx" >&2

  # 4) receipt 반환
  wait_tx_receipt_full "1" "$exec_tx" 30
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

# propose_burn <from_addr> <proof_data_hex> <value_wei>
# Calls GovMinter.proposeBurn(bytes) with msg.value. Returns tx hash.
propose_burn() {
  _init_govminter_v2_sigs
  local from_addr="${1:?propose_burn: from_addr required}"
  local proof_data="${2:?propose_burn: proof_data required}"
  local value_wei="${3:?propose_burn: value_wei required}"

  local node_num
  node_num=$(addr_to_node "$from_addr")
  unlock_validator "$node_num"

  # ABI encode: proposeBurn(bytes) = selector + offset(0x20) + length + data
  proof_data="${proof_data#0x}"
  local data_len=$(( ${#proof_data} / 2 ))
  local padded_len
  padded_len=$(printf '%064x' "$data_len")
  local padded_data="$proof_data"
  # Pad to 32-byte boundary
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

# claim_burn_refund <from_addr>
# Calls claimBurnRefund() on GOV_MINTER. Returns tx hash.
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

# get_burn_balance <target> <address>
# Returns burnBalance in decimal wei.
get_burn_balance() {
  _init_govminter_v2_sigs
  local target="${1:?get_burn_balance: target required}"
  local address="${2:?get_burn_balance: address required}"
  local data="0x${BURN_BALANCE_SIG}$(pad_address "$address")"
  local result
  result=$(eth_call_raw "$target" "$GOV_MINTER" "$data") || return 1
  hex_to_dec "$result"
}

# get_refundable_balance <target> <address>
# Returns refundableBalance in decimal wei.
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
# Calls disapproveProposal(uint256). Returns tx hash.
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
# Calls cancelProposal(uint256). Returns tx hash.
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
