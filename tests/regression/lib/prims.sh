#!/usr/bin/env bash
# tests/regression/lib/prims.sh — 저수준 primitives 백엔드 (도구 추상화)
#
# 환경마다 가용 도구가 다르다(로컬=python eth-*, 폐쇄망=cast). 서명 백엔드처럼,
# hex/dec·checksum·selector·padding·abi-decode·클라이언트 서명을 cast 또는 python으로
# 자동 디스패치한다. JSON 파싱은 jq(양 환경 공통)라 여기서 다루지 않는다.
#
# 감지 1회: CB_PRIM_BACKEND = cast | python | "" (둘 다 없으면 빈 값 → check_env가 막음)

[[ -n "${_CB_PRIMS_LOADED:-}" ]] && return 0
_CB_PRIMS_LOADED=1

_cb_prim_detect() {
  if command -v cast >/dev/null 2>&1; then
    CB_PRIM_BACKEND="cast"
  elif command -v python3 >/dev/null 2>&1 && \
       python3 -c "import eth_account, eth_utils, eth_abi, requests" 2>/dev/null; then
    CB_PRIM_BACKEND="python"
  else
    CB_PRIM_BACKEND=""
  fi
  export CB_PRIM_BACKEND
}
_cb_prim_detect

# ----------------------------------------------------------------------------
# hex_to_dec <hex_with_0x>  — 큰 정수(2^63↑) 안전
# ----------------------------------------------------------------------------
hex_to_dec() {
  local hex="${1#0x}"
  [[ -z "$hex" || "$hex" == "null" ]] && { echo "0"; return; }
  case "$CB_PRIM_BACKEND" in
    cast)   cast to-dec "0x${hex}" ;;
    python) python3 -c "print(int('${hex}', 16))" ;;
    *)      echo "0" ;;
  esac
}

# ----------------------------------------------------------------------------
# to_checksum <address>  — EIP-55
# ----------------------------------------------------------------------------
to_checksum() {
  local addr="$1"
  [[ -z "$addr" || "$addr" == "null" ]] && { echo ""; return; }
  case "$CB_PRIM_BACKEND" in
    cast)   cast to-checksum "$addr" ;;
    python) python3 -c "from eth_utils import to_checksum_address; print(to_checksum_address('$addr'))" ;;
    *)      echo "$addr" ;;
  esac
}

# ----------------------------------------------------------------------------
# selector <function_signature>  — "transfer(address,uint256)" → "0xa9059cbb"
# ----------------------------------------------------------------------------
selector() {
  local sig="$1"
  case "$CB_PRIM_BACKEND" in
    cast)   cast sig "$sig" ;;
    python) python3 -c "from eth_utils import keccak; print('0x' + keccak(text='${sig}')[:4].hex())" ;;
  esac
}

# ----------------------------------------------------------------------------
# pad_uint256 <decimal_or_hex>  → 0x + 64 hex
# ----------------------------------------------------------------------------
pad_uint256() {
  local val="$1"
  case "$CB_PRIM_BACKEND" in
    cast)
      local hex
      if [[ "$val" == 0x* ]]; then hex="${val#0x}"; else hex=$(cast to-hex "$val" 2>/dev/null | sed 's/^0x//'); fi
      printf '0x%064s\n' "$hex" | tr ' ' '0'
      ;;
    python)
      python3 -c "
v='${val}'
n=int(v,16) if v.startswith('0x') else int(v)
print('0x'+format(n,'064x'))
"
      ;;
  esac
}

# ----------------------------------------------------------------------------
# _cb_abi_decode_proposal_status <call_result>  — proposals() tuple → status(uint8, idx 9)
# ----------------------------------------------------------------------------
_cb_abi_decode_proposal_status() {
  local call_result="$1"
  [[ -z "$call_result" || "$call_result" == "0x" ]] && { echo "0"; return; }
  case "$CB_PRIM_BACKEND" in
    cast)
      cast abi-decode \
        "proposals()(bytes32,uint256,uint256,uint256,uint256,address,uint32,uint32,uint32,uint8)" \
        "$call_result" 2>/dev/null | tail -1 || echo "0"
      ;;
    python)
      python3 -c "
from eth_abi import decode
raw='${call_result}'
raw=raw[2:] if raw.startswith('0x') else raw
if not raw:
    print(0)
else:
    try:
        r=decode(['bytes32','uint256','uint256','uint256','uint256','address','uint32','uint32','uint32','uint8'], bytes.fromhex(raw))
        print(r[9])
    except Exception:
        print(0)
"
      ;;
  esac
}

# ----------------------------------------------------------------------------
# send_raw_tx <target> <pk> <to> <value> [data] [gas] [type] [tip] [feecap]
# 클라이언트 서명 전송. cast send 또는 python(eth-account)으로 디스패치.
# 반환: txHash. 폐쇄망 본문은 직접 호출 말고 tx_send_as(node_keystore)를 쓸 것.
# ----------------------------------------------------------------------------
send_raw_tx() {
  case "$CB_PRIM_BACKEND" in
    cast)   _cb_send_raw_tx_cast "$@" ;;
    python) _cb_send_raw_tx_python "$@" ;;
    *)      printf '[ERROR] send_raw_tx: no primitives backend (cast/python)\n' >&2; return 1 ;;
  esac
}

_cb_send_raw_tx_cast() {
  local target="$1" pk="$2" to="$3" value="$4"
  local data="${5:-}" gas_limit="${6:-21000}" tx_type="${7:-dynamic}"
  local tip_cap="${8:-}" fee_cap="${9:-}"

  local url; url=$(get_node_url "$target") || return 1
  local cast_args=(--rpc-url "$url" --private-key "$pk" --gas-limit "$gas_limit" --async)

  local base_fee priority_fee
  base_fee=$(get_base_fee "$target")
  priority_fee=$(get_priority_fee "$target")
  [[ -n "$tip_cap" ]] && priority_fee="$tip_cap"
  [[ -n "$fee_cap" ]] || fee_cap="$(( base_fee + priority_fee ))"

  case "$tx_type" in
    legacy)     cast_args+=(--legacy --gas-price "$fee_cap") ;;
    accesslist) cast_args+=(--access-list --priority-gas-price "$priority_fee" --gas-price "$fee_cap") ;;
    dynamic|*)  cast_args+=(--priority-gas-price "$priority_fee" --gas-price "$fee_cap") ;;
  esac

  local result
  if [[ -n "$data" ]]; then
    [[ "$data" != 0x* ]] && data="0x$data"
    if [[ -z "$to" ]]; then
      result=$(cast send "${cast_args[@]}" --value "$value" --create "$data" 2>&1) || { printf 'TX_ERROR:%s\n' "$result" >&2; return 1; }
    else
      result=$(cast send "${cast_args[@]}" --value "$value" "$to" "$data" 2>&1) || { printf 'TX_ERROR:%s\n' "$result" >&2; return 1; }
    fi
  else
    result=$(cast send "${cast_args[@]}" --value "$value" "$to" 2>&1) || { printf 'TX_ERROR:%s\n' "$result" >&2; return 1; }
  fi
  local hash; hash=$(printf '%s\n' "$result" | grep -oE '0x[0-9a-fA-F]{64}' | head -1)
  printf '[INFO]  txHash=%s\n' "${hash:-$result}" >&2
  printf '%s\n' "${hash:-$result}"
}

_cb_send_raw_tx_python() {
  local target="$1" pk="$2" to="$3" value="$4"
  local data="${5:-}" gas_limit="${6:-21000}" tx_type="${7:-dynamic}"
  local tip_cap="${8:-}" fee_cap="${9:-}"

  local url; url=$(get_node_url "$target") || return 1
  local chain_id; chain_id=$(rpc "$target" "eth_chainId" "[]" | json_get - result); chain_id=$(hex_to_dec "$chain_id")

  python3 <<PYEOF
import json, sys
from eth_account import Account
import requests

pk="${pk}"; to="${to}"; value=int("${value}"); data="${data}"
gas_limit=int("${gas_limit}"); tx_type="${tx_type}"; chain_id=int("${chain_id}")
url="${url}"
tip_override="${tip_cap}"; fee_override="${fee_cap}"

acct=Account.from_key(pk)
def call(m,p):
    return requests.post(url, json={"jsonrpc":"2.0","method":m,"params":p,"id":1}).json()
nonce=int(call("eth_getTransactionCount",[acct.address,"pending"])["result"],16)
base_fee=int(call("eth_getBlockByNumber",["latest",False])["result"].get("baseFeePerGas","0x0"),16)
max_priority=int(tip_override) if tip_override else 27600000000000
max_fee=int(fee_override) if fee_override else base_fee + max_priority*2

tx={"nonce":nonce,"value":value,"gas":gas_limit,"chainId":chain_id}
if to:
    tx["to"]=to
if data:
    tx["data"]=data if data.startswith("0x") else "0x"+data
if tx_type=="legacy":
    tx["gasPrice"]=max_fee
elif tx_type=="accesslist":
    tx["gasPrice"]=max_fee; tx["accessList"]=[]; tx["type"]=1
else:
    tx["maxFeePerGas"]=max_fee; tx["maxPriorityFeePerGas"]=max_priority; tx["type"]=2

signed=acct.sign_transaction(tx)
resp=call("eth_sendRawTransaction",[signed.raw_transaction.to_0x_hex()])
if "error" in resp:
    print("TX_ERROR:"+resp["error"].get("message","unknown"), file=sys.stderr); sys.exit(1)
print(resp["result"])
PYEOF
}
