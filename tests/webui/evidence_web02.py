"""Validate fresh WEB-02 configuration import, edit, export and setup acceptance."""
from evidence import digest  # capture.py publishes evidence digests through this module
from proof_acceptance import verify_proofs

REQUIRED = ['bundle-import-preview-commit', 'concrete-field-and-extension-errors', 'structured-edit-revision-conflict', 'export-round-trip-semantics', 'setup-from-imported-configuration', 'pinned-assets-and-revisions']
PROOFS = {
    'web02-bundle': ('run_web02_bundle.py', ['workspace.bundle.json', 'web02-bundle.png']),
    'preset-jobs': ('run_preset_jobs.py', ['preset-plan.png']),
    'asset-jobs': ('run_asset_jobs.py', ['asset-library.png']),
}


def verify(directory):
    return verify_proofs(directory, 'WEB-02', REQUIRED, PROOFS)


if __name__ == '__main__':
    import sys
    errors = verify(sys.argv[1])
    if errors:
        print('\n'.join(errors), file=sys.stderr)
        sys.exit(1)
    print('WEB-02 PASS')
