"""Space-bounded fixtures; completed proofs retain provenance, not node databases.

Only runtimes created by this module are eligible for automatic reclamation.
Historical unmarked runtimes require explicit conservative maintenance instead.
"""
import datetime
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tempfile
import time
import uuid

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'scripts'))
import webui_storage as maintenance
from runtime_contract import runtime_root

MARKER = '.webui-owned-runtime.json'
MIN_FREE = 20 * 1024**3


def preflight(root):
    if shutil.disk_usage(root).free < MIN_FREE:
        raise maintenance.UnsafeMaintenance('native proof requires at least 20 GiB free; reclaim completed owned data first')


def create_runtime(name=None, prefix=None):
    root = runtime_root(); root.mkdir(parents=True, exist_ok=True)
    maintenance.safe_tree(root)
    preflight(root)
    if prefix:
        runtime = Path(tempfile.mkdtemp(prefix=prefix, dir=root))
    else:
        name = name or str(uuid.uuid4())
        if Path(name).name != name or name in ('.', '..'):
            raise maintenance.UnsafeMaintenance('runtime name must be a direct child')
        runtime = root / name; runtime.mkdir(mode=0o700)
    marker = runtime / MARKER
    with marker.open('x') as output:
        json.dump({'version': 1, 'path': str(runtime), 'policy': 'completed-disposable-data'}, output)
    marker.chmod(0o600)
    return runtime


def clone_file(source, destination):
    """APFS copy-on-write, with independent inode; never hard-link live inputs."""
    source, destination = Path(source), Path(destination)
    if destination.is_dir():
        destination = destination / source.name
    if sys.platform == 'darwin' and source.stat().st_size >= 1024**2:
        fd, name = tempfile.mkstemp(prefix='.clone-', dir=destination.parent)
        os.close(fd)
        temporary = Path(name)
        try:
            clone = subprocess.run(['/bin/cp', '-c', '-p', str(source), str(temporary)], capture_output=True, text=True)
            if clone.returncode == 0:
                os.replace(temporary, destination)
                return str(destination)
        finally:
            temporary.unlink(missing_ok=True)
    return shutil.copy2(source, destination)


def validate_runtime(runtime):
    runtime = Path(runtime).absolute()
    if runtime.parent != runtime_root():
        raise maintenance.UnsafeMaintenance('runtime is outside the owned root')
    maintenance.safe_tree(runtime)
    marker = runtime / MARKER
    if marker.is_symlink() or not marker.is_file():
        raise maintenance.UnsafeMaintenance('unmarked runtime is not disposable')
    owned = json.loads(marker.read_text())
    if owned != {'version': 1, 'path': str(runtime), 'policy': 'completed-disposable-data'}:
        raise maintenance.UnsafeMaintenance('runtime ownership mismatch')
    return runtime


class RuntimeBusy(maintenance.UnsafeMaintenance):
    """A fresh inventory saw activity; callers may wait, never force deletion."""


def reclaim_runtime(runtime, references=()):
    runtime = validate_runtime(runtime)
    if maintenance.active(runtime, maintenance.inventory()):
        raise RuntimeBusy('runtime still has a process or open file')
    candidates = []
    # These are node databases, not logs, configuration, process ledgers,
    # snapshots exported by the application, binaries, source or receipts.
    for current, directories, _ in os.walk(runtime, followlinks=False):
        for name in list(directories):
            path = Path(current) / name
            relative = path.relative_to(runtime)
            dependency = relative.as_posix() in ('source/node_modules', 'source/web/node_modules', 'source/tests/webui/node_modules')
            if name not in ('chaindata', 'lightchaindata', 'etcd', 'triecache') and not dependency:
                continue
            if not dependency and (set(relative.parts) & {'source', 'snapshots', 'sessions', 'logs', 'artifacts',
                                                         'reports', 'diagnostics', 'inputs', 'proofs'}
                                   or not any(re.fullmatch(r'node[0-9]+', part) for part in relative.parts)):
                continue
            directories.remove(name)
            maintenance.safe_tree(path)
            if not dependency:
                maintenance.newest_mtime(path)  # Reject links inside node data.
            if any(str(path) == ref or str(path) + '/' in ref for ref in references):
                continue
            candidates.append(path)
    audit = runtime / 'storage-audit.jsonl'
    if audit.is_symlink():
        raise maintenance.UnsafeMaintenance('audit path is a symlink')
    reclaimed = []
    def record_audit(record):
        with audit.open('a') as output:
            os.chmod(audit, 0o600)
            output.write(json.dumps(record) + '\n')
            output.flush(); os.fsync(output.fileno())
    for path in candidates:
        maintenance.safe_tree(path)
        if path.name != 'node_modules':
            maintenance.newest_mtime(path)
        allocated = sum(p.lstat().st_blocks * 512 for p in path.rglob('*') if p.is_file() and not p.is_symlink())
        record_audit({'path': str(path), 'event': 'reclaim-start', 'allocatedBytes': allocated})
        # Close our own audit descriptor before inventorying the runtime again.
        if maintenance.active(runtime, maintenance.inventory()):
            raise RuntimeBusy('runtime became active before reclamation')
        shutil.rmtree(path)
        record = {'path': str(path), 'event': 'reclaimed', 'allocatedBytes': allocated,
                  'finishedAt': datetime.datetime.now(datetime.timezone.utc).isoformat()}
        record_audit(record)
        reclaimed.append(record)
    return reclaimed


def reclaim_after_shutdown(runtime, directory, timeout=10):
    """Finalize both passed and failed proofs after their owned processes exit.

    The caller's finally block closes its servers/logs first. Partial evidence,
    logs and source inputs survive; an explicitly cited database survives too.
    No process is killed here, and an unavailable inventory refuses deletion.
    """
    runtime = validate_runtime(runtime)
    deadline = time.monotonic() + timeout
    while True:
        if maintenance.active(runtime, maintenance.inventory()):
            if time.monotonic() >= deadline:
                raise RuntimeBusy('runtime remains active after shutdown')
            time.sleep(.1)
            continue
        try:
            return reclaim_evidence(directory, (runtime,))
        except RuntimeBusy:
            # A process or descriptor can become visible between inventories.
            # Retry the entire safe operation with fresh evidence references;
            # ownership, links and unavailable inventories still fail immediately.
            if time.monotonic() >= deadline:
                raise
            time.sleep(.1)


def reclaim_evidence(directory, owned_runtimes=()):
    # A failed invocation may have no success receipt. Its caller still knows
    # exactly which runtime it created; validate that explicit ownership first.
    runtimes = {validate_runtime(runtime) for runtime in owned_runtimes}
    references = []
    directory = Path(directory)
    maintenance.safe_tree(directory)
    def visit(value):
        if isinstance(value, dict):
            runtime = value.get('runtime')
            if isinstance(runtime, str) and Path(runtime).parent == runtime_root():
                runtimes.add(Path(runtime))
            # Criterion fixtures use their invocation UUID as the runtime name.
            # A partial failure receipt can precede the full runtime field.
            invocation = value.get('invocationId')
            if isinstance(invocation, str) and re.fullmatch(r'[0-9a-f]{8}-[0-9a-f-]{27}', invocation):
                candidate = runtime_root() / invocation
                if (candidate / MARKER).is_file():
                    runtimes.add(candidate)
            for child in value.values():
                visit(child)
        elif isinstance(value, list):
            for child in value:
                visit(child)
        elif isinstance(value, str):
            references.append(value)
            if value and '\0' not in value and not value.startswith('/'):
                references.append(str(Path(value).absolute()))
    for path in directory.rglob('*'):
        if path.is_symlink():
            raise maintenance.UnsafeMaintenance('evidence contains a symlink')
        if not path.is_file() or path.suffix not in ('.json', '.jsonl'):
            continue
        raw = path.read_text()
        try:
            visit(json.loads(raw))
        except json.JSONDecodeError:
            references.append(raw)  # Malformed-input fixtures are still protected.
    records = []
    for runtime in sorted(runtimes):
        if not (runtime / MARKER).exists():
            continue  # Historical evidence is never opted into a new policy.
        records.extend(reclaim_runtime(runtime, references))
    return records
