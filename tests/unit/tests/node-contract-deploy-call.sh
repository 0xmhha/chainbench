#!/usr/bin/env bash
# tests/unit/tests/node-contract-deploy-call.sh
# Exercises node.contract_deploy + node.contract_call against a python
# JSON-RPC mock — the bash subprocess counterpart to handlers_test.go's
# in-process coverage of the Sprint 4d contract surface.
#
# Why subprocess (not just handler test):
#   handlers_test.go covers deploy/call argument validation and packing
#   in-process. This bash layer exercises the full spawn path — env-sourced
#   signer, run.go envelope parsing, the broadcast RoundTrip via ethclient,
#   the eth_call read RoundTrip, and the network_client.sh NDJSON terminator
#   parsing. Same shape as node-tx-set-code.sh but for the deploy + call
#   command pair.
#
# Mock transport: inline python3 JSON-RPC server on an ephemeral port. The
# mock answers eth_chainId, eth_estimateGas, eth_gasPrice, eth_getTransactionCount,
# eth_sendRawTransaction (deploy path) and eth_call (read path). The eth_call
# response is a 32-byte uint256 encoding of decimal 66 (0x42) so callers can
# assert on a recognisable pattern.
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

# Synthetic test-only signer key — not associated with any real funds.
ALICE_KEY_HEX="$(python3 -c 'import secrets; print(secrets.token_hex(32))')"
export CHAINBENCH_SIGNER_ALICE_KEY="0x${ALICE_KEY_HEX}"

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
        elif m == 'eth_getTransactionCount':
            body = {'jsonrpc':'2.0','id':rid,'result':'0x0'}
        elif m == 'eth_estimateGas':
            body = {'jsonrpc':'2.0','id':rid,'result':'0x186a0'}
        elif m == 'eth_gasPrice':
            body = {'jsonrpc':'2.0','id':rid,'result':'0x1'}
        elif m == 'eth_sendRawTransaction':
            body = {'jsonrpc':'2.0','id':rid,
                    'result':'0x' + 'a' * 64}
        elif m == 'eth_call':
            # 32-byte uint256 encoding of 66 (=0x42). Caller uses this to
            # assert result_raw round-trips verbatim.
            body = {'jsonrpc':'2.0','id':rid,
                    'result':'0x' + '0' * 62 + '42'}
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

describe "contract_deploy: bytecode broadcast + locally-computed address"
attach_data="$(cb_net_call "network.attach" "{\"rpc_url\":\"http://127.0.0.1:${PORT}\",\"name\":\"cd\"}")"
assert_eq "$(jq -r .name <<<"$attach_data")" "cd" "attach name"

# Tiny bytecode stub — content does not matter for the mock since
# eth_sendRawTransaction echoes a fake tx_hash unconditionally. The handler
# computes contract_address = keccak(rlp(sender, nonce))[12:] locally, so
# that field surfaces regardless of mock bytecode handling.
deploy_data="$(cb_net_call "node.contract_deploy" "{\"network\":\"cd\",\"node_id\":\"node1\",\"signer\":\"alice\",\"bytecode\":\"0x6080604052348015600f57600080fd5b50\",\"gas\":100000,\"gas_price\":\"0x1\",\"nonce\":0}")"

tx_hash="$(jq -r .tx_hash <<<"$deploy_data")"
assert_eq "$(printf '%s' "$tx_hash" | cut -c1-2)" "0x" "tx_hash starts with 0x"
assert_eq "${#tx_hash}" "66"                            "tx_hash is 66 chars"

contract_addr="$(jq -r .contract_address <<<"$deploy_data")"
assert_eq "$(printf '%s' "$contract_addr" | cut -c1-2)" "0x" "contract_address starts with 0x"
assert_eq "${#contract_addr}"                            "42" "contract_address is 42 chars"

describe "contract_call: raw calldata round-trip"
# balanceOf(address) selector + zero-padded address arg. The mock ignores the
# input and always returns the same 32-byte result; we only verify the wire
# parse + result_raw shape here.
call_data="$(cb_net_call "node.contract_call" "{\"network\":\"cd\",\"node_id\":\"node1\",\"contract_address\":\"${contract_addr}\",\"calldata\":\"0x70a082310000000000000000000000000000000000000000000000000000000000000001\"}")"

result_raw="$(jq -r .result_raw <<<"$call_data")"
assert_eq "${result_raw}" "0x0000000000000000000000000000000000000000000000000000000000000042" \
  "result_raw matches mock eth_call response"
# Note: the mock returns 0x + 62 zeros + "42" (32-byte uint256 encoding of
# decimal 66). The handler preserves the full hex including the 0x prefix.
# The expected value above is mock-output exact — no client-side trimming.

block_label="$(jq -r .block <<<"$call_data")"
assert_eq "${block_label}" "latest" "block label echoes default 'latest'"

unit_summary
