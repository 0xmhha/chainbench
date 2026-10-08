"""Verify reviewed non-producer reset in real browser/native test jobs."""
import hashlib
import json
from pathlib import Path
from run_test_jobs import main
from runtime_contract import source_digest

if __name__ == '__main__':
    before = source_digest(Path.cwd())
    main(browser_script='browser_node_reset.mjs', proof_name='node-reset')
    if source_digest(Path.cwd()) != before:
        raise RuntimeError('source changed during native reset verification')
    path = Path('chainbench-out/web-ui-development/node-reset/receipt.json')
    receipt = json.loads(path.read_text())
    receipt['sourceDigest'] = before
    receipt['dashboardSHA256'] = hashlib.sha256((Path(receipt['runtime']) / 'dashboard').read_bytes()).hexdigest()
    path.write_text(json.dumps(receipt, indent=2))
