"""WEB-01 evidence contract. Never equate offline tests with live acceptance."""
import hashlib
import json
from pathlib import Path

REQUIRED = [*(f'{chain}:preset-fields' for chain in ('stablenet', 'wbft', 'wemix')),
            'inheritance-defaults-effective', 'unsupported-ui-api',
            'preset-option-field-coverage', 'binary-surface-provenance']


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def verify(directory):
    directory = Path(directory)
    result = json.loads((directory / 'result.json').read_text())
    evidence = json.loads((directory / 'evidence.json').read_text())
    errors = []
    if result['criterion'] != 'WEB-01' or evidence['criterion'] != 'WEB-01':
        errors.append('criterion mismatch')
    if not result['invocationId'] or result['invocationId'] != evidence['invocationId']:
        errors.append('invocation mismatch')
    if result['requiredScenarios'] != REQUIRED:
        errors.append('required denominator changed')
    if sorted(result['executedScenarios']) != sorted(REQUIRED):
        errors.append('mandatory scenarios missing or duplicated')
    if result['skippedScenarios'] or result['failedAssertions']:
        errors.append('skips or failed assertions')
    if result['evidenceDigest'] != digest(directory / 'evidence.json'):
        errors.append('evidence digest mismatch')
    scenarios = evidence.get('scenarios', [])
    if sorted(s['id'] for s in scenarios) != sorted(REQUIRED):
        errors.append('scenario observations missing')
    for scenario in scenarios:
        if not scenario.get('assertions') or not all(a.get('passed') is True for a in scenario['assertions']):
            errors.append('missing/failed observed assertions: ' + scenario['id'])
        if not scenario.get('observedAt') or not scenario.get('artifacts'):
            errors.append('missing observation/artifacts: ' + scenario['id'])
        for artifact in scenario.get('artifacts', []):
            path = Path(artifact['path'])
            if path.is_absolute() or '..' in path.parts or not path.is_file():
                errors.append('invalid artifact path')
            elif digest(path) != artifact['sha256']:
                errors.append('artifact digest mismatch')
    for field in ('server', 'frontend', 'binaries', 'fixture', 'coverage'):
        if not evidence.get(field):
            errors.append('missing live provenance: ' + field)
    for check in evidence.get('checks', []):
        if check.get('exitCode') != 0 or not Path(check.get('log', '')).is_file():
            errors.append('failed/missing prerequisite check')
        elif digest(check['log']) != check.get('sha256'):
            errors.append('check log digest mismatch')
    if len(evidence.get('checks', [])) != 2:
        errors.append('prerequisite checks missing')
    for item in evidence.get('frontend', []):
        path = Path(item.get('path', ''))
        if not path.is_file() or digest(path) != item.get('sha256'):
            errors.append('frontend build digest mismatch')
    server = evidence.get('server', {})
    server_path = Path(server.get('path', ''))
    if not server_path.is_file() or digest(server_path) != server.get('sha256'):
        errors.append('actual server binary missing or changed')
    coverage = evidence.get('coverage', {})
    binaries = evidence.get('binaries', [])
    if sorted(b.get('chain') for b in binaries) != ['stablenet', 'wbft', 'wemix']:
        errors.append('binary chain denominator incomplete')
    for binary in binaries:
        chain = binary['chain']
        path = Path(binary.get('binaryPath', ''))
        if not path.is_file() or digest(path) != binary.get('binarySha256'):
            errors.append('actual chain binary missing or changed')
        if not binary.get('version') or not binary.get('sourceCommit') or not binary.get('sourceConfigSha256') or not binary.get('host'):
            errors.append('binary identity missing')
        docs = binary.get('documents', [])
        if len(docs) != 4:
            errors.append('chain document provenance missing')
        for document in docs:
            path = Path(document['path'])
            if not path.is_file() or digest(path) != document.get('sha256'):
                errors.append('chain document digest mismatch')
        mapped = binary.get('mapping', [])
        expected = sorted(m['optionKey'] for m in mapped)
        contract_path = directory / (chain + '-contract.json')
        if not contract_path.is_file():
            errors.append('observed contract missing: ' + chain)
        else:
            properties = json.loads(contract_path.read_text())['$defs']['envSpec']['properties']
            declared = properties['launch']['additionalProperties']['properties']
            if expected != sorted(declared):
                errors.append('mapping denominator differs from engine contract: ' + chain)
            for mapping in mapped:
                if mapping.get('schema') != declared.get(mapping['optionKey']):
                    errors.append('option schema changed: ' + chain)
            field_rows = [f for f in coverage.get('fields', []) if f['chain'] == chain]
            if len(field_rows) != 1 or field_rows[0]['expected'] != sorted(properties):
                errors.append('field denominator differs from parser contract: ' + chain)
        actual = sorted(o['key'] for o in coverage.get('options', []) if o['chain'] == chain and o.get('rendered') is True and o.get('engineValid') is True)
        if not expected or actual != expected:
            errors.append('option coverage incomplete: ' + chain)
        field_rows = [f for f in coverage.get('fields', []) if f['chain'] == chain]
        if len(field_rows) != 1 or not field_rows[0]['expected'] or field_rows[0]['expected'] != field_rows[0]['rendered']:
            errors.append('field coverage incomplete: ' + chain)
        presets = [p for p in coverage.get('presets', []) if p['chain'] == chain]
        if not presets or not all(p.get('rendered') and p.get('validated') for p in presets):
            errors.append('preset coverage incomplete: ' + chain)
    browser_path = directory / 'browser.json'
    if browser_path.is_file():
        browser = json.loads(browser_path.read_text())
        catalogs = [r['response'] for r in browser.get('records', []) if r['endpoint'] == 'chain-presets']
        if len(catalogs) != 1 or sorted(p['id'] for p in catalogs[0]) != sorted(p['id'] for p in coverage.get('presets', [])):
            errors.append('preset denominator differs from observed engine catalog')
        if browser.get('failures') or browser.get('coverage') != coverage:
            errors.append('browser observation mismatch')
    else:
        errors.append('browser observation missing')
    if result['outcome'] != 'pass':
        errors.append('outcome is ' + result['outcome'])
    return errors


if __name__ == '__main__':
    import sys
    errors = verify(sys.argv[1])
    if errors:
        print('\n'.join(errors), file=sys.stderr)
        sys.exit(1)
    print('WEB-01 PASS')
