#!/usr/bin/env bash
# lib/adapters/stablenet.sh - Adapter for go-stablenet (gstable) with WBFT/Istanbul consensus
# Implements the chain adapter interface for genesis generation, TOML config,
# start flags, and log parsing.

# Guard against double-sourcing
[[ -n "${_CB_ADAPTER_STABLENET_LOADED:-}" ]] && return 0
readonly _CB_ADAPTER_STABLENET_LOADED=1

# ---- Genesis generation ------------------------------------------------------

# adapter_generate_genesis <profile_json> <template_file> <output_file> <metadata_file> <num_validators> <base_p2p>
# Generate genesis.json for go-stablenet WBFT consensus.
adapter_generate_genesis() {
  local profile_json="$1" template_file="$2" output_file="$3" metadata_file="$4"
  local num_validators="$5" base_p2p="$6"

  python3 - \
    "$profile_json" \
    "$template_file" \
    "$output_file" \
    "$metadata_file" \
    "$num_validators" \
    "$base_p2p" <<'PYEOF'
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

meta_nodes = metadata["nodes"][:num_validators]
validators = [n["address"] for n in meta_nodes]
bls_keys = [n["blsPublicKey"] for n in meta_nodes]

sys_contracts = overrides.get("systemContracts", {})
members_csv = ",".join(validators)
bls_csv = ",".join(bls_keys)

if "govValidator" in sys_contracts:
    params = sys_contracts["govValidator"].setdefault("params", {})
    params["validators"] = members_csv
    params["members"] = members_csv
    params["blsPublicKeys"] = bls_csv

for contract_name in ["govMinter", "govMasterMinter", "govCouncil"]:
    if contract_name in sys_contracts:
        params = sys_contracts[contract_name].setdefault("params", {})
        params["members"] = members_csv

alloc_obj = {}
for n in metadata["nodes"]:
    addr = n["address"].replace("0x", "").replace("0X", "")
    alloc_obj[addr] = {"balance": "0x84595161401484a000000"}

for addr, balance in overrides.get("alloc", {}).items():
    clean = addr.strip('"').replace("0x", "").replace("0X", "")
    alloc_obj[clean] = {"balance": str(balance)}

c2 = "4e59b44847b379578588920cA78FbF26c0B4956C"
if c2.lower() not in {k.lower() for k in alloc_obj}:
    alloc_obj[c2] = {
        "code": "0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe03601600081602082378035828234f58015156014578182fd5b80825250506014600cf3",
        "balance": "0x0"
    }

extra_data = metadata.get("extraData", overrides.get("extraData", "0x00"))

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
}

# ---- TOML config generation --------------------------------------------------

# adapter_generate_toml <metadata_file> <template_file> <data_dir> <total_nodes> <num_validators> <base_p2p> <base_http> <base_ws> <base_auth> <base_metrics>
# Generate per-node TOML configuration files.
adapter_generate_toml() {
  local metadata_file="$1" template_file="$2" data_dir="$3"
  local total_nodes="$4" num_validators="$5"
  local base_p2p="$6" base_http="$7" base_ws="$8" base_auth="$9" base_metrics="${10}"

  python3 - \
    "$metadata_file" "$template_file" "$data_dir" \
    "$total_nodes" "$num_validators" \
    "$base_p2p" "$base_http" "$base_ws" "$base_auth" "$base_metrics" <<'PYEOF'
import sys, json, os

metadata_path  = sys.argv[1]
template_path  = sys.argv[2]
data_dir       = sys.argv[3]
total_nodes    = int(sys.argv[4])
num_validators = int(sys.argv[5])
base_p2p       = int(sys.argv[6])
base_http      = int(sys.argv[7])
base_ws        = int(sys.argv[8])
base_auth      = int(sys.argv[9])
base_metrics   = int(sys.argv[10])

with open(metadata_path) as f:
    metadata = json.load(f)
with open(template_path) as f:
    template = f.read()

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
}

# ---- Start flags -------------------------------------------------------------

# adapter_extra_start_flags <node_id> <role>
# Return extra gstable start flags as a space-separated string.
# role: "validator" | "endpoint"
adapter_extra_start_flags() {
  local node_id="$1" role="$2"
  local flags="--allow-insecure-unlock --rpc.enabledeprecatedpersonal --rpc.allow-unprotected-txs"
  if [[ "$role" == "validator" ]]; then
    flags+=" --mine"
  fi
  printf '%s\n' "$flags"
}

# ---- Consensus RPC namespace -------------------------------------------------

# adapter_consensus_rpc_namespace
# Return the RPC namespace for consensus queries.
adapter_consensus_rpc_namespace() {
  printf 'istanbul\n'
}
