#!/usr/bin/env bash
# tests/unit/tests/node-tx-fee-delegation.sh
# Exercises node.tx_fee_delegation_send (Sprint 4c) against a python JSON-RPC
# mock that captures the broadcast raw-hex into a file.
#
# Two scenarios share the same binary, mock, and signer envs:
#   A. happy stablenet  — attach with --override stablenet, broadcast
#                          succeeds, raw tx prefix == 0x16, tx_hash 66-char.
#   B. NOT_SUPPORTED    — attach with --override ethereum, fee-delegation
#                          must reject with APIError code NOT_SUPPORTED.
#
# Why subprocess (not just handler test):
#   handlers_test.go covers both branches in-process. This bash layer
#   exercises the full spawn path — env-sourced two-signer envelope, the
#   broadcast RoundTrip via ethclient, and chain-type adapter gating from
#   the persisted state file — same shape as node-tx-set-code.sh.
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

# Synthetic test-only keys — sender (alice) signs the inner envelope, fee
# payer (fpayer) signs the outer 0x16 hash. Distinct keys so recovered
# addresses differ on chain.
ALICE_KEY_HEX="$(python3 -c 'import secrets; print(secrets.token_hex(32))')"
FPAYER_KEY_HEX="$(python3 -c 'import secrets; print(secrets.token_hex(32))')"
export CHAINBENCH_SIGNER_ALICE_KEY="0x${ALICE_KEY_HEX}"
export CHAINBENCH_SIGNER_FPAYER_KEY="0x${FPAYER_KEY_HEX}"

PORT="$(python3 -c 'import socket
s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()')"

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

# ---- Scenario A: happy stablenet — fee delegation broadcasts. ----
describe "tx_fee_delegation_send: stablenet broadcast emits 0x16 tx"
attach_a="$(cb_net_call "network.attach" \
  "{\"rpc_url\":\"http://127.0.0.1:${PORT}\",\"name\":\"fd-stab\",\"override\":\"stablenet\"}")"
assert_eq "$(jq -r .name <<<"$attach_a")"       "fd-stab"   "fd-stab attach name"
assert_eq "$(jq -r .chain_type <<<"$attach_a")" "stablenet" "fd-stab chain_type"

: > "${RAW_FILE}"
fd_data="$(cb_net_call "node.tx_fee_delegation_send" \
  "{\"network\":\"fd-stab\",\"node_id\":\"node1\",\"signer\":\"alice\",\"fee_payer\":\"fpayer\",\"to\":\"0x0000000000000000000000000000000000000003\",\"value\":\"0x0\",\"max_fee_per_gas\":\"0x59682f00\",\"max_priority_fee_per_gas\":\"0x3b9aca00\",\"gas\":21000,\"nonce\":7}")"

tx_hash="$(jq -r .tx_hash <<<"$fd_data")"
assert_eq "$(printf '%s' "$tx_hash" | cut -c1-2)" "0x" "fd tx_hash starts with 0x"
assert_eq "${#tx_hash}" "66"                          "fd tx_hash is 66 chars"

[[ -s "${RAW_FILE}" ]] || { echo "FATAL: mock did not capture broadcast" >&2; exit 1; }
sent_raw="$(cat "${RAW_FILE}")"
sent_first_byte="$(printf '%s' "$sent_raw" | sed 's/^0x//' | cut -c1-2 | tr 'A-Z' 'a-z')"
assert_eq "${sent_first_byte}" "16" "broadcast first byte == 0x16 (fee-delegation)"

# ---- Scenario B: ethereum override — adapter must reject with NOT_SUPPORTED. ----
describe "tx_fee_delegation_send: ethereum chain_type rejects with NOT_SUPPORTED"
attach_b="$(cb_net_call "network.attach" \
  "{\"rpc_url\":\"http://127.0.0.1:${PORT}\",\"name\":\"fd-eth\",\"override\":\"ethereum\"}")"
assert_eq "$(jq -r .chain_type <<<"$attach_b")" "ethereum" "fd-eth chain_type"

# Run binary directly so we can inspect both exit code and the result
# terminator (cb_net_call would mask non-zero exits as a fatal). The
# envelope shape mirrors Scenario A; only the network changes.
FD_OUT="${TMPDIR_ROOT}/fd-eth.out"
FD_ERR="${TMPDIR_ROOT}/fd-eth.err"
ENV_B='{"command":"node.tx_fee_delegation_send","args":{"network":"fd-eth","node_id":"node1","signer":"alice","fee_payer":"fpayer","to":"0x0000000000000000000000000000000000000003","value":"0x0","max_fee_per_gas":"0x59682f00","max_priority_fee_per_gas":"0x3b9aca00","gas":21000,"nonce":0}}'
rc_b=0
printf '%s\n' "${ENV_B}" | "${BINARY}" run >"${FD_OUT}" 2>"${FD_ERR}" || rc_b=$?
[[ "${rc_b}" -ne 0 ]] || { echo "FATAL: ethereum fd attempt should have failed (rc=${rc_b})" >&2; exit 1; }

# Last NDJSON line is the result terminator; classify the error code.
err_code="$(grep '^{' "${FD_OUT}" \
  | awk '/"type":"result"/ { last = $0 } END { print last }' \
  | jq -r '.error.code // ""')"
assert_eq "${err_code}" "NOT_SUPPORTED" "ethereum override fee-delegation error.code"

unit_summary
