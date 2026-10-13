"""Real browser + CLI job failure must keep diagnosis but reclaim its ended DBs."""
import hashlib
import json
from pathlib import Path
from runtime_contract import source_digest
from run_test_jobs import main as run_jobs


def main():
    before = source_digest(Path.cwd())
    output = Path('chainbench-out/web-ui-development/storage-failure')
    try:
        run_jobs(browser_script='browser_storage_failure.mjs', proof_name='storage-failure')
    except RuntimeError as error:
        if 'DELIBERATE_STORAGE_FAILURE_AFTER_NATIVE_SETUP' not in str(error):
            raise
    else:
        raise AssertionError('deliberate failure was swallowed or not exercised')
    partial = output / 'partial.json'
    facts = json.loads(partial.read_text())
    runtime = Path(facts['runtime'])
    assert facts['outcome'] == 'incomplete' and facts['initializedNodeDatabases'] == 4 and facts['runningNodesBeforeFailure'] == 4
    assert not [p for p in runtime.rglob('*') if p.is_dir() and p.name in ('chaindata', 'lightchaindata', 'etcd')]
    assert 'DELIBERATE_STORAGE_FAILURE_AFTER_NATIVE_SETUP' in (output / 'browser.log').read_text()
    records = list(runtime.rglob('chain-record.json'))
    assert len(records) == 1
    nodes = json.loads(records[0].read_text())['nodes']
    assert len(nodes) == 4
    diagnostics = [Path(node[field]) for node in nodes for field in ('configPath', 'logPath')]
    assert all(path.is_relative_to(runtime) and path.is_file() for path in diagnostics)
    assert source_digest(Path.cwd()) == before, 'source changed during proof'
    audit = runtime / 'storage-audit.jsonl'
    assert audit.is_file()
    proof = {'scope': 'deliberate failed native Web job cleanup; no Seed criterion awarded',
             'runtime': str(runtime), 'sourceDigest': before, 'originalFailurePreserved': True,
             'initializedNodeDatabases': 4, 'endedDatabasesRemaining': 0,
             'artifacts': {str(p): hashlib.sha256(p.read_bytes()).hexdigest()
                           for p in [partial, audit, output / 'browser.log', records[0], *diagnostics]},
             'seedAcceptanceAwarded': False}
    (output / 'proof.json').write_text(json.dumps(proof, indent=2) + '\n')
    print('Expected native failure cleanup PASS: original failure retained; four ended DBs removed.')


if __name__ == '__main__':
    main()
