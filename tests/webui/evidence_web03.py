"""Fail-closed verification of this invocation's WEB-03 live observations."""
from runtime_contract import runtime_root, base_commit

import json
import datetime
from pathlib import Path
from evidence import digest

REQUIRED = ['two-operator-shared-edit', 'personal-ssh-binding',
            'ports-paths-placement-validation', 'ssh-access-permissions', 'secret-free-export']


def verify(directory):
    directory = Path(directory)
    result = json.loads((directory / 'result.json').read_text())
    evidence = json.loads((directory / 'evidence.json').read_text())
    errors = []
    if result['criterion'] != 'WEB-03' or evidence['criterion'] != 'WEB-03':
        errors.append('criterion mismatch')
    if not result['invocationId'] or result['invocationId'] != evidence['invocationId']:
        errors.append('invocation mismatch')
    if result['requiredScenarios'] != REQUIRED or sorted(result['executedScenarios']) != sorted(REQUIRED):
        errors.append('mandatory scenario denominator/execution mismatch')
    if result['skippedScenarios'] or result['failedAssertions'] or result['outcome'] != 'pass':
        errors.append('incomplete or failed acceptance')
    if result['evidenceDigest'] != digest(directory / 'evidence.json'):
        errors.append('evidence digest mismatch')
    if sorted(s['id'] for s in evidence['scenarios']) != sorted(REQUIRED):
        errors.append('missing scenario observations')
    for scenario in evidence['scenarios']:
        if not scenario.get('observedAt') or not scenario.get('assertions') or not scenario.get('artifacts'):
            errors.append('missing observations')
        if not all(a.get('passed') is True for a in scenario['assertions']):
            errors.append('assertion failed')
        if scenario.get('actors') != [{'id': 'operator-a', 'role': 'operator'}, {'id': 'operator-b', 'role': 'operator'}]:
            errors.append('two operator identities missing')
    for artifact in evidence.get('artifacts', []):
        path = Path(artifact['path'])
        if path.is_absolute() or '..' in path.parts or not path.is_file() or digest(path) != artifact['sha256']:
            errors.append('invalid artifact or digest')
    for scenario in evidence['scenarios']:
        for artifact in scenario.get('artifacts', []):
            if artifact not in evidence.get('artifacts', []):
                errors.append('unverified scenario artifact')
    for field in ('server', 'frontend', 'fixture', 'target', 'coverage', 'checks'):
        if not evidence.get(field):
            errors.append('missing live provenance: ' + field)
    if any(c['exitCode'] != 0 for c in evidence.get('checks', [])):
        errors.append('build/tests failed')
    if not evidence.get('target', {}).get('hostFingerprint') or not evidence.get('target', {}).get('sshdSha256'):
        errors.append('missing actual SSH target identity')
    runtime = runtime_root() / result['invocationId']
    if evidence.get('fixture', {}).get('runtime') != str(runtime):
        errors.append('fixture belongs to another invocation')
    elif not (runtime / 'chainbench-dashboard').is_file() or digest(runtime / 'chainbench-dashboard') != evidence.get('server', {}).get('sha256'):
        errors.append('server binary digest mismatch')
    for artifact in evidence.get('source', []) + evidence.get('frontend', []):
        path = Path(artifact['path'])
        if path.is_absolute() or '..' in path.parts or not path.is_file() or digest(path) != artifact['sha256']:
            errors.append('build/source provenance changed')
    if not evidence.get('source'):
        errors.append('source provenance absent')
    for scenario in evidence['scenarios']:
        if scenario.get('invocationId') != result['invocationId']:
            errors.append('scenario belongs to another invocation')
        try:
            start = datetime.datetime.fromisoformat(result['startedAt'].replace('Z', '+00:00'))
            finish = datetime.datetime.fromisoformat(result['finishedAt'].replace('Z', '+00:00'))
            observed = datetime.datetime.fromisoformat(scenario['observedAt'].replace('Z', '+00:00'))
            if not start <= observed <= finish:
                errors.append('observation outside invocation')
        except (KeyError, ValueError):
            errors.append('invalid observation time')
    observations_path = directory / 'api-observations.json'
    if observations_path.is_file():
        records = json.loads(observations_path.read_text())
        access = [r['response'] for r in records if r['endpoint'].endswith('/check') and r['status'] == 200]
        if not any(a['authenticated'] and a['allowedOperations'] == ['deploy'] for a in access):
            errors.append('actual successful SSH check missing')
        if not any(not a['authenticated'] and not a['allowedOperations'] for a in access):
            errors.append('actual rejected SSH key missing')
        if not any(a['authenticated'] and not a['allowedOperations'] for a in access):
            errors.append('actual target permission denial missing')
        if not any(r['status'] == 409 for r in records):
            errors.append('shared revision rejection missing')
        if not any(r['status'] == 404 and '/credential-bindings' in r['endpoint'] for r in records):
            errors.append('foreign binding rejection missing')
        if not all(r.get('actor') in ('operator-a', 'operator-b') for r in records):
            errors.append('untrusted observation actor')
    else:
        errors.append('API observations missing')
    validation_path = directory / 'validation.json'
    expected_cases = {'valid remote placement': True, 'valid local placement': True,
                      'overlapping port bands': False, 'port above 65535': False,
                      'negative stride': False, 'duplicate address': False,
                      'mixed local and remote pool': False, 'unknown host field': False,
                      'relative target root': False, 'path traversal': False, 'invalid input mode': False}
    if validation_path.is_file():
        cases = json.loads(validation_path.read_text())
        if {c['label']: c['result']['valid'] for c in cases} != expected_cases or len(cases) != len(expected_cases):
            errors.append('placement validation coverage mismatch')
    else:
        errors.append('placement validation observations missing')
    if not evidence.get('restart', {}).get('authenticated'):
        errors.append('encrypted overlay persistence not verified')
    return errors


if __name__ == '__main__':
    import sys
    errors = verify(sys.argv[1])
    if errors:
        print('\n'.join(errors), file=sys.stderr)
        sys.exit(1)
    print('WEB-03 PASS')
