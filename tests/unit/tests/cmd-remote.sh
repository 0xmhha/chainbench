#!/usr/bin/env bash
# tests/unit/tests/cmd-remote.sh
# Characterization test for the `chainbench remote` command surface
# (lib/cmd_remote.sh). Locks the state CRUD behavior so the P2-2a split
# (handlers -> lib/remote_commands.sh) is provably behavior-preserving.
#
# No live RPC: an unreachable URL makes `remote add` register with status
# 'unreachable' while still writing state, so add/list/info/select/remove are
# exercised without a mock server.
#
# NB: no `set -e` — the production dispatcher (chainbench.sh) runs these handlers
# without errexit, and several intentionally return non-zero in normal flow
# (e.g. a graceful connectivity miss). We capture exit codes explicitly instead.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/assert.sh"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

TMPDIR_ROOT="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_ROOT"' EXIT

# CHAINBENCH_DIR points at the temp dir (remote_state.sh derives remotes.json
# from it at source time); the library itself is loaded from the repo.
export CHAINBENCH_DIR="$TMPDIR_ROOT"
export CHAINBENCH_QUIET=1
mkdir -p "${CHAINBENCH_DIR}/state"

REMOTES="${CHAINBENCH_DIR}/state/remotes.json"
URL="http://127.0.0.1:1" # connection-refused → registered 'unreachable'

# Load the command (dispatcher runs once with no args → usage; ignore its rc).
set --
# shellcheck disable=SC1091
source "${REPO_ROOT}/lib/cmd_remote.sh" >/dev/null 2>&1 || true

has_alias() { python3 -c "import json,sys; d=json.load(open('$REMOTES')); sys.exit(0 if sys.argv[1] in d.get('remotes',{}) else 1)" "$1"; }

# ---- add ----
describe "remote add: registers alias in remotes.json"
cmd_remote_main add r1 "$URL" --type testnet >/dev/null 2>&1
assert_file_exists "$REMOTES" "remotes.json created"
if has_alias r1; then assert_eq ok ok "r1 registered"; else assert_eq missing present "r1 should be in remotes.json"; fi

# ---- list ----
describe "remote list: shows registered alias"
list_out="$(cmd_remote_main list 2>/dev/null)"
assert_contains "$list_out" "r1" "list shows r1"

# ---- info ----
describe "remote info: shows alias details"
info_out="$(cmd_remote_main info r1 2>/dev/null || true)"
assert_contains "$info_out" "r1" "info shows r1"

# ---- select ----
describe "remote select: writes current-remote"
cmd_remote_main select r1 >/dev/null 2>&1 || true
assert_file_exists "${CHAINBENCH_DIR}/state/current-remote" "current-remote written"
assert_contains "$(cat "${CHAINBENCH_DIR}/state/current-remote")" "r1" "current-remote = r1"

# ---- duplicate add rejected ----
describe "remote add: duplicate alias rejected"
set +e
cmd_remote_main add r1 "$URL" --type testnet >/dev/null 2>&1
rc=$?
set -e
assert_eq "$rc" "1" "duplicate add returns 1"

# ---- remove ----
describe "remote remove: deletes alias"
cmd_remote_main remove r1 >/dev/null 2>&1 || true
set +e
has_alias r1
gone_rc=$?
set -e
assert_eq "$gone_rc" "1" "r1 removed from remotes.json"

# ---- remove nonexistent ----
describe "remote remove: nonexistent alias rejected"
set +e
cmd_remote_main remove nope >/dev/null 2>&1
rc=$?
set -e
assert_eq "$rc" "1" "remove of unknown alias returns 1"

# ---- unknown subcommand ----
describe "remote: unknown subcommand rejected"
set +e
cmd_remote_main bogus >/dev/null 2>&1
rc=$?
set -e
assert_eq "$rc" "1" "unknown subcommand returns 1"

unit_summary
