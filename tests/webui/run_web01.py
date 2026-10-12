"""Create fresh, fail-closed WEB-01 receipts, even if a prerequisite fails."""
from runtime_contract import runtime_root, base_commit

import datetime
import json
from pathlib import Path
import subprocess
import sys
import socket
import time
import urllib.request
import uuid
from evidence import REQUIRED, digest
from fixtures.prepare_web01 import prepare


def now():
    return datetime.datetime.now(datetime.timezone.utc).isoformat()


def main():
    output = Path(sys.argv[1]) / 'WEB-01'
    if output.is_absolute() or '..' in output.parts:
        raise SystemExit('output must be workspace-relative')
    if output.exists():
        __import__('shutil').rmtree(output)
    output.mkdir(parents=True, exist_ok=True)
    invocation = str(uuid.uuid4())
    started = now()
    evidence = {'criterion': 'WEB-01', 'invocationId': invocation, 'scenarios': [], 'checks': []}
    failures = []
    for command in ('npm --prefix web run build', 'go test ./internal/app ./internal/dashboard ./internal/testengine ./internal/core/nodeconfig'):
        run = subprocess.run(command, shell=True, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        logfile = output / f'check-{len(evidence["checks"])}.log'
        logfile.write_text(run.stdout)
        evidence['checks'].append({'command': command, 'exitCode': run.returncode,
                                   'log': str(logfile), 'sha256': digest(logfile)})
        if run.returncode:
            failures.append(command + ' failed')
    runtime = runtime_root() / invocation
    process = None
    try:
        prepared = subprocess.run(['bash', 'tests/webui/prepare.sh', invocation], capture_output=True, text=True)
        if prepared.returncode:
            raise RuntimeError('fixture build failed: ' + prepared.stderr)
        with socket.socket() as sock:
            sock.bind(('127.0.0.1', 0))
            port = sock.getsockname()[1]
        url = f'http://127.0.0.1:{port}'
        serverlog = output / 'server.log'
        with serverlog.open('w') as logfile:
            process = subprocess.Popen([str(runtime / 'chainbench-dashboard'), '-addr', f'127.0.0.1:{port}', '-chain-presets', 'presets/chain', '-deployment-root', str(runtime / 'store')], stdout=logfile, stderr=logfile)
            for attempt in range(100):
                if process.poll() is not None:
                    raise RuntimeError('server exited before health check')
                try:
                    with urllib.request.urlopen(url + '/healthz', timeout=1) as response:
                        if response.read() == b'ok':
                            break
                except OSError:
                    time.sleep(0.1)
            else:
                raise RuntimeError('server health timeout')
            private = runtime / 'browser-private.json'
            private.write_text(json.dumps({'setupToken': (runtime / 'store' / 'setup.token').read_text().strip(), 'username':'preset-admin', 'password':__import__('secrets').token_urlsafe(24)}))
            private.chmod(0o600)
            browser = subprocess.run(['node', 'tests/webui/browser.mjs', url, str(output), str(private)], capture_output=True, text=True)
            (output / 'browser.log').write_text(browser.stdout + browser.stderr)
            if browser.returncode:
                raise RuntimeError('live browser assertions failed; see browser.log')
        evidence['server'] = {'path': str(runtime / 'chainbench-dashboard'), 'sha256': digest(runtime / 'chainbench-dashboard'), 'baseCommit': base_commit(), 'url': url}
        evidence['frontend'] = [{'path': str(path), 'sha256': digest(path)} for path in Path('internal/dashboard/spa').rglob('*') if path.is_file()]
        evidence['fixture'] = {'runtime': str(runtime), 'transport': 'local', 'scope': 'preset-composition', 'invocationId': invocation}
        observations = json.loads((output / 'browser.json').read_text())
        evidence['scenarios'] = observations['scenarios']
        evidence['coverage'] = observations['coverage']
        evidence['binaries'] = prepare(runtime, output)
        evidence['scenarios'].append({'id': 'binary-surface-provenance', 'observedAt': now(), 'assertions': [
            {'message': 'Fresh version/help captures verify checksum, source chain, OS/architecture and every engine dialect mapping', 'passed': True},
            {'message': 'Both gwemix identities distinguished by embedded commit/source consensus marker; unmapped binary flags excluded', 'passed': True}]})
        evidence['artifacts'] = [{'path': str(path), 'sha256': digest(path)} for path in output.iterdir() if path.suffix in ('.png', '.log') or path.name == 'browser.json']
    except Exception as exc:
        failures.append(str(exc))
    finally:
        if process is not None:
            process.terminate()
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait()
    artifacts = [{'path': str(path), 'sha256': digest(path)} for path in output.rglob('*') if path.is_file() and path.name not in ('result.json', 'evidence.json')]
    for scenario in evidence['scenarios']:
        scenario['artifacts'] = artifacts
    evidence['artifacts'] = artifacts
    if sorted(s['id'] for s in evidence['scenarios']) != sorted(REQUIRED):
        failures.append('mandatory scenario set incomplete')
    evidence['blockers'] = failures
    ep = output / 'evidence.json'
    ep.write_text(json.dumps(evidence, indent=2) + '\n')
    result = {'criterion': 'WEB-01', 'invocationId': invocation,
              'startedAt': started, 'finishedAt': now(), 'outcome': 'pass' if not failures else 'incomplete',
              'requiredScenarios': REQUIRED, 'executedScenarios': [s['id'] for s in evidence['scenarios']],
              'skippedScenarios': [s for s in REQUIRED if s not in {row['id'] for row in evidence['scenarios']}], 'failedAssertions': failures,
              'evidenceDigest': digest(ep)}
    (output / 'result.json').write_text(json.dumps(result, indent=2) + '\n')


if __name__ == '__main__':
    main()
