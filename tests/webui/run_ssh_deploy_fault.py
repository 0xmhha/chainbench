"""Owned loopback SSH proof: the connection carrying the initial node binary
upload is cut after the copy completes; full WEB-06 acceptance remains open."""
import hashlib
import json
from pathlib import Path
from run_ssh_jobs import main
from runtime_contract import source_digest
from ssh_fault_gate import prepare_ssh_gate


def fault_inputs(runtime, source_keys, output):
    base = next(a for a in json.loads((runtime / 'assets.json').read_text()) if a['id'] == 'wbft')
    return {'sshGatePath': str(prepare_ssh_gate(runtime, base['sha256'])), 'baseSHA256': base['sha256']}


if __name__ == '__main__':
    before = source_digest(Path.cwd())
    main(browser_script='browser_ssh_deploy_fault.mjs', proof_name='ssh-deploy-fault', fixture_inputs=fault_inputs)
    if source_digest(Path.cwd()) != before:
        raise RuntimeError('source changed during SSH deployment fault verification')
    path = Path('chainbench-out/web-ui-development/ssh-deploy-fault/receipt.json')
    receipt = json.loads(path.read_text())
    receipt['sourceDigest'] = before
    receipt['dashboardSHA256'] = hashlib.sha256((Path(receipt['runtime']) / 'dashboard').read_bytes()).hexdigest()
    path.write_text(json.dumps(receipt, indent=2))
