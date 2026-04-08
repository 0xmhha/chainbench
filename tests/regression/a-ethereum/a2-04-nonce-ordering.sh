#!/usr/bin/env bash
# Test: regression/a-ethereum/a2-04-nonce-ordering
# RT-A-2-04 — Nonce 순서 보장
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a2-04-nonce-ordering"
check_env || { test_result; exit 1; }

# 3개 tx 연속 발행 (TEST_ACC_A → TEST_ACC_B)
python3 <<'PYEOF' > /tmp/nonce_tx_hashes.txt
import json, requests
from eth_account import Account

pk = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
to = "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
url = "http://127.0.0.1:8501"

acct = Account.from_key(pk)
nonce = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getTransactionCount","params":[acct.address, "pending"],"id":1}).json()["result"], 16)
chain_id = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}).json()["result"], 16)
base_fee = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest", False],"id":1}).json()["result"]["baseFeePerGas"], 16)

for i in range(3):
    tx = {"nonce": nonce + i, "to": to, "value": 1, "gas": 21000, "chainId": chain_id,
          "maxFeePerGas": base_fee + 50_000_000_000_000, "maxPriorityFeePerGas": 27_600_000_000_000, "type": 2}
    signed = acct.sign_transaction(tx)
    resp = requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed.rawTransaction.hex()],"id":1}).json()
    print(resp.get("result", "ERROR"))
PYEOF

mapfile -t tx_hashes < /tmp/nonce_tx_hashes.txt
assert_eq "${#tx_hashes[@]}" "3" "3 tx hashes returned"

# 모든 tx의 receipt 대기
for h in "${tx_hashes[@]}"; do
  status=$(wait_receipt "1" "$h" 30)
  assert_eq "$status" "success" "tx $h mined"
done

# 블록 포함 순서 검증 (nonce 오름차순)
declare -a nonces=()
for h in "${tx_hashes[@]}"; do
  n=$(rpc "1" "eth_getTransactionByHash" "[\"$h\"]" | json_get - "result.nonce")
  nonces+=("$(hex_to_dec "$n")")
done

# 오름차순 확인
ordered=true
for i in 1 2; do
  (( nonces[i] < nonces[i-1] )) && ordered=false
done
assert_true "$ordered" "nonces are in ascending order: ${nonces[*]}"

rm -f /tmp/nonce_tx_hashes.txt
test_result
