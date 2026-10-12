"""Fresh WEB-13 acceptance: retention, cancellation, cleanup and revocation on
real local and SSH native nodes. Each proof reruns inside this invocation."""
import sys
from evidence_web13 import PROOFS, REQUIRED, verify
from proof_acceptance import assertion, run

CHECKS = ['go test -race -count=1 ./internal/app ./internal/dashboard -run "TestWebJobs|TestWebJobCancel|TestCredentialLease|TestWebRemoteLogs"']
SOURCES = ('internal/app/web_jobs.go', 'internal/app/web_job_execution.go', 'internal/app/web_chain_engine.go', 'internal/app/deployment_job_credentials.go', 'internal/dashboard/jobs.go', 'internal/dashboard/security.go', 'web/src/JobPanel.svelte', 'tests/webui/*web13*', 'tests/webui/proof_acceptance.py', 'tests/webui/browser_cancel_retention.mjs', 'tests/webui/browser_ssh_cleanup.mjs', 'tests/webui/browser_ssh_binary_replacement.mjs', 'tests/webui/run_ssh_transfer_faults.py')


def scenarios(receipts, output):
    """Map proof receipts to the WEB-13 observations of web-ui-acceptance.md."""
    c, s, r = receipts['cancel-retention'], receipts['ssh-cleanup'], receipts['ssh-transfer-credential-revoke']
    notice = (output / 'cancel-retention-attach-notice.txt').read_text()
    return [
        {'id': 'normal-end-retain-and-cleanup', 'mode': 'team', 'transport': 'local+SSH', 'ownership': 'owned',
         'assertions': [assertion('a normal end with default retention keeps four live nodes', c['normalEndRetain']['disposition'] == 'retained'),
                        assertion('a normal end with selected cleanup removes the local composition', c['cleanupRetry']['disposition'] == 'cleaned'),
                        assertion('an SSH test run with selected cleanup stops and removes the remote nodes', s['job']['nodeDisposition'] == 'cleaned' and s['remoteNodesStartedAndRemoved'])],
         'artifacts': ['cancel-retention-receipt.json', 'ssh-cleanup-receipt.json']},
        {'id': 'user-cancel-retain-and-cleanup', 'mode': 'team', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('cancel with default retention keeps the record and live nodes', c['userCancelRetain']['disposition'] == 'retained' and c['userCancelRetain']['liveNodes'] == 4),
                        assertion('cancel with selected cleanup stops and removes the nodes', c['adminCancelCleanup']['disposition'] == 'cleaned')],
         'artifacts': ['cancel-retention-receipt.json']},
        {'id': 'executor-and-admin-cancel', 'mode': 'team', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('another operator cannot cancel the job', c['otherOperatorCancelRefused'] == 403),
                        assertion('an administrator can cancel another user\'s job', bool(c['adminCancelCleanup']['id']))],
         'artifacts': ['cancel-retention-receipt.json']},
        {'id': 'cleanup-failure-reported', 'mode': 'team', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('a failed cleanup is reported with unresolved resources', c['cleanupFailure']['unresolved'] > 0),
                        assertion('a later explicit cleanup removes what remained', c['cleanupRetry']['disposition'] == 'cleaned')],
         'artifacts': ['cancel-retention-receipt.json']},
        {'id': 'revocation-retains-and-blocks-access', 'mode': 'team', 'transport': 'local+SSH', 'ownership': 'owned',
         'assertions': [assertion('account deactivation cancels with retention and blocks the account', c['accountRevocation']['cancelReason'] == 'account_revoked' and c['accountRevocation']['disposition'] == 'retained'),
                        assertion('credential revocation during a real SSH copy keeps processes, record and the completed copy', r['actualTransferFault'] == 'credential-revoke' and r['oldProcessesAndRecordRetained'] and r['completedCopyRetained']),
                        assertion('no automatic resume; only an explicit new job retries', r['explicitRetryWithoutAutomaticResume'])],
         'artifacts': ['cancel-retention-receipt.json', 'ssh-transfer-credential-revoke-receipt.json', 'ssh-transfer-credential-revoke-ssh-binary-plan.png']},
        {'id': 'attach-protection', 'mode': 'team', 'transport': 'local', 'ownership': 'attached',
         'assertions': [assertion('composition, control and test plans on an attached network are refused by the API', all(400 <= v < 500 for v in c['attachRefusals'].values()) and len(c['attachRefusals']) == 3),
                        assertion('the job form explains the refusal and disables planning', 'attach' in notice)],
         'artifacts': ['cancel-retention-receipt.json', 'cancel-retention-attach-refused.png', 'cancel-retention-attach-notice.txt']},
    ]


def main():
    return run('WEB-13', PROOFS, REQUIRED, CHECKS, scenarios, SOURCES, verify)


if __name__ == '__main__':
    sys.exit(main())
