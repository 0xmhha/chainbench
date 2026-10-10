"""Prove a registered native binary replacement in one owned browser fixture.

WEBUI_REPLACEMENT_BINARY must name a genuine second WBFT native build. This
development proof never substitutes simulated processes or awards a criterion.
"""
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess

from run_test_jobs import main
from runtime_contract import source_digest


def replacement_input(runtime, source_keys, output):
    source = Path(os.environ['WEBUI_REPLACEMENT_BINARY']).resolve(strict=True)
    assets = json.loads((runtime / 'assets.json').read_text())
    base = next(asset for asset in assets if asset['id'] == 'wbft')
    version = subprocess.check_output([str(source), 'version'], text=True)
    if 'Git Commit: ' + base['commit'] not in version:
        raise RuntimeError('replacement must carry the fixture source commit')
    digest = hashlib.sha256(source.read_bytes()).hexdigest()
    if digest == base['sha256']:
        raise RuntimeError('replacement fixture must have distinct executable bytes')
    candidate = runtime / 'replacement-wbft'
    shutil.copy2(source, candidate)
    (output / 'replacement-version.txt').write_text(version)
    return {'replacementPath': str(candidate), 'replacementSHA256': digest,
            'replacementCommit': base['commit']}


if __name__ == '__main__':
    before = source_digest(Path.cwd())
    main(browser_script='browser_binary_replacement.mjs', proof_name='binary-replacement',
         fixture_inputs=replacement_input, restart=True, browser_timeout=600)
    if source_digest(Path.cwd()) != before:
        raise RuntimeError('source changed during binary replacement verification')
    path = Path('chainbench-out/web-ui-development/binary-replacement/receipt.json')
    receipt = json.loads(path.read_text())
    receipt['sourceDigest'] = before
    receipt['dashboardSHA256'] = hashlib.sha256((Path(receipt['runtime']) / 'dashboard').read_bytes()).hexdigest()
    path.write_text(json.dumps(receipt, indent=2))
