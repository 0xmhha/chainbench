#!/usr/bin/env bash
# tests/unit/tests/profile-inheritance.sh
# Locks profile inheritance behavior in scripts/merge_profile.py:
#   - `extends:` resolves the parent (regression test for the P2-1 fix; before
#     it, only `inherits:` worked and `extends:` profiles silently lost their
#     parent and failed validation).
#   - `inherits:` still works (no regression).
#   - a missing parent raises an error (load fails, not silently empty).
#
# Calls scripts/merge_profile.py directly with a temp profiles_root so the
# fixtures are isolated from the real profiles/ tree.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/assert.sh"

REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
MERGE="${REPO_ROOT}/scripts/merge_profile.py"

ROOT="$(mktemp -d)"
trap 'rm -rf "$ROOT"' EXIT

# Parent with the required fields a child would inherit.
cat > "${ROOT}/base.yaml" <<'YAML'
name: base
chain:
  binary: gstable
  type: stablenet
nodes:
  validators: 4
  endpoints: 1
ports:
  base_p2p: 30301
  base_http: 8501
YAML

field() { python3 -c "import json,sys; d=json.load(sys.stdin); print(d$1)"; }

# ---- extends: resolves parent (the fix) --------------------------------------
describe "inheritance: extends resolves parent + child overrides"
cat > "${ROOT}/child-extends.yaml" <<'YAML'
name: child-extends
extends: base
nodes:
  validators: 2
YAML
out="$(python3 "$MERGE" "${ROOT}/child-extends.yaml" "$ROOT" "")"
assert_eq "$(printf '%s' "$out" | field "['chain']['binary']")" "gstable" "inherited chain.binary from parent"
assert_eq "$(printf '%s' "$out" | field "['nodes']['validators']")" "2" "child override wins (validators 4->2)"
assert_eq "$(printf '%s' "$out" | field "['nodes']['endpoints']")" "1" "inherited sibling preserved (endpoints)"
# meta key removed after merge
assert_eq "$(printf '%s' "$out" | python3 -c "import json,sys; print('extends' in json.load(sys.stdin))")" "False" "extends meta key stripped"

# ---- inherits: still works (no regression) -----------------------------------
describe "inheritance: inherits still resolves parent"
cat > "${ROOT}/child-inherits.yaml" <<'YAML'
name: child-inherits
inherits: base
nodes:
  validators: 7
YAML
out="$(python3 "$MERGE" "${ROOT}/child-inherits.yaml" "$ROOT" "")"
assert_eq "$(printf '%s' "$out" | field "['chain']['binary']")" "gstable" "inherits: inherited chain.binary"
assert_eq "$(printf '%s' "$out" | field "['nodes']['validators']")" "7" "inherits: child override wins"

# ---- missing parent -> error -------------------------------------------------
describe "inheritance: missing parent fails loudly"
cat > "${ROOT}/child-missing.yaml" <<'YAML'
name: child-missing
extends: __does_not_exist__
YAML
set +e
err_out="$(python3 "$MERGE" "${ROOT}/child-missing.yaml" "$ROOT" "" 2>&1 >/dev/null)"
rc=$?
set -e
assert_eq "$rc" "1" "missing parent exits 1"
assert_contains "$err_out" "not found" "error names the missing parent"

unit_summary
