"""Fresh WEB-03 browser + dedicated SSH acceptance, with no mock or skipped target."""
from runtime_contract import runtime_root, base_commit

import base64
import datetime
import json
import os
from pathlib import Path
import platform
import secrets
import shutil
import socket
import subprocess
import sys
import time
import urllib.request
import uuid
from evidence import digest
from evidence_web03 import REQUIRED, verify


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


def output_directories(requested):
    """Keep evidence paths portable even when a caller requests an external receipt."""
    output = Path('chainbench-out/web-ui-acceptance/WEB-03')
    if not output.resolve().is_relative_to(Path.cwd().resolve()):
        raise ValueError('workspace evidence directory must stay inside the workspace')
    destination = (Path(requested) / 'WEB-03').resolve()
    return output, destination


def publish_output(output, destination):
    if output.resolve() != destination:
        destination.mkdir(parents=True, exist_ok=True)
        for artifact in output.iterdir():
            if artifact.is_file():
                shutil.copy2(artifact, destination / artifact.name)


def main():
    output, destination = output_directories(sys.argv[1])
    output.mkdir(parents=True, exist_ok=True)
    # A failed invocation must not retain old successful observations.
    for p in output.iterdir():
        if p.is_file():
            p.unlink()
    invocation = str(uuid.uuid4())
    started = now()
    runtime = runtime_root() / invocation
    runtime.mkdir(parents=True, mode=0o700)
    evidence = {'criterion': 'WEB-03', 'invocationId': invocation, 'scenarios': [], 'checks': []}
    failures = []
    server = sshd = None
    handles = []
    try:
        for command in ('npm --prefix web run build', 'go test ./internal/app ./internal/dashboard ./internal/resource'):
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
        accounts = [{'id': 'operator-' + suffix, 'username': 'operator-' + suffix, 'role': 'operator', 'password': secrets.token_urlsafe(24)} for suffix in ('a', 'b')]
        account_source = runtime / 'accounts.go'
        account_source.write_bytes(Path('tests/webui/fixtures/accounts.go.txt').read_bytes())
        provision = subprocess.run(['go', 'run', str(account_source)], input=json.dumps(accounts), capture_output=True, text=True)
        if provision.returncode:
            raise RuntimeError('private account provisioning failed')
        accountfile = runtime / 'accounts.json'
        accountfile.write_text(provision.stdout)
        accountfile.chmod(0o600)
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
        fixture = {'accounts': accounts, 'ssh': ssh, 'store': str(runtime / 'store')}
        private_fixture = runtime / 'browser-fixture.json'
        private_fixture.write_text(json.dumps(fixture))
        private_fixture.chmod(0o600)
        with socket.socket() as sock:
            sock.bind(('127.0.0.1', 0))
            port = sock.getsockname()[1]
        url = f'http://127.0.0.1:{port}'
        command = [str(runtime / 'chainbench-dashboard'), '-addr', f'127.0.0.1:{port}', '-deployment-accounts', str(accountfile), '-deployment-root', str(runtime / 'store')]
        def launch(logname):
            logfile = (output / logname).open('w')
            handles.append(logfile)
            process = subprocess.Popen(command, stdout=logfile, stderr=logfile)
            wait_http(url, process)
            return process
        server = launch('server.log')
        browser = subprocess.run(['node', 'tests/webui/browser_web03.mjs', url, str(output), str(private_fixture)], capture_output=True, text=True)
        (output / 'browser.log').write_text(browser.stdout + browser.stderr)
        if browser.returncode:
            raise RuntimeError('live browser assertions failed; see browser.log')
        observed = json.loads((output / 'browser.json').read_text())
        stop(server)
        server = launch('server-restart.log')
        credentials = base64.b64encode((accounts[0]['username'] + ':' + accounts[0]['password']).encode()).decode()
        def request(path, body=None):
            req = urllib.request.Request(url + '/api/v1/' + path, data=json.dumps(body).encode() if body else None,
                                         headers={'Authorization': 'Basic ' + credentials, 'Content-Type': 'application/json'})
            with urllib.request.urlopen(req, timeout=30) as response:
                return json.load(response)
        restored = request('workspaces/' + observed['workspaceId'])
        if restored['revision'] != observed['workspaceRevision']:
            raise RuntimeError('shared revision not restored after restart')
        bindings = request('workspaces/' + observed['workspaceId'] + '/credential-bindings')
        if bindings['ssh'] != observed['credentialId']:
            raise RuntimeError('personal binding not restored')
        access = request('credentials/' + observed['credentialId'] + '/check', {'workspaceId': observed['workspaceId'], 'serverRef': 'ssh'})
        if not access['authenticated'] or access['allowedOperations'] != ['deploy']:
            raise RuntimeError('encrypted private overlay not usable after restart')
        evidence['restart'] = access
        evidence['server'] = {'sha256': digest(runtime / 'chainbench-dashboard'), 'baseCommit': base_commit(), 'url': url, 'contractVersion': '2', 'goVersion': subprocess.check_output(['go', 'version'], text=True).strip()}
        evidence['frontend'] = [{'path': str(p), 'sha256': digest(p)} for p in sorted(Path('internal/dashboard/spa').rglob('*')) if p.is_file()]
        evidence['target'] = {'host': 'localhost.', 'port': ssh['port'], 'hostFingerprint': fingerprint, 'sshdSha256': digest('/usr/sbin/sshd'), 'os': platform.system(), 'architecture': platform.machine(), 'allowedPathObserved': os.stat(runtime / 'ssh' / 'allowed').st_mode}
        evidence['fixture'] = {'runtime': str(runtime), 'transport': ['local', 'SSH'], 'ownership': 'owned', 'accountIds': [a['id'] for a in accounts], 'browserVersion': observed['browserVersion'], 'dedicatedSSH': True}
        evidence['coverage'] = {'required': REQUIRED, 'executed': [s['id'] for s in observed['scenarios']], 'missing': [], 'mockTargets': 0}
        evidence['scenarios'] = observed['scenarios']
        for scenario in evidence['scenarios']:
            scenario['invocationId'] = invocation
            scenario['artifacts'] = [{'path': str(output / p), 'sha256': digest(output / p)} for p in scenario['artifacts']]
        for p in output.iterdir():
            if p.suffix in ('.json', '.log'):
                content = p.read_text()
                for secret in [a['password'] for a in accounts] + [(runtime / 'ssh' / key).read_text() for key in ('client', 'unauthorized')]:
                    if secret in content:
                        raise RuntimeError('secret found in acceptance artifact ' + p.name)
        (output / 'sshd.log').write_text((runtime / 'ssh' / 'sshd.log').read_text())
        public_manifest = {**ssh, 'accountIds': [a['id'] for a in accounts], 'sshdSha256': evidence['target']['sshdSha256'], 'invocationId': invocation}
        (output / 'fixture.json').write_text(json.dumps(public_manifest, indent=2))
        evidence['source'] = [{'path': str(p), 'sha256': digest(p)} for pattern in ('internal/app/deployment*.go', 'internal/dashboard/deployment*.go', 'internal/dashboard/server.go', 'internal/dashboard/chainpreset.go', 'internal/core/session/webstore.go', 'internal/resource/serverset.go', 'internal/resource/serverset_load.go', 'internal/resource/ports.go', 'internal/resource/workspaceconfig.go', 'cmd/chainbench-dashboard/main.go', 'web/src/Deployment*.svelte', 'web/src/App.svelte', 'tests/webui/*web03*', 'tests/webui/fixtures/*') for p in sorted(Path('.').glob(pattern)) if p.is_file()]
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
    result = {'criterion': 'WEB-03', 'invocationId': invocation, 'startedAt': started, 'finishedAt': now(),
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
        publish_output(output, destination)
        print('\n'.join(failures + errors), file=sys.stderr)
        return 1
    publish_output(output, destination)
    return 0


if __name__ == '__main__':
    sys.exit(main())
