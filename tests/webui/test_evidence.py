import json
from pathlib import Path
import tempfile
import unittest
from evidence import REQUIRED, digest, verify


class EvidenceTests(unittest.TestCase):
    def test_unit_only_cannot_pass(self):
        with tempfile.TemporaryDirectory() as temp:
            directory = Path(temp)
            evidence = {'criterion': 'WEB-01', 'invocationId': 'fresh', 'scenarios': []}
            ep = directory / 'evidence.json'
            ep.write_text(json.dumps(evidence))
            result = {'criterion': 'WEB-01', 'invocationId': 'fresh', 'outcome': 'pass',
                      'requiredScenarios': REQUIRED, 'executedScenarios': REQUIRED,
                      'skippedScenarios': [], 'failedAssertions': [], 'evidenceDigest': digest(ep)}
            (directory / 'result.json').write_text(json.dumps(result))
            errors = verify(directory)
            self.assertIn('scenario observations missing', errors)
            self.assertIn('missing live provenance: binaries', errors)

    def test_incomplete_and_reduced_denominator_rejected(self):
        with tempfile.TemporaryDirectory() as temp:
            directory = Path(temp)
            ep = directory / 'evidence.json'
            ep.write_text(json.dumps({'criterion': 'WEB-01', 'invocationId': 'old'}))
            (directory / 'result.json').write_text(json.dumps({
                'criterion': 'WEB-01', 'invocationId': 'fresh', 'outcome': 'incomplete',
                'requiredScenarios': [], 'executedScenarios': [], 'skippedScenarios': REQUIRED,
                'failedAssertions': [], 'evidenceDigest': 'wrong'}))
            errors = verify(directory)
            for expected in ('invocation mismatch', 'required denominator changed',
                             'evidence digest mismatch', 'outcome is incomplete'):
                self.assertIn(expected, errors)


if __name__ == '__main__':
    unittest.main()
