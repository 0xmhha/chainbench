#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/assert.sh"
REAL_CB_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
source "${REAL_CB_DIR}/tests/lib/system_contracts.sh"

describe "system_contracts: addresses are 42-char hex"
assert_eq "${#SC_NATIVE_COIN_ADAPTER}" "42" "NativeCoinAdapter address length"
assert_eq "${#SC_GOV_VALIDATOR}" "42" "GovValidator address length"
assert_eq "${#SC_ACCOUNT_MANAGER}" "42" "AccountManager address length"

describe "system_contracts: selectors are 10-char hex (0x + 8)"
assert_eq "${#SEL_TRANSFER}" "10" "transfer selector length"
assert_eq "${#SEL_BALANCE_OF}" "10" "balanceOf selector length"

describe "system_contracts: event topics are 66-char hex (0x + 64)"
assert_eq "${#EVENT_TRANSFER}" "66" "Transfer event topic length"

unit_summary
