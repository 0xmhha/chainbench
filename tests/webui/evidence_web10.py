"""Validate fresh WEB-10 identity, coverage and digests independently of runner exit."""
from runtime_contract import runtime_root, base_commit

import json
import datetime
from pathlib import Path
from evidence import digest
REQUIRED = ['personal-team-login', 'bootstrap-restrictions', 'role-ui-api-matrix', 'csrf-ownership', 'foreign-credential-refusal', 'secret-free-output', 'encrypted-storage-session']
def verify(directory):
    directory = Path(directory)
    r = json.loads((directory/'result.json').read_text())
    e = json.loads((directory/'evidence.json').read_text())
    errors = []
    if r['criterion'] != 'WEB-10' or e['criterion'] != 'WEB-10' or r['invocationId'] != e['invocationId']: errors.append('identity mismatch')
    if r['outcome'] != 'pass' or r['failedAssertions'] or r['skippedScenarios']: errors.append('incomplete acceptance')
    if r['requiredScenarios'] != REQUIRED or sorted(r['executedScenarios']) != sorted(REQUIRED): errors.append('coverage mismatch')
    if r['evidenceDigest'] != digest(directory/'evidence.json'): errors.append('evidence digest mismatch')
    if sorted(s['id'] for s in e['scenarios']) != sorted(REQUIRED): errors.append('scenario mismatch')
    coverage=e.get('coverage',{})
    if coverage.get('required')!=REQUIRED or sorted(coverage.get('executed',[]))!=sorted(REQUIRED) or coverage.get('missing'): errors.append('live coverage denominator mismatch')
    start=datetime.datetime.fromisoformat(r['startedAt']);finish=datetime.datetime.fromisoformat(r['finishedAt'])
    for s in e['scenarios']:
        if not s.get('artifacts') or any(a not in e.get('artifacts',[]) for a in s['artifacts']): errors.append('scenario artifacts unverified')
        if not start<=datetime.datetime.fromisoformat(s['observedAt'].replace('Z','+00:00'))<=finish: errors.append('stale observation')
        if s['invocationId'] != r['invocationId'] or not s['assertions'] or not all(a['passed'] is True for a in s['assertions']): errors.append('invalid observations')
    for a in e.get('artifacts', []) + e.get('source', []) + e.get('frontend', []):
        p=Path(a['path'])
        if p.is_absolute() or '..' in p.parts or not p.is_file() or digest(p)!=a['sha256']: errors.append('artifact changed')
    runtime=runtime_root()/r['invocationId']
    if not (runtime/'chainbench-dashboard').is_file() or digest(runtime/'chainbench-dashboard') != e.get('server',{}).get('sha256'): errors.append('server binary mismatch')
    if not e.get('frontend') or not e.get('source') or e.get('coverage',{}).get('mockTargets')!=0: errors.append('missing live provenance')
    if not e.get('target',{}).get('hostFingerprint') or not e.get('target',{}).get('authenticated'): errors.append('missing actual SSH observation')
    if not e.get('checks') or any(c['exitCode']!=0 for c in e['checks']): errors.append('checks failed')
    for c in e.get('checks',[]):
        p=Path(c['log'])
        if not p.is_file() or digest(p)!=c['sha256']: errors.append('check log altered')
    if e.get('target',{}).get('sshdSha256') != digest('/usr/sbin/sshd'): errors.append('SSH binary identity mismatch')
    if e.get('fixture',{}).get('runtime') != str(runtime) or not e.get('fixture',{}).get('dedicatedSSH'): errors.append('fixture invocation mismatch')
    if not (directory/'browser.json').is_file(): return errors+['browser observations absent']
    observations=json.loads((directory/'browser.json').read_text())
    if observations['secretScan']['leaks'] != 0 or observations['secretScan']['responsesScanned'] < 20: errors.append('secret scan insufficient')
    matrix=observations['matrix']
    for role in ('administrator','operator','viewer'):
        if matrix[role]['uiWritable'] != (role!='viewer') or matrix[role]['writeStatus'] != (403 if role=='viewer' else 201) or matrix[role]['usersStatus'] != (200 if role=='administrator' else 403): errors.append('role matrix mismatch')
    if observations['foreignStatuses'] != [404,404,404]: errors.append('foreign credential access not denied')
    if observations['csrfStatuses'] != [403,403]: errors.append('CSRF not rejected')
    if observations['sessionStatuses'] != [401,401]: errors.append('session revocation missing')
    return errors
if __name__ == '__main__':
    import sys
    errors=verify(sys.argv[1])
    if errors: print('\n'.join(errors),file=sys.stderr); sys.exit(1)
    print('WEB-10 PASS')
