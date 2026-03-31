#!/usr/bin/env bash
# tests/lib/rpc.sh - RPC utility library for chainbench tests
# Usage: source tests/lib/rpc.sh
#
# Supports two modes:
#   - Local:  rpc <node_number> <method> [params]   (reads port from pids.json)
#   - Remote: rpc @<alias> <method> [params]         (resolves URL from remotes.json)
#
# When CHAINBENCH_REMOTE is set, functions default to '@<CHAINBENCH_REMOTE>'
# instead of local node '1'.

CHAINBENCH_DIR="${CHAINBENCH_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"

# RPC timeout: longer for remote (network latency), shorter for local
_CB_RPC_TIMEOUT_LOCAL="${CHAINBENCH_RPC_TIMEOUT:-3}"
_CB_RPC_TIMEOUT_REMOTE="${CHAINBENCH_RPC_TIMEOUT:-10}"

# ---------------------------------------------------------------------------
# Remote state helpers (loaded lazily)
# ---------------------------------------------------------------------------

_cb_rpc_remote_loaded=0

_cb_rpc_ensure_remote_state() {
  if [[ "$_cb_rpc_remote_loaded" == "0" ]]; then
    if [[ -f "${CHAINBENCH_DIR}/lib/remote_state.sh" ]]; then
      source "${CHAINBENCH_DIR}/lib/common.sh" 2>/dev/null || true
      source "${CHAINBENCH_DIR}/lib/remote_state.sh" 2>/dev/null || true
      _cb_rpc_remote_loaded=1
    fi
  fi
}

# ---------------------------------------------------------------------------
# Internal helpers
# ---------------------------------------------------------------------------

_load_state() {
  local state_file="${CHAINBENCH_DIR}/state/pids.json"
  if [[ ! -f "$state_file" ]]; then
    echo "ERROR: Chain is not running. Run 'chainbench start' first." >&2
    return 1
  fi
}

# _get_http_port <node_id>
# Prints the HTTP RPC port for the given node.
_get_http_port() {
  local node="${1:?_get_http_port: node id required}"
  python3 -c "
import json, sys
with open('${CHAINBENCH_DIR}/state/pids.json') as f:
    d = json.load(f)
node = '${node}'
if node not in d.get('nodes', {}):
    print('ERROR: node not found: ' + node, file=sys.stderr)
    sys.exit(1)
print(d['nodes'][node]['http_port'])
"
}

# _cb_rpc_default_target
# Returns the default RPC target: '@<CHAINBENCH_REMOTE>' if set, else '1'.
_cb_rpc_default_target() {
  if [[ -n "${CHAINBENCH_REMOTE:-}" ]]; then
    printf '@%s' "$CHAINBENCH_REMOTE"
  else
    printf '1'
  fi
}

# _cb_rpc_is_remote <target>
# Returns 0 if target starts with '@', 1 otherwise.
_cb_rpc_is_remote() {
  [[ "$1" == @* ]]
}

# _cb_rpc_resolve_remote_url <alias>
# Resolves remote alias to RPC URL. Prints URL on stdout.
_cb_rpc_resolve_remote_url() {
  local alias="$1"
  _cb_rpc_ensure_remote_state
  _cb_remote_get_url "$alias"
}

# _cb_rpc_resolve_remote_auth <alias>
# Resolves remote alias to auth header. Prints header on stdout (empty if none).
_cb_rpc_resolve_remote_auth() {
  local alias="$1"
  _cb_rpc_ensure_remote_state
  _cb_remote_get_auth_header "$alias" 2>/dev/null || echo ""
}

# ---------------------------------------------------------------------------
# Core RPC
# ---------------------------------------------------------------------------

# rpc_url <url> <method> [params] [auth_header]
# Makes a raw JSON-RPC call to an arbitrary URL. Prints the full response JSON.
rpc_url() {
  local url="${1:?rpc_url: url required}"
  local method="${2:?rpc_url: method required}"
  local params="${3:-[]}"
  local auth_header="${4:-}"

  local -a curl_args=(
    -s --max-time "$_CB_RPC_TIMEOUT_REMOTE"
    -X POST
    -H "Content-Type: application/json"
    --data "{\"jsonrpc\":\"2.0\",\"method\":\"${method}\",\"params\":${params},\"id\":1}"
  )

  if [[ -n "$auth_header" ]]; then
    curl_args+=(-H "Authorization: ${auth_header}")
  fi

  curl_args+=("$url")
  curl "${curl_args[@]}"
}

# rpc <target> <method> [params]
# Makes a raw JSON-RPC call and prints the full response JSON.
#
# target:
#   - Numeric (e.g. "1", "3"): local node — resolves port from pids.json
#   - "@alias" (e.g. "@eth-main"): remote — resolves URL from remotes.json
rpc() {
  local target="${1:?rpc: target required}" method="${2:?rpc: method required}" params="${3:-[]}"

  if _cb_rpc_is_remote "$target"; then
    # Remote mode: resolve alias to URL
    local alias="${target#@}"
    local url auth_header
    url="$(_cb_rpc_resolve_remote_url "$alias")" || {
      echo '{"jsonrpc":"2.0","error":{"code":-1,"message":"Failed to resolve remote alias"},"id":1}'
      return 1
    }
    auth_header="$(_cb_rpc_resolve_remote_auth "$alias")"
    rpc_url "$url" "$method" "$params" "$auth_header"
  else
    # Local mode: original behavior
    local port
    port=$(_get_http_port "$target") || return 1
    curl -s --max-time "$_CB_RPC_TIMEOUT_LOCAL" -X POST "http://127.0.0.1:${port}" \
      -H "Content-Type: application/json" \
      -d "{\"jsonrpc\":\"2.0\",\"method\":\"${method}\",\"params\":${params},\"id\":1}"
  fi
}

# ---------------------------------------------------------------------------
# Block / chain queries
# ---------------------------------------------------------------------------

# block_number [target]
# Prints current block number as a decimal integer.
block_number() {
  local target="${1:-$(_cb_rpc_default_target)}"
  rpc "$target" "eth_blockNumber" "[]" \
    | python3 -c "import sys,json; print(int(json.load(sys.stdin)['result'], 16))"
}

# get_block <target> <number>
# Prints the full block object JSON for the given decimal block number.
get_block() {
  local target="${1:?get_block: target required}" number="${2:?get_block: number required}"
  local hex_num
  hex_num=$(printf '0x%x' "$number")
  rpc "$target" "eth_getBlockByNumber" "[\"${hex_num}\", false]"
}

# get_block_miner <target> <number>
# Prints the miner address of the block at the given decimal number.
get_block_miner() {
  local target="${1:?get_block_miner: target required}" number="${2:?get_block_miner: number required}"
  local hex_num
  hex_num=$(printf '0x%x' "$number")
  rpc "$target" "eth_getBlockByNumber" "[\"${hex_num}\", false]" \
    | python3 -c "import sys,json; print(json.load(sys.stdin)['result']['miner'])"
}

# ---------------------------------------------------------------------------
# Network / peer queries
# ---------------------------------------------------------------------------

# peer_count [target]
# Prints connected peer count as a decimal integer.
peer_count() {
  local target="${1:-$(_cb_rpc_default_target)}"
  rpc "$target" "net_peerCount" "[]" \
    | python3 -c "import sys,json; print(int(json.load(sys.stdin)['result'], 16))"
}

# check_sync
# Prints a JSON object: {min, max, diff, synced}.
# For remote targets (CHAINBENCH_REMOTE set), queries single endpoint.
check_sync() {
  if [[ -n "${CHAINBENCH_REMOTE:-}" ]]; then
    # Remote mode: single endpoint, always "synced" from our perspective
    local block
    block=$(block_number "@${CHAINBENCH_REMOTE}" 2>/dev/null || echo "0")
    printf '{"min":%d,"max":%d,"diff":0,"synced":true}\n' "$block" "$block"
    return 0
  fi

  # Local mode: original behavior
  local state_file="${CHAINBENCH_DIR}/state/pids.json"
  python3 -c "
import json, urllib.request, sys

with open('${state_file}') as f:
    state = json.load(f)

blocks = []
for nid, info in state.get('nodes', {}).items():
    if info.get('status') != 'running':
        continue
    port = info['http_port']
    try:
        req = urllib.request.Request(
            f'http://127.0.0.1:{port}',
            data=json.dumps({'jsonrpc':'2.0','method':'eth_blockNumber','params':[],'id':1}).encode(),
            headers={'Content-Type': 'application/json'}
        )
        resp = urllib.request.urlopen(req, timeout=5)
        data = json.loads(resp.read())
        blocks.append(int(data['result'], 16))
    except Exception:
        pass

if not blocks:
    print(json.dumps({'min':0,'max':0,'diff':0,'synced':False,'error':'no_running_nodes'}))
else:
    mn, mx = min(blocks), max(blocks)
    print(json.dumps({'min':mn,'max':mx,'diff':mx-mn,'synced':(mx-mn)<=2}))
"
}

# ---------------------------------------------------------------------------
# Account / wallet
# ---------------------------------------------------------------------------

# get_coinbase [target]
# Prints the coinbase (etherbase) address of the node.
get_coinbase() {
  local target="${1:-$(_cb_rpc_default_target)}"
  rpc "$target" "eth_coinbase" "[]" \
    | python3 -c "import sys,json; print(json.load(sys.stdin)['result'])"
}

# unlock_account <target> <address> [password] [duration_secs]
# Unlocks the given account. Suppresses output.
unlock_account() {
  local target="${1:?unlock_account: target required}"
  local address="${2:?unlock_account: address required}"
  local password="${3:-1}"
  local duration="${4:-600}"
  rpc "$target" "personal_unlockAccount" \
    "[\"${address}\",\"${password}\",${duration}]" > /dev/null 2>&1
}

# get_balance <target> <address>
# Prints the balance of address in wei (decimal integer).
get_balance() {
  local target="${1:?get_balance: target required}" address="${2:?get_balance: address required}"
  rpc "$target" "eth_getBalance" "[\"${address}\",\"latest\"]" \
    | python3 -c "import sys,json; print(int(json.load(sys.stdin)['result'], 16))"
}

# ---------------------------------------------------------------------------
# Transactions
# ---------------------------------------------------------------------------

# send_tx <target> <from> <to> [value]
# Prints the tx hash on success, or TX_ERROR:<message> on failure.
send_tx() {
  local target="${1:?send_tx: target required}"
  local from="${2:?send_tx: from required}"
  local to="${3:?send_tx: to required}"
  local value="${4:-0x1}"
  rpc "$target" "eth_sendTransaction" \
    "[{\"from\":\"${from}\",\"to\":\"${to}\",\"value\":\"${value}\"}]" \
    | python3 -c "
import sys, json
d = json.load(sys.stdin)
if 'result' in d:
    print(d['result'])
elif 'error' in d:
    print('TX_ERROR:' + d['error'].get('message', 'unknown'))
else:
    print('TX_ERROR:unknown')
"
}

# wait_receipt <target> <tx_hash> [timeout_secs]
# Polls until a receipt is available. Prints: success | failed | timeout.
# Returns 0 on success/failed, 1 on timeout.
wait_receipt() {
  local target="${1:?wait_receipt: target required}"
  local tx_hash="${2:?wait_receipt: tx_hash required}"
  local timeout="${3:-30}"

  local i
  for i in $(seq 1 "$timeout"); do
    local result status
    result=$(rpc "$target" "eth_getTransactionReceipt" "[\"${tx_hash}\"]")
    status=$(echo "$result" | python3 -c "
import sys, json
d = json.load(sys.stdin)
r = d.get('result')
if r is None:
    print('pending')
else:
    print('success' if r.get('status') == '0x1' else 'failed')
" 2>/dev/null || echo "pending")

    if [[ "$status" != "pending" ]]; then
      echo "$status"
      return 0
    fi
    sleep 1
  done

  echo "timeout"
  return 1
}

# ---------------------------------------------------------------------------
# Transaction pool
# ---------------------------------------------------------------------------

# txpool_status [target]
# Prints JSON: {pending: N, queued: N}.
txpool_status() {
  local target="${1:-$(_cb_rpc_default_target)}"
  rpc "$target" "txpool_status" "[]" \
    | python3 -c "
import sys, json
d = json.load(sys.stdin)['result']
pending = int(d['pending'], 16)
queued  = int(d['queued'],  16)
print(json.dumps({'pending': pending, 'queued': queued}))
"
}

# ---------------------------------------------------------------------------
# Wait helpers (block-level)
# ---------------------------------------------------------------------------

# wait_for_block <target> <target_block> [timeout_secs]
# Polls until the node's block number >= target. Prints the reached block number.
# Returns 0 on success, 1 on timeout.
wait_for_block() {
  local target="${1:?wait_for_block: target required}"
  local target_block="${2:?wait_for_block: target_block required}"
  local timeout="${3:-60}"

  local i
  for i in $(seq 1 "$timeout"); do
    local current
    current=$(block_number "$target" 2>/dev/null || echo "0")
    if [[ "$current" -ge "$target_block" ]]; then
      echo "$current"
      return 0
    fi
    sleep 1
  done

  echo "timeout"
  return 1
}

# ---------------------------------------------------------------------------
# Chain ID query (works for both local and remote)
# ---------------------------------------------------------------------------

# get_chain_id [target]
# Prints the chain ID as a decimal integer.
get_chain_id() {
  local target="${1:-$(_cb_rpc_default_target)}"
  rpc "$target" "eth_chainId" "[]" \
    | python3 -c "import sys,json; print(int(json.load(sys.stdin)['result'], 16))"
}

# get_client_version [target]
# Prints the client version string.
get_client_version() {
  local target="${1:-$(_cb_rpc_default_target)}"
  rpc "$target" "web3_clientVersion" "[]" \
    | python3 -c "import sys,json; print(json.load(sys.stdin)['result'])"
}

# get_gas_price [target]
# Prints the gas price in wei (decimal).
get_gas_price() {
  local target="${1:-$(_cb_rpc_default_target)}"
  rpc "$target" "eth_gasPrice" "[]" \
    | python3 -c "import sys,json; print(int(json.load(sys.stdin)['result'], 16))"
}

# get_syncing [target]
# Prints "false" if not syncing, or JSON sync object if syncing.
get_syncing() {
  local target="${1:-$(_cb_rpc_default_target)}"
  rpc "$target" "eth_syncing" "[]" \
    | python3 -c "
import sys, json
d = json.load(sys.stdin)
r = d.get('result', False)
if r is False or r == False:
    print('false')
else:
    print(json.dumps(r))
"
}
