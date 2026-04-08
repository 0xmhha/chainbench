#!/usr/bin/env bash
# Test: regression/c-anzeon/c-07-max-basefee
# RT-C-07 — baseFee가 MaxBaseFee(20,000 Gwei) 상한을 초과하지 않음
#
# MaxBaseFee = 20,000,000,000,000,000 wei (20,000 Gwei)는 매우 큰 값이라
# 실제 부하 테스트로 도달은 비현실적. 단위 검증만 수행
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/c-anzeon/c-07-max-basefee"

MAX_BASE_FEE=20000000000000000  # 20,000 Gwei

# 현재 baseFee가 상한 이하인지만 확인
current_bf=$(get_base_fee "1")
assert_true "$( [[ $current_bf -le $MAX_BASE_FEE ]] && echo true || echo false )" "current baseFee ($current_bf) <= MaxBaseFee ($MAX_BASE_FEE)"

# 최근 10블록에 대해 상한 준수 확인
current=$(block_number "1")
for i in 0 1 2 3 4 5 6 7 8 9; do
  n=$(( current - i ))
  (( n < 0 )) && break
  bf=$(hex_to_dec "$(rpc "1" "eth_getBlockByNumber" "[\"$(dec_to_hex "$n")\", false]" | json_get - 'result.baseFeePerGas')")
  if (( bf > MAX_BASE_FEE )); then
    _assert_fail "block $n baseFee $bf > MaxBaseFee $MAX_BASE_FEE"
  fi
done
_assert_pass "last 10 blocks all have baseFee <= MaxBaseFee"

printf '[INFO]  Note: reaching MaxBaseFee requires sustained heavy load — skipped in regression suite\n' >&2

test_result
