#!/usr/bin/env python3
"""Conservative, opt-in maintenance for disposable Web acceptance runtimes.

cleanup --created-after ISO_DATE defaults to a dry run; --delete applies it.
compress-wal defaults to a dry run; --compress uses macOS transparent compression.
Pass --evidence for every external proof archive or campaign receipt to retain.
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
import subprocess
import sys
import tempfile
import time


class UnsafeMaintenance(RuntimeError):
    """An inventory or preservation check failed; no further changes are safe."""


def inventory():
    commands = subprocess.run(['ps', '-axo', 'command='], capture_output=True, text=True)
    if commands.returncode != 0 or commands.stderr.strip():
        raise UnsafeMaintenance('cannot inventory processes')
    opened = subprocess.run(['lsof', '-nP', '-F', 'n'], capture_output=True, text=True)
    if opened.returncode not in (0, 1) or opened.stderr.strip():
        raise UnsafeMaintenance('cannot inventory open files')
    return commands.stdout, [line[1:] for line in opened.stdout.splitlines() if line.startswith('n')]


def active(path, state):
    commands, opened = state
    name = str(path)
    return name in commands or any(p == name or p.startswith(name + '/') for p in opened)


def safe_tree(path):
    """Reject links, including an ancestor redirected outside the runtime root."""
    for parent in [path, *path.parents]:
        if parent.is_symlink():
            raise UnsafeMaintenance('symlink in maintenance path: ' + str(parent))
    if not path.is_dir():
        raise UnsafeMaintenance('runtime root must be a directory')


def newest_mtime(path):
    newest = path.lstat().st_mtime
    def failed(error):
        raise UnsafeMaintenance('cannot inspect runtime: ' + str(error))
    for directory, dirs, files in os.walk(path, onerror=failed):
        for name in dirs + files:
            entry = Path(directory) / name
            if entry.is_symlink():
                raise UnsafeMaintenance('runtime contains symlink: ' + str(entry))
            newest = max(newest, entry.stat().st_mtime)
    return newest


def evidence_text(repository, entries, extras):
    # Follow-up proofs live beside worklists, not only in web-ui-evidence.
    # Handoff prose also cites runtimes; losing that reference must not make
    # the referenced source, logs or receipts disposable.
    roots = [repository / 'chainbench-out', repository / 'docs']
    roots += list(repository.glob('*.md'))
    roots += [entry / 'source/chainbench-out' for entry in entries if entry.name.startswith('recheck-')]
    roots += list(extras)
    text = []
    for root in roots:
        if root in extras and not root.exists():
            raise UnsafeMaintenance('missing evidence: ' + str(root))
        if root.is_symlink():
            raise UnsafeMaintenance('evidence symlink: ' + str(root))
        if not root.exists():
            continue
        def readable_files(directory):
            def failed(error):
                raise UnsafeMaintenance('cannot inspect evidence: ' + str(error))
            for current, directories, names in os.walk(directory, onerror=failed):
                for name in directories + names:
                    if (Path(current) / name).is_symlink():
                        raise UnsafeMaintenance('evidence symlink: ' + str(Path(current) / name))
                for name in names:
                    yield Path(current) / name
        files = [root] if root.is_file() else readable_files(root)
        for path in files:
            if path.suffix not in ('.json', '.jsonl', '.md'):
                continue
            if path.is_symlink():
                raise UnsafeMaintenance('evidence symlink: ' + str(path))
            if path.is_file():
                # Some fault fixtures intentionally contain invalid JSON. Raw text
                # still conservatively protects every runtime mentioned by them.
                text.append(path.read_text())
    return '\n'.join(text)


def criterion(path):
    directory = path / 'source/chainbench-out/web-ui-acceptance'
    if not directory.is_dir():
        return None
    names = [p.name for p in directory.iterdir()]
    return names[0] if len(names) == 1 and re.fullmatch(r'WEB-(0[1-9]|1[0-4])', names[0]) else None


def cleanup_plan(root, repository, created_after, extras=(), now=None):
    safe_tree(root)
    if root == repository or root.is_relative_to(repository) or repository.is_relative_to(root):
        raise UnsafeMaintenance('runtime root must be separate from the repository')
    now = time.time() if now is None else now
    entries = sorted(p for p in root.iterdir() if p.is_dir() or p.is_symlink())
    references = evidence_text(repository, entries, extras)
    state = inventory()
    latest = {}
    for path in entries:
        if path.is_symlink() or not path.name.startswith('recheck-'):
            continue
        c = criterion(path)
        birth = getattr(path.stat(), 'st_birthtime', path.stat().st_ctime)
        if c:
            latest[c] = max(birth, latest.get(c, birth))
    plan = []
    for path in entries:
        reason = None
        if path.is_symlink():
            reason = 'symlink'
        elif path.name in references:
            reason = 'named by evidence'
        elif not (path.name.startswith('recheck-') or re.fullmatch(r'[0-9a-f]{8}-[0-9a-f-]{27}', path.name)):
            reason = 'unknown runtime name'
        elif getattr(path.stat(), 'st_birthtime', path.stat().st_ctime) < created_after:
            reason = 'older than this session'
        elif active(path, state):
            reason = 'active runtime'
        else:
            try:
                if newest_mtime(path) > now - 3600:
                    reason = 'changed in the last hour'
            except UnsafeMaintenance as error:
                reason = str(error)
        if reason is None and path.name.startswith('recheck-'):
            c = criterion(path)
            if c is None:
                reason = 'unknown recheck criterion'
            elif getattr(path.stat(), 'st_birthtime', path.stat().st_ctime) >= latest[c]:
                reason = 'latest recheck of ' + c
        plan.append((path, reason))
    return plan


def cleanup(root, repository, created_after, extras=(), delete=False):
    plan = cleanup_plan(root, repository, created_after, extras)
    for path, reason in plan:
        if reason is None and delete:
            before = path.stat()
            # Inventory and references can change after a dry run or between deletions.
            fresh = dict(cleanup_plan(root, repository, created_after, extras))
            current = path.stat()
            if fresh.get(path, 'disappeared') is not None or (before.st_dev, before.st_ino) != (current.st_dev, current.st_ino):
                raise UnsafeMaintenance('runtime changed before deletion: ' + str(path))
            shutil.rmtree(path)
        print(json.dumps({'path': str(path), 'action': 'keep' if reason else ('deleted' if delete else 'would-delete'), 'reason': reason}))
    return plan


def digest(path):
    value = hashlib.sha256()
    with path.open('rb') as source:
        for chunk in iter(lambda: source.read(1024 * 1024), b''):
            value.update(chunk)
    return value.hexdigest()


def wal_candidate(path, now=None):
    now = time.time() if now is None else now
    if path.is_symlink() or not path.is_file():
        return False
    if [path.parent.name, path.parent.parent.name, path.parent.parent.parent.name] != ['wal', 'member', 'etcd']:
        return False
    if path.name != '0.tmp' and path.suffix != '.wal':
        return False
    s = path.stat()
    return s.st_nlink == 1 and s.st_size >= 40 * 1024**2 and s.st_mtime <= now - 3600 and not getattr(s, 'st_flags', 0) & getattr(stat, 'UF_COMPRESSED', 0)


def metadata(path):
    s = path.stat()
    names = subprocess.run(['/usr/bin/xattr', str(path)], capture_output=True, text=True, check=True).stdout.splitlines()
    attrs = {name: subprocess.run(['/usr/bin/xattr', '-px', name, str(path)], capture_output=True, text=True, check=True).stdout.strip() for name in names if name != 'com.apple.decmpfs'}
    acl = subprocess.run(['/bin/ls', '-lde', str(path)], capture_output=True, text=True, check=True).stdout.splitlines()[1:]
    flags = getattr(s, 'st_flags', 0) & ~getattr(stat, 'UF_COMPRESSED', 0)
    return (stat.S_IMODE(s.st_mode), s.st_uid, s.st_gid, s.st_mtime_ns, attrs, acl, flags)


def compress_wal(root, audit, compress=False):
    safe_tree(root)
    if compress and sys.platform != 'darwin':
        raise UnsafeMaintenance('transparent compression requires macOS')
    for path in root.rglob('*'):
        if not wal_candidate(path):
            continue
        runtime = root / path.relative_to(root).parts[0]
        safe_tree(path.parent)
        if active(runtime, inventory()):
            continue
        if not compress:
            print(json.dumps({'path': str(path), 'action': 'would-compress'}))
            continue
        before = path.stat()
        original = metadata(path)
        sha = digest(path)
        fd, temporary = tempfile.mkstemp(prefix=path.name + '.compress-', dir=path.parent)
        os.close(fd)
        copy = Path(temporary)
        try:
            subprocess.run(['/usr/bin/ditto', '--hfsCompression', '--nocache', str(path), str(copy)], check=True, capture_output=True)
            os.utime(copy, ns=(before.st_atime_ns, before.st_mtime_ns))
            if digest(copy) != sha or metadata(copy) != original:
                raise UnsafeMaintenance('compressed copy changed bytes or metadata')
            if copy.stat().st_blocks >= before.st_blocks:
                continue
            current = path.stat()
            if active(runtime, inventory()) or (current.st_dev, current.st_ino, current.st_size, current.st_mtime_ns, current.st_ctime_ns) != (before.st_dev, before.st_ino, before.st_size, before.st_mtime_ns, before.st_ctime_ns) or digest(path) != sha:
                raise UnsafeMaintenance('WAL became active or changed before replacement')
            if audit.is_symlink():
                raise UnsafeMaintenance('audit path is a symlink')
            safe_tree(audit.parent)
            # Check that the audit can be opened before modifying a proof artifact.
            with audit.open('a') as output:
                os.replace(copy, path)
                if digest(path) != sha or metadata(path) != original:
                    raise UnsafeMaintenance('replacement failed preservation checks')
                record = {'path': str(path), 'sha256': sha, 'bytes': before.st_size,
                          'mode': oct(original[0]), 'uid': original[1], 'gid': original[2],
                          'mtime_ns': original[3], 'savedBytes': (before.st_blocks - path.stat().st_blocks) * 512}
                output.write(json.dumps(record) + '\n')
                output.flush()
                os.fsync(output.fileno())
            print(json.dumps(record))
        finally:
            copy.unlink(missing_ok=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument('--root', type=Path, default=Path(os.environ.get('WEBUI_RUNTIME_ROOT', '/private/tmp/cbui')))
    commands = parser.add_subparsers(dest='command', required=True)
    clean = commands.add_parser('cleanup')
    clean.add_argument('--repository', type=Path, default=Path(__file__).resolve().parents[1])
    clean.add_argument('--created-after', required=True, help='Timezone-aware ISO 8601 session boundary')
    clean.add_argument('--evidence', type=Path, action='append', default=[])
    clean.add_argument('--delete', action='store_true')
    wal = commands.add_parser('compress-wal')
    wal.add_argument('--audit', type=Path, default=Path('/private/tmp/chainbench-wal-compression-audit.jsonl'))
    wal.add_argument('--compress', action='store_true')
    args = parser.parse_args()
    try:
        if args.command == 'cleanup':
            boundary = datetime.datetime.fromisoformat(args.created_after)
            if boundary.tzinfo is None:
                raise UnsafeMaintenance('created-after requires a timezone')
            cleanup(args.root.absolute(), args.repository.absolute(), boundary.timestamp(), args.evidence, args.delete)
        else:
            compress_wal(args.root.absolute(), args.audit, args.compress)
    except (OSError, ValueError, UnsafeMaintenance, subprocess.SubprocessError) as error:
        parser.exit(2, 'maintenance refused: ' + str(error) + '\n')


if __name__ == '__main__':
    main()
