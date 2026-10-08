"""Verify WEB-05 coverage against engine registration, never a reduced denominator."""
from runtime_contract import runtime_root, base_commit

import hashlib
import json
from pathlib import Path
import sys

REQUIRED = ['grammar', 'actions', 'assertions', 'readers', 'arguments', 'references', 'v1-migration', 'invalid-unknown', 'live-execution']

def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()

def verify_coverage(contract, coverage):
    entries = contract['vocabulary']['entries']
    required = {(e['kind'], e['name']) for e in entries}
    actual = {(e['kind'], e['name']) for e in coverage}
    assert len(coverage) == len(actual) == len(required), 'duplicate/missing registrations'
    assert actual == required, 'registration coverage missing'
    definitions = contract['contract']['$defs']
    excluded = {'do', 'expect', 'source', 'isPerChain', 'expectPerChain'}
    by_key = {(e['kind'], e['name']): e for e in coverage}
    for entry in entries:
        schema = definitions[entry['schemaRef'].removeprefix('#/$defs/')]
        # read/waitFor have a reader-specific contract per choice.
        schemas = schema.get('oneOf', [schema])
        fields = sorted({name for choice in schemas for name in choice['properties']} - excluded)
        observed = by_key[(entry['kind'], entry['name'])]
        assert all(observed[k] is True for k in ('schema', 'edited', 'roundTrip', 'executed'))
        for key in ('argumentPaths', 'editedArgumentPaths', 'executedArgumentPaths'):
            assert observed[key] == fields, f"{entry['kind']}:{entry['name']} {key} coverage missing"


def verify(root):
    root = Path(root)
    result = json.loads((root / 'result.json').read_text())
    evidence = json.loads((root / 'evidence.json').read_text())
    assert result['criterion'] == evidence['criterion'] == 'WEB-05'
    assert result['invocationId'] == evidence['invocationId']
    assert result['evidenceDigest'] == digest(root / 'evidence.json')
    assert result['outcome'] == 'pass', result['failedAssertions']
    assert set(result['requiredScenarios']) == set(result['executedScenarios']) == set(REQUIRED)
    assert result['skippedScenarios'] == result['failedAssertions'] == []
    assert {s['id'] for s in evidence['scenarios']} == set(REQUIRED)
    for scenario in evidence['scenarios']:
        assert scenario['assertions'] and all(a['passed'] is True for a in scenario['assertions'])
    for item in evidence['artifacts'] + evidence['frontend']:
        p = Path(item['path'])
        assert not p.is_absolute() and '..' not in p.parts
        assert digest(p) == item['sha256']
    runtime = Path(evidence['fixture']['runtime'])
    assert runtime.parent == runtime_root() and runtime.name == result['invocationId']
    assert digest(runtime / 'chainbench-dashboard') == evidence['server']['sha256']
    assert digest(runtime / 'stablenet-node') == evidence['live']['binary']['sha256']
    assert evidence['checks'] and all(check['exitCode'] == 0 for check in evidence['checks'])
    coverage = json.loads((root / 'coverage.json').read_text())
    contract = json.loads((root / 'contract.json').read_text())
    verify_coverage(contract, coverage)
    assert evidence['live']['mockTargets'] == evidence['live']['skips'] == evidence['live']['failures'] == 0
    assert evidence['live']['binary']['sha256'] and evidence['live']['targetFingerprint']
    print('WEB-05 PASS')

if __name__ == '__main__':
    verify(sys.argv[1])
