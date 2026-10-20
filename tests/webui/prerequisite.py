"""Run non-live checks without changing frozen source or opening native gates."""
import hashlib
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
from runtime_contract import runtime_root
from runtime_storage import clone_file

LIVE_GATES = frozenset({
    'GSTABLE_BIN', 'GWBFT_BIN', 'GWEMIX_BIN', 'CHAINBENCH_DOCKER_SERVERS',
    'CHAINBENCH_E2E_FROM_BIN', 'CHAINBENCH_E2E_TO_BIN', 'CHAINBENCH_W3_PROOF',
    'CHAINBENCH_W4_PROOF', 'CHAINBENCH_HANDOFF_STORAGE_PROOF', 'CHAINBENCH_HANDOFF_REATTACH_PROOF',
})
SPA_BUILD = 'npm --prefix web run build'


def check_environment(environment=None):
    source = os.environ if environment is None else environment
    return {key: value for key, value in source.items() if key not in LIVE_GATES}


def spa_identity(directory):
    directory = Path(directory)
    if directory.is_symlink():
        raise ValueError('embedded SPA cannot be a symlink')
    result = {}
    for path in sorted(directory.rglob('*')):
        if path.is_symlink():
            raise ValueError('embedded SPA cannot contain symlinks')
        if path.is_file():
            result[path.relative_to(directory).as_posix()] = hashlib.sha256(path.read_bytes()).hexdigest()
    return result


def run_check(command, **kwargs):
    """Unit checks stay in their checkout; SPA rebuilds use disposable copies.

    A staged rebuild must reproduce the checked-in embedded bytes. A mismatch
    fails the check and never repairs the checkout during an acceptance run.
    Actual native proof subprocesses and WEB-14's explicit race use their own
    environments, outside this non-live check boundary.
    """
    checkout = Path(kwargs.pop('cwd', None) or Path.cwd()).resolve()
    kwargs['env'] = check_environment(kwargs.pop('env', None))
    if command != SPA_BUILD:
        return subprocess.run(command, cwd=checkout, **kwargs)
    expected = spa_identity(checkout / 'internal/dashboard/spa')
    runtime_root().mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='spa-check-', dir=runtime_root()) as raw:
        staged = Path(raw)
        shutil.copytree(checkout / 'web', staged / 'web', symlinks=True,
                        copy_function=clone_file, ignore=shutil.ignore_patterns('__pycache__', '.DS_Store'))
        result = subprocess.run(command, cwd=staged, **kwargs)
        if result.returncode == 0 and (not expected or spa_identity(staged / 'internal/dashboard/spa') != expected):
            result.returncode = 1
            message = '\nStaged SPA build differs from frozen embedded assets; rebuild and review before acceptance.\n'
            if kwargs.get('stderr') == subprocess.STDOUT:
                result.stdout = (result.stdout or '') + message
            else:
                result.stderr = (result.stderr or '') + message
        return result
