#!/usr/bin/env bash
# tests/unit/tests/failure-context.sh - Test failure context auto-capture
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/assert.sh"

CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
export CHAINBENCH_DIR

TMPDIR_ROOT="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_ROOT"' EXIT

# Source the failure context module
source "${CHAINBENCH_DIR}/tests/lib/failure_context.sh"

# ---- Test 1: no pids.json -> graceful skip -----------------------------------
describe "failure_context: no pids.json skips gracefully"
FAKE_CB="${TMPDIR_ROOT}/nopids"
mkdir -p "${FAKE_CB}/state"
rc=0
(CHAINBENCH_DIR="$FAKE_CB" _cb_capture_failure_context "test/nopids" 2>/dev/null) || rc=$?
assert_eq "$rc" "0" "exits 0 when no pids.json"

# Verify no failures directory created
if [[ -d "${FAKE_CB}/state/failures" ]]; then
  found_dirs="$(find "${FAKE_CB}/state/failures" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | wc -l | tr -d ' ')"
else
  found_dirs="0"
fi
assert_eq "$found_dirs" "0" "no failure dir created without pids.json"

# ---- Test 2: mock pids.json -> context.json created --------------------------
describe "failure_context: creates context.json with mock pids"
FAKE_CB2="${TMPDIR_ROOT}/withpids"
mkdir -p "${FAKE_CB2}/state"

# Create mock pids.json (ports won't be reachable, that's fine)
cat > "${FAKE_CB2}/state/pids.json" <<'JSON'
{
  "nodes": {
    "1": {"pid": 99999, "type": "validator", "http_port": 59999, "log_file": "/nonexistent/node1.log"},
    "2": {"pid": 99998, "type": "endpoint", "http_port": 59998, "log_file": "/nonexistent/node2.log"}
  }
}
JSON

CHAINBENCH_DIR="$FAKE_CB2" _cb_capture_failure_context "test/with-pids" 2>/dev/null || true

# Find the created directory
ctx_dirs=($(ls -d "${FAKE_CB2}/state/failures"/*/ 2>/dev/null))
assert_nonempty "${ctx_dirs[0]:-}" "failure dir created"

if [[ -n "${ctx_dirs[0]:-}" ]]; then
  assert_file_exists "${ctx_dirs[0]}context.json" "context.json exists"

  # Verify context.json has nodes field
  has_nodes="$(python3 -c "import json; d=json.load(open('${ctx_dirs[0]}context.json')); print('yes' if 'nodes' in d else 'no')" 2>/dev/null || echo "no")"
  assert_eq "$has_nodes" "yes" "context.json contains nodes field"

  # Nodes should have unreachable RPC results (ports not listening)
  node1_block="$(python3 -c "import json; d=json.load(open('${ctx_dirs[0]}context.json')); print(d.get('nodes',{}).get('1',{}).get('eth_blockNumber',''))" 2>/dev/null)"
  assert_eq "$node1_block" "unreachable" "unreachable node recorded correctly"
fi

# ---- Test 3: directory naming convention -------------------------------------
describe "failure_context: directory naming convention"
dir_name="$(basename "${ctx_dirs[0]:-empty}")"
assert_contains "$dir_name" "test_with" "dir name contains safe test name"

unit_summary
