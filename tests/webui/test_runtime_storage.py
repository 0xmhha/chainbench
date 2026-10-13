"""Contracts for automatic cleanup of completed, exclusively owned proofs."""
import ast
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
import runtime_storage as storage


class RuntimeStorageTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name).resolve()
        self.environment = patch.dict(os.environ, WEBUI_RUNTIME_ROOT=str(self.root))
        self.environment.start(); self.addCleanup(self.environment.stop)

    def runtime(self):
        with patch.object(storage, 'preflight'):
            return storage.create_runtime()

    def fixture(self, runtime):
        db = runtime / 'store/networks/test/nodes/node1/etcd/member/wal/0.tmp'
        db.parent.mkdir(parents=True); db.write_bytes(b'disposable')
        config = db.parents[3] / 'config.toml'; config.write_text('config')
        binary = runtime / 'dashboard'; binary.write_bytes(b'binary')
        return db, config, binary

    def reclaim(self, runtime, references=()):
        with patch.object(storage.maintenance, 'inventory', return_value=('', [])):
            return storage.reclaim_runtime(runtime, references)

    def test_completion_removes_only_disposable_data_and_preserves_proof(self):
        runtime = self.runtime(); db, config, binary = self.fixture(runtime)
        self.reclaim(runtime)
        self.assertFalse(db.exists())
        self.assertEqual(config.read_text(), 'config')
        self.assertEqual(binary.read_bytes(), b'binary')
        self.assertTrue((runtime / 'storage-audit.jsonl').is_file())

    def test_explicitly_referenced_database_is_preserved(self):
        runtime = self.runtime(); db, _, _ = self.fixture(runtime)
        self.reclaim(runtime, [str(db)])
        self.assertEqual(db.read_bytes(), b'disposable')

    def test_exported_snapshots_and_source_fixtures_are_not_node_data(self):
        runtime = self.runtime()
        paths = [runtime / prefix / 'node1/etcd/member/wal/0.tmp'
                 for prefix in ('source/tests/fixtures', 'store/snapshots', 'store/sessions', 'user-data')]
        for path in paths:
            path.parent.mkdir(parents=True); path.write_bytes(b'proof')
        self.reclaim(runtime, [str(paths[-1])])
        for path in paths:
            self.assertEqual(path.read_bytes(), b'proof')

    def test_activity_reappearing_before_removal_refuses_cleanup(self):
        runtime = self.runtime(); db, _, _ = self.fixture(runtime)
        with patch.object(storage.maintenance, 'inventory', side_effect=[('', []), ('', [str(db)])]):
            with self.assertRaises(storage.maintenance.UnsafeMaintenance):
                storage.reclaim_runtime(runtime)
        self.assertTrue(db.exists())

    def test_unknown_or_legacy_runtime_is_never_reclaimed(self):
        runtime = self.root / 'user-data'; runtime.mkdir(); db, _, _ = self.fixture(runtime)
        with self.assertRaises(storage.maintenance.UnsafeMaintenance):
            self.reclaim(runtime)
        self.assertTrue(db.exists())

    def test_inventory_failure_active_process_or_open_file_refuses_cleanup(self):
        for inventory in [RuntimeError('unavailable'), (str(self.root), []), ('', [])]:
            runtime = self.runtime(); db, _, _ = self.fixture(runtime)
            state = inventory if isinstance(inventory, Exception) else (str(runtime), []) if inventory[0] else ('', [str(db)])
            with patch.object(storage.maintenance, 'inventory', side_effect=state if isinstance(state, Exception) else None, return_value=state):
                with self.assertRaises((RuntimeError, storage.maintenance.UnsafeMaintenance)):
                    storage.reclaim_runtime(runtime)
            self.assertTrue(db.exists())

    def test_nested_symlink_and_redirected_root_refuse_cleanup(self):
        runtime = self.runtime(); db, _, _ = self.fixture(runtime)
        (db.parent / 'linked').symlink_to(self.root)
        with self.assertRaises(storage.maintenance.UnsafeMaintenance):
            self.reclaim(runtime)
        self.assertTrue(db.exists())

    def test_ownership_marker_cannot_claim_another_runtime(self):
        first, second = self.runtime(), self.runtime()
        (second / storage.MARKER).write_bytes((first / storage.MARKER).read_bytes())
        with self.assertRaises(storage.maintenance.UnsafeMaintenance):
            self.reclaim(second)

    def test_staged_dependencies_are_removed_but_source_and_receipts_remain(self):
        runtime = self.runtime()
        dep = runtime / 'source/web/node_modules/tool'; dep.mkdir(parents=True)
        (dep / 'index.js').write_text('dependency')
        (dep.parent / 'linked').symlink_to('tool')
        src = runtime / 'source/app.go'; src.write_text('package app')
        proof = runtime / 'source/chainbench-out/receipt.json'; proof.parent.mkdir(); proof.write_text('{}')
        self.reclaim(runtime)
        self.assertFalse(dep.exists())
        self.assertTrue(src.exists()); self.assertTrue(proof.exists())

    def test_preflight_rejects_under_twenty_gib_before_creating_runtime(self):
        with patch.object(storage.shutil, 'disk_usage', return_value=type('Usage', (), {'free': 19 * 1024**3})()):
            with self.assertRaisesRegex(storage.maintenance.UnsafeMaintenance, '20 GiB'):
                storage.create_runtime()
        self.assertEqual(list(self.root.iterdir()), [])

    def test_copy_is_independent_and_preserves_mode_and_mtime(self):
        source = self.root / 'original'; source.write_bytes(b'original'); source.chmod(0o600)
        copy = self.root / 'copy'
        storage.clone_file(source, copy)
        self.assertEqual(copy.read_bytes(), source.read_bytes())
        self.assertEqual(copy.stat().st_mode, source.stat().st_mode)
        self.assertEqual(copy.stat().st_mtime_ns, source.stat().st_mtime_ns)
        copy.write_bytes(b'changed')
        self.assertEqual(source.read_bytes(), b'original')

    def test_repeated_completion_does_not_accumulate_databases(self):
        for _ in range(3):
            runtime = self.runtime(); db, _, _ = self.fixture(runtime)
            self.reclaim(runtime); self.reclaim(runtime)
            self.assertFalse(db.exists())
        self.assertEqual(list(self.root.rglob('0.tmp')), [])

    def test_standalone_fixtures_reclaim_after_process_shutdown(self):
        for name in ('run_test_jobs.py', 'run_native_jobs.py', 'run_ssh_jobs.py'):
            tree = ast.parse((Path(__file__).parent / name).read_text())
            main = next(n for n in tree.body if isinstance(n, ast.FunctionDef) and n.name == 'main')
            self.assertTrue(isinstance(main.body[-1], ast.Expr) and
                            isinstance(main.body[-1].value, ast.Call) and
                            isinstance(main.body[-1].value.func, ast.Name) and
                            main.body[-1].value.func.id == 'reclaim_evidence', name)

    def test_capture_collects_runtimes_and_artifact_paths_without_deleting_receipts(self):
        runtime = self.runtime(); db, _, _ = self.fixture(runtime)
        out = self.root / 'evidence'; out.mkdir()
        receipt = out / 'evidence.json'
        receipt.write_text(json.dumps({'fixture': {'runtime': str(runtime)}, 'artifacts': [{'path': str(db)}]}))
        before = receipt.read_bytes()
        with patch.object(storage.maintenance, 'inventory', return_value=('', [])):
            storage.reclaim_evidence(out)
        self.assertEqual(receipt.read_bytes(), before)
        self.assertTrue(db.exists())


if __name__ == '__main__':
    unittest.main()
