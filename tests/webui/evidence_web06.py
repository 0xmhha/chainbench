"""Validate fresh WEB-06 SSH deployment acceptance."""
from evidence import digest  # capture.py publishes evidence digests through this module
from proof_acceptance import verify_proofs

REQUIRED = ['ssh-native-deployment', 'checksum-database-rpc-pid', 'mid-deployment-ssh-failure', 'mid-transfer-disconnect', 'revocation-during-ssh', 'no-automatic-retry']
# The transfer proofs copy a real second native build; run the verify command
# with WEBUI_REPLACEMENT_BINARY naming it.
PROOFS = {
    'ssh-jobs': ('run_ssh_jobs.py', ['ssh-plan.png', 'ssh-jobs.png']),
    'ssh-deploy-fault': ('run_ssh_deploy_fault.py', ['ssh-deploy-fault.png']),
    'ssh-transfer-disconnect': ('run_ssh_transfer_faults.py', ['ssh-binary-plan.png'], {'WEBUI_SSH_FAULT_MODE': 'disconnect'}),
    'ssh-transfer-credential-revoke': ('run_ssh_transfer_faults.py', ['ssh-binary-plan.png'], {'WEBUI_SSH_FAULT_MODE': 'credential-revoke'}),
}


def verify(directory):
    return verify_proofs(directory, 'WEB-06', REQUIRED, PROOFS)


if __name__ == '__main__':
    import sys
    errors = verify(sys.argv[1])
    if errors:
        print('\n'.join(errors), file=sys.stderr)
        sys.exit(1)
    print('WEB-06 PASS')
