"""Fresh independent live reproduction without mutating protected inputs."""
import importlib
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
from capture import RUNNERS, verify_published
from runtime_contract import base_commit, runtime_root, source_digest, stage_sources


def check(criterion, output):
    workspace = Path.cwd()
    if Path(output).is_absolute() or '..' in Path(output).parts:
        raise ValueError('output must be workspace-relative')
    directory = workspace / output / criterion
    before = source_digest(workspace)
    if not (directory / 'evidence.json').is_file():
        raise ValueError('Missing published live evidence; generate it with --capture first')
    evidence = json.loads((directory / 'evidence.json').read_text())
    if evidence.get('sourceDigest') != before:
        raise ValueError('Published evidence belongs to a different source/SPA; recapture the frozen source')
    _, verifier = RUNNERS[criterion]
    verify_published(verifier, directory)
    protected = {str(p.relative_to(directory)): p.read_bytes() for p in directory.rglob('*') if p.is_file()}
    root = runtime_root(); root.mkdir(parents=True, exist_ok=True)
    env = dict(os.environ, WEBUI_BASE_COMMIT=base_commit(), PYTHONDONTWRITEBYTECODE='1')
    runtime = Path(tempfile.mkdtemp(prefix='recheck-', dir=root))
    source = stage_sources(workspace, runtime)
    if source_digest(source) != before:
        raise ValueError('Staged source differs from protected source')
    result = subprocess.run([sys.executable, 'tests/webui/capture.py', criterion, output], cwd=source, env=env)
    fresh_directory = source / output / criterion
    if result.returncode:
        raise ValueError('Fresh independent live reproduction failed; evidence: ' + str(fresh_directory))
    fresh = json.loads((fresh_directory / 'result.json').read_text())
    if fresh['invocationId'] == evidence['invocationId']:
        raise ValueError('Fresh reproduction reused the published invocation')
    if source_digest(source) != before:
        raise ValueError('Reproduction changed source/SPA')
    if source_digest(workspace) != before:
        raise ValueError('Protected source/SPA mutated during recheck')
    after = {str(p.relative_to(directory)): p.read_bytes() for p in directory.rglob('*') if p.is_file()}
    if protected != after:
        raise ValueError('Protected evidence mutated during recheck')
    print('Independent fresh evidence: ' + str(fresh_directory))
    print(criterion + ' PASS (fresh reproduction; protected source/evidence unchanged)')


if __name__ == '__main__':
    try:
        check(*sys.argv[1:])
    except (AssertionError, ValueError, OSError) as exc:
        print('Live recheck incomplete: ' + str(exc), file=sys.stderr)
        raise SystemExit(1)
