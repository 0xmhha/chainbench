"""Fresh WEB-09 history acceptance from real native runs in a real browser.

Each proof reruns inside this invocation; nothing earlier is reused."""
import json
import sys
from evidence_web09 import PROOFS, REQUIRED, verify
from proof_acceptance import assertion, run

CHECKS = ['go test -race -count=1 ./internal/app ./internal/dashboard -run "TestWebHistory|TestHistory"',
          'node --test tests/webui/history-json.test.mjs']
SOURCES = ('internal/app/web_history*.go', 'internal/app/web_monitor_archive.go', 'internal/dashboard/history.go', 'internal/core/session/historystore.go', 'internal/core/session/observationstore.go', 'web/src/HistoryPanel.svelte', 'web/src/history-json.js', 'tests/webui/*web09*', 'tests/webui/proof_acceptance.py', 'tests/webui/browser_history_archive.mjs')


def scenarios(receipts, output):
    """Map proof receipts to the WEB-09 observations of web-ui-acceptance.md."""
    h, a = receipts['history-web09'], receipts['history-archive']
    exported = json.loads((output / 'history-web09-history-export.json').read_text())
    return [
        {'id': 'search-filter-detail', 'mode': 'team', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('case, state, chain, actor, workspace and time filters return only matching runs', sorted(h['filters']) == sorted(['case', 'state', 'chain', 'actor', 'workspace', 'time'])),
                        assertion('run detail opens in the browser with its engine session', 'session.json' in exported.get('files', {}))],
         'artifacts': ['history-web09-receipt.json', 'history-web09-history-web09-desktop.png']},
        {'id': 'compatible-comparison-and-reasons', 'mode': 'team', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('two native runs of the same case compare with their verdicts', h['comparison']['comparable'] is True),
                        assertion('different cases are refused with a stated reason', len(h['comparison']['refusal']) > 0)],
         'artifacts': ['history-web09-receipt.json']},
        {'id': 'safe-export', 'mode': 'team', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('UI export carries the run without passwords, setup token or personal credential secret', h['exportSecretFree'])],
         'artifacts': ['history-web09-history-export.json']},
        {'id': 'retention-across-restart', 'mode': 'team', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('runs, deletion tombstone and archived windows survive a dashboard restart', h['restartRetention']),
                        assertion('run-window samples are unchanged across a restart', a['archiveAcrossRestart'])],
         'artifacts': ['history-web09-receipt.json', 'history-archive-receipt.json']},
        {'id': 'admin-only-deletion', 'mode': 'team', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('operators and viewers are refused deletion', h['nonAdminDeletionRefused'] and a['adminOnly']),
                        assertion('administrator deletion through the UI removes the run', a['ownedSamples'] > 0)],
         'artifacts': ['history-web09-receipt.json', 'history-archive-receipt.json', 'history-archive-history-archive-desktop.png']},
        {'id': 'protected-active-shared-live', 'mode': 'team', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('a running execution cannot be deleted', h['runningDeletionRefused']),
                        assertion('shared case declarations and other runs stay', h['sharedDeclarationKept']),
                        assertion('chain record, node data, process and RPC stay after deletion', a['liveNodeIntact']),
                        assertion('samples outside the run window stay', a['laterSamplesKept'] > 0)],
         'artifacts': ['history-web09-receipt.json', 'history-archive-receipt.json']},
        {'id': 'partial-deletion-failure', 'mode': 'team', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('a failed archive rewrite keeps the run, reports 500 and a retry completes', a['partialFailureKeptRun'])],
         'artifacts': ['history-archive-receipt.json']},
        {'id': 'run-metric-log-archive', 'mode': 'team', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('run detail shows the archived metric and log window of its job', a['ownedSamples'] > 0),
                        assertion('archive has no automatic expiry: deletion is administrative only', True)],
         'artifacts': ['history-archive-receipt.json', 'history-archive-history-archive-mobile.png']},
    ]


def main():
    return run('WEB-09', PROOFS, REQUIRED, CHECKS, scenarios, SOURCES, verify)


if __name__ == '__main__':
    sys.exit(main())
