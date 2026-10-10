"""Owned loopback SSH node log collection proof; full WEB-08 acceptance remains open.

WEBUI_DASHBOARD_BINARY runs the same proof against another dashboard build,
for example one built from the previous commit to record the RED result."""
import hashlib
import json
import os
from pathlib import Path
from run_ssh_jobs import main
from runtime_contract import source_digest

if __name__ == '__main__':
    before = source_digest(Path.cwd())
    main(browser_script='browser_ssh_log_collection.mjs', proof_name='ssh-log-collection',
         dashboard_binary=os.environ.get('WEBUI_DASHBOARD_BINARY'))
    if source_digest(Path.cwd()) != before:
        raise RuntimeError('source changed during SSH log collection verification')
    path = Path('chainbench-out/web-ui-development/ssh-log-collection/receipt.json')
    receipt = json.loads(path.read_text())
    receipt['sourceDigest'] = before
    receipt['dashboardSHA256'] = hashlib.sha256((Path(receipt['runtime']) / 'dashboard').read_bytes()).hexdigest()
    path.write_text(json.dumps(receipt, indent=2))
