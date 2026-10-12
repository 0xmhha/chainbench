"""Verify unrecorded node control execution in real browser/native test jobs."""
import hashlib
import json
from pathlib import Path
from run_test_jobs import main
from runtime_contract import source_digest

if __name__ == '__main__':
    before = source_digest(Path.cwd())
    main(browser_script='browser_residual_composition.mjs', proof_name='residual-composition')
    if source_digest(Path.cwd()) != before:
        raise RuntimeError('source changed during unrecorded control verification')
    path = Path('chainbench-out/web-ui-development/residual-composition/receipt.json')
    receipt = json.loads(path.read_text())
    receipt['sourceDigest'] = before
    receipt['dashboardSHA256'] = hashlib.sha256((Path(receipt['runtime']) / 'dashboard').read_bytes()).hexdigest()
    path.write_text(json.dumps(receipt, indent=2))
