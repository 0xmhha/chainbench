import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest import mock
from evidence_web14 import CHECKS, CRITERIA, RACE, REQUIRED, acceptance_root, coverage, digest, go_tests, verify


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


if __name__ == '__main__':
    unittest.main()
