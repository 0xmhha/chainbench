#!/usr/bin/env bash
# tests/unit/tests/cmd-test-capabilities.sh — capability gating unit test
#
# Exercises _cb_test_check_capabilities in lib/cmd_test.sh. Mocks
# cb_net_call (the wire client) with a shell function override so the
# tests run without building chainbench-net or spawning a JSON-RPC
# mock — capability gating is pure decision logic given (a) test
# frontmatter and (b) wire response.
#
# Five cases:
#   1. requires_capabilities: [process] + active provides process → run (0)
#   2. requires_capabilities: [process] + active = [rpc ws]      → skip (1)
#   3. No frontmatter                                            → run (0)
#   4. requires_capabilities: []                                 → run (0)
#   5. cb_net_call unreachable + frontmatter present             → run (0, WARN)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/assert.sh"

CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
export CHAINBENCH_DIR

TMPDIR_ROOT="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_ROOT"' EXIT

# common.sh expects log_* helpers; sourcing it pulls in the rest of the
# bash runtime (color codes etc.) and is needed by cmd_test.sh.
# shellcheck disable=SC1091
source "${CHAINBENCH_DIR}/lib/common.sh"

# We do NOT source cmd_test.sh directly because it dispatches at the
# bottom (cmd_test_main "$@") and would intercept our pwd. Instead pull
# only the helpers we need by sourcing a stripped copy.
#
# Trick: extract the function definitions into a tmp file via awk and
# source that. Keeps the test isolated from the dispatcher.
HELPERS="${TMPDIR_ROOT}/cmd_test_helpers.sh"
awk '
  /^_cb_test_active_capabilities\(\)/ { in_fn=1 }
  /^_cb_test_check_capabilities\(\)/ { in_fn=1 }
  in_fn { print }
  in_fn && /^}$/ { in_fn=0 }
' "${CHAINBENCH_DIR}/lib/cmd_test.sh" > "$HELPERS"

# The helpers reference the color codes; declare minimal stubs.
_CB_TEST_YELLOW=''
_CB_TEST_RESET=''

# test_meta is needed for cb_parse_meta inside _cb_test_check_capabilities.
# shellcheck disable=SC1091
source "${CHAINBENCH_DIR}/lib/test_meta.sh"

# shellcheck disable=SC1090
source "$HELPERS"

# ---- Override cb_net_call ----------------------------------------------------
# The mock surfaces whatever JSON is in $MOCK_NET_DATA on stdout (return 0)
# or returns 1 (unreachable) when MOCK_NET_DATA is empty.
cb_net_call() {
  if [[ -z "${MOCK_NET_DATA:-}" ]]; then
    return 1
  fi
  printf '%s\n' "$MOCK_NET_DATA"
  return 0
}
export -f cb_net_call 2>/dev/null || true

# ---- Helpers -----------------------------------------------------------------

# Write a fixture script with optional requires_capabilities frontmatter.
make_fixture() {
  local path="$1"
  local req="${2:-}"
  if [[ -z "$req" ]]; then
    cat > "$path" <<'NOMETA'
#!/usr/bin/env bash
echo "noop test"
NOMETA
    return 0
  fi
  cat > "$path" <<EOF
#!/usr/bin/env bash
# ---chainbench-meta---
# description: capability-gated fixture
# requires_capabilities: [${req}]
# ---end-meta---
echo "noop test"
EOF
}

# ---- Case 1: requires_capabilities present + cap satisfied → run -------------

describe "capability gate: required cap satisfied → run"
F1="${TMPDIR_ROOT}/case1.sh"
make_fixture "$F1" "process"
MOCK_NET_DATA='{"network":"local","capabilities":["admin","fs","network-topology","process","rpc","ws"]}'
set +e
_cb_test_check_capabilities "$F1" 2>/dev/null
rc=$?
set -e
assert_eq "$rc" "0" "should return 0 when cap is satisfied"

# ---- Case 2: requires_capabilities present + cap missing → skip --------------

describe "capability gate: required cap missing → skip"
F2="${TMPDIR_ROOT}/case2.sh"
make_fixture "$F2" "process"
MOCK_NET_DATA='{"network":"sepolia","capabilities":["rpc","ws"]}'
set +e
SKIP_OUTPUT=$(_cb_test_check_capabilities "$F2" 2>&1 >/dev/null)
rc=$?
set -e
assert_eq "$rc" "1" "should return 1 when cap is missing"
assert_contains "$SKIP_OUTPUT" "process" "skip diagnostic mentions missing cap"
assert_contains "$SKIP_OUTPUT" "SKIP" "skip diagnostic uses SKIP marker"

# ---- Case 3: no frontmatter → run unconditionally ----------------------------

describe "capability gate: no frontmatter → run"
F3="${TMPDIR_ROOT}/case3.sh"
make_fixture "$F3"  # no req — produces a script with no chainbench-meta block
MOCK_NET_DATA=''  # even with cb_net_call unreachable, no requirements means run
set +e
_cb_test_check_capabilities "$F3" 2>/dev/null
rc=$?
set -e
assert_eq "$rc" "0" "should return 0 when no frontmatter"

# ---- Case 4: empty requires_capabilities → run -------------------------------

describe "capability gate: empty requires_capabilities → run"
F4="${TMPDIR_ROOT}/case4.sh"
cat > "$F4" <<'EMPTY'
#!/usr/bin/env bash
# ---chainbench-meta---
# description: empty list
# requires_capabilities: []
# ---end-meta---
echo "noop"
EMPTY
MOCK_NET_DATA='{"network":"local","capabilities":["rpc","ws"]}'
set +e
_cb_test_check_capabilities "$F4" 2>/dev/null
rc=$?
set -e
assert_eq "$rc" "0" "should return 0 when requires_capabilities is empty"

# ---- Case 5: cb_net_call unreachable + frontmatter → WARN + run --------------

describe "capability gate: unreachable wire → WARN + run"
F5="${TMPDIR_ROOT}/case5.sh"
make_fixture "$F5" "process"
MOCK_NET_DATA=''  # forces cb_net_call to return 1
set +e
WARN_OUTPUT=$(_cb_test_check_capabilities "$F5" 2>&1 >/dev/null)
rc=$?
set -e
assert_eq "$rc" "0" "should return 0 (permissive) when wire is unreachable"
assert_contains "$WARN_OUTPUT" "WARN" "diagnostic includes WARN marker"
assert_contains "$WARN_OUTPUT" "without gating" "diagnostic explains permissive fallback"

unit_summary
