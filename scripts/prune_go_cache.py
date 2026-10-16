#!/usr/bin/env python3
"""Prune only cold, unreferenced regular entries of this task's Go build cache.

Dry run by default. No module sources, executable directories, runtime proofs or
control files are deleted. Go refreshes access mtimes at most once per hour;
keep at least eight hours and refuse maintenance while a Go tool is running.
"""
import argparse
import json
import os
from pathlib import Path
import re
import shutil
import stat
import time

from webui_storage import UnsafeMaintenance, evidence_text, inventory, safe_tree

ROOT = Path('/private/tmp/chainbench-go-cache')
README = 'This directory holds cached build artifacts from the Go build system.\n'
GIB = 1024 ** 3
ENTRY = re.compile(r'([0-9a-f]{64})-[ad]')
TOOLS = {'go', 'compile', 'link', 'asm', 'cgo'}


def identity(s):
    return [s.st_dev, s.st_ino, s.st_mode, s.st_uid, s.st_gid,
            s.st_nlink, s.st_size, s.st_mtime_ns, s.st_ctime_ns, s.st_blocks]


def validate_root(root):
    root = Path(root)
    safe_tree(root)
    if root != root.resolve() or root.stat().st_uid != os.getuid():
        raise UnsafeMaintenance('noncanonical or foreign cache root')
    marker = root / 'README'
    s = marker.lstat()
    if (not stat.S_ISREG(s.st_mode) or s.st_uid != os.getuid() or
            s.st_nlink != 1 or not marker.read_text().startswith(README)):
        raise UnsafeMaintenance('not an owned Go build cache')
    return identity(root.stat())[:2]


def inactive(root):
    commands, opened = inventory()
    for command in commands.splitlines():
        words = command.strip().split()
        if words and Path(words[0]).name in TOOLS:
            raise UnsafeMaintenance('Go build tool active; retry after it exits')
    prefix = str(root) + '/'
    if any(p == str(root) or p.startswith(prefix) for p in opened):
        raise UnsafeMaintenance('cache file is open; retry after it closes')


def reference_ids(references):
    # Every recognized path contains its complete 64-hex identifier. Retaining
    # every mentioned identifier also protects bare digest references, without
    # rescanning the full evidence catalog for each of thousands of cache files.
    return {value.lower() for value in re.findall(r'[0-9a-fA-F]{64}', references)}


def referenced(path, identifiers):
    return path.name[:64] in identifiers


def plan(root, references, *, now=None, age_hours=8, free_bytes=None,
         target_bytes=25 * GIB, max_bytes=4 * GIB, keep_bytes=GIB):
    root = Path(root)
    if age_hours < 8 or min(target_bytes, max_bytes, keep_bytes) < 0:
        raise UnsafeMaintenance('unsafe age or byte budget')
    root_id = validate_root(root)
    inactive(root)
    now = time.time() if now is None else now
    free = shutil.disk_usage(root).free if free_bytes is None else free_bytes
    cutoff = now - age_hours * 3600
    identifiers = reference_ids(references)
    candidates = []
    total = 0
    retained = {}
    for bucket in sorted(root.iterdir()):
        bs = bucket.lstat()
        if (not re.fullmatch('[0-9a-f]{2}', bucket.name) or
                not stat.S_ISDIR(bs.st_mode) or bs.st_uid != os.getuid()):
            continue
        for path in sorted(bucket.iterdir()):
            s = path.lstat()
            match = ENTRY.fullmatch(path.name)
            if not match or not match[1].startswith(bucket.name):
                reason = 'unknown name'
            elif not stat.S_ISREG(s.st_mode) or s.st_uid != os.getuid() or s.st_nlink != 1:
                reason = 'not a singly linked owned regular file'
            else:
                total += s.st_blocks * 512
                if max(s.st_mtime, s.st_ctime) > cutoff:
                    reason = 'recent or future timestamp'
                elif referenced(path, identifiers):
                    reason = 'evidence reference'
                else:
                    candidates.append({'path': str(path), 'identity': identity(s),
                                       'bucketIdentity': identity(bs)[:2],
                                       'allocatedBytes': s.st_blocks * 512})
                    continue
            retained[reason] = retained.get(reason, 0) + 1
    limit = min(max_bytes, max(0, target_bytes - free), max(0, total - keep_bytes))
    chosen = []
    size = 0
    for row in sorted(candidates, key=lambda r: (r['identity'][7], r['path'])):
        if row['allocatedBytes'] and size + row['allocatedBytes'] <= limit:
            chosen.append(row)
            size += row['allocatedBytes']
    return {'root': str(root), 'rootIdentity': root_id, 'createdAt': now,
            'ageHours': age_hours, 'freeBytesBefore': free, 'targetBytes': target_bytes,
            'maximumBytes': max_bytes, 'minimumRetainedBytes': keep_bytes,
            'recognizedAllocatedBytes': total, 'selectedAllocatedBytes': size,
            'retainedCounts': retained, 'coldCandidates': len(candidates), 'entries': chosen}


def apply(root, selection, references, *, now=None, progress=None):
    root = Path(root)
    now = time.time() if now is None else now
    if str(root) != selection['root'] or validate_root(root) != selection['rootIdentity']:
        raise UnsafeMaintenance('cache root changed')
    if selection['ageHours'] < 8 or not 0 <= now - selection['createdAt'] <= 300:
        raise UnsafeMaintenance('stale or unsafe cache plan')
    inactive(root)
    refs = reference_ids(references())
    rows = selection['entries']
    if len({r['path'] for r in rows}) != len(rows):
        raise UnsafeMaintenance('duplicate cache selection')
    if (sum(r['allocatedBytes'] for r in rows) != selection['selectedAllocatedBytes'] or
            selection['selectedAllocatedBytes'] > min(selection['maximumBytes'],
                max(0, selection['recognizedAllocatedBytes'] - selection['minimumRetainedBytes']))):
        raise UnsafeMaintenance('cache plan exceeds budget')
    for row in rows:
        path = Path(row['path'])
        if (path.parent.parent != root or not re.fullmatch('[0-9a-f]{2}', path.parent.name) or
                not ENTRY.fullmatch(path.name) or not path.name.startswith(path.parent.name)):
            raise UnsafeMaintenance('cache selection escaped namespace')
        if path.parent.is_symlink() or identity(path.parent.stat())[:2] != row['bucketIdentity']:
            raise UnsafeMaintenance('cache bucket changed')
        if identity(path.lstat()) != row['identity'] or referenced(path, refs):
            raise UnsafeMaintenance('cache entry changed or newly referenced')
        if max(path.stat().st_mtime, path.stat().st_ctime) > now - selection['ageHours'] * 3600:
            raise UnsafeMaintenance('cache entry became recent')
    # A second fresh process/descriptor inventory follows all filesystem checks.
    inactive(root)
    if validate_root(root) != selection['rootIdentity']:
        raise UnsafeMaintenance('cache root changed before deletion')
    deleted = [] if progress is None else progress
    for row in rows:
        path = Path(row['path'])
        fd = os.open(path.parent, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
        try:
            if identity(os.fstat(fd))[:2] != row['bucketIdentity']:
                raise UnsafeMaintenance('cache bucket changed before deletion')
            current = os.stat(path.name, dir_fd=fd, follow_symlinks=False)
            if identity(current) != row['identity']:
                raise UnsafeMaintenance('cache entry changed before deletion')
            os.unlink(path.name, dir_fd=fd)
            deleted.append(str(path))
        finally:
            os.close(fd)
    return {'deletedPaths': deleted, 'deletedCount': len(deleted),
            'selectedAllocatedBytes': selection['selectedAllocatedBytes'],
            'freeBytesAfter': shutil.disk_usage(root).free}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--delete', action='store_true')
    parser.add_argument('--repository', type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument('--evidence', type=Path, action='append', default=[])
    parser.add_argument('--report', required=True, type=Path)
    args = parser.parse_args()
    # This command deliberately has no arbitrary cache-root option.
    refs = lambda: evidence_text(args.repository.resolve(), [], args.evidence)
    report_fd = os.open(args.report, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(report_fd, 'w') as output:
        result = {'scope': 'cold task Go build-cache entries only', 'applied': False}
        try:
            selection = plan(ROOT, refs())
            result['plan'] = selection
            if args.delete:
                result['result'] = {'deletedPaths': []}
                result['result'].update(apply(ROOT, selection, refs,
                                              progress=result['result']['deletedPaths']))
                result['applied'] = True
        except Exception as error:
            result['error'] = str(error)
            raise
        finally:
            output.write(json.dumps(result, indent=2) + '\n')
            output.flush()
            os.fsync(output.fileno())
    print(json.dumps({'report': str(args.report), 'applied': result['applied'],
                      'selectedCount': len(selection['entries']),
                      'selectedAllocatedBytes': selection['selectedAllocatedBytes'],
                      'deletedCount': result.get('result', {}).get('deletedCount', 0)}))


if __name__ == '__main__':
    main()
