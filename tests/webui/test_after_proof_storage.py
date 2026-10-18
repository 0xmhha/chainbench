"""Keep proof verdicts and retention failures visible at the runner boundary."""
import importlib
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch


class VerifyStorageHookTests(unittest.TestCase):
    def test_capture_and_recheck_always_maintain_storage_without_hiding_failure(self):
        script = Path(__file__).parent / 'verify.sh'
        with tempfile.TemporaryDirectory() as name:
            root = Path(name); interpreter = root / 'python3'
            interpreter.write_text('''#!/bin/sh
if [ "$1" = '-B' ]; then shift; fi
printf '%s\\n' "$1" >> "$INVOKE_LOG"
case "$1" in
  tests/webui/capture.py|tests/webui/recheck.py) exit "$PROOF_EXIT" ;;
  tests/webui/after_proof_storage.py) exit "$COMPACTION_EXIT" ;;
  *) exit 99 ;;
esac
''')
            interpreter.chmod(0o755)
            for capture in (True, False):
                for proof, retention in ((0, 0), (23, 0), (0, 24), (23, 24)):
                    with self.subTest(capture=capture, proof=proof, retention=retention):
                        log = root / 'invocations'; log.unlink(missing_ok=True)
                        args = ['bash', str(script), '--criterion', 'WEB-01', '--require-live']
                        if capture: args.append('--capture')
                        result = subprocess.run(args, env=dict(os.environ, PATH=str(root) + os.pathsep + os.environ['PATH'], INVOKE_LOG=str(log), PROOF_EXIT=str(proof), COMPACTION_EXIT=str(retention)), capture_output=True, text=True)
                        self.assertEqual(result.returncode, proof or retention)
                        self.assertEqual(log.read_text().splitlines(), ['tests/webui/capture.py' if capture else 'tests/webui/recheck.py', 'tests/webui/after_proof_storage.py'])


class AfterProofPolicyTests(unittest.TestCase):
    def setUp(self):
        self.hook = importlib.import_module('after_proof_storage')
        self.temp = tempfile.TemporaryDirectory(); self.addCleanup(self.temp.cleanup)
        base = Path(self.temp.name).resolve(); self.workspace = base / 'repo'; self.workspace.mkdir()
        self.root = base / 'runtime'; self.root.mkdir()
        self.source = self.workspace / 'source.go'; self.source.write_text('package demo\n')
        self.directory = self.workspace / 'chainbench-out/proofs/WEB-01'; self.directory.mkdir(parents=True)
        self.proof = self.directory / 'evidence.json'; self.proof.write_text('{"original":true}')
        self.catalog = base / 'catalog.json'
        self.pair = {'anchor': {'path': str(self.root / 'a/gwbft')}, 'target': {'path': str(self.root / 'b/gwbft')}, 'privateBytesBefore': 1048576}
        self.env = patch.dict(os.environ, WEBUI_RUNTIME_ROOT=str(self.root), WEBUI_SELECTED_BINARIES=str(self.catalog), WEBUI_REPLACEMENT_BINARY='', WEBUI_STORAGE_EXTRA_BINARIES='[]')
        self.env.start(); self.addCleanup(self.env.stop)
        self.platform = patch.object(self.hook.sys, 'platform', 'darwin'); self.platform.start(); self.addCleanup(self.platform.stop)
        for name, options in [('load_pins', {'return_value': {'known'}}), ('plan', {'return_value': ([self.pair], [])}), ('clone_pair', {'return_value': {'contentAndMetadataUnchanged': True, 'independentInode': True}}), ('private_bytes', {'return_value': 0})]:
            item = patch.object(self.hook.compactor, name, **options); mock = item.start(); self.addCleanup(item.stop); setattr(self, name, mock)

    def run_hook(self):
        return self.hook.run('WEB-01', 'chainbench-out/proofs', self.workspace)

    def failure(self):
        with self.assertRaises(self.hook.compactor.maintenance.UnsafeMaintenance): self.run_hook()
        reports = list((self.root / '.storage-maintenance').glob('*.json'))
        self.assertEqual(len(reports), 1)
        self.assertEqual(reports[0].stat().st_mode & 0o777, 0o600)
        return json.loads(reports[0].read_text())

    def test_completes_with_private_report_and_preserved_source_and_proof(self):
        before = self.hook.proof_snapshot(self.directory)
        result = self.run_hook(); self.assertEqual(result['targets'], 1)
        data = json.loads(Path(result['report']).read_text())
        self.assertEqual(data['status'], 'completed')
        self.assertTrue(data['protectedSourceUnchanged']); self.assertTrue(data['publishedProofUnchanged'])
        self.assertEqual(before, self.hook.proof_snapshot(self.directory))
        self.assertEqual(self.source.read_text(), 'package demo\n')
        self.assertEqual(Path(data['audit']).stat().st_mode & 0o777, 0o600)

    def test_missing_catalog_skips_without_claiming_cleanup(self):
        os.environ['WEBUI_SELECTED_BINARIES'] = ''
        result = self.run_hook(); self.assertEqual(result['status'], 'skipped'); self.assertEqual(result['targets'], 0)
        self.plan.assert_not_called(); self.assertFalse((self.root / '.storage-maintenance').exists())

    def test_nonmacos_skips_without_mutations(self):
        with patch.object(self.hook.sys, 'platform', 'linux'): result = self.run_hook()
        self.assertEqual(result['status'], 'skipped'); self.plan.assert_not_called()

    def test_loader_failure_retains_report_without_modifying_proof(self):
        self.load_pins.side_effect = ValueError('invalid catalog')
        data = self.failure(); self.assertEqual(data['status'], 'incomplete')
        self.assertTrue(data['publishedProofUnchanged']); self.assertTrue(data['protectedSourceUnchanged'])
        self.clone_pair.assert_not_called()

    def test_native_failure_is_not_reported_as_completed(self):
        self.clone_pair.side_effect = RuntimeError('copy refused')
        data = self.failure(); self.assertEqual(data['status'], 'incomplete')
        self.assertEqual(data['results'], []); self.assertTrue(data['publishedProofUnchanged'])

    def test_source_change_is_reported_without_reverting_user_edit(self):
        def changed(*args, **kwargs):
            self.source.write_text('package changed\n')
            return {'contentAndMetadataUnchanged': True, 'independentInode': True}
        self.clone_pair.side_effect = changed
        data = self.failure(); self.assertFalse(data['protectedSourceUnchanged'])
        self.assertEqual(self.source.read_text(), 'package changed\n')

    def test_published_proof_change_refuses_success(self):
        def changed(*args, **kwargs):
            self.proof.write_text('{"changed":true}')
            return {'contentAndMetadataUnchanged': True, 'independentInode': True}
        self.clone_pair.side_effect = changed
        data = self.failure(); self.assertFalse(data['publishedProofUnchanged'])

    def test_extra_binary_paths_are_typed_and_deduplicated(self):
        extra = str(self.root / 'linux-binary'); replacement = str(self.root / 'replacement')
        os.environ['WEBUI_STORAGE_EXTRA_BINARIES'] = json.dumps([extra, extra]); os.environ['WEBUI_REPLACEMENT_BINARY'] = replacement
        self.run_hook(); self.assertEqual(self.load_pins.call_args.args[1], [extra, replacement])

    def test_invalid_extra_input_is_not_silently_skipped(self):
        os.environ['WEBUI_STORAGE_EXTRA_BINARIES'] = '{"path":"wrong shape"}'
        data = self.failure(); self.assertEqual(data['status'], 'incomplete'); self.load_pins.assert_not_called()

    def test_zero_targets_keep_skip_reasons_without_freed_space_claim(self):
        self.plan.return_value = ([], [{'runtime': 'busy', 'reason': 'active runtime'}])
        result = self.run_hook(); data = json.loads(Path(result['report']).read_text())
        self.assertEqual(result['targets'], 0); self.assertEqual(data['skipped'][0]['reason'], 'active runtime')
        self.clone_pair.assert_not_called(); self.assertFalse(Path(data['audit']).exists())



if __name__ == '__main__':
    unittest.main()
