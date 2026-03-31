#!/usr/bin/env bash
# lib/cmd_status.sh - Show chain and node status
# Sourced by the main chainbench dispatcher; $@ contains subcommand arguments.

# Guard against double-sourcing
[[ -n "${_CB_CMD_STATUS_SH_LOADED:-}" ]] && return 0
readonly _CB_CMD_STATUS_SH_LOADED=1

_CB_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${_CB_LIB_DIR}/common.sh"
source "${_CB_LIB_DIR}/remote_state.sh" 2>/dev/null || true

# ---- Constants ---------------------------------------------------------------

readonly _CB_STATUS_PIDS_FILE="${CHAINBENCH_DIR}/state/pids.json"
readonly _CB_STATUS_CURRENT_PROFILE="${CHAINBENCH_DIR}/state/current-profile.yaml"
readonly _CB_STATUS_RPC_TIMEOUT=3
readonly _CB_STATUS_REMOTE_RPC_TIMEOUT=10

# ---- Argument parsing --------------------------------------------------------

_CB_STATUS_JSON_MODE=0
_CB_STATUS_REMOTE_ALIAS=""

_cb_status_parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --json) _CB_STATUS_JSON_MODE=1; shift ;;
      --remote) _CB_STATUS_REMOTE_ALIAS="${2:?--remote requires an alias}"; shift 2 ;;
      --remote=*) _CB_STATUS_REMOTE_ALIAS="${1#--remote=}"; shift ;;
      --help|-h)
        printf 'Usage: chainbench status [--json] [--remote <alias>]\n' >&2
        return 0
        ;;
      *)
        log_warn "Unknown status option: $1 (ignoring)"
        shift
        ;;
    esac
  done

  # Support CHAINBENCH_REMOTE env var as fallback
  if [[ -z "$_CB_STATUS_REMOTE_ALIAS" && -n "${CHAINBENCH_REMOTE:-}" ]]; then
    _CB_STATUS_REMOTE_ALIAS="$CHAINBENCH_REMOTE"
  fi
}

# ---- RPC helpers -------------------------------------------------------------

# _cb_status_rpc_call <port> <method> [params]
# Sends a JSON-RPC request to a LOCAL node and prints the raw result field on stdout.
# Returns 1 if curl fails or the response contains an error.
_cb_status_rpc_call() {
  local port="$1"
  local method="$2"
  local params="${3:-[]}"

  local response
  response="$(curl -s --max-time "$_CB_STATUS_RPC_TIMEOUT" \
    -X POST \
    -H "Content-Type: application/json" \
    --data "{\"jsonrpc\":\"2.0\",\"method\":\"${method}\",\"params\":${params},\"id\":1}" \
    "http://127.0.0.1:${port}" 2>/dev/null)" || return 1

  if [[ -z "$response" ]]; then
    return 1
  fi

  # Extract .result via python3
  python3 - "$response" <<'PYEOF'
import sys, json

try:
    data = json.loads(sys.argv[1])
    if "error" in data:
        sys.exit(1)
    result = data.get("result", "")
    print(result)
except Exception:
    sys.exit(1)
PYEOF
}

# _cb_status_remote_rpc_call <url> <method> [params] [auth_header]
# Sends a JSON-RPC request to a REMOTE URL and prints the raw result field.
_cb_status_remote_rpc_call() {
  local url="$1"
  local method="$2"
  local params="${3:-[]}"
  local auth_header="${4:-}"

  local -a curl_args=(
    -s --max-time "$_CB_STATUS_REMOTE_RPC_TIMEOUT"
    -X POST
    -H "Content-Type: application/json"
    --data "{\"jsonrpc\":\"2.0\",\"method\":\"${method}\",\"params\":${params},\"id\":1}"
  )
  if [[ -n "$auth_header" ]]; then
    curl_args+=(-H "Authorization: ${auth_header}")
  fi
  curl_args+=("$url")

  local response
  response="$(curl "${curl_args[@]}" 2>/dev/null)" || return 1

  if [[ -z "$response" ]]; then
    return 1
  fi

  python3 - "$response" <<'PYEOF'
import sys, json

try:
    data = json.loads(sys.argv[1])
    if "error" in data:
        sys.exit(1)
    result = data.get("result", "")
    if isinstance(result, bool):
        print(str(result).lower())
    elif isinstance(result, dict):
        print(json.dumps(result))
    else:
        print(result)
except Exception:
    sys.exit(1)
PYEOF
}

# _cb_status_hex_to_dec <hex_string>
# Converts a 0x-prefixed hex string to a decimal integer.
_cb_status_hex_to_dec() {
  python3 -c "print(int('${1}', 16))" 2>/dev/null || printf '0\n'
}

# ---- PID liveness check ------------------------------------------------------

# _cb_status_pid_alive <pid>
_cb_status_pid_alive() {
  kill -0 "$1" 2>/dev/null
}

# ---- pids.json update --------------------------------------------------------

# _cb_status_mark_node_dead <node_index>
# Updates pids.json: set the specified node's status to "stopped".
_cb_status_mark_node_dead() {
  local idx="$1"
  local stopped_at
  stopped_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

  python3 - "$_CB_STATUS_PIDS_FILE" "$idx" "$stopped_at" <<'PYEOF'
import sys, json

pids_file  = sys.argv[1]
idx        = int(sys.argv[2])
stopped_at = sys.argv[3]

with open(pids_file) as fh:
    data = json.load(fh)

nodes = data.get("nodes", {})
key = str(idx)
if isinstance(nodes, dict) and key in nodes:
    nodes[key]["status"]     = "stopped"
    nodes[key]["stopped_at"] = stopped_at
elif isinstance(nodes, list) and 0 <= idx < len(nodes):
    nodes[idx]["status"]     = "stopped"
    nodes[idx]["stopped_at"] = stopped_at

with open(pids_file, "w") as fh:
    json.dump(data, fh, indent=2)
    fh.write("\n")
PYEOF
}

# ---- Chain metadata ----------------------------------------------------------

# _cb_status_read_chain_name
# Reads the chain name from pids.json .chain_name field.
_cb_status_read_chain_name() {
  python3 - "$_CB_STATUS_PIDS_FILE" <<'PYEOF'
import sys, json

with open(sys.argv[1]) as fh:
    data = json.load(fh)

print(data.get("chain_id", data.get("chain_name", "unknown")))
PYEOF
}

# _cb_status_read_started_at
# Reads the chain start timestamp from pids.json .started_at field.
_cb_status_read_started_at() {
  python3 - "$_CB_STATUS_PIDS_FILE" <<'PYEOF'
import sys, json

with open(sys.argv[1]) as fh:
    data = json.load(fh)

print(data.get("started_at", ""))
PYEOF
}

# _cb_status_read_profile_name
# Reads the profile name: tries pids.json first, then current-profile.yaml.
_cb_status_read_profile_name() {
  python3 - "$_CB_STATUS_PIDS_FILE" "$_CB_STATUS_CURRENT_PROFILE" <<'PYEOF'
import sys, json, re, os

pids_file    = sys.argv[1]
profile_file = sys.argv[2]

# Try pids.json first
try:
    with open(pids_file) as fh:
        data = json.load(fh)
    name = data.get("profile", "")
    if name:
        print(name)
        sys.exit(0)
except Exception:
    pass

# Fall back to current-profile.yaml
if os.path.isfile(profile_file):
    with open(profile_file) as fh:
        for line in fh:
            m = re.match(r'^\s*profile\s*:\s*(\S+)', line)
            if m:
                print(m.group(1))
                sys.exit(0)

print("unknown")
PYEOF
}

# _cb_status_calc_uptime <started_at_iso>
# Returns a human-friendly uptime string like "5m 12s".
_cb_status_calc_uptime() {
  local started_at="$1"
  python3 - "$started_at" <<'PYEOF'
import sys
from datetime import datetime, timezone

started_str = sys.argv[1]
if not started_str:
    print("unknown")
    sys.exit(0)

try:
    started = datetime.fromisoformat(started_str.replace("Z", "+00:00"))
    now     = datetime.now(timezone.utc)
    delta   = int((now - started).total_seconds())
    if delta < 0:
        delta = 0
    days    = delta // 86400
    hours   = (delta % 86400) // 3600
    minutes = (delta % 3600) // 60
    secs    = delta % 60
    parts = []
    if days:
        parts.append(f"{days}d")
    if hours:
        parts.append(f"{hours}h")
    if minutes:
        parts.append(f"{minutes}m")
    parts.append(f"{secs}s")
    print(" ".join(parts))
except Exception:
    print("unknown")
PYEOF
}

# ---- Node list parsing -------------------------------------------------------

# _cb_status_list_nodes
# Prints one line per node: <index> <pid> <type> <http_port> <ws_port> <label> <status>
# Fields separated by \x1f (ASCII unit separator).
_cb_status_list_nodes() {
  python3 - "$_CB_STATUS_PIDS_FILE" <<'PYEOF'
import sys, json

SEP = "\x1f"

with open(sys.argv[1]) as fh:
    data = json.load(fh)

nodes = data.get("nodes", {})
if isinstance(nodes, dict):
    items = sorted(nodes.items(), key=lambda x: int(x[0]))
else:
    items = [(str(i + 1), n) for i, n in enumerate(nodes)]

for key, node in items:
    print(SEP.join([
        str(key),
        str(node.get("pid",       "")),
        str(node.get("type",      "validator")),
        str(node.get("http_port", "")),
        str(node.get("ws_port",   "")),
        str(node.get("label",     f"node{key}")),
        str(node.get("status",    "unknown")),
    ]))
PYEOF
}

# ---- Consensus calculation ---------------------------------------------------

# _cb_status_consensus <validator_count> <active_count>
# Prints "OK", "DEGRADED", or "DOWN".
_cb_status_consensus() {
  local validators="$1"
  local active="$2"
  python3 - "$validators" "$active" <<'PYEOF'
import sys, math

validators = int(sys.argv[1])
active     = int(sys.argv[2])

if validators == 0:
    print("DOWN")
    sys.exit(0)

threshold = math.floor(validators * 2 / 3) + 1

if active == 0:
    print("DOWN")
elif active >= threshold:
    print("OK")
else:
    print("DEGRADED")
PYEOF
}

# _cb_status_bft_threshold <validator_count>
_cb_status_bft_threshold() {
  python3 -c "import math; print(math.floor(int('$1') * 2 / 3) + 1)" 2>/dev/null || printf '0\n'
}

# ---- Block time calculation --------------------------------------------------

# _cb_status_block_time_label <avg_float>
# Formats a float seconds value as "1.02s".
_cb_status_block_time_label() {
  python3 -c "print(f'{float(\"$1\"):.2f}s')" 2>/dev/null || printf 'N/A\n'
}

# ---- Human-readable table output --------------------------------------------

_cb_status_print_table() {
  local chain_name="$1"
  local profile_name="$2"
  local uptime="$3"
  local node_data="$4"       # newline-separated rows from _cb_status_collect_node_data
  local validator_count="$5"
  local active_validators="$6"
  local consensus="$7"
  local threshold="$8"

  # Header
  printf '\n'
  printf 'Chain:   %s\n' "$chain_name"
  printf 'Profile: %s\n' "$profile_name"
  printf 'Uptime:  %s\n' "$uptime"
  printf '\n'

  # Column widths (fixed for alignment)
  local w_node=5  w_type=10  w_pid=6  w_status=8  w_block=7  w_peers=6  w_http=12  w_ws=12

  # Header row
  printf " %-${w_node}s  %-${w_type}s  %-${w_pid}s  %-${w_status}s  %-${w_block}s  %-${w_peers}s  %-${w_http}s  %-${w_ws}s\n" \
    "Node" "Type" "PID" "Status" "Block" "Peers" "HTTP" "WS"

  # Separator
  local line
  line=" $(printf '%0.s─' $(seq 1 $w_node))  $(printf '%0.s─' $(seq 1 $w_type))  $(printf '%0.s─' $(seq 1 $w_pid))  $(printf '%0.s─' $(seq 1 $w_status))  $(printf '%0.s─' $(seq 1 $w_block))  $(printf '%0.s─' $(seq 1 $w_peers))  $(printf '%0.s─' $(seq 1 $w_http))  $(printf '%0.s─' $(seq 1 $w_ws))"
  printf '%s\n' "$line"

  # Data rows: each row is tab-separated: node_num type pid status block peers http ws
  while IFS=$'\t' read -r n_num n_type n_pid n_status n_block n_peers n_http n_ws; do
    printf " %-${w_node}s  %-${w_type}s  %-${w_pid}s  %-${w_status}s  %-${w_block}s  %-${w_peers}s  %-${w_http}s  %-${w_ws}s\n" \
      "$n_num" "$n_type" "$n_pid" "$n_status" "$n_block" "$n_peers" "$n_http" "$n_ws"
  done <<< "$node_data"

  printf '\n'

  # Consensus summary
  local consensus_label
  case "$consensus" in
    OK)       consensus_label="${_CB_GREEN}OK${_CB_RESET}" ;;
    DEGRADED) consensus_label="${_CB_YELLOW}DEGRADED${_CB_RESET}" ;;
    *)        consensus_label="${_CB_RED}DOWN${_CB_RESET}" ;;
  esac

  printf "Consensus: ${consensus_label} (%d/%d validators active, BFT threshold %d)\n" \
    "$active_validators" "$validator_count" "$threshold"
  printf '\n'
}

# ---- JSON output -------------------------------------------------------------

_cb_status_print_json() {
  local chain_name="$1"
  local profile_name="$2"
  local started_at="$3"
  local node_rows="$4"
  local validator_count="$5"
  local active_validators="$6"
  local consensus="$7"
  local threshold="$8"

  python3 - \
    "$chain_name" "$profile_name" "$started_at" \
    "$validator_count" "$active_validators" "$consensus" "$threshold" \
    "$node_rows" <<'PYEOF'
import sys, json

chain_name       = sys.argv[1]
profile_name     = sys.argv[2]
started_at       = sys.argv[3]
validator_count  = int(sys.argv[4])
active_validators= int(sys.argv[5])
consensus        = sys.argv[6]
threshold        = int(sys.argv[7])
node_rows_raw    = sys.argv[8]

nodes_out = []
for line in node_rows_raw.strip().splitlines():
    if not line.strip():
        continue
    parts = line.split("\t")
    if len(parts) < 8:
        continue
    nodes_out.append({
        "node":      parts[0],
        "type":      parts[1],
        "pid":       parts[2],
        "status":    parts[3],
        "block":     parts[4],
        "peers":     parts[5],
        "http_port": parts[6],
        "ws_port":   parts[7],
    })

output = {
    "chain_name":        chain_name,
    "profile":           profile_name,
    "started_at":        started_at,
    "validator_count":   validator_count,
    "active_validators": active_validators,
    "bft_threshold":     threshold,
    "consensus":         consensus,
    "nodes":             nodes_out,
}

print(json.dumps(output, indent=2))
PYEOF
}

# ---- Remote status logic -----------------------------------------------------

_cb_status_remote() {
  local alias="$1"

  if ! _cb_remote_exists "$alias" 2>/dev/null; then
    log_error "Remote alias '${alias}' not found"
    return 1
  fi

  local rpc_url auth_header
  rpc_url="$(_cb_remote_get_url "$alias")"
  auth_header="$(_cb_remote_get_auth_header "$alias" 2>/dev/null || echo "")"

  local entry_json chain_type
  entry_json="$(_cb_remote_get "$alias")"
  chain_type=$(python3 -c "import json,sys; print(json.loads(sys.argv[1]).get('chain_type','N/A'))" "$entry_json")

  # Query chain data
  local chain_id_hex block_hex peers_hex client_ver
  chain_id_hex="$(_cb_status_remote_rpc_call "$rpc_url" "eth_chainId" "[]" "$auth_header" 2>/dev/null)" || chain_id_hex=""
  block_hex="$(_cb_status_remote_rpc_call "$rpc_url" "eth_blockNumber" "[]" "$auth_header" 2>/dev/null)" || block_hex=""
  peers_hex="$(_cb_status_remote_rpc_call "$rpc_url" "net_peerCount" "[]" "$auth_header" 2>/dev/null)" || peers_hex=""
  client_ver="$(_cb_status_remote_rpc_call "$rpc_url" "web3_clientVersion" "[]" "$auth_header" 2>/dev/null)" || client_ver=""

  local chain_id="N/A" block_num="N/A" peer_count="N/A" status="unreachable"

  if [[ -n "$chain_id_hex" && "$chain_id_hex" =~ ^0x ]]; then
    chain_id="$(_cb_status_hex_to_dec "$chain_id_hex")"
    status="connected"
  fi
  if [[ -n "$block_hex" && "$block_hex" =~ ^0x ]]; then
    block_num="$(_cb_status_hex_to_dec "$block_hex")"
  fi
  if [[ -n "$peers_hex" && "$peers_hex" =~ ^0x ]]; then
    peer_count="$(_cb_status_hex_to_dec "$peers_hex")"
  fi

  if [[ "$_CB_STATUS_JSON_MODE" == "1" ]]; then
    python3 -c "
import json, sys
data = {
    'type':           'remote',
    'alias':          sys.argv[1],
    'rpc_url':        sys.argv[2],
    'chain_type':     sys.argv[3],
    'chain_id':       sys.argv[4],
    'latest_block':   sys.argv[5],
    'peer_count':     sys.argv[6],
    'client_version': sys.argv[7],
    'status':         sys.argv[8],
}
for k in ('chain_id', 'latest_block', 'peer_count'):
    try: data[k] = int(data[k])
    except: pass
print(json.dumps(data, indent=2))
" "$alias" "$rpc_url" "$chain_type" "$chain_id" "$block_num" "$peer_count" \
  "${client_ver:-N/A}" "$status"
  else
    printf '\n'
    printf '  Remote Chain Status: %s\n' "$alias"
    printf '  %s\n' "$(printf '%.0s─' $(seq 1 50))"
    printf '  %-18s  %s\n' "RPC URL" "$rpc_url"
    printf '  %-18s  %s\n' "Chain Type" "$chain_type"
    printf '  %-18s  %s\n' "Status" "$status"
    printf '  %-18s  %s\n' "Chain ID" "$chain_id"
    printf '  %-18s  %s\n' "Latest Block" "$block_num"
    printf '  %-18s  %s\n' "Peer Count" "$peer_count"
    printf '  %-18s  %s\n' "Client Version" "${client_ver:-N/A}"
    printf '\n'
  fi

  return 0
}

# ---- Main status logic -------------------------------------------------------

cmd_status_main() {
  _cb_status_parse_args "$@"

  # Remote status mode
  if [[ -n "$_CB_STATUS_REMOTE_ALIAS" ]]; then
    _cb_status_remote "$_CB_STATUS_REMOTE_ALIAS"
    return $?
  fi

  if [[ ! -f "$_CB_STATUS_PIDS_FILE" ]]; then
    log_warn "No pids.json found — chain has not been started"
    if [[ "$_CB_STATUS_JSON_MODE" == "1" ]]; then
      printf '{"error":"no pids.json","chain_name":"","nodes":[]}\n'
    fi
    return 0
  fi

  local chain_name profile_name started_at uptime
  chain_name="$(_cb_status_read_chain_name)"
  profile_name="$(_cb_status_read_profile_name)"
  started_at="$(_cb_status_read_started_at)"
  uptime="$(_cb_status_calc_uptime "$started_at")"

  # Collect per-node data
  local node_list
  node_list="$(_cb_status_list_nodes)"

  local -a table_rows=()
  local validator_count=0 active_validators=0

  while IFS=$'\x1f' read -r idx pid node_type http_port ws_port label stored_status; do
    local status="$stored_status"

    # Verify PID liveness
    if [[ "$stored_status" == "running" ]]; then
      if [[ -n "$pid" ]] && _cb_status_pid_alive "$pid"; then
        status="running"
      else
        status="stopped"
        _cb_status_mark_node_dead "$idx" 2>/dev/null || true
      fi
    fi

    # Count validators for consensus
    if [[ "$node_type" == "validator" ]]; then
      (( ++validator_count )) || true
      [[ "$status" == "running" ]] && { (( ++active_validators )) || true; }
    fi

    # Query block number and peer count for running nodes
    local block_num="N/A" peer_count="N/A"
    if [[ "$status" == "running" && -n "$http_port" ]]; then
      local raw_block
      raw_block="$(_cb_status_rpc_call "$http_port" "eth_blockNumber" "[]" 2>/dev/null)" || true
      if [[ -n "$raw_block" && "$raw_block" != "None" ]]; then
        block_num="$(_cb_status_hex_to_dec "$raw_block")"
      fi

      local raw_peers
      raw_peers="$(_cb_status_rpc_call "$http_port" "net_peerCount" "[]" 2>/dev/null)" || true
      if [[ -n "$raw_peers" && "$raw_peers" != "None" ]]; then
        local peers_dec
        peers_dec="$(_cb_status_hex_to_dec "$raw_peers")"
        peer_count="${peers_dec}/${validator_count}"
      fi
    fi

    local node_num="${idx}"
    local http_label ws_label
    http_label="${http_port:+:${http_port}}"
    ws_label="${ws_port:+:${ws_port}}"

    table_rows+=( "${node_num}"$'\t'"${node_type}"$'\t'"${pid:-—}"$'\t'"${status}"$'\t'"${block_num}"$'\t'"${peer_count}"$'\t'"${http_label:-—}"$'\t'"${ws_label:-—}" )
  done <<< "$node_list"

  local consensus threshold
  consensus="$(_cb_status_consensus "$validator_count" "$active_validators")"
  threshold="$(_cb_status_bft_threshold "$validator_count")"

  # Join table rows into a single newline-delimited string
  local joined_rows
  joined_rows="$(printf '%s\n' "${table_rows[@]+"${table_rows[@]}"}")"

  if [[ "$_CB_STATUS_JSON_MODE" == "1" ]]; then
    _cb_status_print_json \
      "$chain_name" "$profile_name" "$started_at" \
      "$joined_rows" \
      "$validator_count" "$active_validators" \
      "$consensus" "$threshold"
  else
    _cb_status_print_table \
      "$chain_name" "$profile_name" "$uptime" \
      "$joined_rows" \
      "$validator_count" "$active_validators" \
      "$consensus" "$threshold"
  fi

  return 0
}

# ---- Entry point -------------------------------------------------------------

cmd_status_main "$@"
