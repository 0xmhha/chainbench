#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-D-01
# name: FeeDelegateDynamicFeeTx (type 0x16) 정상 처리
# category: regression/d-fee-delegation
# tags: [fee-delegation, gas, delegation]
# estimated_seconds: 35
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/d-fee-delegation/d-01-fee-delegate-normal
# RT-D-01 — FeeDelegateDynamicFeeTx (type 0x16) 정상 처리
#   Sender(TEST_ACC_A)가 value만 지불, FeePayer(TEST_ACC_C)가 가스비 지불
#
# 사용 도구: tests/regression/lib/fee_delegate.py
#   (go-stablenet tx_fee_delegation.go 기반 RLP 인코더)
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/d-fee-delegation/d-01-fee-delegate-normal"
check_env || { test_result; exit 1; }

# rlp 패키지 설치 확인
python3 -c "import rlp, eth_keys" 2>/dev/null || {
  printf '[ERROR] Required Python packages missing. Install: pip3 install rlp eth-keys eth-account requests\n' >&2
  test_result
  exit 1
}

HELPER="${CHAINBENCH_DIR}/tests/regression/lib/fee_delegate.py"
value_wei=1000000000000000  # 0.001 ether

# 사전 잔액
sender_before=$(hex_to_dec "$(rpc 1 eth_getBalance "[\"${TEST_ACC_A_ADDR}\", \"latest\"]" | json_get - result)")
payer_before=$(hex_to_dec "$(rpc 1 eth_getBalance "[\"${TEST_ACC_C_ADDR}\", \"latest\"]" | json_get - result)")
recipient_before=$(hex_to_dec "$(rpc 1 eth_getBalance "[\"${TEST_ACC_B_ADDR}\", \"latest\"]" | json_get - result)")

# Sign & send
result=$(python3 "$HELPER" send \
  --rpc http://127.0.0.1:8501 \
  --sender-pk "$TEST_ACC_A_PK" \
  --fee-payer-pk "$TEST_ACC_C_PK" \
  --to "$TEST_ACC_B_ADDR" \
  --value "$value_wei" \
  --gas 21000 2>&1)

tx_hash=$(printf '%s' "$result" | python3 -c "
import sys, json
try:
    d = json.loads(sys.stdin.read())
    print(d.get('txHash', ''))
except Exception as e:
    print('')
")

if [[ -z "$tx_hash" || "$tx_hash" == "null" ]]; then
  printf '[ERROR] result: %s\n' "$result" >&2
  _assert_fail "FeeDelegate tx send failed"
  test_result
  exit 1
fi

assert_contains "$tx_hash" "0x" "FeeDelegateDynamicFeeTx hash returned"

receipt=$(wait_tx_receipt_full "1" "$tx_hash" 30)
status=$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', ''))")
assert_eq "$status" "0x1" "receipt.status == 0x1"

# tx type == 0x16
tx_type=$(rpc 1 eth_getTransactionByHash "[\"${tx_hash}\"]" | json_get - "result.type")
assert_eq "$tx_type" "0x16" "tx type is 0x16 (FeeDelegateDynamicFeeTx)"

# Post 잔액
sender_after=$(hex_to_dec "$(rpc 1 eth_getBalance "[\"${TEST_ACC_A_ADDR}\", \"latest\"]" | json_get - result)")
payer_after=$(hex_to_dec "$(rpc 1 eth_getBalance "[\"${TEST_ACC_C_ADDR}\", \"latest\"]" | json_get - result)")
recipient_after=$(hex_to_dec "$(rpc 1 eth_getBalance "[\"${TEST_ACC_B_ADDR}\", \"latest\"]" | json_get - result)")

# Sender: value만 차감 (가스비 없음)
sender_diff=$(( sender_before - sender_after ))
assert_eq "$sender_diff" "$value_wei" "Sender lost exactly $value_wei (no gas charge)"

# Recipient: value 증가
recipient_diff=$(( recipient_after - recipient_before ))
assert_eq "$recipient_diff" "$value_wei" "Recipient gained $value_wei"

# FeePayer: 가스비만 차감 (value 차감 없음)
payer_diff=$(( payer_before - payer_after ))
# gasUsed × effectiveGasPrice
gas_used=$(hex_to_dec "$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('gasUsed', ''))")")
eff_gp=$(hex_to_dec "$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('effectiveGasPrice', ''))")")
expected_gas_cost=$(( gas_used * eff_gp ))
assert_eq "$payer_diff" "$expected_gas_cost" "FeePayer charged exactly gasUsed × effectiveGasPrice ($expected_gas_cost)"

test_result
