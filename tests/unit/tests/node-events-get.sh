#!/usr/bin/env bash
# tests/unit/tests/node-events-get.sh
# Exercises node.events_get against a python JSON-RPC mock returning a
# single Transfer-like log entry. Two scenarios in one script:
#   1. Unfiltered: no address / topics — assert one log surfaces with all
#      required fields.
#   2. Filtered: address + topic[0] — assert the same log surfaces (mock is
#      fixed) and the wire shape carries the topic array.
#
# Why subprocess (not just handler test):
#   handlers_test.go covers parseBlockArg / topic parsing in-process. This
#   bash layer exercises the full spawn path — run.go envelope parsing, the
#   eth_getLogs RoundTrip via ethclient, and the network_client.sh NDJSON
#   terminator parsing. Same shape as the other Sprint 4d bash tests.
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
# Synthetic Transfer-like log:
#   topic[0] = keccak256("Transfer(address,address,uint256)")
#   topic[1] = from address (zero-padded to 32 bytes)
#   topic[2] = to   address (zero-padded to 32 bytes)
#   data     = 32-byte value
TRANSFER_TOPIC = '0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef'
LOG_ENTRY = {
    'address':          '0x000000000000000000000000000000000000abcd',
    'topics':           [TRANSFER_TOPIC,
                         '0x000000000000000000000000000000000000000000000000000000000000beef',
                         '0x000000000000000000000000000000000000000000000000000000000000cafe'],
    'data':             '0x0000000000000000000000000000000000000000000000000000000000000064',
    'blockNumber':      '0x10',
    'blockHash':        '0x' + 'b' * 64,
    'transactionHash':  '0x' + 'a' * 64,
    'transactionIndex': '0x0',
    'logIndex':         '0x0',
    'removed':          False,
}
class H(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get('Content-Length', 0))
        try: req = json.loads(self.rfile.read(n))
        except: req = {}
        m = req.get('method'); rid = req.get('id')
        if m == 'eth_chainId':
            body = {'jsonrpc':'2.0','id':rid,'result':'0x1'}
        elif m == 'eth_getLogs':
            body = {'jsonrpc':'2.0','id':rid,'result':[LOG_ENTRY]}
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

describe "events_get: unfiltered eth_getLogs round-trip"
attach_data="$(cb_net_call "network.attach" "{\"rpc_url\":\"http://127.0.0.1:${PORT}\",\"name\":\"ev\"}")"
assert_eq "$(jq -r .name <<<"$attach_data")" "ev" "attach name"

logs_unfiltered="$(cb_net_call "node.events_get" "{\"network\":\"ev\",\"node_id\":\"node1\"}")"
log_count="$(jq '.logs | length' <<<"$logs_unfiltered")"
assert_eq "${log_count}" "1" "unfiltered returns 1 log"

# Spot-check required fields surface verbatim from the mock.
assert_eq "$(jq -r '.logs[0].address' <<<"$logs_unfiltered")" \
          "0x000000000000000000000000000000000000ABcD" \
          "log[0].address (EIP-55 checksummed by go-ethereum on the way out)"
assert_eq "$(jq -r '.logs[0].block_number' <<<"$logs_unfiltered")" "16" \
          "log[0].block_number = 0x10 (16)"
assert_eq "$(jq '.logs[0].topics | length' <<<"$logs_unfiltered")" "3" \
          "log[0].topics has 3 entries"
assert_eq "$(jq -r '.logs[0].removed' <<<"$logs_unfiltered")" "false" \
          "log[0].removed is false"

describe "events_get: filtered by address + topic[0]"
logs_filtered="$(cb_net_call "node.events_get" "{\"network\":\"ev\",\"node_id\":\"node1\",\"address\":\"0x000000000000000000000000000000000000abcd\",\"topics\":[\"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef\"]}")"
filtered_count="$(jq '.logs | length' <<<"$logs_filtered")"
assert_eq "${filtered_count}" "1" "filtered returns 1 log (mock is fixed)"

# Topic round-trip — first topic in response matches the filter we sent.
assert_eq "$(jq -r '.logs[0].topics[0]' <<<"$logs_filtered")" \
          "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef" \
          "log[0].topics[0] is the Transfer signature hash"

unit_summary
