#!/usr/bin/env bash
# tests/unit/tests/network-hybrid-capabilities.sh
# Covers a HYBRID network (mixed local+remote providers) end-to-end through the
# real chainbench-net binary via lib/network_client.sh.
#
# Unlike tests/unit/tests/cmd-test-capabilities.sh (which mocks cb_net_call and
# exercises only the gating decision logic with hand-written cap lists), this
# test feeds a real mixed-provider state file to the real binary and asserts:
#
#   A. The intersection runs end-to-end: a 3×local + 1×remote network resolves
#      to capabilities ["rpc","ws"] (the conservative lower bound).
#   B. Wired to that real result, the gating helper SKIPs a process-requiring
#      test (process is not in the hybrid lower bound).
#   C. ...and RUNs an rpc-requiring test (rpc IS in the lower bound).
#
# network.capabilities is a pure state-file read (no RPC dial), so no JSON-RPC
# mock is needed — only a built binary and a networks/<name>.json on disk.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/assert.sh"

CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
export CHAINBENCH_DIR

TMPDIR_ROOT="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_ROOT"' EXIT INT TERM HUP

# ---- Build the binary once for this test run. ----
BINARY="${TMPDIR_ROOT}/chainbench-net-test"
if ! ( cd "${CHAINBENCH_DIR}/network" && go build -o "${BINARY}" ./cmd/chainbench-net ) >/dev/null 2>&1; then
  echo "FATAL: failed to build chainbench-net" >&2
  exit 1
fi
export CHAINBENCH_NET_BIN="${BINARY}"

# ---- Materialize a hybrid network state file. ----
# Inline (self-contained) — mirrors examples/networks/hybrid-example.json:
# 3 local validators + 1 remote endpoint.
export CHAINBENCH_STATE_DIR="${TMPDIR_ROOT}/state"
mkdir -p "${CHAINBENCH_STATE_DIR}/networks"
cat > "${CHAINBENCH_STATE_DIR}/networks/hybrid-demo.json" <<'JSON'
{
  "name": "hybrid-demo",
  "chain_type": "ethereum",
  "chain_id": 1337,
  "nodes": [
    { "id": "v1",  "role": "validator", "provider": "local",  "http": "http://127.0.0.1:8501" },
    { "id": "v2",  "role": "validator", "provider": "local",  "http": "http://127.0.0.1:8502" },
    { "id": "v3",  "role": "validator", "provider": "local",  "http": "http://127.0.0.1:8503" },
    { "id": "rpc", "role": "endpoint",  "provider": "remote", "http": "https://devnet.example.com" }
  ]
}
JSON

# ---- Source the libraries under test. ----
# shellcheck disable=SC1091
source "${CHAINBENCH_DIR}/lib/common.sh"
# shellcheck disable=SC1091
source "${CHAINBENCH_DIR}/lib/test_meta.sh"
# shellcheck disable=SC1091
source "${CHAINBENCH_DIR}/lib/network_client.sh"

# Pull _cb_test_check_capabilities out of cmd_test.sh without triggering its
# bottom-of-file dispatcher (same technique as cmd-test-capabilities.sh).
HELPERS="${TMPDIR_ROOT}/cmd_test_helpers.sh"
awk '
  /^_cb_test_check_capabilities\(\)/ { in_fn=1 }
  in_fn { print }
  in_fn && /^}$/ { in_fn=0 }
' "${CHAINBENCH_DIR}/lib/cmd_test.sh" > "$HELPERS"
_CB_TEST_YELLOW=''
_CB_TEST_RESET=''
# shellcheck disable=SC1090
source "$HELPERS"

# Drive the gating decision off the REAL hybrid network (not the local default).
# This keeps the cap source the real binary + real state file, so cases B/C
# verify the capability-inference → gating wiring end-to-end for a hybrid net.
_cb_test_active_capabilities() {
  local data
  if ! data=$(cb_net_call "network.capabilities" '{"network":"hybrid-demo"}' 2>/dev/null); then
    return 1
  fi
  echo "$data" | jq -r '.capabilities[]?' 2>/dev/null | tr '\n' ' ' | sed 's/ $//'
}

make_fixture() {
  local path="$1" req="$2"
  cat > "$path" <<EOF
#!/usr/bin/env bash
# ---chainbench-meta---
# description: hybrid gating fixture
# requires_capabilities: [${req}]
# ---end-meta---
echo "noop test"
EOF
}

# ---- Case A: real binary computes the intersection for a hybrid network ----
describe "network.capabilities: hybrid (3 local + 1 remote) → [rpc, ws]"
rc=0
data="$(cb_net_call "network.capabilities" '{"network":"hybrid-demo"}' 2>/dev/null)" || rc=$?
assert_eq "$rc" "0" "success exit code"
assert_eq "$(jq -r .network <<<"$data")" "hybrid-demo" "network echoed back"
assert_eq "$(jq -r '.capabilities | join(",")' <<<"$data")" "rpc,ws" "capability intersection"

# ---- Case B: process-requiring test is gated against the hybrid network ----
describe "capability gate (hybrid): requires process → skip"
FB="${TMPDIR_ROOT}/case_process.sh"
make_fixture "$FB" "process"
set +e
SKIP_OUTPUT=$(_cb_test_check_capabilities "$FB" 2>&1 >/dev/null)
rc=$?
set -e
assert_eq "$rc" "1" "should skip when process is absent from hybrid lower bound"
assert_contains "$SKIP_OUTPUT" "process" "skip diagnostic names the missing cap"
assert_contains "$SKIP_OUTPUT" "SKIP" "skip diagnostic uses SKIP marker"

# ---- Case C: rpc-requiring test runs against the hybrid network ----
describe "capability gate (hybrid): requires rpc → run"
FC="${TMPDIR_ROOT}/case_rpc.sh"
make_fixture "$FC" "rpc"
set +e
_cb_test_check_capabilities "$FC" 2>/dev/null
rc=$?
set -e
assert_eq "$rc" "0" "should run when rpc is in the hybrid lower bound"

unit_summary
