#!/usr/bin/env bash
# tests/unit/tests/node-tx-wait.sh
# Exercises node.tx_wait against a mock RPC returning a successful receipt.
#
# Why subprocess (not in-process):
#   The Go E2E in e2e_test.go drives cobra in-memory; this one exercises
#   the full spawn path — process-level env inheritance, the runtime
#   slog handler, and the bash client wrapper (lib/network_client.sh)
#   parsing the NDJSON terminator.
#
# Mock transport: inline python3 JSON-RPC server on an ephemeral port.
# The receipt JSON includes cumulativeGasUsed + logsBloom because
# go-ethereum's Client.TransactionReceipt parser rejects receipts
# missing those fields.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/assert.sh"
CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
export CHAINBENCH_DIR

TMPDIR_ROOT="$(mktemp -d)"
STATE_DIR="${TMPDIR_ROOT}/state"
mkdir -p "${STATE_DIR}"
export CHAINBENCH_STATE_DIR="${STATE_DIR}"

BINARY="${TMPDIR_ROOT}/chainbench-net-test"
( cd "${CHAINBENCH_DIR}/network" && go build -o "${BINARY}" ./cmd/chainbench-net ) >/dev/null 2>&1 || {
  echo "FATAL: build failed" >&2; rm -rf "${TMPDIR_ROOT}"; exit 1
}
export CHAINBENCH_NET_BIN="${BINARY}"

PORT="$(python3 -c 'import socket
s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()')"

MOCK_LOG="${TMPDIR_ROOT}/mock.log"
python3 - "$PORT" >"${MOCK_LOG}" 2>&1 <<'PYEOF' &
import sys, json, http.server, socketserver
port = int(sys.argv[1])
class H(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get('Content-Length', 0))
        try: req = json.loads(self.rfile.read(n))
        except: req = {}
        m = req.get('method'); rid = req.get('id')
        if m == 'eth_chainId':
            body = {'jsonrpc':'2.0','id':rid,'result':'0x1'}
        elif m == 'eth_getTransactionReceipt':
            body = {'jsonrpc':'2.0','id':rid,'result':{
                'transactionHash':'0x'+'a'*64,
                'blockHash':'0x'+'b'*64,
                'blockNumber':'0x10',
                'cumulativeGasUsed':'0x5208',
                'gasUsed':'0x5208',
                'effectiveGasPrice':'0x1',
                'status':'0x1',
                'contractAddress':None,
                'logsBloom':'0x'+'0'*512,
                'logs':[]}}
        elif m in ('istanbul_getValidators','wemix_getReward'):
            body = {'jsonrpc':'2.0','id':rid,'error':{'code':-32601,'message':'nf'}}
        else:
            body = {'jsonrpc':'2.0','id':rid,'error':{'code':-32601,'message':'nf'}}
        raw = json.dumps(body).encode()
        self.send_response(200)
        self.send_header('Content-Type','application/json')
        self.send_header('Content-Length', str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)
    def log_message(self, *a, **k): pass
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

mock_ready=0
for _ in 1 2 3 4 5 6 7 8 9 10; do
  if (echo > "/dev/tcp/127.0.0.1/${PORT}") 2>/dev/null; then mock_ready=1; break; fi
  sleep 0.1
done
[[ "${mock_ready}" -eq 1 ]] || { echo "FATAL: mock not listening" >&2; cat "${MOCK_LOG}" >&2; exit 1; }

# shellcheck disable=SC1091
source "${CHAINBENCH_DIR}/lib/network_client.sh"

describe "tx_wait: attach mock then poll receipt"
attach_data="$(cb_net_call "network.attach" "{\"rpc_url\":\"http://127.0.0.1:${PORT}\",\"name\":\"wt\"}")"
assert_eq "$(jq -r .name <<<"$attach_data")" "wt" "attach name"

wait_data="$(cb_net_call "node.tx_wait" "{\"network\":\"wt\",\"node_id\":\"node1\",\"tx_hash\":\"0x$(printf 'a%.0s' {1..64})\",\"timeout_ms\":3000}")"
assert_eq "$(jq -r .status <<<"$wait_data")" "success" "status is success"
assert_eq "$(jq -r .block_number <<<"$wait_data")" "16" "block_number = 0x10 (16)"

unit_summary
