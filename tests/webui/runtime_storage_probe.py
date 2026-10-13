"""Native, bounded repeated-fixture reclamation and APFS isolation proof."""
import hashlib
import json
import os
from pathlib import Path
import subprocess
from runtime_storage import create_runtime, clone_file, reclaim_runtime, maintenance


def main():
    protected = {str(p): hashlib.sha256(p.read_bytes()).hexdigest()
                 for p in Path(os.environ['WEBUI_RUNTIME_ROOT']).rglob('*.json') if p.is_file() and not p.is_symlink()}
    results = []
    for _ in range(3):
        runtime = create_runtime(prefix='storage-probe-')
        database = runtime / 'store/networks/owned/nodes/node1/etcd/member/wal/0.tmp'
        database.parent.mkdir(parents=True)
        with database.open('wb') as output:
            for _ in range(16):
                output.write(b'wal-data' * (1024**2))
        config = runtime / 'config.toml'; config.write_bytes(b'keep configuration'); config.chmod(0o600)
        log = runtime / 'node.log'; log.write_bytes(b'keep log')
        preserved = {p: (p.read_bytes(), p.stat().st_mode, p.stat().st_mtime_ns) for p in (config, log)}
        with database.open('rb'):
            try:
                reclaim_runtime(runtime)
            except maintenance.UnsafeMaintenance as error:
                assert 'process or open file' in str(error), str(error)
            else:
                raise AssertionError('open node database was reclaimed')
        allocated = sum(p.stat().st_blocks * 512 for p in runtime.rglob('*') if p.is_file())
        records = reclaim_runtime(runtime)
        assert not database.exists()
        for path, metadata in preserved.items():
            assert (path.read_bytes(), path.stat().st_mode, path.stat().st_mtime_ns) == metadata
        # Reclamation is idempotent, and large clones are independent on APFS.
        assert reclaim_runtime(runtime) == []
        source = runtime / 'clone-source'; source.write_bytes(b'original' * 1024**2); source.chmod(0o600)
        copy = runtime / 'clone-copy'; clone_file(source, copy)
        assert copy.stat().st_ino != source.stat().st_ino
        assert copy.read_bytes() == source.read_bytes()
        assert copy.stat().st_mode == source.stat().st_mode
        copy.write_bytes(b'changed')
        assert source.read_bytes() == b'original' * 1024**2
        copy.unlink(); source.unlink()
        remaining = sum(p.stat().st_blocks * 512 for p in runtime.rglob('*') if p.is_file())
        assert remaining < 1024**2
        results.append({'runtime': str(runtime), 'beforeAllocatedBytes': allocated, 'afterAllocatedBytes': remaining,
                        'reclaimed': records, 'openFileProtected': True, 'proofBytesModeMtimePreserved': True,
                        'copyOnWriteIsolation': True})
    for path, expected in protected.items():
        assert hashlib.sha256(Path(path).read_bytes()).hexdigest() == expected, path
    proof = {'iterations': results, 'protectedJSONFiles': len(protected),
             'sourceSHA256': hashlib.sha256(Path('tests/webui/runtime_storage.py').read_bytes()).hexdigest(),
             'platform': subprocess.check_output(['uname', '-sm'], text=True).strip()}
    path = Path('/private/tmp/chainbench-runtime-storage-proof.json')
    path.write_text(json.dumps(proof, indent=2) + '\n'); path.chmod(0o600)
    print('native repeated reclamation PASS: ' + str(path))


if __name__ == '__main__':
    main()
