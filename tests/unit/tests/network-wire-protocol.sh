#!/usr/bin/env bash
# tests/unit/tests/network-wire-protocol.sh
# Covers lib/network_client.sh end-to-end against a freshly-built chainbench-net.
#
# Scenarios:
#   1. binary_not_found     — CHAINBENCH_NET_BIN points at nothing, PATH empty -> rc=2
#   2. happy_network_load   — network.load with name=local returns data -> rc=0
#   3. api_error            — network.load with bogus name -> INVALID_ARGS, rc=1
#   4. malformed_args_json  — bad JSON arg -> diagnostic, rc=2
#
# Assertions are collected in the outer shell (no nested subshells around
# scenarios) so that `unit_summary` sees accurate pass/fail counts.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/assert.sh"

CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
export CHAINBENCH_DIR

TMPDIR_ROOT="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_ROOT"' EXIT

# ---- Build the binary once for this test run. ----
BINARY="${TMPDIR_ROOT}/chainbench-net-test"
if ! ( cd "${CHAINBENCH_DIR}/network" && go build -o "${BINARY}" ./cmd/chainbench-net ) >/dev/null 2>&1; then
  echo "FATAL: failed to build chainbench-net" >&2
  exit 1
fi

# ---- Prepare a state directory with fixtures. ----
STATE_DIR="${TMPDIR_ROOT}/state"
mkdir -p "${STATE_DIR}"
cp "${CHAINBENCH_DIR}/network/internal/state/testdata/pids-default.json" \
   "${STATE_DIR}/pids.json"
cp "${CHAINBENCH_DIR}/network/internal/state/testdata/profile-default.yaml" \
   "${STATE_DIR}/current-profile.yaml"

# ---- Source the library under test. ----
# shellcheck disable=SC1091
source "${CHAINBENCH_DIR}/lib/network_client.sh"

# Preserve CHAINBENCH_DIR so later scenarios can restore it. PATH is kept
# as-is (the test host normally does NOT have chainbench-net on PATH, and
# we need jq to be resolvable for the library's jq-availability check).
_ORIG_CHAINBENCH_DIR="$CHAINBENCH_DIR"

# Sanity: confirm chainbench-net is NOT on the host PATH so scenario 1 is a
# real "binary not found" case rather than a PATH-pickup.
if command -v chainbench-net >/dev/null 2>&1; then
  echo "SKIP: chainbench-net is on PATH; scenario 1 would pick it up" >&2
  exit 0
fi

# ---- Scenario 1: binary not found ----
describe "network_client: binary not found -> rc=2"
export CHAINBENCH_NET_BIN="/nonexistent/chainbench-net"
unset CHAINBENCH_DIR

rc=0
output=$(cb_net_call "network.load" '{"name":"local"}' 2>&1 >/dev/null) || rc=$?
assert_eq "$rc" "2" "binary-missing exit code"
assert_contains "$output" "chainbench-net binary not found" "stderr mentions missing binary"

# Restore CHAINBENCH_DIR before later scenarios.
export CHAINBENCH_DIR="$_ORIG_CHAINBENCH_DIR"
unset CHAINBENCH_NET_BIN

# ---- Scenario 2: happy path network.load ----
describe "network_client: happy path network.load -> rc=0 + data"
export CHAINBENCH_NET_BIN="${BINARY}"
export CHAINBENCH_STATE_DIR="${STATE_DIR}"

rc=0
data=$(cb_net_call "network.load" '{"name":"local"}' 2>/dev/null) || rc=$?
assert_eq "$rc" "0" "success exit code"
assert_contains "$data" '"name":"local"' "data contains name"
assert_contains "$data" '"chain_type":"stablenet"' "data contains chain_type"
assert_contains "$data" '"chain_id":8283' "data contains chain_id"
assert_contains "$data" '"nodes"' "data contains nodes"

# ---- Scenario 3: API error — structurally-invalid name ----
# Post-Sprint 3b: "bogus" is a valid remote-name pattern and would be treated
# as "attached network not found" (UPSTREAM_ERROR). To still exercise the
# INVALID_ARGS code path via the wire client, use a name that violates the
# schema pattern so the handler rejects it at the boundary.
describe "network_client: API error (bad name pattern) -> rc=1 + INVALID_ARGS"
rc=0
err_output=$(cb_net_call "network.load" '{"name":"Has-Upper"}' 2>&1 >/dev/null) || rc=$?
assert_eq "$rc" "1" "api-error exit code"
assert_contains "$err_output" "INVALID_ARGS" "stderr mentions INVALID_ARGS"

# ---- Scenario 4: invalid args_json ----
describe "network_client: malformed args_json -> rc=2"
rc=0
err_output=$(cb_net_call "network.load" 'not json' 2>&1 >/dev/null) || rc=$?
assert_eq "$rc" "2" "malformed-args exit code"
assert_contains "$err_output" "invalid args_json" "stderr mentions invalid args_json"

unit_summary
