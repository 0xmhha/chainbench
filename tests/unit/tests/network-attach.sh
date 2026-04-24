#!/usr/bin/env bash
# tests/unit/tests/network-attach.sh
# Covers the network.attach wire handler end-to-end via lib/network_client.sh,
# then round-trips the persisted state through network.load.
#
# Scenario:
#   1. attach a stablenet RPC under the name "bash-attach" — mock serves
#      eth_chainId=0x205b and istanbul_getValidators=[]; cb_net_call returns
#      chain_type=stablenet, name=bash-attach, created=true, and the
#      state/networks/bash-attach.json file is written.
#   2. network.load resolves the same name and yields the persisted record
#      with chain_id=8283.
#
# Mock transport: inline python3 JSON-RPC handler bound to an ephemeral
# port picked at runtime (no hard-coded 8545 collision). Mock process is
# torn down via a trap covering EXIT INT TERM HUP.
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
            body = {'jsonrpc': '2.0', 'id': rid, 'result': '0x205b'}
        elif method == 'istanbul_getValidators':
            body = {'jsonrpc': '2.0', 'id': rid, 'result': []}
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

# ---- Scenario 1: attach persists remote network ----
describe "network.attach: persists remote network as bash-attach"
rc=0
data="$(cb_net_call "network.attach" "{\"rpc_url\":\"http://127.0.0.1:${PORT}\",\"name\":\"bash-attach\"}" 2>/dev/null)" || rc=$?
assert_eq "$rc" "0" "attach success exit code"
assert_eq "$(jq -r .chain_type <<<"$data")" "stablenet"   "attach chain_type"
assert_eq "$(jq -r .name       <<<"$data")" "bash-attach" "attach name"
assert_eq "$(jq -r .created    <<<"$data")" "true"        "attach created flag"

assert_file_exists "${STATE_DIR}/networks/bash-attach.json" "state file persisted"

# ---- Scenario 2: load resolves the attached network by name ----
describe "network.load: resolves attached remote by name"
rc=0
loaded="$(cb_net_call "network.load" '{"name":"bash-attach"}' 2>/dev/null)" || rc=$?
assert_eq "$rc" "0" "load success exit code"
assert_eq "$(jq -r .name     <<<"$loaded")" "bash-attach" "load name"
assert_eq "$(jq -r .chain_id <<<"$loaded")" "8283"        "load chain_id"

unit_summary
