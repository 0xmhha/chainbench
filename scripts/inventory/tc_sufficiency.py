"""Validate evidence-bound criterion reviews; this is not a native test runner."""
from collections import Counter
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys

REVIEW = 'docs/dev/wemix4-tc-sufficiency-review.json'
CRITERIA = 'docs/tc/wemix4.0/01-test-scenarios.md'
STATES = {'partial', 'confirmed_scope', 'conflict', 'not_verified', 'merged', 'accepted'}


def go_symbols(root, sources):
    """Read real Go AST declarations, including methods and gated test files."""
    if not sources:
        return {}
    result = subprocess.run(['go', 'run', './scripts/inventory/source-ast'],
                            input=json.dumps([{'path': p, 'source': s} for p, s in sources.items()]),
                            text=True, capture_output=True, check=True, timeout=60, cwd=root)
    parsed = json.loads(result.stdout)
    if len(parsed) != len(sources) or any(p.get('error') for p in parsed):
        raise ValueError('Go AST parsing failed')
    return {p['path']: {n['name'] for n in p['nodes'] if n['kind'] in {'function', 'method'}} for p in parsed}


def validate_review(root, review):
    """Check actual criterion/source bytes, anchors and honest acceptance states."""
    if not isinstance(review, dict) or not isinstance(review.get('criterionSource'), dict):
        return ['invalid review or criterion source shape']
    records = review.get('records')
    if not isinstance(records, list) or any(not isinstance(r, dict) or not isinstance(r.get('id'), str) for r in records):
        return ['invalid criterion record shape']
    for row in records:
        for field in ['targets', 'verifiedEvidence', 'observations']:
            values = row.get(field)
            if not isinstance(values, list) or any(not isinstance(v, dict) for v in values):
                return ['invalid criterion ' + field + ' shape']
        if not isinstance(row.get('remainingConditions'), list) or any(not isinstance(v, str) for v in row['remainingConditions']):
            return ['invalid remaining conditions shape']
    root = Path(root).resolve()
    errors, cache = [], {}

    def fail(message):
        errors.append(message)

    def read(path):
        if not isinstance(path, str) or Path(path).is_absolute() or '..' in Path(path).parts:
            fail('invalid repository-relative path: ' + str(path)); return None
        p = root / path
        if p.is_symlink() or not p.resolve().is_relative_to(root):
            fail('linked or escaping source: ' + path); return None
        if path not in cache:
            try:
                cache[path] = p.read_bytes()
            except OSError:
                fail('missing source: ' + path); return None
        return cache[path]

    def check_ref(ref):
        if not isinstance(ref, dict):
            fail('invalid source reference'); return None
        data = read(ref.get('path'))
        if data is not None and hashlib.sha256(data).hexdigest() != ref.get('sha256'):
            fail('stale source: ' + ref['path'])
        return data

    source = review.get('criterionSource', {})
    if source.get('path') != CRITERIA:
        fail('wrong original criterion source')
    data = check_ref(source)
    if data is None:
        return errors
    text = data.decode()
    heads = list(re.finditer(r'^### ((?:NODE|TX|WBFT|GOV|RPC)-\d{3}) — (.+)$', text, re.M))
    sections = {m[1]: text[m.start():heads[i+1].start() if i+1 < len(heads) else len(text)] for i, m in enumerate(heads)}
    ids = [r.get('id') for r in records]
    if len(heads) != 87 or len(ids) != 87 or len(set(ids)) != 87 or set(ids) != set(sections):
        fail('criterion IDs missing, duplicated or unknown')
    by_id = {r.get('id'): r for r in records}
    if dict(Counter(r.get('status') for r in records)) != review.get('statusCounts'):
        fail('status counts differ from actual records')
    go_sources = {}
    for row in records:
        for ref in row.get('targets', []):
            if 'function' in ref:
                actual = read(ref.get('path'))
                if actual is not None:
                    go_sources[ref['path']] = actual.decode()
    try:
        symbols = go_symbols(root, go_sources)
    except (OSError, ValueError, subprocess.SubprocessError):
        fail('cannot verify Go AST declarations')
        symbols = {}
    for row in records:
        name, state = row.get('id'), row.get('status')
        if name in sections and hashlib.sha256(sections[name].encode()).hexdigest() != row.get('criterionSha256'):
            fail('stale original criterion: ' + name)
        if state not in STATES:
            fail('unknown review status: ' + str(name))
        if state in {'partial', 'conflict', 'not_verified', 'merged'} and not row.get('remainingConditions'):
            fail('hidden remaining conditions: ' + str(name))
        if row.get('fullCriterionAccepted') is not False and state != 'accepted':
            fail('partial or scoped evidence claimed as full acceptance: ' + str(name))
        if state == 'accepted' and (row.get('fullCriterionAccepted') is not True or row.get('remainingConditions')):
            fail('accepted criterion still has unverified conditions: ' + str(name))
        evidence = row.get('verifiedEvidence', [])
        if state in {'confirmed_scope', 'accepted'} and (not evidence or not row.get('confirmedScope')):
            fail('scoped acceptance without evidence or scope: ' + str(name))
        if state == 'merged':
            seen, current = {name}, row
            while current.get('status') == 'merged':
                target = current.get('mergedInto')
                if target not in by_id or target in seen:
                    fail('missing or cyclic merge target: ' + str(name)); break
                seen.add(target); current = by_id[target]
        for ref in row.get('targets', []) + evidence:
            actual = check_ref(ref)
            if actual is None:
                continue
            if 'function' in ref:
                if ref['function'] not in symbols.get(ref['path'], set()):
                    fail('missing function: ' + ref['function'])
            if 'id' in ref and json.loads(actual).get('id') != ref['id']:
                fail('case ID mismatch: ' + str(name))
        for observation in row.get('observations', []):
            if 'pointer' not in observation:
                continue
            actual = read(observation.get('path'))
            if actual is None:
                continue
            try:
                value = json.loads(actual)
                pointer = observation['pointer']
                if not pointer.startswith('/'):
                    raise ValueError('not a JSON pointer')
                for segment in pointer[1:].split('/'):
                    segment = segment.replace('~1', '/').replace('~0', '~')
                    if isinstance(value, list):
                        if not segment.isdecimal():
                            raise ValueError('invalid array index')
                        value = value[int(segment)]
                    else:
                        value = value[segment]
                if json.dumps(value, sort_keys=True) != json.dumps(observation['statement'], sort_keys=True):
                    fail('stale assertion: ' + str(name) + pointer)
            except (ValueError, KeyError, IndexError, TypeError):
                fail('missing assertion: ' + str(name))
    complete = bool(records) and all(r.get('fullCriterionAccepted') is True and r.get('status') == 'accepted' for r in records)
    if review.get('allCriteriaAccepted') is not complete:
        fail('overall acceptance differs from actual criterion states')
    return errors


if __name__ == '__main__':
    root = Path(__file__).resolve().parents[2]
    review = json.loads((root / REVIEW).read_text())
    errors = validate_review(root, review)
    print(json.dumps({'errors': errors, 'criteria': len(review['records']),
                      'statusCounts': review['statusCounts'],
                      'allCriteriaAccepted': review['allCriteriaAccepted']}, ensure_ascii=False))
    sys.exit(bool(errors))
