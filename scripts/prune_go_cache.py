#!/usr/bin/env python3
"""Prune unreferenced regular entries of this task's Go build cache.

Dry run by default. No module sources, executable directories, runtime proofs or
control files are deleted. Go refreshes access mtimes at most once per hour;
keep at least eight hours and refuse maintenance while a Go tool is running.
An explicit opt-in also permits recent entries containing only verified closed
runtime source paths. Source, shutdown evidence and activity are rechecked.
"""
import argparse
import datetime
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import stat
import time

from webui_storage import UnsafeMaintenance, active, evidence_text, inventory, safe_tree

ROOT = Path('/private/tmp/chainbench-go-cache')
COMPLETED_ROOT = Path('/private/tmp/cbui')
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


def completion_file(path):
    info = path.lstat()
    if not stat.S_ISREG(info.st_mode) or info.st_uid != os.getuid() or info.st_nlink != 1:
        raise UnsafeMaintenance('completion evidence must be an owned regular file')
    data = path.read_bytes()
    return data, {'identity': identity(info), 'sha256': hashlib.sha256(data).hexdigest()}


def dependency_reclaimed(row, dependencies):
    if row.get('event') != 'reclaimed' or row.get('path') not in dependencies:
        return False
    allocated = row.get('allocatedBytes')
    try:
        finished = datetime.datetime.fromisoformat(row['finishedAt'].replace('Z', '+00:00'))
        valid = (type(allocated) is int and allocated >= 0 and finished.tzinfo is not None
                 and finished.timestamp() <= time.time())
    except (KeyError, AttributeError, TypeError, ValueError) as error:
        raise UnsafeMaintenance('invalid completed dependency record') from error
    if not valid:
        raise UnsafeMaintenance('invalid completed dependency record')
    return True


def completed_sources(root):
    """Prove shutdown from the owned runtime's dependency-reclamation journal.

    Ownership alone does not prove completion. Unknown, active or unfinished
    sources never authorize removing a recent cache entry.
    """
    root = Path(root)
    safe_tree(root)
    if root != root.resolve() or root.stat().st_uid != os.getuid():
        raise UnsafeMaintenance('noncanonical or foreign runtime root')
    state = inventory()
    sources = {}
    for runtime in sorted(root.iterdir()):
        if runtime.is_symlink():
            raise UnsafeMaintenance('runtime root contains a symlink')
        marker = runtime / '.webui-owned-runtime.json'
        journal = runtime / 'storage-audit.jsonl'
        source = runtime / 'source'
        if (not runtime.is_dir() or active(runtime, state) or
                not marker.exists() or not journal.exists() or not source.exists()):
            continue
        safe_tree(source)
        marker_data, marker_stamp = completion_file(marker)
        journal_data, journal_stamp = completion_file(journal)
        try:
            owned = json.loads(marker_data)
            records = [json.loads(line) for line in journal_data.splitlines() if line.strip()]
        except (ValueError, UnicodeError) as error:
            raise UnsafeMaintenance('invalid runtime completion evidence') from error
        if owned != {'version': 1, 'path': str(runtime), 'policy': 'completed-disposable-data'}:
            raise UnsafeMaintenance('runtime completion ownership mismatch')
        if any(not isinstance(row, dict) for row in records):
            raise UnsafeMaintenance('invalid runtime completion journal record')
        dependencies = {str(source / suffix) for suffix in
                        ('node_modules', 'web/node_modules', 'tests/webui/node_modules')}
        if not any(dependency_reclaimed(row, dependencies) for row in records):
            continue
        sources[str(source)] = {'runtime': str(runtime), 'sourceIdentity': identity(source.stat()),
                                'marker': marker_stamp, 'journal': journal_stamp}
    return sources


def associated_sources(path, root, sources):
    pattern = re.escape(str(root).encode()) + rb'/([A-Za-z0-9_-]+)/source/'
    names = {str(root / match.decode() / 'source')
             for match in re.findall(pattern, path.read_bytes())}
    if not names or not names <= sources.keys():
        return {}
    return {name: sources[name] for name in sorted(names)}


def plan(root, references, *, now=None, age_hours=8, free_bytes=None,
         target_bytes=25 * GIB, max_bytes=4 * GIB, keep_bytes=GIB,
         completed_runtime_root=None):
    root = Path(root)
    if age_hours < 8 or min(target_bytes, max_bytes, keep_bytes) < 0:
        raise UnsafeMaintenance('unsafe age or byte budget')
    root_id = validate_root(root)
    inactive(root)
    now = time.time() if now is None else now
    free = shutil.disk_usage(root).free if free_bytes is None else free_bytes
    cutoff = now - age_hours * 3600
    identifiers = reference_ids(references)
    runtime_root = Path(completed_runtime_root) if completed_runtime_root is not None else None
    sources = completed_sources(runtime_root) if runtime_root is not None else {}
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
                associated = {}
                if max(s.st_mtime, s.st_ctime) > now:
                    reason = 'future timestamp'
                elif referenced(path, identifiers):
                    reason = 'evidence reference'
                elif max(s.st_mtime, s.st_ctime) > cutoff and not (
                        sources and (associated := associated_sources(path, runtime_root, sources))):
                    reason = 'recent or future timestamp'
                else:
                    candidate = {'path': str(path), 'identity': identity(s),
                                 'bucketIdentity': identity(bs)[:2],
                                 'allocatedBytes': s.st_blocks * 512}
                    if associated:
                        candidate['completedSources'] = associated
                    candidates.append(candidate)
                    continue
            retained[reason] = retained.get(reason, 0) + 1
    limit = min(max_bytes, max(0, target_bytes - free), max(0, total - keep_bytes))
    chosen = []
    size = 0
    for row in sorted(candidates, key=lambda r: (r['identity'][7], r['path'])):
        if row['allocatedBytes'] and size + row['allocatedBytes'] <= limit:
            chosen.append(row)
            size += row['allocatedBytes']
    result = {'root': str(root), 'rootIdentity': root_id, 'createdAt': now,
            'ageHours': age_hours, 'freeBytesBefore': free, 'targetBytes': target_bytes,
            'maximumBytes': max_bytes, 'minimumRetainedBytes': keep_bytes,
            'recognizedAllocatedBytes': total, 'selectedAllocatedBytes': size,
            'retainedCounts': retained,
            'coldCandidates': sum(not r.get('completedSources') for r in candidates), 'entries': chosen}
    if runtime_root is not None:
        result['completedRuntimeRoot'] = str(runtime_root)
        result['completedSourceCandidates'] = sum(bool(r.get('completedSources')) for r in candidates)
    return result


def apply(root, selection, references, *, now=None, progress=None):
    root = Path(root)
    now = time.time() if now is None else now
    if str(root) != selection['root'] or validate_root(root) != selection['rootIdentity']:
        raise UnsafeMaintenance('cache root changed')
    if selection['ageHours'] < 8 or not 0 <= now - selection['createdAt'] <= 300:
        raise UnsafeMaintenance('stale or unsafe cache plan')
    inactive(root)
    refs = reference_ids(references())
    runtime_root = Path(selection['completedRuntimeRoot']) if selection.get('completedRuntimeRoot') else None
    sources = completed_sources(runtime_root) if runtime_root is not None else {}
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
        if row.get('completedSources'):
            if (runtime_root is None or associated_sources(path, runtime_root, sources) != row['completedSources']
                    or max(path.stat().st_mtime, path.stat().st_ctime) > now):
                raise UnsafeMaintenance('completed source changed or became active')
        elif max(path.stat().st_mtime, path.stat().st_ctime) > now - selection['ageHours'] * 3600:
            raise UnsafeMaintenance('cache entry became recent')
    if runtime_root is not None:
        fresh_sources = completed_sources(runtime_root)
        for row in rows:
            if row.get('completedSources') and associated_sources(
                    Path(row['path']), runtime_root, fresh_sources) != row['completedSources']:
                raise UnsafeMaintenance('completed source changed before deletion')
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
    parser.add_argument('--completed-runtime-cache', action='store_true',
                        help='also reclaim entries containing only proved completed runtime source paths')
    parser.add_argument('--repository', type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument('--evidence', type=Path, action='append', default=[])
    parser.add_argument('--report', required=True, type=Path)
    args = parser.parse_args()
    # This command deliberately has no arbitrary cache-root option.
    refs = lambda: evidence_text(args.repository.resolve(), [], args.evidence)
    report_fd = os.open(args.report, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(report_fd, 'w') as output:
        scope = ('cold or verified completed-source task Go build-cache entries only'
                 if args.completed_runtime_cache else 'cold task Go build-cache entries only')
        result = {'scope': scope, 'applied': False}
        try:
            selection = plan(ROOT, refs(), completed_runtime_root=(
                COMPLETED_ROOT if args.completed_runtime_cache else None))
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
