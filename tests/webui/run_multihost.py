"""Two-server placement and per-node work on Docker servers; development proof only.

The dashboard runs in a control container on the chainbench-servers network,
because this machine cannot route to container addresses. server1 is the
existing env/docker fixture; this proof adds its own server2 and control
containers, uses a data root of its own on both servers, and stops what it
started when it ends. The browser drives the dashboard through one published
loopback port. No Seed criterion is awarded."""
import datetime
import hashlib
import json
import os
from pathlib import Path
import platform
import secrets
import shutil
import socket
import ssl
import subprocess
import time
import urllib.request
import uuid
from runtime_contract import runtime_root
from browser_process import run_browser

NETWORK = 'chainbench-servers_chainbench'
IMAGE = 'chainbench-server:ubuntu24'
SERVER2, CONTROL = 'chainbench-mh-server2', 'chainbench-mh-control'


def docker(*args, check=True, capture=True):
    return subprocess.run(['docker', *args], check=check, text=True, capture_output=capture)


def running(name):
    r = docker('inspect', '-f', '{{.State.Running}}', name, check=False)
    return r.returncode == 0 and r.stdout.strip() == 'true'


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def account_password():
    for line in Path('env/docker/accounts.env').read_text().splitlines():
        if line.startswith('DEV_ACCOUNTS_PASSWORD='):
            return line.split('=', 1)[1].strip().strip("'\"")
    raise RuntimeError('env/docker/accounts.env names no DEV_ACCOUNTS_PASSWORD')


def main():
    output = Path('chainbench-out/web-ui-development/multihost')
    output.mkdir(parents=True, exist_ok=True)
    for p in output.iterdir():
        if p.is_file():
            p.unlink()
    runtime = runtime_root() / str(uuid.uuid4())
    runtime.mkdir(parents=True, mode=0o700)
    invocation = uuid.uuid4().hex[:12]
    data_root = f'/data/chainbench/mh-{invocation}'
    started_server1 = False
    fixture = runtime / 'browser-fixture.json'
    try:
        if docker('network', 'inspect', NETWORK, check=False).returncode:
            raise RuntimeError(NETWORK + ' does not exist; bring the servers up with env/docker/setup.sh up')
        if not running('chainbench-server1'):
            docker('start', 'chainbench-server1')
            started_server1 = True
        for name in (SERVER2, CONTROL):
            if docker('inspect', name, check=False).returncode == 0:
                docker('rm', '-f', name)  # this proof's own containers from an earlier run
        arch = {'arm64': 'arm64', 'aarch64': 'arm64', 'x86_64': 'amd64'}[platform.machine()]
        work = runtime / 'work'
        work.mkdir()
        env = {**os.environ, 'CGO_ENABLED': '0', 'GOOS': 'linux', 'GOARCH': arch}
        subprocess.run(['go', 'build', '-o', str(work / 'chainbench-dashboard'), './cmd/chainbench-dashboard'], env=env, check=True)
        subprocess.run(['go', 'build', '-o', str(work / 'chainbench'), './cmd/chainbench'], env=env, check=True)
        gstable = Path(os.environ.get('LINUX_GSTABLE_BIN', str(Path.home() / 'cbw/linuxbin/gstable')))
        shutil.copy2(gstable, work / 'gstable')
        shutil.copytree('presets/keys', work / 'keys')
        shutil.copytree('presets/chain', work / 'presets')
        shutil.copytree('docs/chain-analysis', work / 'chain-analysis')
        accounts = Path('env/docker/accounts.env').resolve()
        scripts = Path('env/docker').resolve()
        # server2 is started the way gen-env.sh starts every server.
        docker('run', '-d', '--name', SERVER2, '--hostname', 'server2', '--network', NETWORK, '--ip', '172.30.0.12', '--cap-add', 'NET_ADMIN',
               '-v', f'{accounts}:/etc/chainbench/accounts.env:ro', '-v', f'{scripts}/setup-accounts.sh:/usr/local/lib/chainbench/setup-accounts.sh:ro',
               '-v', f'{scripts}/firewall.sh:/usr/local/lib/chainbench/firewall.sh:ro', IMAGE, '/bin/sh', '-c',
               'SLOTS=4 P2P_SPAN=12 sh /usr/local/lib/chainbench/firewall.sh && sh /usr/local/lib/chainbench/setup-accounts.sh && mkdir -p /data/chainbench && chown devuser1:devuser1 /data/chainbench && exec /usr/sbin/sshd -D -e')
        # The image has an empty /etc/machine-id (nothing generates one without
        # systemd), and the Web tells machines apart by it. A server without
        # one gets a random id once; server1 keeps it for later runs.
        for server in ('chainbench-server1', SERVER2):
            docker('exec', '-u', 'root', server, 'sh', '-c', f'test -s /etc/machine-id || echo {uuid.uuid4().hex} > /etc/machine-id')
        version = docker('run', '--rm', '-v', f'{work}:/work:ro', '--entrypoint', '/work/gstable', IMAGE, 'version').stdout
        commit = next((l.split(':', 1)[1].strip() for l in version.splitlines() if l.startswith('Git Commit:')), '')
        if len(commit) != 40:
            raise RuntimeError('linux gstable reports no commit')
        (work / 'assets.json').write_text(json.dumps([{'id': 'stablenet', 'chain': 'stablenet', 'path': '/work/gstable', 'sha256': digest(work / 'gstable'), 'commit': commit}]))
        # Team mode on a reachable address needs TLS: a throwaway certificate
        # for the loopback address the browser uses.
        subprocess.run(['openssl', 'req', '-x509', '-newkey', 'ec', '-pkeyopt', 'ec_paramgen_curve:prime256v1', '-nodes', '-days', '1',
                        '-subj', '/CN=127.0.0.1', '-addext', 'subjectAltName=IP:127.0.0.1', '-keyout', str(work / 'tls.key'), '-out', str(work / 'tls.crt')],
                       check=True, capture_output=True)
        (work / 'tls.key').chmod(0o644)  # read by the control container's user
        with socket.socket() as s:
            s.bind(('127.0.0.1', 0))
            port = s.getsockname()[1]
        state = runtime / 'state'
        state.mkdir()
        docker('run', '-d', '--name', CONTROL, '--hostname', 'control', '--network', NETWORK, '--ip', '172.30.0.5', '-p', f'127.0.0.1:{port}:8080',
               '-v', f'{work}:/work:ro', '-v', f'{state}:/state', '--entrypoint', '/work/chainbench-dashboard', IMAGE,
               '-addr', '0.0.0.0:8080', '-web-mode', 'team', '-deployment-root', '/state', '-manifest-assets', '/work/assets.json',
               '-manifest-keys', '/work/keys', '-chain-presets', '/work/presets', '-chain-analysis', '/work/chain-analysis',
               '-tls-cert', '/work/tls.crt', '-tls-key', '/work/tls.key')
        # The control plane inspects its own machine as well; same empty id.
        docker('exec', '-u', 'root', CONTROL, 'sh', '-c', f'test -s /etc/machine-id || echo {uuid.uuid4().hex} > /etc/machine-id')
        url = f'https://127.0.0.1:{port}'
        insecure = ssl.create_default_context()
        insecure.check_hostname = False
        insecure.verify_mode = ssl.CERT_NONE
        for _ in range(300):
            try:
                urllib.request.urlopen(url + '/healthz', timeout=1, context=insecure).close()
                break
            except OSError:
                if not running(CONTROL):
                    raise RuntimeError('control dashboard exited: ' + docker('logs', CONTROL, check=False).stdout[-2000:] + docker('logs', CONTROL, check=False).stderr[-2000:])
                time.sleep(0.2)
        else:
            raise RuntimeError('control dashboard did not listen')
        ssh_password = account_password()
        fixture.write_text(json.dumps({'url': url, 'setupToken': (state / 'setup.token').read_text().strip(), 'password': secrets.token_urlsafe(24),
                                       'sshUser': 'devuser1', 'sshPassword': ssh_password, 'dataRoot': data_root, 'invocation': invocation}))
        fixture.chmod(0o600)
        result = run_browser(['node', 'tests/webui/browser_multihost.mjs', str(fixture), str(output.resolve())], timeout=1800)
        (output / 'browser.log').write_text(result.stdout + result.stderr)
        (output / 'control.log').write_text(docker('logs', CONTROL, check=False).stdout + docker('logs', CONTROL, check=False).stderr)
        for p in output.iterdir():
            if p.is_file() and ssh_password in p.read_text(errors='ignore'):
                raise RuntimeError('the SSH password reached ' + p.name)
        if result.returncode:
            raise RuntimeError('multihost browser proof failed; see browser.log')
        receipt = json.loads((output / 'browser.json').read_text())
        receipt.update({'runtime': str(runtime), 'invocation': invocation, 'dashboardSHA256': digest(work / 'chainbench-dashboard'),
                        'gstableSHA256': digest(work / 'gstable'), 'gstableCommit': commit, 'finishedAt': datetime.datetime.now(datetime.timezone.utc).isoformat(),
                        'seedAcceptanceAwarded': False})
        (output / 'receipt.json').write_text(json.dumps(receipt, indent=2))
        print('multihost development proof PASS; no Seed criterion awarded.')
    finally:
        fixture.unlink(missing_ok=True)
        # Nodes of this proof live under its own data root on both servers.
        for server in ('chainbench-server1', SERVER2):
            if running(server):
                docker('exec', '-u', 'root', server, 'sh', '-c', f'pkill -f "{data_root}" ; rm -rf "{data_root}"', check=False)
        for name in (CONTROL, SERVER2):
            docker('stop', name, check=False)
        if started_server1:
            docker('stop', 'chainbench-server1', check=False)


if __name__ == '__main__':
    main()
