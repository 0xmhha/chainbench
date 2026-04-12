#!/usr/bin/env bash
# tests/unit/tests/test-meta-parse.sh - Test frontmatter parser
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/assert.sh"

CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
export CHAINBENCH_DIR
source "${CHAINBENCH_DIR}/lib/test_meta.sh"

FIXTURES="${SCRIPT_DIR}/fixtures"

# ---- Test 1: parse frontmatter correctly ------------------------------------
describe "meta: parse valid frontmatter"
meta="$(cb_parse_meta "${FIXTURES}/test-with-meta.sh")"
id="$(echo "$meta" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))")"
assert_eq "$id" "RT-TEST-01" "id field parsed"

name="$(echo "$meta" | python3 -c "import sys,json; print(json.load(sys.stdin).get('name',''))")"
assert_eq "$name" "Sample test with frontmatter" "name field parsed"

tags="$(echo "$meta" | python3 -c "import sys,json; print(json.load(sys.stdin).get('tags',[])[0])")"
assert_eq "$tags" "tx" "first tag is tx"

est="$(echo "$meta" | python3 -c "import sys,json; print(json.load(sys.stdin).get('estimated_seconds',0))")"
assert_eq "$est" "15" "estimated_seconds parsed"

# ---- Test 2: no frontmatter -> empty JSON -----------------------------------
describe "meta: no frontmatter returns empty object"
meta2="$(cb_parse_meta "${FIXTURES}/test-without-meta.sh")"
keys="$(echo "$meta2" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))")"
assert_eq "$keys" "0" "empty JSON object for no frontmatter"

# ---- Test 3: nonexistent file -> empty JSON ---------------------------------
describe "meta: nonexistent file returns empty object"
meta3="$(cb_parse_meta "/nonexistent/path.sh" 2>/dev/null)"
keys3="$(echo "$meta3" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))")"
assert_eq "$keys3" "0" "empty JSON for nonexistent file"

unit_summary
