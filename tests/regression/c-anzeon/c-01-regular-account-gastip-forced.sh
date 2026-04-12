#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-C-01
# name: 일반 계정의 tipCap이 header.GasTip()으로 강제 대체됨
# category: regression/c-anzeon
# tags: [anzeon, gas]
# estimated_seconds: 35
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/c-anzeon/c-01-regular-account-gastip-forced
# RT-C-01 — 일반 계정의 tipCap이 header.GasTip()으로 강제 대체됨
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/c-anzeon/c-01-regular-account-gastip-forced"
check_env || { test_result; exit 1; }

# 현재 header.GasTip
header_tip=$(get_header_gas_tip "1")
printf '[INFO]  header.GasTip = %s wei\n' "$header_tip" >&2

# baseFee
base_fee=$(get_base_fee "1")

# TEST_ACC_A (비-Authorized 계정)로 tipCap = header_tip * 5 (매우 큰 값) 전송
# 실제 effectiveGasPrice는 header_tip이 강제되어야 함
high_tip=$(( header_tip * 5 ))

tx_hash=$(python3 <<PYEOF
import json, requests
from eth_account import Account
pk = "${TEST_ACC_A_PK}"
url = "http://127.0.0.1:8501"
acct = Account.from_key(pk)
nonce = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getTransactionCount","params":[acct.address, "pending"],"id":1}).json()["result"], 16)
chain_id = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}).json()["result"], 16)
base_fee = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest", False],"id":1}).json()["result"]["baseFeePerGas"], 16)
tx = {"nonce": nonce, "to": "0x70997970C51812dc3A010C7d01b50e0d17dc79C8", "value": 1, "gas": 21000, "chainId": chain_id,
      "maxFeePerGas": base_fee * 2 + ${high_tip},
      "maxPriorityFeePerGas": ${high_tip},  # 매우 큰 tipCap (header보다 훨씬 큼)
      "type": 2}
signed = acct.sign_transaction(tx)
resp = requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed.raw_transaction.to_0x_hex()],"id":1}).json()
print(resp.get("result", ""))
PYEOF
)

assert_contains "$tx_hash" "0x" "tx submitted with high tipCap"

receipt=$(wait_tx_receipt_full "1" "$tx_hash" 30)
effective_gp=$(printf '%s' "$receipt" | python3 -c "import sys, json; print(int(json.load(sys.stdin).get('effectiveGasPrice', '0x0'), 16))")

# Regular 계정이면 tip이 header.GasTip으로 강제되었으므로
# effectiveGasPrice = baseFee + header_tip (high_tip이 아님)
expected=$(( base_fee + header_tip ))
printf '[INFO]  effectiveGasPrice=%s, expected=%s (baseFee+header_tip)\n' "$effective_gp" "$expected" >&2

# 오차 허용 (baseFee가 블록 사이에 변동 가능)
# tipCap만 비교: effectiveGasPrice - baseFee == header_tip
tip_used=$(( effective_gp - base_fee ))
# 작은 오차 허용: 90% ~ 110%
tip_lower=$(( header_tip * 90 / 100 ))
tip_upper=$(( header_tip * 110 / 100 ))
in_range=$( [[ $tip_used -ge $tip_lower && $tip_used -le $tip_upper ]] && echo true || echo false )
assert_true "$in_range" "effective tip ($tip_used) ≈ header.GasTip ($header_tip), NOT the high_tip ($high_tip)"

test_result
