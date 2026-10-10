"""Native dashboard restart during a real deployment, without automatic recovery; full WEB-12 acceptance remains open."""
import hashlib
import json
from pathlib import Path
from run_test_jobs import main
from runtime_contract import source_digest

if __name__ == '__main__':
    before = source_digest(Path.cwd())
    main(browser_script='browser_deploy_restart.mjs', proof_name='deploy-restart', restart=True, browser_timeout=600)
    if source_digest(Path.cwd()) != before:
        raise RuntimeError('source changed during deployment restart verification')
    path = Path('chainbench-out/web-ui-development/deploy-restart/receipt.json')
    receipt = json.loads(path.read_text())
    receipt['sourceDigest'] = before
    receipt['dashboardSHA256'] = hashlib.sha256((Path(receipt['runtime']) / 'dashboard').read_bytes()).hexdigest()
    path.write_text(json.dumps(receipt, indent=2))
