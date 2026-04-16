#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/assert.sh"
export CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
source "${CHAINBENCH_DIR}/tests/lib/contract.sh"

describe "contract: cast is available"
assert_exit_code 0 command -v cast

describe "contract: cb_selector computes correct selector"
sel=$(cb_selector "transfer(address,uint256)")
assert_eq "$sel" "0xa9059cbb" "transfer selector"

describe "contract: cb_abi_encode produces valid calldata"
encoded=$(cb_abi_encode "transfer(address,uint256)" \
  "0x0000000000000000000000000000000000001000" "100")
assert_contains "$encoded" "a9059cbb" "starts with transfer selector"
# 0x + 4 bytes selector + 32 bytes addr + 32 bytes uint = 138 chars
assert_eq "${#encoded}" "138" "calldata length"

describe "contract: _cb_resolve_rpc_url handles http:// passthrough"
url=$(_cb_resolve_rpc_url "http://localhost:8545")
assert_eq "$url" "http://localhost:8545" "http URL passthrough"

describe "contract: _cb_resolve_rpc_url handles https:// passthrough"
url=$(_cb_resolve_rpc_url "https://rpc.example.com")
assert_eq "$url" "https://rpc.example.com" "https URL passthrough"

unit_summary
