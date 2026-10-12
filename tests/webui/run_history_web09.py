"""Native history search, comparison, export, protection and retention proof; full WEB-09 acceptance remains open."""
import hashlib
import json
from pathlib import Path
from run_test_jobs import main
from runtime_contract import source_digest

if __name__ == '__main__':
    before = source_digest(Path.cwd())
    main(browser_script='browser_history_web09.mjs', proof_name='history-web09', restart=True, browser_timeout=600)
    if source_digest(Path.cwd()) != before:
        raise RuntimeError('source changed during history verification')
    path = Path('chainbench-out/web-ui-development/history-web09/receipt.json')
    receipt = json.loads(path.read_text())
    receipt['sourceDigest'] = before
    receipt['dashboardSHA256'] = hashlib.sha256((Path(receipt['runtime']) / 'dashboard').read_bytes()).hexdigest()
    path.write_text(json.dumps(receipt, indent=2))
