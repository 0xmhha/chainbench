#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-1-1-11
# name: GovMinter v2 bytecode verification after Boho
# category: regression/h-hardfork
# tags: [hardfork, boho, govminter, system-contract]
# estimated_seconds: 15
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/h-hardfork/h-03-govminter-v2-code
# TC-1-1-11 — Verify GovMinter contract has v2 bytecode after Boho hardfork
#   eth_getCode at GOV_MINTER address should return non-empty code
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-03-govminter-v2-code"
check_env || { test_result; exit 1; }

# --- 1) Fetch bytecode at GOV_MINTER system contract address ---
printf '[INFO]  checking bytecode at GOV_MINTER = %s\n' "$GOV_MINTER" >&2

resp=$(rpc "1" "eth_getCode" "[\"${GOV_MINTER}\", \"latest\"]")
code=$(json_get "$resp" "result")
error=$(json_get "$resp" "error.message")

# Check no RPC error
if [[ -n "$error" && "$error" != "" ]]; then
  _assert_fail "eth_getCode returned error: $error"
fi

# --- 2) Verify code is non-empty (v2 deployed) ---
assert_not_empty "$code" "GovMinter bytecode is not empty"

# Code should be more than just "0x" (empty contract)
code_len=${#code}
observe "govminter_code_length_hex_chars" "$code_len"
printf '[INFO]  GovMinter code length = %d hex chars\n' "$code_len" >&2

assert_gt "$code_len" "2" "GovMinter code length > 2 (not just '0x')"

# --- 3) Verify code starts with 0x (valid hex) ---
assert_contains "$code" "0x" "GovMinter code is hex-encoded"

# --- 4) Verify code has substantial size (v2 should be non-trivial) ---
# Typical compiled Solidity contract is at least a few hundred bytes
# 0x prefix + at least 100 hex chars = 50 bytes minimum
assert_gt "$code_len" "100" "GovMinter code is substantial (v2 bytecode deployed)"

observe "govminter_address" "$GOV_MINTER"
observe "govminter_has_code" "true"

test_result
