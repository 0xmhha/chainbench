#!/usr/bin/env bash
# lib/adapters/wemix.sh - Adapter stub for go-wemix (gwemix)
# TODO: Implement when gwemix support is needed.
# go-wemix uses etcd-based consensus, not WBFT/Istanbul.

[[ -n "${_CB_ADAPTER_WEMIX_LOADED:-}" ]] && return 0
readonly _CB_ADAPTER_WEMIX_LOADED=1

_cb_wemix_not_implemented() {
  echo "ERROR: gwemix adapter is not yet implemented. Use chain.type=stablenet for now." >&2
  return 1
}

adapter_generate_genesis()          { _cb_wemix_not_implemented; }
adapter_generate_toml()             { _cb_wemix_not_implemented; }
adapter_extra_start_flags()         { printf '--allow-insecure-unlock\n'; }
adapter_consensus_rpc_namespace()   { printf 'wemix\n'; }
