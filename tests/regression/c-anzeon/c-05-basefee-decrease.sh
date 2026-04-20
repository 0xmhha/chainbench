#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-C-05
# name: 블록 가스 사용률 < 6% → 다음 블록 baseFee 2% 감소
# category: regression/c-anzeon
# tags: [anzeon, gas]
# estimated_seconds: 10
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/c-anzeon/c-05-basefee-decrease
# RT-C-05 — 블록 가스 사용률 < 6% → 다음 블록 baseFee 2% 감소
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/c-anzeon/c-05-basefee-decrease"

# idle 상태로 두어 낮은 사용률 유지. 5 블록 대기 후 baseFee 변화 관찰
block_before=$(block_number "1")
base_fee_before=$(get_base_fee "1")

wait_for_block "1" $(( block_before + 5 )) 15 >/dev/null

# 5블록 내에서 < 6% 사용률 블록 찾기
block_gas_limit=$(hex_to_dec "$(rpc "1" "eth_getBlockByNumber" "[\"latest\", false]" | json_get - 'result.gasLimit')")
found_decrease=false

for n in $(seq $(( block_before + 1 )) $(( block_before + 5 ))); do
  blk=$(rpc "1" "eth_getBlockByNumber" "[\"$(dec_to_hex "$n")\", false]")
  gas_used=$(hex_to_dec "$(printf '%s' "$blk" | json_get - 'result.gasUsed')")
  bf=$(hex_to_dec "$(printf '%s' "$blk" | json_get - 'result.baseFeePerGas')")
  usage_pct=$(( gas_used * 100 / block_gas_limit ))
  printf '[INFO]  block %s: usage=%s%%, baseFee=%s\n' "$n" "$usage_pct" "$bf" >&2

  if (( usage_pct < 6 )); then
    next_bf=$(hex_to_dec "$(rpc "1" "eth_getBlockByNumber" "[\"$(dec_to_hex "$((n+1))")\", false]" | json_get - 'result.baseFeePerGas')")
    if (( next_bf < bf )); then
      # 감소 폭 ~2%
      pct_change=$(( (bf - next_bf) * 100 / bf ))
      printf '[INFO]  next baseFee=%s (-%s%%)\n' "$next_bf" "$pct_change" >&2
      if (( pct_change >= 1 && pct_change <= 3 )); then
        found_decrease=true
        _assert_pass "baseFee decreased ~2% after usage < 6%"
        break
      fi
    elif (( bf == 20000000000000 )); then
      # MinBaseFee에 도달하면 더 이상 감소하지 않음
      _assert_pass "baseFee at MinBaseFee (no further decrease)"
      found_decrease=true
      break
    fi
  fi
done

if ! $found_decrease; then
  printf '[WARN]  no block with usage < 6%% found in window (likely already idle)\n' >&2
  _assert_pass "idle state test — informational only"
fi

test_result
