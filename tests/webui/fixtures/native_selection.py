"""Verify the shared, externally selected native binaries without altering them."""
import glob
import hashlib
import json
import os
from pathlib import Path
import platform
import re
import subprocess

CHAINS = ('stablenet', 'wbft', 'wemix')
MARKERS = {'stablenet': 'Anzeon', 'wbft': 'Croissant', 'wemix': 'Wemix'}
DEFAULT_SELECTION = '.ouroboros/web-ui-analysis/selected-binaries/manifest.json'


def binary_digest(path):
    digest = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b''):
            digest.update(chunk)
    return digest.hexdigest()



def go_build_identity(source):
    """Use immutable Go VCS metadata when the legacy version stamp is absent."""
    try:
        info = subprocess.check_output(['go', 'version', '-m', str(source)], text=True)
    except (OSError, subprocess.CalledProcessError) as error:
        raise RuntimeError('selected binary has no verifiable Go build identity') from error
    settings = {}
    for line in info.splitlines():
        fields = line.strip().split('\t')
        if len(fields) == 2 and fields[0] == 'build' and '=' in fields[1]:
            key, value = fields[1].split('=', 1)
            settings[key] = value
    revision = settings.get('vcs.revision', '')
    if settings.get('vcs') != 'git' or settings.get('vcs.modified') != 'false' or not re.fullmatch('[0-9a-f]{40}', revision):
        raise RuntimeError('selected binary needs an unmodified Git revision in Go build metadata')
    return revision, info


def selected_binaries():
    """One verified profile per family; never fall back to legacy host paths."""
    selection = Path(os.environ.get('WEBUI_SELECTED_BINARIES', DEFAULT_SELECTION))
    try:
        document = json.loads(selection.read_text())
    except (OSError, ValueError) as error:
        raise RuntimeError('native binary selection cannot be read') from error
    entries = document.get('binaries') if isinstance(document, dict) else None
    if not isinstance(entries, list) or any(not isinstance(x, dict) for x in entries):
        raise RuntimeError('native selection must list three chain profiles')
    if sorted(x.get('chain', '') for x in entries if isinstance(x.get('chain', ''), str)) != sorted(CHAINS) or len(entries) != 3:
        raise RuntimeError('native selection must contain each supported chain exactly once')
    by_chain = {x['chain']: x for x in entries}
    architecture = {'x86_64': 'amd64', 'aarch64': 'arm64'}.get(platform.machine(), platform.machine())
    result = []
    for chain in CHAINS:
        selected = by_chain[chain]
        if any(not isinstance(selected.get(key), str) or not selected[key] for key in ('binary', 'repository', 'embeddedCommit', 'sha256')):
            raise RuntimeError('native selection profile is incomplete: ' + chain)
        if not re.fullmatch('[0-9a-f]{40}', selected['embeddedCommit']) or not re.fullmatch('[0-9a-f]{64}', selected['sha256']):
            raise RuntimeError('native selection commit or checksum is invalid: ' + chain)
        matches = glob.glob(selected['binary'])
        if len(matches) != 1 or not Path(matches[0]).is_file():
            raise RuntimeError('missing or ambiguous selected native binary: ' + chain)
        source = Path(matches[0]).resolve()
        if binary_digest(source) != selected['sha256']:
            raise RuntimeError('selected native binary checksum stale: ' + chain)
        try:
            config = subprocess.check_output(['git', '-C', selected['repository'], 'show', selected['embeddedCommit'] + ':params/config.go'], text=True)
            version = subprocess.check_output([str(source), 'version'], text=True)
        except (OSError, subprocess.CalledProcessError) as error:
            raise RuntimeError('selected native source/version cannot be verified: ' + chain) from error
        commit = re.search(r'Git Commit: ([0-9a-f]{40})', version)
        build_info = ''
        revision = commit[1] if commit else ''
        identity_source = 'version'
        if not commit:
            revision, build_info = go_build_identity(source)
            identity_source = 'go-build-info'
        if revision != selected['embeddedCommit'] or MARKERS[chain] not in config:
            raise RuntimeError('selected chain/source identity mismatch: ' + chain)
        if 'Operating System: ' + platform.system().lower() not in version or 'Architecture: ' + architecture not in version:
            raise RuntimeError('selected native binary incompatible with host: ' + chain)
        result.append({**selected, 'binary': str(source), 'version': version, 'sourceConfig': config,
                       'sourceConfigSHA256': hashlib.sha256(config.encode()).hexdigest(),
                       'identitySource': identity_source, 'buildInfo': build_info})
    return result
