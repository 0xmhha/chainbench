"""Repeat failed proof cleanup with real installed browser dependencies."""
import hashlib
import json
import os
from pathlib import Path
import shutil

from runtime_storage import create_runtime, clone_file, reclaim_after_shutdown, reclaim_runtime, maintenance


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def allocated(root):
    return sum(p.lstat().st_blocks * 512 for p in root.rglob('*') if p.is_file() and not p.is_symlink())


def main():
    dependency = Path('tests/webui/node_modules/playwright-core')
    if not dependency.is_dir():
        raise RuntimeError('install the locked browser test dependencies first')
    root = Path(os.environ['WEBUI_RUNTIME_ROOT'])
    protected = {p: (digest(p), p.stat().st_mode, p.stat().st_mtime_ns)
                 for p in root.rglob('*.json') if p.is_file() and not p.is_symlink()}
    results = []
    for _ in range(3):
        runtime = create_runtime(prefix='dependency-storage-probe-')
        target = runtime / 'source/tests/webui/node_modules/playwright-core'
        shutil.copytree(dependency, target, copy_function=clone_file, symlinks=True)
        lock = target.parents[1] / 'package-lock.json'
        shutil.copy2('tests/webui/package-lock.json', lock)
        original = (digest(lock), lock.stat().st_mode, lock.stat().st_mtime_ns)
        out = runtime / 'source/chainbench-out/partial'; out.mkdir(parents=True)
        partial = out / 'result.json'; partial.write_text('{"outcome":"incomplete"}'); partial.chmod(0o600)
        with (target / 'package.json').open('rb'):
            try:
                reclaim_runtime(runtime)
            except maintenance.UnsafeMaintenance:
                pass
            else:
                raise AssertionError('open dependency was reclaimed')
        before = allocated(runtime)
        error = RuntimeError('deliberate failed reproduction')
        try:
            try:
                raise error
            finally:
                records = reclaim_after_shutdown(runtime, out)
        except RuntimeError as actual:
            assert actual is error
        assert not target.parent.exists()
        assert (digest(lock), lock.stat().st_mode, lock.stat().st_mtime_ns) == original
        assert partial.read_text() == '{"outcome":"incomplete"}' and partial.stat().st_mode & 0o777 == 0o600
        assert reclaim_runtime(runtime) == []
        remaining = allocated(runtime)
        assert remaining < 1024**2
        results.append({'runtime': str(runtime), 'beforeAllocatedBytes': before, 'afterAllocatedBytes': remaining,
                        'reclaimed': records, 'openDependencyProtected': True, 'failurePreserved': True,
                        'lockBytesModeMtimePreserved': True})
    for path, expected in protected.items():
        assert (digest(path), path.stat().st_mode, path.stat().st_mtime_ns) == expected, path
    report = {'iterations': results, 'protectedJSONFiles': len(protected),
              'dependencyLockSHA256': digest(Path('tests/webui/package-lock.json')),
              'sources': {name: digest(Path(name)) for name in
                          ('tests/webui/runtime_storage.py', 'tests/webui/dependency_storage_probe.py')}}
    destination = Path('/private/tmp/chainbench-browser-dependency-proof.json')
    destination.write_text(json.dumps(report, indent=2) + '\n'); destination.chmod(0o600)
    print('native browser dependency reclamation PASS: ' + str(destination))


if __name__ == '__main__':
    main()
