#!/usr/bin/env bash
# uninstall.sh - Remove chainbench installation
# Usage: chainbench uninstall  OR  ~/.chainbench/uninstall.sh
set -euo pipefail

INSTALL_DIR="${HOME}/.chainbench"
SYMLINK_PATH="/usr/local/bin/chainbench"

echo "========================================="
echo "  chainbench uninstaller"
echo "========================================="
echo ""

# ---- Remove symlink ----------------------------------------------------------

if [[ -L "${SYMLINK_PATH}" ]]; then
  LINK_TARGET="$(readlink -f "${SYMLINK_PATH}" 2>/dev/null || realpath "${SYMLINK_PATH}" 2>/dev/null || echo "")"
  if [[ "${LINK_TARGET}" == *"chainbench"* ]]; then
    if [[ -w "$(dirname "${SYMLINK_PATH}")" ]]; then
      rm -f "${SYMLINK_PATH}"
    else
      sudo rm -f "${SYMLINK_PATH}"
    fi
    echo "[OK] Removed symlink: ${SYMLINK_PATH}"
  else
    echo "[SKIP] ${SYMLINK_PATH} points to ${LINK_TARGET}, not chainbench"
  fi
elif [[ -f "${SYMLINK_PATH}" ]]; then
  echo "[SKIP] ${SYMLINK_PATH} is not a symlink — skipping"
else
  echo "[SKIP] No symlink at ${SYMLINK_PATH}"
fi

# ---- Remove installation directory ------------------------------------------

if [[ -d "${INSTALL_DIR}" ]]; then
  echo ""
  printf "Remove ${INSTALL_DIR}? [y/N]: "
  read -r _confirm
  if [[ "${_confirm}" =~ ^[Yy]$ ]]; then
    rm -rf "${INSTALL_DIR}"
    echo "[OK] Removed ${INSTALL_DIR}"
  else
    echo "[SKIP] Kept ${INSTALL_DIR}"
  fi
else
  echo "[SKIP] No installation found at ${INSTALL_DIR}"
fi

echo ""
echo "chainbench has been uninstalled."
