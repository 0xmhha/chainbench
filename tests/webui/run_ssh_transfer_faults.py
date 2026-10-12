"""Interrupt an actual copied native binary on an exclusively owned SSH host."""
import hashlib
import json
import os
from pathlib import Path

from run_binary_replacement import replacement_input
from run_ssh_jobs import main
from runtime_contract import source_digest
from ssh_fault_gate import prepare_ssh_gate


def fault_inputs(runtime, source_keys, output):
    extra = replacement_input(runtime, source_keys, output)
    mode = os.environ.get('WEBUI_SSH_FAULT_MODE', 'disconnect')
    if mode not in {'disconnect', 'credential-revoke'}:
        raise ValueError('select an owned SSH disconnection or credential revocation')
    gate = prepare_ssh_gate(runtime, extra['replacementSHA256'])
    extra.update({'sshGatePath': str(gate), 'sshFaultMode': mode})
    return extra


if __name__ == '__main__':
    before = source_digest(Path.cwd())
    mode = os.environ.get('WEBUI_SSH_FAULT_MODE', 'disconnect')
    name = 'ssh-transfer-' + mode
    main(browser_script='browser_ssh_binary_replacement.mjs', proof_name=name,
         fixture_inputs=fault_inputs)
    if source_digest(Path.cwd()) != before:
        raise RuntimeError('source changed during real SSH transfer interruption')
    path = Path('chainbench-out/web-ui-development') / name / 'receipt.json'
    receipt = json.loads(path.read_text())
    receipt['sourceDigest'] = before
    receipt['dashboardSHA256'] = hashlib.sha256((Path(receipt['runtime']) / 'dashboard').read_bytes()).hexdigest()
    path.write_text(json.dumps(receipt, indent=2))
