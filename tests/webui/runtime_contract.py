"""Trusted fixture paths and immutable source identity for live verification."""
import hashlib
import os
from pathlib import Path
import shutil
import subprocess

EXCLUDED = {'.git', '.ouroboros', '.codex', '.agents', '.aws', '.venv', 'venv', 'chainbench-out', 'bin', 'node_modules', '__pycache__', '.DS_Store'}
SUFFIXES = {'.go', '.svelte', '.js', '.mjs', '.cjs', '.json', '.yaml', '.yml', '.py', '.sh', '.txt', '.md', '.html', '.css', '.mod', '.sum'}


def runtime_root():
    return Path(os.environ.get('WEBUI_RUNTIME_ROOT', '/private/tmp/chainbench-web-ui-e2e')).resolve()


def base_commit():
    return os.environ.get('WEBUI_BASE_COMMIT') or subprocess.check_output(['git', 'rev-parse', 'HEAD'], text=True).strip()


def source_digest(root):
    root = Path(root).resolve()
    digest = hashlib.sha256()
    for directory, dirs, names in os.walk(root):
        dirs[:] = sorted(d for d in dirs if d not in EXCLUDED)
        for name in sorted(names):
            path = Path(directory) / name
            if path.is_symlink():
                raise ValueError('source symlinks cannot be verified: ' + str(path.relative_to(root)))
            if path.suffix not in SUFFIXES and name not in {'Makefile', 'Dockerfile'}:
                continue
            digest.update(path.relative_to(root).as_posix().encode() + b'\0')
            digest.update(hashlib.sha256(path.read_bytes()).digest())
    return digest.hexdigest()


def stage_sources(workspace, runtime):
    workspace = Path(workspace).resolve()
    source = Path(runtime) / 'source'
    def ignored(directory, names):
        # Installed dependencies are needed for offline browser/build execution,
        # but VCS, published receipts and orchestration state are not source.
        excluded = {'__pycache__', '.DS_Store'}
        if Path(directory).resolve() == workspace:
            excluded |= EXCLUDED - {'node_modules'}
        return excluded.intersection(names)
    shutil.copytree(workspace, source, ignore=ignored, symlinks=True)
    return source
