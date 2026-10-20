"""Exercise emitted sweep invocations, independently of their printed labels."""
import json
import unittest

import test_tcsweep_retention as retention


class SweepRounds(unittest.TestCase):
    def setUp(self):
        self.fixture = retention.SweepRetention("test_explicit_binary_is_used")
        self.fixture.setUp()
        self.addCleanup(self.fixture.doCleanups)
        self.root = self.fixture.root
        for number in (1, 2):
            (self.root / f"tests/tc/fixture/{number}.json").unlink()
        for chain in ("stablenet", "wbft", "wemix"):
            self.preset(chain + "-bp1", {"chain": chain})

    def preset(self, name, value):
        (self.root / "presets/chain" / (name + ".json")).write_text(json.dumps(value))

    def case(self, reference, common=True):
        directory = self.root / ("tests/tc/common/fixture" if common else "tests/tc/fixture")
        directory.mkdir(parents=True, exist_ok=True)
        path = directory / "case.json"
        path.write_text(json.dumps({"chainPreset": reference}))
        return path

    def runs(self):
        return [args for args in self.fixture.calls() if args[0] == "run"]

    def selected_chains(self, source):
        selected = []
        for args in self.runs():
            name = args[args.index("--chain-preset") + 1] if "--chain-preset" in args else source
            seen = set()
            while name not in seen:
                seen.add(name)
                value = json.loads((self.root / "presets/chain" / (name + ".json")).read_text())
                if value.get("chain"):
                    selected.append(value["chain"])
                    break
                name = value["extends"]
            else:
                self.fail("emitted invocation resolves a cyclic preset")
        return selected

    def check_three_chains(self, reference, source):
        path = self.case(reference)
        before = path.read_bytes()
        result = self.fixture.invoke()
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(self.selected_chains(source), ["stablenet", "wemix", "wbft"])
        self.assertEqual(path.read_bytes(), before)
        self.assertEqual(len(self.fixture.calls("hooks")), 3)

    def test_wbft_common_case_runs_stablenet_in_default_round(self):
        self.check_three_chains({"extends": "wbft-bp1", "capabilities": ["rpc"]}, "wbft-bp1")

    def test_wemix_common_case_runs_stablenet_in_default_round(self):
        self.check_three_chains("wemix-bp1", "wemix-bp1")

    def test_stablenet_common_case_keeps_all_three_families(self):
        self.check_three_chains("stablenet-bp1", "stablenet-bp1")

    def test_alias_resolves_before_default_round_selection(self):
        self.preset("alias", {"extends": "wbft-bp1"})
        self.check_three_chains({"extends": "alias"}, "alias")

    def test_missing_stablenet_sibling_is_blocked_without_wrong_chain_run(self):
        self.case("wbft-bp1")
        (self.root / "presets/chain/stablenet-bp1.json").unlink()
        result = self.fixture.invoke()
        self.assertEqual(result.returncode, 2, result.stdout + result.stderr)
        self.assertEqual(self.selected_chains("wbft-bp1"), ["wemix", "wbft"])
        self.assertEqual(self.fixture.stop_totals(), {"pass": 2, "fail": 0, "blocked": 1, "skip": 0, "notrun": 0})

    def test_target_filename_does_not_prove_its_chain(self):
        self.case("wbft-bp1")
        self.preset("stablenet-bp1", {"chain": "wbft"})
        result = self.fixture.invoke()
        self.assertEqual(result.returncode, 2, result.stdout + result.stderr)
        self.assertEqual(self.selected_chains("wbft-bp1"), ["wemix", "wbft"])

    def test_inherited_target_chain_is_supported(self):
        self.preset("stable-base", {"chain": "stablenet"})
        self.preset("stablenet-bp1", {"extends": "stable-base"})
        self.check_three_chains("wbft-bp1", "wbft-bp1")

    def test_cyclic_target_is_blocked_before_invocation(self):
        self.case("wbft-bp1")
        self.preset("stablenet-bp1", {"extends": "cycle"})
        self.preset("cycle", {"extends": "stablenet-bp1"})
        result = self.fixture.invoke()
        self.assertEqual(result.returncode, 2, result.stdout + result.stderr)
        self.assertEqual(self.selected_chains("wbft-bp1"), ["wemix", "wbft"])

    def test_malformed_target_is_blocked_before_invocation(self):
        self.case("wbft-bp1")
        (self.root / "presets/chain/stablenet-bp1.json").write_text("{")
        result = self.fixture.invoke()
        self.assertEqual(result.returncode, 2, result.stdout + result.stderr)
        self.assertEqual(self.selected_chains("wbft-bp1"), ["wemix", "wbft"])

    def test_nonscalar_target_chain_is_not_a_family_match(self):
        self.case("wbft-bp1")
        self.preset("stablenet-bp1", {"chain": ["stablenet"]})
        result = self.fixture.invoke()
        self.assertEqual(result.returncode, 2, result.stdout + result.stderr)
        self.assertEqual(self.selected_chains("wbft-bp1"), ["wemix", "wbft"])

    def test_unmapped_base_has_no_implicit_default_chain_run(self):
        self.case("custom")
        self.preset("custom", {"chain": "wbft"})
        result = self.fixture.invoke()
        self.assertEqual(result.returncode, 2, result.stdout + result.stderr)
        self.assertEqual(self.runs(), [])
        self.assertEqual(self.fixture.calls("hooks"), [])
        self.assertEqual(self.fixture.stop_totals()["blocked"], 3)

    def test_noncommon_case_is_not_expanded(self):
        self.case("wbft-bp1", common=False)
        result = self.fixture.invoke()
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(self.selected_chains("wbft-bp1"), ["wbft"])


class SweepAttachOutputs(unittest.TestCase):
    def setUp(self):
        self.fixture = retention.SweepRetention("test_explicit_binary_is_used")
        self.fixture.setUp()
        self.addCleanup(self.fixture.doCleanups)
        self.root = self.fixture.root
        self.flags = self.fixture.attach_fixture()

    def attached_calls(self):
        return [args for args in self.fixture.calls() if args[0] == "run" and "--workspace-dir" not in args]

    def assert_output_flags(self, args, output):
        self.assertIn("--artifact-root", args)
        self.assertEqual(args[args.index("--artifact-root") + 1], str(output))
        self.assertIn("--no-skips", args)
        for forbidden in ("--server-set", "--workspace-config", "--docker", "--all-servers", "--keys-source"):
            self.assertNotIn(forbidden, args)

    def test_attached_target_receives_output_root_and_strict_verdict(self):
        output = self.root / "receipts"
        result = self.fixture.invoke(TCSWEEP_FLAGS=self.flags + f" --artifact-root {output} --no-skips")
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(len(self.attached_calls()), 1)
        self.assert_output_flags(self.attached_calls()[0], output)

    def test_equals_output_root_is_forwarded_without_machine_flags(self):
        output = self.root / "receipts"
        result = self.fixture.invoke(TCSWEEP_FLAGS=self.flags + f" --artifact-root={output} --no-skips")
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assert_output_flags(self.attached_calls()[0], output)

    def test_external_attach_uses_the_same_output_contract(self):
        self.preset_external()
        output = self.root / "external-receipts"
        result = self.fixture.invoke(TCSWEEP_FLAGS=self.flags + f" --artifact-root {output} --no-skips", GSTABLE_RPC="http://127.0.0.1:46001")
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(len(self.attached_calls()), 1)
        self.assert_output_flags(self.attached_calls()[0], output)
        self.assertEqual(self.fixture.calls("hooks"), [])

    def preset_external(self):
        (self.root / "presets/chain/stablenet-attached.json").write_text(json.dumps({
            "chain": "stablenet", "attach": {"rpc": ["${GSTABLE_TESTNET_RPC}"]}}))

    def test_missing_output_value_refuses_before_starting_host(self):
        result = self.fixture.invoke(TCSWEEP_FLAGS=self.flags + " --artifact-root")
        self.assertEqual(result.returncode, 2, result.stdout + result.stderr)
        self.assertEqual(self.fixture.calls(), [])


if __name__ == "__main__":
    unittest.main()
