"""Contracts for automatic cleanup of completed, exclusively owned proofs."""
import ast
import contextlib
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
import runtime_storage as storage
import capture
import recheck


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

    def test_browser_test_dependencies_do_not_accumulate_across_completions(self):
        for _ in range(3):
            runtime = self.runtime()
            dependency = runtime / 'source/tests/webui/node_modules/playwright/index.js'
            dependency.parent.mkdir(parents=True); dependency.write_text('installed browser dependency')
            lock = runtime / 'source/tests/webui/package-lock.json'; lock.write_text('{}'); lock.chmod(0o600)
            original = (lock.read_bytes(), lock.stat().st_mode, lock.stat().st_mtime_ns)
            out = runtime / 'source/chainbench-out/proof'; out.mkdir(parents=True)
            (out / 'partial.json').write_text('{"outcome":"incomplete"}')
            with patch.object(storage.maintenance, 'inventory', return_value=('', [])):
                storage.reclaim_after_shutdown(runtime, out)
            self.assertFalse(dependency.exists())
            self.assertEqual((lock.read_bytes(), lock.stat().st_mode, lock.stat().st_mtime_ns), original)

    def test_explicitly_cited_dependency_artifacts_are_preserved(self):
        for prefix in ('source/node_modules', 'source/web/node_modules', 'source/tests/webui/node_modules'):
            with self.subTest(prefix=prefix):
                runtime = self.runtime()
                artifact = runtime / prefix / 'tool/index.js'
                artifact.parent.mkdir(parents=True); artifact.write_text('cited artifact')
                self.reclaim(runtime, [str(artifact)])
                self.assertEqual(artifact.read_text(), 'cited artifact')

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
            lifecycle = next(n for n in main.body if isinstance(n, ast.Try) and n.finalbody)
            final = lifecycle.finalbody[-1]
            self.assertTrue(isinstance(final, ast.Expr) and isinstance(final.value, ast.Call) and
                            isinstance(final.value.func, ast.Name) and
                            final.value.func.id == 'reclaim_after_shutdown', name)

    def test_failed_partial_proof_without_success_receipt_reclaims_only_unreferenced_data(self):
        runtime = self.runtime(); db, config, binary = self.fixture(runtime)
        out = self.root / 'failed'; out.mkdir()
        log = out / 'browser.log'; log.write_text('original assertion failed')
        evidence = out / 'partial.json'; evidence.write_text('{"outcome":"incomplete"}')
        protected = {p: (p.read_bytes(), p.stat().st_mode, p.stat().st_mtime_ns)
                     for p in (config, binary, log, evidence)}
        with patch.object(storage.maintenance, 'inventory', return_value=('', [])):
            storage.reclaim_after_shutdown(runtime, out)
        self.assertFalse(db.exists())
        for path, expected in protected.items():
            self.assertEqual((path.read_bytes(), path.stat().st_mode, path.stat().st_mtime_ns), expected)

    def test_failed_proof_explicit_database_evidence_survives_reclamation(self):
        runtime = self.runtime(); db, _, _ = self.fixture(runtime)
        out = self.root / 'failed'; out.mkdir()
        receipt = out / 'partial.json'; receipt.write_text(json.dumps({'artifact': str(db)}))
        with patch.object(storage.maintenance, 'inventory', return_value=('', [])):
            storage.reclaim_after_shutdown(runtime, out)
        self.assertEqual(db.read_bytes(), b'disposable')

    def test_partial_jsonl_database_evidence_is_preserved(self):
        runtime = self.runtime(); db, _, _ = self.fixture(runtime)
        out = self.root / 'failed'; out.mkdir()
        (out / 'observations.jsonl').write_text(json.dumps({'artifact': str(db)}) + '\n{"partial":')
        with patch.object(storage.maintenance, 'inventory', return_value=('', [])):
            storage.reclaim_after_shutdown(runtime, out)
        self.assertEqual(db.read_bytes(), b'disposable')

    def test_evidence_inventory_with_a_link_refuses_reclamation(self):
        runtime = self.runtime(); db, _, _ = self.fixture(runtime)
        out = self.root / 'failed'; out.mkdir()
        (out / 'external').symlink_to(self.root, target_is_directory=True)
        with patch.object(storage.maintenance, 'inventory', return_value=('', [])):
            with self.assertRaises(storage.maintenance.UnsafeMaintenance):
                storage.reclaim_after_shutdown(runtime, out)
        self.assertTrue(db.exists())

    def test_shutdown_waits_for_open_files_instead_of_racing_async_node_exit(self):
        runtime = self.runtime(); db, _, _ = self.fixture(runtime)
        out = self.root / 'failed'; out.mkdir()
        with patch.object(storage.maintenance, 'inventory', side_effect=[('', [str(db)]), ('', []), ('', []), ('', [])]), \
             patch.object(storage.time, 'sleep') as pause:
            storage.reclaim_after_shutdown(runtime, out)
        pause.assert_called_once()
        self.assertFalse(db.exists())

    def test_shutdown_timeout_preserves_data_and_refuses_cleanup(self):
        runtime = self.runtime(); db, _, _ = self.fixture(runtime)
        with patch.object(storage.maintenance, 'inventory', return_value=('', [str(db)])):
            with self.assertRaisesRegex(storage.maintenance.UnsafeMaintenance, 'active'):
                storage.reclaim_after_shutdown(runtime, self.root, timeout=0)
        self.assertEqual(db.read_bytes(), b'disposable')

    def test_shutdown_rechecks_activity_that_appears_during_reclamation(self):
        runtime = self.runtime(); db, config, _ = self.fixture(runtime)
        out = self.root / 'failed'; out.mkdir()
        states = [('', []), ('', []), ('', [str(db)]),
                  ('', [str(db)]), ('', []), ('', []), ('', [])]
        with patch.object(storage.maintenance, 'inventory', side_effect=states), \
             patch.object(storage.time, 'sleep') as pause:
            storage.reclaim_after_shutdown(runtime, out)
        self.assertEqual(pause.call_count, 2)
        self.assertFalse(db.exists())
        self.assertEqual(config.read_text(), 'config')

    def test_shutdown_mid_reclamation_activity_timeout_preserves_database(self):
        runtime = self.runtime(); db, _, _ = self.fixture(runtime)
        out = self.root / 'failed'; out.mkdir()
        with patch.object(storage.maintenance, 'inventory', side_effect=[('', []), ('', []), ('', [str(db)])]):
            with self.assertRaises(storage.maintenance.UnsafeMaintenance):
                storage.reclaim_after_shutdown(runtime, out, timeout=0)
        self.assertEqual(db.read_bytes(), b'disposable')

    def test_shutdown_does_not_retry_unavailable_inventory(self):
        runtime = self.runtime(); db, _, _ = self.fixture(runtime)
        with patch.object(storage.maintenance, 'inventory', side_effect=storage.maintenance.UnsafeMaintenance('cannot inventory open files')), \
             patch.object(storage.time, 'sleep') as pause:
            with self.assertRaisesRegex(storage.maintenance.UnsafeMaintenance, 'cannot inventory'):
                storage.reclaim_after_shutdown(runtime, self.root)
        pause.assert_not_called()
        self.assertEqual(db.read_bytes(), b'disposable')

    @contextlib.contextmanager
    def workspace(self):
        before = Path.cwd()
        os.chdir(self.root)
        try:
            yield
        finally:
            os.chdir(before)

    def test_failed_capture_reclaims_partial_invocation_without_awarding_acceptance(self):
        runtime = self.runtime(); db, _, _ = self.fixture(runtime)
        with self.workspace():
            out = Path('proofs/WEB-08'); out.mkdir(parents=True)
            evidence = out / 'evidence.json'
            evidence.write_text(json.dumps({'invocationId': runtime.name, 'outcome': 'incomplete'}))
            original = evidence.read_bytes()
            with patch.object(capture.sys, 'argv', ['capture.py', 'WEB-08', 'proofs']), \
                 patch.object(capture.subprocess, 'run', return_value=type('Result', (), {'returncode': 1})()), \
                 patch.object(storage.maintenance, 'inventory', return_value=('', [])):
                with self.assertRaises(SystemExit) as error:
                    capture.main()
            self.assertEqual(error.exception.code, 1)
            self.assertEqual(evidence.read_bytes(), original)
            self.assertFalse(db.exists())

    def test_failed_independent_reproduction_reclaims_dependencies_and_keeps_fresh_evidence(self):
        with self.workspace():
            out = Path('proofs/WEB-08'); out.mkdir(parents=True)
            protected = out / 'evidence.json'; protected.write_text('{"sourceDigest":"frozen"}')
            before = protected.read_bytes()
            staged = []
            def stage(workspace, runtime):
                source = runtime / 'source'; source.mkdir()
                dependency = source / 'web/node_modules/tool/index.js'
                dependency.parent.mkdir(parents=True); dependency.write_text('disposable dependency')
                fresh = source / out / 'result.json'; fresh.parent.mkdir(parents=True)
                fresh.write_text('{"outcome":"incomplete","reason":"native failure"}')
                staged.append((dependency, fresh))
                return source
            with patch.object(recheck, 'source_digest', return_value='frozen'), \
                 patch.object(recheck, 'base_commit', return_value='commit'), \
                 patch.object(recheck, 'stage_git_metadata', return_value={'mode': 'unit-fixture'}), \
                 patch.object(storage, 'preflight'), \
                 patch.object(recheck, 'verify_published'), patch.object(recheck, 'stage_sources', side_effect=stage), \
                 patch.object(recheck.subprocess, 'run', return_value=type('Result', (), {'returncode': 1})()), \
                 patch.object(storage.maintenance, 'inventory', return_value=('', [])):
                with self.assertRaisesRegex(ValueError, 'Fresh independent live reproduction failed'):
                    recheck.check('WEB-08', 'proofs')
            self.assertEqual(protected.read_bytes(), before)
            self.assertFalse(staged[0][0].exists())
            self.assertIn('native failure', staged[0][1].read_text())

    def test_ci_runs_runtime_and_maintenance_preservation_contracts(self):
        makefile = Path(__file__).resolve().parents[2] / 'Makefile'
        fixture = makefile.read_text().split('fixture-test:', 1)[1].split('\ncase-docs:', 1)[0]
        for filename in ('test_storage.py', 'test_runtime_storage.py'):
            self.assertIn('python3 -B tests/webui/' + filename, fixture)

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
