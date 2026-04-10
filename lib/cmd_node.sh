#!/usr/bin/env bash
# lib/cmd_node.sh - Control individual chain nodes
# Sourced by the main chainbench dispatcher; $@ contains subcommand arguments.
#
# Sub-subcommands:
#   node stop  <N>
#   node start <N>
#   node log   <N> [--follow]
#   node rpc   <N> <method> [params_json]

# Guard against double-sourcing
[[ -n "${_CB_CMD_NODE_SH_LOADED:-}" ]] && return 0
readonly _CB_CMD_NODE_SH_LOADED=1

_CB_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${_CB_LIB_DIR}/common.sh"
source "${_CB_LIB_DIR}/remote_state.sh" 2>/dev/null || true

# ---- Constants ---------------------------------------------------------------

readonly _CB_NODE_PIDS_FILE="${CHAINBENCH_DIR}/state/pids.json"
readonly _CB_NODE_DATA_DIR="${CHAINBENCH_DIR}/data"
readonly _CB_NODE_RPC_TIMEOUT=10
readonly _CB_NODE_STOP_GRACE_SECONDS=10

# ---- Usage -------------------------------------------------------------------

_cb_node_usage() {
  cat >&2 <<'EOF'
Usage: chainbench node <subcommand> <N> [options]

Subcommands:
  stop  <N>                          Stop node N
  start <N>                          Re-start a stopped node N
  log   <N> [--follow]               Show log for node N (last 50 lines)
  rpc   <N> <method> [params]        Send a JSON-RPC call to node N
  rpc   --remote <alias> <method> [params]  Send a JSON-RPC call to a remote chain

<N> is a 1-based node index.
EOF
}

# ---- Validation --------------------------------------------------------------

# _cb_node_require_pids_file
# Exits with an error when pids.json is absent.
_cb_node_require_pids_file() {
  if [[ ! -f "$_CB_NODE_PIDS_FILE" ]]; then
    log_error "pids.json not found at $_CB_NODE_PIDS_FILE — has the chain been started?"
    return 1
  fi
}

# _cb_node_total_nodes
# Prints the total number of nodes recorded in pids.json.
_cb_node_total_nodes() {
  python3 - "$_CB_NODE_PIDS_FILE" <<'PYEOF'
import sys, json

with open(sys.argv[1]) as fh:
    data = json.load(fh)

nodes = data.get("nodes", {})
print(len(nodes))
PYEOF
}

# _cb_node_validate_index <N>
# Validates that N is a positive integer and within the valid node range.
# Sets _CB_NODE_KEY (string key matching pids.json nodes dict) on success.
_cb_node_validate_index() {
  local n="$1"

  if ! [[ "$n" =~ ^[1-9][0-9]*$ ]]; then
    log_error "Node number must be a positive integer, got: '$n'"
    return 1
  fi

  local total
  total="$(_cb_node_total_nodes)"

  if (( n > total )); then
    log_error "Node $n does not exist — only $total node(s) configured"
    return 1
  fi

  # pids.json stores nodes as {"1": {...}, "2": {...}} with 1-based string keys
  _CB_NODE_KEY="$n"
  return 0
}

# ---- pids.json accessors -----------------------------------------------------

# _cb_node_get_field <key> <field>
# Reads a single field from pids.json for the node with the given string key.
_cb_node_get_field() {
  local key="$1"
  local field="$2"

  python3 - "$_CB_NODE_PIDS_FILE" "$key" "$field" <<'PYEOF'
import sys, json

with open(sys.argv[1]) as fh:
    data = json.load(fh)

key   = sys.argv[2]
field = sys.argv[3]

nodes = data.get("nodes", {})
node = nodes.get(key)
if node is not None:
    val = node.get(field, "")
    print("" if val is None else val)
else:
    print("")
PYEOF
}

# _cb_node_update_pids <key> <field=value> [<field=value> ...]
# Patches pids.json for the node with the given string key.
_cb_node_update_pids() {
  local node_key="$1"
  shift
  local -a kvs=("$@")

  python3 - "$_CB_NODE_PIDS_FILE" "$node_key" "${kvs[@]}" <<'PYEOF'
import sys, json

pids_file = sys.argv[1]
node_key  = sys.argv[2]
kvs       = sys.argv[3:]

with open(pids_file) as fh:
    data = json.load(fh)

nodes = data.get("nodes", {})
if node_key in nodes:
    for kv in kvs:
        if "=" not in kv:
            continue
        key, _, value = kv.partition("=")
        # Attempt int coercion for numeric fields
        if value.isdigit():
            value = int(value)
        nodes[node_key][key] = value

with open(pids_file, "w") as fh:
    json.dump(data, fh, indent=2)
    fh.write("\n")
PYEOF
}

# ---- PID helpers -------------------------------------------------------------

_cb_node_pid_alive() {
  kill -0 "$1" 2>/dev/null
}

_cb_node_wait_exit() {
  local pid="$1"
  local timeout_s="$2"
  local elapsed=0

  while (( elapsed < timeout_s * 2 )); do
    if ! _cb_node_pid_alive "$pid"; then
      return 0
    fi
    sleep 0.5
    (( elapsed++ ))
  done

  _cb_node_pid_alive "$pid" && return 1 || return 0
}

# ---- node stop <N> -----------------------------------------------------------

_cb_node_cmd_stop() {
  local node_num="$1"

  _cb_node_require_pids_file || return 1

  local _CB_NODE_KEY
  _cb_node_validate_index "$node_num" || return 1

  local pid label status
  pid="$(_cb_node_get_field "$_CB_NODE_KEY" "pid")"
  label="$(_cb_node_get_field "$_CB_NODE_KEY" "label")"
  label="${label:-node${_CB_NODE_KEY}}"
  status="$(_cb_node_get_field "$_CB_NODE_KEY" "status")"

  if [[ "$status" != "running" ]]; then
    log_warn "Node $label is not running (status: $status)"
    return 0
  fi

  if [[ -z "$pid" ]]; then
    log_error "Node $label has no PID recorded"
    return 1
  fi

  if ! _cb_node_pid_alive "$pid"; then
    log_warn "Node $label (PID $pid) is already dead"
  else
    log_info "Sending SIGTERM to $label (PID $pid)"
    kill -TERM "$pid" 2>/dev/null || true

    if _cb_node_wait_exit "$pid" "$_CB_NODE_STOP_GRACE_SECONDS"; then
      log_success "$label stopped gracefully"
    else
      log_warn "$label did not exit within ${_CB_NODE_STOP_GRACE_SECONDS}s, sending SIGKILL"
      kill -KILL "$pid" 2>/dev/null || true
      _cb_node_wait_exit "$pid" 3 || log_error "Could not kill $label (PID $pid)"
    fi
  fi

  local stopped_at
  stopped_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

  _cb_node_update_pids "$_CB_NODE_KEY" \
    "status=stopped" \
    "stop_reason=manual" \
    "stopped_at=${stopped_at}"

  log_success "pids.json updated for $label"
  return 0
}

# ---- node start <N> ----------------------------------------------------------

# _cb_node_build_start_args <key>
# Reconstructs the gstable launch arguments from pids.json for the given node key.
# Prints the binary path and all arguments, one per line (newline-separated).
# NOTE: NUL-separation is unusable here because bash command substitution
# strips NUL bytes. gstable arguments never contain newlines, so \n is safe.
_cb_node_build_start_args() {
  local key="$1"

  python3 - "$_CB_NODE_PIDS_FILE" "$key" "$_CB_NODE_DATA_DIR" <<'PYEOF'
import sys, json, os

pids_file = sys.argv[1]
node_key  = sys.argv[2]
data_dir  = sys.argv[3]

with open(pids_file) as fh:
    data = json.load(fh)

nodes = data.get("nodes", {})
if node_key not in nodes:
    print("", end="")
    sys.exit(1)

node = nodes[node_key]

# If the node has a saved_args field, use it verbatim
saved = node.get("saved_args", [])
if saved:
    print("\n".join(saved))
    sys.exit(0)

# Otherwise reconstruct from individual port/path fields
binary   = node.get("binary",    "gstable")
datadir  = node.get("datadir",   os.path.join(data_dir, f"node{node_key}"))
http_port= node.get("http_port", "")
ws_port  = node.get("ws_port",   "")
p2p_port = node.get("p2p_port",  "")
auth_port= node.get("auth_port", "")
network_id = node.get("network_id", "")
verbosity  = node.get("verbosity",  "3")
gcmode     = node.get("gcmode",     "full")
cache      = node.get("cache",      "1024")

args = [binary, "--datadir", datadir]

if http_port:
    args += ["--http", "--http.port", str(http_port),
             "--http.api", "eth,net,web3,debug,txpool"]
if ws_port:
    args += ["--ws", "--ws.port", str(ws_port),
             "--ws.api", "eth,net,web3"]
if p2p_port:
    args += ["--port", str(p2p_port)]
if auth_port:
    args += ["--authrpc.port", str(auth_port)]
if network_id:
    args += ["--networkid", str(network_id)]

args += [
    "--verbosity", str(verbosity),
    "--gcmode",    gcmode,
    "--cache",     str(cache),
]

extra = node.get("extra_flags", [])
if isinstance(extra, list):
    args.extend(extra)
elif extra:
    args.extend(extra.split())

print("\n".join(args))
PYEOF
}

_cb_node_cmd_start() {
  local node_num="$1"

  _cb_node_require_pids_file || return 1

  local _CB_NODE_KEY
  _cb_node_validate_index "$node_num" || return 1

  local label status
  label="$(_cb_node_get_field "$_CB_NODE_KEY" "label")"
  label="${label:-node${_CB_NODE_KEY}}"
  status="$(_cb_node_get_field "$_CB_NODE_KEY" "status")"

  if [[ "$status" == "running" ]]; then
    local existing_pid
    existing_pid="$(_cb_node_get_field "$_CB_NODE_KEY" "pid")"
    if _cb_node_pid_alive "${existing_pid:-0}"; then
      log_warn "Node $label is already running (PID $existing_pid)"
      return 0
    fi
    log_warn "Node $label has status 'running' but PID $existing_pid is dead; restarting"
  fi

  # Build command arguments
  local raw_args
  raw_args="$(_cb_node_build_start_args "$_CB_NODE_KEY")" || {
    log_error "Failed to reconstruct start arguments for $label"
    return 1
  }

  if [[ -z "$raw_args" ]]; then
    log_error "No saved arguments found for $label in pids.json"
    return 1
  fi

  # Split newline-separated args into an array
  local -a launch_args=()
  while IFS= read -r arg; do
    [[ -n "$arg" ]] && launch_args+=( "$arg" )
  done <<< "$raw_args"

  if (( ${#launch_args[@]} == 0 )); then
    log_error "Empty argument list after parsing start args for $label"
    return 1
  fi

  local binary="${launch_args[0]}"
  # If binary is not an absolute executable path, resolve it via common.sh
  # (handles: short name like "gstable", relative path, or missing binary in pids.json).
  if [[ "$binary" != /* ]] || [[ ! -x "$binary" ]]; then
    local resolved
    resolved="$(resolve_binary "$binary" "${CHAINBENCH_BINARY_PATH:-}")" || {
      log_error "Cannot resolve binary '$binary' for $label"
      return 1
    }
    launch_args[0]="$resolved"
    binary="$resolved"
  fi

  # Determine log file: prefer the log_file field saved by cmd_start.sh in pids.json,
  # which respects the profile's data.directory and logging.directory. Fall back to a
  # sane default only if the field is missing (older pids.json formats).
  local log_file
  log_file="$(_cb_node_get_field "$_CB_NODE_KEY" "log_file")"
  if [[ -z "$log_file" ]]; then
    log_file="${CHAINBENCH_DIR}/data/logs/${label}.log"
  fi
  mkdir -p "$(dirname "$log_file")"

  log_info "Starting $label: ${launch_args[*]}"
  "${launch_args[@]}" >>"$log_file" 2>&1 &
  local new_pid=$!

  # Brief wait to detect immediate crashes
  sleep 0.3
  if ! _cb_node_pid_alive "$new_pid"; then
    log_error "$label exited immediately — check $log_file"
    return 1
  fi

  local started_at
  started_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

  _cb_node_update_pids "$_CB_NODE_KEY" \
    "pid=${new_pid}" \
    "status=running" \
    "started_at=${started_at}"

  log_success "$label started (PID $new_pid)"
  return 0
}

# ---- node log <N> [--follow] -------------------------------------------------

_cb_node_cmd_log() {
  local node_num="$1"
  local follow=0

  shift
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --follow|-f) follow=1; shift ;;
      *)           log_warn "Unknown log option: $1"; shift ;;
    esac
  done

  _cb_node_require_pids_file || return 1

  local _CB_NODE_KEY
  _cb_node_validate_index "$node_num" || return 1

  local label
  label="$(_cb_node_get_field "$_CB_NODE_KEY" "label")"
  label="${label:-node${_CB_NODE_KEY}}"

  local log_file="${CHAINBENCH_DIR}/data/logs/${label}.log"

  if [[ ! -f "$log_file" ]]; then
    log_error "Log file not found: $log_file"
    return 1
  fi

  if [[ "$follow" == "1" ]]; then
    tail -f "$log_file"
  else
    tail -50 "$log_file"
  fi
}

# ---- node rpc <N> <method> [params_json] -------------------------------------

_cb_node_cmd_rpc() {
  local node_num="$1"
  local method="${2:?node rpc requires a method name}"
  local params="${3:-[]}"

  _cb_node_require_pids_file || return 1

  local _CB_NODE_KEY
  _cb_node_validate_index "$node_num" || return 1

  local http_port status label
  http_port="$(_cb_node_get_field "$_CB_NODE_KEY" "http_port")"
  status="$(_cb_node_get_field "$_CB_NODE_KEY" "status")"
  label="$(_cb_node_get_field "$_CB_NODE_KEY" "label")"
  label="${label:-node${_CB_NODE_KEY}}"

  if [[ -z "$http_port" ]]; then
    log_error "Node $label has no HTTP port configured"
    return 1
  fi

  if [[ "$status" != "running" ]]; then
    log_warn "Node $label is not running (status: $status)"
  fi

  local response
  response="$(curl -s --max-time "$_CB_NODE_RPC_TIMEOUT" \
    -X POST \
    -H "Content-Type: application/json" \
    --data "{\"jsonrpc\":\"2.0\",\"method\":\"${method}\",\"params\":${params},\"id\":1}" \
    "http://127.0.0.1:${http_port}")" || {
    log_error "RPC call failed for $label on port $http_port"
    return 1
  }

  if [[ -z "$response" ]]; then
    log_error "Empty response from $label"
    return 1
  fi

  printf '%s\n' "$response"
  return 0
}

# ---- node rpc --remote <alias> <method> [params_json] ------------------------

_cb_node_cmd_rpc_remote() {
  local alias="$1"
  local method="${2:?node rpc --remote requires a method name}"
  local params="${3:-[]}"

  if ! _cb_remote_exists "$alias" 2>/dev/null; then
    log_error "Remote alias '${alias}' not found"
    return 1
  fi

  local rpc_url auth_header
  rpc_url="$(_cb_remote_get_url "$alias")"
  auth_header="$(_cb_remote_get_auth_header "$alias" 2>/dev/null || echo "")"

  local -a curl_args=(
    -s --max-time "$_CB_NODE_RPC_TIMEOUT"
    -X POST
    -H "Content-Type: application/json"
    --data "{\"jsonrpc\":\"2.0\",\"method\":\"${method}\",\"params\":${params},\"id\":1}"
  )
  if [[ -n "$auth_header" ]]; then
    curl_args+=(-H "Authorization: ${auth_header}")
  fi
  curl_args+=("$rpc_url")

  local response
  response="$(curl "${curl_args[@]}" 2>/dev/null)" || {
    log_error "RPC call failed for remote '${alias}' at ${rpc_url}"
    return 1
  }

  if [[ -z "$response" ]]; then
    log_error "Empty response from remote '${alias}'"
    return 1
  fi

  printf '%s\n' "$response"
  return 0
}

# ---- Subcommand dispatcher ---------------------------------------------------

cmd_node_main() {
  if [[ $# -lt 1 ]]; then
    _cb_node_usage
    return 1
  fi

  local subcmd="$1"
  shift

  case "$subcmd" in
    stop)
      if [[ "${1:-}" == "--remote" ]]; then
        log_error "'node stop' is not available for remote chains — remote nodes cannot be controlled"
        return 1
      fi
      if [[ $# -lt 1 ]]; then
        log_error "Usage: chainbench node stop <N>"
        return 1
      fi
      _cb_node_cmd_stop "$1"
      ;;
    start)
      if [[ "${1:-}" == "--remote" ]]; then
        log_error "'node start' is not available for remote chains — remote nodes cannot be controlled"
        return 1
      fi
      if [[ $# -lt 1 ]]; then
        log_error "Usage: chainbench node start <N>"
        return 1
      fi
      _cb_node_cmd_start "$1"
      ;;
    log)
      if [[ "${1:-}" == "--remote" ]]; then
        log_error "'node log' is not available for remote chains — no local log files"
        return 1
      fi
      if [[ $# -lt 1 ]]; then
        log_error "Usage: chainbench node log <N> [--follow]"
        return 1
      fi
      _cb_node_cmd_log "$@"
      ;;
    rpc)
      # Check for --remote flag
      if [[ "${1:-}" == "--remote" ]]; then
        local remote_alias="${2:?--remote requires an alias}"
        shift 2
        if [[ $# -lt 1 ]]; then
          log_error "Usage: chainbench node rpc --remote <alias> <method> [params_json]"
          return 1
        fi
        _cb_node_cmd_rpc_remote "$remote_alias" "$@"
        return $?
      fi
      if [[ $# -lt 2 ]]; then
        log_error "Usage: chainbench node rpc <N> <method> [params_json]"
        return 1
      fi
      _cb_node_cmd_rpc "$@"
      ;;
    --help|-h|help)
      _cb_node_usage
      return 0
      ;;
    *)
      log_error "Unknown node subcommand: '$subcmd'"
      _cb_node_usage
      return 1
      ;;
  esac
}

# ---- Entry point -------------------------------------------------------------

cmd_node_main "$@"
