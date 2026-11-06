#!/usr/bin/env bash
# Keep chainbench-dashboard running on this Mac across logins and crashes.
# The Web's data stays in the checkout's chainbench-out/web-ui on this machine:
# this registers a service and moves no data between machines.
# Usage: dashboard-service.sh install|uninstall|status
set -euo pipefail

readonly LABEL=com.chainbench.dashboard
readonly PORT=8787
readonly HEALTH_URL="http://127.0.0.1:$PORT/healthz"
readonly HEALTH_ATTEMPTS=15

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
readonly root
readonly plist="$HOME/Library/LaunchAgents/$LABEL.plist"
readonly log="$root/chainbench-out/dashboard.log"
target="gui/$(id -u)"
readonly target

die() { echo "dashboard-service: $*" >&2; exit 1; }

require_macos() {
  local system
  system=$(uname -s)
  [ "$system" = Darwin ] || die "launchd is macOS only; on $system run bin/chainbench-dashboard under your own service manager"
}

loaded() { launchctl print "$target/$LABEL" >/dev/null 2>&1; }

unload() { if loaded; then launchctl bootout "$target/$LABEL"; fi; }

xml_escape() { printf '%s' "$1" | sed -e 's/&/\&amp;/g' -e 's/</\&lt;/g' -e 's/>/\&gt;/g'; }

write_plist() {
  local binary_xml root_xml log_xml path_xml
  binary_xml=$(xml_escape "$root/bin/chainbench-dashboard")
  root_xml=$(xml_escape "$root")
  log_xml=$(xml_escape "$log")
  path_xml=$(xml_escape "$PATH")
  mkdir -p "$(dirname "$plist")" "$(dirname "$log")"
  (umask 077; cat > "$plist") <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>$LABEL</string>
  <key>ProgramArguments</key><array><string>$binary_xml</string></array>
  <key>WorkingDirectory</key><string>$root_xml</string>
  <key>EnvironmentVariables</key><dict><key>PATH</key><string>$path_xml</string></dict>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardOutPath</key><string>$log_xml</string>
  <key>StandardErrorPath</key><string>$log_xml</string>
</dict>
</plist>
EOF
}

wait_healthy() {
  local attempt
  for ((attempt = 0; attempt < HEALTH_ATTEMPTS; attempt++)); do
    if curl -fs --max-time 1 -o /dev/null "$HEALTH_URL"; then return 0; fi
    sleep 1
  done
  return 1
}

install_service() {
  # launchd keeps the replaced build running until the old job is gone.
  unload
  local holder
  holder=$(lsof -nP -iTCP:"$PORT" -sTCP:LISTEN -t 2>/dev/null || true)
  [ -z "$holder" ] || die "port $PORT is held by process $holder; stop it first"
  write_plist
  launchctl bootstrap "$target" "$plist"
  if ! wait_healthy; then
    unload
    rm -f "$plist"
    die "the dashboard did not answer on port $PORT; see $log"
  fi
  echo "dashboard running on http://127.0.0.1:$PORT and starting at login"
}

uninstall_service() {
  unload
  rm -f "$plist"
  echo "dashboard service removed; data is kept in $root/chainbench-out/web-ui"
}

status_service() {
  if loaded; then echo "dashboard service loaded"; return 0; fi
  echo "dashboard service not loaded"
  return 1
}

case "${1:-}" in
  install) require_macos; install_service ;;
  uninstall) require_macos; uninstall_service ;;
  status) require_macos; status_service ;;
  *) echo "usage: dashboard-service.sh install|uninstall|status" >&2; exit 2 ;;
esac
