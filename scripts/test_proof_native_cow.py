"""Closed-proof binary compaction contracts; cloning is isolated in unit tests."""
import contextlib
import io
import hashlib
import json
import os
from pathlib import Path
import shutil
import tempfile
import subprocess
import unittest
from unittest.mock import patch
import proof_native_cow as compactor


class ProofNativeCopyTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name).resolve()
        self.sha = hashlib.sha256(b'x' * 1048576).hexdigest()
        self.pins = {self.sha}
        for name in ('a', 'b'):
            runtime = self.root / name; runtime.mkdir()
            marker = runtime / '.webui-owned-runtime.json'
            marker.write_text(json.dumps({'version': 1, 'path': str(runtime), 'policy': 'completed-disposable-data'}))
            marker.chmod(0o600)
            path = runtime / 'source/web/node_modules'
            records = [{'event': 'reclaim-start', 'path': str(path)},
                       {'event': 'reclaimed', 'path': str(path), 'allocatedBytes': 1,
                        'finishedAt': '2026-01-01T00:00:00+00:00'}]
            (runtime / 'storage-audit.jsonl').write_text(''.join(json.dumps(r) + '\n' for r in records))
        self.a = self.root / 'a/gstable'
        self.b = self.root / 'b/fork/binaries' / self.sha / 'gstable'
        self.b.parent.mkdir(parents=True)
        self.a.write_bytes(b'x' * 1048576); self.a.chmod(0o755); shutil.copy2(self.a, self.b)
        def portable_metadata(p):
            s = p.stat()
            return (s.st_mode & 0o7777, s.st_uid, s.st_gid, s.st_mtime_ns, {}, [], 0)
        self.meta = patch.object(compactor.maintenance, 'metadata', side_effect=portable_metadata)
        self.meta.start(); self.addCleanup(self.meta.stop)
        self.clone = patch.object(compactor, 'clone_command', side_effect=lambda a, b: shutil.copy2(a, b))
        self.clone.start(); self.addCleanup(self.clone.stop)
        self.ra, self.rb = compactor.describe(self.a), compactor.describe(self.b)
        self.audit = self.root / 'compaction.jsonl'

    def call(self, inventory=lambda: ('', [])):
        return compactor.clone_pair(self.ra, self.rb, self.root, self.pins, self.audit, inventory=inventory)

    def unchanged(self):
        self.assertEqual(compactor.describe(self.b), self.rb)
        self.assertEqual(list(self.b.parent.glob('.native-cow-*')), [])

    def test_clone_keeps_content_metadata_distinct_inode_and_parent_mtime(self):
        parent = self.b.parent.stat().st_mtime_ns
        self.call()
        self.assertEqual(compactor.identity(compactor.describe(self.b)), compactor.identity(self.rb))
        self.assertNotEqual(self.b.stat().st_ino, self.a.stat().st_ino)
        self.assertNotEqual(self.b.stat().st_ino, self.rb['inode'])
        self.assertEqual(self.b.parent.stat().st_mtime_ns, parent)
        self.assertEqual([json.loads(s)['event'] for s in self.audit.read_text().splitlines()], ['clone-intent', 'clone-complete'])

    def test_target_mtime_is_preserved_instead_of_anchor_time(self):
        s = self.b.stat(); os.utime(self.b, ns=(s.st_atime_ns, s.st_mtime_ns + 22))
        self.rb = compactor.describe(self.b); self.call()
        self.assertEqual(compactor.describe(self.b)['mtimeNS'], self.rb['mtimeNS'])

    def test_asset_payload_scope(self):
        p = self.root / 'b/store/assets' / ('a' * 32) / 'payload'; p.parent.mkdir(parents=True)
        self.b.rename(p); self.b = p; self.rb = compactor.describe(p); self.call()
        self.assertEqual(compactor.identity(compactor.describe(p)), compactor.identity(self.rb))

    def test_refuses_unpinned_source(self):
        self.pins.clear()
        with self.assertRaises(compactor.maintenance.UnsafeMaintenance): self.call()
        self.unchanged()

    def test_refuses_changed_content_permissions_and_hardlinks(self):
        for change in ('content', 'mode', 'hardlink'):
            with self.subTest(change=change):
                if change == 'content': self.b.write_bytes(b'y' * 1048576)
                elif change == 'mode': shutil.copy2(self.a, self.b); self.b.chmod(0o700)
                else: shutil.copy2(self.a, self.b); os.link(self.b, self.root / 'user-link')
                before = compactor.describe(self.b)
                with self.assertRaises(compactor.maintenance.UnsafeMaintenance): self.call()
                self.assertEqual(compactor.describe(self.b), before)

    def test_refuses_symlink_and_unknown_scope(self):
        self.b.unlink(); self.b.symlink_to(self.a)
        with self.assertRaises(compactor.maintenance.UnsafeMaintenance): self.call()
        self.b.unlink(); p = self.root / 'b/user-data'; shutil.copy2(self.a, p)
        self.b = p; self.rb = compactor.describe(p)
        with self.assertRaises(compactor.maintenance.UnsafeMaintenance): self.call()
        self.unchanged()

    def test_refuses_missing_pending_future_or_unpaired_completion(self):
        path = self.root / 'b/storage-audit.jsonl'; original = path.read_text()
        for state in ('missing', 'pending', 'future', 'unpaired'):
            with self.subTest(state=state):
                if path.exists(): path.unlink()
                rows = [json.loads(s) for s in original.splitlines()]
                if state == 'pending': rows.append(rows[0])
                elif state == 'future': rows[-1]['finishedAt'] = '2099-01-01T00:00:00+00:00'
                elif state == 'unpaired': rows = rows[1:]
                if state != 'missing': path.write_text(''.join(json.dumps(r) + '\n' for r in rows))
                with self.assertRaises(compactor.maintenance.UnsafeMaintenance): self.call()
                self.unchanged()

    def test_refuses_active_and_becoming_active_runtime(self):
        for states in [[(str(self.root / 'b'), [])], [('', []), ('', [str(self.b)])]]:
            with self.subTest(states=states):
                iterator = iter(states)
                with self.assertRaises(compactor.maintenance.UnsafeMaintenance): self.call(lambda: next(iterator))
                self.unchanged()

    def test_refuses_audit_alias_without_appending_native_bytes(self):
        self.audit = self.a
        with self.assertRaises(compactor.maintenance.UnsafeMaintenance): self.call()
        self.assertEqual(compactor.describe(self.a), self.ra); self.unchanged()

    def test_failed_or_corrupt_clone_preserves_target_and_parent(self):
        parent = self.b.parent.stat().st_mtime_ns
        for failure in ('exception', 'corrupt'):
            with self.subTest(failure=failure):
                def clone(a, b):
                    b.write_bytes(b'incomplete')
                    if failure == 'exception': raise OSError('copy failed')
                with patch.object(compactor, 'clone_command', side_effect=clone):
                    with self.assertRaises((OSError, compactor.maintenance.UnsafeMaintenance)): self.call()
                self.unchanged(); self.assertEqual(self.b.parent.stat().st_mtime_ns, parent)

    def test_supports_completed_triecache_and_etcd_records(self):
        journal = self.root / 'b/storage-audit.jsonl'
        for name in ('triecache', 'etcd'):
            path = self.root / 'b/w/node1/geth' / name
            with journal.open('a') as f:
                for r in [{'event': 'reclaim-start', 'path': str(path)}, {'event': 'reclaimed', 'path': str(path), 'allocatedBytes': 0, 'finishedAt': '2026-01-01T00:00:00+00:00'}]: f.write(json.dumps(r) + '\n')
        self.call()

    def test_reappearing_reclaimed_directory_is_not_closed(self):
        path = self.root / 'b/source/web/node_modules'; path.mkdir(parents=True)
        with self.assertRaises(compactor.maintenance.UnsafeMaintenance): self.call()
        self.unchanged()

    def test_redirected_completion_ancestor_is_refused(self):
        outside = self.root / 'user-files'; outside.mkdir()
        (self.root / 'b/source').symlink_to(outside, target_is_directory=True)
        with self.assertRaises(compactor.maintenance.UnsafeMaintenance): self.call()
        self.unchanged()

    def catalog(self):
        rows = []
        for family in ('stablenet', 'wbft', 'wemix'):
            p = self.root / ('catalog-' + family); p.write_bytes(family.encode()); p.chmod(0o755)
            rows.append({'chain': family, 'binary': str(p), 'sha256': hashlib.sha256(p.read_bytes()).hexdigest()})
        p = self.root / 'catalog.json'; p.write_text(json.dumps({'binaries': rows}))
        return p, rows

    def test_catalog_refuses_changed_or_nonexecutable_binary(self):
        catalog, rows = self.catalog(); self.assertEqual(len(compactor.load_pins(catalog)), 3)
        binary = Path(rows[0]['binary']); binary.chmod(0o600)
        with self.assertRaises(compactor.maintenance.UnsafeMaintenance): compactor.load_pins(catalog)
        binary.chmod(0o755); binary.write_bytes(b'changed')
        with self.assertRaises(compactor.maintenance.UnsafeMaintenance): compactor.load_pins(catalog)

    def test_plan_selects_private_target_and_skips_shared_target(self):
        original = compactor.describe(self.b)
        pairs, _ = compactor.plan(self.root, self.pins, measure=lambda p: 0 if p == self.a else 1048576, minimum=1)
        self.assertEqual(len(pairs), 1); self.assertEqual(pairs[0]['target'], self.rb)
        self.assertEqual(compactor.plan(self.root, self.pins, measure=lambda p: 0, minimum=1)[0], [])
        self.assertEqual(compactor.describe(self.b), original)

    def test_cli_default_is_dry_and_apply_requires_audit(self):
        with patch.object(compactor, 'load_pins', return_value=self.pins), patch.object(compactor, 'plan', return_value=([{'anchor': self.ra, 'target': self.rb}], [])), patch.object(compactor, 'clone_pair') as clone, contextlib.redirect_stdout(io.StringIO()) as out:
            compactor.main(['--root', str(self.root), '--catalog', 'ignored.json'])
            clone.assert_not_called(); self.assertFalse(json.loads(out.getvalue())['applied'])
        with contextlib.redirect_stderr(io.StringIO()), self.assertRaises(SystemExit):
            compactor.main(['--root', str(self.root), '--catalog', 'ignored.json', '--apply'])
        self.unchanged()

    def test_cli_refuses_existing_audit_before_mutation(self):
        self.audit.write_text('user proof\n')
        pair = {'anchor': self.ra, 'target': self.rb, 'privateBytesBefore': 1048576}
        with patch.object(compactor, 'load_pins', return_value=self.pins), patch.object(compactor, 'plan', return_value=([pair], [])), patch.object(compactor, 'clone_pair', return_value={}) as clone, patch.object(compactor, 'private_bytes', return_value=0), contextlib.redirect_stdout(io.StringIO()):
            with self.assertRaises(FileExistsError):
                compactor.main(['--root', str(self.root), '--catalog', 'ignored.json', '--apply', '--audit', str(self.audit)])
            clone.assert_not_called()
        self.assertEqual(self.audit.read_text(), 'user proof\n'); self.unchanged()

    def test_completion_change_during_clone_refuses_replacement(self):
        def changed(a, b):
            shutil.copy2(a, b)
            journal = self.root / 'b/storage-audit.jsonl'
            with journal.open('a') as f: f.write(json.dumps({'event': 'reclaim-start', 'path': str(self.root / 'b/w/node1/etcd')}) + '\n')
        with patch.object(compactor, 'clone_command', side_effect=changed):
            with self.assertRaises(compactor.maintenance.UnsafeMaintenance): self.call()
        self.unchanged()

    def test_clone_timeout_keeps_original_and_parent(self):
        parent = self.b.parent.stat().st_mtime_ns
        self.clone.stop()
        def timed_out(command, **kwargs):
            self.assertEqual(kwargs.get('timeout'), 60)
            Path(command[-1]).write_bytes(b'partial')
            raise subprocess.TimeoutExpired(command, 60)
        with patch.object(compactor.sys, 'platform', 'darwin'), patch.object(compactor.maintenance.subprocess, 'run', side_effect=timed_out):
            with self.assertRaises(compactor.maintenance.UnsafeMaintenance): self.call()
        self.unchanged(); self.assertEqual(self.b.parent.stat().st_mtime_ns, parent)



if __name__ == '__main__':
    unittest.main()
