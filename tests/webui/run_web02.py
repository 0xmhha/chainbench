"""Fresh WEB-02 acceptance: configuration import, structured edit, export and
real setup in a browser, with pinned inputs. Each proof reruns here."""
import json
import sys
from evidence_web02 import PROOFS, REQUIRED, verify
from proof_acceptance import assertion, run

CHECKS = ['go test -race -count=1 ./internal/app ./internal/dashboard -run "TestDocumentBundle|TestCaseBundle|Import|TestWebSecurityShowsRedacted|TestDeployment"',
          'node --test tests/webui/api-error.test.mjs']
SOURCES = ('internal/app/document_*.go', 'internal/app/deployment*.go', 'internal/dashboard/deployment.go', 'internal/dashboard/security.go', 'web/src/BundleImport.svelte', 'web/src/DeploymentEditor.svelte', 'web/src/api-error.mjs', 'tests/webui/*web02*', 'tests/webui/proof_acceptance.py', 'tests/webui/browser_preset_jobs.mjs', 'tests/webui/browser_asset_jobs.mjs')


def scenarios(receipts, output):
    """Map proof receipts to the WEB-02 observations of web-ui-acceptance.md."""
    b, p, a = receipts['web02-bundle'], receipts['preset-jobs'], receipts['asset-jobs']
    exported = json.loads((output / 'web02-bundle-workspace.bundle.json').read_text())
    return [
        {'id': 'bundle-import-preview-commit', 'mode': 'team', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('a server and path bundle is previewed in the browser and saved as immutable revisions', b['imported']['documents'] == 2),
                        assertion('private SSH values are moved out of the shared declaration', 'password' not in json.dumps(exported))],
         'artifacts': ['web02-bundle-receipt.json', 'web02-bundle-web02-bundle.png']},
        {'id': 'concrete-field-and-extension-errors', 'mode': 'team', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('an unregistered extension is refused with its document position and field', b['invalidBundle'].startswith('/documents/1/content') and 'extension' in b['invalidBundle'])],
         'artifacts': ['web02-bundle-receipt.json']},
        {'id': 'structured-edit-revision-conflict', 'mode': 'team', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('a stale revision is refused with its reason instead of overwriting the edit', bool(b['staleRevision']) and b['staleRevision'] != 'Conflict'),
                        assertion('a stale preset revision is refused after review', p['stalePresetRejected'])],
         'artifacts': ['web02-bundle-receipt.json', 'preset-jobs-receipt.json']},
        {'id': 'export-round-trip-semantics', 'mode': 'team', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('the exported workspace bundle imports back with the same declarations', b['roundTrip']['semanticallyEqual'] and b['roundTrip']['exportedDocuments'] == 2),
                        assertion('a saved case exports with its pinned preset and imports back pinned to that preset with the same executable meaning',
                                  b['caseBundle']['documents'] == 2 and b['caseBundle']['presetPinned'] and b['caseBundle']['semanticallyEqual']),
                        assertion('a case bundle without the preset it extends is refused by the preset id', b['caseBundle']['missingPresetNamed'])],
         'artifacts': ['web02-bundle-workspace.bundle.json', 'web02-bundle-case.bundle.json']},
        {'id': 'setup-from-imported-configuration', 'mode': 'team', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('a native deployment from the imported and edited workspace applies the edited ports', b['setup']['p2p'] == [39200, 39210, 39220, 39230] and b['setup']['chainId'] == '0x205c'),
                        assertion('saved presets set up native chains whose databases are verified', p['nativeDatabasesVerified'])],
         'artifacts': ['web02-bundle-receipt.json', 'preset-jobs-receipt.json', 'preset-jobs-preset-plan.png']},
        {'id': 'pinned-assets-and-revisions', 'mode': 'team', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('a preset edit after acceptance does not reach the running job', p['acceptedPresetEditIgnored']),
                        assertion('a changed uploaded binary is refused before any effect', a['changedBinaryRejectedBeforeEffects']),
                        assertion('uploaded assets persist across a restart', a['restartPersisted'])],
         'artifacts': ['preset-jobs-receipt.json', 'asset-jobs-receipt.json', 'asset-jobs-asset-library.png']},
    ]


def main():
    return run('WEB-02', PROOFS, REQUIRED, CHECKS, scenarios, SOURCES, verify)


if __name__ == '__main__':
    sys.exit(main())
