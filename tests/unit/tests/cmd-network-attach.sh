#!/usr/bin/env bash
# tests/unit/tests/cmd-network-attach.sh
# Covers the `chainbench network attach` CLI surface (lib/cmd_network.sh) over
# the network.attach wire command.
#
#   1. remote happy: real chainbench-net + mock JSON-RPC -> networks/<name>.json
#      created, human summary printed.
#   2. ssh-remote arg validation: missing required ssh flags -> CLI error (no
#      wire call). The SSH tunnel e2e itself is covered by the Go handler test.
#   3. ssh-remote args pass through: with required flags the CLI builds the JSON
#      and reaches the wire (fails at SSH dial, not at CLI validation).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/assert.sh"

CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
export CHAINBENCH_DIR

TMPDIR_ROOT="$(mktemp -d)"

BINARY="${TMPDIR_ROOT}/chainbench-net-test"
if ! ( cd "${CHAINBENCH_DIR}/network" && go build -o "${BINARY}" ./cmd/chainbench-net ) >/dev/null 2>&1; then
  echo "FATAL: failed to build chainbench-net" >&2
  rm -rf "${TMPDIR_ROOT}"; exit 1
fi
export CHAINBENCH_NET_BIN="${BINARY}"
export CHAINBENCH_STATE_DIR="${TMPDIR_ROOT}/state"
mkdir -p "${CHAINBENCH_STATE_DIR}"

PORT="$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()')"

MOCK_LOG="${TMPDIR_ROOT}/mock.log"
python3 - "$PORT" >"${MOCK_LOG}" 2>&1 <<'PYEOF' &
import sys, json, http.server, socketserver
port = int(sys.argv[1])
class H(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get('Content-Length', 0))
        try: req = json.loads(self.rfile.read(n))
        except Exception: req = {}
        m = req.get('method'); rid = req.get('id')
        if m == 'eth_chainId':
            body = {'jsonrpc':'2.0','id':rid,'result':'0x205b'}
        elif m == 'istanbul_getValidators':
            body = {'jsonrpc':'2.0','id':rid,'result':[]}
        else:
            body = {'jsonrpc':'2.0','id':rid,'error':{'code':-32601,'message':'nf'}}
        raw = json.dumps(body).encode()
        self.send_response(200); self.send_header('Content-Type','application/json')
        self.send_header('Content-Length', str(len(raw))); self.end_headers()
        self.wfile.write(raw)
    def log_message(self, *a, **k): pass
with socketserver.TCPServer(('127.0.0.1', port), H) as srv:
    srv.serve_forever()
PYEOF
MOCK_PID=$!
cleanup() { kill "${MOCK_PID}" 2>/dev/null || true; wait "${MOCK_PID}" 2>/dev/null || true; rm -rf "${TMPDIR_ROOT}"; }
trap cleanup EXIT INT TERM HUP

ready=0
for _ in 1 2 3 4 5 6 7 8 9 10; do
  if (echo > "/dev/tcp/127.0.0.1/${PORT}") 2>/dev/null; then ready=1; break; fi
  sleep 0.1
done
[[ "${ready}" -eq 1 ]] || { echo "FATAL: mock not listening" >&2; cat "${MOCK_LOG}" >&2; exit 1; }

# Load the CLI helpers. `set --` so the bottom-of-file dispatcher runs with no
# args (prints usage harmlessly); we then call the action function directly.
set --
# shellcheck disable=SC1091
source "${CHAINBENCH_DIR}/lib/cmd_network.sh" >/dev/null 2>&1
# shellcheck disable=SC1091
source "${CHAINBENCH_DIR}/lib/json_helpers.sh"  # cb_json_read for assertions

# ---- Case 1: remote attach happy (real binary + mock RPC) ----
describe "network attach: remote happy -> networks file + summary"
out="$(_cb_network_cmd_attach netx "http://127.0.0.1:${PORT}" 2>/dev/null)"
assert_file_exists "${CHAINBENCH_STATE_DIR}/networks/netx.json" "networks/netx.json created"
assert_contains "$out" "stablenet" "summary names chain_type"
assert_contains "$out" "8283" "summary names chain_id"
assert_eq "$(cb_json_read "${CHAINBENCH_STATE_DIR}/networks/netx.json" "nodes.0.provider")" "remote" "persisted provider=remote"

# ---- Case 2: ssh-remote missing required flags -> CLI error (no wire call) ----
describe "network attach: ssh-remote missing flags -> CLI error"
set +e
err="$(_cb_network_cmd_attach n2 "http://127.0.0.1:${PORT}" --provider ssh-remote 2>&1 >/dev/null)"
rc=$?
set -e
assert_eq "$rc" "1" "missing ssh flags -> exit 1"
assert_contains "$err" "ssh-user" "error names required ssh flags"

# ---- Case 3: ssh-remote with flags reaches the wire (past CLI validation) ----
# No SSH server here, so the wire call fails at SSH dial — but reaching that
# point proves the CLI built the args and dispatched (not a CLI-side reject).
describe "network attach: ssh-remote args pass CLI validation -> wire dispatch"
set +e
err3="$(CHAINBENCH_SSH_INSECURE_HOST_KEY=1 CB_TEST_SSH_PW=x \
  _cb_network_cmd_attach n3 "http://127.0.0.1:${PORT}" \
  --provider ssh-remote --ssh-user u --ssh-host 127.0.0.1 --ssh-port 1 --ssh-env CB_TEST_SSH_PW 2>&1 >/dev/null)"
rc3=$?
set -e
assert_eq "$rc3" "1" "ssh dial failure -> exit 1"
# It must NOT be a CLI validation error (those mention 'requires').
if printf '%s' "$err3" | grep -q "requires --ssh"; then
  assert_eq "reached-validation-error" "reached-wire" "should pass CLI validation and reach wire"
else
  assert_eq "ok" "ok" "passed CLI validation, dispatched to wire"
fi

unit_summary
