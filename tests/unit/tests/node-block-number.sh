#!/usr/bin/env bash
# tests/unit/tests/node-block-number.sh
# Subprocess-level E2E for node.block_number against an attached remote network.
#
# Scenario:
#   1. Attach a mock JSON-RPC endpoint under the name "bash-bn". The mock
#      responds to eth_chainId (attach probe) with 0x1, and returns -32601
#      for istanbul_getValidators / wemix_getReward so the probe classifies
#      the endpoint as a generic ethereum remote.
#   2. Invoke node.block_number for (network=bash-bn, node_id=node1). The
#      mock answers eth_blockNumber with 0x2a (42). Handler should surface
#      {network:"bash-bn", block_number:42} via the ethclient-backed
#      remote.Client path.
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
            body = {'jsonrpc': '2.0', 'id': rid, 'result': '0x1'}
        elif method == 'eth_blockNumber':
            body = {'jsonrpc': '2.0', 'id': rid, 'result': '0x2a'}
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

# ---- Scenario 1: attach persists remote network as bash-bn ----
describe "network.attach: persists remote network as bash-bn"
rc=0
data="$(cb_net_call "network.attach" "{\"rpc_url\":\"http://127.0.0.1:${PORT}\",\"name\":\"bash-bn\"}" 2>/dev/null)" || rc=$?
assert_eq "$rc" "0"                                 "attach success exit code"
assert_eq "$(jq -r .name    <<<"$data")" "bash-bn"  "attach name"
assert_eq "$(jq -r .created <<<"$data")" "true"     "attach created flag"

# ---- Scenario 2: node.block_number surfaces 0x2a (=42) via ethclient path ----
describe "node.block_number: surfaces eth_blockNumber through remote.Client"
rc=0
data2="$(cb_net_call "node.block_number" '{"network":"bash-bn","node_id":"node1"}' 2>/dev/null)" || rc=$?
assert_eq "$rc" "0"                                        "block_number success exit code"
assert_eq "$(jq -r .network      <<<"$data2")" "bash-bn"  "block_number network echo"
assert_eq "$(jq -r .node_id      <<<"$data2")" "node1"    "block_number node_id echo"
assert_eq "$(jq -r .block_number <<<"$data2")" "42"       "block_number matches 0x2a"

unit_summary
