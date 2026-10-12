"""Verify reviewed executable replacement over a private owned SSH server."""
import hashlib
import json
import os
from pathlib import Path

from run_binary_replacement import replacement_input
from run_ssh_jobs import main
from runtime_contract import source_digest


if __name__ == '__main__':
    before = source_digest(Path.cwd())
    main(browser_script='browser_ssh_binary_replacement.mjs',
         proof_name='ssh-binary-replacement', fixture_inputs=replacement_input,
         dashboard_binary=os.environ.get('WEBUI_BASELINE_DASHBOARD'))
    if source_digest(Path.cwd()) != before:
        raise RuntimeError('source changed during SSH executable verification')
    path = Path('chainbench-out/web-ui-development/ssh-binary-replacement/receipt.json')
    receipt = json.loads(path.read_text())
    receipt['sourceDigest'] = before
    receipt['dashboardSHA256'] = hashlib.sha256((Path(receipt['runtime']) / 'dashboard').read_bytes()).hexdigest()
    receipt['historicalBaseline'] = bool(os.environ.get('WEBUI_BASELINE_DASHBOARD'))
    path.write_text(json.dumps(receipt, indent=2))
