#!/usr/bin/env bash
# tests/unit/tests/node-remote-reads.sh
# Subprocess-level E2E for the three new remote read commands
# (node.chain_id, node.balance, node.gas_price) against an attached remote
# network — parity coverage for the Go E2E TestE2E_NodeRemoteReads_WithAuth.
#
# Scenario:
#   1. Attach a mock JSON-RPC endpoint under the name "reads". The mock
#      answers eth_chainId (attach probe + node.chain_id call), and returns
#      -32601 for istanbul_getValidators / wemix_getReward so the probe
#      classifies the endpoint as a generic ethereum remote.
#   2. Invoke node.chain_id, node.balance, node.gas_price in sequence. The
#      mock returns canned values; the handlers surface them verbatim.
#
# Auth is intentionally omitted here — the Go E2E exercises X-Api-Key wiring
# and node-block-number-auth.sh exercises the default Authorization header.
# This test focuses on read-command coverage through the wire protocol.
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

# ---- Launch inline JSON-RPC mock. ----
# eth_chainId is called twice: once during attach's probe step, once for the
# node.chain_id command. Both share the same "0x2a" response so the handler's
# uint64 surface value is stable (42).
MOCK_LOG="${TMPDIR_ROOT}/mock.log"
python3 - "$PORT" >"${MOCK_LOG}" 2>&1 <<'PYEOF' &
import sys, json, http.server, socketserver
port = int(sys.argv[1])
class H(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get('Content-Length', 0))
        try:
            req = json.loads(self.rfile.read(n))
        except Exception:
            req = {}
        method = req.get('method')
        rid = req.get('id')
        if method == 'eth_chainId':
            body = {'jsonrpc': '2.0', 'id': rid, 'result': '0x2a'}
        elif method == 'eth_getBalance':
            body = {'jsonrpc': '2.0', 'id': rid, 'result': '0x500'}
        elif method == 'eth_gasPrice':
            body = {'jsonrpc': '2.0', 'id': rid, 'result': '0x64'}
        elif method in ('istanbul_getValidators', 'wemix_getReward'):
            body = {'jsonrpc': '2.0', 'id': rid,
                    'error': {'code': -32601, 'message': 'method not found'}}
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

# ---- Scenario 1: attach persists remote network as "reads" ----
describe "network.attach: persists remote network reads"
rc=0
data="$(cb_net_call "network.attach" "{\"rpc_url\":\"http://127.0.0.1:${PORT}\",\"name\":\"reads\"}" 2>/dev/null)" || rc=$?
assert_eq "$rc" "0"                                "attach success exit code"
assert_eq "$(jq -r .name    <<<"$data")" "reads"  "attach name"
assert_eq "$(jq -r .created <<<"$data")" "true"   "attach created flag"

# ---- Scenario 2: node.chain_id surfaces 0x2a (=42) via ethclient path ----
describe "node.chain_id: surfaces eth_chainId through remote.Client"
rc=0
data="$(cb_net_call "node.chain_id" '{"network":"reads","node_id":"node1"}' 2>/dev/null)" || rc=$?
assert_eq "$rc" "0"                              "chain_id success exit code"
assert_eq "$(jq -r .network  <<<"$data")" "reads" "chain_id network echo"
assert_eq "$(jq -r .node_id  <<<"$data")" "node1" "chain_id node_id echo"
assert_eq "$(jq -r .chain_id <<<"$data")" "42"    "chain_id matches 0x2a"

# ---- Scenario 3: node.balance surfaces 0x500 via ethclient path ----
describe "node.balance: surfaces eth_getBalance through remote.Client"
rc=0
data="$(cb_net_call "node.balance" '{"network":"reads","node_id":"node1","address":"0x0000000000000000000000000000000000000001"}' 2>/dev/null)" || rc=$?
assert_eq "$rc" "0"                                                                                 "balance success exit code"
assert_eq "$(jq -r .network <<<"$data")" "reads"                                                    "balance network echo"
assert_eq "$(jq -r .node_id <<<"$data")" "node1"                                                    "balance node_id echo"
assert_eq "$(jq -r .address <<<"$data")" "0x0000000000000000000000000000000000000001"              "balance address echo"
assert_eq "$(jq -r .block   <<<"$data")" "latest"                                                   "balance block defaults to latest"
assert_eq "$(jq -r .balance <<<"$data")" "0x500"                                                    "balance matches 0x500"

# ---- Scenario 4: node.gas_price surfaces 0x64 via ethclient path ----
describe "node.gas_price: surfaces eth_gasPrice through remote.Client"
rc=0
data="$(cb_net_call "node.gas_price" '{"network":"reads","node_id":"node1"}' 2>/dev/null)" || rc=$?
assert_eq "$rc" "0"                                 "gas_price success exit code"
assert_eq "$(jq -r .network   <<<"$data")" "reads"  "gas_price network echo"
assert_eq "$(jq -r .node_id   <<<"$data")" "node1"  "gas_price node_id echo"
assert_eq "$(jq -r .gas_price <<<"$data")" "0x64"   "gas_price matches 0x64"

unit_summary
