#!/usr/bin/env bash
# Test: regression/e-blacklist-authorized/e-06-precompile-transfer
# RT-E-06 — Precompile 주소로 value 전송 차단
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/e-blacklist-authorized/e-06-precompile-transfer"
check_env || { test_result; exit 1; }

# 대상 precompile 주소 (폐쇄망 Boho 활성 기준)
# 0x01 = ecrecover, 0x100 = secp256r1, 0xb00003 = AccountManager
declare -a precompiles=(
  "0x0000000000000000000000000000000000000001"
  "0x0000000000000000000000000000000000000100"
  "0x00000000000000000000000000000000000b0003"
)

fail_count=0
for addr in "${precompiles[@]}"; do
  err=$(python3 <<PYEOF 2>&1 || true
import json, requests
from eth_account import Account
pk = "${TEST_ACC_A_PK}"
url = "http://127.0.0.1:8501"
acct = Account.from_key(pk)
nonce = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getTransactionCount","params":[acct.address, "pending"],"id":1}).json()["result"], 16)
chain_id = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}).json()["result"], 16)
base_fee = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest", False],"id":1}).json()["result"]["baseFeePerGas"], 16)
tx = {"nonce": nonce, "to": "${addr}", "value": 1, "gas": 100000, "chainId": chain_id,
      "maxFeePerGas": base_fee + 50_000_000_000_000, "maxPriorityFeePerGas": 27_600_000_000_000, "type": 2}
signed = acct.sign_transaction(tx)
resp = requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed.rawTransaction.hex()],"id":1}).json()
print(json.dumps(resp))
PYEOF
  )
  printf '[INFO]  %s → %s\n' "$addr" "$err" >&2

  if [[ "$err" == *"precompile"* ]] || [[ "$err" == *"Precompile"* ]]; then
    fail_count=$(( fail_count + 1 ))
  else
    # tx가 들어갔다면 receipt.status == 0
    tx_hash=$(printf '%s' "$err" | python3 -c "
import sys, json
try: print(json.loads(sys.stdin.read()).get('result', ''))
except: print('')
")
    if [[ -n "$tx_hash" && "$tx_hash" != "null" ]]; then
      status=$(wait_receipt "1" "$tx_hash" 20 2>/dev/null || echo "timeout")
      [[ "$status" == "failed" ]] && fail_count=$(( fail_count + 1 ))
    fi
  fi
done

assert_ge "$fail_count" "1" "at least 1 precompile transfer blocked (got $fail_count/${#precompiles[@]})"

test_result
