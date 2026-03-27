#!/usr/bin/env bash
# lib/cmd_report.sh - Dispatch handler for `chainbench report`
# Sourced by the main chainbench dispatcher; $@ contains subcommand arguments.
#
# Usage: chainbench report [--format <text|json|markdown>]
#
# Delegates to tests/lib/report.sh::generate_report().

# Guard against double-sourcing
[[ -n "${_CB_CMD_REPORT_SH_LOADED:-}" ]] && return 0
readonly _CB_CMD_REPORT_SH_LOADED=1

_CB_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${_CB_LIB_DIR}/common.sh"

# ---- Constants ---------------------------------------------------------------

readonly _CB_REPORT_LIB="${_CB_LIB_DIR}/../tests/lib/report.sh"

# ---- Usage -------------------------------------------------------------------

_cb_report_usage() {
  cat >&2 <<'EOF'
Usage: chainbench report [--format <format>]

Options:
  --format <text|json|markdown>   Output format (default: text)
  -h, --help                      Show this help

Formats:
  text      Plain text summary table (default)
  json      Machine-readable JSON with summary and per-test details
  markdown  Markdown table suitable for CI/CD comments
EOF
}

# ---- Argument parsing --------------------------------------------------------

_cb_report_parse_args() {
  _CB_REPORT_FORMAT="text"

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --format)
        if [[ -z "${2:-}" ]]; then
          log_error "--format requires a value (text|json|markdown)"
          return 1
        fi
        _CB_REPORT_FORMAT="$2"
        shift 2
        ;;
      --format=*)
        _CB_REPORT_FORMAT="${1#--format=}"
        if [[ -z "$_CB_REPORT_FORMAT" ]]; then
          log_error "--format requires a non-empty value"
          return 1
        fi
        shift
        ;;
      --help|-h|help)
        _cb_report_usage
        return 0
        ;;
      *)
        log_warn "Unknown report option: '$1' (ignoring)"
        shift
        ;;
    esac
  done

  # Validate format value
  case "$_CB_REPORT_FORMAT" in
    text|json|markdown|md)
      ;;
    *)
      log_error "Unknown format '$_CB_REPORT_FORMAT'. Valid options: text, json, markdown"
      return 1
      ;;
  esac
}

# ---- Main --------------------------------------------------------------------

cmd_report_main() {
  local _CB_REPORT_FORMAT

  _cb_report_parse_args "$@" || return $?

  # Verify the report library exists
  if [[ ! -f "$_CB_REPORT_LIB" ]]; then
    log_error "Report library not found: $_CB_REPORT_LIB"
    return 1
  fi

  # shellcheck source=tests/lib/report.sh
  source "$_CB_REPORT_LIB"

  generate_report "$_CB_REPORT_FORMAT"
}

# ---- Entry point -------------------------------------------------------------

cmd_report_main "$@"
