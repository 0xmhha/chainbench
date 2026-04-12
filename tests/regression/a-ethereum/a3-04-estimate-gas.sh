#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-A-3-04
# name: eth_estimateGas 정상 동작
# category: regression/a-ethereum
# tags: [contract, estimateGas]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: [RT-A-3-01]
# ---end-meta---
# Test: regression/a-ethereum/a3-04-estimate-gas
# RT-A-3-04 — eth_estimateGas 정상 동작
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a3-04-estimate-gas"
check_env || { test_result; exit 1; }

# 단순 송금 tx에 대한 eth_estimateGas
estimate=$(rpc "1" "eth_estimateGas" "[{\"from\":\"${TEST_ACC_A_ADDR}\",\"to\":\"${TEST_ACC_B_ADDR}\",\"value\":\"0x1\"}]" | json_get - result)
assert_not_empty "$estimate" "eth_estimateGas returned a value"
estimate_dec=$(hex_to_dec "$estimate")
printf '[INFO]  estimateGas for simple transfer = %s\n' "$estimate_dec" >&2

# 단순 송금은 21000 가스 이상
assert_ge "$estimate_dec" "21000" "estimate >= 21000 (intrinsic gas)"

# 컨트랙트 호출에 대한 estimate
contract_addr=$(cat /tmp/chainbench-regression/simple_storage.addr 2>/dev/null || echo "")
if [[ -n "$contract_addr" ]]; then
  set_sel=$(selector "set(uint256)")
  data="${set_sel}$(pad_uint256 100 | sed 's/^0x//')"
  estimate2=$(rpc "1" "eth_estimateGas" "[{\"from\":\"${TEST_ACC_A_ADDR}\",\"to\":\"${contract_addr}\",\"data\":\"${data}\"}]" | json_get - result)
  estimate2_dec=$(hex_to_dec "$estimate2")
  assert_gt "$estimate2_dec" "21000" "contract call estimate > 21000"
fi

test_result
