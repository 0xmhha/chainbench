#!/usr/bin/env bash
# lib/cmd_stop.sh - Stop all running chain nodes
# Intentionally NOT using set -e: we want to continue stopping remaining nodes
# even if one fails.

source "${CHAINBENCH_DIR}/lib/pids_state.sh"

if ! pids_exists; then
  log_info "No pids.json found — nothing to stop"
  return 0 2>/dev/null || exit 0
fi

# Use pkill as the primary mechanism (reliable regardless of PID tracking)
_BINARY_NAME="${CHAINBENCH_BINARY:-gstable}"

log_info "Stopping all ${_BINARY_NAME} processes ..."

# First try SIGTERM
_pids_before=$(pgrep -f "${_BINARY_NAME} --datadir" 2>/dev/null || true)

if [[ -z "$_pids_before" ]]; then
  log_info "No running ${_BINARY_NAME} processes found"
else
  # Send SIGTERM to all matching processes
  echo "$_pids_before" | while read -r _pid; do
    if kill -0 "$_pid" 2>/dev/null; then
      log_info "  SIGTERM → PID ${_pid}"
      kill -TERM "$_pid" 2>/dev/null || true
    fi
  done

  # Wait up to 10 seconds for graceful shutdown
  log_info "Waiting for graceful shutdown (max 10s) ..."
  _waited=0
  while (( _waited < 20 )); do
    _remaining=$(pgrep -f "${_BINARY_NAME} --datadir" 2>/dev/null || true)
    if [[ -z "$_remaining" ]]; then
      break
    fi
    sleep 0.5
    (( _waited++ )) || true
  done

  # SIGKILL any remaining
  _remaining=$(pgrep -f "${_BINARY_NAME} --datadir" 2>/dev/null || true)
  if [[ -n "$_remaining" ]]; then
    log_warn "Some processes still alive, sending SIGKILL ..."
    echo "$_remaining" | while read -r _pid; do
      kill -KILL "$_pid" 2>/dev/null || true
    done
    sleep 1
  fi

  # Also kill any logrot processes
  pkill -f "logrot.*node.*log" 2>/dev/null || true
fi

# Verify all stopped
_final=$(pgrep -f "${_BINARY_NAME} --datadir" 2>/dev/null || true)
if [[ -n "$_final" ]]; then
  log_error "Failed to stop some processes: ${_final}"
else
  log_success "All ${_BINARY_NAME} processes stopped"
fi

# Update pids.json
pids_mark_all_stopped 2>/dev/null || true

log_success "pids.json updated — all nodes marked stopped"
