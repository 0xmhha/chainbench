#!/usr/bin/env bash
# tests/unit/tests/node-block-number-auth.sh
# Subprocess-level E2E for node.block_number against an attached remote
# network that requires an API key in the Authorization header.
#
# Scenario:
#   1. Attach a mock JSON-RPC endpoint under the name "bash-bn-auth" with an
#      api-key auth block pointing at env var TEST_AUTH_KEY. The mock accepts
#      the attach probe (eth_chainId / istanbul_getValidators /
#      wemix_getReward) unauthenticated — the probe itself predates any
#      persisted Auth config by design.
#   2. Invoke node.block_number for (network=bash-bn-auth, node_id=node1).
#      The mock requires "Authorization: <TEST_AUTH_KEY value>" on the
#      eth_blockNumber call and returns a JSON-RPC error otherwise. Handler
#      success therefore proves the auth RoundTripper was injected into the
#      dial path used by node.block_number (not just attach).
#
# Header choice: default Authorization header (no "header" field in the auth
# config). Complements the Go E2E test, which exercises the explicit
# "header":"X-Api-Key" override.
#
# Mock transport: inline python3 JSON-RPC handler on an ephemeral port picked
# at runtime. Mock process is torn down via a trap covering EXIT INT TERM HUP.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/assert.sh"

CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
export CHAINBENCH_DIR

TMPDIR_ROOT="$(mktemp -d)"

# Per-test state dir — attach writes land here and nowhere else.
STATE_DIR="${TMPDIR_ROOT}/state"
mkdir -p "${STATE_DIR}"
export CHAINBENCH_STATE_DIR="${STATE_DIR}"

# ---- Build the binary once for this test run. ----
BINARY="${TMPDIR_ROOT}/chainbench-net-test"
if ! ( cd "${CHAINBENCH_DIR}/network" && go build -o "${BINARY}" ./cmd/chainbench-net ) >/dev/null 2>&1; then
  echo "FATAL: failed to build chainbench-net" >&2
  rm -rf "${TMPDIR_ROOT}"
  exit 1
fi
export CHAINBENCH_NET_BIN="${BINARY}"

# ---- Pick an ephemeral port. ----
PORT="$(python3 -c 'import socket
s=socket.socket()
s.bind(("127.0.0.1",0))
print(s.getsockname()[1])
s.close()')"

# ---- Credential resolved at dial time via env var. ----
export TEST_AUTH_KEY="bash-auth-secret"

# ---- Launch inline JSON-RPC mock. ----
# The mock reads EXPECTED_KEY from its argv so its view of the "good" value is
# captured at spawn time — independent of later exported changes in this
# shell. That keeps the auth assertion deterministic even if a future test
# mutates TEST_AUTH_KEY.
MOCK_LOG="${TMPDIR_ROOT}/mock.log"
python3 - "$PORT" "$TEST_AUTH_KEY" >"${MOCK_LOG}" 2>&1 <<'PYEOF' &
import sys, json, http.server, socketserver
port = int(sys.argv[1])
expected = sys.argv[2]
class H(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get('Content-Length', 0))
        try:
            req = json.loads(self.rfile.read(n))
        except Exception:
            req = {}
        method = req.get('method')
        rid = req.get('id')
        auth = self.headers.get('Authorization', '')
        # Probe methods (attach path) are unauthenticated by design.
        if method == 'eth_chainId':
            body = {'jsonrpc': '2.0', 'id': rid, 'result': '0x1'}
        elif method in ('istanbul_getValidators', 'wemix_getReward'):
            body = {'jsonrpc': '2.0', 'id': rid,
                    'error': {'code': -32601, 'message': 'method not found'}}
        elif method == 'eth_blockNumber':
            # Fail loud on a missing or mismatched header — a regression in
            # auth wiring would surface here as an UPSTREAM error from the
            # handler, not a silent pass.
            if auth != expected:
                body = {'jsonrpc': '2.0', 'id': rid,
                        'error': {'code': -32001,
                                  'message': 'missing or invalid Authorization header'}}
            else:
                body = {'jsonrpc': '2.0', 'id': rid, 'result': '0x99'}
        else:
            body = {'jsonrpc': '2.0', 'id': rid,
                    'error': {'code': -32601, 'message': 'method not found'}}
        raw = json.dumps(body).encode()
        self.send_response(200)
        self.send_header('Content-Type', 'application/json')
        self.send_header('Content-Length', str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)
    def log_message(self, *a, **k):
        pass
with socketserver.TCPServer(('127.0.0.1', port), H) as srv:
    srv.serve_forever()
PYEOF
MOCK_PID=$!

cleanup() {
  kill "${MOCK_PID}" 2>/dev/null || true
  wait "${MOCK_PID}" 2>/dev/null || true
  rm -rf "${TMPDIR_ROOT}"
}
trap cleanup EXIT INT TERM HUP

# ---- Wait for the mock to start listening (bounded). ----
mock_ready=0
for _ in 1 2 3 4 5 6 7 8 9 10; do
  if (echo > "/dev/tcp/127.0.0.1/${PORT}") 2>/dev/null; then
    mock_ready=1
    break
  fi
  sleep 0.1
done
if [[ "${mock_ready}" -ne 1 ]]; then
  echo "FATAL: mock RPC server failed to listen on 127.0.0.1:${PORT}" >&2
  echo "--- mock log ---" >&2
  cat "${MOCK_LOG}" >&2 || true
  exit 1
fi

# ---- Source the library under test. ----
# shellcheck disable=SC1091
source "${CHAINBENCH_DIR}/lib/network_client.sh"

# ---- Scenario 1: attach persists remote network with auth config ----
describe "network.attach: persists remote network bash-bn-auth with api-key auth"
rc=0
attach_args="$(jq -cn --arg url "http://127.0.0.1:${PORT}" '{
  rpc_url: $url,
  name: "bash-bn-auth",
  auth: { type: "api-key", env: "TEST_AUTH_KEY" }
}')"
data="$(cb_net_call "network.attach" "${attach_args}" 2>/dev/null)" || rc=$?
assert_eq "$rc" "0"                                      "attach success exit code"
assert_eq "$(jq -r .name    <<<"$data")" "bash-bn-auth" "attach name"
assert_eq "$(jq -r .created <<<"$data")" "true"         "attach created flag"

# ---- Scenario 2: state file persists env var name but NOT the secret value ----
describe "state file: persists env var name, never the credential"
state_file="${STATE_DIR}/networks/bash-bn-auth.json"
assert_file_exists "${state_file}"                      "state file landed on disk"
state_json="$(cat "${state_file}")"
assert_contains "${state_json}" "TEST_AUTH_KEY"         "state references env var name"
# Invert the contains check — credential value must NOT appear anywhere.
if [[ "${state_json}" == *"bash-auth-secret"* ]]; then
  (( ++_UT_FAIL )) || true
  _UT_FAILURES+=("state file leaked credential value: ${state_json}")
  printf "    \033[0;31m✗ state file leaked credential value\033[0m\n" >&2
else
  (( ++_UT_PASS )) || true
  printf "    \033[0;32m✓ state file does not leak credential value\033[0m\n" >&2
fi

# ---- Scenario 3: node.block_number uses injected auth; surfaces 0x99 (=153) ----
describe "node.block_number: Authorization header injected via AuthFromNode"
rc=0
data2="$(cb_net_call "node.block_number" '{"network":"bash-bn-auth","node_id":"node1"}' 2>/dev/null)" || rc=$?
assert_eq "$rc" "0"                                             "block_number success exit code"
assert_eq "$(jq -r .network      <<<"$data2")" "bash-bn-auth"  "block_number network echo"
assert_eq "$(jq -r .node_id      <<<"$data2")" "node1"         "block_number node_id echo"
assert_eq "$(jq -r .block_number <<<"$data2")" "153"           "block_number matches 0x99"

unit_summary
