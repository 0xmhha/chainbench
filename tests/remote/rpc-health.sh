#!/usr/bin/env bash
# Test: remote/rpc-health
# Description: Verify remote RPC endpoint is responding to basic queries
set -euo pipefail

source "$(dirname "$0")/../lib/rpc.sh"
source "$(dirname "$0")/../lib/assert.sh"

test_start "remote/rpc-health"

# Determine target
target="$(_cb_rpc_default_target)"
if ! _cb_rpc_is_remote "$target"; then
  _assert_fail "CHAINBENCH_REMOTE is not set — cannot run remote tests"
  test_result
  exit 1
fi

printf '[INFO]  Testing remote target: %s\n' "$target" >&2

# Test 1: eth_blockNumber
raw_response=$(rpc "$target" "eth_blockNumber" "[]" 2>/dev/null || echo "")
assert_not_empty "$raw_response" "eth_blockNumber returns a response"

if [[ -n "$raw_response" ]]; then
  block_hex=$(python3 -c "
import json, sys
try:
    d = json.loads(sys.argv[1])
    result = d.get('result', '')
    if result and result.startswith('0x'):
        print(result)
    else:
        print('')
except Exception:
    print('')
" "$raw_response")

  assert_not_empty "$block_hex" "eth_blockNumber returns a valid hex result"

  if [[ -n "$block_hex" ]]; then
    block_dec=$(python3 -c "print(int('$block_hex', 16))" 2>/dev/null || echo "-1")
    assert_ge "$block_dec" "0" "block number is non-negative ($block_dec)"
    printf '[INFO]  Remote chain at block %s\n' "$block_dec" >&2
  fi
fi

# Test 2: net_peerCount (may be unsupported on some public RPCs)
peers_response=$(rpc "$target" "net_peerCount" "[]" 2>/dev/null || echo "")
if [[ -n "$peers_response" ]]; then
  peers_result=$(python3 -c "
import json, sys
try:
    d = json.loads(sys.argv[1])
    if 'result' in d and d['result']:
        print(d['result'])
    else:
        print('unsupported')
except:
    print('unsupported')
" "$peers_response")

  if [[ "$peers_result" != "unsupported" ]]; then
    _assert_pass "net_peerCount is supported"
  else
    printf '[INFO]  net_peerCount not supported on this RPC (skipping)\n' >&2
  fi
fi

# Test 3: web3_clientVersion
client_response=$(rpc "$target" "web3_clientVersion" "[]" 2>/dev/null || echo "")
if [[ -n "$client_response" ]]; then
  client_result=$(python3 -c "
import json, sys
try:
    d = json.loads(sys.argv[1])
    if 'result' in d and d['result']:
        print(d['result'])
    else:
        print('')
except:
    print('')
" "$client_response")

  if [[ -n "$client_result" ]]; then
    _assert_pass "web3_clientVersion returns: ${client_result:0:60}"
  else
    printf '[INFO]  web3_clientVersion not available (skipping)\n' >&2
  fi
fi

test_result
