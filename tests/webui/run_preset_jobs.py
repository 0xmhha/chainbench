"""Saved preset revisions applied to owned native chain setup."""
import hashlib
import json
from pathlib import Path
from run_test_jobs import main
from runtime_contract import source_digest

if __name__ == "__main__":
    before = source_digest(Path.cwd())
    main(browser_script="browser_preset_jobs.mjs", proof_name="preset-jobs")
    if source_digest(Path.cwd()) != before:
        raise RuntimeError('source changed during preset job verification')
    path = Path('chainbench-out/web-ui-development/preset-jobs/receipt.json')
    receipt = json.loads(path.read_text())
    receipt['sourceDigest'] = before
    receipt['dashboardSHA256'] = hashlib.sha256((Path(receipt['runtime']) / 'dashboard').read_bytes()).hexdigest()
    path.write_text(json.dumps(receipt, indent=2))
