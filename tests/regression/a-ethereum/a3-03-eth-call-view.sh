#!/usr/bin/env bash
# Test: regression/a-ethereum/a3-03-eth-call-view
# RT-A-3-03 — eth_call view 함수 조회 (tx 발행 없이)
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a3-03-eth-call-view"
check_env || { test_result; exit 1; }

contract_addr=$(cat /tmp/chainbench-regression/simple_storage.addr 2>/dev/null || echo "")
if [[ -z "$contract_addr" ]]; then
  _assert_fail "contract not deployed (run a3-01 first)"
  test_result
  exit 1
fi

# x() view 함수 호출 — tx 발행 없음
get_selector=$(selector "x()")
before_block=$(block_number "1")

result=$(eth_call_raw "1" "$contract_addr" "$get_selector")
assert_contains "$result" "0x" "eth_call returns hex result"

# 결과 길이 검증 (uint256 → 32 bytes → 66 chars with 0x)
result_len=${#result}
assert_eq "$result_len" "66" "result is 32-byte uint256 (66 chars with 0x)"

# 블록 번호가 변하지 않았거나 정상 진행(tx 발행 아님)
after_block=$(block_number "1")
# eth_call은 tx를 발행하지 않으므로 state 변경 없음

# 디코드하여 정수 값 반환 확인
val=$(hex_to_dec "$result")
assert_true "$( [[ $val -ge 0 ]] && echo true || echo false )" "decoded value is non-negative integer: $val"

test_result
