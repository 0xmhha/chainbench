#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-A-2-05a
# name: GasTipCap < MinTip tx 처리 (Anzeon 활성)
# category: regression/a-ethereum
# tags: [tx, anzeon]
# estimated_seconds: 35
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/a-ethereum/a2-05a-tipcap-underpriced
# RT-A-2-05a — GasTipCap < MinTip tx 처리 (Anzeon 활성)
#
# 명세 불일치 기록:
#   REGRESSION_TEST_CASES_v2.md는 "txpool 진입 거부, ErrUnderpriced" 를 기대하지만
#   이는 pre-Anzeon 기준이다. regression 프로필은 AnzeonBlock=0로 Anzeon 활성 상태이며,
#   이 환경에서는 regular account의 낮은 tipCap이 자동으로 MinPriorityFee로 보정된다
#   (c-anzeon/c-01-regular-account-gastip-forced 가 이 동작의 정식 검증).
#   따라서 이 테스트는 "자동 보정 경로"를 검증하도록 재정의한다.
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a2-05a-tipcap-underpriced"
check_env || { test_result; exit 1; }

# MinPriorityFee 조회 (anzeon이 강제하는 하한)
min_tip_hex=$(rpc "1" "eth_maxPriorityFeePerGas" "[]" | json_get - result)
min_tip=$(hex_to_dec "$min_tip_hex")
printf '[INFO]  chain MinPriorityFee = %s wei\n' "$min_tip" >&2
assert_gt "$min_tip" "0" "MinPriorityFee > 0 (Anzeon 활성 확인)"

# 매우 낮은 tipCap (1 wei) 으로 tx 전송 → 자동 보정 기대
tx_hash=$(python3 <<'PYEOF' 2>&1 || true
import json, requests
from eth_account import Account

pk = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
to = "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
url = "http://127.0.0.1:8501"

acct = Account.from_key(pk)
nonce = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getTransactionCount","params":[acct.address, "pending"],"id":1}).json()["result"], 16)
chain_id = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}).json()["result"], 16)
base_fee = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest", False],"id":1}).json()["result"]["baseFeePerGas"], 16)

tx = {
    "nonce": nonce, "to": to, "value": 1, "gas": 21000, "chainId": chain_id,
    "maxFeePerGas": base_fee + 100_000_000_000_000,  # 충분히 높은 budget
    "maxPriorityFeePerGas": 1,                        # 매우 낮은 tip → 보정 기대
    "type": 2,
}
signed = acct.sign_transaction(tx)
resp = requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed.raw_transaction.to_0x_hex()],"id":1}).json()
print(resp.get("result", "") if "result" in resp else f"ERROR:{resp}")
PYEOF
)

assert_contains "$tx_hash" "0x" "tx submitted (Anzeon accepts low tip with auto-correction)"

# receipt의 effectiveGasPrice가 MinPriorityFee 이상으로 보정됐는지 확인
receipt=$(wait_tx_receipt_full "1" "$tx_hash" 30)
assert_not_empty "$receipt" "receipt retrieved"

status=$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', ''))")
assert_eq "$status" "0x1" "receipt.status == 0x1 (tx 성공)"

eff_gp_hex=$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('effectiveGasPrice', ''))")
eff_gp=$(hex_to_dec "$eff_gp_hex")
printf '[INFO]  effectiveGasPrice = %s wei (min_tip = %s wei)\n' "$eff_gp" "$min_tip" >&2
assert_ge "$eff_gp" "$min_tip" "effectiveGasPrice >= MinPriorityFee (자동 보정 확인)"

test_result
