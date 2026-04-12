#!/usr/bin/env bash
# tests/lib/failure_context.sh - Auto-capture chain state on test failure
#
# Called by test_result when fail > 0. Collects per-node block height,
# peer count, syncing status, recent blocks, and log tails into
# state/failures/<test_name>_<timestamp>/

[[ -n "${_CB_FAILURE_CTX_LOADED:-}" ]] && return 0
readonly _CB_FAILURE_CTX_LOADED=1

# _cb_capture_failure_context <test_name>
# Saves diagnostic snapshot to state/failures/ directory.
# Non-fatal: errors during capture are logged but do not affect test result.
_cb_capture_failure_context() {
  local test_name="${1:-unknown}"
  local pids_file="${CHAINBENCH_DIR}/state/pids.json"

  # Skip if no pids.json (chain not running via chainbench)
  if [[ ! -f "$pids_file" ]]; then
    return 0
  fi

  local safe_name ts ctx_dir
  safe_name=$(printf '%s' "$test_name" | tr -cs '[:alnum:]-_' '_' | tr '[:upper:]' '[:lower:]')
  ts=$(date +%Y%m%d_%H%M%S)
  ctx_dir="${CHAINBENCH_DIR}/state/failures/${safe_name}_${ts}"
  mkdir -p "$ctx_dir" 2>/dev/null || return 0

  python3 - "$pids_file" "$ctx_dir" <<'PYEOF' 2>/dev/null || true
import sys, json, subprocess, os

pids_file = sys.argv[1]
ctx_dir = sys.argv[2]

with open(pids_file) as f:
    data = json.load(f)

nodes = data.get("nodes", {})
context = {"nodes": {}, "recent_blocks": []}

def rpc_call(port, method, params=None):
    """Send a JSON-RPC call and return the result field."""
    if params is None:
        params = []
    try:
        r = subprocess.run(
            ["curl", "-s", "--max-time", "3", "-X", "POST",
             "-H", "Content-Type: application/json",
             "--data", json.dumps({"jsonrpc": "2.0", "method": method, "params": params, "id": 1}),
             f"http://127.0.0.1:{port}"],
            capture_output=True, text=True, timeout=5
        )
        if not r.stdout or r.returncode != 0:
            return "unreachable"
        resp = json.loads(r.stdout)
        return resp.get("result", "unreachable")
    except Exception:
        return "unreachable"

for key, node in sorted(nodes.items()):
    http_port = node.get("http_port")
    log_file = node.get("log_file", "")
    node_ctx = {"port": http_port, "type": node.get("type", "")}

    if http_port:
        node_ctx["eth_blockNumber"] = rpc_call(http_port, "eth_blockNumber")
        node_ctx["net_peerCount"] = rpc_call(http_port, "net_peerCount")
        node_ctx["eth_syncing"] = rpc_call(http_port, "eth_syncing")

    # Log tail
    if log_file and os.path.isfile(log_file):
        tail_path = os.path.join(ctx_dir, f"node{key}.log.tail")
        try:
            subprocess.run(["tail", "-200", log_file],
                           stdout=open(tail_path, "w"), timeout=5)
        except Exception:
            pass

    context["nodes"][key] = node_ctx

# Recent 5 blocks from first node
first_node = next(iter(sorted(nodes.items())), (None, {}))[1]
first_port = first_node.get("http_port") if first_node else None
if first_port:
    head_hex = rpc_call(first_port, "eth_blockNumber")
    if head_hex and head_hex != "unreachable":
        try:
            head = int(head_hex, 16)
            for i in range(max(0, head - 4), head + 1):
                blk = rpc_call(first_port, "eth_getBlockByNumber", [hex(i), False])
                if isinstance(blk, dict):
                    context["recent_blocks"].append({
                        "number": i,
                        "hash": blk.get("hash", ""),
                        "stateRoot": blk.get("stateRoot", ""),
                        "miner": blk.get("miner", "")
                    })
        except (ValueError, TypeError):
            pass

with open(os.path.join(ctx_dir, "context.json"), "w") as f:
    json.dump(context, f, indent=2)
PYEOF

  printf "${_ASSERT_YELLOW:-}[FAIL-CTX]${_ASSERT_RESET:-} Failure context saved to %s\n" "$ctx_dir" >&2
}
