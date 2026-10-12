"""Browser bundle import, structured edit, export and native setup proof; full WEB-02 acceptance remains open."""
import hashlib
import json
from pathlib import Path
from run_test_jobs import main
from runtime_contract import source_digest

if __name__ == '__main__':
    before = source_digest(Path.cwd())
    main(browser_script='browser_web02_bundle.mjs', proof_name='web02-bundle', browser_timeout=600)
    if source_digest(Path.cwd()) != before:
        raise RuntimeError('source changed during bundle verification')
    path = Path('chainbench-out/web-ui-development/web02-bundle/receipt.json')
    receipt = json.loads(path.read_text())
    receipt['sourceDigest'] = before
    receipt['dashboardSHA256'] = hashlib.sha256((Path(receipt['runtime']) / 'dashboard').read_bytes()).hexdigest()
    path.write_text(json.dumps(receipt, indent=2))
