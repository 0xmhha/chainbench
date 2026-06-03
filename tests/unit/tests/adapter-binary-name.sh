#!/usr/bin/env bash
# tests/unit/tests/adapter-binary-name.sh - Pin adapter_binary_name contract (G3)
#
# Each chain adapter must expose adapter_binary_name, the canonical short
# binary name for its chain type. It is a label/default source only — the stop
# path keys off the launched basename (see cmd-stop tests), not this name — but
# the adapter contract must carry it so future non-gstable chains stay
# symmetric. Each adapter is loaded in its own subshell because the adapter
# files guard against double-sourcing.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/assert.sh"

CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
export CHAINBENCH_DIR
export CHAINBENCH_QUIET=1
source "${CHAINBENCH_DIR}/lib/chain_adapter.sh"

# ---- Test 1: stablenet -> gstable -----------------------------------------
describe "adapter_binary_name: stablenet"
name="$( cb_adapter_load stablenet >/dev/null 2>&1; adapter_binary_name )"
assert_eq "${name}" "gstable" "stablenet adapter binary name"

# ---- Test 2: wbft -> gwbft ------------------------------------------------
describe "adapter_binary_name: wbft"
name="$( cb_adapter_load wbft >/dev/null 2>&1; adapter_binary_name )"
assert_eq "${name}" "gwbft" "wbft adapter binary name"

# ---- Test 3: wemix -> gwemix ----------------------------------------------
describe "adapter_binary_name: wemix"
name="$( cb_adapter_load wemix >/dev/null 2>&1; adapter_binary_name )"
assert_eq "${name}" "gwemix" "wemix adapter binary name"

unit_summary
