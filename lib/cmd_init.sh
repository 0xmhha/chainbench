#!/usr/bin/env bash
# lib/cmd_init.sh - 'chainbench init' subcommand handler
# Uses preset keys (static mode) by default for reproducible testing.
# Keys: chainbench/keys/preset/ with metadata.json
set -euo pipefail

# ---- Load profile ------------------------------------------------------------
source "${CHAINBENCH_DIR}/lib/profile.sh"
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
if [[ -f "${_INIT_STATE_DIR}/pids.json" ]]; then
  _any_alive=0
  for _pid in $(python3 -c "
import json
with open('${_INIT_STATE_DIR}/pids.json') as f:
    d = json.load(f)
for n in d.get('nodes',{}).values():
    if n.get('pid',0) > 0: print(n['pid'])
" 2>/dev/null); do
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

# ---- Generate genesis.json using metadata -----------------------------------
log_info "Generating genesis.json from template + metadata ..."

python3 - \
  "${CHAINBENCH_PROFILE_JSON}" \
  "${_INIT_GENESIS_TEMPLATE}" \
  "${_INIT_GENESIS_OUT}" \
  "${_METADATA}" \
  "${CHAINBENCH_VALIDATORS}" \
  "${CHAINBENCH_BASE_P2P}" <<'PYEOF'
import sys, json, os

profile_path   = sys.argv[1]
template_path  = sys.argv[2]
output_path    = sys.argv[3]
metadata_path  = sys.argv[4]
num_validators = int(sys.argv[5])
base_p2p       = int(sys.argv[6])

with open(profile_path) as f:
    profile = json.load(f)
with open(template_path) as f:
    template = f.read()
with open(metadata_path) as f:
    metadata = json.load(f)

chain = profile.get("chain", {})
overrides = profile.get("genesis", {}).get("overrides", {})
wbft = overrides.get("wbft", {})

chain_id = chain.get("chain_id", 8283)
request_timeout = wbft.get("requestTimeoutSeconds", 2)
block_period = wbft.get("blockPeriodSeconds", 1)
epoch_length = wbft.get("epochLength", 140)
proposer_policy = wbft.get("proposerPolicy", 0)
max_request_timeout = wbft.get("maxRequestTimeoutSeconds", None)

# Get validators and BLS keys from metadata (first N nodes)
meta_nodes = metadata["nodes"][:num_validators]
validators = [n["address"] for n in meta_nodes]
bls_keys = [n["blsPublicKey"] for n in meta_nodes]

# System contracts: use profile overrides, inject validator/member/bls info
sys_contracts = overrides.get("systemContracts", {})
members_csv = ",".join(validators)
bls_csv = ",".join(bls_keys)

# Inject into govValidator params
if "govValidator" in sys_contracts:
    params = sys_contracts["govValidator"].setdefault("params", {})
    params["validators"] = members_csv
    params["members"] = members_csv
    params["blsPublicKeys"] = bls_csv

# Inject members into other governance contracts
for contract_name in ["govMinter", "govMasterMinter", "govCouncil"]:
    if contract_name in sys_contracts:
        params = sys_contracts[contract_name].setdefault("params", {})
        params["members"] = members_csv

# Alloc: from metadata (all nodes get balance) + profile overrides
alloc_obj = {}
# Add all nodes from metadata
for n in metadata["nodes"]:
    addr = n["address"].replace("0x", "").replace("0X", "")
    alloc_obj[addr] = {"balance": "0x84595161401484a000000"}

# Add profile alloc overrides
for addr, balance in overrides.get("alloc", {}).items():
    clean = addr.strip('"').replace("0x", "").replace("0X", "")
    alloc_obj[clean] = {"balance": str(balance)}

# CREATE2 deployer
c2 = "4e59b44847b379578588920cA78FbF26c0B4956C"
if c2.lower() not in {k.lower() for k in alloc_obj}:
    alloc_obj[c2] = {
        "code": "0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe03601600081602082378035828234f58015156014578182fd5b80825250506014600cf3",
        "balance": "0x0"
    }

# ExtraData: from metadata (pre-computed for these validators)
extra_data = metadata.get("extraData", overrides.get("extraData", "0x00"))

# Substitute
result = template
result = result.replace("__CHAIN_ID__", str(chain_id))
result = result.replace("__REQUEST_TIMEOUT_SECONDS__", str(request_timeout))
result = result.replace("__BLOCK_PERIOD_SECONDS__", str(block_period))
result = result.replace("__EPOCH_LENGTH__", str(epoch_length))
result = result.replace("__PROPOSER_POLICY__", str(proposer_policy))
result = result.replace("__MAX_REQUEST_TIMEOUT_SECONDS__", "null" if max_request_timeout is None else str(max_request_timeout))
result = result.replace('"__VALIDATORS_JSON__"', json.dumps(validators))
result = result.replace("__VALIDATORS_JSON__", json.dumps(validators))
result = result.replace('"__BLS_PUBLIC_KEYS_JSON__"', json.dumps(bls_keys))
result = result.replace("__BLS_PUBLIC_KEYS_JSON__", json.dumps(bls_keys))
result = result.replace('"__SYSTEM_CONTRACTS_JSON__"', json.dumps(sys_contracts, indent=4))
result = result.replace("__SYSTEM_CONTRACTS_JSON__", json.dumps(sys_contracts, indent=4))
result = result.replace('"__ALLOC_JSON__"', json.dumps(alloc_obj, indent=4))
result = result.replace("__ALLOC_JSON__", json.dumps(alloc_obj, indent=4))
result = result.replace('"__EXTRA_DATA__"', f'"{extra_data}"')
result = result.replace("__EXTRA_DATA__", extra_data)

try:
    json.loads(result)
except json.JSONDecodeError as e:
    print(f"ERROR: Invalid genesis JSON: {e}", file=sys.stderr)
    sys.exit(1)

os.makedirs(os.path.dirname(output_path), exist_ok=True)
with open(output_path, "w") as f:
    f.write(result)
print(f"Genesis written to {output_path}")
PYEOF

log_success "genesis.json written to ${_INIT_GENESIS_OUT}"

# ---- Generate TOML configs with enode URLs from metadata --------------------
log_info "Generating node TOML configs for ${CHAINBENCH_TOTAL_NODES} node(s) ..."

python3 - \
  "${_METADATA}" \
  "${_INIT_NODE_TEMPLATE}" \
  "${_INIT_DATA_DIR}" \
  "${CHAINBENCH_TOTAL_NODES}" \
  "${CHAINBENCH_VALIDATORS}" \
  "${CHAINBENCH_BASE_P2P}" \
  "${CHAINBENCH_BASE_HTTP}" \
  "${CHAINBENCH_BASE_WS}" \
  "${CHAINBENCH_BASE_AUTH}" \
  "${CHAINBENCH_BASE_METRICS}" <<'PYEOF'
import sys, json, os

metadata_path = sys.argv[1]
template_path = sys.argv[2]
data_dir      = sys.argv[3]
total_nodes   = int(sys.argv[4])
num_validators = int(sys.argv[5])
base_p2p      = int(sys.argv[6])
base_http     = int(sys.argv[7])
base_ws       = int(sys.argv[8])
base_auth     = int(sys.argv[9])
base_metrics  = int(sys.argv[10])

with open(metadata_path) as f:
    metadata = json.load(f)
with open(template_path) as f:
    template = f.read()

# Build static nodes list from all nodes
meta_nodes = metadata["nodes"][:total_nodes]
static_entries = []
for n in meta_nodes:
    idx = n["index"]
    port = base_p2p + idx - 1
    enode = f'enode://{n["publicKey"]}@127.0.0.1:{port}?discport=0'
    static_entries.append(f'    "{enode}"')

static_nodes_str = ",\n".join(static_entries)
discovery_section = f'NoDiscovery = true\nStaticNodes = [\n{static_nodes_str}\n]'

for i in range(1, total_nodes + 1):
    idx = i - 1
    p2p_port = base_p2p + idx
    http_port = base_http + idx
    auth_port = base_auth + idx
    metrics_port = base_metrics + idx

    is_validator = i <= num_validators
    miner_section = '[Eth.Miner]\nRecommit = "2s"' if is_validator else ""

    keystore_dir = os.path.join(data_dir, "keystores", f"node{i}")
    ethstats_url = f"node{i}:local@localhost:3000"

    result = template
    result = result.replace("__SYNC_MODE__", "full")
    result = result.replace("__MINER_SECTION__", miner_section)
    result = result.replace("__KEYSTORE_DIR__", keystore_dir)
    result = result.replace("__AUTH_PORT__", str(auth_port))
    result = result.replace("__HTTP_HOST__", "0.0.0.0")
    result = result.replace("__HTTP_PORT__", str(http_port))
    result = result.replace("__P2P_PORT__", str(p2p_port))
    result = result.replace("__DISCOVERY_SECTION__", discovery_section)
    result = result.replace("__ETHSTATS_URL__", ethstats_url)
    result = result.replace("__METRICS_ENABLED__", "true")
    result = result.replace("__METRICS_HTTP__", "127.0.0.1")
    result = result.replace("__METRICS_PORT__", str(metrics_port))

    out_path = os.path.join(data_dir, f"config_node{i}.toml")
    with open(out_path, "w") as f:
        f.write(result)

    role = "validator" if is_validator else "endpoint"
    print(f"  config_node{i}.toml  [{role}]")
PYEOF

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
