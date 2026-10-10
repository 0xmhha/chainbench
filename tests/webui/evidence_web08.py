"""Validate fresh WEB-08 identity, coverage, native proofs and digests."""
from evidence import digest  # capture.py publishes evidence digests through this module
from proof_acceptance import verify_proofs

REQUIRED = ['native-state-metric-charts', 'time-linked-logs', 'browser-close-logout-continuity', 'reconnect-snapshot-restore', 'sse-gap-drop-stale', 'collection-failure-reporting']
# Native proof script and the observation files each scenario cites.
PROOFS = {
    'monitoring': ('run_monitoring.py', ['metrics-before-restart.json', 'metrics-after-restart.json', 'monitoring-desktop.png', 'monitoring-mobile.png']),
    'snapshot-restore': ('run_snapshot_restore.py', ['observations-before-restart.json', 'observations-after-restart.json', 'snapshot-desktop.png']),
    'ssh-log-collection': ('run_ssh_log_collection.py', ['logs-after-revocation.json', 'ssh-log-collection.png']),
    'logout-continuity': ('run_logout_continuity.py', ['stream-disconnected.png', 'continuity-mobile.png']),
}


def verify(directory):
    return verify_proofs(directory, 'WEB-08', REQUIRED, PROOFS)


if __name__ == '__main__':
    import sys
    errors = verify(sys.argv[1])
    if errors:
        print('\n'.join(errors), file=sys.stderr)
        sys.exit(1)
    print('WEB-08 PASS')
