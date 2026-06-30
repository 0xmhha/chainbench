#!/usr/bin/env bash
# Command: network — Attach remote / ssh-remote nodes as a named network
# lib/cmd_network.sh - user surface over the network.attach wire command.
# Sourced by the main chainbench dispatcher; $@ contains subcommand arguments.

# Guard against double-sourcing
[[ -n "${_CB_CMD_NETWORK_SH_LOADED:-}" ]] && return 0
readonly _CB_CMD_NETWORK_SH_LOADED=1

_CB_NETWORK_CMD_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${_CB_NETWORK_CMD_LIB_DIR}/common.sh"
# shellcheck source=lib/network_client.sh
source "${_CB_NETWORK_CMD_LIB_DIR}/network_client.sh"

_cb_network_usage() {
  cat <<'EOF'
Usage: chainbench network attach <name> <rpc_url> [options]

Attach a remote or ssh-remote node as a named network. Probes the endpoint for
chain_id / chain_type and persists state/networks/<name>.json.

Options:
  --type <chain_type>            probe override: stablenet|wbft|wemix|ethereum
  --provider remote|ssh-remote   default: remote
  --json                         print the raw wire result

  remote auth (optional):
  --auth-type api-key|jwt        auth scheme
  --auth-env <VAR>               env var holding the credential
  --auth-header <H>              header name (api-key; default Authorization)

  ssh-remote (--provider ssh-remote):
  --ssh-user <U>  --ssh-host <H>  --ssh-port <N>  --ssh-env <VAR>
  --log-file <PATH>
  --start-cmd <C>  --stop-cmd <C>  --restart-cmd <C>

Credentials are referenced by env-var NAME only — never pass a secret inline.
EOF
}

# _cb_network_cmd_attach <name> <rpc_url> [flags]
_cb_network_cmd_attach() {
  local name="" rpc_url="" chain_type="" provider="" as_json=0
  local auth_type="" auth_env="" auth_header=""
  local ssh_user="" ssh_host="" ssh_port="" ssh_env=""
  local log_file="" start_cmd="" stop_cmd="" restart_cmd=""
  local positionals=()

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --type)          chain_type="${2:?--type requires a value}"; shift 2 ;;
      --type=*)        chain_type="${1#--type=}"; shift ;;
      --provider)      provider="${2:?--provider requires a value}"; shift 2 ;;
      --provider=*)    provider="${1#--provider=}"; shift ;;
      --json)          as_json=1; shift ;;
      --auth-type)     auth_type="${2:?--auth-type requires a value}"; shift 2 ;;
      --auth-type=*)   auth_type="${1#--auth-type=}"; shift ;;
      --auth-env)      auth_env="${2:?--auth-env requires a value}"; shift 2 ;;
      --auth-env=*)    auth_env="${1#--auth-env=}"; shift ;;
      --auth-header)   auth_header="${2:?--auth-header requires a value}"; shift 2 ;;
      --auth-header=*) auth_header="${1#--auth-header=}"; shift ;;
      --ssh-user)      ssh_user="${2:?--ssh-user requires a value}"; shift 2 ;;
      --ssh-user=*)    ssh_user="${1#--ssh-user=}"; shift ;;
      --ssh-host)      ssh_host="${2:?--ssh-host requires a value}"; shift 2 ;;
      --ssh-host=*)    ssh_host="${1#--ssh-host=}"; shift ;;
      --ssh-port)      ssh_port="${2:?--ssh-port requires a value}"; shift 2 ;;
      --ssh-port=*)    ssh_port="${1#--ssh-port=}"; shift ;;
      --ssh-env)       ssh_env="${2:?--ssh-env requires a value}"; shift 2 ;;
      --ssh-env=*)     ssh_env="${1#--ssh-env=}"; shift ;;
      --log-file)      log_file="${2:?--log-file requires a value}"; shift 2 ;;
      --log-file=*)    log_file="${1#--log-file=}"; shift ;;
      --start-cmd)     start_cmd="${2:?--start-cmd requires a value}"; shift 2 ;;
      --start-cmd=*)   start_cmd="${1#--start-cmd=}"; shift ;;
      --stop-cmd)      stop_cmd="${2:?--stop-cmd requires a value}"; shift 2 ;;
      --stop-cmd=*)    stop_cmd="${1#--stop-cmd=}"; shift ;;
      --restart-cmd)   restart_cmd="${2:?--restart-cmd requires a value}"; shift 2 ;;
      --restart-cmd=*) restart_cmd="${1#--restart-cmd=}"; shift ;;
      -h|--help)       _cb_network_usage; return 0 ;;
      -*)              log_warn "Unknown option: $1"; shift ;;
      *)               positionals+=("$1"); shift ;;
    esac
  done

  name="${positionals[0]:-}"
  rpc_url="${positionals[1]:-}"
  [[ -z "$provider" ]] && provider="remote"

  if [[ -z "$name" || -z "$rpc_url" ]]; then
    log_error "Usage: chainbench network attach <name> <rpc_url> [options]"
    return 1
  fi
  if [[ "$provider" == "ssh-remote" ]] && { [[ -z "$ssh_user" ]] || [[ -z "$ssh_host" ]] || [[ -z "$ssh_env" ]]; }; then
    log_error "ssh-remote requires --ssh-user, --ssh-host, and --ssh-env"
    return 1
  fi

  # Build the wire args object with jq (already required by cb_net_call). Only
  # set fields are included; credentials are passed as env-var NAMES, not values.
  local args_json
  if ! args_json=$(jq -cn \
      --arg name "$name" --arg url "$rpc_url" --arg type "$chain_type" --arg provider "$provider" \
      --arg atype "$auth_type" --arg aenv "$auth_env" --arg aheader "$auth_header" \
      --arg suser "$ssh_user" --arg shost "$ssh_host" --arg sport "$ssh_port" --arg senv "$ssh_env" \
      --arg logf "$log_file" --arg startc "$start_cmd" --arg stopc "$stop_cmd" --arg restartc "$restart_cmd" '
      {name: $name, rpc_url: $url}
      + (if $type     != "" then {override: $type} else {} end)
      + (if $provider != "" and $provider != "remote" then {provider: $provider} else {} end)
      + (if $provider == "ssh-remote" then
           {auth: ({type: "ssh-password", user: $suser, host: $shost, env: $senv}
                   + (if $sport != "" then {port: ($sport | tonumber)} else {} end))}
           + (((if $logf     != "" then {log_file:    $logf}     else {} end)
             + (if $startc   != "" then {start_cmd:   $startc}   else {} end)
             + (if $stopc    != "" then {stop_cmd:    $stopc}    else {} end)
             + (if $restartc != "" then {restart_cmd: $restartc} else {} end)) as $meta
             | if ($meta | length) > 0 then {provider_meta: $meta} else {} end)
         elif $atype != "" then
           {auth: ({type: $atype}
                   + (if $aenv    != "" then {env: $aenv}       else {} end)
                   + (if $aheader != "" then {header: $aheader} else {} end))}
         else {} end)
      '); then
    log_error "Failed to build attach arguments (check --ssh-port is numeric)"
    return 1
  fi

  local result rc=0
  result="$(cb_net_call "network.attach" "$args_json")" || rc=$?
  if [[ $rc -ne 0 ]]; then
    log_error "Attach failed"
    [[ -n "$result" ]] && printf '%s\n' "$result" >&2
    return "$rc"
  fi

  if [[ "$as_json" -eq 1 ]]; then
    printf '%s\n' "$result"
    return 0
  fi
  printf '%s\n' "$result" | jq -r \
    '"Attached \(.name) (\(.chain_type), chain_id \(.chain_id)) — created=\(.created)"' \
    2>/dev/null || printf '%s\n' "$result"
}

cmd_network_main() {
  local action="${1:-}"
  [[ $# -gt 0 ]] && shift
  case "$action" in
    attach)            _cb_network_cmd_attach "$@" ;;
    ""|-h|--help|help) _cb_network_usage ;;
    *)                 log_error "Unknown network action: ${action}"; _cb_network_usage; return 1 ;;
  esac
}

cmd_network_main "$@"
