"""Fail-closed verification of this invocation's manifest acceptance evidence."""
from runtime_contract import runtime_root, base_commit

import hashlib
import json
from pathlib import Path
import sys
import subprocess
import datetime

REQUIRED = ['builtin-' + c for c in ('stablenet', 'wbft', 'wemix')] + ['external-' + c for c in ('stablenet', 'wbft', 'wemix')] + ['unsupported-family', 'unsupported-dialect', 'wrong-binary-identity', 'manifest-management']

def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()

def verify(root):
    root = Path(root)
    result = json.loads((root / 'result.json').read_text())
    evidence = json.loads((root / 'evidence.json').read_text())
    assert result['criterion'] == evidence['criterion'] == 'WEB-04'
    assert result['invocationId'] == evidence['invocationId']
    assert result['evidenceDigest'] == digest(root / 'evidence.json')
    assert result['outcome'] == 'pass', result.get('failedAssertions')
    assert len(result['requiredScenarios']) == len(result['executedScenarios']) == len(REQUIRED)
    assert set(result['requiredScenarios']) == set(result['executedScenarios']) == set(REQUIRED)
    assert result['skippedScenarios'] == result['failedAssertions'] == []
    assert set(s['id'] for s in evidence['scenarios']) == set(REQUIRED)
    for item in evidence['artifacts']:
        p = Path(item['path'])
        assert not p.is_absolute() and '..' not in p.parts
        assert p.is_file() and digest(p) == item['sha256']
    for scenario in evidence['scenarios']:
        assert scenario['assertions'] and all(a['passed'] is True for a in scenario['assertions'])
    assert evidence['server']['sha256'] and evidence['frontend'] and evidence['binaries']
    runtime=Path(evidence['fixture']['runtime'])
    assert runtime.parent == runtime_root() and runtime.name == result['invocationId']
    assert digest(runtime/'chainbench-dashboard') == evidence['server']['sha256']
    for asset in evidence['frontend']:
        assert digest(asset['path']) == asset['sha256']
    for binary in evidence['binaries']:
        assert binary['sha256'] and binary['version'] and binary['helpDigest']
        executable=runtime/(binary['chain']+'-node')
        assert digest(executable) == binary['sha256']
        assert subprocess.check_output([str(executable),'version'],text=True,timeout=10) == binary['version']
        help_text=subprocess.check_output([str(executable),'--help'],timeout=10)
        assert hashlib.sha256(help_text).hexdigest() == binary['helpDigest']
    observations = json.loads((root / 'observations.json').read_text())
    assert observations['scenarios'] == evidence['scenarios']
    for scenario in evidence['scenarios']:
        observed=datetime.datetime.fromisoformat(scenario['observedAt'].replace('Z','+00:00'))
        assert datetime.datetime.fromisoformat(result['startedAt']) <= observed <= datetime.datetime.fromisoformat(result['finishedAt'])
    for scenario in observations['setups']:
        assert scenario['state']['chain'] == scenario['chain']
        assert len(scenario['databaseFiles']) == 4, 'real init did not create all databases'
        assert {f['node'] for f in scenario['databaseFiles']} == {1,2,3,4}
        for artifact in scenario['databaseFiles'] + [scenario['genesis']]:
            assert digest(artifact['path']) == artifact['sha256']
        assert scenario['binary']['sha256'] == next(b['sha256'] for b in evidence['binaries'] if b['chain'] == scenario['chain'].removeprefix('external-'))
    assert len(observations['setups']) == 6
    assert all(s['state']['steps']['init']['done'] for s in observations['setups'])
    assert all(len(s['state']['nodes']) == 4 for s in observations['setups'])
    print('WEB-04 PASS')

if __name__ == '__main__':
    verify(sys.argv[1])
