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
 if '--workspace-dir' in args:
  ws=pathlib.Path(args[args.index('--workspace-dir')+1]);ws.mkdir(parents=True,exist_ok=True)
  (ws/'evidence.log').write_text('original evidence')
  (ws/'chain-record.json').write_text(json.dumps({'keysDir':'presets/keys'}))
 elif os.environ.get('GSTABLE_RPC')!='http://127.0.0.1:46001':
  print('attach endpoint is not reachable from caller');sys.exit(2)
 if os.environ.get('REQUEST_STOP_DURING_RUN')=='1' and ws.name=='c'+os.environ.get('STOP_ON_CASE','1'):pathlib.Path(os.environ['TCSWEEP_STOP_FILE']).touch()
 print('pass=0 fail=1 blocked=0 skip=0' if os.environ.get('CASE_FAIL') else 'pass=1 fail=0 blocked=0 skip=0')
 sys.exit(1 if os.environ.get('CASE_FAIL') else 0)
if args[:2]==['chain','stop']:sys.exit(int(os.environ.get('STOP_EXIT','0')))
if args[:2]==['chain','rm']:sys.exit(0)
if args[:2]==['chain','show']:
 entry={'host':'172.31.42.11','http':8601}
 if '--dial-rpc' in args and not os.environ.get('SHOW_NO_RPC'):entry['rpcURL']='http://127.0.0.1:46001'
 print(json.dumps({'entries':[entry]}));sys.exit(0)
raise SystemExit(3)
''')
        cli.chmod(0o700)
        self.hook = self.root / "retention hook"
        self.hook.write_text('''#!/usr/bin/env python3
import json, os, pathlib, sys
with open(os.environ['HOOK_CALLS'],'a') as out:out.write(json.dumps(sys.argv[1:])+'\\n')
assert (pathlib.Path(sys.argv[1])/'evidence.log').read_text()=='original evidence'
if os.environ.get('REQUEST_STOP_IN_HOOK')=='1' and pathlib.Path(sys.argv[1]).name=='c'+os.environ.get('STOP_ON_CASE','1'):pathlib.Path(os.environ['TCSWEEP_STOP_FILE']).touch()
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
            "SHOW_NO_RPC": "",
            "TCSWEEP_STOP_FILE": "",
            "REQUEST_STOP_DURING_RUN": "0",
            "REQUEST_STOP_IN_HOOK": "0",
            "STOP_ON_CASE": "1",
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

    def test_attach_uses_cli_dial_address_with_block_yaml(self):
        flags = self.attach_fixture()
        result = self.invoke(TCSWEEP_FLAGS=flags)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        shows = [args for args in self.calls() if args[:2] == ["chain", "show"]]
        self.assertEqual(len(shows), 1)
        self.assertIn("--dial-rpc", shows[0])
        self.assertEqual(self.calls("hooks"), [[str(self.root / "work/c1"), "1"]])

    def test_attach_refuses_missing_dial_address_and_finalizes_host(self):
        result = self.invoke(TCSWEEP_FLAGS=self.attach_fixture(), SHOW_NO_RPC="1")
        self.assertEqual(result.returncode, 2, result.stdout + result.stderr)
        self.assertEqual(self.calls("hooks"), [[str(self.root / "work/c1"), "0"]])
        self.assertIn("returned no reachable RPC URL", (self.root / "out.log").read_text())

    def stop_totals(self):
        lines = (self.root / "out.log").read_text().splitlines()
        return json.loads(next(line for line in reversed(lines) if line.startswith('{"pass":')))

    def test_stop_request_before_first_case_keeps_marker_and_runs_nothing(self):
        marker = self.root / "stop requested"
        marker.write_text("operator request")
        result = self.invoke(TCSWEEP_STOP_FILE=str(marker))
        self.assertEqual(result.returncode, 130, result.stdout + result.stderr)
        self.assertEqual(self.calls(), [])
        self.assertEqual(self.calls("hooks"), [])
        self.assertEqual(marker.read_text(), "operator request")
        self.assertEqual(self.stop_totals(), {"pass": 0, "fail": 0, "blocked": 0, "skip": 0, "notrun": 2})

    def test_stop_request_during_case_finishes_stop_and_retention_first(self):
        result = self.invoke(TCSWEEP_STOP_FILE=str(self.root / "stop requested"), REQUEST_STOP_DURING_RUN="1")
        self.assertEqual(result.returncode, 130, result.stdout + result.stderr)
        self.assertEqual([args[0] for args in self.calls()], ["run", "chain"])
        self.assertEqual(self.calls()[-1][:2], ["chain", "stop"])
        self.assertEqual(self.calls("hooks"), [[str(self.root / "work/c1"), "1"]])
        self.assertFalse((self.root / "work/c2").exists())
        self.assertEqual((self.root / "work/c1/evidence.log").read_text(), "original evidence")
        self.assertEqual(self.stop_totals(), {"pass": 1, "fail": 0, "blocked": 0, "skip": 0, "notrun": 1})
        self.assertNotIn("\nDONE\n", result.stdout)

    def test_stop_request_from_retention_halts_before_next_case(self):
        result = self.invoke(TCSWEEP_STOP_FILE=str(self.root / "stop"), REQUEST_STOP_IN_HOOK="1")
        self.assertEqual(result.returncode, 130, result.stdout + result.stderr)
        self.assertEqual(len(self.calls("hooks")), 1)
        self.assertEqual(len([args for args in self.calls() if args[0] == "run"]), 1)
        self.assertEqual(self.stop_totals()["notrun"], 1)

    def test_stop_request_keeps_failed_case_verdict(self):
        result = self.invoke(TCSWEEP_STOP_FILE=str(self.root / "stop"), REQUEST_STOP_DURING_RUN="1", CASE_FAIL="1")
        self.assertEqual(result.returncode, 130, result.stdout + result.stderr)
        self.assertEqual(self.calls("hooks"), [[str(self.root / "work/c1"), "0"]])
        self.assertEqual(self.stop_totals(), {"pass": 0, "fail": 1, "blocked": 0, "skip": 0, "notrun": 1})

    def test_stop_request_cannot_hide_stop_or_retention_failure(self):
        for field in ["STOP_EXIT", "HOOK_EXIT"]:
            with self.subTest(field=field):
                # Each failure needs a fresh untouched workspace.
                if (self.root / "work").exists():
                    shutil.rmtree(self.root / "work")
                result = self.invoke(TCSWEEP_STOP_FILE=str(self.root / "stop"), REQUEST_STOP_DURING_RUN="1", **{field: "9"})
                self.assertEqual(result.returncode, 2, result.stdout + result.stderr)
                self.assertNotIn("STOPPED at case boundary", result.stdout)
                self.assertFalse((self.root / "work/c2").exists())
                (self.root / "stop").unlink()

    def test_stop_request_does_not_start_next_attach_host(self):
        (self.root / "tests/tc/fixture/2.json").write_text('{"chainPreset":"stablenet-attached"}')
        (self.root / "presets/chain/stablenet-attached.json").write_text(json.dumps({
            "chain": "stablenet", "attach": {"rpc": ["${GSTABLE_RPC}"]}}))
        result = self.invoke(TCSWEEP_STOP_FILE=str(self.root / "stop"), REQUEST_STOP_IN_HOOK="1")
        self.assertEqual(result.returncode, 130, result.stdout + result.stderr)
        runs = [args for args in self.calls() if args[0] == "run"]
        self.assertEqual(len(runs), 1)
        self.assertTrue(runs[0][1].endswith("fixture/1.json"))
        self.assertFalse(any(args[:2] == ["chain", "show"] for args in self.calls()))
        self.assertFalse((self.root / "work/c2").exists())
        self.assertEqual(self.stop_totals()["notrun"], 1)

    def test_stop_request_after_last_case_is_completed_run(self):
        result = self.invoke(TCSWEEP_STOP_FILE=str(self.root / "stop"), REQUEST_STOP_IN_HOOK="1", STOP_ON_CASE="2")
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(len(self.calls("hooks")), 2)
        self.assertEqual(self.stop_totals(), {"pass": 2, "fail": 0, "blocked": 0, "skip": 0, "notrun": 0})

    def test_stop_request_broken_link_is_preserved_and_stops(self):
        marker = self.root / "stop"
        marker.symlink_to(self.root / "absent")
        result = self.invoke(TCSWEEP_STOP_FILE=str(marker))
        self.assertEqual(result.returncode, 130, result.stdout + result.stderr)
        self.assertEqual(self.calls(), [])
        self.assertTrue(marker.is_symlink())

    def attach_fixture(self):
        for number in (1, 2):
            (self.root / f"tests/tc/fixture/{number}.json").unlink()
        basic = self.root / "tests/tc/basic"
        basic.mkdir()
        (basic / "08-attached-chain-produces.json").write_text('{"chainPreset":"stablenet-attached"}')
        (self.root / "presets/chain/stablenet-attached.json").write_text(json.dumps({
            "chain": "stablenet", "attach": {"rpc": ["${GSTABLE_RPC}"]}}))
        (self.root / "localmap.yaml").write_text(
            "hosts:\n  172.31.42.11:\n    host: 127.0.0.1\n    ports:\n      8601: 46001\n")
        return f"--docker --server-set {self.root}/server-set.yaml"


if __name__ == "__main__":
    unittest.main()
