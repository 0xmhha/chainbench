"""Fresh WEB-06 acceptance: deployment to an owned SSH target with real native
binaries, and real SSH failures during deployment and transfer. Each proof
reruns inside this invocation."""
import sys
from evidence_web06 import PROOFS, REQUIRED, verify
from proof_acceptance import assertion, run

CHECKS = ['go test -race -count=1 ./internal/core/remote ./internal/app -run "SSH|Remote|CredentialLease|Transfer"']
SOURCES = ('internal/core/remote/*.go', 'internal/app/web_chain_*.go', 'internal/app/deployment_job_credentials.go', 'tests/webui/*web06*', 'tests/webui/proof_acceptance.py', 'tests/webui/owned-ssh-gate.mjs', 'tests/webui/ssh_fault_gate.py', 'tests/webui/browser_ssh_jobs.mjs', 'tests/webui/browser_ssh_deploy_fault.mjs', 'tests/webui/browser_ssh_binary_replacement.mjs', 'tests/webui/run_ssh_jobs.py', 'tests/webui/run_ssh_transfer_faults.py')


def scenarios(receipts, output):
    """Map proof receipts to the WEB-06 observations of web-ui-acceptance.md."""
    j, d, x, r = (receipts[k] for k in ('ssh-jobs', 'ssh-deploy-fault', 'ssh-transfer-disconnect', 'ssh-transfer-credential-revoke'))
    return [
        {'id': 'ssh-native-deployment', 'mode': 'team', 'transport': 'SSH', 'ownership': 'owned',
         'assertions': [assertion('four native nodes are deployed to the owned SSH target through the job form', j['nativeNodes'] == 4),
                        assertion('the native binary is uploaded separately and verified on the target', j['separateBinaryUpload'] == 'passed')],
         'artifacts': ['ssh-jobs-receipt.json', 'ssh-jobs-ssh-plan.png', 'ssh-jobs-ssh-jobs.png']},
        {'id': 'checksum-database-rpc-pid', 'mode': 'team', 'transport': 'SSH', 'ownership': 'owned',
         'assertions': [assertion('remote native databases hold the declared genesis', j['nativeDatabaseCheck'] == 'passed' and d['nativeDatabaseCheck'] == 'passed'),
                        assertion('the redeployed SSH node serves its chain id over RPC with recorded PIDs', d['explicitRedeploy']['chainId'] == '0x205c'),
                        assertion('no private SSH key is stored in plain text', j['privateSSHKeyPlaintextScan'] == 'passed')],
         'artifacts': ['ssh-jobs-receipt.json', 'ssh-deploy-fault-receipt.json']},
        {'id': 'mid-deployment-ssh-failure', 'mode': 'team', 'transport': 'SSH', 'ownership': 'owned',
         'assertions': [assertion('cutting the SSH connection after the binary upload fails the deployment', d['fault']['state'] == 'failed'),
                        assertion('completed steps and unfinished remote resources are recorded', d['fault']['partialEffects'] > 0 and d['fault']['unresolved'] > 0),
                        assertion('the completed upload is kept and no node starts', d['fault']['completedUploadRetained'] and d['fault']['noNodeStarted'])],
         'artifacts': ['ssh-deploy-fault-receipt.json', 'ssh-deploy-fault-ssh-deploy-fault.png']},
        {'id': 'mid-transfer-disconnect', 'mode': 'team', 'transport': 'SSH', 'ownership': 'owned',
         'assertions': [assertion('a real SSH disconnect during a binary transfer keeps the completed copy, processes and record', x['actualTransferFault'] == 'disconnect' and x['completedCopyRetained'] and x['oldProcessesAndRecordRetained'])],
         'artifacts': ['ssh-transfer-disconnect-receipt.json', 'ssh-transfer-disconnect-ssh-binary-plan.png']},
        {'id': 'revocation-during-ssh', 'mode': 'team', 'transport': 'SSH', 'ownership': 'owned',
         'assertions': [assertion('credential revocation during a transfer cancels and keeps processes, record and copy', r['actualTransferFault'] == 'credential-revoke' and r['oldProcessesAndRecordRetained'] and r['completedCopyRetained'])],
         'artifacts': ['ssh-transfer-credential-revoke-receipt.json']},
        {'id': 'no-automatic-retry', 'mode': 'team', 'transport': 'SSH', 'ownership': 'owned',
         'assertions': [assertion('a failed deployment is not retried on its own; an explicit new plan completes', d['noAutomaticRetry'] and d['explicitRedeploy']['state'] == 'succeeded'),
                        assertion('interrupted transfers require an explicit new job', x['explicitRetryWithoutAutomaticResume'] and r['explicitRetryWithoutAutomaticResume'])],
         'artifacts': ['ssh-deploy-fault-receipt.json', 'ssh-transfer-disconnect-receipt.json']},
    ]


def main():
    return run('WEB-06', PROOFS, REQUIRED, CHECKS, scenarios, SOURCES, verify)


if __name__ == '__main__':
    sys.exit(main())
