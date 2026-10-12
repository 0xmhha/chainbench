"""The receipt verifier must reject stale/missing/partial/fabricated receipts."""
import copy
import json
from pathlib import Path
import tempfile
import unittest
from evidence_web04 import digest, verify
from run_web04 import output_directories, publish_output

class ReceiptTests(unittest.TestCase):
    def test_absolute_output_keeps_workspace_relative_evidence(self):
        with tempfile.TemporaryDirectory() as tmp:
            source, destination = output_directories(tmp)
            self.assertEqual(source, Path('chainbench-out/web-ui-acceptance/WEB-04'))
            self.assertFalse(source.is_absolute())
            self.assertEqual(destination, Path(tmp).resolve() / 'WEB-04')
            publish_output(source, destination)
            self.assertEqual((source / 'evidence.json').read_bytes(),
                             (destination / 'evidence.json').read_bytes())
            verify(destination)

    def test_incomplete_empty_scenarios_cannot_pass(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)
            evidence={'criterion':'WEB-04','invocationId':'current'}
            ep=root/'evidence.json';ep.write_text(json.dumps(evidence))
            (root/'result.json').write_text(json.dumps({'criterion':'WEB-04','invocationId':'current','evidenceDigest':digest(ep),'outcome':'incomplete','failedAssertions':['missing live setup']}))
            with self.assertRaises(AssertionError):verify(root)

    def test_latest_live_receipt_and_tampering(self):
        source=Path('chainbench-out/web-ui-acceptance/WEB-04')
        # This test intentionally requires an actual successful live invocation.
        verify(source)
        original=json.loads((source/'result.json').read_text())
        for field,value in [('invocationId','stale'),('evidenceDigest','0'*64),('executedScenarios',[]),('skippedScenarios',['builtin-wbft']),('failedAssertions',['failure'])]:
            with tempfile.TemporaryDirectory() as tmp:
                root=Path(tmp);(root/'evidence.json').write_bytes((source/'evidence.json').read_bytes())
                result=copy.deepcopy(original);result[field]=value
                (root/'result.json').write_text(json.dumps(result))
                with self.assertRaises(AssertionError,msg=field):verify(root)

if __name__=='__main__':unittest.main()
