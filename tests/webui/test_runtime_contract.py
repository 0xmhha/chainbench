"""Unit tests for verifier isolation and fail-closed adapter contracts."""
import contextlib
import os
from pathlib import Path
import tempfile
import types
import unittest
from unittest.mock import patch
from capture import verify_published
from recheck import check
from runtime_contract import source_digest, stage_sources


class RuntimeContractTests(unittest.TestCase):
    def test_verifier_returned_errors_cannot_be_ignored(self):
        for errors in [['missing scenario'], ['binary provenance mismatch']]:
            with patch('capture.importlib.import_module', return_value=types.SimpleNamespace(verify=lambda _: errors)):
                with self.assertRaises(AssertionError):
                    verify_published('validator', Path('.'))

    def test_staging_preserves_source_and_excludes_receipts_and_state(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory) / 'project'; root.mkdir()
            (root / 'app.go').write_text('package app\n')
            for name in ['.git', '.ouroboros', '.aws', 'chainbench-out']:
                (root / name).mkdir(); (root / name / 'state.json').write_text('private-state')
            (root / 'node_modules/.bin').mkdir(parents=True)
            (root / 'node_modules/tool.js').write_text('tool')
            (root / 'node_modules/tool/bin').mkdir(parents=True)
            (root / 'node_modules/tool/bin/cli.js').write_text('cli')
            (root / 'node_modules/.bin/tool').symlink_to('../tool.js')
            expected = source_digest(root)
            stage = stage_sources(root, Path(directory))
            self.assertEqual(source_digest(stage), expected)
            self.assertTrue((stage / 'node_modules/.bin/tool').is_symlink())
            self.assertTrue((stage / 'node_modules/tool/bin/cli.js').is_file())
            for name in ['.git', '.ouroboros', '.aws', 'chainbench-out']:
                self.assertFalse((stage / name).exists())
            (root / 'app.go').write_text('package changed\n')
            self.assertNotEqual(source_digest(root), expected)

    def test_missing_or_stale_evidence_never_launches_or_writes(self):
        with tempfile.TemporaryDirectory() as directory:
            previous = Path.cwd()
            try:
                os.chdir(directory)
                with patch('recheck.subprocess.run') as runner:
                    with self.assertRaisesRegex(ValueError, 'Missing published'):
                        check('WEB-03', 'chainbench-out/acceptance')
                    self.assertFalse(Path('chainbench-out').exists())
                    artifact = Path('chainbench-out/acceptance/WEB-03/evidence.json')
                    artifact.parent.mkdir(parents=True); artifact.write_text('{"sourceDigest":"stale"}')
                    with self.assertRaisesRegex(ValueError, 'different source'):
                        check('WEB-03', 'chainbench-out/acceptance')
                    self.assertEqual(artifact.read_text(), '{"sourceDigest":"stale"}')
                    runner.assert_not_called()
            finally:
                os.chdir(previous)


if __name__ == '__main__':
    unittest.main()
