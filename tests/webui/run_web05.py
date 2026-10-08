"""Fresh WEB-05 browser + owned local engine fixture with fail-closed coverage."""
from runtime_contract import runtime_root, base_commit
from runtime_contract import stage_sources

import base64
import datetime
import glob
import hashlib
import json
import os
import platform
import secrets
import shutil
import socket
import subprocess
import sys
import time
import urllib.request
import uuid
from pathlib import Path
from evidence_web05 import REQUIRED, digest, verify

def now():
    return datetime.datetime.now(datetime.timezone.utc).isoformat()

def clean_invocation_output(out):
    """A receipt must never merge session files from an earlier invocation."""
    out.mkdir(parents=True, exist_ok=True)
    for p in out.iterdir():
        if p.is_dir() and not p.is_symlink():
            shutil.rmtree(p)
        else:
            p.unlink()

def main():
    out = Path(sys.argv[1]) / 'WEB-05'
    if out.is_absolute() or '..' in out.parts:
        raise SystemExit('output must be workspace-relative')
    clean_invocation_output(out)
    invocation = str(uuid.uuid4())
    started = now()
    commit = base_commit()
    runtime = runtime_root() / invocation
    runtime.mkdir(parents=True, mode=0o700)
    evidence = {'criterion': 'WEB-05', 'invocationId': invocation, 'scenarios': [], 'checks': [], 'artifacts': []}
    failures = []
    source = stage_sources(Path.cwd(), runtime)
    server = None
    handles = []
    try:
        for command in ('node --test tests/webui/dsl-form.test.mjs', 'go test ./internal/app ./internal/testhelper ./internal/dashboard ./internal/dsl', 'npm --prefix web run build'):
            run = subprocess.run(command, shell=True, text=True, capture_output=True, cwd=source)
            logfile = out / ('check-' + str(len(evidence['checks'])) + '.log')
            logfile.write_text(run.stdout + run.stderr)
            evidence['checks'].append({'command': command, 'exitCode': run.returncode, 'path': str(logfile), 'sha256': digest(logfile)})
            if run.returncode: raise RuntimeError(command + ' failed')
        subprocess.run(['bash', 'tests/webui/prepare.sh', invocation], check=True, capture_output=True, cwd=source)
        build = runtime / 'build'; build.mkdir()
        (build / 'main.go').write_bytes(Path('tests/webui/fixtures/dslrun.go.txt').read_bytes())
        (build / 'go.mod').write_text('module github.com/0xmhha/chainbench/webui-fixture\n\ngo 1.26.8\n\nrequire github.com/0xmhha/chainbench v0.0.0\nreplace github.com/0xmhha/chainbench => ' + str(source) + '\n')
        subprocess.run(['go', 'build', '-mod=mod', '-o', str(runtime / 'dslrun'), '.'], cwd=build, check=True, capture_output=True)
        account = {'id': 'dsl-operator', 'username': 'dsl-operator', 'role': 'operator', 'password': secrets.token_urlsafe(24)}
        (runtime / 'accounts.go').write_bytes(Path('tests/webui/fixtures/accounts.go.txt').read_bytes())
        provision = subprocess.run(['go', 'run', str(runtime / 'accounts.go')], input=json.dumps([account]), text=True, capture_output=True, check=True, cwd=source)
        (runtime / 'accounts.json').write_text(provision.stdout)
        (runtime / 'accounts.json').chmod(0o600)
        (runtime / 'browser-fixture.json').write_text(json.dumps({'account': account}))
        (runtime / 'browser-fixture.json').chmod(0o600)
        with socket.socket() as sock:
            sock.bind(('127.0.0.1', 0)); port = sock.getsockname()[1]
        url = 'http://127.0.0.1:' + str(port)
        log = (out / 'server.log').open('w'); handles.append(log)
        server = subprocess.Popen([str(runtime / 'chainbench-dashboard'), '-addr', '127.0.0.1:' + str(port), '-deployment-accounts', str(runtime / 'accounts.json'), '-deployment-root', str(runtime / 'store')], stdout=log, stderr=log)
        for attempt in range(100):
            if server.poll() is not None: raise RuntimeError('server exited')
            try:
                urllib.request.urlopen(url + '/healthz', timeout=1).close(); break
            except OSError: time.sleep(.1)
        else: raise RuntimeError('server health timeout')
        browser = subprocess.run(['node', str(source / 'tests/webui/browser_web05.mjs'), url, str(out.resolve()), str(runtime / 'browser-fixture.json')], text=True, capture_output=True)
        (out / 'browser.log').write_text(browser.stdout + browser.stderr)
        if browser.returncode: raise RuntimeError('browser assertions failed; see browser.log')
        observed = json.loads((out / 'browser.json').read_text())
        evidence['scenarios'] = observed['scenarios']
        evidence['server'] = {'sha256': digest(runtime / 'chainbench-dashboard'), 'baseCommit': commit, 'contractVersion': '2', 'url': url}
        shutil.copytree(source / 'internal/dashboard/spa', out / 'frontend')
        evidence['frontend'] = [{'path': str(p), 'sha256': digest(p)} for p in (out / 'frontend').rglob('*') if p.is_file()]
        evidence['fixture'] = {'runtime': str(runtime), 'transport': 'local', 'ownership': 'owned', 'browserVersion': observed['browserVersion']}
        sources = glob.glob('/Users/0xtopaz/work/github/0xmhha/chain/go-stablenet/build/bin/gstable')
        if len(sources) != 1: raise RuntimeError('native stablenet binary unavailable')
        binary_source = Path(sources[0]); binary = runtime / 'stablenet-node'; shutil.copy2(binary_source, binary)
        version = subprocess.check_output([str(binary), 'version'], text=True)
        help_text = subprocess.check_output([str(binary), '--help'], text=True)
        import re
        architecture = {'aarch64': 'arm64', 'x86_64': 'amd64'}.get(platform.machine(), platform.machine())
        operating_system = platform.system().lower()
        if ('Architecture: ' + architecture) not in version or ('Operating System: ' + operating_system) not in version:
            raise RuntimeError('native fixture binary platform mismatch')
        commit = re.search(r'Git Commit: ([0-9a-f]{40})', version).group(1)
        config = subprocess.check_output(['git', '-C', str(binary_source.parents[2]), 'show', commit + ':params/config.go'], text=True)
        if 'Anzeon' not in config: raise RuntimeError('selected binary source is not stablenet')
        (out / 'binary-version.txt').write_text(version); (out / 'binary-help.txt').write_text(help_text)
        # Independent port bands avoid the engine unit tests' default bands.
        for attempt in range(100):
            rpc_base = 20000 + secrets.randbelow(6000)
            p2p_base = 40000 + secrets.randbelow(6000)
            reserved = []
            try:
                for i in range(4):
                    for port in [p2p_base + 10 * i] + [rpc_base + 10 * i + j for j in range(4)]:
                        for kind in (socket.SOCK_STREAM, socket.SOCK_DGRAM):
                            probe = socket.socket(socket.AF_INET, kind); reserved.append(probe)
                            probe.bind(('127.0.0.1', port))
                break
            except OSError:
                if attempt == 99: raise RuntimeError('independent fixture port bands unavailable')
            finally:
                for probe in reserved: probe.close()
        server_set = {'version': 2, 'pool': {'hosts': [{'name': 'fixture', 'addr': '127.0.0.1'}], 'slots': 4, 'ports': {'p2p': {'base': p2p_base, 'step': 10}, 'rpc': {'base': rpc_base, 'step': 10}}}}
        (runtime / 'server-set.json').write_text(json.dumps(server_set))
        (out / 'fixture-target.json').write_text(json.dumps(server_set, indent=2))
        live = subprocess.run([str(runtime / 'dslrun'), str(runtime), str(binary), str((out / 'edited.json').resolve()), str((out / 'migrated.json').resolve()), str((out / 'legacy.json').resolve())], text=True, capture_output=True, timeout=420, cwd=source)
        (out / 'live.log').write_text(live.stderr)
        (out / 'live.json').write_text(live.stdout)
        if live.returncode: raise RuntimeError('exported DSL live execution failed; see live.log')
        result = json.loads(live.stdout)
        summary = result['Summary']['summary']
        if summary != {'pass': 3, 'fail': 0, 'blocked': 0, 'skip': 0}: raise RuntimeError('live session did not run all three imports')
        session = Path(result['SessionRoot'])
        shutil.copytree(session, out / 'session', dirs_exist_ok=True)
        target = {'serverSet': server_set, 'endpoints': result['Endpoints'], 'dataPath': str((runtime / 'network').resolve())}
        evidence['live'] = {'mockTargets': 0, 'skips': 0, 'failures': 0, 'summary': summary, 'sessionId': session.name, 'binary': {'sha256': digest(binary), 'chain': 'stablenet', 'os': operating_system, 'architecture': architecture, 'version': version, 'commit': commit, 'helpDigest': digest(out / 'binary-help.txt')}, 'targetFingerprint': hashlib.sha256(json.dumps(target, sort_keys=True).encode()).hexdigest(), 'endpoints': result['Endpoints'], 'stopped': result['Stopped']}
        evidence['fixture']['targetManifest'] = {'path': str(out / 'fixture-target.json'), 'sha256': digest(out / 'fixture-target.json')}
        evidence['partialLiveObservations'] = ['read:blockNumber', 'expect:blockNumber', 'expect:chainId', 'v1-original', 'v1-migrated']
        # Full WEB-05 requires every registration AND every argument to reach
        # actual execution. A representative live suite never satisfies that.
        coverage = json.loads((out / 'coverage.json').read_text())
        for item in coverage:
            if (item['kind'], item['name']) in [('action', 'read'), ('reader', 'blockNumber'), ('assertion', 'blockNumber'), ('assertion', 'chainId')]:
                item['roundTrip'] = item['executed'] = True
        (out / 'coverage.json').write_text(json.dumps(coverage, indent=2))
        missing = [item['kind'] + ':' + item['name'] for item in coverage if not item['executed'] or item['argumentPaths'] != item['executedArgumentPaths']]
        if missing: failures.append('full argument/registration live execution coverage missing: ' + ', '.join(missing))
    except Exception as exc:
        failures.append(str(exc))
    finally:
        if server is not None:
            server.terminate()
            try: server.wait(timeout=10)
            except subprocess.TimeoutExpired: server.kill(); server.wait()
        for handle in handles: handle.close()
        for name in ('browser-fixture.json', 'accounts.json'): (runtime / name).unlink(missing_ok=True)
    evidence['blockers'] = failures
    evidence['artifacts'] = [{'path': str(p), 'sha256': digest(p)} for p in sorted(out.rglob('*')) if p.is_file()]
    ep = out / 'evidence.json'; ep.write_text(json.dumps(evidence, indent=2) + '\n')
    executed = [s['id'] for s in evidence['scenarios']]
    (out / 'result.json').write_text(json.dumps({'criterion': 'WEB-05', 'invocationId': invocation, 'startedAt': started, 'finishedAt': now(), 'outcome': 'incomplete' if failures else 'pass', 'requiredScenarios': REQUIRED, 'executedScenarios': executed, 'skippedScenarios': [s for s in REQUIRED if s not in executed], 'failedAssertions': failures, 'evidenceDigest': digest(ep)}, indent=2) + '\n')
    if failures:
        for failure in failures:
            print('WEB-05 incomplete: ' + failure, file=sys.stderr)
        raise SystemExit(1)
    verify(out)

if __name__ == '__main__':
    main()
