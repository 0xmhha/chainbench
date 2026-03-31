#!/usr/bin/env bash
# lib/cmd_start.sh - 'chainbench start' subcommand handler
# Sourced by the main chainbench script. Inherits: CHAINBENCH_DIR, CHAINBENCH_PROFILE,
# CHAINBENCH_QUIET, and all functions from lib/common.sh.
set -euo pipefail

# ---- Paths -------------------------------------------------------------------

readonly _START_STATE_DIR="${CHAINBENCH_DIR}/state"
readonly _START_CURRENT_PROFILE="${_START_STATE_DIR}/current-profile.yaml"
readonly _START_MERGED_PROFILE="${_START_STATE_DIR}/current-profile-merged.json"
readonly _START_PIDS_FILE="${_START_STATE_DIR}/pids.json"

# ---- Guard: must be initialized first ----------------------------------------

if [[ ! -f "${_START_CURRENT_PROFILE}" ]]; then
  log_error "Chain is not initialized. Run 'chainbench init --profile <name>' first."
  exit 1
fi

# ---- Guard: already running --------------------------------------------------

if [[ -f "${_START_PIDS_FILE}" ]]; then
  _start_any_alive=0
  _start_pids_check=""
  _start_pids_check="$(python3 - "${_START_PIDS_FILE}" <<'PYEOF'
import sys, json
with open(sys.argv[1]) as fh:
    data = json.load(fh)
nodes = data.get("nodes", {})
pids = [str(n["pid"]) for n in nodes.values() if "pid" in n]
print(" ".join(pids))
PYEOF
  )"

  _start_live_pid=""
  for _start_pid in ${_start_pids_check}; do
    if kill -0 "${_start_pid}" 2>/dev/null; then
      _start_any_alive=1
      _start_live_pid="${_start_pid}"
      break
    fi
  done

  if [[ "${_start_any_alive}" -eq 1 ]]; then
    log_error "Nodes appear to be running (found live PID ${_start_live_pid} in ${_START_PIDS_FILE})."
    log_error "Run 'chainbench stop' first, or remove ${_START_PIDS_FILE} manually."
    exit 1
  fi

  log_warn "Stale pids.json found (all processes dead). Removing and continuing."
  rm -f "${_START_PIDS_FILE}"
fi

# ---- Load profile from saved state -------------------------------------------

# shellcheck source=lib/profile.sh
source "${CHAINBENCH_DIR}/lib/profile.sh"

# Use the merged JSON saved by init when available to skip re-parsing YAML.
if [[ -f "${_START_MERGED_PROFILE}" ]]; then
  _cb_export_profile_vars "${_START_MERGED_PROFILE}" "${CHAINBENCH_PROFILE}"
  log_info "Profile loaded from saved state: ${_START_MERGED_PROFILE}"
else
  load_profile "${CHAINBENCH_PROFILE}" || exit 1
fi

# ---- Resolve data directory (needs profile loaded) --------------------------

_DATA_DIR_RAW="${CHAINBENCH_DATA_DIR:-data}"
if [[ "${_DATA_DIR_RAW}" == /* ]]; then
  _START_DATA_DIR="${_DATA_DIR_RAW}"
else
  _START_DATA_DIR="${CHAINBENCH_DIR}/${_DATA_DIR_RAW}"
fi

# ---- Resolve binary ----------------------------------------------------------

BINARY=""
BINARY="$(resolve_binary "${CHAINBENCH_BINARY}" "${CHAINBENCH_BINARY_PATH:-}")" || {
  log_error "Cannot find '${CHAINBENCH_BINARY}' binary."
  exit 1
}
log_info "Using binary: ${BINARY}"

# ---- Prepare directories -----------------------------------------------------

_LOG_DIR_RAW="${CHAINBENCH_LOG_DIR:-data/logs}"
if [[ "${_LOG_DIR_RAW}" == /* ]]; then
  LOG_DIR="${_LOG_DIR_RAW}"
else
  LOG_DIR="${_START_DATA_DIR}/logs"
fi
mkdir -p "${LOG_DIR}"

# Keys are copied to data dir during init, always use the copied location
KEYSTORE_DIR="${_START_DATA_DIR}/keystores"
NODEKEY_DIR="${_START_DATA_DIR}/nodekeys"

# ---- Port helper (delegates to common.sh) ------------------------------------
# Uses get_node_port from common.sh (already sourced via profile.sh)

# ---- Node launch function ----------------------------------------------------

# Stores launched node info as "pid:type:p2p:http:ws:auth:metrics" indexed by node number.
declare -A _start_node_info

_start_launch_node() {
  local node_index="$1"   # 1-based
  local node_type="$2"    # "validator" | "endpoint"

  local zero_index=$(( node_index - 1 ))

  local p2p_port;      p2p_port="$(get_node_port "${CHAINBENCH_BASE_P2P}"      "${zero_index}")"
  local http_port;    http_port="$(get_node_port "${CHAINBENCH_BASE_HTTP}"     "${zero_index}")"
  local ws_port;        ws_port="$(get_node_port "${CHAINBENCH_BASE_WS}"       "${zero_index}")"
  local auth_port;    auth_port="$(get_node_port "${CHAINBENCH_BASE_AUTH}"     "${zero_index}")"
  local metrics_port; metrics_port="$(get_node_port "${CHAINBENCH_BASE_METRICS}" "${zero_index}")"

  local node_datadir="${_START_DATA_DIR}/node${node_index}"
  local node_keystore="${KEYSTORE_DIR}/node${node_index}"
  local node_nodekey="${NODEKEY_DIR}/node${node_index}/nodekey"
  local node_config="${_START_DATA_DIR}/config_node${node_index}.toml"
  local node_log="${LOG_DIR}/node${node_index}.log"

  if [[ ! -d "${node_datadir}" ]]; then
    log_error "Datadir for node${node_index} not found: ${node_datadir}"
    log_error "Run 'chainbench init' first."
    exit 1
  fi

  if [[ ! -f "${node_config}" ]]; then
    log_error "Config for node${node_index} not found: ${node_config}"
    exit 1
  fi

  local verbosity="${CHAINBENCH_VERBOSITY:-3}"
  if [[ "${node_type}" == "endpoint" ]]; then
    verbosity="${CHAINBENCH_EN_VERBOSITY:-${CHAINBENCH_VERBOSITY:-3}}"
  fi

  local gcmode="${CHAINBENCH_GCMODE:-full}"
  local cache="${CHAINBENCH_CACHE:-1024}"

  local -a extra_flags_arr=()
  local _ef="${CHAINBENCH_EXTRA_FLAGS:-}"
  # Strip YAML artifacts: "[]", "None", empty
  _ef="${_ef//\[/}"
  _ef="${_ef//\]/}"
  _ef="${_ef//,/ }"
  _ef="$(echo "${_ef}" | xargs)"  # trim whitespace
  if [[ -n "${_ef}" && "${_ef}" != "None" ]]; then
    # shellcheck disable=SC2206
    extra_flags_arr=( ${_ef} )
  fi

  local -a launch_cmd=(
    "${BINARY}"
    --datadir    "${node_datadir}"
    --identity   "node${node_index}"
    --allow-insecure-unlock
    --rpc.enabledeprecatedpersonal
    --ws
    --ws.addr    "0.0.0.0"
    --ws.port    "${ws_port}"
    --ws.origins "*"
    --config     "${node_config}"
    --verbosity  "${verbosity}"
    --gcmode     "${gcmode}"
    --netrestrict "127.0.0.0/8"
    --cache      "${cache}"
    --rpc.allow-unprotected-txs
  )

  if [[ -d "${node_keystore}" ]]; then
    launch_cmd+=( --keystore "${node_keystore}" )
  fi

  if [[ -f "${node_nodekey}" ]]; then
    launch_cmd+=( --nodekey "${node_nodekey}" )
  fi

  if [[ "${node_type}" == "validator" ]]; then
    launch_cmd+=( --mine )
  fi

  if [[ ${#extra_flags_arr[@]} -gt 0 ]]; then
    launch_cmd+=( "${extra_flags_arr[@]}" )
  fi

  log_info "  Starting node${node_index} [${node_type}] -> log: ${node_log}"

  # Launch node with logrot for log rotation if available.
  # Strategy: redirect to file, launch logrot as a separate background watcher.
  local _logrot_bin="${CHAINBENCH_DIR}/bin/logrot"
  local _log_max_size="${CHAINBENCH_LOG_MAX_SIZE:-10M}"
  local _log_max_files="${CHAINBENCH_LOG_MAX_FILES:-5}"

  nohup "${launch_cmd[@]}" >> "${node_log}" 2>&1 &
  local node_pid=$!
  disown "${node_pid}" 2>/dev/null || true

  # Store as "pid|type|p2p|http|ws|auth|metrics" (pipe separator avoids colon conflicts)
  _start_node_info["${node_index}"]="${node_pid}|${node_type}|${p2p_port}|${http_port}|${ws_port}|${auth_port}|${metrics_port}"

  log_info "  node${node_index} launched with PID ${node_pid}"
}

# ---- Launch all nodes --------------------------------------------------------

log_info "Starting ${CHAINBENCH_TOTAL_NODES} node(s) ..."

for (( _ni = 1; _ni <= CHAINBENCH_VALIDATORS; _ni++ )); do
  _start_launch_node "${_ni}" "validator"
done

for (( _ni = CHAINBENCH_VALIDATORS + 1; _ni <= CHAINBENCH_TOTAL_NODES; _ni++ )); do
  _start_launch_node "${_ni}" "endpoint"
done

unset -f _start_launch_node

# ---- Write pids.json ---------------------------------------------------------

_start_timestamp="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
_start_chain_id="local-${CHAINBENCH_PROFILE}-$(date -u +"%Y%m%d%H%M%S")"

# Build a newline-separated node-info string to pass to Python.
_start_nodes_payload=""
for _ni in "${!_start_node_info[@]}"; do
  _start_nodes_payload+="${_ni}|${_start_node_info[${_ni}]}"$'\n'
done

python3 - \
  "${_START_PIDS_FILE}" \
  "${_start_chain_id}" \
  "${CHAINBENCH_PROFILE}" \
  "${_start_timestamp}" \
  "${LOG_DIR}" \
  "${_start_nodes_payload}" \
  <<'PYEOF'
import sys, json, os

OUTPUT_PATH   = sys.argv[1]
chain_id      = sys.argv[2]
profile_name  = sys.argv[3]
started_at    = sys.argv[4]
log_dir       = sys.argv[5]
nodes_raw     = sys.argv[6]   # "idx|pid|type|p2p|http|ws|auth|metrics\n..."

nodes = {}
for line in nodes_raw.strip().splitlines():
    line = line.strip()
    if not line:
        continue
    # format: idx|pid|type|p2p|http|ws|auth|metrics
    parts = line.split("|")
    if len(parts) < 8:
        continue
    idx        = parts[0]
    pid        = int(parts[1])
    ntype      = parts[2]
    p2p        = int(parts[3])
    http_p     = int(parts[4])
    ws_p       = int(parts[5])
    auth_p     = int(parts[6])
    metrics_p  = int(parts[7])
    log_file   = os.path.join(log_dir, f"node{idx}.log")

    nodes[idx] = {
        "pid":          pid,
        "type":         ntype,
        "p2p_port":     p2p,
        "http_port":    http_p,
        "ws_port":      ws_p,
        "auth_port":    auth_p,
        "metrics_port": metrics_p,
        "status":       "running",
        "log_file":     log_file,
    }

doc = {
    "chain_id":   chain_id,
    "profile":    profile_name,
    "started_at": started_at,
    "nodes":      nodes,
}

with open(OUTPUT_PATH, "w") as fh:
    json.dump(doc, fh, indent=2)
    fh.write("\n")

print(f"pids.json written with {len(nodes)} node(s)")
PYEOF

log_info "Waiting 3 seconds for processes to settle ..."
sleep 3

# ---- Verify all PIDs are alive -----------------------------------------------

_start_all_alive=1
_start_alive_count=0
_start_dead_nodes=()

for _ni in "${!_start_node_info[@]}"; do
  _entry="${_start_node_info[${_ni}]}"
  _pid="${_entry%%|*}"
  if kill -0 "${_pid}" 2>/dev/null; then
    (( ++_start_alive_count )) || true
  else
    _start_all_alive=0
    _start_dead_nodes+=( "node${_ni}(pid=${_pid})" )
    log_error "  node${_ni} (PID ${_pid}) is NOT running - check log: ${LOG_DIR}/node${_ni}.log"
  fi
done

# ---- Print status summary ----------------------------------------------------

log_success "============================================"
log_success "  chainbench start complete"
log_success "  Profile   : ${CHAINBENCH_PROFILE}"
log_success "  Total     : ${CHAINBENCH_TOTAL_NODES} node(s)"
log_success "  Alive     : ${_start_alive_count}/${CHAINBENCH_TOTAL_NODES}"

for (( _ni = 1; _ni <= CHAINBENCH_TOTAL_NODES; _ni++ )); do
  if [[ -n "${_start_node_info[${_ni}]:-}" ]]; then
    _entry="${_start_node_info[${_ni}]}"
    IFS='|' read -r _pid _type _p2p _http _ws _auth _metrics <<< "${_entry}"

    _node_status="running"
    if ! kill -0 "${_pid}" 2>/dev/null; then
      _node_status="DEAD"
    fi

    log_success "  node${_ni}  [${_type}]  PID=${_pid}  p2p=${_p2p}  http=${_http}  ws=${_ws}  [${_node_status}]"
  fi
done

log_success "  Logs dir  : ${LOG_DIR}"
log_success "  State     : ${_START_PIDS_FILE}"
log_success "============================================"

if [[ "${_start_all_alive}" -eq 0 ]]; then
  log_error "Some nodes failed to start: ${_start_dead_nodes[*]}"
  log_error "Check logs in ${LOG_DIR} for details."
  exit 1
fi

log_info "Run 'chainbench status' to check node health."
