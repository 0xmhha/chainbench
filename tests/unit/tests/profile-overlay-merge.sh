#!/usr/bin/env bash
# tests/unit/tests/profile-overlay-merge.sh - Test local overlay merge in profile loading
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/assert.sh"

CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
export CHAINBENCH_DIR
export CHAINBENCH_QUIET=1

TMPDIR_ROOT="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_ROOT"' EXIT

# Create a temporary state directory to hold the overlay
FAKE_STATE="${TMPDIR_ROOT}/state"
mkdir -p "$FAKE_STATE"

# Reset profile.sh guard for re-sourcing
unset _CHAINBENCH_PROFILE_SH_LOADED 2>/dev/null || true
source "${CHAINBENCH_DIR}/lib/common.sh"
source "${CHAINBENCH_DIR}/lib/profile.sh"

# Use _cb_python_merge_yaml to test overlay merge
# We'll call it with a profile that has a known value and check if overlay overrides it

# Create a minimal profile
PROFILE="${TMPDIR_ROOT}/test-profile.yaml"
cat > "$PROFILE" <<'YAML'
chain:
  binary: gstable
  binary_path: /original/path/gstable
  type: stablenet
nodes:
  validators: 4
  endpoints: 1
  verbosity: 3
  en_verbosity: 3
  gcmode: full
  cache: 1024
  extra_flags: ""
data:
  directory: data
genesis:
  template: ""
keys:
  mode: static
  source: keys/preset
ports:
  base_p2p: 30301
  base_http: 8501
  base_ws: 9501
  base_auth: 8551
  base_metrics: 6061
logging:
  rotation: true
  max_size: 10M
  max_files: 5
  directory: data/logs
tests:
  auto_run: ""
YAML

# ---- Test 1: No overlay file -> profile only -------------------------------
describe "overlay: no overlay file uses profile values only"
rm -f "${FAKE_STATE}/local-config.yaml"
merged="$(CHAINBENCH_DIR="${TMPDIR_ROOT}" _cb_python_merge_yaml "$PROFILE" 2>/dev/null)"
binary_path="$(echo "$merged" | python3 -c "import sys,json; print(json.load(sys.stdin).get('chain',{}).get('binary_path',''))")"
assert_eq "$binary_path" "/original/path/gstable" "original binary_path preserved"

# ---- Test 2: Overlay overrides a leaf field --------------------------------
describe "overlay: overrides a leaf field"
cat > "${FAKE_STATE}/local-config.yaml" <<'YAML'
chain:
  binary_path: /overlay/path/gstable
YAML
merged="$(CHAINBENCH_DIR="${TMPDIR_ROOT}" _cb_python_merge_yaml "$PROFILE" 2>/dev/null)"
binary_path="$(echo "$merged" | python3 -c "import sys,json; print(json.load(sys.stdin).get('chain',{}).get('binary_path',''))")"
assert_eq "$binary_path" "/overlay/path/gstable" "overlay value overrides profile"

# ---- Test 3: Overlay adds a new field ------------------------------------
describe "overlay: adds a new field"
cat > "${FAKE_STATE}/local-config.yaml" <<'YAML'
chain:
  logrot_path: /opt/logrot
YAML
merged="$(CHAINBENCH_DIR="${TMPDIR_ROOT}" _cb_python_merge_yaml "$PROFILE" 2>/dev/null)"
logrot_path="$(echo "$merged" | python3 -c "import sys,json; print(json.load(sys.stdin).get('chain',{}).get('logrot_path',''))")"
assert_eq "$logrot_path" "/opt/logrot" "new field added from overlay"

# ---- Test 4: Deep merge preserves sibling fields --------------------------
describe "overlay: deep merge preserves siblings"
cat > "${FAKE_STATE}/local-config.yaml" <<'YAML'
chain:
  binary_path: /overlay/path
YAML
merged="$(CHAINBENCH_DIR="${TMPDIR_ROOT}" _cb_python_merge_yaml "$PROFILE" 2>/dev/null)"
binary="$(echo "$merged" | python3 -c "import sys,json; print(json.load(sys.stdin).get('chain',{}).get('binary',''))")"
assert_eq "$binary" "gstable" "sibling field chain.binary preserved"

# ---- Test 5: Overlay with 'inherits' field -> ignored + warn ---------------
describe "overlay: inherits field in overlay is ignored"
cat > "${FAKE_STATE}/local-config.yaml" <<'YAML'
inherits: regression
chain:
  binary_path: /overlay/path
YAML
merged="$(CHAINBENCH_DIR="${TMPDIR_ROOT}" _cb_python_merge_yaml "$PROFILE" 2>/dev/null)"
binary_path="$(echo "$merged" | python3 -c "import sys,json; print(json.load(sys.stdin).get('chain',{}).get('binary_path',''))")"
assert_eq "$binary_path" "/overlay/path" "overlay value applied despite inherits"

# ---- Cleanup ---------------------------------------------------------------
rm -f "${FAKE_STATE}/local-config.yaml"

unit_summary
