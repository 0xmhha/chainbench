#!/usr/bin/env bash
# tests/unit/tests/node-account-state.sh
# Exercises node.account_state against a python JSON-RPC mock answering
# eth_getBalance + eth_getTransactionCount + eth_getCode + eth_getStorageAt.
# Two scenarios in one script:
#   1. Default fields: omit `fields` → handler queries balance/nonce/code
#      and the response carries those three; storage is absent.
#   2. With storage: `fields: ["balance", "storage"]` + storage_key →
#      handler queries only the requested two; nonce/code are absent.
#
# Why subprocess (not just handler test):
#   handlers_test.go covers field selection + fields-validation in-process.
#   This bash layer exercises the full spawn path — run.go envelope parsing,
#   the four eth_get* RoundTrips via ethclient, and the network_client.sh
#   NDJSON terminator parsing. Same shape as the other Sprint 4d bash tests.
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
        elif m == 'eth_getBalance':
            # 1 ETH = 1e18 wei = 0xde0b6b3a7640000
            body = {'jsonrpc':'2.0','id':rid,'result':'0xde0b6b3a7640000'}
        elif m == 'eth_getTransactionCount':
            # 0x2a = 42
            body = {'jsonrpc':'2.0','id':rid,'result':'0x2a'}
        elif m == 'eth_getCode':
            body = {'jsonrpc':'2.0','id':rid,'result':'0x6080604052348015600f57600080fd5b50'}
        elif m == 'eth_getStorageAt':
            body = {'jsonrpc':'2.0','id':rid,
                    'result':'0x' + '0' * 62 + '99'}
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

describe "account_state: default fields = balance + nonce + code"
attach_data="$(cb_net_call "network.attach" "{\"rpc_url\":\"http://127.0.0.1:${PORT}\",\"name\":\"as\"}")"
assert_eq "$(jq -r .name <<<"$attach_data")" "as" "attach name"

state_default="$(cb_net_call "node.account_state" "{\"network\":\"as\",\"node_id\":\"node1\",\"address\":\"0x0000000000000000000000000000000000000001\"}")"

# balance is hex-encoded big.Int.Text(16) — leading zeros stripped relative
# to 32-byte padding. 1 ETH = 0xde0b6b3a7640000.
assert_eq "$(jq -r .balance <<<"$state_default")" "0xde0b6b3a7640000" \
          "balance = 1 ETH (0xde0b6b3a7640000)"
# nonce is the uint64 value emitted as a JSON number; jq -r prints the
# decimal integer.
assert_eq "$(jq -r .nonce <<<"$state_default")" "42" \
          "nonce = 42 (0x2a)"
# code surfaces with 0x prefix; just assert prefix + non-empty body.
code="$(jq -r .code <<<"$state_default")"
assert_eq "$(printf '%s' "$code" | cut -c1-2)" "0x" "code starts with 0x"
[[ "${#code}" -gt 2 ]] || { echo "FAIL: code is empty" >&2; exit 1; }
# storage is opt-in only — must be absent when not requested.
storage_present="$(jq 'has("storage")' <<<"$state_default")"
assert_eq "${storage_present}" "false" \
          "storage absent when not requested"

describe "account_state: explicit fields = balance + storage (nonce/code omitted)"
state_storage="$(cb_net_call "node.account_state" "{\"network\":\"as\",\"node_id\":\"node1\",\"address\":\"0x0000000000000000000000000000000000000001\",\"fields\":[\"balance\",\"storage\"],\"storage_key\":\"0x0000000000000000000000000000000000000000000000000000000000000000\"}")"

assert_eq "$(jq -r .balance <<<"$state_storage")" "0xde0b6b3a7640000" \
          "balance still surfaces"
storage_val="$(jq -r .storage <<<"$state_storage")"
assert_eq "${storage_val}" "0x0000000000000000000000000000000000000000000000000000000000000099" \
          "storage matches mock eth_getStorageAt response"
nonce_present="$(jq 'has("nonce")' <<<"$state_storage")"
assert_eq "${nonce_present}" "false" \
          "nonce absent when not in fields"
code_present="$(jq 'has("code")' <<<"$state_storage")"
assert_eq "${code_present}" "false" \
          "code absent when not in fields"

unit_summary
