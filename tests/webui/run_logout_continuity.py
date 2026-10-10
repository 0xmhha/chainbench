"""Native job continuity across sign-out, browser close and stream loss; full WEB-08 acceptance remains open."""
import hashlib
import json
from pathlib import Path
from run_test_jobs import main
from runtime_contract import source_digest

if __name__ == '__main__':
    before = source_digest(Path.cwd())
    main(browser_script='browser_logout_continuity.mjs', proof_name='logout-continuity', browser_timeout=600)
    if source_digest(Path.cwd()) != before:
        raise RuntimeError('source changed during logout continuity verification')
    path = Path('chainbench-out/web-ui-development/logout-continuity/receipt.json')
    receipt = json.loads(path.read_text())
    receipt['sourceDigest'] = before
    receipt['dashboardSHA256'] = hashlib.sha256((Path(receipt['runtime']) / 'dashboard').read_bytes()).hexdigest()
    path.write_text(json.dumps(receipt, indent=2))
