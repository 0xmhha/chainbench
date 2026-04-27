#!/usr/bin/env bash
# tests/unit/tests/security-key-boundary.sh
# Subprocess-level enforcement of the S4 key-material boundary
# (VISION §5.17.5). Spawns a freshly built chainbench-net binary, feeds
# it an attach + node.tx_send pair with a known-synthetic private key
# supplied via CHAINBENCH_SIGNER_ALICE_KEY, and asserts that the raw
# hex NEVER surfaces in stdout, stderr, or any configured log file.
#
# Why subprocess, not in-process:
#   The Go E2E test in e2e_test.go drives cobra in-memory; this one
#   exercises the full spawn path — the runtime slog.Handler, the
#   process-level env inheritance, and any stray fmt.Print* calls that
#   would only surface when the binary is executed as a subprocess.
#
# Log-file note:
#   chainbench-net's `run` subcommand routes slog via
#   wire.SetupLoggerWithFallback(stderr) — when CHAINBENCH_NET_LOG is set
#   logs land in that file; otherwise they fall back to the caller's
#   stderr. This test sets CHAINBENCH_NET_LOG to a tmp file and scans
#   stdout, stderr, AND that file: the audit surface for key-material
#   leakage covers all three streams.
#
# Mock transport: inline python3 JSON-RPC server on an ephemeral port.
# Mock process is torn down via a trap covering EXIT INT TERM HUP.
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

# ---- Known-synthetic test key (NOT a real funded account). ----
# This value MUST NOT appear in stdout / stderr / log after tx_send.
KEY_HEX="b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291"
export CHAINBENCH_SIGNER_ALICE_KEY="0x${KEY_HEX}"

# ---- Pick an ephemeral port for the mock RPC. ----
PORT="$(python3 -c 'import socket
s=socket.socket()
s.bind(("127.0.0.1",0))
print(s.getsockname()[1])
s.close()')"

# ---- Launch inline JSON-RPC mock. ----
# Handles: eth_chainId (probe + tx_send chain_id lookup),
#          eth_sendRawTransaction (broadcast),
#          istanbul_getValidators / wemix_getReward (probe path -32601).
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
        elif method == 'eth_sendRawTransaction':
            body = {'jsonrpc': '2.0', 'id': rid, 'result': '0x' + 'a' * 64}
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

# ---- Scenario 1: attach the mock under "sec-key-boundary". ----
describe "security: attach sets up test network"
rc=0
attach_data="$(cb_net_call "network.attach" \
  "{\"rpc_url\":\"http://127.0.0.1:${PORT}\",\"name\":\"sec-key-boundary\"}" \
  2>/dev/null)" || rc=$?
assert_eq "$rc" "0"                                           "attach success exit code"
assert_eq "$(jq -r .name <<<"$attach_data")" "sec-key-boundary" "attach name"

# ---- Scenario 2: tx.send through the attached network. ----
# Explicit nonce / gas / gas_price short-circuit the auto-fill paths so
# the mock only has to answer eth_chainId + eth_sendRawTransaction.
describe "security: tx.send does not leak key material"

TX_STDOUT_STREAM="${TMPDIR_ROOT}/tx.stream"
TX_ERR="${TMPDIR_ROOT}/tx.err"
TX_LOG="${TMPDIR_ROOT}/tx.log"
# See header comment — run.go bypasses this env var today; we set it so
# any future wiring is covered without test churn.
export CHAINBENCH_NET_LOG="${TX_LOG}"
: > "${TX_LOG}"  # ensure the file exists so cat later is safe

# Invoke the binary directly so we can capture stdout, stderr, and log
# as three distinct streams (cb_net_call collapses them through a pipe).
TX_ENVELOPE='{"command":"node.tx_send","args":{"network":"sec-key-boundary","node_id":"node1","signer":"alice","to":"0x0000000000000000000000000000000000000002","value":"0x0","gas":21000,"gas_price":"0x1","nonce":0}}'
tx_rc=0
printf '%s\n' "${TX_ENVELOPE}" | "${BINARY}" run \
  >"${TX_STDOUT_STREAM}" 2>"${TX_ERR}" || tx_rc=$?
assert_eq "$tx_rc" "0" "tx.send binary exit code"

# Extract tx_hash from the result terminator in stdout.
tx_hash="$(grep '^{' "${TX_STDOUT_STREAM}" \
  | awk '/"type":"result"/ { last = $0 } END { print last }' \
  | jq -r '.data.tx_hash // ""')"
assert_eq "$(printf '%s' "$tx_hash" | cut -c1-2)" "0x" "tx_hash starts with 0x"
assert_eq "${#tx_hash}" "66"                             "tx_hash is 66 chars (0x + 64 hex)"

# ---- The boundary assertion: scan all three streams for the raw key. ----
# Case-sensitive catches exact lowercase hex (how it entered via env).
# Case-insensitive catches any accidental upper-case re-render (e.g. via
# %X formatting) that would still be a leak.
leak_found=0
for label in stdout stderr log; do
  case "$label" in
    stdout) content_file="${TX_STDOUT_STREAM}" ;;
    stderr) content_file="${TX_ERR}" ;;
    log)    content_file="${TX_LOG}" ;;
  esac
  if [[ ! -f "${content_file}" ]]; then
    continue
  fi
  if grep -q -- "${KEY_HEX}" "${content_file}" 2>/dev/null; then
    echo "FAIL: ${label} leaks raw key material (case-sensitive match)" >&2
    echo "  file: ${content_file}" >&2
    leak_found=1
  fi
  if grep -qi -- "${KEY_HEX}" "${content_file}" 2>/dev/null; then
    # grep -i is a superset of grep -q above; only record a distinct
    # failure when the lowercase match missed (i.e. uppercase leak).
    if ! grep -q -- "${KEY_HEX}" "${content_file}" 2>/dev/null; then
      echo "FAIL: ${label} leaks key (case-insensitive match only — uppercase rendered)" >&2
      echo "  file: ${content_file}" >&2
      leak_found=1
    fi
  fi
done

if [[ "${leak_found}" -ne 0 ]]; then
  echo "FATAL: private key hex appeared in one or more observable streams" >&2
  exit 1
fi

assert_eq "leak-checked" "leak-checked" "no key material in stdout/stderr/log"

# ---- Scenario 3: keystore variant of tx.send. ----
# Generates a fresh keystore via a tiny Go helper (rotated per-run), runs
# tx.send through the keystore-loading branch of signer.Manager, then scans
# stdout / stderr / log for both the underlying raw-hex AND the password.
# Re-uses the mock RPC (still alive — cleanup trap fires on EXIT) and the
# already-attached "sec-key-boundary" network. CHAINBENCH_SIGNER_ALICE_KEY
# from Scenario 2 stays set; we use a different signer ("bob") so the
# keystore branch is the one exercised.
describe "security: keystore variant of tx.send"

GEN_DIR="${TMPDIR_ROOT}/gen-keystore"
mkdir -p "${GEN_DIR}"
cat > "${GEN_DIR}/main.go" <<'GOEOF'
package main

import (
    "crypto/ecdsa"
    "encoding/hex"
    "fmt"
    "os"

    "github.com/ethereum/go-ethereum/accounts/keystore"
    "github.com/ethereum/go-ethereum/crypto"
    "github.com/google/uuid"
)

func main() {
    if len(os.Args) < 3 {
        fmt.Fprintln(os.Stderr, "usage: gen-keystore <out-path> <password>")
        os.Exit(2)
    }
    outPath := os.Args[1]
    password := os.Args[2]
    pk, err := crypto.GenerateKey()
    if err != nil { panic(err) }
    id, _ := uuid.NewRandom()
    k := &keystore.Key{Id: id, Address: crypto.PubkeyToAddress(pk.PublicKey), PrivateKey: pk}
    enc, err := keystore.EncryptKey(k, password, keystore.LightScryptN, keystore.LightScryptP)
    if err != nil { panic(err) }
    if err := os.WriteFile(outPath, enc, 0o600); err != nil { panic(err) }
    // Print the raw hex to stderr for the test harness to capture and grep
    // against. The hex is rotated per-test via crypto.GenerateKey so it
    // never matches a real funded account.
    var pkVal *ecdsa.PrivateKey = pk
    fmt.Fprintln(os.Stderr, hex.EncodeToString(crypto.FromECDSA(pkVal)))
}
GOEOF

KS_PATH="${TMPDIR_ROOT}/keystore.json"
KS_PASSWORD="ks-test-pass"

GEN_LOG="${TMPDIR_ROOT}/gen.log"
gen_rc=0
( cd "${CHAINBENCH_DIR}/network" && go run "${GEN_DIR}/main.go" "${KS_PATH}" "${KS_PASSWORD}" ) 2>"${GEN_LOG}" >/dev/null || gen_rc=$?
if [[ "$gen_rc" -ne 0 ]]; then
  echo "FATAL: gen-keystore failed rc=$gen_rc" >&2
  cat "${GEN_LOG}" >&2 || true
  exit 1
fi
# Helper prints the raw hex on its last stderr line. Anything else (build chatter
# from `go run`) would land before that line, so use tail -n1.
GEN_KEY_HEX="$(tail -n1 "${GEN_LOG}")"
if ! [[ "${GEN_KEY_HEX}" =~ ^[0-9a-f]{64}$ ]]; then
  echo "FATAL: gen-keystore did not emit a 64-char hex private key" >&2
  echo "--- gen log ---" >&2
  cat "${GEN_LOG}" >&2 || true
  exit 1
fi

export CHAINBENCH_SIGNER_BOB_KEYSTORE="${KS_PATH}"
export CHAINBENCH_SIGNER_BOB_KEYSTORE_PASSWORD="${KS_PASSWORD}"

KS_TX_LOG="${TMPDIR_ROOT}/ks-tx.log"
KS_TX_ERR="${TMPDIR_ROOT}/ks-tx.err"
export CHAINBENCH_NET_LOG="${KS_TX_LOG}"
ks_tx_data="$(cb_net_call "node.tx_send" "{\"network\":\"sec-key-boundary\",\"node_id\":\"node1\",\"signer\":\"bob\",\"to\":\"0x0000000000000000000000000000000000000002\",\"value\":\"0x0\",\"gas\":21000,\"gas_price\":\"0x1\",\"nonce\":0}" 2>"${KS_TX_ERR}")"
assert_eq "$(jq -r .tx_hash <<<"$ks_tx_data" | cut -c1-2)" "0x" "ks tx_hash starts with 0x"

for label in "stdout" "stderr" "log"; do
  case "$label" in
    stdout) content="$ks_tx_data" ;;
    stderr) content="$(cat "${KS_TX_ERR}" 2>/dev/null || true)" ;;
    log)    content="$(cat "${KS_TX_LOG}" 2>/dev/null || true)" ;;
  esac
  if echo "$content" | grep -qi "${GEN_KEY_HEX}"; then
    echo "FAIL: ${label} leaks keystore raw key" >&2; exit 1
  fi
  if echo "$content" | grep -q "${KS_PASSWORD}"; then
    echo "FAIL: ${label} leaks keystore password" >&2; exit 1
  fi
done
assert_eq "leak-checked" "leak-checked" "no keystore key/password in stdout/stderr/log"

unit_summary
