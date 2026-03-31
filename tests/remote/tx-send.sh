#!/usr/bin/env bash
# Test: remote/tx-send
# Description: Send a test transaction on the remote chain (requires env vars)
set -euo pipefail

source "$(dirname "$0")/../lib/rpc.sh"
source "$(dirname "$0")/../lib/assert.sh"

test_start "remote/tx-send"

target="$(_cb_rpc_default_target)"
if ! _cb_rpc_is_remote "$target"; then
  _assert_fail "CHAINBENCH_REMOTE is not set — cannot run remote tests"
  test_result
  exit 1
fi

printf '[INFO]  Testing remote target: %s\n' "$target" >&2

# Required env vars for transaction sending
# CHAINBENCH_TEST_FROM: sender address (must be unlocked on the remote node)
# CHAINBENCH_TEST_TO: recipient address
# CHAINBENCH_TEST_PASSWORD: password to unlock the sender (optional)
from_addr="${CHAINBENCH_TEST_FROM:-}"
to_addr="${CHAINBENCH_TEST_TO:-}"
password="${CHAINBENCH_TEST_PASSWORD:-}"

if [[ -z "$from_addr" || -z "$to_addr" ]]; then
  printf '[INFO]  Skipping tx-send: set CHAINBENCH_TEST_FROM and CHAINBENCH_TEST_TO to enable\n' >&2
  printf '[INFO]  Example:\n' >&2
  printf '[INFO]    export CHAINBENCH_TEST_FROM=0x...\n' >&2
  printf '[INFO]    export CHAINBENCH_TEST_TO=0x...\n' >&2
  printf '[INFO]    export CHAINBENCH_TEST_PASSWORD=mypassword\n' >&2
  _assert_pass "tx-send skipped (no CHAINBENCH_TEST_FROM/TO set)"
  test_result
  exit 0
fi

printf '[INFO]  From: %s\n' "$from_addr" >&2
printf '[INFO]  To:   %s\n' "$to_addr" >&2

# Unlock account if password is provided
if [[ -n "$password" ]]; then
  printf '[INFO]  Unlocking sender account...\n' >&2
  unlock_account "$target" "$from_addr" "$password" 60
fi

# Get initial nonce
nonce_response=$(rpc "$target" "eth_getTransactionCount" "[\"${from_addr}\",\"latest\"]" 2>/dev/null || echo "")
if [[ -n "$nonce_response" ]]; then
  nonce=$(python3 -c "
import json, sys
try:
    d = json.loads(sys.argv[1])
    print(int(d['result'], 16))
except:
    print('?')
" "$nonce_response")
  printf '[INFO]  Current nonce: %s\n' "$nonce" >&2
fi

# Send transaction (1 wei)
tx_hash=$(send_tx "$target" "$from_addr" "$to_addr" "0x1")
assert_not_empty "$tx_hash" "transaction hash returned"

if [[ "$tx_hash" == TX_ERROR:* ]]; then
  error_msg="${tx_hash#TX_ERROR:}"
  _assert_fail "send_tx failed: $error_msg"
  printf '[INFO]  This is expected if the account is not unlocked or has no funds\n' >&2
  test_result
  exit 1
fi

assert_contains "$tx_hash" "0x" "tx hash is a hex string"
printf '[INFO]  Tx hash: %s\n' "$tx_hash" >&2

# Wait for receipt (timeout 60s for remote chains)
receipt_status=$(wait_receipt "$target" "$tx_hash" 60)
assert_eq "$receipt_status" "success" "transaction receipt status is success"

test_result
