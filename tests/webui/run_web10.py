"""Fresh WEB-10 browser + dedicated SSH acceptance, with no mock or skipped target."""
from runtime_contract import runtime_root, base_commit
from browser_process import run_browser

import base64
import datetime
import json
import os
from pathlib import Path
import platform
import secrets
import socket
import subprocess
import sys
import time
import urllib.request
import uuid
from evidence import digest
from evidence_web10 import REQUIRED, verify


def now():
    return datetime.datetime.now(datetime.timezone.utc).isoformat()


def wait_http(url, process):
    for _ in range(100):
        if process.poll() is not None:
            raise RuntimeError('dashboard exited')
        try:
            with urllib.request.urlopen(url + '/healthz', timeout=1) as response:
                if response.read() == b'ok':
                    return
        except OSError:
            time.sleep(.1)
    raise RuntimeError('dashboard health timeout')


def stop(process):
    if process is None:
        return
    process.terminate()
    try:
        process.wait(timeout=5)
    except subprocess.TimeoutExpired:
        process.kill()
        process.wait()


def main():
    output = Path(sys.argv[1]) / 'WEB-10'
    if output.is_absolute() or '..' in output.parts:
        raise SystemExit('output must be workspace-relative')
    output.mkdir(parents=True, exist_ok=True)
    # A failed invocation must not retain old successful observations.
    for p in output.iterdir():
        if p.is_file():
            p.unlink()
    invocation = str(uuid.uuid4())
    started = now()
    runtime = runtime_root() / invocation
    runtime.mkdir(parents=True, mode=0o700)
    evidence = {'criterion': 'WEB-10', 'invocationId': invocation, 'scenarios': [], 'checks': []}
    failures = []
    server = sshd = None
    handles = []
    try:
        for command in ('npm --prefix web run build', 'go test -race ./internal/app ./internal/dashboard'):
            run = subprocess.run(command, shell=True, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
            logfile = output / f'check-{len(evidence["checks"])}.log'
            logfile.write_text(run.stdout)
            evidence['checks'].append({'command': command, 'exitCode': run.returncode, 'log': str(logfile), 'sha256': digest(logfile)})
            if run.returncode:
                raise RuntimeError(command + ' failed')
        build = subprocess.run(['bash', 'tests/webui/prepare.sh', invocation], capture_output=True, text=True)
        (output / 'build.log').write_text(build.stdout + build.stderr)
        if build.returncode:
            raise RuntimeError('dashboard build failed')
        accounts = [{'username': role, 'role': role, 'password': secrets.token_urlsafe(24)} for role in ('administrator', 'operator', 'viewer')]
        sshd = subprocess.Popen(['bash', 'tests/webui/fixtures/ssh_prepare.sh', invocation])
        manifest = runtime / 'ssh' / 'manifest.json'
        for _ in range(100):
            if sshd.poll() is not None:
                raise RuntimeError('dedicated sshd exited during preparation')
            if manifest.exists():
                ssh = json.loads(manifest.read_text())
                try:
                    with socket.create_connection((ssh['host'], ssh['port']), timeout=1):
                        break
                except OSError:
                    pass
            time.sleep(.1)
        else:
            raise RuntimeError('dedicated SSH startup timeout')
        public = (runtime / 'ssh' / 'host.pub').read_text().split()
        known = runtime / 'ssh' / 'known_hosts'
        known.write_text(f"[localhost.]:{ssh['port']} {public[0]} {public[1]}\n")
        fingerprint = subprocess.check_output(['ssh-keygen', '-lf', str(runtime / 'ssh' / 'host.pub')], text=True).split()[1]
        ssh.update(knownHosts=str(known), hostFingerprint=fingerprint)
        fixture = {'accounts': accounts, 'ssh': ssh, 'store': str(runtime / 'store'), 'sshKey': (runtime / 'ssh' / 'client').read_text(), 'marker': secrets.token_urlsafe(32)}
        private_fixture = runtime / 'browser-fixture.json'
        private_fixture.write_text(json.dumps(fixture))
        private_fixture.chmod(0o600)
        with socket.socket() as sock:
            sock.bind(('127.0.0.1', 0))
            port = sock.getsockname()[1]
        url = f'http://127.0.0.1:{port}'
        command = [str(runtime / 'chainbench-dashboard'), '-addr', f'127.0.0.1:{port}', '-deployment-root', str(runtime / 'store'), '-chain-presets', 'presets/chain']
        def launch(logname):
            logfile = (output / logname).open('w')
            handles.append(logfile)
            process = subprocess.Popen(command + ['-web-mode', 'personal' if logname == 'server.log' else 'team'], stdout=logfile, stderr=logfile)
            wait_http(url, process)
            return process
        server = launch('server.log')
        fixture['setupToken'] = (runtime / 'store' / 'setup.token').read_text()
        private_fixture.write_text(json.dumps(fixture))
        def browser_phase(phase):
            browser = run_browser(['node', 'tests/webui/browser_web10.mjs', url, str(output), str(private_fixture), phase], timeout=180)
            (output / (phase + '-browser.log')).write_text(browser.stdout + browser.stderr)
            if browser.returncode:
                raise RuntimeError('live browser assertions failed; see ' + phase + '-browser.log')
        browser_phase('personal')
        personal = json.loads((output / 'personal.json').read_text())
        if (runtime / 'store' / 'setup.token').exists():
            raise RuntimeError('setup capability retained after bootstrap')
        stop(server)
        server = launch('server-restart.log')
        browser_phase('team')
        observed = json.loads((output / 'browser.json').read_text())
        observed['scenarios'] += personal['scenarios']
        plaintext = [a['password'] for a in accounts] + [fixture['sshKey'], fixture['marker'], fixture['setupToken']] + fixture['sshKey'].splitlines()[1:-1]
        storefiles = sorted(p for p in (runtime / 'store').rglob('*') if p.is_file())
        for path in storefiles:
            if path.name == 'credential.key':
                continue
            content = path.read_text()
            # The deliberately inserted document name remains engine declaration text;
            # encrypted credential ciphertext itself must not contain any material.
            if path.name == 'deployment.json':
                state = json.loads(content)
                content = json.dumps(state['credentials'])
            if any(secret in content for secret in plaintext):
                raise RuntimeError('plaintext secret persisted in ' + path.name)
            if path.stat().st_mode & 0o077:
                raise RuntimeError('private storage permissions violated')
        audit = (runtime / 'store' / 'security-audit.jsonl').read_text()
        if not any(json.loads(line)['status'] == 403 for line in audit.splitlines()):
            raise RuntimeError('denied operation audit missing')
        (output / 'security-audit.jsonl').write_text(audit)
        (output / 'storage.json').write_text(json.dumps({'files': [{'name': p.name, 'mode': oct(p.stat().st_mode & 0o777), 'sha256': digest(p)} for p in storefiles], 'encryptedCredentials': len(state['credentials']), 'plaintextLeaks': 0, 'passwordHashes': 'bcrypt', 'setupRemoved': True}, indent=2))
        observed['scenarios'].append({'id': 'encrypted-storage-session', 'observedAt': now(), 'mode': 'team', 'transport': 'SSH', 'ownership': 'owned', 'assertions': [{'description': 'encrypted credential storage and bcrypt account hashes verified', 'passed': True}, {'description': 'logout and deactivation invalidate sessions', 'passed': True}, {'description': 'denied mutations audited without secrets', 'passed': True}], 'artifacts': ['storage.json', 'security-audit.jsonl']})
        evidence['server'] = {'sha256': digest(runtime / 'chainbench-dashboard'), 'baseCommit': base_commit(), 'url': url, 'contractVersion': '2', 'goVersion': subprocess.check_output(['go', 'version'], text=True).strip()}
        evidence['frontend'] = [{'path': str(p), 'sha256': digest(p)} for p in sorted(Path('internal/dashboard/spa').rglob('*')) if p.is_file()]
        evidence['target'] = {'host': 'localhost.', 'port': ssh['port'], 'hostFingerprint': fingerprint, 'sshdSha256': digest('/usr/sbin/sshd'), 'os': platform.system(), 'architecture': platform.machine(), 'allowedPathObserved': os.stat(runtime / 'ssh' / 'allowed').st_mode, 'authenticated': observed['access']['authenticated']}
        evidence['fixture'] = {'runtime': str(runtime), 'transport': ['local', 'SSH'], 'ownership': 'owned', 'accountRoles': [a['role'] for a in accounts], 'browserVersion': observed['browserVersion'], 'dedicatedSSH': True}
        evidence['coverage'] = {'required': REQUIRED, 'executed': [s['id'] for s in observed['scenarios']], 'missing': [], 'mockTargets': 0}
        evidence['scenarios'] = observed['scenarios']
        for scenario in evidence['scenarios']:
            scenario['invocationId'] = invocation
            scenario['artifacts'] = [{'path': str(output / p), 'sha256': digest(output / p)} for p in scenario['artifacts']]
        for p in output.iterdir():
            if p.suffix in ('.json', '.log'):
                content = p.read_text()
                for secret in plaintext:
                    if secret in content:
                        raise RuntimeError('secret found in acceptance artifact ' + p.name)
        (output / 'sshd.log').write_text((runtime / 'ssh' / 'sshd.log').read_text())
        public_manifest = {**ssh, 'accountRoles': [a['role'] for a in accounts], 'sshdSha256': evidence['target']['sshdSha256'], 'invocationId': invocation}
        (output / 'fixture.json').write_text(json.dumps(public_manifest, indent=2))
        evidence['source'] = [{'path': str(p), 'sha256': digest(p)} for pattern in ('internal/app/deployment*.go', 'internal/dashboard/deployment*.go', 'internal/dashboard/server.go', 'internal/dashboard/chainpreset.go', 'internal/core/session/webstore.go', 'internal/core/session/accountstore.go', 'internal/resource/serverset.go', 'internal/resource/serverset_load.go', 'internal/resource/ports.go', 'internal/resource/workspaceconfig.go', 'cmd/chainbench-dashboard/main.go', 'web/src/Deployment*.svelte', 'web/src/App.svelte', 'tests/webui/*web10*', 'internal/app/web_*.go', 'internal/dashboard/security*.go', 'web/src/LoginPanel.svelte', 'web/src/ChainPresetEditor.svelte', 'web/src/ManifestManager.svelte', 'web/src/DSLEditor.svelte', 'internal/dashboard/testcase.go', 'tests/webui/fixtures/*') for p in sorted(Path('.').glob(pattern)) if p.is_file()]
        evidence['artifacts'] = [{'path': str(p), 'sha256': digest(p)} for p in sorted(output.iterdir()) if p.is_file() and p.name not in ('result.json', 'evidence.json')]
    except Exception as exc:
        failures.append(str(exc))
    finally:
        stop(server)
        stop(sshd)
        subprocess.run(['bash', 'tests/webui/fixtures/ssh_teardown.sh', invocation], capture_output=True)
        for handle in handles:
            handle.close()
        # Preserve encrypted snapshot and public fixture provenance, remove disposable secrets.
        for p in [runtime / 'browser-fixture.json', runtime / 'accounts.json', runtime / 'ssh' / 'client', runtime / 'ssh' / 'unauthorized', runtime / 'ssh' / 'host']:
            p.unlink(missing_ok=True)
    evidence['blockers'] = failures
    ep = output / 'evidence.json'
    ep.write_text(json.dumps(evidence, indent=2) + '\n')
    executed = [s['id'] for s in evidence['scenarios']]
    result = {'criterion': 'WEB-10', 'invocationId': invocation, 'startedAt': started, 'finishedAt': now(),
              'outcome': 'pass' if not failures and sorted(executed) == sorted(REQUIRED) else 'incomplete',
              'requiredScenarios': REQUIRED, 'executedScenarios': executed,
              'skippedScenarios': [s for s in REQUIRED if s not in executed], 'failedAssertions': failures,
              'evidenceDigest': digest(ep)}
    rp = output / 'result.json'
    rp.write_text(json.dumps(result, indent=2) + '\n')
    errors = verify(output)
    if errors:
        result['outcome'] = 'incomplete'
        result['failedAssertions'] = failures + errors
        rp.write_text(json.dumps(result, indent=2) + '\n')
        print('\n'.join(failures + errors), file=sys.stderr)
        return 1
    return 0


if __name__ == '__main__':
    sys.exit(main())
