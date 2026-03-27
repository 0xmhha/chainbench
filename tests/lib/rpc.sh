#!/usr/bin/env bash
# tests/lib/rpc.sh - RPC utility library for chainbench tests
# Usage: source tests/lib/rpc.sh

CHAINBENCH_DIR="${CHAINBENCH_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"

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

# ---------------------------------------------------------------------------
# Core RPC
# ---------------------------------------------------------------------------

# rpc <node> <method> [params]
# Makes a raw JSON-RPC call and prints the full response JSON.
rpc() {
  local node="${1:?rpc: node required}" method="${2:?rpc: method required}" params="${3:-[]}"
  local port
  port=$(_get_http_port "$node") || return 1
  curl -s -X POST "http://127.0.0.1:${port}" \
    -H "Content-Type: application/json" \
    -d "{\"jsonrpc\":\"2.0\",\"method\":\"${method}\",\"params\":${params},\"id\":1}"
}

# ---------------------------------------------------------------------------
# Block / chain queries
# ---------------------------------------------------------------------------

# block_number [node]
# Prints current block number as a decimal integer.
block_number() {
  local node="${1:-1}"
  rpc "$node" "eth_blockNumber" "[]" \
    | python3 -c "import sys,json; print(int(json.load(sys.stdin)['result'], 16))"
}

# get_block <node> <number>
# Prints the full block object JSON for the given decimal block number.
get_block() {
  local node="${1:?get_block: node required}" number="${2:?get_block: number required}"
  local hex_num
  hex_num=$(printf '0x%x' "$number")
  rpc "$node" "eth_getBlockByNumber" "[\"${hex_num}\", false]"
}

# get_block_miner <node> <number>
# Prints the miner address of the block at the given decimal number.
get_block_miner() {
  local node="${1:?get_block_miner: node required}" number="${2:?get_block_miner: number required}"
  local hex_num
  hex_num=$(printf '0x%x' "$number")
  rpc "$node" "eth_getBlockByNumber" "[\"${hex_num}\", false]" \
    | python3 -c "import sys,json; print(json.load(sys.stdin)['result']['miner'])"
}

# ---------------------------------------------------------------------------
# Network / peer queries
# ---------------------------------------------------------------------------

# peer_count [node]
# Prints connected peer count as a decimal integer.
peer_count() {
  local node="${1:-1}"
  rpc "$node" "net_peerCount" "[]" \
    | python3 -c "import sys,json; print(int(json.load(sys.stdin)['result'], 16))"
}

# check_sync
# Prints a JSON object: {min, max, diff, synced}.
# Queries all nodes whose status is "running" in pids.json.
check_sync() {
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

# get_coinbase [node]
# Prints the coinbase (etherbase) address of the node.
get_coinbase() {
  local node="${1:-1}"
  rpc "$node" "eth_coinbase" "[]" \
    | python3 -c "import sys,json; print(json.load(sys.stdin)['result'])"
}

# unlock_account <node> <address> [password] [duration_secs]
# Unlocks the given account. Suppresses output.
unlock_account() {
  local node="${1:?unlock_account: node required}"
  local address="${2:?unlock_account: address required}"
  local password="${3:-1}"
  local duration="${4:-600}"
  rpc "$node" "personal_unlockAccount" \
    "[\"${address}\",\"${password}\",${duration}]" > /dev/null 2>&1
}

# get_balance <node> <address>
# Prints the balance of address in wei (decimal integer).
get_balance() {
  local node="${1:?get_balance: node required}" address="${2:?get_balance: address required}"
  rpc "$node" "eth_getBalance" "[\"${address}\",\"latest\"]" \
    | python3 -c "import sys,json; print(int(json.load(sys.stdin)['result'], 16))"
}

# ---------------------------------------------------------------------------
# Transactions
# ---------------------------------------------------------------------------

# send_tx <node> <from> <to> [value]
# Prints the tx hash on success, or TX_ERROR:<message> on failure.
send_tx() {
  local node="${1:?send_tx: node required}"
  local from="${2:?send_tx: from required}"
  local to="${3:?send_tx: to required}"
  local value="${4:-0x1}"
  rpc "$node" "eth_sendTransaction" \
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

# wait_receipt <node> <tx_hash> [timeout_secs]
# Polls until a receipt is available. Prints: success | failed | timeout.
# Returns 0 on success/failed, 1 on timeout.
wait_receipt() {
  local node="${1:?wait_receipt: node required}"
  local tx_hash="${2:?wait_receipt: tx_hash required}"
  local timeout="${3:-30}"

  local i
  for i in $(seq 1 "$timeout"); do
    local result status
    result=$(rpc "$node" "eth_getTransactionReceipt" "[\"${tx_hash}\"]")
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

# txpool_status [node]
# Prints JSON: {pending: N, queued: N}.
txpool_status() {
  local node="${1:-1}"
  rpc "$node" "txpool_status" "[]" \
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

# wait_for_block <node> <target_block> [timeout_secs]
# Polls until the node's block number >= target. Prints the reached block number.
# Returns 0 on success, 1 on timeout.
wait_for_block() {
  local node="${1:?wait_for_block: node required}"
  local target="${2:?wait_for_block: target required}"
  local timeout="${3:-60}"

  local i
  for i in $(seq 1 "$timeout"); do
    local current
    current=$(block_number "$node" 2>/dev/null || echo "0")
    if [[ "$current" -ge "$target" ]]; then
      echo "$current"
      return 0
    fi
    sleep 1
  done

  echo "timeout"
  return 1
}
