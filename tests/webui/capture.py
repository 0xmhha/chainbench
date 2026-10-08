"""Publish a fresh live receipt, bound to the final source and embedded SPA."""
import importlib
import contextlib
import io
import json
from pathlib import Path
import subprocess
import sys
from runtime_contract import source_digest

RUNNERS = {
    'WEB-01': ('run_web01.py', 'evidence'),
    'WEB-03': ('run_web03.py', 'evidence_web03'),
    'WEB-04': ('run_web04.py', 'evidence_web04'),
    'WEB-05': ('run_web05.py', 'evidence_web05'),
    'WEB-07': ('run_web07.py', 'evidence_web07'),
    'WEB-08': ('run_web08.py', 'evidence_web08'),
    'WEB-09': ('run_web09.py', 'evidence_web09'),
    'WEB-10': ('run_web10.py', 'evidence_web10'),
    'WEB-13': ('run_web13.py', 'evidence_web13'),
}


def verify_published(verifier, directory):
    # Older validators return a list, newer ones raise on failure. Both fail
    # closed. Suppress their PASS text until fresh reproduction also succeeds.
    with contextlib.redirect_stdout(io.StringIO()):
        errors = importlib.import_module(verifier).verify(directory)
    if errors:
        raise AssertionError('; '.join(errors))


def main():
    criterion, output = sys.argv[1:]
    if Path(output).is_absolute() or '..' in Path(output).parts:
        raise SystemExit('output must be workspace-relative')
    runner, verifier = RUNNERS[criterion]
    result = subprocess.run([sys.executable, 'tests/webui/' + runner, output])
    directory = Path(output) / criterion
    if (directory / 'evidence.json').exists() and (directory / 'result.json').exists():
        evidence_path = directory / 'evidence.json'
        evidence = json.loads(evidence_path.read_text())
        evidence['sourceDigest'] = source_digest(Path.cwd())
        evidence_path.write_text(json.dumps(evidence, indent=2) + '\n')
        receipt_path = directory / 'result.json'
        receipt = json.loads(receipt_path.read_text())
        module = importlib.import_module(verifier)
        receipt['evidenceDigest'] = module.digest(evidence_path)
        receipt_path.write_text(json.dumps(receipt, indent=2) + '\n')
    if result.returncode:
        raise SystemExit(result.returncode)
    verify_published(verifier, directory)
    print(criterion + ' PASS (fresh published capture)')


if __name__ == '__main__':
    main()
