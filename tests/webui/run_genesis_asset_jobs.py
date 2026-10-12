"""Use browser-selected finished genesis assets in owned native databases."""
import hashlib
import json
from pathlib import Path
import subprocess
from run_test_jobs import main
from runtime_contract import source_digest


def prepare_genesis(runtime, source_keys, output):
    cli = runtime / "chainbench"
    with (output / "cli-build.log").open("w") as log:
        subprocess.run(["go", "build", "-o", str(cli), "./cmd/chainbench"], stdout=log, stderr=log, check=True)
    files = []
    assets = json.loads((runtime / "assets.json").read_text())
    for index, chain in enumerate(("stablenet", "wbft", "wemix")):
        source = runtime / ("source" + str(index))
        binary = next(a for a in assets if a["id"] == chain)["path"]
        with (output / (chain + "-genesis-source.log")).open("w") as log:
            subprocess.run([str(cli), "chain", "up", "--workspace-dir", str(source), "--chain", chain,
                            "--binary", binary, "--keys", str(source_keys), "--bp", "4", "--stage", "deploy",
                            "--chain-id", str(9410 + index)], stdout=log, stderr=log, check=True, timeout=90)
        record = json.loads((source / "chain-record.json").read_text())
        path = Path(record["genesisPath"])
        if any(n.get("pid", 0) for n in record["nodes"]):
            raise RuntimeError("genesis preparation unexpectedly launched nodes")
        files.append({"chain": chain, "path": str(path), "checksum": hashlib.sha256(path.read_bytes()).hexdigest(), "chainId": 9410 + index})
    return {"genesis": files}


if __name__ == "__main__":
    before = source_digest(Path.cwd())
    main(browser_script="browser_genesis_asset_jobs.mjs", proof_name="genesis-asset-jobs", fixture_inputs=prepare_genesis)
    if source_digest(Path.cwd()) != before:
        raise RuntimeError("source changed during genesis asset verification")
    path = Path("chainbench-out/web-ui-development/genesis-asset-jobs/receipt.json")
    receipt = json.loads(path.read_text())
    receipt["sourceDigest"] = before
    receipt["dashboardSHA256"] = hashlib.sha256((Path(receipt["runtime"]) / "dashboard").read_bytes()).hexdigest()
    path.write_text(json.dumps(receipt, indent=2))
