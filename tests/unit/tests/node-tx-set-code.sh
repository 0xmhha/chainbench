#!/usr/bin/env bash
# tests/unit/tests/node-tx-set-code.sh
# Exercises node.tx_send with a single EIP-7702 authorization tuple against
# a python JSON-RPC mock that captures the broadcast raw-hex into a file.
# Asserts the wire byte == 0x04 (SetCode tx type) — proving the handler emits
# the SetCodeTx envelope rather than falling back to legacy / 1559.
#
# Why subprocess (not just handler test):
#   handlers_test.go covers the SetCode handler logic in-process. This bash
#   layer exercises the full spawn path — env-sourced signer + authorizer
#   keys, run.go envelope parsing, the broadcast RoundTrip via ethclient,
#   and the network_client.sh NDJSON terminator parsing — same shape as
#   node-tx-wait.sh but for the 0x4 tx type added in Sprint 4c.
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

# Synthetic test-only keys — never associated with funded accounts. Sender
# (alice) signs the outer SetCodeTx; authorizer (bob) signs the single
# authorization tuple. Distinct keys → distinct recovered addresses, which
# is the wire shape EIP-7702 requires.
ALICE_KEY_HEX="$(python3 -c 'import secrets; print(secrets.token_hex(32))')"
BOB_KEY_HEX="$(python3 -c 'import secrets; print(secrets.token_hex(32))')"
export CHAINBENCH_SIGNER_ALICE_KEY="0x${ALICE_KEY_HEX}"
export CHAINBENCH_SIGNER_BOB_KEY="0x${BOB_KEY_HEX}"

PORT="$(python3 -c 'import socket
s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()')"

# The mock writes the raw param[0] of eth_sendRawTransaction to RAW_FILE so
# the bash harness can inspect the wire byte without parsing structured
# stdout from the network process.
RAW_FILE="${TMPDIR_ROOT}/sent-raw"
MOCK_LOG="${TMPDIR_ROOT}/mock.log"
python3 - "$PORT" "$RAW_FILE" >"${MOCK_LOG}" 2>&1 <<'PYEOF' &
import sys, json, http.server, socketserver
port = int(sys.argv[1])
raw_path = sys.argv[2]
class H(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get('Content-Length', 0))
        try: req = json.loads(self.rfile.read(n))
        except: req = {}
        m = req.get('method'); rid = req.get('id')
        if m == 'eth_chainId':
            body = {'jsonrpc':'2.0','id':rid,'result':'0x1'}
        elif m == 'eth_sendRawTransaction':
            params = req.get('params') or []
            if params:
                with open(raw_path, 'w') as f:
                    f.write(params[0] if isinstance(params[0], str) else '')
            body = {'jsonrpc':'2.0','id':rid,'result':'0x' + 'a' * 64}
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

describe "tx_send with authorization_list (EIP-7702 SetCode)"
attach_data="$(cb_net_call "network.attach" "{\"rpc_url\":\"http://127.0.0.1:${PORT}\",\"name\":\"sc\"}")"
assert_eq "$(jq -r .name <<<"$attach_data")" "sc" "attach name"

# Single-authorization SetCode tx. address/nonce/chain_id are hex-encoded
# per the schema; bob's signer alias selects CHAINBENCH_SIGNER_BOB_KEY.
# 1559 fee fields force the tx envelope toward DynamicFee shape; the
# authorization_list flips it to type 0x04 (SetCode).
sc_data="$(cb_net_call "node.tx_send" "{\"network\":\"sc\",\"node_id\":\"node1\",\"signer\":\"alice\",\"to\":\"0x0000000000000000000000000000000000000002\",\"value\":\"0x0\",\"gas\":100000,\"max_fee_per_gas\":\"0x59682f00\",\"max_priority_fee_per_gas\":\"0x3b9aca00\",\"nonce\":0,\"authorization_list\":[{\"chain_id\":\"0x1\",\"address\":\"0x000000000000000000000000000000000000beef\",\"nonce\":\"0x0\",\"signer\":\"bob\"}]}")"

tx_hash="$(jq -r .tx_hash <<<"$sc_data")"
assert_eq "$(printf '%s' "$tx_hash" | cut -c1-2)" "0x" "tx_hash starts with 0x"
assert_eq "${#tx_hash}" "66"                          "tx_hash is 66 chars"

# Wire-byte assertion: SetCodeTx leading byte is 0x04. Hex strings from the
# RPC carry the 0x prefix; strip it then read the first two chars.
[[ -f "${RAW_FILE}" ]] || { echo "FATAL: mock did not capture broadcast" >&2; exit 1; }
sent_raw="$(cat "${RAW_FILE}")"
sent_first_byte="$(printf '%s' "$sent_raw" | sed 's/^0x//' | cut -c1-2 | tr 'A-Z' 'a-z')"
assert_eq "${sent_first_byte}" "04" "broadcast first byte == 0x04 (SetCode)"

unit_summary
