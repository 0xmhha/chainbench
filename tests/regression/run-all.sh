#!/usr/bin/env bash
# tests/regression/run-all.sh — 회귀 테스트 suite 일괄 실행
#
# Usage:
#   ./tests/regression/run-all.sh                  # 전체 실행
#   ./tests/regression/run-all.sh a-ethereum       # 특정 카테고리만
#   ./tests/regression/run-all.sh a-ethereum b-wbft  # 복수 카테고리
#
# 사전 조건:
#   chainbench init --profile regression
#   chainbench start
#
set -euo pipefail

CHAINBENCH_DIR="${CHAINBENCH_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
REGRESSION_DIR="${CHAINBENCH_DIR}/tests/regression"

# 카테고리 목록 (기본)
DEFAULT_CATEGORIES=(
  "a-ethereum"
  "b-wbft"
  "c-anzeon"
  "d-fee-delegation"
  "e-blacklist-authorized"
  "f-system-contracts"
  "g-api"
)

if [[ $# -gt 0 ]]; then
  CATEGORIES=("$@")
else
  CATEGORIES=("${DEFAULT_CATEGORIES[@]}")
fi

# 통계
total=0
passed=0
failed=0
skipped=0
declare -a failed_tests=()

printf '\n========================================\n'
printf 'go-stablenet v2 Regression Test Suite\n'
printf '========================================\n\n'
printf 'Categories: %s\n\n' "${CATEGORIES[*]}"

start_ts=$(date +%s)

# 각 카테고리 실행
for category in "${CATEGORIES[@]}"; do
  cat_dir="${REGRESSION_DIR}/${category}"
  if [[ ! -d "$cat_dir" ]]; then
    printf '[SKIP]  Category %s: directory not found\n' "$category" >&2
    continue
  fi

  printf '\n--- Category: %s ---\n' "$category"

  # 정렬된 순서로 실행
  for test_file in "$cat_dir"/*.sh; do
    [[ ! -f "$test_file" ]] && continue
    total=$(( total + 1 ))
    tname=$(basename "$test_file" .sh)
    printf '  Running %s ... ' "$tname"

    if bash "$test_file" >/tmp/regression_out_$$.log 2>&1; then
      passed=$(( passed + 1 ))
      printf 'PASS\n'
    else
      rc=$?
      if grep -q "SKIP" /tmp/regression_out_$$.log; then
        skipped=$(( skipped + 1 ))
        printf 'SKIP\n'
      else
        failed=$(( failed + 1 ))
        failed_tests+=("$category/$tname")
        printf 'FAIL (rc=%d)\n' "$rc"
        # 실패 상세 마지막 10 라인만 출력
        printf '    ---- last 10 lines ----\n'
        tail -n 10 /tmp/regression_out_$$.log | sed 's/^/    /'
        printf '    ------------------------\n'
      fi
    fi
    rm -f /tmp/regression_out_$$.log
  done
done

end_ts=$(date +%s)
duration=$(( end_ts - start_ts ))

# 결과 요약
printf '\n========================================\n'
printf 'Regression Test Summary\n'
printf '========================================\n'
printf 'Total:    %d\n' "$total"
printf 'Passed:   %d\n' "$passed"
printf 'Failed:   %d\n' "$failed"
printf 'Skipped:  %d\n' "$skipped"
printf 'Duration: %ds\n' "$duration"
printf '========================================\n'

if [[ $failed -gt 0 ]]; then
  printf '\nFailed tests:\n'
  for t in "${failed_tests[@]}"; do
    printf '  - %s\n' "$t"
  done
  exit 1
fi

exit 0
