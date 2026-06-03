#!/usr/bin/env bash
# tests/unit/tests/pids-binary-basename.sh - Pin pids_binary_basename (G3/M2.b)
#
# The stop/init kill path must target the basename of the binary actually
# launched (recorded in pids.json), not the profile-default name, so a renamed
# PR binary (e.g. gstable-pr1234) is stopped without leaking. These tests pin
# the resolver against fixture pids.json files: it prefers the top-level
# binary_basename, falls back to the basename of the first node's binary path,
# and returns non-zero (empty) when neither is present so callers fall back to
# CHAINBENCH_BINARY. A scratch CHAINBENCH_DIR keeps the real state/ untouched.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/assert.sh"

REPO_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

_scratch="$(mktemp -d)"
trap 'rm -rf "${_scratch}"' EXIT
mkdir -p "${_scratch}/state"

# Source pids_state.sh against the scratch dir so _CB_PIDS_FILE (readonly,
# derived at source time) points into the scratch state/.
CHAINBENCH_DIR="${_scratch}"
export CHAINBENCH_DIR
export CHAINBENCH_QUIET=1
source "${REPO_DIR}/lib/pids_state.sh"

_PIDS="${_scratch}/state/pids.json"

# ---- Test 1: top-level binary_basename is preferred -----------------------
describe "pids_binary_basename: top-level field wins"
cat > "${_PIDS}" <<'JSON'
{ "profile": "default", "binary_basename": "gstable-pr1234",
  "nodes": { "1": { "binary": "/opt/x/build/bin/gstable", "pid": 100 } } }
JSON
assert_eq "$(pids_binary_basename)" "gstable-pr1234" "uses top-level binary_basename"

# ---- Test 2: fall back to first node's binary basename --------------------
describe "pids_binary_basename: derive from node binary path"
cat > "${_PIDS}" <<'JSON'
{ "profile": "default",
  "nodes": { "1": { "binary": "/tmp/build/bin/gstable-pr99", "pid": 100 } } }
JSON
assert_eq "$(pids_binary_basename)" "gstable-pr99" "derives basename from node binary"

# ---- Test 3: neither present -> non-zero, empty ---------------------------
describe "pids_binary_basename: empty when no binary info"
cat > "${_PIDS}" <<'JSON'
{ "profile": "default", "nodes": { "1": { "pid": 100 } } }
JSON
rc=0
out="$(pids_binary_basename)" || rc=$?
assert_neq "$rc" "0" "returns non-zero when no binary recorded"
assert_empty "$out" "prints nothing when no binary recorded"

unit_summary
