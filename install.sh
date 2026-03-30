#!/usr/bin/env bash
# install.sh - One-command installer for chainbench
# Usage: curl -fsSL https://raw.githubusercontent.com/0xmhha/chainbench/main/install.sh | bash
set -euo pipefail

# Ensure a valid CWD — previous uninstall may have removed the directory
# the user was standing in, leaving a dangling CWD (getcwd ENOENT).
cd "${HOME}" 2>/dev/null || cd /

REPO_URL="https://github.com/0xmhha/chainbench.git"
INSTALL_DIR="${HOME}/.chainbench"

echo "========================================="
echo "  chainbench installer"
echo "========================================="
echo ""

# ---- Check prerequisites ----------------------------------------------------

_check_cmd() {
  if ! command -v "$1" &>/dev/null; then
    echo "ERROR: $1 is required but not found."
    echo "  $2"
    exit 1
  fi
}

_check_cmd git     "Install git: https://git-scm.com"
_check_cmd bash    "Install bash 4.0+"
_check_cmd python3 "Install python3 3.6+"
_check_cmd curl    "Install curl"
_check_cmd node    "Install Node.js 18+: https://nodejs.org"
_check_cmd npm     "Install npm (comes with Node.js)"

NODE_VERSION=$(node -v | sed 's/v//' | cut -d. -f1)
if (( NODE_VERSION < 18 )); then
  echo "ERROR: Node.js 18+ required (found v${NODE_VERSION})"
  exit 1
fi

echo "[OK] Prerequisites: git, bash, python3, curl, node $(node -v), npm $(npm -v)"

# ---- Clone or update ---------------------------------------------------------

echo ""
if [[ -d "${INSTALL_DIR}" ]]; then
  echo "[1/2] Updating existing installation ..."
  cd "${INSTALL_DIR}"
  git pull --ff-only origin main 2>/dev/null || {
    echo "  [WARN] git pull failed — reinstalling ..."
    cd "${HOME}"
    rm -rf "${INSTALL_DIR}"
    git clone "${REPO_URL}" "${INSTALL_DIR}"
    cd "${INSTALL_DIR}"
  }
else
  echo "[1/2] Cloning chainbench ..."
  git clone "${REPO_URL}" "${INSTALL_DIR}"
  cd "${INSTALL_DIR}"
fi

echo "  [OK] Installed to ${INSTALL_DIR}"

# ---- Run setup ---------------------------------------------------------------

echo ""
echo "[2/2] Running setup ..."
echo ""

bash "${INSTALL_DIR}/setup.sh"
