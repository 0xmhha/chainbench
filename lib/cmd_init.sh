#!/usr/bin/env bash
# lib/cmd_init.sh - 'chainbench init' subcommand handler
# Uses preset keys (static mode) by default for reproducible testing.
# Keys: chainbench/keys/preset/ with metadata.json
set -euo pipefail

# ---- Parse runtime overrides -------------------------------------------------
source "${CHAINBENCH_DIR}/lib/common.sh"
_CB_INIT_REMAINING=()
_cb_parse_runtime_overrides _CB_INIT_REMAINING "$@"
set -- "${_CB_INIT_REMAINING[@]+"${_CB_INIT_REMAINING[@]}"}"

# ---- Load dependencies -------------------------------------------------------
source "${CHAINBENCH_DIR}/lib/profile.sh"
source "${CHAINBENCH_DIR}/lib/pids_state.sh"
source "${CHAINBENCH_DIR}/lib/chain_adapter.sh"
load_profile "${CHAINBENCH_PROFILE}" || exit 1

# ---- Resolve binary ----------------------------------------------------------
BINARY=""
BINARY="$(resolve_binary "${CHAINBENCH_BINARY}" "${CHAINBENCH_BINARY_PATH:-}")" || {
  log_error "Cannot find '${CHAINBENCH_BINARY}' binary. Set binary_path in your profile or add it to \$PATH."
  exit 1
}
log_info "Using binary: ${BINARY}"

# Persist resolved binary path to merged JSON so that start/status can find it
python3 - "${CHAINBENCH_PROFILE_JSON}" "${BINARY}" <<'PYEOF'
import json, sys
with open(sys.argv[1]) as f:
    data = json.load(f)
data.setdefault("chain", {})["binary_path"] = sys.argv[2]
with open(sys.argv[1], "w") as f:
    json.dump(data, f, indent=2)
PYEOF

require_cmd python3 "python3 is required" || exit 1

# ---- Resolve data directory --------------------------------------------------
_DATA_DIR_RAW="${CHAINBENCH_DATA_DIR:-data}"
if [[ "${_DATA_DIR_RAW}" == /* ]]; then
  _INIT_DATA_DIR="${_DATA_DIR_RAW}"
else
  _INIT_DATA_DIR="${CHAINBENCH_DIR}/${_DATA_DIR_RAW}"
fi
_INIT_STATE_DIR="${CHAINBENCH_DIR}/state"
_INIT_GENESIS_TEMPLATE="${CHAINBENCH_DIR}/${CHAINBENCH_GENESIS_TEMPLATE:-templates/genesis.template.json}"
_INIT_NODE_TEMPLATE="${CHAINBENCH_DIR}/templates/node.template.toml"
_INIT_GENESIS_OUT="${_INIT_DATA_DIR}/genesis.json"

# ---- Resolve keys directory --------------------------------------------------
_KEYS_MODE="${CHAINBENCH_KEYS_MODE:-static}"
_KEYS_SOURCE="${CHAINBENCH_KEYS_SOURCE:-keys/preset}"
if [[ "${_KEYS_SOURCE}" != /* ]]; then
  _KEYS_SOURCE="${CHAINBENCH_DIR}/${_KEYS_SOURCE}"
fi
_METADATA="${_KEYS_SOURCE}/metadata.json"

if [[ "${_KEYS_MODE}" == "static" ]]; then
  if [[ ! -f "${_METADATA}" ]]; then
    log_error "Preset keys metadata not found: ${_METADATA}"
    log_error "Run with keys.mode=generate or provide preset keys in ${_KEYS_SOURCE}"
    exit 1
  fi
  log_info "Using preset keys from ${_KEYS_SOURCE} (static mode)"
fi

# ---- Validate templates -----------------------------------------------------
for _tpl in "${_INIT_GENESIS_TEMPLATE}" "${_INIT_NODE_TEMPLATE}"; do
  if [[ ! -f "${_tpl}" ]]; then
    log_error "Template not found: ${_tpl}"
    exit 1
  fi
done

# ---- Stop any running nodes first -------------------------------------------
if pids_exists; then
  _any_alive=0
  for _pid in $(pids_get_all_pids 2>/dev/null); do
    if kill -0 "$_pid" 2>/dev/null; then
      _any_alive=1
      break
    fi
  done
  if [[ "$_any_alive" -eq 1 ]]; then
    log_info "Running nodes detected, stopping first ..."
    pkill -15 "${CHAINBENCH_BINARY}" 2>/dev/null || true
    sleep 2
    pkill -9 "${CHAINBENCH_BINARY}" 2>/dev/null || true
    log_success "Existing nodes stopped"
  fi
fi

# ---- Clean existing data ----------------------------------------------------
log_info "Cleaning existing data under ${_INIT_DATA_DIR}/ ..."
# Remove all node data, configs, genesis, logs, keys
rm -rf "${_INIT_DATA_DIR}"/node* 2>/dev/null || true
rm -rf "${_INIT_DATA_DIR}"/keystores 2>/dev/null || true
rm -rf "${_INIT_DATA_DIR}"/nodekeys 2>/dev/null || true
rm -f "${_INIT_DATA_DIR}"/genesis.json 2>/dev/null || true
rm -f "${_INIT_DATA_DIR}"/config_node*.toml 2>/dev/null || true
rm -f "${_INIT_DATA_DIR}"/password 2>/dev/null || true
rm -f "${_INIT_DATA_DIR}"/logs/*.log* 2>/dev/null || true
# Clean state
rm -f "${_INIT_STATE_DIR}/pids.json" 2>/dev/null || true
rm -f "${_INIT_STATE_DIR}/current-profile.yaml" 2>/dev/null || true
rm -f "${_INIT_STATE_DIR}/current-profile-merged.json" 2>/dev/null || true
log_success "Existing data cleaned"

mkdir -p "${_INIT_DATA_DIR}/logs" "${_INIT_STATE_DIR}"

# ---- Load chain adapter for genesis/TOML generation -------------------------
_CHAIN_TYPE="${CHAINBENCH_CHAIN_TYPE:-stablenet}"
cb_adapter_load "$_CHAIN_TYPE" || {
  log_error "Failed to load chain adapter for type '${_CHAIN_TYPE}'"
  exit 1
}

# ---- Generate genesis.json using adapter ------------------------------------
log_info "Generating genesis.json from template + metadata (adapter: ${_CHAIN_TYPE}) ..."

adapter_generate_genesis \
  "${CHAINBENCH_PROFILE_JSON}" \
  "${_INIT_GENESIS_TEMPLATE}" \
  "${_INIT_GENESIS_OUT}" \
  "${_METADATA}" \
  "${CHAINBENCH_VALIDATORS}" \
  "${CHAINBENCH_BASE_P2P}" || {
  log_error "Genesis generation failed"
  exit 1
}

log_success "genesis.json written to ${_INIT_GENESIS_OUT}"

# ---- Generate TOML configs using adapter ------------------------------------
log_info "Generating node TOML configs for ${CHAINBENCH_TOTAL_NODES} node(s) ..."

adapter_generate_toml \
  "${_METADATA}" \
  "${_INIT_NODE_TEMPLATE}" \
  "${_INIT_DATA_DIR}" \
  "${CHAINBENCH_TOTAL_NODES}" \
  "${CHAINBENCH_VALIDATORS}" \
  "${CHAINBENCH_BASE_P2P}" \
  "${CHAINBENCH_BASE_HTTP}" \
  "${CHAINBENCH_BASE_WS}" \
  "${CHAINBENCH_BASE_AUTH}" \
  "${CHAINBENCH_BASE_METRICS}"

# ---- Copy keys from preset --------------------------------------------------
log_info "Setting up keys from preset ..."

_PASSWORD_FILE="${_KEYS_SOURCE}/password"
for (( _ni = 1; _ni <= CHAINBENCH_TOTAL_NODES; _ni++ )); do
  _node_key_src="${_KEYS_SOURCE}/node${_ni}"
  if [[ ! -d "${_node_key_src}" ]]; then
    log_warn "  Preset keys for node${_ni} not found, skipping"
    continue
  fi

  # Copy keystore
  _ks_dest="${_INIT_DATA_DIR}/keystores/node${_ni}"
  mkdir -p "${_ks_dest}"
  if [[ -d "${_node_key_src}/keystore" ]]; then
    cp "${_node_key_src}/keystore/"* "${_ks_dest}/" 2>/dev/null || true
  fi

  # Copy nodekey to data directory
  _nodekey_dest="${_INIT_DATA_DIR}/nodekeys/node${_ni}"
  mkdir -p "${_nodekey_dest}"
  if [[ -f "${_node_key_src}/nodekey" ]]; then
    cp "${_node_key_src}/nodekey" "${_nodekey_dest}/nodekey"
  fi

  log_info "  node${_ni}: keys copied"
done

# Copy password file
if [[ -f "${_PASSWORD_FILE}" ]]; then
  cp "${_PASSWORD_FILE}" "${_INIT_DATA_DIR}/password"
fi

# ---- Run gstable init -------------------------------------------------------
log_info "Initializing node datadirs ..."

for (( _ni = 1; _ni <= CHAINBENCH_TOTAL_NODES; _ni++ )); do
  _datadir="${_INIT_DATA_DIR}/node${_ni}"
  mkdir -p "${_datadir}"

  log_info "  node${_ni}: gstable init"
  if ! "${BINARY}" init --datadir "${_datadir}" "${_INIT_GENESIS_OUT}" > /dev/null 2>&1; then
    log_error "gstable init failed for node${_ni}"
    # Show error detail
    "${BINARY}" init --datadir "${_datadir}" "${_INIT_GENESIS_OUT}" 2>&1 || true
    exit 1
  fi
  log_success "  node${_ni} initialized"
done

# ---- Save state --------------------------------------------------------------
_profile_yaml="${CHAINBENCH_DIR}/profiles/${CHAINBENCH_PROFILE}.yaml"
[[ ! -f "${_profile_yaml}" ]] && _profile_yaml="${CHAINBENCH_DIR}/profiles/custom/${CHAINBENCH_PROFILE}.yaml"
[[ -f "${_profile_yaml}" ]] && cp "${_profile_yaml}" "${_INIT_STATE_DIR}/current-profile.yaml"
cp "${CHAINBENCH_PROFILE_JSON}" "${_INIT_STATE_DIR}/current-profile-merged.json"

# ---- Summary -----------------------------------------------------------------
log_success "============================================"
log_success "  chainbench init complete"
log_success "  Profile   : ${CHAINBENCH_PROFILE}"
log_success "  Keys      : ${_KEYS_MODE} (${_KEYS_SOURCE})"
log_success "  Validators: ${CHAINBENCH_VALIDATORS}"
log_success "  Endpoints : ${CHAINBENCH_ENDPOINTS}"
log_success "  Total     : ${CHAINBENCH_TOTAL_NODES} node(s)"
log_success "  Data      : ${_INIT_DATA_DIR}"
log_success "  Genesis   : ${_INIT_GENESIS_OUT}"
log_success "============================================"
log_info "Run 'chainbench start' to launch all nodes."
