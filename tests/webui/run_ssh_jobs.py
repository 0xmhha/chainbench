"""Real owned loopback SSH deployment proof; does not award a Seed criterion."""
import importlib.util
import json
import os
from pathlib import Path
import secrets
import socket
import subprocess
import time
import urllib.request
import uuid
from runtime_contract import runtime_root
from browser_process import run_browser


def main():
    output = Path('chainbench-out/web-ui-development/ssh-jobs')
    output.mkdir(parents=True, exist_ok=True)
    identity = str(uuid.uuid4())
    runtime = runtime_root() / identity
    runtime.mkdir(parents=True, mode=0o700)
    spec = importlib.util.spec_from_file_location('ssh_native_fixture', 'tests/webui/fixtures/prepare_web04.py')
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    provenance = module.prepare(runtime, output)
    ssh = subprocess.Popen(['bash', 'tests/webui/fixtures/ssh_prepare.sh', identity])
    server = None
    try:
        for _ in range(100):
            if ssh.poll() is not None:
                raise RuntimeError('owned SSH fixture exited')
            if (runtime / 'ssh/manifest.json').exists():
                break
            time.sleep(.1)
        ssh_fixture = json.loads((runtime / 'ssh/manifest.json').read_text())
        for _ in range(100):
            try:
                with socket.create_connection(('127.0.0.1', ssh_fixture['port']), timeout=1):
                    break
            except OSError:
                time.sleep(.1)
        key_parts = (runtime / 'ssh/host.pub').read_text().split()
        known_hosts = runtime / 'ssh/known_hosts'
        known_hosts.write_text(f"[localhost.]:{ssh_fixture['port']} {key_parts[0]} {key_parts[1]}\n")
        known_hosts.chmod(0o600)
        with (output / 'build.log').open('w') as log:
            subprocess.run(['go', 'build', '-o', str(runtime / 'dashboard'), './cmd/chainbench-dashboard'], stdout=log, stderr=log, check=True)
        with socket.socket() as listener:
            listener.bind(('127.0.0.1', 0))
            port = listener.getsockname()[1]
        store = runtime / 'store'
        with (output / 'server.log').open('w') as log:
            server = subprocess.Popen([str(runtime / 'dashboard'), '-addr', f'127.0.0.1:{port}', '-deployment-root', str(store), '-manifest-assets', str(runtime / 'assets.json'), '-manifest-keys', str(Path('presets/keys').resolve())], stdout=log, stderr=log)
            url = f'http://127.0.0.1:{port}'
            for _ in range(200):
                if server.poll() is not None:
                    raise RuntimeError('dashboard exited')
                try:
                    urllib.request.urlopen(url + '/healthz', timeout=1).close()
                    break
                except OSError:
                    time.sleep(.1)
            fixture = {'url': url, 'setupToken': (store / 'setup.token').read_text().strip(), 'password': secrets.token_urlsafe(24), 'runtime': str(runtime), 'store': str(store), 'ssh': ssh_fixture, 'knownHosts': str(known_hosts)}
            private = runtime / 'browser-fixture.json'
            private.write_text(json.dumps(fixture))
            private.chmod(0o600)
            result = run_browser(['node', 'tests/webui/browser_ssh_jobs.mjs', str(private), str(output.resolve())], timeout=600)
            (output / 'browser.log').write_text(result.stdout + result.stderr)
            if result.returncode:
                raise RuntimeError('SSH browser deployment failed; see browser.log')
            records = list((store / 'networks').glob('*/chain-record.json'))
            if len(records) != 1:
                raise RuntimeError('expected one SSH-owned composition')
            state = json.loads(records[0].read_text())
            asset = next(a for a in json.loads((runtime / 'assets.json').read_text()) if a['id'] == 'wbft')
            from evidence_web04 import digest
            if not str(state['binary']).startswith(str(runtime / 'remote-data/binaries/')) or digest(Path(state['binary'])) != asset['sha256']:
                raise RuntimeError('remote binary was not separately uploaded and verified')
            declared = json.loads(Path(state['genesisPath']).read_text())
            for node in state['nodes']:
                data = Path(node['dataDir'])
                if len(list(data.glob('*/chaindata/CURRENT'))) != 1:
                    raise RuntimeError('remote native database missing')
                dumped = subprocess.run([state['binary'], '--datadir', str(data), 'dumpgenesis'], capture_output=True, text=True, check=True, timeout=20)
                if json.loads(dumped.stdout)['config']['chainId'] != declared['config']['chainId']:
                    raise RuntimeError('remote database genesis mismatch')
            for path in store.rglob('*'):
                if path.is_file() and path.suffix in {'.json', '.yaml'}:
                    text = path.read_text()
                    if 'BEGIN OPENSSH PRIVATE KEY' in text or 'BEGIN RSA PRIVATE KEY' in text:
                        raise RuntimeError('SSH private key persisted in plaintext')
            receipt = json.loads((output / 'browser.json').read_text())
            receipt.update({'nativeDatabaseCheck': 'passed', 'separateBinaryUpload': 'passed', 'privateSSHKeyPlaintextScan': 'passed', 'binaries': provenance, 'runtime': str(runtime), 'seedAcceptanceAwarded': False})
            (output / 'receipt.json').write_text(json.dumps(receipt, indent=2))
            print('SSH development proof PASS: real owned SSH deployment, native database, private credential and physical alias checks.')
    finally:
        # Only this exclusively owned fixture's recorded node PIDs qualify.
        for record in (runtime / 'store/networks').glob('*/chain-record.json'):
            for node in json.loads(record.read_text()).get('nodes', []):
                if node.get('pid'):
                    try:
                        os.kill(node['pid'], 15)
                    except ProcessLookupError:
                        pass
        for process in [server, ssh]:
            if process is not None:
                process.terminate()
                try:
                    process.wait(timeout=10)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait()


if __name__ == '__main__':
    main()
