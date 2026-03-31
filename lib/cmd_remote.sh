#!/usr/bin/env bash
# lib/cmd_remote.sh - Manage remote chain RPC connections
# Sourced by the main chainbench dispatcher; $@ contains subcommand arguments.
#
# Sub-subcommands:
#   remote add    <alias> <rpc_url> [--type <type>] [--ws <ws_url>] [--auth-header <header>]
#   remote list   [--json]
#   remote remove <alias>
#   remote select <alias>
#   remote info   <alias> [--json]

# Guard against double-sourcing
[[ -n "${_CB_CMD_REMOTE_SH_LOADED:-}" ]] && return 0
readonly _CB_CMD_REMOTE_SH_LOADED=1

_CB_REMOTE_CMD_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${_CB_REMOTE_CMD_LIB_DIR}/common.sh"
source "${_CB_REMOTE_CMD_LIB_DIR}/remote_state.sh"
source "${_CB_REMOTE_CMD_LIB_DIR}/rpc_client.sh"

# ---- Constants ---------------------------------------------------------------

readonly _CB_REMOTE_CURRENT_FILE="${CHAINBENCH_DIR}/state/current-remote"
readonly _CB_REMOTE_RPC_TIMEOUT=10

# ---- Usage -------------------------------------------------------------------

_cb_remote_usage() {
  cat >&2 <<'EOF'
Usage: chainbench remote <subcommand> [options]

Subcommands:
  add    <alias> <rpc_url> [options]   Register a remote chain endpoint
  list   [--json]                      List registered remote chains
  remove <alias>                       Remove a registered remote chain
  select <alias>                       Set active remote chain
  info   <alias> [--json]              Show chain info from remote RPC

Add options:
  --type <testnet|mainnet|devnet>      Chain type (default: testnet)
  --ws <ws_url>                        WebSocket RPC URL
  --auth-header <header>               Authorization header (stored securely)

Examples:
  chainbench remote add eth-mainnet https://eth.llamarpc.com --type mainnet
  chainbench remote add my-testnet https://rpc.testnet.example.com --type testnet
  chainbench remote list
  chainbench remote info eth-mainnet --json
  chainbench remote select eth-mainnet
  chainbench remote remove eth-mainnet
EOF
}

# ---- remote add --------------------------------------------------------------

_cb_remote_cmd_add() {
  local alias="" rpc_url="" chain_type="testnet" ws_url="" auth_header=""

  # Parse positional + named args
  local positionals=()
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --type)       chain_type="${2:?--type requires a value}"; shift 2 ;;
      --type=*)     chain_type="${1#--type=}"; shift ;;
      --ws)         ws_url="${2:?--ws requires a value}"; shift 2 ;;
      --ws=*)       ws_url="${1#--ws=}"; shift ;;
      --auth-header) auth_header="${2:?--auth-header requires a value}"; shift 2 ;;
      --auth-header=*) auth_header="${1#--auth-header=}"; shift ;;
      -*)           log_warn "Unknown option: $1"; shift ;;
      *)            positionals+=("$1"); shift ;;
    esac
  done

  alias="${positionals[0]:-}"
  rpc_url="${positionals[1]:-}"

  if [[ -z "$alias" || -z "$rpc_url" ]]; then
    log_error "Usage: chainbench remote add <alias> <rpc_url> [--type <type>]"
    return 1
  fi

  _cb_remote_validate_alias "$alias" || return 1
  _cb_remote_validate_url "$rpc_url" || return 1
  _cb_remote_validate_chain_type "$chain_type" || return 1

  if [[ -n "$ws_url" ]]; then
    if ! [[ "$ws_url" =~ ^wss?:// ]]; then
      log_error "Invalid WebSocket URL: must start with ws:// or wss://"
      return 1
    fi
  fi

  # Check for duplicate
  if _cb_remote_exists "$alias"; then
    log_error "Remote alias '${alias}' already exists. Use 'remove' first."
    return 1
  fi

  # Connectivity check
  log_info "Checking connectivity to ${rpc_url} ..."
  local chain_id_hex=""
  chain_id_hex="$(_cb_remote_check_connectivity "$rpc_url" "$auth_header" 2>/dev/null)" || true

  local status="active"
  local chain_id_for_add=""
  if [[ -n "$chain_id_hex" ]]; then
    chain_id_for_add="$chain_id_hex"
    local chain_id_dec
    chain_id_dec=$(python3 -c "print(int('${chain_id_hex}', 16))" 2>/dev/null || echo "?")
    log_success "Connected. Chain ID: ${chain_id_dec} (${chain_id_hex})"
  else
    status="unreachable"
    log_warn "Could not connect to ${rpc_url} — added with status 'unreachable'"
  fi

  # Add to state
  local new_state
  new_state="$(_cb_remote_add "$alias" "$rpc_url" "$chain_type" "$ws_url" "$auth_header" "$chain_id_for_add")" || return 1

  # If unreachable, patch status
  if [[ "$status" == "unreachable" ]]; then
    new_state=$(python3 -c "
import json, sys
state = json.loads(sys.argv[1])
state['remotes'][sys.argv[2]]['status'] = 'unreachable'
print(json.dumps(state, indent=2))
" "$new_state" "$alias")
  fi

  _cb_remote_save_state "$new_state"
  log_success "Remote '${alias}' registered (${chain_type}, ${rpc_url})"
  return 0
}

# ---- remote list -------------------------------------------------------------

_cb_remote_cmd_list() {
  local json_mode=0
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --json) json_mode=1; shift ;;
      *)      shift ;;
    esac
  done

  local list_json
  list_json="$(_cb_remote_list)"

  if [[ "$json_mode" == "1" ]]; then
    printf '%s\n' "$list_json"
    return 0
  fi

  # Human-readable table
  local count
  count=$(python3 -c "import json,sys; print(len(json.loads(sys.argv[1])))" "$list_json")

  if [[ "$count" == "0" ]]; then
    log_info "No remote chains registered. Use 'chainbench remote add' to register one."
    return 0
  fi

  printf '\n'
  printf " %-15s  %-40s  %-10s  %-10s  %-12s\n" \
    "Alias" "RPC URL" "Type" "Chain ID" "Status"
  printf " %-15s  %-40s  %-10s  %-10s  %-12s\n" \
    "───────────────" "────────────────────────────────────────" \
    "──────────" "──────────" "────────────"

  python3 -c "
import json, sys

data = json.loads(sys.argv[1])
for entry in data:
    alias    = entry.get('alias', '')[:15]
    rpc_url  = entry.get('rpc_url', '')[:40]
    ctype    = entry.get('chain_type', '')[:10]
    chain_id = str(entry.get('chain_id', 'N/A'))[:10]
    status   = entry.get('status', 'unknown')[:12]
    print(f' {alias:<15s}  {rpc_url:<40s}  {ctype:<10s}  {chain_id:<10s}  {status:<12s}')
" "$list_json"

  printf '\n'
  log_info "${count} remote chain(s) registered"
  return 0
}

# ---- remote remove -----------------------------------------------------------

_cb_remote_cmd_remove() {
  local alias="${1:?Usage: chainbench remote remove <alias>}"

  if ! _cb_remote_exists "$alias"; then
    log_error "Remote alias '${alias}' not found"
    return 1
  fi

  local new_state
  new_state="$(_cb_remote_remove "$alias")" || return 1
  _cb_remote_save_state "$new_state"

  # Clear current-remote if it points to this alias
  if [[ -f "$_CB_REMOTE_CURRENT_FILE" ]]; then
    local current
    current="$(cat "$_CB_REMOTE_CURRENT_FILE" 2>/dev/null || echo "")"
    if [[ "$current" == "$alias" ]]; then
      rm -f "$_CB_REMOTE_CURRENT_FILE"
      log_info "Cleared active remote selection (was '${alias}')"
    fi
  fi

  log_success "Remote '${alias}' removed"
  return 0
}

# ---- remote select -----------------------------------------------------------

_cb_remote_cmd_select() {
  local alias="${1:?Usage: chainbench remote select <alias>}"

  if ! _cb_remote_exists "$alias"; then
    log_error "Remote alias '${alias}' not found"
    return 1
  fi

  _cb_remote_ensure_state_dir
  printf '%s' "$alias" > "$_CB_REMOTE_CURRENT_FILE"
  log_success "Active remote set to '${alias}'"
  return 0
}

# ---- remote info -------------------------------------------------------------

_cb_remote_cmd_info() {
  local alias="" json_mode=0

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --json) json_mode=1; shift ;;
      -*)     log_warn "Unknown option: $1"; shift ;;
      *)      alias="$1"; shift ;;
    esac
  done

  if [[ -z "$alias" ]]; then
    log_error "Usage: chainbench remote info <alias> [--json]"
    return 1
  fi

  if ! _cb_remote_exists "$alias"; then
    log_error "Remote alias '${alias}' not found"
    return 1
  fi

  local rpc_url auth_header
  rpc_url="$(_cb_remote_get_url "$alias")"
  auth_header="$(_cb_remote_get_auth_header "$alias" 2>/dev/null || echo "")"

  log_info "Querying ${rpc_url} ..."

  # Helper: query a single RPC method against this remote
  _rpc_query() {
    local method="$1" params="${2:-[]}"
    local result
    result=$(cb_rpc_raw "$rpc_url" "$method" "$params" "$auth_header" "$_CB_REMOTE_RPC_TIMEOUT" 2>/dev/null) || { echo "N/A"; return; }
    cb_json_get_result "$result" 2>/dev/null || echo "N/A"
  }

  local chain_id_hex network_id latest_block_hex gas_price_hex
  local peer_count_hex client_version protocol_version syncing mining

  chain_id_hex="$(_rpc_query "eth_chainId")"
  network_id="$(_rpc_query "net_version")"
  latest_block_hex="$(_rpc_query "eth_blockNumber")"
  gas_price_hex="$(_rpc_query "eth_gasPrice")"
  peer_count_hex="$(_rpc_query "net_peerCount")"
  client_version="$(_rpc_query "web3_clientVersion")"
  protocol_version="$(_rpc_query "eth_protocolVersion")"
  syncing="$(_rpc_query "eth_syncing")"
  mining="$(_rpc_query "eth_mining")"

  # Convert hex values using cb_hex_to_dec
  local chain_id_dec latest_block_dec gas_price_dec peer_count_dec
  if [[ "$chain_id_hex" =~ ^0x ]]; then
    chain_id_dec=$(cb_hex_to_dec "$chain_id_hex" 2>/dev/null || echo "N/A")
  else
    chain_id_dec="$chain_id_hex"
  fi

  if [[ "$latest_block_hex" =~ ^0x ]]; then
    latest_block_dec=$(cb_hex_to_dec "$latest_block_hex" 2>/dev/null || echo "N/A")
  else
    latest_block_dec="$latest_block_hex"
  fi

  if [[ "$gas_price_hex" =~ ^0x ]]; then
    gas_price_dec=$(cb_hex_to_dec "$gas_price_hex" 2>/dev/null || echo "N/A")
  else
    gas_price_dec="$gas_price_hex"
  fi

  if [[ "$peer_count_hex" =~ ^0x ]]; then
    peer_count_dec=$(cb_hex_to_dec "$peer_count_hex" 2>/dev/null || echo "N/A")
  else
    peer_count_dec="$peer_count_hex"
  fi

  # Determine sync status
  local sync_status="synced"
  if [[ "$syncing" != "false" && "$syncing" != "N/A" ]]; then
    sync_status="syncing"
  fi

  # Update last_checked
  _cb_remote_update_last_checked "$alias" 2>/dev/null || true

  # Update chain_id in state if we got one
  if [[ "$chain_id_hex" != "N/A" && "$chain_id_hex" =~ ^0x ]]; then
    local updated_state
    updated_state="$(_cb_remote_update_field "$alias" "chain_id" "$chain_id_hex")" || true
    if [[ -n "$updated_state" ]]; then
      _cb_remote_save_state "$updated_state"
    fi
  fi

  if [[ "$json_mode" == "1" ]]; then
    python3 -c "
import json, sys

data = {
    'alias':            sys.argv[1],
    'rpc_url':          sys.argv[2],
    'chain_id':         sys.argv[3],
    'chain_id_hex':     sys.argv[4],
    'network_id':       sys.argv[5],
    'latest_block':     sys.argv[6],
    'gas_price_wei':    sys.argv[7],
    'peer_count':       sys.argv[8],
    'client_version':   sys.argv[9],
    'protocol_version': sys.argv[10],
    'syncing':          sys.argv[11] not in ('false', 'N/A'),
    'sync_status':      sys.argv[12],
    'mining':           sys.argv[13],
}
# Convert numeric strings
for k in ('chain_id', 'latest_block', 'gas_price_wei', 'peer_count'):
    try:
        data[k] = int(data[k])
    except (ValueError, TypeError):
        pass

print(json.dumps(data, indent=2))
" "$alias" "$rpc_url" \
  "$chain_id_dec" "$chain_id_hex" "$network_id" "$latest_block_dec" \
  "$gas_price_dec" "$peer_count_dec" "$client_version" "$protocol_version" \
  "$syncing" "$sync_status" "$mining"
    return 0
  fi

  # Human-readable output
  local entry_json
  entry_json="$(_cb_remote_get "$alias")"
  local chain_type
  chain_type=$(python3 -c "import json,sys; print(json.loads(sys.argv[1]).get('chain_type','N/A'))" "$entry_json")

  printf '\n'
  printf '  Remote Chain Info: %s\n' "$alias"
  printf '  %s\n' "$(printf '%.0s─' $(seq 1 50))"
  printf '  %-20s  %s\n' "RPC URL" "$rpc_url"
  printf '  %-20s  %s\n' "Chain Type" "$chain_type"
  printf '  %-20s  %s (%s)\n' "Chain ID" "$chain_id_dec" "$chain_id_hex"
  printf '  %-20s  %s\n' "Network ID" "$network_id"
  printf '  %-20s  %s\n' "Latest Block" "$latest_block_dec"
  printf '  %-20s  %s wei\n' "Gas Price" "$gas_price_dec"
  printf '  %-20s  %s\n' "Peer Count" "$peer_count_dec"
  printf '  %-20s  %s\n' "Client Version" "$client_version"
  printf '  %-20s  %s\n' "Protocol Version" "$protocol_version"
  printf '  %-20s  %s\n' "Sync Status" "$sync_status"
  printf '  %-20s  %s\n' "Mining" "$mining"
  printf '\n'

  return 0
}

# ---- Subcommand dispatcher ---------------------------------------------------

cmd_remote_main() {
  if [[ $# -lt 1 ]]; then
    _cb_remote_usage
    return 1
  fi

  local subcmd="$1"
  shift

  case "$subcmd" in
    add)
      _cb_remote_cmd_add "$@"
      ;;
    list)
      _cb_remote_cmd_list "$@"
      ;;
    remove)
      if [[ $# -lt 1 ]]; then
        log_error "Usage: chainbench remote remove <alias>"
        return 1
      fi
      _cb_remote_cmd_remove "$1"
      ;;
    select)
      if [[ $# -lt 1 ]]; then
        log_error "Usage: chainbench remote select <alias>"
        return 1
      fi
      _cb_remote_cmd_select "$1"
      ;;
    info)
      _cb_remote_cmd_info "$@"
      ;;
    --help|-h|help)
      _cb_remote_usage
      return 0
      ;;
    *)
      log_error "Unknown remote subcommand: '$subcmd'"
      _cb_remote_usage
      return 1
      ;;
  esac
}

# ---- Entry point -------------------------------------------------------------

cmd_remote_main "$@"
