#!/usr/bin/env bash
# Test: regression/c-anzeon/c-03-basefee-increase
# RT-C-03 — 블록 가스 사용률 > 20% → 다음 블록 baseFee 2% 증가
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/c-anzeon/c-03-basefee-increase"
check_env || { test_result; exit 1; }

# 블록 gasLimit 조회
block_gas_limit_hex=$(rpc "1" "eth_getBlockByNumber" "[\"latest\", false]" | json_get - "result.gasLimit")
block_gas_limit=$(hex_to_dec "$block_gas_limit_hex")
target_gas_used=$(( block_gas_limit * 25 / 100 ))  # 25% 사용

printf '[INFO]  block gasLimit=%s, target >= 20%%: %s\n' "$block_gas_limit" "$target_gas_used" >&2

# 여러 tx로 목표 gasUsed 달성 — 단순 송금은 21000씩
# target_gas_used / 21000 = 필요 tx 개수
tx_count=$(( target_gas_used / 21000 + 1 ))
printf '[INFO]  sending %s txs to exceed 20%% block gas\n' "$tx_count" >&2

# 블록 번호와 baseFee 기록
block_before=$(block_number "1")
base_fee_before=$(get_base_fee "1")

# 다수 tx 전송
python3 <<PYEOF
import json, requests
from eth_account import Account

pk = "${TEST_ACC_A_PK}"
url = "http://127.0.0.1:8501"
acct = Account.from_key(pk)
nonce = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getTransactionCount","params":[acct.address, "pending"],"id":1}).json()["result"], 16)
chain_id = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}).json()["result"], 16)
base_fee = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest", False],"id":1}).json()["result"]["baseFeePerGas"], 16)

for i in range(${tx_count}):
    tx = {"nonce": nonce + i, "to": "0x70997970C51812dc3A010C7d01b50e0d17dc79C8", "value": 1, "gas": 21000, "chainId": chain_id,
          "maxFeePerGas": base_fee + 100_000_000_000_000,
          "maxPriorityFeePerGas": 27_600_000_000_000, "type": 2}
    signed = acct.sign_transaction(tx)
    requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed.raw_transaction.to_0x_hex()],"id":1})
PYEOF

# 다음 블록 대기 후 baseFee 변화 확인
wait_for_block "1" $(( block_before + 3 )) 10 >/dev/null

# 가스 사용률 > 20%인 블록을 찾고, 그 다음 블록의 baseFee 확인
found_increase=false
for n in $(seq $(( block_before + 1 )) $(( block_before + 5 ))); do
  blk=$(rpc "1" "eth_getBlockByNumber" "[\"$(dec_to_hex "$n")\", false]")
  gas_used=$(hex_to_dec "$(printf '%s' "$blk" | json_get - 'result.gasUsed')")
  bf=$(hex_to_dec "$(printf '%s' "$blk" | json_get - 'result.baseFeePerGas')")
  usage_pct=$(( gas_used * 100 / block_gas_limit ))
  printf '[INFO]  block %s: gasUsed=%s (%s%%), baseFee=%s\n' "$n" "$gas_used" "$usage_pct" "$bf" >&2

  if (( usage_pct > 20 )); then
    # 다음 블록 baseFee 조회
    next_blk=$(rpc "1" "eth_getBlockByNumber" "[\"$(dec_to_hex "$((n+1)))\", false]")
    next_bf=$(hex_to_dec "$(printf '%s' "$next_blk" | json_get - 'result.baseFeePerGas')")
    if (( next_bf > bf )); then
      pct_change=$(( (next_bf - bf) * 100 / bf ))
      printf '[INFO]  next baseFee=%s (+%s%%)\n' "$next_bf" "$pct_change" >&2
      if (( pct_change >= 1 && pct_change <= 3 )); then
        found_increase=true
        break
      fi
    fi
  fi
done

assert_true "$found_increase" "baseFee increased by ~2% after block usage > 20%"

test_result
