"""Shared acceptance runner and verifier for criteria assembled from native proofs.

A criterion lists development proofs (each a real browser + native binary run
that writes a source-bound receipt). The runner reruns every proof inside one
invocation, copies the cited observation files, maps receipts to the required
scenarios and publishes result.json and evidence.json. The verifier checks
identity, coverage, artifact digests, invocation timing and that every proof
used the same dashboard build in its own runtime."""
import datetime
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import uuid
from evidence import digest
from runtime_contract import base_commit, runtime_root, source_digest


def now():
    return datetime.datetime.now(datetime.timezone.utc).isoformat()


def assertion(description, passed):
    return {'description': description, 'passed': bool(passed)}


def run(criterion, proofs, required, checks, scenarios, source_patterns, verify):
    output = Path(sys.argv[1]) / criterion
    if output.is_absolute() or '..' in output.parts:
        raise SystemExit('output must be workspace-relative')
    output.mkdir(parents=True, exist_ok=True)
    for p in output.iterdir():  # A failed invocation must not keep older observations.
        if p.is_file():
            p.unlink()
    invocation = str(uuid.uuid4())
    started = now()
    evidence = {'criterion': criterion, 'invocationId': invocation, 'scenarios': [], 'checks': [], 'proofs': []}
    failures = []
    try:
        for command in checks:
            result = subprocess.run(command, shell=True, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
            log = output / f'check-{len(evidence["checks"])}.log'
            log.write_text(result.stdout)
            evidence['checks'].append({'command': command, 'exitCode': result.returncode, 'log': str(log), 'sha256': digest(log)})
            if result.returncode:
                raise RuntimeError(command + ' failed')
        source = source_digest(Path.cwd())
        receipts = {}
        for name, (script, files) in proofs.items():
            proof_started = now()
            result = subprocess.run([sys.executable, 'tests/webui/' + script], text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, env=dict(os.environ, PYTHONDONTWRITEBYTECODE='1'))
            log = output / f'{name}-run.log'
            log.write_text(result.stdout)
            if result.returncode:
                raise RuntimeError(f'{name} native proof failed; see {log}')
            development = Path('chainbench-out/web-ui-development') / name
            receipt = json.loads((development / 'receipt.json').read_text())
            if receipt.get('sourceDigest') != source or receipt.get('seedAcceptanceAwarded') is not False:
                raise RuntimeError(f'{name} receipt is not bound to this source')
            receipts[name] = receipt
            for file in ['receipt.json'] + files:
                shutil.copy2(development / file, output / f'{name}-{file}')
            evidence['proofs'].append({'name': name, 'startedAt': proof_started, 'runtime': receipt['runtime'], 'dashboardSHA256': receipt['dashboardSHA256'], 'browserVersion': receipt.get('browserVersion'), 'binaries': receipt.get('binaries')})
        observed = now()
        evidence['scenarios'] = scenarios(receipts, output)
        for scenario in evidence['scenarios']:
            scenario['invocationId'] = invocation
            scenario['observedAt'] = observed
            scenario['artifacts'] = [{'path': str(output / p), 'sha256': digest(output / p)} for p in scenario['artifacts']]
            if not all(a['passed'] for a in scenario['assertions']):
                failures.append('scenario failed: ' + scenario['id'])
        first = next(iter(receipts.values()))
        evidence['server'] = {'sha256': first['dashboardSHA256'], 'baseCommit': base_commit(), 'contractVersion': '2', 'goVersion': subprocess.check_output(['go', 'version'], text=True).strip()}
        evidence['frontend'] = [{'path': str(p), 'sha256': digest(p)} for p in sorted(Path('internal/dashboard/spa').rglob('*')) if p.is_file()]
        evidence['coverage'] = {'required': required, 'executed': [s['id'] for s in evidence['scenarios']], 'missing': [], 'mockTargets': 0}
        evidence['source'] = [{'path': str(p), 'sha256': digest(p)} for pattern in source_patterns for p in sorted(Path('.').glob(pattern)) if p.is_file()]
        evidence['artifacts'] = [{'path': str(p), 'sha256': digest(p)} for p in sorted(output.iterdir()) if p.is_file() and p.name not in ('result.json', 'evidence.json')]
    except Exception as exc:
        failures.append(str(exc))
    evidence['blockers'] = failures
    ep = output / 'evidence.json'
    ep.write_text(json.dumps(evidence, indent=2) + '\n')
    executed = [s['id'] for s in evidence['scenarios']]
    result = {'criterion': criterion, 'invocationId': invocation, 'startedAt': started, 'finishedAt': now(),
              'outcome': 'pass' if not failures and sorted(executed) == sorted(required) else 'incomplete',
              'requiredScenarios': required, 'executedScenarios': executed,
              'skippedScenarios': [s for s in required if s not in executed], 'failedAssertions': failures,
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


def verify_proofs(directory, criterion, required, proofs):
    directory = Path(directory)
    r = json.loads((directory / 'result.json').read_text())
    e = json.loads((directory / 'evidence.json').read_text())
    errors = []
    if r['criterion'] != criterion or e['criterion'] != criterion or r['invocationId'] != e['invocationId']:
        errors.append('identity mismatch')
    if r['outcome'] != 'pass' or r['failedAssertions'] or r['skippedScenarios'] or e.get('blockers'):
        errors.append('incomplete acceptance')
    if r['requiredScenarios'] != required or sorted(r['executedScenarios']) != sorted(required):
        errors.append('coverage mismatch')
    if r['evidenceDigest'] != digest(directory / 'evidence.json'):
        errors.append('evidence digest mismatch')
    coverage = e.get('coverage', {})
    if coverage.get('required') != required or sorted(coverage.get('executed', [])) != sorted(required) or coverage.get('missing') or coverage.get('mockTargets') != 0:
        errors.append('live coverage denominator mismatch')
    start = datetime.datetime.fromisoformat(r['startedAt'])
    finish = datetime.datetime.fromisoformat(r['finishedAt'])
    artifacts = e.get('artifacts', [])
    for s in e.get('scenarios', []):
        if not s.get('artifacts') or any(a not in artifacts for a in s['artifacts']):
            errors.append('scenario artifacts unverified')
        if not start <= datetime.datetime.fromisoformat(s['observedAt']) <= finish:
            errors.append('stale observation')
        if s.get('invocationId') != r['invocationId'] or not s.get('assertions') or not all(a['passed'] is True for a in s['assertions']):
            errors.append('invalid observations: ' + s.get('id', '?'))
    for a in artifacts + e.get('source', []) + e.get('frontend', []):
        p = Path(a['path'])
        if p.is_absolute() or '..' in p.parts or not p.is_file() or digest(p) != a['sha256']:
            errors.append('artifact changed')
    executed = e.get('proofs', [])
    if sorted(p['name'] for p in executed) != sorted(proofs):
        errors.append('native proofs missing')
    servers = {p.get('dashboardSHA256') for p in executed}
    if len(servers) != 1 or e.get('server', {}).get('sha256') not in servers:
        errors.append('proofs used different dashboard builds')
    for p in executed:
        runtime = Path(p['runtime'])
        if runtime.parent != runtime_root() or not (runtime / 'dashboard').is_file() or digest(runtime / 'dashboard') != p['dashboardSHA256']:
            errors.append('native proof runtime mismatch: ' + p['name'])
        if not start <= datetime.datetime.fromisoformat(p['startedAt']) <= finish:
            errors.append('proof outside invocation: ' + p['name'])
    if not e.get('checks') or any(c['exitCode'] != 0 for c in e['checks']):
        errors.append('checks failed')
    for c in e.get('checks', []):
        p = Path(c['log'])
        if not p.is_file() or digest(p) != c['sha256']:
            errors.append('check log altered')
    if not e.get('frontend') or not e.get('source'):
        errors.append('missing live provenance')
    return errors
