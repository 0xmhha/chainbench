"""Verify WEB-14: the synthesis of WEB-01..13 plus its own build, route and
regression assertions, never the runner's own pass line."""
from runtime_contract import runtime_root

import hashlib
import importlib
import contextlib
import io
import json
import os
from pathlib import Path
import sys

REQUIRED = ['criteria', 'spa-build', 'routes', 'mandatory-checks', 'live-go-tests', 'modes-transports']
CRITERIA = ['WEB-%02d' % i for i in range(1, 14)]
CHECKS = ['make check', 'go test -race -json -p 1 ./...', 'go vet -tags e2e ./...', 'npm --prefix web run build',
          'node --test tests/webui/*.test.mjs']
RACE = CHECKS[1]
# A gated Go test skips without these. A skipped test is not a passed one, so
# the regression run needs every gate opened.
LIVE_ENV = ['GSTABLE_BIN', 'GWEMIX_BIN', 'CHAINBENCH_DOCKER_SERVERS']
VERIFIERS = {'WEB-01': 'evidence', **{'WEB-%02d' % i: 'evidence_web%02d' % i for i in range(2, 14)}}


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def acceptance_root(root, acceptance=None):
    """Where WEB-01..13 were published. A fresh reproduction runs in a staged
    copy that has none of them, so the recheck names the original output."""
    return Path(acceptance or os.environ.get('WEBUI_ACCEPTANCE_ROOT') or Path(root).parent)


def published_workspace(directory):
    """The workspace a criterion was published from. Its artifact paths are
    relative to it: <output>/<criterion>/<file> under that workspace."""
    directory = Path(directory).resolve()
    pending = [json.loads((directory / 'evidence.json').read_text())]
    while pending:
        value = pending.pop()
        if isinstance(value, dict):
            pending.extend(value.values())
        elif isinstance(value, list):
            pending.extend(value)
        elif isinstance(value, str) and not Path(value).is_absolute():
            parts = Path(value).parts
            if directory.name in parts[1:-1]:
                workspace = directory.parents[parts.index(directory.name)]
                if (workspace / value).is_file():
                    return workspace
    return Path.cwd()


def verify_criterion(directory):
    """A criterion's published result, judged by its own verifier from the
    workspace it was published in; a fresh reproduction runs elsewhere."""
    cwd = os.getcwd()
    try:
        os.chdir(published_workspace(directory))
        with contextlib.redirect_stdout(io.StringIO()):
            errors = importlib.import_module(VERIFIERS[directory.name]).verify(directory)
    finally:
        os.chdir(cwd)
    if errors:
        raise AssertionError(directory.name + ': ' + '; '.join(errors))


def coverage(evidences):
    """Modes and transports the criteria observed, from their scenarios."""
    modes, transports = set(), set()
    for evidence in evidences:
        for s in evidence.get('scenarios', []):
            if s.get('mode'):
                modes.add(str(s['mode']).lower())
            for part in str(s.get('transport') or '').lower().split('+'):
                if part:
                    transports.add(part)
    return modes, transports


def go_tests(lines):
    """Outcome counts and skipped tests from `go test -json` output."""
    counts, skipped = {'pass': 0, 'fail': 0, 'skip': 0}, []
    for line in lines:
        try:
            event = json.loads(line)
        except ValueError:
            continue
        if event.get('Test') and event.get('Action') in counts:
            counts[event['Action']] += 1
            if event['Action'] == 'skip':
                skipped.append(event['Package'] + ' ' + event['Test'])
    return counts, skipped


def verify(root, acceptance=None):
    root = Path(root)
    acceptance = acceptance_root(root, acceptance)
    result = json.loads((root / 'result.json').read_text())
    evidence = json.loads((root / 'evidence.json').read_text())
    assert result['criterion'] == evidence['criterion'] == 'WEB-14'
    assert result['invocationId'] == evidence['invocationId']
    assert result['evidenceDigest'] == digest(root / 'evidence.json')
    assert result['outcome'] == 'pass', result['failedAssertions']
    assert set(result['requiredScenarios']) == set(result['executedScenarios']) == set(REQUIRED)
    assert result['skippedScenarios'] == result['failedAssertions'] == []
    assert {s['id'] for s in evidence['scenarios']} == set(REQUIRED)
    for scenario in evidence['scenarios']:
        assert scenario['assertions'] and all(a['passed'] is True for a in scenario['assertions'])
    for item in evidence['artifacts']:
        p = Path(item['path'])
        assert not p.is_absolute() and '..' not in p.parts
        assert digest(p) == item['sha256'], 'artifact changed: ' + str(p)
    runtime = Path(evidence['fixture']['runtime'])
    assert runtime.parent == runtime_root() and runtime.name == result['invocationId']
    assert digest(runtime / 'chainbench-dashboard') == evidence['server']['sha256']
    assert [c['command'] for c in evidence['checks']] == CHECKS
    assert all(c['exitCode'] == 0 for c in evidence['checks'])
    race = next(c for c in evidence['checks'] if c['command'] == RACE)
    counts, skipped = go_tests(Path(race['path']).read_text().splitlines())
    assert counts['fail'] == 0 and counts['pass'] > 0, counts
    assert skipped == [], 'skipped Go tests: ' + ', '.join(skipped)
    assert evidence['spa']['reproducible'] is True and evidence['spa']['servedMatchesEmbedded'] is True
    # Every criterion is re-judged here, on the same frozen source.
    report = evidence['criteria']
    assert [c['criterion'] for c in report] == CRITERIA
    evidences = []
    for entry in report:
        directory = acceptance / entry['criterion']
        verify_criterion(directory)
        published = json.loads((directory / 'evidence.json').read_text())
        assert published['sourceDigest'] == evidence['sourceDigest'], entry['criterion'] + ' belongs to another source'
        assert published['invocationId'] == entry['invocationId']
        evidences.append(published)
    modes, transports = coverage(evidences)
    assert {'personal', 'team'} <= modes and {'local', 'ssh'} <= transports
    print('WEB-14 PASS')


if __name__ == '__main__':
    verify(*sys.argv[1:])
