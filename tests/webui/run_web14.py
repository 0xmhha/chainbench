"""Fresh WEB-14 synthesis: the published WEB-01..13 results on this frozen
source, a reproducible SPA build served by the real Go server, every route
navigated and refreshed in a browser, and the mandatory regression checks with
every gated live Go test opened."""
from runtime_contract import runtime_root, base_commit, source_digest, stage_sources

import datetime
import filecmp
import hashlib
import json
import os
from pathlib import Path
import secrets
import shutil
import socket
import subprocess
import sys
import time
import urllib.request
import uuid
from browser_process import run_browser
from evidence_web14 import REQUIRED, CRITERIA, CHECKS, RACE, LIVE_ENV, acceptance_root, coverage, digest, go_tests, verify, verify_criterion


def now():
    return datetime.datetime.now(datetime.timezone.utc).isoformat()


def scenario(id, messages, mode='team', transport='local'):
    return {'id': id, 'mode': mode, 'transport': transport, 'ownership': 'owned', 'role': 'admin',
            'observedAt': now(), 'assertions': [{'message': m, 'passed': True} for m in messages]}


def clean(out):
    out.mkdir(parents=True, exist_ok=True)
    for p in out.iterdir():
        shutil.rmtree(p) if p.is_dir() and not p.is_symlink() else p.unlink()


def checkout_of(acceptance):
    """The git checkout the criteria were published from. Regression checks run
    there because some read the tracked tree, which a staged copy has not."""
    top = subprocess.run(['git', '-C', str(acceptance if acceptance.is_dir() else acceptance.parent), 'rev-parse', '--show-toplevel'],
                         text=True, capture_output=True)
    if top.returncode:
        raise RuntimeError('no git checkout holds ' + str(acceptance))
    return Path(top.stdout.strip())


def identical(left, right):
    compare = filecmp.dircmp(left, right)
    stack = [compare]
    while stack:
        c = stack.pop()
        if c.left_only or c.right_only or c.diff_files or c.funny_files:
            return False
        stack.extend(c.subdirs.values())
    return True


def run_checks(out, checkout, staged, evidence):
    missing = [name for name in LIVE_ENV if not os.environ.get(name)]
    if missing:
        raise RuntimeError('gated live Go tests would skip; set ' + ', '.join(missing))
    prepare = subprocess.run(['bash', 'tests/webui/fixtures/docker_regression_prepare.sh'], cwd=checkout, text=True, capture_output=True)
    (out / 'docker-prepare.log').write_text(prepare.stdout + prepare.stderr)
    if prepare.returncode:
        raise RuntimeError('docker regression fixture failed; see docker-prepare.log')
    # Only the serial race run opens the live gates. Elsewhere packages test in
    # parallel, and a live network on default ports would collide with the
    # port checks of another package's unit tests.
    plain = {k: v for k, v in os.environ.items() if k not in LIVE_ENV}
    for command in CHECKS:
        # The web build writes the embedded SPA, so it runs in the staged copy.
        cwd = staged if command.startswith('npm') else checkout
        run = subprocess.run(command, shell=True, text=True, capture_output=True, cwd=cwd,
                             env=os.environ if command == RACE else plain)
        log = out / ('check-' + str(len(evidence['checks'])) + '.log')
        log.write_text(run.stdout + run.stderr)
        evidence['checks'].append({'command': command, 'exitCode': run.returncode, 'cwd': 'staged' if cwd == staged else 'checkout',
                                   'path': str(log), 'sha256': digest(log)})
        if run.returncode:
            raise RuntimeError(command + ' failed; see ' + str(log))
    evidence['scenarios'].append(scenario('mandatory-checks', [c['command'] + ' exit 0' for c in evidence['checks']]))
    race = next(c for c in evidence['checks'] if c['command'] == RACE)
    counts, skipped = go_tests(Path(race['path']).read_text().splitlines())
    evidence['goTests'] = {'counts': counts, 'skipped': skipped, 'env': LIVE_ENV}
    if skipped or counts['fail'] or not counts['pass']:
        raise RuntimeError('race run: %s; skipped: %s' % (counts, ', '.join(skipped) or 'none'))
    evidence['scenarios'].append(scenario('live-go-tests', [
        '%d Go tests passed under -race, none skipped and none failed' % counts['pass'],
        'gated live tests ran against real binaries and the docker servers: ' + ', '.join(LIVE_ENV)], transport='local+SSH'))


def serve(out, runtime, staged, handles):
    subprocess.run(['go', 'build', '-o', str(runtime / 'chainbench-dashboard'), './cmd/chainbench-dashboard'], cwd=staged, check=True, capture_output=True)
    with socket.socket() as sock:
        sock.bind(('127.0.0.1', 0)); port = sock.getsockname()[1]
    url = 'http://127.0.0.1:' + str(port)
    store = runtime / 'store'
    log = (out / 'server.log').open('w'); handles.append(log)
    server = subprocess.Popen([str(runtime / 'chainbench-dashboard'), '-addr', '127.0.0.1:' + str(port), '-deployment-root', str(store)], stdout=log, stderr=log)
    for _ in range(200):
        if server.poll() is not None:
            raise RuntimeError('dashboard exited; see server.log')
        try:
            urllib.request.urlopen(url + '/healthz', timeout=1).close()
            return server, url, store
        except OSError:
            time.sleep(.1)
    server.terminate()
    raise RuntimeError('dashboard did not listen')


def main():
    output = Path(sys.argv[1])
    out = output / 'WEB-14'
    if out.is_absolute() or '..' in out.parts:
        raise SystemExit('output must be workspace-relative')
    clean(out)
    # A fresh reproduction runs in a staged copy without chainbench-out; the
    # criteria it synthesizes are read, never written, where they were published.
    acceptance = acceptance_root(out).resolve()
    invocation = str(uuid.uuid4())
    started = now()
    before = source_digest(Path.cwd())
    runtime = runtime_root() / invocation
    runtime.mkdir(parents=True, mode=0o700)
    evidence = {'criterion': 'WEB-14', 'invocationId': invocation, 'sourceDigest': before, 'scenarios': [], 'checks': [], 'artifacts': []}
    failures, server, handles = [], None, []
    staged = stage_sources(Path.cwd(), runtime)
    try:
        # The thirteen criteria, judged by their own verifiers on this source.
        report, evidences = [], []
        for criterion in CRITERIA:
            directory = acceptance / criterion
            verify_criterion(directory)
            published = json.loads((directory / 'evidence.json').read_text())
            result = json.loads((directory / 'result.json').read_text())
            if published.get('sourceDigest') != before:
                raise RuntimeError(criterion + ' was captured from another source; recapture it on this one')
            report.append({'criterion': criterion, 'invocationId': result['invocationId'], 'outcome': result['outcome'],
                           'scenarios': result['executedScenarios'], 'evidenceDigest': result['evidenceDigest']})
            evidences.append(published)
        evidence['criteria'] = report
        evidence['scenarios'].append(scenario('criteria', [f'{c["criterion"]} {c["outcome"]}: {len(c["scenarios"])} scenarios' for c in report]))
        modes, transports = coverage(evidences)
        if not ({'personal', 'team'} <= modes and {'local', 'ssh'} <= transports):
            raise RuntimeError(f'modes {sorted(modes)} and transports {sorted(transports)} miss personal/team or local/SSH')
        evidence['scenarios'].append(scenario('modes-transports', [f'modes observed: {", ".join(sorted(modes))}', f'transports observed: {", ".join(sorted(transports))}']))

        checkout = checkout_of(acceptance)
        if source_digest(checkout) != before:
            raise RuntimeError(str(checkout) + ' holds another source than this proof')
        evidence['checkout'] = {'baseCommit': base_commit(), 'sourceDigest': before}
        run_checks(out, checkout, staged, evidence)

        # The fresh build must reproduce the embedded SPA byte for byte.
        embedded, rebuilt = Path('internal/dashboard/spa'), staged / 'internal/dashboard/spa'
        files = {str(p.relative_to(embedded)): digest(p) for p in sorted(embedded.rglob('*')) if p.is_file()}
        spa = {'files': files, 'reproducible': identical(embedded, rebuilt)}
        if not spa['reproducible']:
            raise RuntimeError('a fresh web build differs from the embedded SPA')

        server, url, store = serve(out, runtime, staged, handles)
        served = {}
        for name in files:
            with urllib.request.urlopen(url + '/' + (name if name != 'index.html' else ''), timeout=5) as response:
                served[name] = hashlib.sha256(response.read()).hexdigest()
        spa['servedMatchesEmbedded'] = served == files
        if not spa['servedMatchesEmbedded']:
            raise RuntimeError('the server serves different SPA bytes than are embedded')
        evidence['spa'] = spa
        evidence['scenarios'].append(scenario('spa-build', [f'{len(files)} SPA files rebuilt byte-identically', 'the Go server serves exactly the embedded files']))

        fixture = {'url': url, 'setupToken': (store / 'setup.token').read_text().strip(), 'password': secrets.token_urlsafe(24)}
        private = runtime / 'browser-fixture.json'; private.write_text(json.dumps(fixture)); private.chmod(0o600)
        browser = run_browser(['node', str(staged / 'tests/webui/browser_web14.mjs'), str(private), str(out.resolve())], timeout=600)
        (out / 'browser.log').write_text(browser.stdout + browser.stderr)
        if browser.returncode:
            raise RuntimeError('route assertions failed; see browser.log')
        routes = json.loads((out / 'browser.json').read_text())
        evidence['scenarios'].append(scenario('routes', [f'{r["path"]}: navigated, refreshed and opened directly' for r in routes['routes']]))
        evidence['server'] = {'sha256': digest(runtime / 'chainbench-dashboard'), 'baseCommit': base_commit(), 'contractVersion': '2', 'url': url}
        evidence['fixture'] = {'runtime': str(runtime), 'transport': 'local', 'ownership': 'owned', 'browserVersion': routes['browserVersion']}
        (out / 'report.json').write_text(json.dumps({'sourceDigest': before, 'criteria': report, 'modes': sorted(modes), 'transports': sorted(transports),
                                                     'checks': evidence['checks'], 'goTests': evidence['goTests'], 'spa': spa}, indent=2))
        if source_digest(Path.cwd()) != before or source_digest(checkout) != before:
            failures.append('source changed during the proof')
    except Exception as exc:
        failures.append(str(exc))
    finally:
        if server is not None:
            server.terminate()
            try:
                server.wait(timeout=10)
            except subprocess.TimeoutExpired:
                server.kill(); server.wait()
        for handle in handles:
            handle.close()
        (runtime / 'browser-fixture.json').unlink(missing_ok=True)
    evidence['blockers'] = failures
    evidence['artifacts'] = [{'path': str(p), 'sha256': digest(p)} for p in sorted(out.rglob('*')) if p.is_file()]
    ep = out / 'evidence.json'; ep.write_text(json.dumps(evidence, indent=2) + '\n')
    executed = [s['id'] for s in evidence['scenarios']]
    (out / 'result.json').write_text(json.dumps({'criterion': 'WEB-14', 'invocationId': invocation, 'startedAt': started, 'finishedAt': now(),
                                                 'outcome': 'incomplete' if failures else 'pass', 'requiredScenarios': REQUIRED, 'executedScenarios': executed,
                                                 'skippedScenarios': [s for s in REQUIRED if s not in executed], 'failedAssertions': failures,
                                                 'evidenceDigest': digest(ep)}, indent=2) + '\n')
    if failures:
        for failure in failures:
            print('WEB-14 incomplete: ' + failure, file=sys.stderr)
        raise SystemExit(1)
    verify(out, acceptance)


if __name__ == '__main__':
    main()
