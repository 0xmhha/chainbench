"""Owned loopback SSH selected cleanup proof; full WEB-13 acceptance remains open.

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
    main(browser_script='browser_ssh_cleanup.mjs', proof_name='ssh-cleanup',
         dashboard_binary=os.environ.get('WEBUI_DASHBOARD_BINARY'))
    if source_digest(Path.cwd()) != before:
        raise RuntimeError('source changed during SSH cleanup verification')
    path = Path('chainbench-out/web-ui-development/ssh-cleanup/receipt.json')
    receipt = json.loads(path.read_text())
    receipt['sourceDigest'] = before
    receipt['dashboardSHA256'] = hashlib.sha256((Path(receipt['runtime']) / 'dashboard').read_bytes()).hexdigest()
    path.write_text(json.dumps(receipt, indent=2))
