#!/usr/bin/env bash
# Test: regression/e-blacklist-authorized/e-01-sender-blacklisted
# RT-E-01 — 블랙리스트 계정이 Sender인 tx 거부 (ErrBlacklistedAccount)
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/e-blacklist-authorized/e-01-sender-blacklisted"
check_env || { test_result; exit 1; }

unlock_all_validators

# TEST_ACC_E를 GovCouncil로 blacklist 등록
propose_sel=$(selector "proposeAddBlacklist(address)")
target_padded=$(pad_address "$TEST_ACC_E_ADDR" | sed 's/^0x//')
propose_data="${propose_sel}${target_padded}"

receipt=$(gov_full_flow "$GOV_COUNCIL" "$propose_data" "$VALIDATOR_1_ADDR" "$VALIDATOR_2_ADDR") || {
  _assert_fail "gov flow failed"; test_result; exit 1;
}
exec_status=$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', ''))")
assert_eq "$exec_status" "0x1" "blacklist proposal executed"

# isBlacklisted 확인
is_bl_sel=$(selector "isBlacklisted(address)")
is_bl=$(hex_to_dec "$(eth_call_raw 1 "$ACCOUNT_MANAGER" "${is_bl_sel}${target_padded}")")
assert_eq "$is_bl" "1" "TEST_ACC_E is blacklisted"

# TEST_ACC_E가 sender로 tx 발행 시도 → 거부
err_output=$(python3 <<'PYEOF' 2>&1 || true
import json, requests
from eth_account import Account
pk = "0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a"  # TEST_ACC_E
url = "http://127.0.0.1:8501"
acct = Account.from_key(pk)
nonce = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getTransactionCount","params":[acct.address, "pending"],"id":1}).json()["result"], 16)
chain_id = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}).json()["result"], 16)
base_fee = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest", False],"id":1}).json()["result"]["baseFeePerGas"], 16)
tx = {"nonce": nonce, "to": "0x70997970C51812dc3A010C7d01b50e0d17dc79C8", "value": 1, "gas": 21000, "chainId": chain_id,
      "maxFeePerGas": base_fee + 50_000_000_000_000, "maxPriorityFeePerGas": 27_600_000_000_000, "type": 2}
signed = acct.sign_transaction(tx)
resp = requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed.rawTransaction.hex()],"id":1}).json()
print(json.dumps(resp))
PYEOF
)

assert_contains "$err_output" "blacklist" "error contains 'blacklist'"

# 상태 저장: E-04에서 해제
echo "$TEST_ACC_E_ADDR" > /tmp/chainbench-regression/blacklisted_addr.txt

test_result
