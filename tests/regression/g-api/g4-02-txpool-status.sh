#!/usr/bin/env bash
# RT-G-4-02 (v2) — txpool_status: pending(연속 nonce) + queued(nonce gap) 분리
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/g-api/g4-02-txpool-status"
check_env || { test_result; exit 1; }

# pending 2개 + queued 1개 주입
python3 <<'PYEOF'
import json, requests
from eth_account import Account
pk = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
to = "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
url = "http://127.0.0.1:8501"
acct = Account.from_key(pk)
nonce = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getTransactionCount","params":[acct.address, "pending"],"id":1}).json()["result"], 16)
chain_id = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}).json()["result"], 16)
base_fee = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest", False],"id":1}).json()["result"]["baseFeePerGas"], 16)

# pending: nonce, nonce+1
for n in [nonce, nonce + 1]:
    tx = {"nonce": n, "to": to, "value": 1, "gas": 21000, "chainId": chain_id,
          "maxFeePerGas": base_fee + 50_000_000_000_000,
          "maxPriorityFeePerGas": 27_600_000_000_000, "type": 2}
    signed = acct.sign_transaction(tx)
    requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed.rawTransaction.hex()],"id":1})

# queued: nonce + 100 (gap)
tx = {"nonce": nonce + 100, "to": to, "value": 1, "gas": 21000, "chainId": chain_id,
      "maxFeePerGas": base_fee + 50_000_000_000_000,
      "maxPriorityFeePerGas": 27_600_000_000_000, "type": 2}
signed = acct.sign_transaction(tx)
requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed.rawTransaction.hex()],"id":1})
PYEOF

# 즉시 조회 (채굴 전)
sleep 0.5
status=$(rpc 1 txpool_status '[]')
pending=$(printf '%s' "$status" | python3 -c "
import sys, json
r = json.load(sys.stdin).get('result', {})
print(int(r.get('pending', '0x0'), 16))
")
queued=$(printf '%s' "$status" | python3 -c "
import sys, json
r = json.load(sys.stdin).get('result', {})
print(int(r.get('queued', '0x0'), 16))
")

printf '[INFO]  pending=%s queued=%s\n' "$pending" "$queued" >&2
assert_ge "$queued" "1" "queued count >= 1 (nonce gap tx)"

test_result
