"""Validate fresh WEB-12 restart and residual-state acceptance."""
from evidence import digest  # capture.py publishes evidence digests through this module
from proof_acceptance import verify_proofs

REQUIRED = ['restart-marks-running-work-interrupted', 'actual-state-rechecked', 'records-and-claims-preserved', 'no-automatic-redeploy-or-reset', 'explicit-rerun-new-plan']
PROOFS = {
    'interrupted-test': ('run_interrupted_test.py', ['restart-observation.png']),
    'deploy-restart': ('run_deploy_restart.py', ['deploy-interrupted.png']),
}


def verify(directory):
    return verify_proofs(directory, 'WEB-12', REQUIRED, PROOFS)


if __name__ == '__main__':
    import sys
    errors = verify(sys.argv[1])
    if errors:
        print('\n'.join(errors), file=sys.stderr)
        sys.exit(1)
    print('WEB-12 PASS')
