#!/usr/bin/env bash
# tests/lib/tx_builder.sh — Unified transaction builder (cast + Python)
#
# 3-tier tx construction:
#   Tier 1 (cast):   Standard tx types (legacy, EIP-1559, EIP-2930)
#   Tier 2 (Python): Custom go-stablenet types (Fee Delegation 0x16)
#   Tier 3 (Python): Intentionally malformed transactions for negative tests
#
# All functions auto-detect gas prices from node state to avoid tx errors.

[[ -n "${_CB_TX_BUILDER_LOADED:-}" ]] && return 0
readonly _CB_TX_BUILDER_LOADED=1

_CB_TX_BUILDER_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${_CB_TX_BUILDER_LIB_DIR}/contract.sh"

# Default gas tip for go-stablenet Anzeon policy: 27.6 Gwei
readonly CB_DEFAULT_GAS_TIP="27600000000000"

# ===========================================================================
# Internal helpers
# ===========================================================================

_cb_fee_delegate_py() {
  local candidates=(
    "${CHAINBENCH_DIR:-}/tests/regression/lib/fee_delegate.py"
    "${_CB_TX_BUILDER_LIB_DIR}/../regression/lib/fee_delegate.py"
  )
  for c in "${candidates[@]}"; do
    [[ -f "$c" ]] && { printf '%s' "$c"; return 0; }
  done
  echo "ERROR: fee_delegate.py not found" >&2
  return 1
}

# _cb_query_gas_prices <rpc_url> [tip]
# Prints: <max_fee_per_gas> <max_priority_fee_per_gas> <base_fee>
_cb_query_gas_prices() {
  local url="${1:?}" tip="${2:-$CB_DEFAULT_GAS_TIP}"
  local base_fee
  base_fee=$(python3 -c "
import json, urllib.request
req = urllib.request.Request('${url}', method='POST',
    headers={'Content-Type': 'application/json'},
    data=json.dumps({'jsonrpc':'2.0','method':'eth_getBlockByNumber','params':['latest',False],'id':1}).encode())
print(int(json.loads(urllib.request.urlopen(req, timeout=5).read())['result']['baseFeePerGas'], 16))
" 2>/dev/null) || { echo "ERROR: failed to query baseFee" >&2; return 1; }
  printf '%s %s %s' "$(( base_fee * 2 + tip ))" "$tip" "$base_fee"
}

# _cb_extract_tx_hash <json_output> <caller_name>
# Extract transactionHash from cast JSON output. Prints hash or returns 1.
_cb_extract_tx_hash() {
  local output="$1" caller="$2"
  local tx_hash
  tx_hash=$(printf '%s' "$output" | python3 -c "import sys,json; print(json.load(sys.stdin).get('transactionHash',''))" 2>/dev/null) \
    || tx_hash=$(printf '%s' "$output" | grep -o '"transactionHash":"0x[0-9a-fA-F]*"' | head -1 | sed 's/.*:"//;s/"//')
  if [[ -z "$tx_hash" ]]; then
    echo "ERROR: ${caller}: could not extract tx hash" >&2; return 1
  fi
  printf '%s' "$tx_hash"
}

# ===========================================================================
# Tier 1: Standard TX (cast-based)
# ===========================================================================

# cb_send_tx <target> <private_key> <to> <value_wei> [data] [gas_limit]
# Send a standard EIP-1559 transaction. Returns tx hash.
cb_send_tx() {
  local target="${1:?cb_send_tx: target required}" private_key="$2" to="$3" value="$4"
  local data="${5:-}" gas_limit="${6:-}"
  local url; url="$(_cb_resolve_rpc_url "$target")" || return 1

  local gas_info max_fee tip
  gas_info=$(_cb_query_gas_prices "$url") || return 1
  read -r max_fee tip _ <<< "$gas_info"

  local -a args=(--rpc-url "$url" --private-key "$private_key" "$to"
    --value "$value" --priority-gas-price "$tip" --gas-price "$max_fee")
  [[ -n "$data" ]] && args+=(--input "$data")
  [[ -n "$gas_limit" ]] && args+=(--gas-limit "$gas_limit")

  local output
  output=$("$_CB_CAST_BIN" send "${args[@]}" --json 2>&1) \
    || { echo "ERROR: cb_send_tx failed: $output" >&2; return 1; }
  _cb_extract_tx_hash "$output" "cb_send_tx"
}

# cb_send_legacy_tx <target> <private_key> <to> <value_wei> [data] [gas_limit]
# Send a legacy (type 0) transaction. Returns tx hash.
cb_send_legacy_tx() {
  local target="${1:?cb_send_legacy_tx: target required}" private_key="$2" to="$3" value="$4"
  local data="${5:-}" gas_limit="${6:-}"
  local url; url="$(_cb_resolve_rpc_url "$target")" || return 1

  local -a args=(--rpc-url "$url" --private-key "$private_key" --legacy "$to" --value "$value")
  [[ -n "$data" ]] && args+=(--input "$data")
  [[ -n "$gas_limit" ]] && args+=(--gas-limit "$gas_limit")

  local output
  output=$("$_CB_CAST_BIN" send "${args[@]}" --json 2>&1) \
    || { echo "ERROR: cb_send_legacy_tx failed: $output" >&2; return 1; }
  _cb_extract_tx_hash "$output" "cb_send_legacy_tx"
}

# cb_sign_tx <target> <private_key> <to> <value_wei> [data] [gas_limit]
# Sign but don't send. Returns raw tx hex (0x...).
cb_sign_tx() {
  local target="${1:?cb_sign_tx: target required}" private_key="$2" to="$3" value="$4"
  local data="${5:-}" gas_limit="${6:-}"
  local url; url="$(_cb_resolve_rpc_url "$target")" || return 1

  local gas_info max_fee tip
  gas_info=$(_cb_query_gas_prices "$url") || return 1
  read -r max_fee tip _ <<< "$gas_info"

  local -a args=(--rpc-url "$url" --private-key "$private_key" "$to"
    --value "$value" --priority-gas-price "$tip" --gas-price "$max_fee")
  [[ -n "$data" ]] && args+=(--input "$data")
  [[ -n "$gas_limit" ]] && args+=(--gas-limit "$gas_limit")

  local raw_tx
  raw_tx=$("$_CB_CAST_BIN" mktx "${args[@]}" 2>&1) \
    || { echo "ERROR: cb_sign_tx failed: $raw_tx" >&2; return 1; }
  printf '%s' "$raw_tx"
}

# cb_send_raw <target> <raw_tx_hex>
# Send a pre-signed raw transaction. Returns tx hash.
cb_send_raw() {
  local target="${1:?cb_send_raw: target required}" raw_tx="${2:?cb_send_raw: raw_tx required}"
  local url; url="$(_cb_resolve_rpc_url "$target")" || return 1

  local output
  output=$("$_CB_CAST_BIN" publish --rpc-url "$url" "$raw_tx" 2>&1) \
    || { echo "ERROR: cb_send_raw failed: $output" >&2; return 1; }

  local tx_hash
  tx_hash=$(printf '%s' "$output" | python3 -c "import sys,json; print(json.load(sys.stdin).get('transactionHash',''))" 2>/dev/null) \
    || tx_hash=$(printf '%s' "$output" | grep -oE '0x[0-9a-fA-F]{64}' | head -1)
  if [[ -z "$tx_hash" ]]; then
    echo "ERROR: cb_send_raw: could not extract tx hash from: $output" >&2; return 1
  fi
  printf '%s' "$tx_hash"
}

# cb_wait_receipt <target> <tx_hash> [timeout_secs]
# Wait for transaction receipt (default 30s). Returns full receipt JSON.
cb_wait_receipt() {
  local target="${1:?}" tx_hash="${2:?}" timeout="${3:-30}"
  local url; url="$(_cb_resolve_rpc_url "$target")" || return 1

  local i output
  for (( i=0; i<timeout; i++ )); do
    output=$("$_CB_CAST_BIN" receipt --rpc-url "$url" "$tx_hash" --json 2>/dev/null) && {
      [[ -n "$output" && "$output" != "null" ]] && { printf '%s' "$output"; return 0; }
    }
    sleep 1
  done
  echo "ERROR: cb_wait_receipt: timeout after ${timeout}s for tx $tx_hash" >&2
  return 1
}

# cb_get_receipt_status <target> <tx_hash>
# Returns "0x0" (failed) or "0x1" (success).
cb_get_receipt_status() {
  local target="${1:?}" tx_hash="${2:?}"
  local url; url="$(_cb_resolve_rpc_url "$target")" || return 1

  local status
  status=$("$_CB_CAST_BIN" receipt --rpc-url "$url" "$tx_hash" "status" 2>/dev/null) \
    || { echo "ERROR: cb_get_receipt_status failed for $tx_hash" >&2; return 1; }
  case "$status" in
    1|0x1|0x01) printf '0x1' ;;
    0|0x0|0x00) printf '0x0' ;;
    *)          printf '%s' "$status" ;;
  esac
}

# ===========================================================================
# Tier 2: Fee Delegation (0x16) via fee_delegate.py
# ===========================================================================

# _cb_fee_delegate_run <subcmd> <target> <sender_pk> <fee_payer_pk> <to> <value>
#                      [data] [gas] [tip] [tamper]
# Internal dispatcher for fee_delegate.py sign/send commands.
_cb_fee_delegate_run() {
  local subcmd="$1" target="$2" sender_pk="$3" fee_payer_pk="$4" to="$5" value="$6"
  local data="${7:-}" gas="${8:-21000}" tip="${9:-$CB_DEFAULT_GAS_TIP}" tamper="${10:-}"

  local url helper_py
  url="$(_cb_resolve_rpc_url "$target")" || return 1
  helper_py="$(_cb_fee_delegate_py)" || return 1

  local -a args=("$helper_py" "$subcmd" --rpc "$url"
    --sender-pk "$sender_pk" --fee-payer-pk "$fee_payer_pk"
    --to "$to" --value "$value" --gas "$gas" --tip "$tip")
  [[ -n "$data" ]] && args+=(--data "$data")
  [[ -n "$tamper" ]] && args+=(--tamper "$tamper")

  python3 "${args[@]}" 2>&1
}

# cb_send_fee_delegate <target> <sender_pk> <fee_payer_pk> <to> <value_wei> [data] [gas] [tip]
# Send a Fee Delegation (type 0x16) transaction.
# Returns: JSON with txHash, senderAddr, feePayerAddr, rawTx, etc.
cb_send_fee_delegate() {
  local output
  output=$(_cb_fee_delegate_run "send" "$@") || {
    echo "ERROR: cb_send_fee_delegate failed: $output" >&2; return 1
  }
  printf '%s' "$output"
}

# cb_sign_fee_delegate <target> <sender_pk> <fee_payer_pk> <to> <value_wei> [data] [gas] [tip]
# Sign but don't send Fee Delegation tx. Returns JSON with rawTx.
cb_sign_fee_delegate() {
  local output
  output=$(_cb_fee_delegate_run "sign" "$@") || {
    echo "ERROR: cb_sign_fee_delegate failed: $output" >&2; return 1
  }
  printf '%s' "$output"
}

# ===========================================================================
# Tier 3: Invalid TX construction (negative tests)
# ===========================================================================

# cb_send_invalid_fee_delegate <target> <sender_pk> <fee_payer_pk> <to> <value_wei> <tamper_mode>
# tamper_mode: "sender" | "feepayer" — tampers with the specified signature.
# Returns: JSON including rpcResponse (which should contain error).
cb_send_invalid_fee_delegate() {
  local target="$1" sender_pk="$2" fee_payer_pk="$3" to="$4" value="$5"
  local tamper_mode="${6:?cb_send_invalid_fee_delegate: tamper_mode required (sender|feepayer)}"
  local output
  output=$(_cb_fee_delegate_run "send" "$target" "$sender_pk" "$fee_payer_pk" \
    "$to" "$value" "" "21000" "$CB_DEFAULT_GAS_TIP" "$tamper_mode") || true
  printf '%s' "$output"
}

# cb_send_invalid_tx <target> <private_key> <to> <value_wei> <invalid_mode>
# Send intentionally invalid standard transactions for rejection testing.
#   nonce_too_low         — set nonce to 0 when account has higher nonce
#   gas_too_low           — set gas to 1
#   value_exceeds_balance — set value to max uint256
#   bad_signature         — tamper with signature bytes
# Returns: JSON with mode + rpcResponse (which should contain error).
cb_send_invalid_tx() {
  local target="${1:?}" private_key="${2:?}" to="${3:?}" value="${4:?}"
  local invalid_mode="${5:?cb_send_invalid_tx: invalid_mode required}"
  local url; url="$(_cb_resolve_rpc_url "$target")" || return 1

  local output
  output=$(python3 -c "
import json, sys, requests
from eth_account import Account

pk, to, url, mode = '${private_key}', '${to}', '${url}', '${invalid_mode}'
value, tip = int('${value}'), ${CB_DEFAULT_GAS_TIP}
acct = Account.from_key(pk)

chain_id = int(requests.post(url, json={'jsonrpc':'2.0','method':'eth_chainId','params':[],'id':1}).json()['result'], 16)
nonce = int(requests.post(url, json={'jsonrpc':'2.0','method':'eth_getTransactionCount','params':[acct.address,'pending'],'id':1}).json()['result'], 16)
base_fee = int(requests.post(url, json={'jsonrpc':'2.0','method':'eth_getBlockByNumber','params':['latest',False],'id':1}).json()['result']['baseFeePerGas'], 16)

tx = {'type':2, 'chainId':chain_id, 'nonce':nonce, 'maxPriorityFeePerGas':tip,
      'maxFeePerGas': base_fee*2+tip, 'gas':21000, 'to':to, 'value':value}

if mode == 'nonce_too_low':       tx['nonce'] = 0
elif mode == 'gas_too_low':       tx['gas'] = 1
elif mode == 'value_exceeds_balance': tx['value'] = (1 << 256) - 1
elif mode != 'bad_signature':
    print(json.dumps({'error': 'unknown invalid_mode: ' + mode})); sys.exit(0)

signed = acct.sign_transaction(tx)
raw_hex = signed.raw_transaction.hex()
if not raw_hex.startswith('0x'): raw_hex = '0x' + raw_hex

if mode == 'bad_signature':
    raw_bytes = bytearray(bytes.fromhex(raw_hex[2:]))
    if len(raw_bytes) > 10: raw_bytes[-10] ^= 0xFF
    raw_hex = '0x' + raw_bytes.hex()

resp = requests.post(url, json={'jsonrpc':'2.0','method':'eth_sendRawTransaction','params':[raw_hex],'id':1}).json()
result = {'mode': mode, 'rpcResponse': resp}
if 'result' in resp: result['txHash'] = resp['result']
print(json.dumps(result, indent=2))
" 2>&1) || true

  printf '%s' "$output"
}
