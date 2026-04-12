#!/usr/bin/env bash
# tests/unit/tests/common-resolve-binary.sh - Pin down resolve_binary behavior
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/assert.sh"

# Source the function under test
CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
export CHAINBENCH_DIR
# Suppress info/warn output during tests
export CHAINBENCH_QUIET=1
source "${CHAINBENCH_DIR}/lib/common.sh"

# Setup: create temp directory structure for testing
TMPDIR_ROOT="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_ROOT"' EXIT

MOCK_BIN="${SCRIPT_DIR}/fixtures/mock-gstable"

# ---- Test 1: explicit path valid and executable ----------------------------
describe "resolve_binary: explicit path valid and executable"
result="$(resolve_binary "gstable" "$MOCK_BIN" 2>/dev/null)"
assert_eq "$result" "$MOCK_BIN" "returns explicit path when executable"

# ---- Test 2: explicit path not executable -> auto-detect fallback ----------
describe "resolve_binary: explicit path not executable falls back"
NOT_EXEC="${TMPDIR_ROOT}/not-executable"
touch "$NOT_EXEC"
chmod -x "$NOT_EXEC"
# Create a fallback in a fake git root
FAKE_GIT="${TMPDIR_ROOT}/fakegit"
mkdir -p "${FAKE_GIT}/build/bin"
cp "$MOCK_BIN" "${FAKE_GIT}/build/bin/gstable"
chmod +x "${FAKE_GIT}/build/bin/gstable"
# override git rev-parse by prepending a fake git to PATH
FAKE_GIT_CMD="${TMPDIR_ROOT}/bin"
mkdir -p "$FAKE_GIT_CMD"
cat > "${FAKE_GIT_CMD}/git" <<GITEOF
#!/usr/bin/env bash
if [[ "\$1" == "rev-parse" && "\$2" == "--show-toplevel" ]]; then
  echo "${FAKE_GIT}"
  exit 0
fi
exec /usr/bin/git "\$@"
GITEOF
chmod +x "${FAKE_GIT_CMD}/git"

result="$(PATH="${FAKE_GIT_CMD}:${PATH}" resolve_binary "gstable" "$NOT_EXEC" 2>/dev/null)"
assert_eq "$result" "${FAKE_GIT}/build/bin/gstable" "falls back to git-root/build/bin/"

# ---- Test 3: git-root/build/bin/<binary> -----------------------------------
describe "resolve_binary: auto-detect from git-root/build/bin/"
result="$(PATH="${FAKE_GIT_CMD}:${PATH}" resolve_binary "gstable" "" 2>/dev/null)"
assert_eq "$result" "${FAKE_GIT}/build/bin/gstable" "finds binary at git-root"

# ---- Test 4: PWD/build/bin/<binary> (non-git fallback) ---------------------
describe "resolve_binary: fallback to PWD/build/bin/"
PWD_ROOT="${TMPDIR_ROOT}/pwdtest"
mkdir -p "${PWD_ROOT}/build/bin"
cp "$MOCK_BIN" "${PWD_ROOT}/build/bin/gstable"
chmod +x "${PWD_ROOT}/build/bin/gstable"
# Use a git that fails (simulating non-git project)
NOGIT="${TMPDIR_ROOT}/nogit"
mkdir -p "$NOGIT"
cat > "${NOGIT}/git" <<'GITEOF'
#!/usr/bin/env bash
exit 1
GITEOF
chmod +x "${NOGIT}/git"

result="$(cd "$PWD_ROOT" && PATH="${NOGIT}:${PATH}" resolve_binary "gstable" "" 2>/dev/null)"
assert_eq "$result" "${PWD_ROOT}/build/bin/gstable" "finds binary at PWD/build/bin/"

# ---- Test 5: PATH hit with warning ----------------------------------------
describe "resolve_binary: PATH lookup as last resort"
PATH_DIR="${TMPDIR_ROOT}/pathbin"
mkdir -p "$PATH_DIR"
cp "$MOCK_BIN" "${PATH_DIR}/gstable"
chmod +x "${PATH_DIR}/gstable"

# cd to a dir with no build/bin, use nogit
result="$(cd "$TMPDIR_ROOT" && PATH="${NOGIT}:${PATH_DIR}" resolve_binary "gstable" "" 2>/dev/null)"
assert_eq "$result" "${PATH_DIR}/gstable" "resolves from PATH"

# ---- Test 6: complete miss -> exit 1 --------------------------------------
describe "resolve_binary: returns 1 when binary not found"
rc=0
(cd "$TMPDIR_ROOT" && PATH="${NOGIT}:/nonexistent" resolve_binary "nonexistent-bin" "" 2>/dev/null) || rc=$?
assert_eq "$rc" "1" "returns exit code 1 on miss"

unit_summary
