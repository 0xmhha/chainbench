#!/usr/bin/env bash
# tests/unit/tests/smoke-meta.sh - Verify the unit test framework itself works
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/assert.sh"

describe "assert_eq works correctly"
assert_eq "hello" "hello" "identical strings match"
assert_eq "" "" "empty strings match"
assert_eq "123" "123" "numeric strings match"

describe "assert_neq works correctly"
assert_neq "a" "b" "different strings differ"
assert_neq "" "notempty" "empty vs non-empty differ"

describe "assert_empty works correctly"
assert_empty "" "empty string is empty"

describe "assert_nonempty works correctly"
assert_nonempty "x" "non-empty string is non-empty"

describe "assert_contains works correctly"
assert_contains "hello world" "world" "substring found"
assert_contains "foobar" "oob" "middle substring found"

describe "assert_file_exists works correctly"
assert_file_exists "${BASH_SOURCE[0]}" "this test file exists"

describe "assert_exit_code works correctly"
assert_exit_code 0 true
assert_exit_code 1 false

unit_summary
