"""Fresh WEB-12 acceptance: dashboard restarts during a real test and a real
deployment. Each proof reruns inside this invocation."""
import sys
from evidence_web12 import PROOFS, REQUIRED, verify
from proof_acceptance import assertion, run

CHECKS = ['go test -race -count=1 ./internal/app -run "TestWebJobsInterrupted|TestWebJobCancelledAcceptance|TestJobObservationReplays"']
SOURCES = ('internal/app/web_jobs.go', 'internal/app/web_job_execution.go', 'internal/app/web_resource_holds.go', 'internal/app/web_node_observations.go', 'tests/webui/*web12*', 'tests/webui/proof_acceptance.py', 'tests/webui/browser_interrupted_test.mjs', 'tests/webui/browser_deploy_restart.mjs')


def scenarios(receipts, output):
    """Map proof receipts to the WEB-12 observations of web-ui-acceptance.md."""
    t, d = receipts['interrupted-test'], receipts['deploy-restart']
    return [
        {'id': 'restart-marks-running-work-interrupted', 'mode': 'personal', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('a real test job running at restart is interrupted with unknown disposition and unresolved resources', t['interrupted']['state'] == 'interrupted'),
                        assertion('a real deployment running at restart is interrupted and keeps unresolved resources', d['interruptedJob']['state'] == 'interrupted' and d['interruptedJob']['unresolved'] > 0)],
         'artifacts': ['interrupted-test-receipt.json', 'deploy-restart-receipt.json']},
        {'id': 'actual-state-rechecked', 'mode': 'personal', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('nodes left running by the interrupted test are observed running by PID and argv', bool(t['observed'])),
                        assertion('nodes of the interrupted deployment are observed, not assumed', len(d['observedNodes']) == 4)],
         'artifacts': ['interrupted-test-receipt.json', 'interrupted-test-restart-observation.png', 'deploy-restart-receipt.json']},
        {'id': 'records-and-claims-preserved', 'mode': 'personal', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('launch record and node data are unchanged by the restart', bool(t['recordHash']) and len(t['dirStats']) == 4),
                        assertion('interrupted work keeps its physical claims against aliases', t['aliasStillExcluded'] and d['claimsKept'])],
         'artifacts': ['interrupted-test-receipt.json', 'deploy-restart-receipt.json']},
        {'id': 'no-automatic-redeploy-or-reset', 'mode': 'personal', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('idempotent replay returns the interrupted job instead of resuming', t['idempotentReplayDidNotResume']),
                        assertion('no reset or other job ran on its own', t['noAutomaticReset']),
                        assertion('the interrupted deployment did not continue after restart', d['noAutomaticResume'])],
         'artifacts': ['interrupted-test-receipt.json', 'deploy-restart-receipt.json', 'deploy-restart-deploy-interrupted.png']},
        {'id': 'explicit-rerun-new-plan', 'mode': 'personal', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('explicit control and test rerun use new plans and jobs', t['newPlanAndJobForExplicitControl'] and t['explicitNativeTestRerun']),
                        assertion('an explicit new deployment plan completes', d['explicitRerun']['state'] == 'succeeded')],
         'artifacts': ['interrupted-test-receipt.json', 'deploy-restart-receipt.json']},
    ]


def main():
    return run('WEB-12', PROOFS, REQUIRED, CHECKS, scenarios, SOURCES, verify)


if __name__ == '__main__':
    sys.exit(main())
