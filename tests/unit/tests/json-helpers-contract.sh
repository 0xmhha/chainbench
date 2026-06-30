#!/usr/bin/env bash
# tests/unit/tests/json-helpers-contract.sh
# Characterization test locking the public contract of lib/json_helpers.sh so
# the P2-1b backend unification (dual jq/python -> single python via
# scripts/json_backend.py) is behavior-preserving for real usage.
#
# Runs against whatever backend is active (jq when installed). All assertions
# except one hold on both backends — verified empirically. The exception is the
# boolean `false` read: the jq path returns empty because `jq '... // empty'`
# treats false as empty (a latent bug, never hit by current callers since
# pids.json stores no boolean fields); the python path returns "false". That
# one case is pinned against the python backend so it locks the CORRECT value
# the unification standardizes on.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/assert.sh"

REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
# shellcheck disable=SC1091
source "${REPO_ROOT}/lib/json_helpers.sh"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

FILE="${TMP}/d.json"
cat > "$FILE" <<'JSON'
{"nodes":{"1":{"http_port":8545,"name":"v1","up":true}},"count":3}
JSON

# ---- read ----
describe "cb_json_read"
assert_eq "$(cb_json_read "$FILE" "nodes.1.name" "NA")" "v1" "string"
assert_eq "$(cb_json_read "$FILE" "nodes.1.http_port" "0")" "8545" "int (nested numeric key)"
assert_eq "$(cb_json_read "$FILE" "nodes.1.up" "x")" "true" "bool true"
assert_eq "$(cb_json_read "$FILE" "nodes.1.missing" "DEF")" "DEF" "missing -> default"
assert_eq "$(cb_json_read "$FILE" "count" "0")" "3" "top-level int"

# bool false: locked against the python backend (the value the unification keeps).
describe "cb_json_read bool false (python contract)"
echo '{"flag":false}' > "${TMP}/f.json"
got_false="$(_CB_JSON_BACKEND=python3 cb_json_read "${TMP}/f.json" "flag" "X")"
assert_eq "$got_false" "false" "bool false reads as 'false' (not default)"

# ---- read_stdin ----
describe "cb_json_read_stdin"
assert_eq "$(echo '{"result":"0x1a"}' | cb_json_read_stdin "result" "NA")" "0x1a" "stdin scalar"
assert_eq "$(echo '{"a":{"b":7}}' | cb_json_read_stdin "a.b" "0")" "7" "stdin nested"

# ---- array_len ----
describe "cb_json_array_len"
echo '{"list":[1,2,3,4]}' > "${TMP}/a.json"
assert_eq "$(cb_json_array_len "${TMP}/a.json" "list")" "4" "array length"

# ---- write (each type) then read back ----
describe "cb_json_write round-trips"
cb_json_write "$FILE" "nodes.1.http_port" "9999"
assert_eq "$(cb_json_read "$FILE" "nodes.1.http_port")" "9999" "write int"
cb_json_write "$FILE" "nodes.1.name" "renamed"
assert_eq "$(cb_json_read "$FILE" "nodes.1.name")" "renamed" "write string"
cb_json_write "$FILE" "newkey" "42"
assert_eq "$(cb_json_read "$FILE" "newkey")" "42" "write new key"
# file stays valid JSON after writes
assert_eq "$(python3 -c "import json;json.load(open('$FILE'));print('ok')")" "ok" "valid JSON after writes"

# ---- merge then read back ----
describe "cb_json_merge"
cb_json_merge "$FILE" '{"merged":{"a":1},"count":99}'
assert_eq "$(cb_json_read "$FILE" "count")" "99" "merge overrides scalar"
assert_eq "$(cb_json_read "$FILE" "merged.a")" "1" "merge adds nested"
assert_eq "$(cb_json_read "$FILE" "nodes.1.name")" "renamed" "merge preserves untouched"

# ---- get_result ----
describe "cb_json_get_result"
assert_eq "$(cb_json_get_result '{"result":"0xabc"}')" "0xabc" "extracts result"
rc=0; cb_json_get_result '{"error":{"message":"boom"}}' >/dev/null 2>&1 || rc=$?
assert_eq "$rc" "1" "error -> exit 1"

# ---- has_error ----
describe "cb_json_has_error"
rc=0; cb_json_has_error '{"error":{"message":"x"}}' || rc=$?
assert_eq "$rc" "0" "has error -> exit 0"
rc=0; cb_json_has_error '{"result":"ok"}' || rc=$?
assert_eq "$rc" "1" "no error -> exit 1"

unit_summary
