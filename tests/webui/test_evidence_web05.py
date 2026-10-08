import json
from pathlib import Path
import tempfile
import unittest
from evidence_web05 import REQUIRED, digest, verify, verify_coverage
from run_web05 import clean_invocation_output, stage_sources

class EvidenceWEB05Tests(unittest.TestCase):
    def test_previous_session_files_removed(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            old = root / 'session' / 'previous'
            old.mkdir(parents=True)
            (old / 'session.json').write_text('{}')
            (root / 'result.json').write_text('{}')
            clean_invocation_output(root)
            self.assertEqual(list(root.iterdir()), [])

    def test_build_snapshot_does_not_mutate_workspace(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            workspace = root / 'workspace'
            workspace.mkdir()
            spa = workspace / 'internal' / 'dashboard' / 'spa'
            spa.mkdir(parents=True)
            (spa / 'index.html').write_text('original')
            (workspace / 'chainbench-out').mkdir()
            (workspace / 'chainbench-out' / 'previous.json').write_text('{}')
            dependency_bin = workspace / 'web/node_modules/vite/bin'
            dependency_bin.mkdir(parents=True)
            (dependency_bin / 'vite.js').write_text('dependency')
            executable = workspace / 'web/node_modules/.bin'
            executable.mkdir()
            (executable / 'vite').symlink_to('../vite/bin/vite.js')
            runtime = root / 'runtime'
            runtime.mkdir()
            source = stage_sources(workspace, runtime)
            (source / 'internal/dashboard/spa/index.html').write_text('generated')
            self.assertEqual((spa / 'index.html').read_text(), 'original')
            self.assertFalse((source / 'chainbench-out').exists())
            self.assertEqual(((source / 'web/node_modules/vite/bin/vite.js').resolve()).read_text(), 'dependency')
            self.assertTrue((source / 'web/node_modules/.bin/vite').is_symlink())
            self.assertEqual((source / 'web/node_modules/.bin/vite').resolve(), (source / 'web/node_modules/vite/bin/vite.js').resolve())

    def coverage(self):
        contract = {'vocabulary': {'entries': [
            {'kind': 'action', 'name': 'read', 'schemaRef': '#/$defs/action-read'}]},
            'contract': {'$defs': {'action-read': {'oneOf': [
                {'properties': {'do': {}, 'source': {}, 'on': {}}},
                {'properties': {'do': {}, 'source': {}, 'on': {}, 'address': {}}}]}}}}
        coverage = [{'kind': 'action', 'name': 'read', 'schema': True,
                     'edited': True, 'roundTrip': True, 'executed': True,
                     'argumentPaths': ['address', 'on'],
                     'editedArgumentPaths': ['address', 'on'],
                     'executedArgumentPaths': ['address', 'on']}]
        return contract, coverage

    def test_complete_coverage(self):
        verify_coverage(*self.coverage())

    def test_self_reported_reduced_argument_denominator_rejected(self):
        contract, coverage = self.coverage()
        for key in ('argumentPaths', 'editedArgumentPaths', 'executedArgumentPaths'):
            coverage[0][key] = ['on']
        with self.assertRaises(AssertionError):
            verify_coverage(contract, coverage)

    def test_duplicate_registration_rejected(self):
        contract, coverage = self.coverage()
        with self.assertRaises(AssertionError):
            verify_coverage(contract, coverage + coverage)

    def test_unexecuted_arguments_rejected(self):
        contract, coverage = self.coverage()
        coverage[0]['executedArgumentPaths'] = []
        with self.assertRaises(AssertionError):
            verify_coverage(contract, coverage)

    def receipt(self, root, outcome='pass', executed=None):
        ep = root / 'evidence.json'
        ep.write_text(json.dumps({'criterion': 'WEB-05', 'invocationId': 'fresh', 'scenarios': [], 'artifacts': []}))
        (root / 'result.json').write_text(json.dumps({'criterion': 'WEB-05', 'invocationId': 'fresh', 'outcome': outcome, 'requiredScenarios': REQUIRED, 'executedScenarios': executed if executed is not None else REQUIRED, 'skippedScenarios': [], 'failedAssertions': [], 'evidenceDigest': digest(ep)}))

    def test_incomplete_cannot_be_pass(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory); self.receipt(root, outcome='incomplete')
            with self.assertRaises(AssertionError): verify(root)

    def test_reduced_scenario_denominator_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory); self.receipt(root, executed=['references'])
            with self.assertRaises(AssertionError): verify(root)

    def test_previous_invocation_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory); self.receipt(root)
            result = json.loads((root / 'result.json').read_text()); result['invocationId'] = 'previous'
            (root / 'result.json').write_text(json.dumps(result))
            with self.assertRaises(AssertionError): verify(root)

    def test_modified_evidence_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory); self.receipt(root)
            with (root / 'evidence.json').open('a') as stream: stream.write(' ')
            with self.assertRaises(AssertionError): verify(root)

if __name__ == '__main__': unittest.main()
