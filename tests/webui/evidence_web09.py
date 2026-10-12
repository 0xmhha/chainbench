"""Validate fresh WEB-09 history acceptance assembled from native proofs."""
from evidence import digest  # capture.py publishes evidence digests through this module
from proof_acceptance import verify_proofs

REQUIRED = ['search-filter-detail', 'compatible-comparison-and-reasons', 'safe-export', 'retention-across-restart', 'admin-only-deletion', 'protected-active-shared-live', 'partial-deletion-failure', 'run-metric-log-archive']
PROOFS = {
    'history-web09': ('run_history_web09.py', ['history-export.json', 'history-web09-desktop.png', 'history-web09-mobile.png']),
    'history-archive': ('run_history_archive.py', ['history-archive-desktop.png', 'history-archive-mobile.png']),
}


def verify(directory):
    return verify_proofs(directory, 'WEB-09', REQUIRED, PROOFS)


if __name__ == '__main__':
    import sys
    errors = verify(sys.argv[1])
    if errors:
        print('\n'.join(errors), file=sys.stderr)
        sys.exit(1)
    print('WEB-09 PASS')
