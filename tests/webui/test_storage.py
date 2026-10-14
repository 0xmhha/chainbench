"""Preservation contracts for destructive runtime maintenance."""
import contextlib
import io
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import time
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'scripts'))
import webui_storage as storage
import storage_probe as probe


class StorageTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.base = Path(self.temporary.name).resolve()
        self.root = self.base / 'runtime'; self.root.mkdir()
        self.repo = self.base / 'repo'; self.repo.mkdir()
        self.now = time.time() + 7200

    def fixture(self, name='00000000-0000-0000-0000-000000000001', criterion=None):
        path = self.root / name; path.mkdir()
        if criterion:
            (path / 'source/chainbench-out/web-ui-acceptance' / criterion).mkdir(parents=True)
        return path

    def plan(self, extras=()):
        with patch.object(storage, 'inventory', return_value=('', [])):
            return dict(storage.cleanup_plan(self.root, self.repo, 0, extras, self.now))

    def test_process_inventory_failure_refuses_maintenance(self):
        responses = [subprocess.CompletedProcess([], 1, '', 'denied'), subprocess.CompletedProcess([], 1, '', '')]
        with patch.object(storage.subprocess, 'run', side_effect=responses):
            with self.assertRaisesRegex(storage.UnsafeMaintenance, 'process'):
                storage.inventory()

    def test_lsof_failure_and_partial_warning_refuse_maintenance(self):
        for code, stderr in [(2, ''), (1, 'cannot stat filesystem')]:
            responses = [subprocess.CompletedProcess([], 0, '', ''), subprocess.CompletedProcess([], code, '', stderr)]
            with patch.object(storage.subprocess, 'run', side_effect=responses):
                with self.assertRaises(storage.UnsafeMaintenance):
                    storage.inventory()

    def test_referenced_recheck_survives_newer_unverified_copy(self):
        old = self.fixture('recheck-old', 'WEB-05')
        new = self.fixture('recheck-new', 'WEB-05')
        archive = self.repo / 'chainbench-out'; archive.mkdir()
        (archive / 'result.json').write_text('{"independentVerification": "' + str(old) + '/source"}')
        plan = self.plan()
        self.assertEqual(plan[old], 'named by evidence')
        self.assertEqual(plan[new], 'latest recheck of WEB-05')

    def test_external_receipt_and_missing_archive(self):
        path = self.fixture()
        receipt = self.base / 'campaign.jsonl'; receipt.write_text(str(path))
        self.assertEqual(self.plan([receipt])[path], 'named by evidence')
        with self.assertRaisesRegex(storage.UnsafeMaintenance, 'missing evidence'):
            self.plan([self.base / 'missing.json'])

    def test_development_proof_documents_are_default_references(self):
        for name in ['network-capability-evidence.json', 'architecture/new-proof.jsonl', 'web-ui-claude-handoff.md']:
            with self.subTest(document=name):
                path = self.fixture('00000000-0000-0000-0000-' + str(len(list(self.root.iterdir())) + 1).zfill(12))
                document = self.repo / 'docs/dev' / name
                document.parent.mkdir(parents=True, exist_ok=True)
                document.write_text('proof: ' + str(path / 'browser-proof/receipt.json'))
                self.assertEqual(self.plan()[path], 'named by evidence')

    def test_root_handoff_is_a_default_reference(self):
        path = self.fixture()
        (self.repo / 'CLAUDE.md').write_text('Continue from ' + str(path / 'source'))
        self.assertEqual(self.plan()[path], 'named by evidence')

    def test_unreadable_development_document_refuses_maintenance(self):
        self.fixture()
        document = self.repo / 'docs/dev/new-proof.json'
        document.parent.mkdir(parents=True)
        document.write_text('{}')
        with patch.object(Path, 'read_text', side_effect=PermissionError('denied')):
            with self.assertRaises(PermissionError):
                self.plan()

    def test_receipt_audit_reports_missing_without_claiming_complete(self):
        present = self.base / 'receipt.json'; present.write_text('{}')
        missing = self.base / 'missing.json'
        result = probe.receipt_audit({str(present): storage.digest(present), str(missing): 'absent'})
        self.assertEqual(result, {'expected': 2, 'verified': 1, 'missing': [str(missing)], 'changed': [], 'complete': False})

    def test_receipt_audit_reports_changed_without_altering_bytes(self):
        receipt = self.base / 'receipt.json'; receipt.write_text('{}')
        result = probe.receipt_audit({str(receipt): 'wrong digest'})
        self.assertEqual(result['changed'], [str(receipt)])
        self.assertFalse(result['complete'])
        self.assertEqual(receipt.read_text(), '{}')

    def test_receipt_audit_is_complete_only_for_matching_existing_files(self):
        receipt = self.base / 'receipt.json'; receipt.write_text('{}')
        result = probe.receipt_audit({str(receipt): storage.digest(receipt)})
        self.assertEqual(result, {'expected': 1, 'verified': 1, 'missing': [], 'changed': [], 'complete': True})

    def test_unreadable_evidence_does_not_turn_into_no_references(self):
        archive = self.repo / 'chainbench-out'; archive.mkdir()
        (archive / 'evidence.json').write_text('{}')
        with patch.object(Path, 'read_text', side_effect=PermissionError('denied')):
            with self.assertRaises(PermissionError):
                self.plan()

    def test_reference_inside_an_older_recheck_is_retained(self):
        path = self.fixture()
        old = self.fixture('recheck-old', 'WEB-01')
        (old / 'source/chainbench-out/receipt.json').write_text(str(path))
        self.fixture('recheck-new', 'WEB-01')
        self.assertEqual(self.plan()[path], 'named by evidence')

    def test_process_arguments_and_open_files_protect_runtime(self):
        path = self.fixture()
        for state in [(str(path) + '/bin/node', []), ('', [str(path) + '/member/wal/file'])]:
            with patch.object(storage, 'inventory', return_value=state):
                plan = dict(storage.cleanup_plan(self.root, self.repo, 0, now=self.now))
                self.assertEqual(plan[path], 'active runtime')

    def test_unknown_recent_and_pre_session_directories_are_retained(self):
        known = self.fixture(); unknown = self.fixture('user-files')
        self.assertEqual(self.plan()[unknown], 'unknown runtime name')
        with patch.object(storage, 'inventory', return_value=('', [])):
            self.assertEqual(dict(storage.cleanup_plan(self.root, self.repo, 0))[known], 'changed in the last hour')
            self.assertEqual(dict(storage.cleanup_plan(self.root, self.repo, self.now, now=self.now))[known], 'older than this session')

    def test_symlink_root_and_nested_link_never_deleted(self):
        link = self.base / 'linked'; link.symlink_to(self.root)
        with self.assertRaises(storage.UnsafeMaintenance):
            storage.cleanup_plan(link, self.repo, 0)
        path = self.fixture(); (path / 'outside').symlink_to(self.repo)
        self.assertIn('symlink', self.plan()[path])

    def test_new_reference_before_apply_refuses_deletion(self):
        path = self.fixture()
        with patch.object(storage, 'cleanup_plan', side_effect=[[(path, None)], [(path, 'named by evidence')]]):
            with self.assertRaises(storage.UnsafeMaintenance):
                storage.cleanup(self.root, self.repo, 0, delete=True)
        self.assertTrue(path.exists())

    def test_dry_run_does_not_delete_candidate(self):
        path = self.fixture()
        with patch.object(storage, 'cleanup_plan', return_value=[(path, None)]), contextlib.redirect_stdout(io.StringIO()):
            storage.cleanup(self.root, self.repo, 0)
        self.assertTrue(path.exists())

    def test_wal_requires_correct_shape_age_size_and_single_link(self):
        runtime = self.fixture()
        path = runtime / 'etcd/member/wal/00000000.wal'; path.parent.mkdir(parents=True)
        with path.open('wb') as output:
            output.truncate(40 * 1024**2)
        self.assertFalse(storage.wal_candidate(path))
        self.assertTrue(storage.wal_candidate(path, self.now))
        os.link(path, path.with_suffix('.linked'))
        self.assertFalse(storage.wal_candidate(path, self.now))

    def test_active_wal_does_not_invoke_compressor(self):
        runtime = self.fixture()
        path = runtime / 'etcd/member/wal/0.tmp'; path.parent.mkdir(parents=True); path.touch()
        with patch.object(storage, 'wal_candidate', return_value=False):
            storage.compress_wal(self.root, self.base / 'audit')
        with patch.object(storage.sys, 'platform', 'darwin'), patch.object(storage, 'wal_candidate', side_effect=lambda p: p == path), patch.object(storage, 'inventory', return_value=('', [str(path)])), patch.object(storage.subprocess, 'run') as run:
            storage.compress_wal(self.root, self.base / 'audit', compress=True)
            run.assert_not_called()

    def test_compressor_that_changes_bytes_leaves_original_intact(self):
        runtime = self.fixture()
        path = runtime / 'etcd/member/wal/0.tmp'; path.parent.mkdir(parents=True); path.write_bytes(b'original')
        def corrupt(command, **kwargs):
            Path(command[-1]).write_bytes(b'corrupt')
        with patch.object(storage.sys, 'platform', 'darwin'), patch.object(storage, 'wal_candidate', side_effect=lambda p: p == path), patch.object(storage, 'inventory', return_value=('', [])), patch.object(storage, 'metadata', return_value=()), patch.object(storage.subprocess, 'run', side_effect=corrupt):
            with self.assertRaisesRegex(storage.UnsafeMaintenance, 'bytes or metadata'):
                storage.compress_wal(self.root, self.base / 'audit', compress=True)
        self.assertEqual(path.read_bytes(), b'original')
        self.assertEqual(list(path.parent.iterdir()), [path])

    def test_unknown_recheck_never_deleted(self):
        path = self.fixture('recheck-unknown')
        self.assertEqual(self.plan()[path], 'unknown recheck criterion')

    def test_same_birth_time_rechecks_are_both_kept(self):
        first = self.fixture('recheck-a', 'WEB-05')
        second = self.fixture('recheck-b', 'WEB-05')
        original_stat = Path.stat
        def equal_birth(path, **kwargs):
            result = original_stat(path, **kwargs)
            if path in (first, second):
                from types import SimpleNamespace
                return SimpleNamespace(st_birthtime=100, st_ctime=100, st_mtime=result.st_mtime, st_mode=result.st_mode)
            return result
        with patch.object(Path, 'stat', equal_birth):
            plan = self.plan()
        self.assertEqual(plan[first], 'latest recheck of WEB-05')
        self.assertEqual(plan[second], 'latest recheck of WEB-05')

    def test_repository_cannot_be_the_cleanup_root(self):
        with self.assertRaisesRegex(storage.UnsafeMaintenance, 'separate'):
            storage.cleanup_plan(self.repo, self.repo, 0)


if __name__ == '__main__':
    unittest.main()
