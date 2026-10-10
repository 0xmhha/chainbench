"""Validate fresh WEB-11 physical resource conflict acceptance."""
from evidence import digest  # capture.py publishes evidence digests through this module
from proof_acceptance import verify_proofs

REQUIRED = ['alias-path-port-conflict-with-owner', 'local-ssh-alias-same-physical', 'process-and-binary-conflict', 'independent-resources-parallel', 'test-internal-vs-external-control', 'cleanup-releases-claims']
PROOFS = {
    'resource-parallel': ('run_resource_parallel.py', ['physical-conflict.png']),
    'resource-holds': ('run_resource_holds.py', ['resource-conflict.png']),
    'ssh-jobs': ('run_ssh_jobs.py', ['ssh-plan.png']),
}


def verify(directory):
    return verify_proofs(directory, 'WEB-11', REQUIRED, PROOFS)


if __name__ == '__main__':
    import sys
    errors = verify(sys.argv[1])
    if errors:
        print('\n'.join(errors), file=sys.stderr)
        sys.exit(1)
    print('WEB-11 PASS')
