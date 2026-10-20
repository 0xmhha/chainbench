import json
import os
from pathlib import Path
import shlex
import sys
import tempfile
import types
import unittest
from unittest import mock
from evidence_web14 import LIVE_ENV, CHECKS, CRITERIA, RACE, REQUIRED, VERIFIERS, acceptance_root, coverage, digest, go_tests, verify, verify_criterion
import run_web14


class EvidenceWEB14Tests(unittest.TestCase):
    def test_transports_split_combined_scenarios(self):
        modes, transports = coverage([{'scenarios': [{'mode': 'team', 'transport': 'local+SSH'}]},
                                      {'scenarios': [{'mode': 'personal', 'transport': 'local'}, {'id': 'no-mode'}]}])
        self.assertEqual(modes, {'team', 'personal'})
        self.assertEqual(transports, {'local', 'ssh'})

    def test_every_criterion_and_check_is_required(self):
        self.assertEqual(CRITERIA, ['WEB-%02d' % i for i in range(1, 14)])
        self.assertIn('criteria', REQUIRED)
        self.assertIn('live-go-tests', REQUIRED)
        for check in ['make check', 'go vet -tags e2e ./...', 'npm --prefix web run build', 'node --test tests/webui/*.test.mjs']:
            self.assertIn(check, CHECKS)
        self.assertIn('-race', RACE)
        self.assertIn('-json', RACE)

    def test_all_three_chain_binary_gates_are_opened_only_by_serial_race(self):
        self.assertIn('GWBFT_BIN', LIVE_ENV)
        self.assertIn('GSTABLE_BIN', LIVE_ENV)
        self.assertIn('GWEMIX_BIN', LIVE_ENV)
        self.assertIn('-p 1', RACE)

    def test_a_skipped_go_test_is_reported_not_passed(self):
        lines = [json.dumps({'Action': 'pass', 'Package': 'p', 'Test': 'TestA'}),
                 json.dumps({'Action': 'skip', 'Package': 'p', 'Test': 'TestLive_B'}),
                 json.dumps({'Action': 'pass', 'Package': 'p'}), 'not json']
        counts, skipped = go_tests(lines)
        self.assertEqual(counts, {'pass': 1, 'fail': 0, 'skip': 1})
        self.assertEqual(skipped, ['p TestLive_B'])

    def test_a_recheck_reads_the_published_criteria(self):
        with mock.patch.dict(os.environ, {'WEBUI_ACCEPTANCE_ROOT': '/published'}):
            self.assertEqual(acceptance_root(Path('staged/out/WEB-14')), Path('/published'))
        with mock.patch.dict(os.environ, {}, clear=True):
            self.assertEqual(acceptance_root(Path('out/WEB-14')), Path('out'))

    def test_a_criterion_is_judged_from_the_workspace_that_published_it(self):
        # A fresh reproduction runs in a staged copy that has no chainbench-out;
        # the criteria's artifact paths are relative to where they were published.
        fake = types.ModuleType('fake_verifier')
        fake.verify = lambda directory: [] if Path('out/accept/WEB-01/a.json').is_file() else ['invalid artifact path']
        with tempfile.TemporaryDirectory() as published, tempfile.TemporaryDirectory() as staged:
            directory = Path(published) / 'out/accept/WEB-01'
            directory.mkdir(parents=True)
            (directory / 'a.json').write_text('{}')
            (directory / 'evidence.json').write_text(json.dumps({'scenarios': [{'artifacts': [{'path': 'out/accept/WEB-01/a.json'}]}]}))
            cwd = os.getcwd()
            try:
                os.chdir(staged)
                with mock.patch.dict(sys.modules, {'fake_verifier': fake}), mock.patch.dict(VERIFIERS, {'WEB-01': 'fake_verifier'}):
                    verify_criterion(directory)
                self.assertEqual(os.getcwd(), os.path.realpath(staged))
            finally:
                os.chdir(cwd)

    def test_incomplete_cannot_be_pass(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory) / 'WEB-14'
            root.mkdir()
            (root / 'evidence.json').write_text(json.dumps({'criterion': 'WEB-14', 'invocationId': 'x', 'scenarios': []}))
            (root / 'result.json').write_text(json.dumps({'criterion': 'WEB-14', 'invocationId': 'x', 'outcome': 'incomplete',
                                                          'failedAssertions': ['WEB-05 belongs to another source'],
                                                          'evidenceDigest': digest(root / 'evidence.json')}))
            with self.assertRaises(AssertionError):
                verify(root)


class WEB14ExecutionGateTests(unittest.TestCase):
    def settings(self):
        return {'GSTABLE_BIN': 'native-stable', 'GWBFT_BIN': 'native-wbft',
                'GWEMIX_BIN': 'native-wemix', 'CHAINBENCH_DOCKER_SERVERS': 'owned-build'}

    def successful_command(self):
        return types.SimpleNamespace(returncode=0, stdout=json.dumps({
            'Action': 'pass', 'Package': 'fixture', 'Test': 'TestLive'}) + '\n', stderr='')

    def test_missing_w4_gate_refuses_before_docker_or_checks(self):
        with tempfile.TemporaryDirectory() as directory, mock.patch.dict(os.environ, self.settings(), clear=True):
            with mock.patch.object(run_web14.subprocess, 'run', return_value=self.successful_command()) as command:
                with self.assertRaisesRegex(RuntimeError, 'CHAINBENCH_W4_PROOF'):
                    run_web14.run_checks(Path(directory), Path(directory), Path(directory), {'checks': [], 'scenarios': []})
                command.assert_not_called()

    def test_non_enabled_diagnostic_gate_refuses_before_any_commands(self):
        for value in ['0', 'true', 'yes']:
            with self.subTest(value=value), tempfile.TemporaryDirectory() as directory:
                with mock.patch.dict(os.environ, {**self.settings(), 'CHAINBENCH_W4_PROOF': value}, clear=True):
                    with mock.patch.object(run_web14.subprocess, 'run', return_value=self.successful_command()) as command:
                        with self.assertRaisesRegex(RuntimeError, 'CHAINBENCH_W4_PROOF'):
                            run_web14.run_checks(Path(directory), Path(directory), Path(directory), {'checks': [], 'scenarios': []})
                        command.assert_not_called()

    def invoke_enabled(self, directory, command):
        with mock.patch.dict(os.environ, {**self.settings(), 'CHAINBENCH_W4_PROOF': '1', 'PRESERVE_SETTING': 'kept', 'CHAINBENCH_W3_PROOF': '1', 'CHAINBENCH_E2E_FROM_BIN': 'old'}, clear=True):
            with mock.patch.object(run_web14.subprocess, 'run', side_effect=command):
                run_web14.run_checks(Path(directory), Path(directory), Path(directory), {'checks': [], 'scenarios': []})

    def test_serial_race_has_an_explicit_bounded_long_suite_timeout(self):
        calls = []
        def command(*args, **kwargs):
            captured = dict(kwargs)
            if 'env' in captured:
                captured['env'] = dict(captured['env'])
            calls.append((args, captured))
            return self.successful_command()
        with tempfile.TemporaryDirectory() as directory:
            self.invoke_enabled(directory, command)
        race = next(args[0] for args, _ in calls if args[0] == RACE)
        self.assertIn('-timeout=3h', shlex.split(race))
        self.assertIn('-count=1', shlex.split(race))
        self.assertIn('-p 1', race)

    def test_only_serial_race_opens_the_diagnostic_and_native_gates(self):
        calls = []
        def command(*args, **kwargs):
            captured = dict(kwargs)
            if 'env' in captured:
                captured['env'] = dict(captured['env'])
            calls.append((args, captured))
            return self.successful_command()
        with tempfile.TemporaryDirectory() as directory:
            self.invoke_enabled(directory, command)
        checks = [(args[0], kwargs['env']) for args, kwargs in calls if 'env' in kwargs]
        self.assertEqual([command for command, _ in checks], CHECKS)
        for command, environment in checks:
            self.assertEqual(environment.get('PRESERVE_SETTING'), 'kept')
            if command == RACE:
                self.assertEqual(environment.get('CHAINBENCH_W4_PROOF'), '1')
                self.assertEqual(environment.get('GSTABLE_BIN'), 'native-stable')
            else:
                self.assertNotIn('CHAINBENCH_W4_PROOF', environment)
                self.assertNotIn('GSTABLE_BIN', environment)
                self.assertNotIn('CHAINBENCH_W3_PROOF', environment)
                self.assertNotIn('CHAINBENCH_E2E_FROM_BIN', environment)


if __name__ == '__main__':
    unittest.main()
