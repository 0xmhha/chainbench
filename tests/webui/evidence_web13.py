"""Validate fresh WEB-13 cancellation, retention, cleanup and revocation acceptance."""
from evidence import digest  # capture.py publishes evidence digests through this module
from proof_acceptance import verify_proofs

REQUIRED = ['normal-end-retain-and-cleanup', 'user-cancel-retain-and-cleanup', 'executor-and-admin-cancel', 'cleanup-failure-reported', 'revocation-retains-and-blocks-access', 'attach-protection']
# The revocation proof copies a real second native build over SSH; the
# verify command must be run with WEBUI_REPLACEMENT_BINARY naming it.
PROOFS = {
    'cancel-retention': ('run_cancel_retention.py', ['attach-refused.png', 'attach-notice.txt']),
    'ssh-cleanup': ('run_ssh_cleanup.py', []),
    'ssh-transfer-credential-revoke': ('run_ssh_transfer_faults.py', ['ssh-binary-plan.png'], {'WEBUI_SSH_FAULT_MODE': 'credential-revoke'}),
}


def verify(directory):
    return verify_proofs(directory, 'WEB-13', REQUIRED, PROOFS)


if __name__ == '__main__':
    import sys
    errors = verify(sys.argv[1])
    if errors:
        print('\n'.join(errors), file=sys.stderr)
        sys.exit(1)
    print('WEB-13 PASS')
