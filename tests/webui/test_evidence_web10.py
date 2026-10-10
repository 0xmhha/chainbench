"""Reject evidence mutations against the current live invocation."""
import copy
import json
import tempfile
import unittest
from pathlib import Path
from evidence import digest
from evidence_web10 import verify, REQUIRED

class WEB10EvidenceTest(unittest.TestCase):
    def test_live_receipt_and_rejected_mutations(self):
        original=Path('chainbench-out/web-ui-acceptance/WEB-10')
        self.assertEqual(verify(original), [])
        result=json.loads((original/'result.json').read_text())
        evidence=json.loads((original/'evidence.json').read_text())
        browser=json.loads((original/'browser.json').read_text())
        mutations=[
            lambda r,e,b:r.update(requiredScenarios=REQUIRED[:-1]),
            lambda r,e,b:r.update(skippedScenarios=[REQUIRED[0]]),
            lambda r,e,b:e['scenarios'][0].update(invocationId='old'),
            lambda r,e,b:e['scenarios'][0].update(observedAt='2000-01-01T00:00:00+00:00'),
            lambda r,e,b:e['scenarios'][0]['assertions'][0].update(passed=False),
            lambda r,e,b:e['coverage'].update(mockTargets=1),
            lambda r,e,b:e['server'].update(sha256='0'*64),
            lambda r,e,b:e['target'].update(authenticated=False),
            lambda r,e,b:e['frontend'][0].update(sha256='0'*64),
            lambda r,e,b:e['checks'][0].update(exitCode=1),
            lambda r,e,b:b['matrix']['viewer'].update(uiWritable=True),
            lambda r,e,b:b['secretScan'].update(leaks=1),
            lambda r,e,b:b.update(foreignStatuses=[200,404,404]),
            lambda r,e,b:b.update(csrfStatuses=[200,403]),
        ]
        for mutation in mutations:
            with self.subTest(mutation=mutation),tempfile.TemporaryDirectory() as temp:
                r,e,b=copy.deepcopy((result,evidence,browser));mutation(r,e,b)
                directory=Path(temp)
                (directory/'browser.json').write_text(json.dumps(b))
                (directory/'evidence.json').write_text(json.dumps(e))
                r['evidenceDigest']=digest(directory/'evidence.json')
                (directory/'result.json').write_text(json.dumps(r))
                self.assertTrue(verify(directory))

if __name__=='__main__':unittest.main()
