"""Verify recorded node layout execution in real browser/native test jobs."""
import hashlib
import json
import subprocess
from pathlib import Path
from run_test_jobs import main
from runtime_contract import source_digest

def extend_fixture_keys(runtime, source_keys, output):
    with (output / 'key-extension.log').open('w') as log:
        subprocess.run(['go', 'run', './cmd/chainbench', 'keyring', 'add', '--keyring-dir', str(source_keys), '--count', '1', '--validators', '1', '--with-bls'], check=True, stdout=log, stderr=log)
    return {}

if __name__ == '__main__':
    before = source_digest(Path.cwd())
    main(browser_script='browser_recorded_controls.mjs', proof_name='recorded-controls', fixture_inputs=extend_fixture_keys)
    if source_digest(Path.cwd()) != before:
        raise RuntimeError('source changed during recorded node verification')
    path = Path('chainbench-out/web-ui-development/recorded-controls/receipt.json')
    receipt = json.loads(path.read_text())
    receipt['sourceDigest'] = before
    receipt['dashboardSHA256'] = hashlib.sha256((Path(receipt['runtime']) / 'dashboard').read_bytes()).hexdigest()
    path.write_text(json.dumps(receipt, indent=2))
