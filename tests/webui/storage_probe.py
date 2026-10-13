#!/usr/bin/env python3
"""Explicit native storage proof; never runs as part of unittest discovery.

Run on macOS with process/open-file inventory access. Creates its own temporary
fixtures, tests live file protection, and retains a JSON proof outside the repo.
"""
import contextlib
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import time

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'scripts'))
import webui_storage as storage


def main():
    if sys.platform != 'darwin':
        raise RuntimeError('native proof requires macOS')
    base = Path(tempfile.mkdtemp(prefix='chainbench-storage-proof-', dir='/private/tmp'))
    root = base / 'runtime'; root.mkdir()
    repo = base / 'repository'; repo.mkdir()
    runtime = root / '00000000-0000-0000-0000-000000000001'; runtime.mkdir()
    wal = runtime / 'etcd/member/wal/0.tmp'; wal.parent.mkdir(parents=True)
    with wal.open('wb') as output:
        for _ in range(40):
            output.write(b'chainbench-wal-proof\n' * 52429)
    wal.chmod(0o600)
    subprocess.run(['/usr/bin/xattr', '-w', 'user.chainbench-proof', 'preserve', str(wal)], check=True)
    os.utime(wal, (time.time() - 7200, time.time() - 7200))
    before = storage.metadata(wal)
    sha = storage.digest(wal)
    audit = base / 'compression.jsonl'
    holder = subprocess.Popen([sys.executable, '-c', 'import sys; f=open(sys.argv[1], "rb"); print("ready",flush=True); sys.stdin.read()', str(wal)], stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True)
    try:
        assert holder.stdout.readline().strip() == 'ready'
        with contextlib.redirect_stdout(io.StringIO()):
            storage.compress_wal(root, audit, compress=True)
        assert not audit.exists(), 'active WAL was compressed'
    finally:
        holder.stdin.close(); holder.wait(timeout=10); holder.stdout.close()
    with contextlib.redirect_stdout(io.StringIO()):
        storage.compress_wal(root, audit, compress=True)
    compressed = json.loads(audit.read_text())
    assert compressed['savedBytes'] > 0
    assert storage.digest(wal) == sha and storage.metadata(wal) == before

    old = root / 'recheck-protected'
    (old / 'source/chainbench-out/web-ui-acceptance/WEB-05').mkdir(parents=True)
    new = root / 'recheck-newer'
    (new / 'source/chainbench-out/web-ui-acceptance/WEB-05').mkdir(parents=True)
    evidence = repo / 'chainbench-out'; evidence.mkdir()
    (evidence / 'result.json').write_text(json.dumps({'recheck': str(old)}))
    disposable = root / '00000000-0000-0000-0000-000000000002'; disposable.mkdir()
    aged = time.time() - 7200
    for path in [*root.rglob('*'), root]:
        os.utime(path, (aged, aged))
    with contextlib.redirect_stdout(io.StringIO()):
        storage.cleanup(root, repo, time.time() - 86400, extras=[audit], delete=True)
    assert old.exists() and new.exists() and runtime.exists()
    assert not disposable.exists()

    # Confirm real accepted proof artifacts survived and are protected by policy.
    accepted = Path('/private/tmp/chainbench-frozen-refactor-publication-audit.json')
    protected = json.loads(accepted.read_text())['protectedReceiptDigests']
    for name, expected in protected.items():
        assert storage.digest(Path(name)) == expected, name
    actual_root = Path(os.environ.get('WEBUI_RUNTIME_ROOT', '/private/tmp/cbui'))
    actual_repo = Path(__file__).resolve().parents[2]
    extras = [accepted, Path('/private/tmp/chainbench-frozen-refactor-results.jsonl'),
              Path('/private/tmp/chainbench-wal-compression-audit.jsonl')]
    plan = dict(storage.cleanup_plan(actual_root, actual_repo, 1791475080, extras))
    for name in protected:
        path = Path(name)
        if path.is_relative_to(actual_root):
            runtime_path = actual_root / path.relative_to(actual_root).parts[0]
            assert plan[runtime_path] is not None, str(runtime_path)
    record = {'activeWALPreserved': True, 'compression': compressed,
              'referencedRecheckPreserved': True, 'unreferencedFixtureDeleted': True,
              'protectedReceiptsVerified': len(protected), 'realRootDryRunOnly': True,
              'sourceFiles': {str(path): storage.digest(path) for path in [actual_repo / 'scripts/webui_storage.py', Path(__file__).resolve(), actual_repo / 'tests/webui/test_storage.py']}}
    (base / 'proof.json').write_text(json.dumps(record, indent=2) + '\n')
    print(json.dumps({'proof': str(base / 'proof.json'), 'verified': record}))


if __name__ == '__main__':
    main()
