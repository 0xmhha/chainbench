#!/usr/bin/env bash
# tests/unit/run.sh - Unit test runner for chainbench
# Iterates tests/*.sh, executes each in a subshell, and reports pass/fail totals.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTS_DIR="${SCRIPT_DIR}/tests"

passed=0
failed=0
errors=()

# Discover and run all test files
for test_file in "${TESTS_DIR}"/*.sh; do
  [[ -f "$test_file" ]] || continue

  test_name="$(basename "$test_file")"
  printf '\n\033[1m━━━ %s ━━━\033[0m\n' "$test_name" >&2

  if ( source "$test_file" ); then
    (( ++passed )) || true
    printf '\033[0;32m  ⟹ PASSED\033[0m\n' >&2
  else
    (( ++failed )) || true
    errors+=("$test_name")
    printf '\033[0;31m  ⟹ FAILED\033[0m\n' >&2
  fi
done

# Summary
total=$(( passed + failed ))
echo "" >&2
printf '\033[1m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m\n' >&2
printf '  Total: %d | Passed: %d | Failed: %d\n' "$total" "$passed" "$failed" >&2
printf '\033[1m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m\n' >&2

if [[ "$failed" -gt 0 ]]; then
  printf '\033[0;31mFailed tests:\033[0m\n' >&2
  for e in "${errors[@]}"; do
    printf '  - %s\n' "$e" >&2
  done
  exit 1
fi

exit 0
