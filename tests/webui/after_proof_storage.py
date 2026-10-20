"""Compact closed native copies after a capture/recheck child has exited.

The caller preserves the original proof status. This step separately fails if
maintenance or its source/evidence preservation checks fail.
"""
import json
import os
from pathlib import Path
import stat
import sys
import uuid

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'scripts'))
import proof_native_cow as compactor
from runtime_contract import RECEIPT_CRITERIA, runtime_root, source_digest


def proof_snapshot(directory):
    if not directory.exists():
        return {}
    compactor.maintenance.safe_tree(directory)
    result = {}
    for path in sorted(directory.rglob('*')):
        if path.is_symlink():
            raise compactor.maintenance.UnsafeMaintenance('published proof contains a symlink')
        if not path.is_file():
            continue
        s = path.stat()
        result[path.relative_to(directory).as_posix()] = {
            'sha256': compactor.maintenance.digest(path), 'mode': s.st_mode,
            'uid': s.st_uid, 'gid': s.st_gid, 'mtimeNS': s.st_mtime_ns,
            'inode': s.st_ino, 'linkCount': s.st_nlink}
    return result


def extra_binaries():
    raw = os.environ.get('WEBUI_STORAGE_EXTRA_BINARIES', '[]')
    values = json.loads(raw)
    if not isinstance(values, list) or any(not isinstance(p, str) or not p or not Path(p).is_absolute() for p in values):
        raise ValueError('WEBUI_STORAGE_EXTRA_BINARIES must be a JSON array of absolute binary paths')
    replacement = os.environ.get('WEBUI_REPLACEMENT_BINARY')
    if replacement:
        if not Path(replacement).is_absolute():
            raise ValueError('replacement binary must be absolute')
        values.append(replacement)
    return list(dict.fromkeys(values))


def run(criterion, output, workspace=None):
    workspace = Path(workspace or Path.cwd()).resolve()
    output = Path(output)
    if criterion not in RECEIPT_CRITERIA or output.is_absolute() or '..' in output.parts:
        raise ValueError('maintenance requires a known criterion and workspace-relative output')
    catalog = os.environ.get('WEBUI_SELECTED_BINARIES')
    if sys.platform != 'darwin' or not catalog:
        return {'status': 'skipped', 'reason': 'APFS maintenance requires macOS and an explicit binary catalog', 'targets': 0, 'all14Accepted': False}
    root = runtime_root()
    if root == workspace or root.is_relative_to(workspace):
        raise ValueError('runtime root must be outside protected workspace')
    if not root.exists():
        return {'status': 'skipped', 'reason': 'no runtime directory exists', 'targets': 0, 'all14Accepted': False}
    compactor.maintenance.safe_tree(root)
    directory = workspace / output / criterion
    source_before = source_digest(workspace)
    protected = proof_snapshot(directory)
    reports = root / '.storage-maintenance'
    reports.mkdir(mode=0o700, exist_ok=True)
    s = reports.lstat()
    if not stat.S_ISDIR(s.st_mode) or s.st_uid != os.getuid() or stat.S_IMODE(s.st_mode) & 0o077:
        raise compactor.maintenance.UnsafeMaintenance('maintenance report directory is not private and owned')
    name = str(uuid.uuid4()); report = reports / (name + '.json'); audit = reports / (name + '.jsonl')
    data = {'status': 'incomplete', 'criterion': criterion, 'sourceDigest': source_before,
            'catalog': catalog, 'results': [], 'all14Accepted': False}
    failure = None
    try:
        extras = extra_binaries()
        pins = compactor.load_pins(catalog, extras)
        pairs, skipped = compactor.plan(root, pins)
        data.update(additionalBinaries=extras, plannedTargets=len(pairs), skipped=skipped, audit=str(audit))
        if pairs:
            fd = os.open(audit, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600); os.close(fd)
        for pair in pairs:
            result = compactor.clone_pair(pair['anchor'], pair['target'], root, pins, audit)
            if result.get('contentAndMetadataUnchanged') is not True or result.get('independentInode') is not True:
                raise compactor.maintenance.UnsafeMaintenance('native copy preservation was not confirmed')
            result.update(target=pair['target']['path'], privateBytesBefore=pair['privateBytesBefore'],
                          privateBytesAfter=compactor.private_bytes(Path(pair['target']['path'])))
            data['results'].append(result)
        data['status'] = 'completed'
    except Exception as error:
        failure = error; data['error'] = str(error)
    finally:
        try:
            data['protectedSourceUnchanged'] = source_digest(workspace) == source_before
            data['publishedProofUnchanged'] = proof_snapshot(directory) == protected
            if not data['protectedSourceUnchanged'] or not data['publishedProofUnchanged']:
                raise compactor.maintenance.UnsafeMaintenance('protected source or published proof changed during maintenance')
        except Exception as error:
            failure = failure or error; data['preservationError'] = str(error)
        if failure is not None:
            data['status'] = 'incomplete'
        fd = os.open(report, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(fd, 'w') as stream:
            json.dump(data, stream, indent=2); stream.write('\n'); stream.flush(); os.fsync(stream.fileno())
    if failure is not None:
        raise compactor.maintenance.UnsafeMaintenance('post-proof storage incomplete; report: ' + str(report)) from failure
    return {'status': data['status'], 'targets': len(data['results']), 'report': str(report), 'all14Accepted': False}


if __name__ == '__main__':
    try:
        print(json.dumps(run(*sys.argv[1:])))
    except (OSError, ValueError, compactor.maintenance.UnsafeMaintenance) as error:
        print(str(error), file=sys.stderr); raise SystemExit(1)
