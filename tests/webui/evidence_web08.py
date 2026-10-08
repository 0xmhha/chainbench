"""Validate fresh WEB-08 identity, coverage, native proofs and digests."""
import datetime
import json
from pathlib import Path
from evidence import digest
from runtime_contract import runtime_root

REQUIRED = ['native-state-metric-charts', 'time-linked-logs', 'browser-close-logout-continuity', 'reconnect-snapshot-restore', 'sse-gap-drop-stale', 'collection-failure-reporting']
# Native proof script and the observation files each scenario cites.
PROOFS = {
    'monitoring': ('run_monitoring.py', ['metrics-before-restart.json', 'metrics-after-restart.json', 'monitoring-desktop.png', 'monitoring-mobile.png']),
    'snapshot-restore': ('run_snapshot_restore.py', ['observations-before-restart.json', 'observations-after-restart.json', 'snapshot-desktop.png']),
    'ssh-log-collection': ('run_ssh_log_collection.py', ['logs-after-revocation.json', 'ssh-log-collection.png']),
    'logout-continuity': ('run_logout_continuity.py', ['stream-disconnected.png', 'continuity-mobile.png']),
}


def verify(directory):
    directory = Path(directory)
    r = json.loads((directory / 'result.json').read_text())
    e = json.loads((directory / 'evidence.json').read_text())
    errors = []
    if r['criterion'] != 'WEB-08' or e['criterion'] != 'WEB-08' or r['invocationId'] != e['invocationId']:
        errors.append('identity mismatch')
    if r['outcome'] != 'pass' or r['failedAssertions'] or r['skippedScenarios'] or e.get('blockers'):
        errors.append('incomplete acceptance')
    if r['requiredScenarios'] != REQUIRED or sorted(r['executedScenarios']) != sorted(REQUIRED):
        errors.append('coverage mismatch')
    if r['evidenceDigest'] != digest(directory / 'evidence.json'):
        errors.append('evidence digest mismatch')
    coverage = e.get('coverage', {})
    if coverage.get('required') != REQUIRED or sorted(coverage.get('executed', [])) != sorted(REQUIRED) or coverage.get('missing') or coverage.get('mockTargets') != 0:
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
    proofs = e.get('proofs', [])
    if sorted(p['name'] for p in proofs) != sorted(PROOFS):
        errors.append('native proofs missing')
    servers = {p.get('dashboardSHA256') for p in proofs}
    if len(servers) != 1 or e.get('server', {}).get('sha256') not in servers:
        errors.append('proofs used different dashboard builds')
    for p in proofs:
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


if __name__ == '__main__':
    import sys
    errors = verify(sys.argv[1])
    if errors:
        print('\n'.join(errors), file=sys.stderr)
        sys.exit(1)
    print('WEB-08 PASS')
