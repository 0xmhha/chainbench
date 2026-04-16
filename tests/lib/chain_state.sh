#!/usr/bin/env bash
# tests/lib/chain_state.sh — Chain state query utilities (cast-based)
# Provides: balance, nonce, block, chain-id, account-extra, hardfork checks, block polling.

[[ -n "${_CB_CHAIN_STATE_LOADED:-}" ]] && return 0
readonly _CB_CHAIN_STATE_LOADED=1

_CB_CHAIN_STATE_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${_CB_CHAIN_STATE_LIB_DIR}/contract.sh"
# system_contracts.sh sourced transitively by contract.sh

# ---------------------------------------------------------------------------
# Balance & Nonce
# ---------------------------------------------------------------------------

# cb_get_balance <target> <address> — balance in wei (decimal)
cb_get_balance() {
  local target="${1:?cb_get_balance: target required}"
  local address="${2:?cb_get_balance: address required}"
  local url; url="$(_cb_resolve_rpc_url "$target")" || return 1
  "$_CB_CAST_BIN" balance --rpc-url "$url" "$address" 2>/dev/null \
    || { echo "ERROR: cb_get_balance failed (target=$target, address=$address)" >&2; return 1; }
}

# cb_get_nonce <target> <address> — nonce (decimal)
cb_get_nonce() {
  local target="${1:?cb_get_nonce: target required}"
  local address="${2:?cb_get_nonce: address required}"
  local url; url="$(_cb_resolve_rpc_url "$target")" || return 1
  "$_CB_CAST_BIN" nonce --rpc-url "$url" "$address" 2>/dev/null \
    || { echo "ERROR: cb_get_nonce failed (target=$target, address=$address)" >&2; return 1; }
}

# ---------------------------------------------------------------------------
# Block queries
# ---------------------------------------------------------------------------

# cb_get_block <target> [block_number] — full block JSON (default: latest)
cb_get_block() {
  local target="${1:?cb_get_block: target required}"
  local block="${2:-latest}"
  local url; url="$(_cb_resolve_rpc_url "$target")" || return 1
  "$_CB_CAST_BIN" block --rpc-url "$url" "$block" --json 2>/dev/null \
    || { echo "ERROR: cb_get_block failed (target=$target, block=$block)" >&2; return 1; }
}

# cb_get_block_field <target> <field> [block_number] — single block field (default: latest)
cb_get_block_field() {
  local target="${1:?cb_get_block_field: target required}"
  local field="${2:?cb_get_block_field: field required}"
  local block="${3:-latest}"
  local url; url="$(_cb_resolve_rpc_url "$target")" || return 1
  "$_CB_CAST_BIN" block --rpc-url "$url" "$block" "$field" 2>/dev/null \
    || { echo "ERROR: cb_get_block_field failed (target=$target, field=$field)" >&2; return 1; }
}

# cb_get_base_fee <target> — current baseFeePerGas in wei (decimal)
cb_get_base_fee() {
  cb_get_block_field "${1:?cb_get_base_fee: target required}" "baseFeePerGas" "latest"
}

# ---------------------------------------------------------------------------
# Chain ID
# ---------------------------------------------------------------------------

# cb_get_chain_id <target> — chain ID (decimal)
cb_get_chain_id() {
  local target="${1:?cb_get_chain_id: target required}"
  local url; url="$(_cb_resolve_rpc_url "$target")" || return 1
  "$_CB_CAST_BIN" chain-id --rpc-url "$url" 2>/dev/null \
    || { echo "ERROR: cb_get_chain_id failed (target=$target)" >&2; return 1; }
}

# ---------------------------------------------------------------------------
# Account Extra — via AccountManager precompile
# Bit 63 = Blacklisted (0x8000000000000000)
# Bit 62 = Authorized  (0x4000000000000000)
# ---------------------------------------------------------------------------

# _cb_bool_from_raw <raw_hex_or_string> — normalise cast bool output → "true"/"false"
_cb_bool_from_raw() {
  case "$1" in
    true|0x0000000000000000000000000000000000000000000000000000000000000001)
      printf 'true' ;;
    *) printf 'false' ;;
  esac
}

# cb_is_blacklisted <target> <address> → "true" or "false"
cb_is_blacklisted() {
  local target="${1:?cb_is_blacklisted: target required}"
  local address="${2:?cb_is_blacklisted: address required}"
  local raw
  raw=$(cb_eth_call "$target" "$SC_ACCOUNT_MANAGER" "isBlacklisted(address)" "$address") \
    || { echo "ERROR: cb_is_blacklisted failed (target=$target)" >&2; return 1; }
  _cb_bool_from_raw "$raw"
}

# cb_is_authorized <target> <address> → "true" or "false"
cb_is_authorized() {
  local target="${1:?cb_is_authorized: target required}"
  local address="${2:?cb_is_authorized: address required}"
  local raw
  raw=$(cb_eth_call "$target" "$SC_ACCOUNT_MANAGER" "isAuthorized(address)" "$address") \
    || { echo "ERROR: cb_is_authorized failed (target=$target)" >&2; return 1; }
  _cb_bool_from_raw "$raw"
}

# cb_get_account_extra <target> <address>
# Returns the reconstructed extra uint64 (decimal) derived from precompile flags.
# No direct eth_getAccountExtra RPC is available in go-stablenet; flags are derived
# from AccountManager precompile calls.
cb_get_account_extra() {
  local target="${1:?cb_get_account_extra: target required}"
  local address="${2:?cb_get_account_extra: address required}"
  local blacklisted authorized extra=0
  blacklisted=$(cb_is_blacklisted "$target" "$address") || return 1
  authorized=$(cb_is_authorized   "$target" "$address") || return 1
  [[ "$blacklisted" == "true" ]] && extra=$(( extra | (1 << 63) ))
  [[ "$authorized"  == "true" ]] && extra=$(( extra | (1 << 62) ))
  printf '%d' "$extra"
}

# ---------------------------------------------------------------------------
# Block-level polling
# ---------------------------------------------------------------------------

# cb_wait_for_block <target> <block_number> [timeout_secs]
# Polls until chain reaches block_number. Prints reached block on success, "timeout" on failure.
# Returns 0 on success, 1 on timeout. Default timeout: 120s.
cb_wait_for_block() {
  local target="${1:?cb_wait_for_block: target required}"
  local target_block="${2:?cb_wait_for_block: block_number required}"
  local timeout="${3:-120}"
  local url elapsed=0 current
  url="$(_cb_resolve_rpc_url "$target")" || return 1
  while [[ "$elapsed" -lt "$timeout" ]]; do
    current=$("$_CB_CAST_BIN" block-number --rpc-url "$url" 2>/dev/null || echo "0")
    if [[ "$current" -ge "$target_block" ]]; then
      printf '%d' "$current"
      return 0
    fi
    sleep 1
    elapsed=$(( elapsed + 1 ))
  done
  printf 'timeout'
  return 1
}

# ---------------------------------------------------------------------------
# Hardfork activation checks
# ---------------------------------------------------------------------------

# cb_check_hardfork_active <target> <fork_name>
# Returns: "active", "inactive", or "unknown"
# Supported fork names (case-insensitive): boho, cancun, shanghai, london,
#   berlin, istanbul, muirglacier, homestead, frontier
cb_check_hardfork_active() {
  local target="${1:?cb_check_hardfork_active: target required}"
  local fork_name="${2:?cb_check_hardfork_active: fork_name required}"
  local fork_lower url
  fork_lower=$(printf '%s' "$fork_name" | tr '[:upper:]' '[:lower:]')
  url="$(_cb_resolve_rpc_url "$target")" || return 1

  case "$fork_lower" in
    boho)
      # secp256r1 precompile (0x100) is only live post-Boho
      local resp
      resp=$("$_CB_CAST_BIN" call --rpc-url "$url" \
        "0x0000000000000000000000000000000000000100" "0x" 2>/dev/null) || resp=""
      [[ -n "$resp" && "$resp" != ERROR* ]] && printf 'active' || printf 'inactive'
      ;;
    cancun)
      # blobGasUsed field present in block JSON from Cancun onward
      local bj
      bj=$(cb_get_block "$target" "latest") || { printf 'unknown'; return 0; }
      printf '%s' "$bj" | python3 -c "
import json,sys; b=json.loads(sys.stdin.read())
print('active' if 'blobGasUsed' in b else 'inactive')" 2>/dev/null || printf 'unknown'
      ;;
    shanghai)
      # withdrawals field present from Shanghai onward
      local bj
      bj=$(cb_get_block "$target" "latest") || { printf 'unknown'; return 0; }
      printf '%s' "$bj" | python3 -c "
import json,sys; b=json.loads(sys.stdin.read())
print('active' if 'withdrawals' in b else 'inactive')" 2>/dev/null || printf 'unknown'
      ;;
    london)
      # baseFeePerGas present from London onward
      local fee
      fee=$("$_CB_CAST_BIN" block --rpc-url "$url" "latest" "baseFeePerGas" 2>/dev/null || echo "")
      [[ -n "$fee" && "$fee" != "null" ]] && printf 'active' || printf 'inactive'
      ;;
    berlin|istanbul|muirglacier|homestead|frontier)
      # All modern go-stablenet chains run at least Berlin
      printf 'active'
      ;;
    *)
      printf 'unknown'
      ;;
  esac
}
