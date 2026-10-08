"""Uploaded native assets used by owned chain jobs, including a daemon restart."""
import hashlib
import json
from pathlib import Path
from run_test_jobs import main
from runtime_contract import source_digest

if __name__ == "__main__":
    before = source_digest(Path.cwd())
    main(browser_script="browser_asset_jobs.mjs", proof_name="asset-jobs", restart=True)
    if source_digest(Path.cwd()) != before:
        raise RuntimeError("source changed while the uploaded asset fixture was running")
    path = Path("chainbench-out/web-ui-development/asset-jobs/receipt.json")
    receipt = json.loads(path.read_text())
    receipt["sourceDigest"] = before
    receipt["dashboardSHA256"] = hashlib.sha256((Path(receipt["runtime"]) / "dashboard").read_bytes()).hexdigest()
    path.write_text(json.dumps(receipt, indent=2))
