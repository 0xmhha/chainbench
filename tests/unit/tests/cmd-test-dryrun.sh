#!/usr/bin/env bash
# tests/unit/tests/cmd-test-dryrun.sh - Test dry-run mode
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/assert.sh"

CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
export CHAINBENCH_DIR

TMPDIR_ROOT="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_ROOT"' EXIT

# Create a fake test directory with fixture scripts
FAKE_CB="${TMPDIR_ROOT}/cb"
mkdir -p "${FAKE_CB}/tests/mycat" "${FAKE_CB}/lib" "${FAKE_CB}/state"

# Copy required libs
cp "${CHAINBENCH_DIR}/lib/common.sh" "${FAKE_CB}/lib/"
cp "${CHAINBENCH_DIR}/lib/test_meta.sh" "${FAKE_CB}/lib/"
cp "${CHAINBENCH_DIR}/lib/cmd_test.sh" "${FAKE_CB}/lib/"

# Create test scripts with and without frontmatter
cat > "${FAKE_CB}/tests/mycat/t1-with-meta.sh" <<'SCRIPT'
#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-T-01
# name: Test with meta
# tags: [unit]
# estimated_seconds: 5
# depends_on: []
# ---end-meta---
echo "test1"
SCRIPT
chmod +x "${FAKE_CB}/tests/mycat/t1-with-meta.sh"

cat > "${FAKE_CB}/tests/mycat/t2-no-meta.sh" <<'SCRIPT'
#!/usr/bin/env bash
# Description: Test without meta
echo "test2"
SCRIPT
chmod +x "${FAKE_CB}/tests/mycat/t2-no-meta.sh"

# Run dry-run via cmd_test.sh
run_dryrun() {
  CHAINBENCH_DIR="${FAKE_CB}" CHAINBENCH_QUIET=0 bash -c '
    export CHAINBENCH_DIR
    unset _CB_CMD_TEST_SH_LOADED
    set -- run mycat --dry-run --format json
    source "$CHAINBENCH_DIR/lib/cmd_test.sh"
  ' 2>/dev/null
}

# ---- Test 1: dry-run produces JSON output ------------------------------------
describe "dryrun: produces JSON output"
output="$(run_dryrun)"
is_json="$(echo "$output" | python3 -c "import sys,json; json.load(sys.stdin); print('yes')" 2>/dev/null || echo "no")"
assert_eq "$is_json" "yes" "output is valid JSON"

# ---- Test 2: lists scripts without executing ---------------------------------
describe "dryrun: lists scripts without executing"
total="$(echo "$output" | python3 -c "import sys,json; print(json.load(sys.stdin).get('total_scripts',0))")"
assert_eq "$total" "2" "two scripts listed"

# ---- Test 3: includes metadata for scripts with frontmatter ------------------
describe "dryrun: includes metadata from frontmatter"
first_id="$(echo "$output" | python3 -c "
import sys,json
d = json.load(sys.stdin)
scripts = d.get('scripts', [])
for s in scripts:
    meta = s.get('meta', {})
    if meta.get('id'):
        print(meta['id'])
        break
else:
    print('')
")"
assert_eq "$first_id" "RT-T-01" "frontmatter id included"

# ---- Test 4: scripts without meta have empty meta ----------------------------
describe "dryrun: scripts without meta show empty meta"
nometa_keys="$(echo "$output" | python3 -c "
import sys,json
d = json.load(sys.stdin)
for s in d.get('scripts', []):
    if 'no-meta' in s.get('script', ''):
        print(len(s.get('meta', {})))
        break
")"
assert_eq "$nometa_keys" "0" "no-meta script has empty meta"

unit_summary
