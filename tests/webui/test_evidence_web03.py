import copy
import json
from pathlib import Path
import tempfile
import unittest
from evidence import digest
from evidence_web03 import REQUIRED, verify
from run_web03 import output_directories, publish_output


class EvidenceWEB03Tests(unittest.TestCase):
    def test_absolute_workspace_output_retains_relative_evidence_paths(self):
        output, destination = output_directories(Path.cwd() / 'chainbench-out/web-ui-acceptance')
        self.assertEqual(output, Path('chainbench-out/web-ui-acceptance/WEB-03'))
        self.assertFalse(output.is_absolute())
        self.assertEqual(destination, output.resolve())

    def test_absolute_external_output_receives_current_receipt(self):
        with tempfile.TemporaryDirectory() as directory:
            output, destination = output_directories(directory)
            self.assertFalse(output.is_absolute())
            self.assertEqual(destination, Path(directory).resolve() / 'WEB-03')
            source = Path(directory, 'source')
            source.mkdir()
            for name in ('result.json', 'evidence.json', 'browser.json'):
                (source / name).write_text('fresh invocation')
            destination.mkdir()
            (destination / 'result.json').write_text('previous invocation')
            publish_output(source, destination)
            for artifact in source.iterdir():
                self.assertEqual((destination / artifact.name).read_bytes(), artifact.read_bytes())

    def fixture(self, directory):
        evidence = {'criterion': 'WEB-03', 'invocationId': 'test-invocation', 'scenarios': [], 'artifacts': []}
        result = {'criterion': 'WEB-03', 'invocationId': 'test-invocation', 'requiredScenarios': REQUIRED,
                  'executedScenarios': REQUIRED, 'skippedScenarios': [], 'failedAssertions': [], 'outcome': 'pass'}
        return result, evidence

    def write(self, directory, result, evidence):
        ep = Path(directory, 'evidence.json')
        ep.write_text(json.dumps(evidence))
        result = copy.deepcopy(result)
        result['evidenceDigest'] = digest(ep)
        Path(directory, 'result.json').write_text(json.dumps(result))

    def test_exit_zero_receipt_without_observations_is_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            result, evidence = self.fixture(directory)
            self.write(directory, result, evidence)
            errors = verify(directory)
            self.assertIn('missing scenario observations', errors)
            self.assertIn('API observations missing', errors)
            self.assertIn('missing actual SSH target identity', errors)

    def test_skips_changed_denominator_and_stale_digest_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            result, evidence = self.fixture(directory)
            result['requiredScenarios'] = REQUIRED[:-1]
            result['skippedScenarios'] = [REQUIRED[-1]]
            self.write(directory, result, evidence)
            with Path(directory, 'evidence.json').open('a') as file:
                file.write(' ')
            errors = verify(directory)
            self.assertIn('mandatory scenario denominator/execution mismatch', errors)
            self.assertIn('incomplete or failed acceptance', errors)
            self.assertIn('evidence digest mismatch', errors)

    def test_missing_and_external_artifacts_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            result, evidence = self.fixture(directory)
            evidence['artifacts'] = [{'path': '/etc/passwd', 'sha256': 'fake'},
                                     {'path': 'chainbench-out/missing-artifact', 'sha256': 'fake'}]
            self.write(directory, result, evidence)
            self.assertEqual(verify(directory).count('invalid artifact or digest'), 2)


if __name__ == '__main__':
    unittest.main()
