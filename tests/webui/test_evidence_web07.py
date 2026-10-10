"""The WEB-07 verifier must reject partial, stale or altered control evidence."""
import copy
import json
from pathlib import Path
import shutil
import tempfile
import unittest
from evidence import digest
from evidence_web07 import REQUIRED, verify


class WEB07EvidenceTest(unittest.TestCase):
    def test_empty_or_incomplete_invocation_cannot_pass(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / 'evidence.json').write_text(json.dumps({'criterion': 'WEB-07', 'invocationId': 'x', 'scenarios': [], 'proofs': []}))
            (root / 'result.json').write_text(json.dumps({'criterion': 'WEB-07', 'invocationId': 'x', 'startedAt': '2026-10-09T00:00:00+00:00', 'finishedAt': '2026-10-09T00:01:00+00:00', 'outcome': 'incomplete', 'requiredScenarios': REQUIRED, 'executedScenarios': [], 'skippedScenarios': REQUIRED, 'failedAssertions': ['no proof'], 'evidenceDigest': digest(root / 'evidence.json')}))
            errors = verify(root)
            self.assertIn('incomplete acceptance', errors)
            self.assertIn('native proofs missing', errors)

    def test_latest_live_receipt_and_tampering(self):
        source = Path('chainbench-out/web-ui-acceptance/WEB-07')
        # This test intentionally requires an actual successful live invocation.
        self.assertEqual(verify(source), [])
        original = json.loads((source / 'evidence.json').read_text())
        for mutate in (lambda e: e['proofs'].pop(), lambda e: e['proofs'][0].update(dashboardSHA256='0' * 64), lambda e: e['scenarios'][0]['assertions'][0].update(passed=False), lambda e: e['coverage'].update(mockTargets=1)):
            evidence = copy.deepcopy(original)
            mutate(evidence)
            with tempfile.TemporaryDirectory() as tmp:
                root = Path(tmp)
                shutil.copy2(source / 'result.json', root / 'result.json')
                (root / 'evidence.json').write_text(json.dumps(evidence))
                self.assertTrue(verify(root))


if __name__ == '__main__':
    unittest.main()
