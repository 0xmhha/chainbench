#!/usr/bin/env bash
# lib/adapters/wbft.sh - Adapter stub for go-wbft (gwbft)
# TODO: Implement when gwbft support is needed.

[[ -n "${_CB_ADAPTER_WBFT_LOADED:-}" ]] && return 0
readonly _CB_ADAPTER_WBFT_LOADED=1

_cb_wbft_not_implemented() {
  echo "ERROR: gwbft adapter is not yet implemented. Use chain.type=stablenet for now." >&2
  return 1
}

adapter_generate_genesis()          { _cb_wbft_not_implemented; }
adapter_generate_toml()             { _cb_wbft_not_implemented; }
adapter_extra_start_flags()         { printf '--allow-insecure-unlock\n'; }
adapter_consensus_rpc_namespace()   { printf 'istanbul\n'; }
