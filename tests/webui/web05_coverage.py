"""Measure which builtin argument paths real passing sessions executed.

A statement counts only when its test passed and the session recorded every
statement: one step record per do statement and one assertion record per
expect statement. The denominator is the served editor contract, never a
list this file keeps.
"""
import json
from pathlib import Path

HEADS = {'do', 'source', 'isPerChain', 'expectPerChain'}
ALIASES = {'rpc': 'rpcCall'}


def contract_paths(contract):
    """Argument paths per (kind, name), exactly as verify_coverage counts them."""
    definitions = contract['contract']['$defs']
    out = {}
    for entry in contract['vocabulary']['entries']:
        schema = definitions[entry['schemaRef'].removeprefix('#/$defs/')]
        fields = set()
        for choice in schema.get('oneOf', [schema]):
            for name, prop in choice['properties'].items():
                if name in HEADS or name == 'expect' and 'const' in prop:
                    continue
                fields.add(name)
        out[(entry['kind'], entry['name'])] = fields
    return out


def statement_paths(statement, readers):
    """(kind, name) -> argument paths one written statement exercises."""
    out = {}
    if isinstance(statement.get('do'), str):
        name = statement['do']
        out[('action', name)] = {k for k in statement if k not in HEADS}
        source = statement.get('source')
        if name in ('read', 'waitFor') and isinstance(source, str):
            out[('reader', source)] = {k for k in statement if k in readers.get(source, set())}
    elif isinstance(statement.get('expect'), str):
        name = ALIASES.get(statement['expect'], statement['expect'])
        out[('assertion', name)] = {k for k in statement if k not in HEADS and k != 'expect'}
    return out


def session_tests(root):
    for status in sorted(Path(root).glob('**/tests/*/status.json')):
        yield status.parent


def executed(roots, contract):
    """Return (paths by registration, per-test records) from passing sessions."""
    denominators = contract_paths(contract)
    readers = {name: fields for (kind, name), fields in denominators.items() if kind == 'reader'}
    covered, tests = {}, []
    for root in roots:
        for test in session_tests(root):
            status = json.loads((test / 'status.json').read_text())
            spec = json.loads((test / 'spec.json').read_text())
            statements = list(spec.get('steps', []))
            steps = json.loads((test / 'steps.json').read_text()) if (test / 'steps.json').exists() else []
            asserts = json.loads((test / 'assert.json').read_text()) if (test / 'assert.json').exists() else []
            do_count = sum(1 for s in statements if isinstance(s.get('do'), str))
            expect_count = len(statements) - do_count
            complete = (status.get('result') == 'pass'
                        and len({s['Index'] for s in steps or []}) == do_count
                        and len(asserts or []) == expect_count
                        and all(not s.get('Error') for s in steps or [])
                        and all(a.get('Pass') is True for a in asserts or []))
            tests.append({'id': status.get('id'), 'path': str(test), 'result': status.get('result'), 'complete': complete})
            if not complete:
                continue
            for statement in statements:
                for key, paths in statement_paths(statement, readers).items():
                    covered.setdefault(key, set()).update(paths)
    return covered, tests


def coverage_rows(contract, covered, edited):
    """Rows in the shape evidence_web05.verify_coverage checks."""
    rows = []
    for (kind, name), fields in sorted(contract_paths(contract).items()):
        ran = sorted(covered.get((kind, name), set()) & fields)
        done = sorted(edited.get((kind, name), set()) & fields)
        rows.append({'kind': kind, 'name': name, 'schema': True, 'edited': done == sorted(fields),
                     'roundTrip': (kind, name) in covered, 'executed': ran == sorted(fields),
                     'argumentPaths': sorted(fields), 'editedArgumentPaths': done, 'executedArgumentPaths': ran})
    return rows


def missing(rows):
    return {r['kind'] + ':' + r['name']: sorted(set(r['argumentPaths']) - set(r['executedArgumentPaths']))
            for r in rows if r['argumentPaths'] != r['executedArgumentPaths']}
