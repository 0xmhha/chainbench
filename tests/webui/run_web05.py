"""Fresh WEB-05 proof: browser editing, corpus round trip and a Web test job
that executes every builtin argument path on native gstable."""
from runtime_contract import runtime_root, base_commit, source_digest, stage_sources

import datetime
import glob
import importlib.util
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
from evidence_web05 import REQUIRED, digest, verify
from web05_coverage import executed as executed_paths, coverage_rows, missing

COVERAGE_CASES = sorted(glob.glob('tests/tc/go-stablenet/vocabulary/1[1-6]-arguments-*.json'))
# The one builtin argument a single-binary network cannot reach: crossFork's
# timeout, on a network that crosses from gwemix to gwbft.
FORK_CASE = 'tests/tc/go-wemix/hardfork/02-state-written-before-the-fork-survives-it.json'
# The corpus attach cases: a Web attach job runs them against a network the
# service composed and recorded. A key file account signs with an account
# key credential the browser saves: node5's preset key, funded at genesis and
# an endpoint, so no block reward moves its balance during the transfer.
ATTACH_CASES = ['tests/tc/basic/08-attached-chain-produces.json', *sorted(glob.glob('tests/tc/go-stablenet/testnet/0*.json'))]
ATTACH_CREDENTIAL_CASES = ['testnet-value-transfer']
# Declared accounts on a composed network; this case's network is the attach target.
DECLARED_ACCOUNTS_CASE = 'tests/tc/go-stablenet/vocabulary/17-declared-accounts.json'


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


def free_band(base_low, span):
    """A port band nothing listens on, for the owned network."""
    for _ in range(200):
        base = base_low + secrets.randbelow(span) // 10 * 10
        probes = []
        try:
            for i in range(5):
                for port in range(base + 10 * i, base + 10 * i + 4):
                    sock = socket.socket(); probes.append(sock); sock.bind(('127.0.0.1', port))
            return base
        except OSError:
            continue
        finally:
            for sock in probes:
                sock.close()
    raise RuntimeError('no free port band')


def corpus_files(source):
    files, presets = [], []
    for p in (sorted(glob.glob(str(source / 'tests/tc/**/*.json'), recursive=True)) + sorted(glob.glob(str(source / 'examples/specs/*.json')))
              + sorted(glob.glob(str(source / 'presets/chain/*.json')))):
        kind = json.loads(Path(p).read_text()).get('kind')
        rel = str(Path(p).relative_to(source))
        (presets if kind == 'chain-preset' else files).append(rel)
    return files, presets


def scenario(id, messages):
    return {'id': id, 'mode': 'team', 'transport': 'local', 'ownership': 'owned', 'role': 'admin',
            'observedAt': now(), 'assertions': [{'message': m, 'passed': True} for m in messages]}


def stop_owned_nodes(runtime):
    # Only processes whose command line names this exclusively owned runtime.
    listing = subprocess.run(['ps', '-axo', 'pid=,command='], capture_output=True, text=True).stdout
    for line in listing.splitlines():
        pid, _, command = line.strip().partition(' ')
        if str(runtime) in command and any(name in command for name in ('gstable', 'gwemix', 'gwbft', '-node')):
            try:
                os.kill(int(pid), 15)
            except (ProcessLookupError, ValueError):
                pass


def main():
    out = Path(sys.argv[1]) / 'WEB-05'
    if out.is_absolute() or '..' in out.parts:
        raise SystemExit('output must be workspace-relative')
    clean_invocation_output(out)
    invocation = str(uuid.uuid4())
    started = now()
    before = source_digest(Path.cwd())
    runtime = runtime_root() / invocation
    runtime.mkdir(parents=True, mode=0o700)
    evidence = {'criterion': 'WEB-05', 'invocationId': invocation, 'scenarios': [], 'checks': [], 'artifacts': [], 'sourceDigest': before}
    failures, server, handles = [], None, []
    source = stage_sources(Path.cwd(), runtime)
    try:
        for command in ('node --test tests/webui/dsl-form.test.mjs',
                        'go test ./internal/app ./internal/testhelper ./internal/testengine ./internal/dashboard ./internal/dsl/...',
                        'npm --prefix web run build'):
            run = subprocess.run(command, shell=True, text=True, capture_output=True, cwd=source)
            logfile = out / ('check-' + str(len(evidence['checks'])) + '.log')
            logfile.write_text(run.stdout + run.stderr)
            evidence['checks'].append({'command': command, 'exitCode': run.returncode, 'path': str(logfile), 'sha256': digest(logfile)})
            if run.returncode:
                raise RuntimeError(command + ' failed')
        subprocess.run(['go', 'build', '-o', str(runtime / 'chainbench-dashboard'), './cmd/chainbench-dashboard'], cwd=source, check=True, capture_output=True)
        spec = importlib.util.spec_from_file_location('native_fixture', 'tests/webui/fixtures/prepare_web04.py')
        module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module)
        provenance = module.prepare(runtime, out)
        keys = runtime / 'source-keys'; shutil.copytree('presets/keys', keys)
        with socket.socket() as sock:
            sock.bind(('127.0.0.1', 0)); port = sock.getsockname()[1]
        url = 'http://127.0.0.1:' + str(port)
        store = runtime / 'store'
        log = (out / 'server.log').open('w'); handles.append(log)
        server = subprocess.Popen([str(runtime / 'chainbench-dashboard'), '-addr', '127.0.0.1:' + str(port), '-deployment-root', str(store),
                                   '-manifest-assets', str(runtime / 'assets.json'), '-manifest-keys', str(keys)], stdout=log, stderr=log)
        for _ in range(200):
            if server.poll() is not None:
                raise RuntimeError('dashboard exited; see server.log')
            try:
                urllib.request.urlopen(url + '/healthz', timeout=1).close(); break
            except OSError:
                time.sleep(.1)
        else:
            raise RuntimeError('dashboard did not listen')
        corpus, presets = corpus_files(source)
        fixture = {'url': url, 'setupToken': (store / 'setup.token').read_text().strip(), 'password': secrets.token_urlsafe(24),
                   'runtime': str(runtime), 'source': str(source), 'corpusFiles': corpus, 'presetFiles': presets,
                   'coverageFiles': COVERAGE_CASES, 'p2pBase': free_band(40000, 3000), 'rpcBase': free_band(20000, 3000),
                   'forkCase': FORK_CASE, 'forkPreset': 'presets/chain/wemix-to-wbft.json',
                   'forkP2PBase': free_band(43000, 3000), 'forkRPCBase': free_band(23000, 3000),
                   'attachFiles': ATTACH_CASES, 'attachCredentialCases': ATTACH_CREDENTIAL_CASES, 'payerKeyFile': str(source / 'presets/keys/node5/private'), 'declaredAccountsCase': DECLARED_ACCOUNTS_CASE,
                   'attachP2PBase': free_band(46000, 3000), 'attachRPCBase': free_band(26000, 3000)}
        private = runtime / 'browser-fixture.json'; private.write_text(json.dumps(fixture)); private.chmod(0o600)
        browser = run_browser(['node', str(source / 'tests/webui/browser_web05.mjs'), str(private), str(out.resolve())], timeout=3600)
        (out / 'browser.log').write_text(browser.stdout + browser.stderr)
        if browser.returncode:
            raise RuntimeError('browser assertions failed; see browser.log')
        observed = json.loads((out / 'browser.json').read_text())
        raw = json.loads((out / 'contract.json').read_text())
        contract = {'vocabulary': raw['vocabulary'], 'contract': raw['contract']}
        # The job's own sessions, by the references it reported.
        refs = [r.removeprefix('web:') for r in observed['job'].get('runIds') or []]
        if not refs:
            raise RuntimeError('the Web test job reported no engine session')
        sessions = [store / 'sessions' / ref for ref in refs]
        covered, tests = executed_paths(sessions, contract)
        edited = {tuple(k.split(':', 1)): set(v) for k, v in json.loads((out / 'edited.json').read_text()).items()}
        rows = coverage_rows(contract, covered, edited)
        (out / 'coverage.json').write_text(json.dumps(rows, indent=2))
        (out / 'sessions.json').write_text(json.dumps(tests, indent=2))
        for ref, path in zip(refs, sessions):
            shutil.copytree(path, out / 'sessions' / ref, dirs_exist_ok=True)
        complete = [t for t in tests if t['complete']]
        expected = len(COVERAGE_CASES) + 1
        if len(complete) != expected or len(tests) != expected:
            failures.append(f'{len(complete)} of {expected} coverage cases passed completely in the Web jobs')
        # The attach job's own sessions: every corpus attach case, each passing
        # completely against the recorded network.
        attach_refs = [r.removeprefix('web:') for r in observed.get('attachJob', {}).get('runIds') or []]
        _, attached = executed_paths([store / 'sessions' / ref for ref in attach_refs], contract)
        for ref in attach_refs:
            shutil.copytree(store / 'sessions' / ref, out / 'sessions' / ref, dirs_exist_ok=True)
        runnable = len(ATTACH_CASES)
        if len(attached) != runnable or not all(t['complete'] for t in attached):
            failures.append(f'{sum(t["complete"] for t in attached)} of {runnable} attach cases passed completely in the Web attach job')
        (out / 'attach-sessions.json').write_text(json.dumps(attached, indent=2))
        target_refs = [r.removeprefix('web:') for r in observed.get('attachJob', {}).get('targetRunIds') or []]
        _, declared = executed_paths([store / 'sessions' / ref for ref in target_refs], contract)
        for ref in target_refs:
            shutil.copytree(store / 'sessions' / ref, out / 'sessions' / ref, dirs_exist_ok=True)
        if len(declared) != 1 or not declared[0]['complete']:
            failures.append('the declared-accounts case did not pass completely in its Web test job')
        (out / 'declared-accounts-session.json').write_text(json.dumps(declared, indent=2))
        gaps = missing(rows)
        unedited = [r['kind'] + ':' + r['name'] for r in rows if not r['edited']]
        if unedited:
            failures.append('argument fields not edited in the browser: ' + ', '.join(unedited))
        if gaps:
            failures.append('argument paths not executed by a passing Web job: ' + json.dumps(gaps))
        by_id = {s['id']: s for s in observed['scenarios']}
        scenarios = [by_id[k] for k in ('grammar', 'references', 'v1-migration', 'invalid-unknown', 'live-execution', 'attach-execution') if k in by_id]
        for kind, label in (('action', 'actions'), ('assertion', 'assertions'), ('reader', 'readers')):
            mine = [r for r in rows if r['kind'] == kind]
            if all(r['executed'] and r['edited'] for r in mine):
                scenarios.append(scenario(label, [f'{len(mine)} {label} edited in the browser and executed by the Web job with every argument path']))
        total = sum(len(r['argumentPaths']) for r in rows)
        ran = sum(len(r['executedArgumentPaths']) for r in rows)
        if not gaps and not unedited:
            scenarios.append(scenario('arguments', [f'{ran} of {total} argument paths edited and executed', *[a['message'] for k in ('arguments-edited', 'round-trip') for a in by_id.get(k, {}).get('assertions', [])]]))
        evidence['scenarios'] = scenarios
        evidence['server'] = {'sha256': digest(runtime / 'chainbench-dashboard'), 'baseCommit': base_commit(), 'contractVersion': '2', 'url': url}
        shutil.copytree(source / 'internal/dashboard/spa', out / 'frontend')
        evidence['frontend'] = [{'path': str(p), 'sha256': digest(p)} for p in (out / 'frontend').rglob('*') if p.is_file()]
        evidence['fixture'] = {'runtime': str(runtime), 'transport': 'local', 'ownership': 'owned', 'browserVersion': observed['browserVersion']}
        stable = next(p for p in provenance if p['chain'] == 'stablenet')
        evidence['live'] = {'mockTargets': 0, 'skips': sum(1 for t in tests if t['result'] == 'skip'), 'failures': sum(1 for t in tests if not t['complete']),
                            'job': observed['job'], 'savedCases': observed['savedCases'], 'binary': {'sha256': stable['sha256'], 'chain': 'stablenet', 'version': stable['version'], 'commit': stable['commit'], 'helpDigest': stable['helpDigest']},
                            'targetFingerprint': digest(out / 'job.json'), 'executedArgumentPaths': ran, 'argumentPaths': total, 'corpusImported': observed['corpusImported']}
        if source_digest(Path.cwd()) != before:
            failures.append('source changed during the proof')
    except Exception as exc:
        failures.append(str(exc))
    finally:
        stop_owned_nodes(runtime)
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
    executed_ids = [s['id'] for s in evidence['scenarios']]
    (out / 'result.json').write_text(json.dumps({'criterion': 'WEB-05', 'invocationId': invocation, 'startedAt': started, 'finishedAt': now(),
                                                 'outcome': 'incomplete' if failures else 'pass', 'requiredScenarios': REQUIRED, 'executedScenarios': executed_ids,
                                                 'skippedScenarios': [s for s in REQUIRED if s not in executed_ids], 'failedAssertions': failures, 'evidenceDigest': digest(ep)}, indent=2) + '\n')
    if failures:
        for failure in failures:
            print('WEB-05 incomplete: ' + failure, file=sys.stderr)
        raise SystemExit(1)
    verify(out)


if __name__ == '__main__':
    main()
