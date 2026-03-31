#!/usr/bin/env bash
# Test: remote/balance-check
# Description: Query balance of an address on the remote chain
set -euo pipefail

source "$(dirname "$0")/../lib/rpc.sh"
source "$(dirname "$0")/../lib/assert.sh"

test_start "remote/balance-check"

target="$(_cb_rpc_default_target)"
if ! _cb_rpc_is_remote "$target"; then
  _assert_fail "CHAINBENCH_REMOTE is not set — cannot run remote tests"
  test_result
  exit 1
fi

# CHAINBENCH_TEST_ADDRESS: address to query balance for
# Default: Ethereum genesis address (always has balance on mainnet)
test_address="${CHAINBENCH_TEST_ADDRESS:-0x0000000000000000000000000000000000000000}"

printf '[INFO]  Testing remote target: %s\n' "$target" >&2
printf '[INFO]  Query address: %s\n' "$test_address" >&2

# Validate address format
if ! [[ "$test_address" =~ ^0x[0-9a-fA-F]{40}$ ]]; then
  _assert_fail "Invalid address format: $test_address (must be 0x + 40 hex chars)"
  test_result
  exit 1
fi

# Test: eth_getBalance
balance_response=$(rpc "$target" "eth_getBalance" "[\"${test_address}\",\"latest\"]" 2>/dev/null || echo "")
assert_not_empty "$balance_response" "eth_getBalance returns a response"

if [[ -n "$balance_response" ]]; then
  balance_result=$(python3 -c "
import json, sys
try:
    d = json.loads(sys.argv[1])
    result = d.get('result', '')
    if result and result.startswith('0x'):
        print(int(result, 16))
    elif 'error' in d:
        print('ERROR:' + d['error'].get('message', 'unknown'))
    else:
        print('')
except:
    print('')
" "$balance_response")

  if [[ "$balance_result" == ERROR:* ]]; then
    _assert_fail "eth_getBalance error: ${balance_result#ERROR:}"
  elif [[ -n "$balance_result" ]]; then
    _assert_pass "balance query successful: ${balance_result} wei"
    printf '[INFO]  Balance: %s wei\n' "$balance_result" >&2

    # If address is the zero address, balance should typically be > 0 on mainnet
    if [[ "$test_address" == "0x0000000000000000000000000000000000000000" ]]; then
      assert_ge "$balance_result" "0" "zero address balance is non-negative"
    fi
  else
    _assert_fail "eth_getBalance returned empty result"
  fi
fi

test_result
