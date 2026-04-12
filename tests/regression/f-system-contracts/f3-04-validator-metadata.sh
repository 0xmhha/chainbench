#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-F-3-04
# name: validatorList() + validatorToOperator(v) + validatorToBlsKey(v) 다중 호출
# category: regression/f-system-contracts
# tags: [governance, validator]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/f-system-contracts/f3-04-validator-metadata
# RT-F-3-04 (v2) — validatorList() + validatorToOperator(v) + validatorToBlsKey(v) 다중 호출
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/f-system-contracts/f3-04-validator-metadata"
check_env || { test_result; exit 1; }

# (1) validatorList() → 검증자 배열
vl_sel=$(selector "validatorList()")
vl_result=$(eth_call_raw 1 "$GOV_VALIDATOR" "$vl_sel")

validators=$(printf '%s' "$vl_result" | python3 -c "
import sys
data = sys.stdin.read().strip()
if data.startswith('0x'):
    data = data[2:]
# ABI: offset (32), length (32), addresses (32 each)
# length 위치 = 32 bytes offset → skip 32, read 32
if len(data) < 128:
    print('')
else:
    length = int(data[64:128], 16)
    addrs = []
    for i in range(length):
        start = 128 + i * 64
        addr_hex = data[start:start+64][-40:]
        addrs.append('0x' + addr_hex)
    print(','.join(addrs))
")
printf '[INFO]  validatorList: %s\n' "$validators" >&2

# 비어있지 않은 결과
assert_contains "$validators" "0x" "validatorList returned addresses"

# 최소 4개 이상
IFS=',' read -ra addr_arr <<< "$validators"
assert_ge "${#addr_arr[@]}" "4" "at least 4 validators"

# (2) 각 validator에 대해 validatorToOperator(v)
# (3) 각 validator에 대해 validatorToBlsKey(v)
to_op_sel=$(selector "validatorToOperator(address)")
to_bls_sel=$(selector "validatorToBlsKey(address)")

checked=0
for v in "${addr_arr[@]}"; do
  [[ -z "$v" ]] && continue
  v_padded=$(pad_address "$v" | sed 's/^0x//')

  operator=$(eth_call_raw 1 "$GOV_VALIDATOR" "${to_op_sel}${v_padded}")
  bls_key=$(eth_call_raw 1 "$GOV_VALIDATOR" "${to_bls_sel}${v_padded}")

  printf '[INFO]  %s operator=%s blsLen=%d\n' "$v" "${operator:0:20}" "${#bls_key}" >&2

  # operator 주소가 non-zero (32 bytes padded → 24 zeros + 20 bytes)
  op_nonzero=$(printf '%s' "$operator" | python3 -c "
import sys
h = sys.stdin.read().strip()
if h.startswith('0x'):
    h = h[2:]
print('yes' if int(h, 16) != 0 else 'no')
")
  [[ "$op_nonzero" == "yes" ]] && checked=$(( checked + 1 ))
done

assert_ge "$checked" "4" "at least 4 validators have non-zero operator mapping"

test_result
