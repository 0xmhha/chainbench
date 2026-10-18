"""Preservation contracts for destructive runtime maintenance."""
import contextlib
import io
import os
import shutil
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

    def test_wal_audit_aliases_are_refused_before_copying(self):
        for alias in ('same path', 'normalized path', 'hard link'):
            with self.subTest(alias=alias):
                runtime = self.fixture('audit-' + alias.replace(' ', '-'))
                path = runtime / 'etcd/member/wal/0.tmp'
                path.parent.mkdir(parents=True); path.write_bytes(b'original proof')
                audit = path
                if alias == 'normalized path':
                    audit = path.parent / '../wal/0.tmp'
                elif alias == 'hard link':
                    audit = self.base / 'linked-audit'; os.link(path, audit)
                before = (path.read_bytes(), path.stat().st_ino, path.stat().st_mtime_ns)
                with patch.object(storage.sys, 'platform', 'darwin'), patch.object(storage, 'wal_candidate', side_effect=lambda p: p == path), patch.object(storage, 'inventory', return_value=('', [])), patch.object(storage, 'metadata', return_value=(0o600, os.getuid(), os.getgid(), 0, {}, [], 0)), patch.object(storage.subprocess, 'run', side_effect=lambda command, **kwargs: shutil.copy2(path, command[-1])) as run:
                    with self.assertRaisesRegex(storage.UnsafeMaintenance, 'audit'):
                        storage.compress_wal(self.root, audit, compress=True)
                    run.assert_not_called()
                self.assertEqual((path.read_bytes(), path.stat().st_ino, path.stat().st_mtime_ns), before)
                self.assertEqual(list(path.parent.glob('*.compress-*')), [])

    def test_wal_audit_descriptor_alias_is_refused_before_replacement(self):
        for alias in ('source', 'compressed copy', 'unrelated descriptor', 'late symlink'):
            with self.subTest(alias=alias):
                runtime = self.fixture('descriptor-' + alias.replace(' ', '-'))
                path = runtime / 'etcd/member/wal/0.tmp'
                path.parent.mkdir(parents=True); path.write_bytes(b'original proof' * 2048)
                audit = self.base / ('audit-' + runtime.name); audit.write_text('previous audit\n')
                unrelated = runtime / 'unrelated-proof'; unrelated.write_text('keep this proof\n')
                before = (path.read_bytes(), path.stat().st_ino, path.stat().st_mtime_ns)
                original_open, original_stat = Path.open, Path.stat
                def switched_open(p, mode='r', *args, **kwargs):
                    target = p
                    if p == audit and mode == 'a':
                        target = path if alias == 'source' else unrelated
                        if alias == 'compressed copy':
                            target = next(path.parent.glob('*.compress-*'))
                        elif alias == 'late symlink':
                            audit.unlink(); audit.symlink_to(unrelated); target = audit
                    return original_open(target, mode, *args, **kwargs)
                def sparse_stat(p, **kwargs):
                    result = original_stat(p, **kwargs)
                    if '.compress-' in p.name:
                        from types import SimpleNamespace
                        values = {name: getattr(result, name) for name in dir(result) if name.startswith('st_')}
                        values['st_blocks'] = 0
                        return SimpleNamespace(**values)
                    return result
                def copied(command, **kwargs):
                    shutil.copy2(path, Path(command[-1]))
                with patch.object(storage.sys, 'platform', 'darwin'), patch.object(storage, 'wal_candidate', side_effect=lambda p: p == path), patch.object(storage, 'inventory', return_value=('', [])), patch.object(storage, 'metadata', return_value=(0o600, os.getuid(), os.getgid(), 0, {}, [], 0)), patch.object(storage.subprocess, 'run', side_effect=copied), patch.object(Path, 'open', switched_open), patch.object(Path, 'stat', sparse_stat):
                    with self.assertRaisesRegex(storage.UnsafeMaintenance, 'audit'):
                        storage.compress_wal(self.root, audit, compress=True)
                self.assertEqual((path.read_bytes(), path.stat().st_ino, path.stat().st_mtime_ns), before)
                self.assertEqual(audit.read_text(), 'keep this proof\n' if alias == 'late symlink' else 'previous audit\n')
                self.assertEqual(unrelated.read_text(), 'keep this proof\n')
                self.assertEqual(list(path.parent.glob('*.compress-*')), [])

    def test_wal_compressor_timeout_preserves_source_and_audit(self):
        runtime = self.fixture()
        path = runtime / 'etcd/member/wal/0.tmp'
        path.parent.mkdir(parents=True); path.write_bytes(b'original proof')
        audit = self.base / 'audit'; audit.write_text('previous audit\n')
        before = (path.read_bytes(), path.stat().st_ino, path.stat().st_mtime_ns)
        def timed_out(command, **kwargs):
            self.assertEqual(kwargs.get('timeout'), 60)
            raise subprocess.TimeoutExpired(command, 60)
        with patch.object(storage.sys, 'platform', 'darwin'), patch.object(storage, 'wal_candidate', side_effect=lambda p: p == path), patch.object(storage, 'inventory', return_value=('', [])), patch.object(storage, 'metadata', return_value=()), patch.object(storage.subprocess, 'run', side_effect=timed_out):
            with self.assertRaisesRegex(storage.UnsafeMaintenance, 'timed out'):
                storage.compress_wal(self.root, audit, compress=True)
        self.assertEqual((path.read_bytes(), path.stat().st_ino, path.stat().st_mtime_ns), before)
        self.assertEqual(audit.read_text(), 'previous audit\n')
        self.assertEqual(list(path.parent.glob('*.compress-*')), [])

    def test_inventory_timeouts_refuse_maintenance(self):
        for command in ('ps', 'lsof'):
            with self.subTest(command=command):
                def timed_out(args, **kwargs):
                    self.assertEqual(kwargs.get('timeout'), 30)
                    if args[0] == command:
                        raise subprocess.TimeoutExpired(args, 30)
                    return subprocess.CompletedProcess(args, 0, '', '')
                with patch.object(storage.subprocess, 'run', side_effect=timed_out):
                    with self.assertRaisesRegex(storage.UnsafeMaintenance, 'timed out'):
                        storage.inventory()

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
