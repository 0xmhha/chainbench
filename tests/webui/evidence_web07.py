"""Validate fresh WEB-07 owned node control acceptance assembled from native proofs."""
from evidence import digest  # capture.py publishes evidence digests through this module
from proof_acceptance import verify_proofs

REQUIRED = ['owned-start-stop', 'config-replacement-restart', 'binary-replacement-restart', 'non-producer-reset-stopped', 'producer-and-attach-refused', 'cli-semantics-preserved']
# Binary replacement proofs copy a real second native build; run the verify
# command with WEBUI_REPLACEMENT_BINARY naming it.
PROOFS = {
    'recorded-controls': ('run_recorded_controls.py', ['wbft-reset.png', 'wbft-table-plan.png', 'stablenet-reset.png', 'wemix-reset.png']),
    'binary-replacement': ('run_binary_replacement.py', ['binary-plan.png']),
    'ssh-binary-replacement': ('run_ssh_binary_replacement.py', ['ssh-binary-plan.png', 'ssh-binary-permission.png']),
    'cancel-retention': ('run_cancel_retention.py', ['attach-refused.png', 'attach-notice.txt']),
}


def verify(directory):
    return verify_proofs(directory, 'WEB-07', REQUIRED, PROOFS)


if __name__ == '__main__':
    import sys
    errors = verify(sys.argv[1])
    if errors:
        print('\n'.join(errors), file=sys.stderr)
        sys.exit(1)
    print('WEB-07 PASS')
