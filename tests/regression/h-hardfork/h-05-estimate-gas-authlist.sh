#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-4-2-01
# name: eth_estimateGas with authorizationList
# category: regression/h-hardfork
# tags: [hardfork, boho, estimateGas, authorizationList, eip7702]
# estimated_seconds: 15
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/h-hardfork/h-05-estimate-gas-authlist
# TC-4-2-01 — Verify eth_estimateGas with authorizationList
#   1) eth_estimateGas without authorizationList succeeds
#   2) eth_estimateGas with authorizationList succeeds (not error)
#   3) Both return valid gas values
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-05-estimate-gas-authlist"
check_env || { test_result; exit 1; }

# --- 1) Baseline: eth_estimateGas without authorizationList ---
printf '[INFO]  estimating gas for simple transfer (no authorizationList)\n' >&2
resp_no_auth=$(rpc "1" "eth_estimateGas" \
  "[{\"from\":\"${TEST_ACC_A_ADDR}\",\"to\":\"${TEST_ACC_B_ADDR}\",\"value\":\"0x1\"}]")
estimate_no_auth=$(json_get "$resp_no_auth" "result")
error_no_auth=$(json_get "$resp_no_auth" "error.message")

assert_not_empty "$estimate_no_auth" "estimateGas without authorizationList returned a value"
if [[ -n "$error_no_auth" && "$error_no_auth" != "" ]]; then
  _assert_fail "estimateGas without authorizationList returned error: $error_no_auth"
else
  _assert_pass "estimateGas without authorizationList did not error"
fi

estimate_no_auth_dec=$(hex_to_dec "$estimate_no_auth")
observe "estimate_without_authlist" "$estimate_no_auth_dec"
printf '[INFO]  estimate (no authList) = %s gas\n' "$estimate_no_auth_dec" >&2
assert_ge "$estimate_no_auth_dec" "21000" "estimate without authList >= 21000"

# --- 2) eth_estimateGas with authorizationList ---
# Build a minimal authorizationList entry (EIP-7702 format)
# chainId, address (delegate target), nonce, v, r, s
printf '[INFO]  estimating gas with authorizationList\n' >&2

chain_id_hex=$(rpc "1" "eth_chainId" "[]" | json_get - result)
node_port=$(pids_get_field "1" "http_port")

# Use python to construct the RPC call with authorizationList and parse response
result_json=$(python3 <<PYEOF
import json, requests

url = "http://127.0.0.1:${node_port}"

# Build the request with authorizationList
tx_obj = {
    "from": "${TEST_ACC_A_ADDR}",
    "to": "${TEST_ACC_B_ADDR}",
    "value": "0x1",
    "authorizationList": [
        {
            "chainId": "${chain_id_hex}",
            "address": "${TEST_ACC_B_ADDR}",
            "nonce": "0x0",
            "v": "0x0",
            "r": "0x0000000000000000000000000000000000000000000000000000000000000000",
            "s": "0x0000000000000000000000000000000000000000000000000000000000000000"
        }
    ]
}

resp = requests.post(url, json={
    "jsonrpc": "2.0",
    "method": "eth_estimateGas",
    "params": [tx_obj],
    "id": 1
}, timeout=5).json()

print(json.dumps(resp))
PYEOF
)

estimate_with_auth=$(printf '%s' "$result_json" | json_get - "result")
error_with_auth=$(printf '%s' "$result_json" | json_get - "error.message")

observe "estimate_with_authlist_raw" "$estimate_with_auth"
observe "estimate_with_authlist_error" "$error_with_auth"

# The call with authorizationList should succeed (not error)
if [[ -n "$error_with_auth" && "$error_with_auth" != "" ]]; then
  _assert_fail "estimateGas with authorizationList returned error: $error_with_auth"
else
  _assert_pass "estimateGas with authorizationList did not error"
fi

assert_not_empty "$estimate_with_auth" "estimateGas with authorizationList returned a value"

estimate_with_auth_dec=$(hex_to_dec "$estimate_with_auth")
observe "estimate_with_authlist" "$estimate_with_auth_dec"
printf '[INFO]  estimate (with authList) = %s gas\n' "$estimate_with_auth_dec" >&2

# --- 3) Both should return valid gas values ---
assert_ge "$estimate_with_auth_dec" "21000" "estimate with authList >= 21000"

test_result
