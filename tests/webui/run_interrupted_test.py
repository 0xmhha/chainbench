"""Restart a real dashboard during an owned native test, without auto recovery."""
import hashlib
import json
from pathlib import Path
from run_test_jobs import main
from runtime_contract import source_digest

if __name__ == "__main__":
    before = source_digest(Path.cwd())
    main(browser_script="browser_interrupted_test.mjs", proof_name="interrupted-test", restart=True)
    if source_digest(Path.cwd()) != before:
        raise RuntimeError('source changed during interrupted test verification')
    path = Path('chainbench-out/web-ui-development/interrupted-test/receipt.json')
    receipt = json.loads(path.read_text())
    receipt['sourceDigest'] = before
    receipt['dashboardSHA256'] = hashlib.sha256((Path(receipt['runtime']) / 'dashboard').read_bytes()).hexdigest()
    path.write_text(json.dumps(receipt, indent=2))
