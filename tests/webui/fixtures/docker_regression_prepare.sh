#!/usr/bin/env bash
# Place the files the docker-gated Go live tests read on chainbench-server1.
# Each test documents its own setup; this runs them all, idempotently, so a
# regression run never reports one of those tests as skipped for want of it.
set -euo pipefail
server=chainbench-server1
[[ "$(docker inspect -f '{{.State.Running}}' "$server" 2>/dev/null)" == true ]] || { echo "$server is not running; bring it up with env/docker/setup.sh" >&2; exit 2; }
root=$(cd "$(dirname "$0")/../../.." && pwd)
# Files the tests read as the login account belong to it: a root-owned keys
# directory would refuse the next chain setup's password upload.
user=devuser1
docker exec -u root "$server" sh -c "mkdir -p /data/chainbench/keys && chown $user: /data/chainbench /data/chainbench/keys"
docker exec -u "$user" "$server" sh -c 'printf "start\n%s\nlast-line\n" "$(seq 1 400)" > /data/chainbench/nodelog-test.log'
docker exec -u root "$server" rm -rf /data/chainbench/keys/kr-src
docker cp "$root/presets/keys" "$server:/data/chainbench/keys/kr-src"
docker exec -u root "$server" chown -R "$user:" /data/chainbench/keys/kr-src
docker exec -u root "$server" sh -c 'echo root-only-test-content-12345 > /root/cb-sudo-test.txt && chmod 600 /root/cb-sudo-test.txt'
docker exec -u root "$server" sh -c 'rm -rf /root/cb-kr-test && mkdir -p /root/cb-kr-test/node1 && echo pw > /root/cb-kr-test/password && \
  echo meta > /root/cb-kr-test/metadata.json && echo k1 > /root/cb-kr-test/node1/keystore.json && \
  chmod -R 600 /root/cb-kr-test && find /root/cb-kr-test -type d -exec chmod 700 {} \;'
docker exec "$server" test -x /data/chainbench/bin/gstable || { echo "$server has no /data/chainbench/bin/gstable; run env/docker/setup.sh binaries go-stablenet" >&2; exit 2; }
