#!/usr/bin/env bash
# run.sh — build and run the SSH stand-in container, then run the RemoteDriver
# E2E against it. Use this to test the SSH RemoteDriver "like a remote server"
# locally (requires a running Docker daemon).
#
#   bash tests/remote/sshd/run.sh          # build + run container + E2E + cleanup
#   bash tests/remote/sshd/run.sh --keep   # leave the container running
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
IMAGE="chainbench-remote-sshd"
NAME="chainbench-remote-sshd"
PORT="${CHAINBENCH_REMOTE_PORT:-2222}"
KEEP=0
[ "${1:-}" = "--keep" ] && KEEP=1

command -v docker >/dev/null || { echo "docker not found"; exit 2; }
docker info >/dev/null 2>&1 || { echo "docker daemon not running"; exit 2; }

cleanup() { [ "$KEEP" = 0 ] && docker rm -f "$NAME" >/dev/null 2>&1 || true; }
trap cleanup EXIT

echo "[1/3] building image..."
docker build -q -t "$IMAGE" "$(dirname "${BASH_SOURCE[0]}")" >/dev/null

echo "[2/3] starting sshd container on 127.0.0.1:${PORT}..."
docker rm -f "$NAME" >/dev/null 2>&1 || true
docker run -d --name "$NAME" -p "127.0.0.1:${PORT}:22" "$IMAGE" >/dev/null
# wait for sshd to accept connections
for _ in $(seq 1 30); do
  if (exec 3<>"/dev/tcp/127.0.0.1/${PORT}") 2>/dev/null; then break; fi
  sleep 0.5
done

echo "[3/3] running RemoteDriver E2E..."
cd "$REPO"
CHAINBENCH_REMOTE_HOST=127.0.0.1 \
CHAINBENCH_REMOTE_PORT="$PORT" \
CHAINBENCH_REMOTE_USER=chainbench \
CHAINBENCH_REMOTE_PASS=chainbench \
  go test -tags e2e -run TestRemoteDriver_E2E -v ./pkg/core/driver/

echo "OK"
