"""Fresh native binary surface evidence for preset composition; no node changes."""
from runtime_storage import clone_file
from fixtures.native_selection import selected_binaries
import hashlib
import json
from pathlib import Path
import platform
import re
import subprocess


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def prepare(runtime, output):
    binaries = []
    for selected in selected_binaries():
        chain = selected['chain']
        source = Path(selected['binary'])
        destination = runtime / (chain + '-node')
        clone_file(source, destination)
        version = selected['version']
        commit = selected['embeddedCommit']
        config = selected['sourceConfig']
        if digest(destination) != selected['sha256']:
            raise RuntimeError('selected binary changed while copying: ' + chain)
        catalog = {'stablenet': 'gstable', 'wbft': 'gwbft', 'wemix': 'gwemix'}[chain]
        directory = output / (chain + '-surface')
        subprocess.check_call(['bash', 'scripts/chain-analysis/capture-cli.sh', selected['repository'], str(destination), str(directory)], stdout=subprocess.DEVNULL)
        clone_file(Path('docs/chain-analysis') / catalog / 'cli-graph.md', directory / 'cli-graph.md')
        probe = subprocess.run(['bash', 'scripts/chain-analysis/verify-docs.sh', str(destination), str(directory)], capture_output=True, text=True)
        (directory / 'verify.log').write_text(probe.stdout + probe.stderr)
        if probe.returncode:
            raise RuntimeError('binary surface documentation mismatch: ' + chain)
        contract = json.loads((output / (chain + '-contract.json')).read_text())
        knobs = contract['$defs']['envSpec']['properties']['launch']['additionalProperties']['properties']
        surface = (directory / 'cli-surface.txt').read_text()
        flags = {m[0]: not bool(m[1]) for m in re.findall(r'^\s{4,}(--[a-zA-Z0-9._-]+)(\s+value)?\s{2,}\S', surface, re.M)}
        mappings = []
        for key, schema in knobs.items():
            flag = schema['x-flag']
            if flag not in flags or flags[flag] != (schema['type'] == 'boolean'):
                raise RuntimeError('engine mapping unsupported by selected binary: ' + chain + ':' + key)
            mappings.append({'optionKey': key, 'flag': flag, 'schema': schema, 'selectable': True})
        excluded = [{'flag': flag, 'selectable': False, 'reason': 'No current engine OptionKey mapping'} for flag in sorted(flags) if flag not in {row['flag'] for row in mappings}]
        docs = [{'path': str(path), 'sha256': digest(path)} for path in (Path('docs/chain-analysis') / catalog).glob('*') if path.name in ('cli-surface.txt', 'cli-flags.txt', 'cli-graph.md', 'rpc-metrics-graph.md')]
        if len(docs) != 4:
            raise RuntimeError('missing chain documentation source')
        report = {'chain': chain, 'dialect': selected['dialect'], 'binaryPath': str(destination), 'binarySha256': digest(destination), 'version': version, 'sourceCommit': commit,
                  'identitySource': selected['identitySource'], 'buildInfoSHA256': hashlib.sha256(selected['buildInfo'].encode()).hexdigest() if selected['buildInfo'] else None,
                  'sourceConfigSha256': hashlib.sha256(config.encode()).hexdigest(), 'host': {'os': platform.system(), 'architecture': platform.machine()},
                  'documents': docs, 'mapping': mappings, 'excluded': excluded, 'capturedAt': __import__('datetime').datetime.now(__import__('datetime').timezone.utc).isoformat()}
        (directory / 'mapping.json').write_text(json.dumps(report, indent=2) + '\n')
        binaries.append(report)
    # Distinguish the two identical basenames by embedded commits and family evidence.
    wbft, wemix = [next(b for b in binaries if b['chain'] == c) for c in ('wbft', 'wemix')]
    if wbft['sourceCommit'] == wemix['sourceCommit'] or wbft['dialect'] == wemix['dialect']:
        raise RuntimeError('ambiguous gwemix chain identity')
    return binaries
