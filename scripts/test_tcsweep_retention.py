"""Observable lifecycle contracts for a sweep with an evidence retention hook."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


class SweepRetention(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix="cb-sweep-")
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        for directory in ("scripts", "bin", "tests/tc/fixture", "presets/chain"):
            (self.root / directory).mkdir(parents=True)
        shutil.copyfile(Path(__file__).with_name("tcsweep.sh"), self.root / "scripts/tcsweep.sh")
        for number in (1, 2):
            (self.root / f"tests/tc/fixture/{number}.json").write_text(json.dumps({"chainPreset": "stablenet-bp1"}))
        (self.root / "presets/chain/stablenet-bp1.json").write_text('{"chain":"stablenet"}')
        cli = self.root / "bin/chainbench"
        cli.write_text('''#!/usr/bin/env python3
import json, os, pathlib, sys
args=sys.argv[1:]
with open(os.environ['CALLS'], 'a') as out: out.write(json.dumps(args)+'\\n')
if args[0]=='run':
 ws=pathlib.Path(args[args.index('--workspace-dir')+1]);ws.mkdir(parents=True,exist_ok=True)
 (ws/'evidence.log').write_text('original evidence')
 print('pass=0 fail=1 blocked=0 skip=0' if os.environ.get('CASE_FAIL') else 'pass=1 fail=0 blocked=0 skip=0')
 sys.exit(1 if os.environ.get('CASE_FAIL') else 0)
if args[:2]==['chain','stop']:sys.exit(int(os.environ.get('STOP_EXIT','0')))
if args[:2]==['chain','rm']:sys.exit(0)
raise SystemExit(3)
''')
        cli.chmod(0o700)
        self.hook = self.root / "retention hook"
        self.hook.write_text('''#!/usr/bin/env python3
import json, os, pathlib, sys
with open(os.environ['HOOK_CALLS'],'a') as out:out.write(json.dumps(sys.argv[1:])+'\\n')
assert (pathlib.Path(sys.argv[1])/'evidence.log').read_text()=='original evidence'
raise SystemExit(int(os.environ.get('HOOK_EXIT','0')))
''')
        self.hook.chmod(0o700)
        self.env = {
            **os.environ,
            "TCSWEEP_BIN": str(cli),
            "TCSWEEP_WS": str(self.root / "work"),
            "TCSWEEP_FLAGS": "",
            "TCSWEEP_FLAGS_STABLENET": "",
            "TCSWEEP_RETAIN_HOOK": str(self.hook),
            "CALLS": str(self.root / "calls"),
            "HOOK_CALLS": str(self.root / "hooks"),
            "CASE_FAIL": "",
            "STOP_EXIT": "0",
            "HOOK_EXIT": "0",
            "PYTHONDONTWRITEBYTECODE": "1",
        }

    def invoke(self, **changes):
        return subprocess.run(["bash", str(self.root / "scripts/tcsweep.sh"), str(self.root / "out.log")], env={**self.env, **changes}, text=True, capture_output=True, timeout=30)

    def calls(self, name="calls"):
        path = self.root / name
        return [json.loads(line) for line in path.read_text().splitlines()] if path.exists() else []

    def test_passing_cases_keep_workspace_and_call_hook(self):
        result = self.invoke()
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(self.calls("hooks"), [[str(self.root / "work/c1"), "1"], [str(self.root / "work/c2"), "1"]])
        self.assertFalse(any(args[:2] == ["chain", "rm"] for args in self.calls()))
        self.assertEqual((self.root / "work/c1/evidence.log").read_text(), "original evidence")

    def test_failed_cases_are_retained_and_reclaimed_too(self):
        result = self.invoke(CASE_FAIL="1")
        self.assertEqual(result.returncode, 1)
        self.assertEqual([args[1] for args in self.calls("hooks")], ["0", "0"])
        self.assertTrue((self.root / "work/c2/evidence.log").exists())

    def test_retention_failure_stops_before_next_case(self):
        result = self.invoke(HOOK_EXIT="8")
        self.assertEqual(result.returncode, 2)
        self.assertEqual(len([args for args in self.calls() if args[0] == "run"]), 1)
        self.assertTrue((self.root / "work/c1/evidence.log").exists())
        self.assertFalse(any(args[:2] == ["chain", "rm"] for args in self.calls()))

    def test_stop_failure_does_not_invoke_retention_or_next_case(self):
        result = self.invoke(STOP_EXIT="9")
        self.assertEqual(result.returncode, 2)
        self.assertEqual(self.calls("hooks"), [])
        self.assertEqual(len([args for args in self.calls() if args[0] == "run"]), 1)
        self.assertTrue((self.root / "work/c1/evidence.log").exists())

    def test_missing_hook_refuses_before_any_case(self):
        result = self.invoke(TCSWEEP_RETAIN_HOOK=str(self.root / "absent"))
        self.assertEqual(result.returncode, 2)
        self.assertEqual(self.calls(), [])

    def test_existing_workspace_is_preserved_before_any_case(self):
        ws = self.root / "work/c1"
        ws.mkdir(parents=True)
        (ws / "evidence.log").write_text("previous evidence")
        result = self.invoke()
        self.assertEqual(result.returncode, 2)
        self.assertEqual(self.calls(), [])
        self.assertEqual((ws / "evidence.log").read_text(), "previous evidence")

    def test_explicit_binary_is_used(self):
        binary = self.root / "bin/pinned cli"
        (self.root / "bin/chainbench").rename(binary)
        result = self.invoke(TCSWEEP_BIN=str(binary))
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(len(self.calls("hooks")), 2)

    def test_without_hook_keeps_existing_success_cleanup(self):
        result = self.invoke(TCSWEEP_RETAIN_HOOK="")
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(self.calls("hooks"), [])
        self.assertEqual(len([args for args in self.calls() if args[:2] == ["chain", "rm"]]), 2)
        self.assertFalse((self.root / "work/c1").exists())

    def test_without_hook_keeps_failure_evidence(self):
        result = self.invoke(TCSWEEP_RETAIN_HOOK="", CASE_FAIL="1")
        self.assertEqual(result.returncode, 1)
        self.assertFalse(any(args[:2] == ["chain", "rm"] for args in self.calls()))
        self.assertTrue((self.root / "work/c1/evidence.log").exists())


if __name__ == "__main__":
    unittest.main()
