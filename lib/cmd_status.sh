#!/usr/bin/env bash
# lib/cmd_status.sh - Show chain and node status
# Sourced by the main chainbench dispatcher; $@ contains subcommand arguments.

# Guard against double-sourcing
[[ -n "${_CB_CMD_STATUS_SH_LOADED:-}" ]] && return 0
readonly _CB_CMD_STATUS_SH_LOADED=1

_CB_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${_CB_LIB_DIR}/common.sh"
source "${_CB_LIB_DIR}/pids_state.sh"
source "${_CB_LIB_DIR}/rpc_client.sh"
source "${_CB_LIB_DIR}/consensus_calc.sh"
source "${_CB_LIB_DIR}/formatter.sh"

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

# ---- Block time label --------------------------------------------------------

# _cb_status_block_time_label <avg_float>
# Formats a float seconds value as "1.02s".
_cb_status_block_time_label() {
  python3 -c "print(f'{float(\"$1\"):.2f}s')" 2>/dev/null || printf 'N/A\n'
}

# ---- Human-readable table output ---------------------------------------------

_cb_status_print_table() {
  local chain_name="$1"
  local profile_name="$2"
  local uptime="$3"
  local node_data="$4"       # newline-separated tab-delimited rows
  local validator_count="$5"
  local active_validators="$6"
  local consensus="$7"
  local threshold="$8"

  printf '\n'
  printf 'Chain:   %s\n' "$chain_name"
  printf 'Profile: %s\n' "$profile_name"
  printf 'Uptime:  %s\n' "$uptime"
  printf '\n'

  cb_format_table_header "Node:5|Type:10|PID:6|Status:8|Block:7|Peers:6|HTTP:12|WS:12"

  while IFS=$'\t' read -r n_num n_type n_pid n_status n_block n_peers n_http n_ws; do
    cb_format_table_row "5,10,6,8,7,6,12,12" \
      "$n_num" "$n_type" "$n_pid" "$n_status" "$n_block" "$n_peers" "$n_http" "$n_ws"
  done <<< "$node_data"

  printf '\n'

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

chain_name        = sys.argv[1]
profile_name      = sys.argv[2]
started_at        = sys.argv[3]
validator_count   = int(sys.argv[4])
active_validators = int(sys.argv[5])
consensus         = sys.argv[6]
threshold         = int(sys.argv[7])
node_rows_raw     = sys.argv[8]

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

  _cb_rpc_ensure_remote_state

  if ! _cb_remote_exists "$alias" 2>/dev/null; then
    log_error "Remote alias '${alias}' not found"
    return 1
  fi

  local rpc_url auth_header entry_json chain_type
  rpc_url="$(_cb_remote_get_url "$alias")"
  auth_header="$(_cb_remote_get_auth_header "$alias" 2>/dev/null || printf '')"
  entry_json="$(_cb_remote_get "$alias")"
  chain_type=$(python3 -c \
    "import json,sys; print(json.loads(sys.argv[1]).get('chain_type','N/A'))" \
    "$entry_json")

  local chain_id_hex block_hex peers_hex client_ver
  chain_id_hex="$(cb_rpc_result "$rpc_url" "eth_chainId"    "[]" "$auth_header" 2>/dev/null)" || chain_id_hex=""
  block_hex="$(   cb_rpc_result "$rpc_url" "eth_blockNumber" "[]" "$auth_header" 2>/dev/null)" || block_hex=""
  peers_hex="$(   cb_rpc_result "$rpc_url" "net_peerCount"   "[]" "$auth_header" 2>/dev/null)" || peers_hex=""
  client_ver="$(  cb_rpc_result "$rpc_url" "web3_clientVersion" "[]" "$auth_header" 2>/dev/null)" || client_ver=""

  local chain_id="N/A" block_num="N/A" peer_count="N/A" status="unreachable"

  if [[ -n "$chain_id_hex" && "$chain_id_hex" =~ ^0x ]]; then
    chain_id="$(cb_hex_to_dec "$chain_id_hex")"
    status="connected"
  fi
  if [[ -n "$block_hex" && "$block_hex" =~ ^0x ]]; then
    block_num="$(cb_hex_to_dec "$block_hex")"
  fi
  if [[ -n "$peers_hex" && "$peers_hex" =~ ^0x ]]; then
    peer_count="$(cb_hex_to_dec "$peers_hex")"
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
    printf '  %-18s  %s\n' "RPC URL"        "$rpc_url"
    printf '  %-18s  %s\n' "Chain Type"     "$chain_type"
    printf '  %-18s  %s\n' "Status"         "$status"
    printf '  %-18s  %s\n' "Chain ID"       "$chain_id"
    printf '  %-18s  %s\n' "Latest Block"   "$block_num"
    printf '  %-18s  %s\n' "Peer Count"     "$peer_count"
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

  if ! pids_exists; then
    log_warn "No pids.json found — chain has not been started"
    if [[ "$_CB_STATUS_JSON_MODE" == "1" ]]; then
      printf '{"error":"no pids.json","chain_name":"","nodes":[]}\n'
    fi
    return 0
  fi

  local chain_name profile_name started_at uptime
  chain_name="$(  pids_read_meta "chain_id"   || pids_read_meta "chain_name" || printf 'unknown')"
  profile_name="$(pids_read_meta "profile"    || printf 'unknown')"
  started_at="$(  pids_read_meta "started_at" || printf '')"
  uptime="$(      cb_format_uptime "$started_at")"

  # Collect per-node data
  local node_ids
  node_ids="$(pids_list_nodes)"

  local -a table_rows=()
  local validator_count=0 active_validators=0

  for idx in $node_ids; do
    local node_type http_port ws_port pid stored_status
    node_type="$(    pids_get_field "$idx" "type"      || printf 'validator')"
    http_port="$(    pids_get_field "$idx" "http_port" || printf '')"
    ws_port="$(      pids_get_field "$idx" "ws_port"   || printf '')"
    pid="$(          pids_get_field "$idx" "pid"       || printf '')"
    stored_status="$(pids_get_field "$idx" "status"    || printf 'unknown')"

    local status="$stored_status"

    # Verify PID liveness
    if [[ "$stored_status" == "running" ]]; then
      if pids_pid_alive "$idx"; then
        status="running"
      else
        status="stopped"
        pids_mark_node_dead "$idx" 2>/dev/null || true
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
      raw_block="$(cb_rpc_result "http://127.0.0.1:${http_port}" "eth_blockNumber" "[]" 2>/dev/null)" || raw_block=""
      if [[ -n "$raw_block" && "$raw_block" != "None" ]]; then
        block_num="$(cb_hex_to_dec "$raw_block")"
      fi

      local raw_peers
      raw_peers="$(cb_rpc_result "http://127.0.0.1:${http_port}" "net_peerCount" "[]" 2>/dev/null)" || raw_peers=""
      if [[ -n "$raw_peers" && "$raw_peers" != "None" ]]; then
        local peers_dec
        peers_dec="$(cb_hex_to_dec "$raw_peers")"
        peer_count="${peers_dec}/${validator_count}"
      fi
    fi

    local http_label ws_label
    http_label="${http_port:+:${http_port}}"
    ws_label="${ws_port:+:${ws_port}}"

    table_rows+=( "${idx}"$'\t'"${node_type}"$'\t'"${pid:-—}"$'\t'"${status}"$'\t'"${block_num}"$'\t'"${peer_count}"$'\t'"${http_label:-—}"$'\t'"${ws_label:-—}" )
  done

  local consensus threshold
  consensus="$(cb_consensus_status "$active_validators" "$validator_count")"
  threshold="$( cb_bft_threshold  "$validator_count")"

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
