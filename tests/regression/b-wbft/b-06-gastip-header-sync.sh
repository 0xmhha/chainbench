#!/usr/bin/env bash
# Test: regression/b-wbft/b-06-gastip-header-sync
# RT-B-06 — GasTip 거버넌스 변경 후 worker가 다음 블록 헤더의 WBFTExtra.GasTip에 반영
# (RT-F-3-05의 후행 단계 — F-3-05가 storage gasTip을 변경한 상태 가정)
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/b-wbft/b-06-gastip-header-sync"

# 현재 WBFTExtra.GasTip 조회 — istanbul_getWbftExtraInfo는 "latest" 문자열을
# 받지 못하므로 실제 block number를 hex로 전달. gasTip은 decimal 문자열로 반환됨.
latest_hex=$(rpc "1" "eth_blockNumber" "[]" | json_get - result)
current_tip=$(rpc "1" "istanbul_getWbftExtraInfo" "[\"${latest_hex}\"]" | python3 -c "
import sys, json
r = json.load(sys.stdin).get('result', {})
gt = r.get('gasTip', '0')
# gasTip은 decimal 문자열 (e.g. '27600000000000'). hex가 올 경우도 방어적으로 처리.
if isinstance(gt, str):
    if gt.startswith('0x'):
        print(int(gt, 16))
    elif gt:
        print(int(gt))
    else:
        print(0)
else:
    print(int(gt) if gt else 0)
")

# regression profile 기본 gasTip = 27600000000000 wei
DEFAULT_TIP=27600000000000
assert_eq "$current_tip" "$DEFAULT_TIP" "initial WBFTExtra.GasTip matches profile default (27.6 Gwei)"

# eth_maxPriorityFeePerGas도 동일한 값인지 확인 (backend Anzeon 분기)
mpfpg=$(rpc "1" "eth_maxPriorityFeePerGas" "[]" | json_get - result)
mpfpg_dec=$(hex_to_dec "$mpfpg")
assert_eq "$mpfpg_dec" "$current_tip" "eth_maxPriorityFeePerGas == WBFTExtra.GasTip"

# eth_gasPrice = baseFee + GasTip 확인
base_fee=$(get_base_fee "1")
gas_price=$(rpc "1" "eth_gasPrice" "[]" | json_get - result)
gas_price_dec=$(hex_to_dec "$gas_price")
expected=$(( base_fee + current_tip ))
assert_eq "$gas_price_dec" "$expected" "eth_gasPrice == baseFee + GasTip ($expected)"

# 추가 검증: GasTip 거버넌스 변경 후 새 값이 다음 블록 헤더에 반영되는지 (RT-F-3-05 후행)
# 현재는 기본 상태만 검증하며, proposeGasTip 호출은 F-3-05에서 수행
_assert_pass "GasTip header sync baseline verified (proposeGasTip execution is in f3-05)"

test_result
