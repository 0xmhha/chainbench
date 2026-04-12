#!/usr/bin/env bash
# tests/unit/tests/common-resolve-logrot.sh - Test resolve_logrot discovery chain
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/assert.sh"

CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
export CHAINBENCH_DIR
export CHAINBENCH_QUIET=1
source "${CHAINBENCH_DIR}/lib/common.sh"

TMPDIR_ROOT="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_ROOT"' EXIT

# Create mock logrot binary
MOCK_LOGROT="${TMPDIR_ROOT}/mock-logrot"
cat > "$MOCK_LOGROT" <<'SCRIPT'
#!/usr/bin/env bash
echo "mock-logrot"
SCRIPT
chmod +x "$MOCK_LOGROT"

# Create mock gstable
MOCK_BINARY="${TMPDIR_ROOT}/mock-binary/gstable"
mkdir -p "$(dirname "$MOCK_BINARY")"
cp "${SCRIPT_DIR}/fixtures/mock-gstable" "$MOCK_BINARY"
chmod +x "$MOCK_BINARY"

# Fake git that always fails (for isolation)
NOGIT="${TMPDIR_ROOT}/nogit"
mkdir -p "$NOGIT"
cat > "${NOGIT}/git" <<'GITEOF'
#!/usr/bin/env bash
exit 1
GITEOF
chmod +x "${NOGIT}/git"

# ---- Test 1: explicit logrot_path -------------------------------------------
describe "resolve_logrot: explicit path"
result="$(PATH="${NOGIT}" resolve_logrot "$MOCK_BINARY" "$MOCK_LOGROT" 2>/dev/null)"
assert_eq "$result" "$MOCK_LOGROT" "returns explicit logrot path"

# ---- Test 2: dirname(binary_path)/logrot ------------------------------------
describe "resolve_logrot: sibling of binary"
cp "$MOCK_LOGROT" "$(dirname "$MOCK_BINARY")/logrot"
result="$(PATH="${NOGIT}:/usr/bin" resolve_logrot "$MOCK_BINARY" "" 2>/dev/null)"
assert_eq "$result" "$(dirname "$MOCK_BINARY")/logrot" "finds logrot next to binary"
rm -f "$(dirname "$MOCK_BINARY")/logrot"

# ---- Test 3: git-root/build/bin/logrot --------------------------------------
describe "resolve_logrot: git-root/build/bin/logrot"
FAKE_GIT_ROOT="${TMPDIR_ROOT}/gitroot"
mkdir -p "${FAKE_GIT_ROOT}/build/bin"
cp "$MOCK_LOGROT" "${FAKE_GIT_ROOT}/build/bin/logrot"

FAKE_GIT_CMD="${TMPDIR_ROOT}/fakegit"
mkdir -p "$FAKE_GIT_CMD"
cat > "${FAKE_GIT_CMD}/git" <<GITEOF
#!/usr/bin/env bash
if [[ "\$1" == "rev-parse" && "\$2" == "--show-toplevel" ]]; then
  echo "${FAKE_GIT_ROOT}"
  exit 0
fi
exit 1
GITEOF
chmod +x "${FAKE_GIT_CMD}/git"

result="$(PATH="${FAKE_GIT_CMD}:${PATH}" resolve_logrot "/different/bin/gstable" "" 2>/dev/null)"
assert_eq "$result" "${FAKE_GIT_ROOT}/build/bin/logrot" "finds logrot at git-root"

# ---- Test 4: auto-build from source (mock go) -------------------------------
describe "resolve_logrot: auto-build from source"
BUILD_ROOT="${TMPDIR_ROOT}/buildroot"
mkdir -p "${BUILD_ROOT}/cmd/logrot" "${BUILD_ROOT}/build/bin"
echo "package main" > "${BUILD_ROOT}/cmd/logrot/main.go"

# Mock go that creates a fake binary
FAKE_GO="${TMPDIR_ROOT}/fakego"
mkdir -p "$FAKE_GO"
cat > "${FAKE_GO}/go" <<GOEOF
#!/usr/bin/env bash
# Mock go build: extract -o flag and create the output binary
for arg in "\$@"; do :; done  # iterate to shift
i=0
for arg in "\$@"; do
  if [[ "\$arg" == "-o" ]]; then
    i=1
  elif [[ "\$i" == "1" ]]; then
    mkdir -p "\$(dirname "\$arg")"
    cp "${MOCK_LOGROT}" "\$arg"
    chmod +x "\$arg"
    break
  fi
done
GOEOF
chmod +x "${FAKE_GO}/go"

FAKE_GIT_BUILD="${TMPDIR_ROOT}/fakegit-build"
mkdir -p "$FAKE_GIT_BUILD"
cat > "${FAKE_GIT_BUILD}/git" <<GITEOF2
#!/usr/bin/env bash
if [[ "\$1" == "rev-parse" && "\$2" == "--show-toplevel" ]]; then
  echo "${BUILD_ROOT}"
  exit 0
fi
exit 1
GITEOF2
chmod +x "${FAKE_GIT_BUILD}/git"

# Remove any pre-existing logrot
rm -f "${BUILD_ROOT}/build/bin/logrot"
mkdir -p "${BUILD_ROOT}/state"
result="$(PATH="${FAKE_GIT_BUILD}:${FAKE_GO}:/usr/bin:/bin" CHAINBENCH_DIR="${BUILD_ROOT}" resolve_logrot "/different/bin/gstable" "" 2>/dev/null)"
assert_eq "$result" "${BUILD_ROOT}/build/bin/logrot" "builds and returns logrot"

# ---- Test 5: source present but go unavailable -> skip ----------------------
describe "resolve_logrot: no go command skips build"
NOGO="${TMPDIR_ROOT}/nogo"
mkdir -p "$NOGO"
rm -f "${BUILD_ROOT}/build/bin/logrot"
result="$(PATH="${FAKE_GIT_BUILD}:${NOGO}:/usr/bin:/bin" CHAINBENCH_DIR="${BUILD_ROOT}" resolve_logrot "/different/bin/gstable" "" 2>/dev/null)"
assert_empty "$result" "empty when go not available and no other source"

# ---- Test 6: PATH hit -------------------------------------------------------
describe "resolve_logrot: PATH lookup as last resort"
PATH_DIR="${TMPDIR_ROOT}/pathlogrot"
mkdir -p "$PATH_DIR"
cp "$MOCK_LOGROT" "${PATH_DIR}/logrot"
result="$(PATH="${NOGIT}:${PATH_DIR}" resolve_logrot "/different/bin/gstable" "" 2>/dev/null)"
assert_eq "$result" "${PATH_DIR}/logrot" "resolves from PATH"

# ---- Test 7: all miss -> empty string ---------------------------------------
describe "resolve_logrot: returns empty when not found"
result="$(PATH="${NOGIT}:/nonexistent" resolve_logrot "/different/bin/gstable" "" 2>/dev/null)"
assert_empty "$result" "empty string on complete miss"

unit_summary
