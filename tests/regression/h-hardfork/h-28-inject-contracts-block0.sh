#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-4-4-03
# name: GovMinter v2 bytecode injected at genesis (BohoBlock=0)
# category: regression/h-hardfork
# tags: [hardfork, boho, genesis, injection]
# estimated_seconds: 15
# preconditions:
#   chain_running: true
# depends_on: []
# ---end-meta---
# TC-4-4-03 — With BohoBlock=0, verify v2 bytecode exists at block 0
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-28-inject-contracts-block0"
check_env || { test_result; exit 1; }

# Fetch GovMinter bytecode at genesis block (block 0)
resp=$(rpc 1 "eth_getCode" "[\"${GOV_MINTER}\",\"0x0\"]")
code=$(json_get "$resp" "result")
code_len=${#code}

printf '[INFO]  GovMinter code at block 0: %d hex chars\n' "$code_len" >&2
observe "govminter_code_len_block0" "$code_len"

assert_not_empty "$code" "GovMinter has bytecode at block 0"
assert_gt "$code_len" "100" "GovMinter bytecode is substantial at genesis"

# Also verify at latest block (should be same code)
resp=$(rpc 1 "eth_getCode" "[\"${GOV_MINTER}\",\"latest\"]")
code_latest=$(json_get "$resp" "result")
assert_eq "$code" "$code_latest" "GovMinter code unchanged since genesis"

test_result
