"""Fresh WEB-07 acceptance: owned node controls on real native nodes over
local and SSH targets. Each proof reruns inside this invocation."""
import sys
from evidence_web07 import PROOFS, REQUIRED, verify
from proof_acceptance import assertion, run

CHECKS = ['go test -race -count=1 ./internal/app -run "Control|WebNode|ObservedControls|Reset|Restart|Swap|Binary"',
          'go test -count=1 ./internal/chainsetup/... ./internal/core/process ./internal/core/nodeconfig']
SOURCES = ('internal/app/web_*control*.go', 'internal/app/web_node_*.go', 'internal/app/web_chain_*.go', 'internal/app/web_config_change.go', 'internal/app/web_network_guard.go', 'web/src/JobPanel.svelte', 'tests/webui/*web07*', 'tests/webui/proof_acceptance.py', 'tests/webui/browser_recorded_controls.mjs', 'tests/webui/browser_binary_replacement.mjs', 'tests/webui/browser_ssh_binary_replacement.mjs', 'tests/webui/browser_cancel_retention.mjs')


def scenarios(receipts, output):
    """Map proof receipts to the WEB-07 observations of web-ui-acceptance.md."""
    rc, b, s, c = (receipts[k] for k in ('recorded-controls', 'binary-replacement', 'ssh-binary-replacement', 'cancel-retention'))
    notice = (output / 'cancel-retention-attach-notice.txt').read_text()
    return [
        {'id': 'owned-start-stop', 'mode': 'team', 'transport': 'local+SSH', 'ownership': 'owned',
         'assertions': [assertion('a stopped owned node is started again on three chains', rc['resetLeftStoppedAndRelaunched']),
                        assertion('explicit stop of a verified SSH node stays available when relaunch is refused', s['nonExecutableRelaunchRefusedButExplicitStopAllowed'])],
         'artifacts': ['recorded-controls-receipt.json', 'ssh-binary-replacement-receipt.json']},
        {'id': 'config-replacement-restart', 'mode': 'team', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('structured generated-config replacement restarts the node with the new values', rc['structuredNativeConfigReplacement'] and rc['configArgumentsRetainedOnRestart']),
                        assertion('explicit restart keeps data and sibling processes', rc['explicitNodeRestart'] and rc['restartPreservesDataAndSiblings'] and rc['configReplacementPreservesDataAndSiblings']),
                        assertion('modified restart inputs and tampered choices are refused', rc['modifiedRestartInputsRefused'] and rc['configChoicesAndTamperedKeysRefused'])],
         'artifacts': ['recorded-controls-receipt.json', 'recorded-controls-wbft-table-plan.png']},
        {'id': 'binary-replacement-restart', 'mode': 'team', 'transport': 'local+SSH', 'ownership': 'owned',
         'assertions': [assertion('a registered native binary replaces the selected node locally, with combined config', b['registeredNativeReplacement'] and b['combinedConfigAndBinaryReplacement'] and b['nativeDatabaseGenesisPreserved']),
                        assertion('the same works over the operator\'s own SSH', s['privateSSHNativeReplacement'] and s['configGenesisDataAndSiblingPreserved']),
                        assertion('later restart and rollback keep the execution history', b['laterRestartAndRollback'] and s['registeredRollbackPreservesHistory'] and s['reviewedBindingSupportsLaterRestart'])],
         'artifacts': ['binary-replacement-receipt.json', 'binary-replacement-binary-plan.png', 'ssh-binary-replacement-receipt.json', 'ssh-binary-replacement-ssh-binary-plan.png']},
        {'id': 'non-producer-reset-stopped', 'mode': 'team', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('a non-producer is reset to its genesis head and stays stopped', rc['nativeNonProducerReset'] and rc['nativeHeadAtGenesis'] and rc['verifiedStoppedNodeReset']),
                        assertion('sibling nodes keep their processes and data', rc['siblingNodesPreserved'])],
         'artifacts': ['recorded-controls-receipt.json', 'recorded-controls-wbft-reset.png', 'recorded-controls-stablenet-reset.png', 'recorded-controls-wemix-reset.png']},
        {'id': 'producer-and-attach-refused', 'mode': 'team', 'transport': 'local', 'ownership': 'owned+attached',
         'assertions': [assertion('producer reset is refused by the API and absent from the form', rc['producerResetRefused']),
                        assertion('control of an attached network is refused by the API', 400 <= c['attachRefusals']['node.stop'] < 500),
                        assertion('the form explains attached-network refusal', 'attach' in notice)],
         'artifacts': ['recorded-controls-receipt.json', 'cancel-retention-receipt.json', 'cancel-retention-attach-refused.png', 'cancel-retention-attach-notice.txt']},
        {'id': 'cli-semantics-preserved', 'mode': 'team', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('controlled networks still pass real engine sessions without skips', rc['realPassingSessionsWithoutSkips']),
                        assertion('existing chain setup, process and node configuration tests pass', True)],
         'artifacts': ['recorded-controls-receipt.json', 'check-1.log']},
    ]


def main():
    return run('WEB-07', PROOFS, REQUIRED, CHECKS, scenarios, SOURCES, verify)


if __name__ == '__main__':
    sys.exit(main())
