"""A private tracked-tree view for independent checks; original Git is read-only."""
import hashlib
import os
from pathlib import Path
import subprocess
import tempfile
from runtime_contract import source_digest
from runtime_storage import clone_file

GIT_LOCATIONS = {'GIT_DIR', 'GIT_WORK_TREE', 'GIT_INDEX_FILE', 'GIT_OBJECT_DIRECTORY',
                 'GIT_ALTERNATE_OBJECT_DIRECTORIES', 'GIT_COMMON_DIR', 'GIT_NAMESPACE'}


def git_environment(environment=None):
    values = os.environ if environment is None else environment
    return {key: value for key, value in values.items() if key not in GIT_LOCATIONS}


def stage_git_metadata(workspace, source):
    workspace, source = Path(workspace).resolve(), Path(source).resolve()
    if workspace == source or (source / '.git').exists() or (source / '.git').is_symlink():
        raise ValueError('independent source must have no existing Git view')
    identity = source_digest(workspace)
    if source_digest(source) != identity:
        raise ValueError('Git view requires the same frozen source')
    env = {**git_environment(), 'GIT_OPTIONAL_LOCKS': '0'}
    def git(*args, cwd=workspace):
        result = subprocess.run(['git', *args], cwd=cwd, env=env, text=True, capture_output=True)
        if result.returncode:
            raise ValueError('independent Git metadata could not be prepared')
        return result.stdout.strip()
    head = git('rev-parse', 'HEAD')
    index = Path(git('rev-parse', '--path-format=absolute', '--git-path', 'index'))
    if index.is_symlink() or not index.is_file():
        raise ValueError('original Git index must be a regular file')
    before = index.read_bytes()
    stat = index.stat()
    with tempfile.TemporaryDirectory(prefix='git-view-', dir=source.parent) as raw:
        cloned = Path(raw) / 'view'
        # Shared objects are read through alternates; index, refs and config are
        # private. No worktree is checked out and no user files are overwritten.
        git('-c', 'core.hooksPath=/dev/null', 'clone', '--quiet', '--shared', '--no-checkout', str(workspace), str(cloned))
        git('config', '--local', 'core.hooksPath', '/dev/null', cwd=cloned)
        prepared = cloned / '.git'
        if prepared.is_symlink() or not prepared.is_dir():
            raise ValueError('independent Git clone has no private metadata')
        clone_file(index, prepared / 'index')
        # Asking Git for the original split-index path freshens its mtime.
        # Copy candidate bases without reading the original index through Git,
        # then identify and keep only the required base inside the private clone.
        candidates = []
        for shared in sorted(index.parent.glob('sharedindex.*')):
            if shared.is_symlink() or not shared.is_file():
                raise ValueError('original split index is not a regular file')
            private = prepared / shared.name
            clone_file(shared, private)
            candidates.append(private)
        shared = git('rev-parse', '--shared-index-path', cwd=cloned)
        needed = Path(shared).name if shared else ''
        for private in candidates:
            if private.name != needed:
                private.unlink()
        exclude = Path(git('rev-parse', '--path-format=absolute', '--git-path', 'info/exclude'))
        if exclude.is_file():
            clone_file(exclude, prepared / 'info/exclude')
        if git('rev-parse', 'HEAD', cwd=cloned) != head or git('rev-parse', 'HEAD') != head:
            raise ValueError('Git HEAD changed during independent staging')
        os.replace(prepared, source / '.git')
    private_stat = (source / '.git/index').stat()
    if (private_stat.st_dev, private_stat.st_ino) == (stat.st_dev, stat.st_ino):
        raise ValueError('independent index is not a separate file')
    after = index.stat()
    if index.read_bytes() != before or (after.st_mode, after.st_uid, after.st_gid, after.st_mtime_ns) != (stat.st_mode, stat.st_uid, stat.st_gid, stat.st_mtime_ns):
        raise ValueError('original Git index changed during staging')
    if source_digest(source) != identity or source_digest(workspace) != identity:
        raise ValueError('source changed during Git staging')
    return {'head': head, 'indexSHA256': hashlib.sha256(before).hexdigest(),
            'privateGitDirectory': str(source / '.git'), 'privateIndex': True,
            'objectPolicy': 'read-only alternates to the original object store',
            'sourceDigest': identity}
