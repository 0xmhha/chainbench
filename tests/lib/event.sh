#!/usr/bin/env bash
# tests/lib/event.sh — Event log query and parsing utilities (cast-based)
# Source this: source tests/lib/event.sh
#
# Provides event log querying via cast, topic hash computation,
# receipt log filtering, and log decoding.

[[ -n "${_CB_EVENT_SH_LOADED:-}" ]] && return 0
readonly _CB_EVENT_SH_LOADED=1

_CB_EVENT_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${_CB_EVENT_LIB_DIR}/contract.sh"

# ---------------------------------------------------------------------------
# cb_get_logs <target> <address> <topic0> [from_block] [to_block]
# Query event logs via cast logs (eth_getLogs).
# Returns: JSON array of log objects.
# Default from_block: "0x0", to_block: "latest"
# ---------------------------------------------------------------------------
cb_get_logs() {
  local target="${1:?cb_get_logs: target required}"
  local address="${2:?cb_get_logs: address required}"
  local topic0="${3:?cb_get_logs: topic0 required}"
  local from_block="${4:-0x0}"
  local to_block="${5:-latest}"

  local url
  url="$(_cb_resolve_rpc_url "$target")" || return 1

  local result
  result=$("$_CB_CAST_BIN" logs \
    --rpc-url "$url" \
    --address "$address" \
    --from-block "$from_block" \
    --to-block "$to_block" \
    "$topic0" \
    --json 2>/dev/null) || {
    echo "ERROR: cb_get_logs failed (target=$target, address=$address, topic0=$topic0)" >&2
    return 1
  }

  printf '%s' "$result"
}

# ---------------------------------------------------------------------------
# cb_topic_hash <event_signature>
# Compute the keccak256 topic hash for an event signature.
# Example: cb_topic_hash "Transfer(address,address,uint256)"
# Output: 0xddf252ad...
# ---------------------------------------------------------------------------
cb_topic_hash() {
  local sig="${1:?cb_topic_hash: event signature required}"

  local hash
  hash=$("$_CB_CAST_BIN" sig-event "$sig" 2>/dev/null) || {
    # Fallback: cast keccak for older versions
    hash=$("$_CB_CAST_BIN" keccak "$sig" 2>/dev/null) || {
      echo "ERROR: cb_topic_hash failed for sig='$sig'" >&2
      return 1
    }
  }

  printf '%s' "$hash"
}

# ---------------------------------------------------------------------------
# cb_find_log <receipt_json> <address> <topic0>
# Find the first log in a receipt matching address (case-insensitive) and topic0.
# Input: full receipt JSON (from cast receipt --json or eth_getTransactionReceipt)
# Output: matching log object as JSON, or empty string if not found.
# Pure JSON parsing — no RPC calls.
# ---------------------------------------------------------------------------
cb_find_log() {
  local receipt_json="${1:?cb_find_log: receipt_json required}"
  local address="${2:?cb_find_log: address required}"
  local topic0="${3:?cb_find_log: topic0 required}"

  printf '%s' "$receipt_json" | python3 -c "
import sys, json

try:
    data = json.load(sys.stdin)
except Exception:
    sys.exit(0)

# Support both receipt JSON structures:
# - top-level logs array (from cast receipt --json)
# - {result: {logs: [...]}} (from eth_getTransactionReceipt via rpc())
logs = data.get('logs') or data.get('result', {}).get('logs', [])
if not isinstance(logs, list):
    sys.exit(0)

addr_lower = sys.argv[1].lower()
topic0_lower = sys.argv[2].lower()

for log in logs:
    log_addr = log.get('address', '').lower()
    topics = log.get('topics', [])
    if log_addr == addr_lower and topics and topics[0].lower() == topic0_lower:
        print(json.dumps(log))
        sys.exit(0)
" "$address" "$topic0"
}

# ---------------------------------------------------------------------------
# cb_count_logs <receipt_json> <address>
# Count logs in a receipt from a specific address (case-insensitive).
# Pure JSON parsing — no RPC calls.
# ---------------------------------------------------------------------------
cb_count_logs() {
  local receipt_json="${1:?cb_count_logs: receipt_json required}"
  local address="${2:?cb_count_logs: address required}"

  printf '%s' "$receipt_json" | python3 -c "
import sys, json

try:
    data = json.load(sys.stdin)
except Exception:
    print(0)
    sys.exit(0)

logs = data.get('logs') or data.get('result', {}).get('logs', [])
if not isinstance(logs, list):
    print(0)
    sys.exit(0)

addr_lower = sys.argv[1].lower()
count = sum(1 for log in logs if log.get('address', '').lower() == addr_lower)
print(count)
" "$address"
}

# ---------------------------------------------------------------------------
# cb_decode_log <event_signature> <log_json>
# Decode a log's topics (indexed) and data (non-indexed) per the event ABI.
# Example: cb_decode_log "Transfer(address indexed,address indexed,uint256)" "$log"
# Output: decoded fields as name=value lines (one per parameter).
# ---------------------------------------------------------------------------
cb_decode_log() {
  local event_sig="${1:?cb_decode_log: event signature required}"
  local log_json="${2:?cb_decode_log: log_json required}"

  printf '%s' "$log_json" | python3 - "$event_sig" "$_CB_CAST_BIN" <<'PYEOF'
import sys, json, re, subprocess

event_sig, cast_bin = sys.argv[1], sys.argv[2]
log = json.load(sys.stdin)
topics, raw_data = log.get('topics', []), log.get('data', '0x')

m = re.match(r'\w+\((.+)\)', event_sig)
if not m:
    print(f'ERROR: invalid event signature: {event_sig}', file=sys.stderr); sys.exit(1)

# Parse params, respecting nested parens
params, depth, buf = [], 0, ''
for ch in m.group(1):
    if ch == ',' and depth == 0:
        params.append(buf.strip()); buf = ''
    else:
        depth += (ch == '(') - (ch == ')'); buf += ch
if buf.strip():
    params.append(buf.strip())

param_info, nonindexed_types = [], []
for i, p in enumerate(params):
    parts = p.split()
    is_idx = 'indexed' in parts
    type_parts = [x for x in parts if x != 'indexed']
    ptype = type_parts[0] if type_parts else 'bytes32'
    pname = type_parts[-1] if len(type_parts) >= 2 else f'arg{i}'
    param_info.append((pname, ptype, is_idx))
    if not is_idx:
        nonindexed_types.append(ptype)

# Decode indexed params from topics[1..]
tidx = 1
for pname, ptype, is_idx in param_info:
    if not is_idx:
        continue
    if tidx >= len(topics):
        print(f'{pname}=<missing topic>'); tidx += 1; continue
    tb = bytes.fromhex(topics[tidx][2:] if topics[tidx].startswith('0x') else topics[tidx])
    if ptype == 'address':
        val = '0x' + tb[-20:].hex()
    elif ptype.startswith(('uint', 'int')):
        val = str(int.from_bytes(tb, 'big'))
    elif ptype == 'bool':
        val = 'true' if int.from_bytes(tb, 'big') else 'false'
    else:
        val = topics[tidx]
    print(f'{pname}={val}'); tidx += 1

# Decode non-indexed params via cast abi-decode
if nonindexed_types and raw_data and raw_data != '0x':
    try:
        r = subprocess.run(
            [cast_bin, 'abi-decode', '--', f'f()({",".join(nonindexed_types)})', raw_data],
            capture_output=True, text=True, timeout=10)
        lines = r.stdout.strip().splitlines() if r.returncode == 0 else []
        ni = 0
        for pname, ptype, is_idx in param_info:
            if is_idx:
                continue
            print(f'{pname}={lines[ni].strip() if ni < len(lines) else raw_data}'); ni += 1
    except Exception as e:
        for pname, ptype, is_idx in param_info:
            if not is_idx:
                print(f'{pname}=<error:{e}>')
PYEOF
}

# ---------------------------------------------------------------------------
# cb_get_receipt_logs <target> <tx_hash>
# Get all logs from a transaction receipt.
# Uses: cast receipt <tx_hash> --rpc-url <url> --json
# Returns: JSON array of logs.
# ---------------------------------------------------------------------------
cb_get_receipt_logs() {
  local target="${1:?cb_get_receipt_logs: target required}"
  local tx_hash="${2:?cb_get_receipt_logs: tx_hash required}"

  local url
  url="$(_cb_resolve_rpc_url "$target")" || return 1

  local receipt_json
  receipt_json=$("$_CB_CAST_BIN" receipt \
    --rpc-url "$url" \
    "$tx_hash" \
    --json 2>/dev/null) || {
    echo "ERROR: cb_get_receipt_logs failed (target=$target, tx=$tx_hash)" >&2
    return 1
  }

  printf '%s' "$receipt_json" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
except Exception:
    print('[]')
    sys.exit(0)

logs = data.get('logs', [])
print(json.dumps(logs))
"
}
