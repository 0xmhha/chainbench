#!/usr/bin/env bash
# tests/unit/tests/profile-env-override.sh - Test _cb_set_var env-first guard
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/assert.sh"

CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
export CHAINBENCH_DIR
export CHAINBENCH_QUIET=1
source "${CHAINBENCH_DIR}/lib/common.sh"

# We need to test _cb_set_var which is defined inside _cb_export_profile_vars.
# Strategy: create a minimal merged profile JSON, call _cb_export_profile_vars,
# and check which value wins.

TMPDIR_ROOT="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_ROOT"' EXIT

PROFILE_JSON="${TMPDIR_ROOT}/profile.json"
cat > "$PROFILE_JSON" <<'JSON'
{
  "chain": {
    "binary": "gstable",
    "binary_path": "/profile/path/gstable",
    "network_id": "",
    "chain_id": "",
    "type": "stablenet"
  },
  "data": { "directory": "data" },
  "genesis": { "template": "" },
  "nodes": { "validators": "4", "endpoints": "1", "verbosity": "3", "en_verbosity": "3", "gcmode": "full", "cache": "1024", "extra_flags": "" },
  "keys": { "mode": "static", "source": "keys/preset" },
  "ports": { "base_p2p": "30301", "base_http": "8501", "base_ws": "9501", "base_auth": "8551", "base_metrics": "6061" },
  "logging": { "rotation": "true", "max_size": "10M", "max_files": "5", "directory": "data/logs" },
  "tests": { "auto_run": "" }
}
JSON

# Need to source profile.sh for _cb_export_profile_vars
# Reset guard to allow re-sourcing
unset _CHAINBENCH_PROFILE_SH_LOADED 2>/dev/null || true
source "${CHAINBENCH_DIR}/lib/profile.sh"

# ---- Test 1: env set (non-empty), profile has value -> env wins -----------
describe "env-first: env set overrides profile value"
export CHAINBENCH_BINARY_PATH="/env/path/gstable"
_cb_export_profile_vars "$PROFILE_JSON" "test"
assert_eq "${CHAINBENCH_BINARY_PATH}" "/env/path/gstable" "env value preserved"

# ---- Test 2: env empty string, profile has value -> profile wins -----------
describe "env-first: empty env string resets to profile"
export CHAINBENCH_BINARY_PATH=""
_cb_export_profile_vars "$PROFILE_JSON" "test"
assert_eq "${CHAINBENCH_BINARY_PATH}" "/profile/path/gstable" "profile value used for empty env"

# ---- Test 3: env unset -> profile wins ------------------------------------
describe "env-first: unset env uses profile value"
unset CHAINBENCH_BINARY_PATH
_cb_export_profile_vars "$PROFILE_JSON" "test"
assert_eq "${CHAINBENCH_BINARY_PATH}" "/profile/path/gstable" "profile value used when env unset"

# ---- Test 4: CHAINBENCH_PROFILE_ENV_OVERRIDE=0 + env set -> profile wins --
describe "env-first: opt-out with PROFILE_ENV_OVERRIDE=0"
export CHAINBENCH_BINARY_PATH="/env/path/gstable"
export CHAINBENCH_PROFILE_ENV_OVERRIDE=0
_cb_export_profile_vars "$PROFILE_JSON" "test"
assert_eq "${CHAINBENCH_BINARY_PATH}" "/profile/path/gstable" "profile wins when override disabled"
unset CHAINBENCH_PROFILE_ENV_OVERRIDE

unit_summary
