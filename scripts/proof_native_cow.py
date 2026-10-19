#!/usr/bin/env python3
"""Share duplicate native bytes in closed owned proofs; default is a dry plan.

APFS clones have independent inodes. No proof is removed and no hard link is
created. The provided binary catalog supplies hashes, not permission to touch
unmarked, active, unfinished or differently configured files.
"""
import argparse
import ctypes
import datetime
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import sys
import tempfile
import time
import webui_storage as maintenance

NAMES = {'stablenet-node', 'wbft-node', 'wemix-node', 'gstable', 'gwbft', 'gwemix'}
DATABASES = {'chaindata', 'lightchaindata', 'etcd', 'triecache', 'node_modules'}


def describe(path):
    path = Path(path); s = path.lstat()
    if not stat.S_ISREG(s.st_mode):
        raise maintenance.UnsafeMaintenance('native input is not a regular file')
    mode, uid, gid, mtime, attrs, acl, flags = maintenance.metadata(path)
    return {'path': str(path), 'sha256': maintenance.digest(path), 'size': s.st_size,
            'mode': mode, 'uid': uid, 'gid': gid, 'mtimeNS': mtime,
            'xattrs': attrs, 'acl': acl, 'flags': flags, 'device': s.st_dev,
            'inode': s.st_ino, 'linkCount': s.st_nlink}


def identity(row):
    return {key: value for key, value in row.items() if key not in {'path', 'inode', 'linkCount'}}


def group_identity(row):
    return {key: value for key, value in identity(row).items() if key != 'mtimeNS'}


def scope(path, root, expected, pins):
    try:
        parts = path.relative_to(root).parts
    except ValueError as error:
        raise maintenance.UnsafeMaintenance('native input is outside the runtime root') from error
    if len(parts) < 2:
        raise maintenance.UnsafeMaintenance('native input has no owned runtime')
    runtime = root / parts[0]; relative = parts[1:]
    allowed = (len(relative) == 1 and relative[0] in NAMES
               or len(relative) == 4 and relative[0] in {'d', 'fork', 'attach'}
               and relative[1] == 'binaries' and relative[2] == expected['sha256']
               and relative[3] in NAMES
               or len(relative) == 4 and relative[:2] == ('store', 'assets')
               and re.fullmatch('[0-9a-f]{28,64}', relative[2]) and relative[3] == 'payload')
    if not allowed or expected['sha256'] not in pins:
        raise maintenance.UnsafeMaintenance('native path or bytes are outside the supplied catalog')
    maintenance.safe_tree(path.parent)
    current = describe(path)
    if current != expected or current['linkCount'] != 1 or current['uid'] != os.getuid():
        raise maintenance.UnsafeMaintenance('native input changed, is linked or is not owned')
    return runtime


def closure(runtime):
    maintenance.safe_tree(runtime)
    marker, journal = runtime / '.webui-owned-runtime.json', runtime / 'storage-audit.jsonl'
    for path in (marker, journal):
        s = path.lstat()
        if not stat.S_ISREG(s.st_mode) or s.st_nlink != 1 or s.st_uid != os.getuid():
            raise maintenance.UnsafeMaintenance('unsafe runtime ownership or completion record')
    if stat.S_IMODE(marker.stat().st_mode) != 0o600 or json.loads(marker.read_text()) != {'version': 1, 'path': str(runtime), 'policy': 'completed-disposable-data'}:
        raise maintenance.UnsafeMaintenance('runtime ownership marker differs')
    states = {}
    for line in journal.read_text().splitlines():
        row = json.loads(line); path = Path(row['path'])
        if not path.is_absolute() or '..' in path.parts or not path.is_relative_to(runtime) or path.name not in DATABASES:
            raise maintenance.UnsafeMaintenance('unexpected completion path')
        if any(parent.is_symlink() for parent in path.parents):
            raise maintenance.UnsafeMaintenance('completion path has a redirected ancestor')
        event = row['event']; previous = states.get(str(path))
        if event == 'reclaim-start':
            if previous == 'reclaim-start':
                raise maintenance.UnsafeMaintenance('duplicate pending reclamation')
        elif event == 'reclaimed':
            finished = datetime.datetime.fromisoformat(row['finishedAt'])
            if (previous != 'reclaim-start' or finished.tzinfo is None
                    or finished.timestamp() > time.time() or type(row['allocatedBytes']) is not int
                    or row['allocatedBytes'] < 0 or path.exists() or path.is_symlink()):
                raise maintenance.UnsafeMaintenance('reclamation is unpaired, future or no longer closed')
        else:
            raise maintenance.UnsafeMaintenance('unexpected completion event')
        states[str(path)] = event
    if not states or set(states.values()) != {'reclaimed'}:
        raise maintenance.UnsafeMaintenance('runtime cleanup is incomplete')
    return describe(marker), describe(journal)


def clone_command(anchor, destination):
    if sys.platform != 'darwin':
        raise maintenance.UnsafeMaintenance('native shared copies require macOS APFS')
    maintenance.maintenance_run(['/bin/cp', '-c', '-p', str(anchor), str(destination)], capture_output=True, check=True, timeout=60)


def clone_pair(anchor, target, root, pins, audit, inventory=maintenance.inventory):
    root, audit = Path(root).absolute(), Path(audit).absolute(); maintenance.safe_tree(root)
    a, b = Path(anchor['path']), Path(target['path'])
    if a == b:
        raise maintenance.UnsafeMaintenance('cannot replace the anchor')
    ar, br = scope(a, root, anchor, pins), scope(b, root, target, pins)
    try:
        completed = closure(ar), closure(br)
    except (OSError, ValueError, KeyError) as error:
        raise maintenance.UnsafeMaintenance('cannot establish completed runtime records') from error
    if group_identity(anchor) != group_identity(target):
        raise maintenance.UnsafeMaintenance('native content or non-time metadata differs')
    for path in (a, b): maintenance.validate_wal_audit(path, audit)
    state = inventory()
    if maintenance.active(ar, state) or maintenance.active(br, state):
        raise maintenance.UnsafeMaintenance('runtime is active')
    parent = b.parent.stat(); previous = b.stat()
    fd, name = tempfile.mkstemp(prefix='.native-cow-', dir=b.parent); os.close(fd); temporary = Path(name)
    try:
        clone_command(a, temporary)
        copied = temporary.stat()
        if copied.st_uid != target['uid'] or copied.st_gid != target['gid']:
            os.chown(temporary, target['uid'], target['gid'])
        os.utime(temporary, ns=(previous.st_atime_ns, target['mtimeNS']))
        cloned = describe(temporary)
        if identity(cloned) != identity(target) or cloned['inode'] in {anchor['inode'], target['inode']} or cloned['linkCount'] != 1:
            raise maintenance.UnsafeMaintenance('clone changed bytes, metadata or inode isolation')
        scope(a, root, anchor, pins); scope(b, root, target, pins)
        if (closure(ar), closure(br)) != completed:
            raise maintenance.UnsafeMaintenance('completion records changed')
        state = inventory()
        if maintenance.active(ar, state) or maintenance.active(br, state):
            raise maintenance.UnsafeMaintenance('runtime became active')
        for path in (a, b, temporary): maintenance.validate_wal_audit(path, audit)
        with audit.open('a') as output:
            opened = os.fstat(output.fileno())
            if (audit.is_symlink() or not stat.S_ISREG(opened.st_mode) or opened.st_nlink != 1
                    or (opened.st_dev, opened.st_ino) in {(r['device'], r['inode']) for r in (anchor, target, cloned)}
                    or not os.path.samestat(opened, audit.stat())):
                raise maintenance.UnsafeMaintenance('opened audit aliases an input or changed identity')
            def record(event, **fields):
                output.write(json.dumps({'event': event, 'anchor': str(a), 'target': str(b), 'sha256': target['sha256'], **fields}) + '\n')
                output.flush(); os.fsync(output.fileno())
            record('clone-intent')
            os.replace(temporary, b)
            after = describe(b)
            if identity(after) != identity(target) or after['inode'] in {anchor['inode'], target['inode']} or after['linkCount'] != 1:
                raise maintenance.UnsafeMaintenance('replacement failed preservation checks')
            result = {'oldInode': target['inode'], 'newInode': after['inode'],
                      'oldCtimeNS': previous.st_ctime_ns, 'newCtimeNS': b.stat().st_ctime_ns,
                      'contentAndMetadataUnchanged': True, 'independentInode': True,
                      'bytes': target['size'], 'parentMtimeNS': parent.st_mtime_ns}
            record('clone-complete', **result)
            return result
    finally:
        temporary.unlink(missing_ok=True)
        os.utime(b.parent, ns=(parent.st_atime_ns, parent.st_mtime_ns))


def private_bytes(path):
    if sys.platform != 'darwin':
        raise maintenance.UnsafeMaintenance('private allocation measurement requires macOS APFS')
    class Attrs(ctypes.Structure):
        _fields_ = [('n', ctypes.c_ushort), ('reserved', ctypes.c_ushort),
                    ('common', ctypes.c_uint32), ('volume', ctypes.c_uint32),
                    ('directory', ctypes.c_uint32), ('file', ctypes.c_uint32), ('fork', ctypes.c_uint32)]
    import struct
    library = ctypes.CDLL(None, use_errno=True)
    library.getattrlist.argtypes = [ctypes.c_char_p, ctypes.POINTER(Attrs), ctypes.c_void_p, ctypes.c_size_t, ctypes.c_ulong]
    attributes = Attrs(5, 0, 0x80000000, 0, 0, 0, 8); data = ctypes.create_string_buffer(32)
    if library.getattrlist(os.fsencode(path), ctypes.byref(attributes), data, 32, 32):
        raise OSError(ctypes.get_errno(), 'cannot measure APFS private allocation')
    values = struct.unpack('<6IQ', data.raw)
    if values[:6] != (32, 0x80000000, 0, 0, 0, 8):
        raise maintenance.UnsafeMaintenance('unexpected APFS allocation response')
    return values[-1]


def load_pins(manifest, additions=()):
    manifest = Path(manifest); maintenance.safe_tree(manifest.parent)
    if manifest.is_symlink(): raise maintenance.UnsafeMaintenance('binary catalog is a symlink')
    rows = json.loads(manifest.read_text())['binaries']
    if len(rows) != 3 or {r['chain'] for r in rows} != {'stablenet', 'wbft', 'wemix'}:
        raise maintenance.UnsafeMaintenance('binary catalog must contain exactly the three chain families')
    pins = set()
    for row in rows:
        path = Path(row['binary'])
        if path.is_symlink() or not path.is_file() or not os.access(path, os.X_OK) or not re.fullmatch('[0-9a-f]{64}', row['sha256']) or maintenance.digest(path) != row['sha256']:
            raise maintenance.UnsafeMaintenance('binary catalog bytes changed')
        pins.add(row['sha256'])
    for name in additions:
        path = Path(name)
        if path.is_symlink() or not path.is_file() or not os.access(path, os.X_OK):
            raise maintenance.UnsafeMaintenance('additional binary must be a regular executable')
        pins.add(maintenance.digest(path))
    return pins


def plan(root, pins, measure=private_bytes, minimum=20*1024**2, inventory=None):
    root = Path(root).absolute(); maintenance.safe_tree(root)
    groups, skipped = {}, []
    state = (inventory or maintenance.inventory)()
    for runtime in sorted(root.iterdir()):
        if runtime.is_symlink() or not runtime.is_dir(): continue
        if maintenance.active(runtime, state):
            skipped.append({'runtime': str(runtime), 'reason': 'active runtime'})
            continue
        try: closure(runtime)
        except (OSError, ValueError, KeyError, maintenance.UnsafeMaintenance) as error:
            skipped.append({'runtime': str(runtime), 'reason': str(error)}); continue
        paths = {runtime / name for name in NAMES}
        for branch in ('d', 'fork', 'attach'): paths.update((runtime / branch / 'binaries').glob('*/*'))
        paths.update((runtime / 'store/assets').glob('*/payload'))
        for path in sorted(paths):
            if not path.exists(): continue
            try:
                row = describe(path); scope(path, root, row, pins)
                row['privateBytes'] = measure(path)
            except (OSError, ValueError, KeyError, maintenance.UnsafeMaintenance) as error:
                skipped.append({'path': str(path), 'reason': str(error)}); continue
            key = json.dumps(group_identity({k: v for k, v in row.items() if k != 'privateBytes'}), sort_keys=True)
            groups.setdefault(key, []).append(row)
    pairs = []
    for members in groups.values():
        members.sort(key=lambda row: (row['privateBytes'], row['path'])); anchor = members[0]
        for target in members[1:]:
            if target['privateBytes'] < minimum: continue
            pairs.append({'anchor': {k: v for k, v in anchor.items() if k != 'privateBytes'},
                          'target': {k: v for k, v in target.items() if k != 'privateBytes'},
                          'privateBytesBefore': target['privateBytes']})
    return pairs, skipped


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--root', type=Path, required=True); parser.add_argument('--catalog', type=Path, required=True)
    parser.add_argument('--additional-binary', action='append', default=[])
    parser.add_argument('--apply', action='store_true'); parser.add_argument('--audit', type=Path)
    args = parser.parse_args(argv)
    if args.apply and not args.audit: parser.error('--apply requires a separate --audit file')
    pins = load_pins(args.catalog, args.additional_binary); pairs, skipped = plan(args.root, pins)
    results = []
    if args.apply:
        maintenance.safe_tree(args.audit.absolute().parent)
        fd = os.open(args.audit, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        os.close(fd)
        for pair in pairs:
            result = clone_pair(pair['anchor'], pair['target'], args.root, pins, args.audit)
            result.update(target=pair['target']['path'], privateBytesBefore=pair['privateBytesBefore'],
                          privateBytesAfter=private_bytes(Path(pair['target']['path'])))
            results.append(result)
    print(json.dumps({'applied': args.apply, 'selectedPairs': len(pairs), 'pairs': pairs,
                      'skipped': skipped, 'results': results, 'all14Accepted': False}))


if __name__ == '__main__':
    try: main()
    except (OSError, ValueError, KeyError, subprocess.SubprocessError, maintenance.UnsafeMaintenance) as error:
        print('Compaction incomplete: ' + str(error), file=sys.stderr); raise SystemExit(1)
