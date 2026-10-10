"""Fresh WEB-11 acceptance: physical host, path, port, process and binary
conflicts across workspace aliases, and independent work in parallel, on real
local and SSH native nodes. Each proof reruns inside this invocation."""
import sys
from evidence_web11 import PROOFS, REQUIRED, verify
from proof_acceptance import assertion, run

CHECKS = ['go test -race -count=1 ./internal/app -run "TestWebPhysical|TestWebExecutableClaims|TestWebJobsDetached|TestWebJobsInterruptedClaims"']
SOURCES = ('internal/app/web_resource_holds.go', 'internal/app/web_job_contract.go', 'internal/app/web_jobs.go', 'internal/app/web_chain_engine.go', 'web/src/JobPanel.svelte', 'tests/webui/*web11*', 'tests/webui/proof_acceptance.py', 'tests/webui/browser_resource_parallel.mjs', 'tests/webui/browser_resource_holds.mjs', 'tests/webui/browser_ssh_jobs.mjs')


def scenarios(receipts, output):
    """Map proof receipts to the WEB-11 observations of web-ui-acceptance.md."""
    p, h, s = receipts['resource-parallel'], receipts['resource-holds'], receipts['ssh-jobs']
    return [
        {'id': 'alias-path-port-conflict-with-owner', 'mode': 'team', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('another workspace naming the same data root and ports is refused with the owning job, workspace and executor', h['retainedAliasRejected'] and h['conflictOwnerVisible'] and p['externalRefused']['aliasConflicts'] > 0),
                        assertion('the form shows the owner and offers no execution', h['uiBlockedUntilCleanup'])],
         'artifacts': ['resource-holds-receipt.json', 'resource-holds-resource-conflict.png', 'resource-parallel-physical-conflict.png']},
        {'id': 'local-ssh-alias-same-physical', 'mode': 'team', 'transport': 'local+SSH', 'ownership': 'owned',
         'assertions': [assertion('a local spelling of the SSH server resolves to the same physical host and root and is refused', bool(s['physicalLocalSSHAlias']))],
         'artifacts': ['ssh-jobs-receipt.json', 'ssh-jobs-ssh-plan.png']},
        {'id': 'process-and-binary-conflict', 'mode': 'team', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('the same node binary on the machine is refused before execution with its owner', bool(p['sameBinaryRefused']['executable'])),
                        assertion('a reused PID that is not the recorded process cannot be controlled', h['reusedPidControlRejected'])],
         'artifacts': ['resource-parallel-receipt.json', 'resource-holds-receipt.json']},
        {'id': 'independent-resources-parallel', 'mode': 'team', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('a workspace with its own root, ports and binary completes while the other test keeps running', p['parallel']['chains'] == ['wbft', 'stablenet'])],
         'artifacts': ['resource-parallel-receipt.json']},
        {'id': 'test-internal-vs-external-control', 'mode': 'team', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('the test stops and restarts its own node inside its scope', p['internalControl']['before'] != p['internalControl']['after']),
                        assertion('an external control of that network is refused while the test runs', p['externalRefused']['conflicts'] > 0)],
         'artifacts': ['resource-parallel-receipt.json']},
        {'id': 'cleanup-releases-claims', 'mode': 'team', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('verified cleanup releases the claim and the alias then runs', h['actualCleanupReleasedAlias'])],
         'artifacts': ['resource-holds-receipt.json']},
    ]


def main():
    return run('WEB-11', PROOFS, REQUIRED, CHECKS, scenarios, SOURCES, verify)


if __name__ == '__main__':
    sys.exit(main())
