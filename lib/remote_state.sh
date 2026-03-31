#!/usr/bin/env bash
# lib/remote_state.sh - Remote chain state management (CRUD for state/remotes.json)
# Usage: source lib/remote_state.sh

# Guard against double-sourcing
[[ -n "${_CB_REMOTE_STATE_SH_LOADED:-}" ]] && return 0
readonly _CB_REMOTE_STATE_SH_LOADED=1

_CB_REMOTE_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${_CB_REMOTE_LIB_DIR}/common.sh"

readonly _CB_REMOTES_FILE="${CHAINBENCH_DIR}/state/remotes.json"

# ---------------------------------------------------------------------------
# Validation helpers
# ---------------------------------------------------------------------------

# _cb_remote_validate_alias <alias>
# Returns 0 if alias is valid (alphanumeric + dashes), 1 otherwise.
_cb_remote_validate_alias() {
  local alias="$1"
  if [[ -z "$alias" ]]; then
    log_error "Alias cannot be empty"
    return 1
  fi
  if ! [[ "$alias" =~ ^[a-zA-Z][a-zA-Z0-9-]*$ ]]; then
    log_error "Invalid alias '${alias}': must start with a letter, contain only alphanumeric and dashes"
    return 1
  fi
  return 0
}

# _cb_remote_validate_url <url>
# Returns 0 if URL starts with http:// or https://, 1 otherwise.
_cb_remote_validate_url() {
  local url="$1"
  if [[ -z "$url" ]]; then
    log_error "RPC URL cannot be empty"
    return 1
  fi
  if ! [[ "$url" =~ ^https?:// ]]; then
    log_error "Invalid RPC URL '${url}': must start with http:// or https://"
    return 1
  fi
  return 0
}

# _cb_remote_validate_chain_type <type>
# Returns 0 if type is testnet|mainnet|devnet, 1 otherwise.
_cb_remote_validate_chain_type() {
  local chain_type="$1"
  case "$chain_type" in
    testnet|mainnet|devnet) return 0 ;;
    *)
      log_error "Invalid chain type '${chain_type}': must be testnet, mainnet, or devnet"
      return 1
      ;;
  esac
}

# ---------------------------------------------------------------------------
# State file I/O
# ---------------------------------------------------------------------------

# _cb_remote_ensure_state_dir
# Creates the state directory if it doesn't exist.
_cb_remote_ensure_state_dir() {
  local state_dir="${CHAINBENCH_DIR}/state"
  if [[ ! -d "$state_dir" ]]; then
    mkdir -p "$state_dir"
  fi
}

# _cb_remote_load_state
# Prints the contents of remotes.json. Returns empty object if file doesn't exist.
_cb_remote_load_state() {
  if [[ -f "$_CB_REMOTES_FILE" ]]; then
    cat "$_CB_REMOTES_FILE"
  else
    printf '{"remotes":{}}\n'
  fi
}

# _cb_remote_save_state <json_content>
# Writes JSON content to remotes.json with restrictive permissions.
_cb_remote_save_state() {
  local content="$1"
  _cb_remote_ensure_state_dir
  printf '%s\n' "$content" > "$_CB_REMOTES_FILE"
  chmod 600 "$_CB_REMOTES_FILE"
}

# ---------------------------------------------------------------------------
# CRUD operations
# ---------------------------------------------------------------------------

# _cb_remote_add <alias> <rpc_url> [chain_type] [ws_url] [auth_header] [chain_id]
# Adds a new remote entry. Returns 1 if alias already exists.
_cb_remote_add() {
  local alias="$1" rpc_url="$2"
  local chain_type="${3:-testnet}" ws_url="${4:-}" auth_header="${5:-}" chain_id="${6:-}"

  _cb_remote_validate_alias "$alias" || return 1
  _cb_remote_validate_url "$rpc_url" || return 1
  _cb_remote_validate_chain_type "$chain_type" || return 1

  # Strip trailing slash from URL
  rpc_url="${rpc_url%/}"
  [[ -n "$ws_url" ]] && ws_url="${ws_url%/}"

  python3 -c "
import json, sys
from datetime import datetime, timezone

alias       = sys.argv[1]
rpc_url     = sys.argv[2]
chain_type  = sys.argv[3]
ws_url      = sys.argv[4] or None
auth_header = sys.argv[5] or None
chain_id    = sys.argv[6]
state_raw   = sys.argv[7]

state = json.loads(state_raw)
remotes = state.get('remotes', {})

if alias in remotes:
    print(f'ERROR: alias \"{alias}\" already exists', file=sys.stderr)
    sys.exit(1)

entry = {
    'rpc_url':      rpc_url,
    'chain_type':   chain_type,
    'status':       'active',
    'added_at':     datetime.now(timezone.utc).strftime('%Y-%m-%dT%H:%M:%SZ'),
}
if ws_url:
    entry['ws_url'] = ws_url
if auth_header:
    entry['auth_header'] = auth_header
if chain_id:
    try:
        entry['chain_id'] = int(chain_id, 16) if chain_id.startswith('0x') else int(chain_id)
    except ValueError:
        pass

remotes[alias] = entry
state['remotes'] = remotes
print(json.dumps(state, indent=2))
" "$alias" "$rpc_url" "$chain_type" "$ws_url" "$auth_header" "$chain_id" \
    "$(_cb_remote_load_state)"
}

# _cb_remote_remove <alias>
# Removes a remote entry. Returns 1 if alias not found.
_cb_remote_remove() {
  local alias="$1"
  _cb_remote_validate_alias "$alias" || return 1

  python3 -c "
import json, sys

alias     = sys.argv[1]
state_raw = sys.argv[2]

state = json.loads(state_raw)
remotes = state.get('remotes', {})

if alias not in remotes:
    print(f'ERROR: alias \"{alias}\" not found', file=sys.stderr)
    sys.exit(1)

del remotes[alias]
state['remotes'] = remotes
print(json.dumps(state, indent=2))
" "$alias" "$(_cb_remote_load_state)"
}

# _cb_remote_get <alias>
# Prints the JSON object for a single remote entry.
_cb_remote_get() {
  local alias="$1"
  _cb_remote_validate_alias "$alias" || return 1

  python3 -c "
import json, sys

alias     = sys.argv[1]
state_raw = sys.argv[2]

state = json.loads(state_raw)
remotes = state.get('remotes', {})

if alias not in remotes:
    print(f'ERROR: alias \"{alias}\" not found', file=sys.stderr)
    sys.exit(1)

entry = remotes[alias]
entry['alias'] = alias
print(json.dumps(entry, indent=2))
" "$alias" "$(_cb_remote_load_state)"
}

# _cb_remote_get_url <alias>
# Prints only the RPC URL for the given alias.
_cb_remote_get_url() {
  local alias="$1"
  _cb_remote_validate_alias "$alias" || return 1

  python3 -c "
import json, sys

alias     = sys.argv[1]
state_raw = sys.argv[2]

state = json.loads(state_raw)
remotes = state.get('remotes', {})

if alias not in remotes:
    print(f'ERROR: alias \"{alias}\" not found', file=sys.stderr)
    sys.exit(1)

print(remotes[alias]['rpc_url'])
" "$alias" "$(_cb_remote_load_state)"
}

# _cb_remote_get_auth_header <alias>
# Prints the auth header for the given alias (empty string if none).
_cb_remote_get_auth_header() {
  local alias="$1"
  python3 -c "
import json, sys

alias     = sys.argv[1]
state_raw = sys.argv[2]

state = json.loads(state_raw)
remotes = state.get('remotes', {})

if alias not in remotes:
    sys.exit(1)

print(remotes[alias].get('auth_header', ''))
" "$alias" "$(_cb_remote_load_state)"
}

# _cb_remote_list
# Prints all remotes as a JSON array.
_cb_remote_list() {
  python3 -c "
import json, sys

state_raw = sys.argv[1]
state = json.loads(state_raw)
remotes = state.get('remotes', {})

result = []
for alias, entry in sorted(remotes.items()):
    item = dict(entry)
    item['alias'] = alias
    # Mask auth_header for display
    if 'auth_header' in item:
        item['auth_header'] = '***masked***'
    result.append(item)

print(json.dumps(result, indent=2))
" "$(_cb_remote_load_state)"
}

# _cb_remote_exists <alias>
# Returns 0 if alias exists, 1 otherwise.
_cb_remote_exists() {
  local alias="$1"
  python3 -c "
import json, sys

alias     = sys.argv[1]
state_raw = sys.argv[2]
state = json.loads(state_raw)

if alias in state.get('remotes', {}):
    sys.exit(0)
else:
    sys.exit(1)
" "$alias" "$(_cb_remote_load_state)"
}

# _cb_remote_update_field <alias> <field> <value>
# Updates a single field in a remote entry.
_cb_remote_update_field() {
  local alias="$1" field="$2" value="$3"

  python3 -c "
import json, sys

alias     = sys.argv[1]
field     = sys.argv[2]
value     = sys.argv[3]
state_raw = sys.argv[4]

state = json.loads(state_raw)
remotes = state.get('remotes', {})

if alias not in remotes:
    print(f'ERROR: alias \"{alias}\" not found', file=sys.stderr)
    sys.exit(1)

# Auto-type for known numeric fields
if field in ('chain_id',):
    try:
        value = int(value, 16) if value.startswith('0x') else int(value)
    except ValueError:
        pass

remotes[alias][field] = value
state['remotes'] = remotes
print(json.dumps(state, indent=2))
" "$alias" "$field" "$value" "$(_cb_remote_load_state)"
}

# _cb_remote_update_last_checked <alias>
# Updates the last_checked timestamp for a remote entry.
_cb_remote_update_last_checked() {
  local alias="$1"
  local now
  now="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

  local new_state
  new_state="$(_cb_remote_update_field "$alias" "last_checked" "$now")" || return 1
  _cb_remote_save_state "$new_state"
}

# ---------------------------------------------------------------------------
# RPC connectivity check
# ---------------------------------------------------------------------------

# _cb_remote_check_connectivity <rpc_url> [auth_header]
# Makes an eth_chainId call to verify the endpoint is reachable.
# Prints the chain_id (hex) on success, empty on failure.
_cb_remote_check_connectivity() {
  local rpc_url="$1" auth_header="${2:-}"

  local -a curl_args=(
    -s --max-time 10
    -X POST
    -H "Content-Type: application/json"
    --data '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}'
  )

  if [[ -n "$auth_header" ]]; then
    curl_args+=(-H "Authorization: ${auth_header}")
  fi

  curl_args+=("$rpc_url")

  local response
  response="$(curl "${curl_args[@]}" 2>/dev/null)" || return 1

  python3 -c "
import json, sys
try:
    d = json.loads(sys.argv[1])
    result = d.get('result', '')
    if result and result.startswith('0x'):
        print(result)
    else:
        sys.exit(1)
except Exception:
    sys.exit(1)
" "$response"
}
