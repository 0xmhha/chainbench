#!/usr/bin/env bash
set -euo pipefail
[[ $# == 1 && "$1" =~ ^[a-zA-Z0-9-]+$ ]] || exit 2
runtime="${WEBUI_RUNTIME_ROOT:-/private/tmp/chainbench-web-ui-e2e}/$1/ssh"
mkdir -p "$runtime/allowed" "$runtime/denied"
chmod 700 "$runtime"
ssh-keygen -q -t ed25519 -N '' -f "$runtime/host"
ssh-keygen -q -t ed25519 -N '' -f "$runtime/client"
ssh-keygen -q -t ed25519 -N '' -f "$runtime/unauthorized"
cp "$runtime/client.pub" "$runtime/authorized_keys"
port=$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1])')
cat > "$runtime/sshd_config" <<CONFIG
ListenAddress 127.0.0.1
Port $port
HostKey $runtime/host
PidFile $runtime/sshd.pid
AuthorizedKeysFile $runtime/authorized_keys
StrictModes no
PasswordAuthentication no
KbdInteractiveAuthentication no
UsePAM no
AllowUsers $(id -un)
Subsystem sftp internal-sftp
CONFIG
pid=$$
echo "$pid" > "$runtime/fixture.pid"
python3 - "$runtime" "$port" "$(id -un)" "$pid" <<'PY'
import json,sys,pathlib
p,port,user,pid=sys.argv[1:]
pathlib.Path(p,'manifest.json').write_text(json.dumps(dict(host='127.0.0.1',port=int(port),user=user,pid=int(pid),runtime=p)))
PY

exec /usr/sbin/sshd -D -e -f "$runtime/sshd_config" > "$runtime/sshd.log" 2>&1
