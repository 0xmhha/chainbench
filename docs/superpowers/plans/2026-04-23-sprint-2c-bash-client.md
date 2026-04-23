# Sprint 2c Bash Client Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship `lib/network_client.sh` — a bash library that exposes `cb_net_call <command> [args_json]` for spawning `chainbench-net run`, piping an envelope over stdin, parsing the NDJSON result terminator, and returning a structured exit code + data/error. Ship a unit test (`tests/unit/tests/network-wire-protocol.sh`) that exercises missing-binary, happy-path, and API-error scenarios against a freshly-built binary in tempdir.

**Architecture:** Pure bash + `jq`. Library has one public function (`cb_net_call`) plus three internal helpers (`_cb_net_binary` / `_cb_net_spawn` / `_cb_net_parse_result`). Test builds `chainbench-net` into tempdir, points `CHAINBENCH_NET_BIN` + `CHAINBENCH_STATE_DIR` at isolated fixtures, and asserts on stdout/stderr/exit code using the existing `tests/unit/lib/assert.sh` harness.

**Tech Stack:** bash 4.0+, `jq`, `go build` (for test harness build step), existing `tests/unit/run.sh` discovery.

**Spec reference:** `docs/superpowers/specs/2026-04-23-sprint-2c-bash-client-design.md`.

---

## File Structure

**Created:**
- `lib/network_client.sh`
- `tests/unit/tests/network-wire-protocol.sh`

**Modified:**
- `network/README.md` (add short "Bash client usage" section)

---

## Task 1: `lib/network_client.sh` — library

**Files:**
- Create: `lib/network_client.sh`

- [ ] **Step 1.1: Write the library file**

Create `lib/network_client.sh`:

```bash
#!/usr/bin/env bash
# lib/network_client.sh — bash client for chainbench-net.
#
# Usage:
#   source "${CHAINBENCH_DIR}/lib/network_client.sh"
#   data=$(cb_net_call "network.load" '{"name":"local"}') || exit $?
#
# Binary resolution (first match wins):
#   1. $CHAINBENCH_NET_BIN
#   2. $CHAINBENCH_DIR/bin/chainbench-net
#   3. $CHAINBENCH_DIR/network/bin/chainbench-net
#   4. `command -v chainbench-net` on PATH
#
# Exit codes:
#   0 — success. data JSON on stdout.
#   1 — API error. "<code>: <message>" on stderr.
#   2 — spawn/parse failure. diagnostic on stderr.
#
# Dependencies: jq.

# Guard against double-sourcing.
if [[ -n "${_CB_NET_CLIENT_LOADED:-}" ]]; then
  return 0
fi
readonly _CB_NET_CLIENT_LOADED=1

if ! command -v jq >/dev/null 2>&1; then
  echo "network_client: WARNING: jq not found on PATH — cb_net_call will fail" >&2
fi

# _cb_net_binary prints the resolved path to chainbench-net or returns 1.
_cb_net_binary() {
  if [[ -n "${CHAINBENCH_NET_BIN:-}" && -x "${CHAINBENCH_NET_BIN}" ]]; then
    echo "${CHAINBENCH_NET_BIN}"
    return 0
  fi
  if [[ -n "${CHAINBENCH_DIR:-}" ]]; then
    if [[ -x "${CHAINBENCH_DIR}/bin/chainbench-net" ]]; then
      echo "${CHAINBENCH_DIR}/bin/chainbench-net"
      return 0
    fi
    if [[ -x "${CHAINBENCH_DIR}/network/bin/chainbench-net" ]]; then
      echo "${CHAINBENCH_DIR}/network/bin/chainbench-net"
      return 0
    fi
  fi
  if command -v chainbench-net >/dev/null 2>&1; then
    command -v chainbench-net
    return 0
  fi
  return 1
}

# _cb_net_spawn <envelope_json>
# Pipes envelope_json to chainbench-net run and prints its stdout stream.
# Returns the subprocess exit code.
_cb_net_spawn() {
  local envelope="$1"
  local bin
  if ! bin=$(_cb_net_binary); then
    echo "network_client: chainbench-net binary not found" >&2
    return 2
  fi
  printf '%s\n' "$envelope" | "$bin" run
}

# _cb_net_parse_result
# Reads NDJSON from stdin, finds the last type=result line, and emits:
#   ok=true  — prints .data JSON to stdout, returns 0
#   ok=false — prints "<code>: <message>" to stderr, returns 1
#   no terminator — prints diagnostic to stderr, returns 2
_cb_net_parse_result() {
  local result_line
  result_line=$(grep '^{' | awk '/"type":"result"/ { last = $0 } END { print last }')
  if [[ -z "$result_line" ]]; then
    echo "network_client: no result terminator in stream" >&2
    return 2
  fi
  local ok
  ok=$(printf '%s' "$result_line" | jq -r '.ok // false')
  if [[ "$ok" == "true" ]]; then
    printf '%s' "$result_line" | jq -c '.data // {}'
    return 0
  fi
  local code msg
  code=$(printf '%s' "$result_line" | jq -r '.error.code // "UNKNOWN"')
  msg=$(printf '%s' "$result_line" | jq -r '.error.message // ""')
  echo "${code}: ${msg}" >&2
  return 1
}

# cb_net_call <command> [args_json]
# See header comment for semantics.
cb_net_call() {
  local command="${1:?cb_net_call: command required}"
  local args_json="${2:-{}}"

  if ! command -v jq >/dev/null 2>&1; then
    echo "network_client: jq not available" >&2
    return 2
  fi

  local envelope
  if ! envelope=$(jq -cn --arg c "$command" --argjson a "$args_json" \
    '{command:$c,args:$a}' 2>/dev/null); then
    echo "network_client: invalid args_json: ${args_json}" >&2
    return 2
  fi

  local stream
  local spawn_rc
  stream=$(_cb_net_spawn "$envelope")
  spawn_rc=$?

  if [[ $spawn_rc -eq 2 ]]; then
    # _cb_net_spawn already printed a diagnostic.
    return 2
  fi

  # Parse regardless of non-zero exit — the binary emits a result terminator
  # for every exit path (protocol/invalid/upstream/internal).
  printf '%s\n' "$stream" | _cb_net_parse_result
}
```

- [ ] **Step 1.2: Quick smoke check (no chainbench-net yet)**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
bash -c 'source lib/network_client.sh && type cb_net_call && type _cb_net_binary && type _cb_net_spawn && type _cb_net_parse_result'
```
Expected: all four functions print "is a function" output.

- [ ] **Step 1.3: Lint**

```bash
bash -n lib/network_client.sh && echo "syntax ok"
```
Expected: `syntax ok`.

Optional (if shellcheck available):
```bash
command -v shellcheck >/dev/null && shellcheck lib/network_client.sh || echo "shellcheck skipped"
```

- [ ] **Step 1.4: Commit**

```bash
git add lib/network_client.sh
git commit -m "network: add bash client library for chainbench-net"
```

---

## Task 2: `tests/unit/tests/network-wire-protocol.sh` — unit test

**Files:**
- Create: `tests/unit/tests/network-wire-protocol.sh`

- [ ] **Step 2.1: Confirm unit test fixture files exist**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
ls network/internal/state/testdata/pids-default.json \
   network/internal/state/testdata/profile-default.yaml
```
Expected: both files exist (from Sprint 2b.1).

- [ ] **Step 2.2: Write the test**

Create `tests/unit/tests/network-wire-protocol.sh`:

```bash
#!/usr/bin/env bash
# tests/unit/tests/network-wire-protocol.sh
# Covers lib/network_client.sh end-to-end against a freshly-built chainbench-net.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/assert.sh"

CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
export CHAINBENCH_DIR

TMPDIR_ROOT="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_ROOT"' EXIT

# ---- Build the binary once for this test run. ----
BINARY="${TMPDIR_ROOT}/chainbench-net-test"
( cd "${CHAINBENCH_DIR}/network" && go build -o "${BINARY}" ./cmd/chainbench-net ) \
  || { echo "FATAL: failed to build chainbench-net"; exit 1; }

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

# ---- Scenario 1: binary not found ----
test_start "binary_not_found"
(
  CHAINBENCH_NET_BIN="/nonexistent/chainbench-net"
  unset CHAINBENCH_DIR
  export PATH="/dev/null"  # ensure PATH lookup also fails
  # The call must return 2 and print a diagnostic to stderr.
  output=$(cb_net_call "network.load" '{"name":"local"}' 2>&1 >/dev/null)
  rc=$?
  assert_eq "$rc" "2" "binary-missing exit code"
  assert_contains "$output" "chainbench-net binary not found" "stderr mentions missing binary"
)
test_result

# ---- Scenario 2: happy path network.load ----
test_start "happy_network_load"
(
  export CHAINBENCH_NET_BIN="${BINARY}"
  export CHAINBENCH_STATE_DIR="${STATE_DIR}"

  data=$(cb_net_call "network.load" '{"name":"local"}')
  rc=$?
  assert_eq "$rc" "0" "success exit code"
  assert_contains "$data" '"name":"local"' "data contains name"
  assert_contains "$data" '"chain_type":"stablenet"' "data contains chain_type"
  assert_contains "$data" '"chain_id":8283' "data contains chain_id"
  assert_contains "$data" '"nodes"' "data contains nodes"
)
test_result

# ---- Scenario 3: API error — unsupported name ----
test_start "api_error_invalid_args"
(
  export CHAINBENCH_NET_BIN="${BINARY}"
  export CHAINBENCH_STATE_DIR="${STATE_DIR}"

  # Capture stdout and stderr separately. stdout should be empty on error.
  err_output=$({ cb_net_call "network.load" '{"name":"bogus"}' 2>&1 >/dev/null; true; })
  # Re-run to capture rc cleanly.
  cb_net_call "network.load" '{"name":"bogus"}' >/dev/null 2>&1 && rc=0 || rc=$?
  assert_eq "$rc" "1" "api-error exit code"
  assert_contains "$err_output" "INVALID_ARGS" "stderr mentions INVALID_ARGS"
)
test_result

# ---- Scenario 4: invalid args_json ----
test_start "invalid_args_json"
(
  export CHAINBENCH_NET_BIN="${BINARY}"
  export CHAINBENCH_STATE_DIR="${STATE_DIR}"

  err_output=$({ cb_net_call "network.load" 'not json' 2>&1 >/dev/null; true; })
  cb_net_call "network.load" 'not json' >/dev/null 2>&1 && rc=0 || rc=$?
  assert_eq "$rc" "2" "malformed-args exit code"
  assert_contains "$err_output" "invalid args_json" "stderr mentions invalid args_json"
)
test_result
```

- [ ] **Step 2.3: Confirm `tests/unit/lib/assert.sh` has the assertions used**

```bash
grep -E '^(test_start|test_result|assert_eq|assert_contains) \(\)' tests/unit/lib/assert.sh
```
Expected: all four names found.

If any are missing, adjust the test to use whatever the real API is (e.g., `ASSERT_EQ`, `pass`, `fail`). Read a sibling test like `tests/unit/tests/assert-observe.sh` to confirm the conventions.

- [ ] **Step 2.4: Run the test directly**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
bash tests/unit/tests/network-wire-protocol.sh
```
Expected: all 4 scenarios pass. The script may emit colored progress output from `test_start` / `test_result`.

If scenarios 3/4 fail due to missing state fixtures in the test binary's invocation environment, verify `CHAINBENCH_STATE_DIR` is actually exported and `ls "$STATE_DIR"` shows the fixtures.

- [ ] **Step 2.5: Run the full unit harness**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
bash tests/unit/run.sh
```
Expected: `network-wire-protocol.sh` shows PASSED; all other existing unit tests still pass.

- [ ] **Step 2.6: Commit**

```bash
git add tests/unit/tests/network-wire-protocol.sh
git commit -m "test: add bash client wire-protocol unit test"
```

---

## Task 3: Documentation update

**Files:**
- Modify: `network/README.md`

- [ ] **Step 3.1: Append usage section to `network/README.md`**

Read the current `network/README.md`, then use Edit to append (after the existing content, before any trailing whitespace):

```markdown

## Bash client (`lib/network_client.sh`)

Bash callers can invoke `chainbench-net` through the library. It handles
envelope construction, subprocess spawn, and NDJSON parsing.

    source "${CHAINBENCH_DIR}/lib/network_client.sh"
    data=$(cb_net_call "network.load" '{"name":"local"}') || exit $?

Exit codes:

- `0` — success; data JSON on stdout.
- `1` — API error; `<code>: <message>` on stderr.
- `2` — spawn/parse failure; diagnostic on stderr.

Binary resolution order:
`$CHAINBENCH_NET_BIN`, `$CHAINBENCH_DIR/bin/chainbench-net`,
`$CHAINBENCH_DIR/network/bin/chainbench-net`, `command -v chainbench-net`.

Requires `jq`.
```

- [ ] **Step 3.2: Verify the README is still well-formed**

```bash
cat /Users/wm-it-22-00661/Work/github/tools/chainbench/network/README.md | tail -20
```
Expected: new section at the bottom, no accidental truncation of earlier content.

- [ ] **Step 3.3: Commit**

```bash
git add network/README.md
git commit -m "docs: document bash client usage for chainbench-net"
```

---

## Final verification

- [ ] **Commit list**

```bash
git log --oneline b28f7d6..HEAD
```
Expected 3 commits:
1. `network: add bash client library for chainbench-net`
2. `test: add bash client wire-protocol unit test`
3. `docs: document bash client usage for chainbench-net`

- [ ] **Re-run unit harness end-to-end**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench && bash tests/unit/run.sh
```
Expected: all unit tests PASS (including the new scenario).

- [ ] **Module-wide Go green (unchanged)**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go build ./... && go test -race ./... && go vet ./... && gofmt -l .
```
Expected: all clean (Sprint 2c didn't touch Go code).

---

## Out of scope

- Replacing existing `chainbench node stop|start|restart` with `cb_net_call` (Sprint 3+)
- Event stream callbacks (current MVP discards non-terminator lines beyond result)
- Auto-install of `chainbench-net` via `install.sh` (separate)
- MCP server integration (separate)
