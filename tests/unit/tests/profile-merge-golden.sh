#!/usr/bin/env bash
# tests/unit/tests/profile-merge-golden.sh
# Characterization (golden) test for the profile YAML->JSON merge.
#
# Locks the EXACT merged output of representative profiles so the P2-1
# extraction of the embedded Python (lib/profile.sh -> scripts/merge_profile.py)
# is provably behavior-preserving: the same merged JSON must come out before and
# after the refactor.
#
# Profiles exercised:
#   default   - standalone (scalars, nested maps, sequences, deep structures)
#   regression- standalone, large/complex
#   minimal   - inherits: default (inheritance + leaf override: validators 4->2)
#   large     - inherits: default (inheritance + override: validators ->7)
#
# Determinism: CHAINBENCH_DIR points at an empty dir so no machine-local
# state/local-config.yaml overlay is merged. Output is normalized
# (indent=2, sort_keys) before comparison.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/assert.sh"

REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
GOLDEN_DIR="${SCRIPT_DIR}/golden/profile-merged"

# Source the loader (CHAINBENCH_DIR is overridden per-call for overlay-free runs).
export CHAINBENCH_QUIET=1
unset _CHAINBENCH_PROFILE_SH_LOADED 2>/dev/null || true
# shellcheck disable=SC1091
source "${REPO_ROOT}/lib/common.sh"
# shellcheck disable=SC1091
source "${REPO_ROOT}/lib/profile.sh"

EMPTY_DIR="$(mktemp -d)"
trap 'rm -rf "$EMPTY_DIR"' EXIT

normalize() {
  python3 -c "import json,sys; json.dump(json.load(sys.stdin), sys.stdout, indent=2, sort_keys=True, ensure_ascii=False); sys.stdout.write('\n')"
}

for profile in default regression minimal large; do
  describe "profile merge golden: ${profile}"
  golden="${GOLDEN_DIR}/${profile}.json"
  assert_file_exists "$golden" "golden file present for ${profile}"

  raw="$(CHAINBENCH_DIR="$EMPTY_DIR" _cb_python_merge_yaml "${REPO_ROOT}/profiles/${profile}.yaml")"
  got="$(printf '%s\n' "$raw" | normalize)"
  want="$(cat "$golden")"

  assert_eq "$got" "$want" "merged JSON matches golden for ${profile}"
done

unit_summary
