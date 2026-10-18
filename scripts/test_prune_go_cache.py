"""Contracts for selective cleanup of this task's rebuildable Go cache."""
import contextlib
import io
import json
import os
from pathlib import Path
import sys
import tempfile
import time
import unittest
from unittest.mock import patch

import prune_go_cache as cache


class CacheTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name).resolve() / 'cache'
        self.root.mkdir()
        (self.root / 'README').write_text(cache.README)
        self.now = time.time() + 40 * 3600

    def entry(self, number=1, suffix='d'):
        digest = f'{number:064x}'
        folder = self.root / digest[:2]
        folder.mkdir(exist_ok=True)
        p = folder / (digest + '-' + suffix)
        p.write_bytes(b'cache' * 1000)
        return p

    def plan(self, references='', **kwargs):
        with patch.object(cache, 'inventory', return_value=('', [])):
            return cache.plan(self.root, references, now=self.now, free_bytes=0,
                              target_bytes=1 << 30, keep_bytes=0, **kwargs)

    def apply(self, plan, references=''):
        with patch.object(cache, 'inventory', return_value=('', [])):
            return cache.apply(self.root, plan, lambda: references, now=self.now)

    def test_old_entry_is_selected_and_removed(self):
        p = self.entry()
        plan = self.plan()
        self.assertEqual([r['path'] for r in plan['entries']], [str(p)])
        self.apply(plan)
        self.assertFalse(p.exists())
        self.assertTrue((self.root / 'README').exists())

    def test_recent_mtime_is_retained(self):
        p = self.entry(); os.utime(p, (self.now, self.now))
        self.assertEqual(self.plan()['entries'], [])

    def test_recent_ctime_is_retained_even_with_old_mtime(self):
        p = self.entry(); os.utime(p, (1, 1))
        with patch.object(cache, 'inventory', return_value=('', [])):
            result = cache.plan(self.root, '', now=time.time(), free_bytes=0,
                                target_bytes=1 << 30, keep_bytes=0)
        self.assertEqual(result['entries'], [])

    def test_future_entry_is_retained(self):
        p = self.entry(); os.utime(p, (self.now + 3600, self.now + 3600))
        self.assertEqual(self.plan()['entries'], [])

    def test_unknown_name_and_prefix_mismatch_are_retained(self):
        p = self.entry(); p.rename(p.parent / ('ab' + p.name[2:]))
        (p.parent / 'notes').write_text('user')
        self.assertEqual(self.plan()['entries'], [])

    def test_symlinks_hardlinks_and_executable_directories_are_retained(self):
        p = self.entry()
        os.link(p, p.parent / 'linked')
        q = self.entry(2); q.unlink(); q.symlink_to(p)
        r = self.entry(3); r.unlink(); r.mkdir()
        self.assertEqual(self.plan()['entries'], [])

    def test_explicit_evidence_reference_is_retained(self):
        p = self.entry()
        self.assertEqual(self.plan('receipt ' + str(p))['entries'], [])
        self.assertEqual(self.plan('digest ' + p.name[:64])['entries'], [])

    def test_generic_cache_configuration_does_not_protect_every_entry(self):
        self.entry()
        self.assertEqual(len(self.plan('GOCACHE=' + str(self.root))['entries']), 1)

    def test_active_cache_descriptor_refuses_plan(self):
        p = self.entry()
        with patch.object(cache, 'inventory', return_value=('', [str(p)])):
            with self.assertRaises(cache.UnsafeMaintenance):
                cache.plan(self.root, '', now=self.now)

    def test_active_go_tool_refuses_plan(self):
        self.entry()
        for cmd in ['/opt/go/bin/go test ./...', '/opt/go/pkg/tool/compile -o target', '/opt/go/pkg/tool/link -o out']:
            with self.subTest(cmd=cmd), patch.object(cache, 'inventory', return_value=(cmd, [])):
                with self.assertRaises(cache.UnsafeMaintenance):
                    cache.plan(self.root, '', now=self.now)

    def test_inventory_failure_refuses_plan_and_apply(self):
        self.entry(); plan = self.plan()
        with patch.object(cache, 'inventory', side_effect=cache.UnsafeMaintenance('inventory failed')):
            with self.assertRaises(cache.UnsafeMaintenance): cache.plan(self.root, '', now=self.now)
            with self.assertRaises(cache.UnsafeMaintenance): cache.apply(self.root, plan, lambda: '', now=self.now)

    def test_changed_entry_or_new_reference_refuses_apply(self):
        for change in ['bytes', 'mtime', 'reference']:
            with self.subTest(change=change):
                p = self.entry(10 + len(change)); plan = self.plan()
                refs = ''
                if change == 'bytes': p.write_bytes(b'new bytes')
                elif change == 'mtime': os.utime(p, (self.now, self.now))
                else: refs = str(p)
                with self.assertRaises(cache.UnsafeMaintenance): self.apply(plan, refs)
                self.assertTrue(p.exists()); p.unlink()

    def test_replaced_root_refuses_apply(self):
        p = self.entry(); plan = self.plan()
        self.root.rename(self.root.with_name('old'))
        self.root.mkdir(); (self.root / 'README').write_text(cache.README)
        with self.assertRaises(cache.UnsafeMaintenance): self.apply(plan)
        self.assertTrue(self.root.with_name('old').joinpath(p.relative_to(self.root)).exists())

    def test_replaced_bucket_refuses_apply(self):
        p = self.entry(); plan = self.plan()
        p.parent.rename(p.parent.with_name('old'))
        p.parent.mkdir()
        with self.assertRaises(cache.UnsafeMaintenance): self.apply(plan)

    def test_root_symlink_or_wrong_marker_refuses_plan(self):
        self.entry()
        link = self.root.with_name('alias'); link.symlink_to(self.root)
        with self.assertRaises(cache.UnsafeMaintenance): cache.plan(link, '', now=self.now)
        (self.root / 'README').write_text('not a Go cache')
        with self.assertRaises(cache.UnsafeMaintenance): self.plan()

    def test_budget_and_minimum_retention_are_enforced(self):
        p = self.entry(); q = self.entry(2)
        block = p.stat().st_blocks * 512
        result = self.plan(max_bytes=block)
        self.assertEqual(len(result['entries']), 1)
        with patch.object(cache, 'inventory', return_value=('', [])):
            result = cache.plan(self.root, '', now=self.now, free_bytes=0,
                                target_bytes=1 << 30, keep_bytes=2 * block)
        self.assertEqual(result['entries'], [])
        self.assertTrue(q.exists())

    def test_target_already_met_does_not_delete(self):
        self.entry()
        with patch.object(cache, 'inventory', return_value=('', [])):
            result = cache.plan(self.root, '', now=self.now, free_bytes=1 << 40)
        self.assertEqual(result['entries'], [])

    def test_age_below_eight_hours_is_rejected(self):
        self.entry()
        with self.assertRaises(cache.UnsafeMaintenance): self.plan(age_hours=1)

    def test_second_inventory_detects_activity_before_delete(self):
        p = self.entry(); plan = self.plan()
        with patch.object(cache, 'inventory', side_effect=[('', []), ('', [str(p)])]):
            with self.assertRaises(cache.UnsafeMaintenance):
                cache.apply(self.root, plan, lambda: '', now=self.now)
        self.assertTrue(p.exists())

    def test_partial_failure_retains_deletion_journal(self):
        first = self.entry(); second = self.entry(2); selection = self.plan()
        progress = []
        unlink = os.unlink
        def fail_second(path, **kwargs):
            if path == second.name:
                raise OSError('injected disk failure')
            return unlink(path, **kwargs)
        with patch.object(cache, 'inventory', return_value=('', [])), patch.object(cache.os, 'unlink', side_effect=fail_second):
            with self.assertRaises(OSError):
                cache.apply(self.root, selection, lambda: '', now=self.now, progress=progress)
        self.assertEqual(progress, [str(first)])
        self.assertFalse(first.exists()); self.assertTrue(second.exists())

    def test_reference_catalog_is_not_rescanned_for_every_cache_entry(self):
        class CountedEvidence(str):
            probes = 0
            def __contains__(self, value):
                self.probes += 1
                return super().__contains__(value)
        files = [self.entry(i) for i in range(1, 41)]
        references = CountedEvidence('x' * 100000 + str(files[0]))
        selection = self.plan(references)
        self.assertEqual(len(selection['entries']), 39)
        self.assertLessEqual(references.probes, 2,
                             'the evidence catalog was scanned once per file')

    def test_cli_dry_run_and_explicit_apply_write_private_reports(self):
        for delete in [False, True]:
            with self.subTest(delete=delete):
                p = self.entry(90 + delete); selection = self.plan()
                report = self.root.parent / ('report-' + str(delete) + '.json')
                argv = ['prune_go_cache.py', '--report', str(report)] + (['--delete'] if delete else [])
                with patch.object(cache, 'ROOT', self.root), patch.object(cache, 'plan', return_value=selection), patch.object(cache, 'inventory', return_value=('', [])), patch.object(cache, 'evidence_text', return_value=''), patch.object(cache.time, 'time', return_value=self.now), patch.object(sys, 'argv', argv), contextlib.redirect_stdout(io.StringIO()):
                    cache.main()
                result = json.loads(report.read_text())
                self.assertEqual(result['applied'], delete)
                self.assertEqual(p.exists(), not delete)
                self.assertEqual(report.stat().st_mode & 0o777, 0o600)
                if p.exists(): p.unlink()

    def test_cli_failure_retains_error_report_and_refuses_overwrite(self):
        report = self.root.parent / 'failed.json'
        with patch.object(cache, 'ROOT', self.root), patch.object(cache, 'inventory', side_effect=cache.UnsafeMaintenance('probe failed')), patch.object(cache, 'evidence_text', return_value=''), patch.object(sys, 'argv', ['prune_go_cache.py', '--delete', '--report', str(report)]):
            with self.assertRaises(cache.UnsafeMaintenance): cache.main()
            old = report.read_bytes()
            self.assertEqual(json.loads(old)['error'], 'probe failed')
            with self.assertRaises(FileExistsError): cache.main()
            self.assertEqual(report.read_bytes(), old)

    def test_ci_runs_cache_contracts(self):
        makefile = Path(__file__).resolve().parents[1] / 'Makefile'
        self.assertIn('scripts/test_prune_go_cache.py', makefile.read_text())
        self.assertIn('scripts/test_proof_native_cow.py', makefile.read_text())
        self.assertIn('tests/webui/test_after_proof_storage.py', makefile.read_text())


class CompletedSourceCacheTests(unittest.TestCase):
    entry = CacheTests.entry
    apply = CacheTests.apply

    def setUp(self):
        CacheTests.setUp(self)
        self.now = time.time() + 1
        self.runtimes = self.root.parent / 'runtimes'
        self.runtimes.mkdir()
        self.runtime = self.runtimes / 'recheck-owned'
        self.runtime.mkdir()
        self.source = self.runtime / 'source'
        self.source.mkdir()
        self.marker = self.runtime / '.webui-owned-runtime.json'
        self.marker.write_text(json.dumps({'version': 1, 'path': str(self.runtime),
                                          'policy': 'completed-disposable-data'}))
        self.audit = self.runtime / 'storage-audit.jsonl'
        self.audit.write_text(json.dumps({'event': 'reclaimed',
            'path': str(self.source / 'web/node_modules'), 'allocatedBytes': 1,
            'finishedAt': '2026-10-18T00:00:00+00:00'}) + '\n')

    def associated(self, number=1):
        path = self.entry(number)
        path.write_bytes(b'go archive\x00' + str(self.source / 'internal/app/run.go').encode())
        return path

    def selection(self, references='', state=('', [])):
        with patch.object(cache, 'inventory', return_value=state):
            return cache.plan(self.root, references, now=self.now, free_bytes=0,
                target_bytes=1 << 30, keep_bytes=0, completed_runtime_root=self.runtimes)

    def test_recent_closed_source_entry_is_opt_in_and_reclaimed(self):
        path = self.associated()
        with patch.object(cache, 'inventory', return_value=('', [])):
            cold = cache.plan(self.root, '', now=self.now, free_bytes=0, keep_bytes=0)
        self.assertEqual(cold['entries'], [])
        selected = self.selection()
        self.assertEqual([r['path'] for r in selected['entries']], [str(path)])
        self.assertIn(str(self.source), selected['entries'][0]['completedSources'])
        self.apply(selected)
        self.assertFalse(path.exists())
        self.assertTrue(self.marker.exists())
        self.assertTrue(self.audit.exists())
        self.assertTrue(self.source.is_dir())

    def test_recent_unrelated_entry_is_retained(self):
        self.entry()
        self.assertEqual(self.selection()['entries'], [])

    def test_active_source_is_retained(self):
        self.associated()
        self.assertEqual(self.selection(state=(str(self.runtime), []))['entries'], [])
        self.assertEqual(self.selection(state=('', [str(self.source / 'run.go')]))['entries'], [])

    def test_ownership_without_cleanup_record_is_retained(self):
        self.associated(); self.audit.unlink()
        self.assertEqual(self.selection()['entries'], [])

    def test_cleanup_start_without_completion_is_retained(self):
        self.associated(); self.audit.write_text(json.dumps({'event': 'reclaim-start',
            'path': str(self.source / 'web/node_modules')}) + '\n')
        self.assertEqual(self.selection()['entries'], [])

    def test_cleanup_of_another_tree_does_not_qualify_source(self):
        self.associated(); self.audit.write_text(json.dumps({'event': 'reclaimed',
            'path': str(self.runtime / 'unrelated'), 'finishedAt': 'now'}) + '\n')
        self.assertEqual(self.selection()['entries'], [])

    def test_unmarked_or_absent_source_is_retained(self):
        self.associated(); self.marker.unlink()
        self.assertEqual(self.selection()['entries'], [])
        self.marker.write_text(json.dumps({'version': 1, 'path': str(self.runtime),
            'policy': 'completed-disposable-data'})); self.source.rmdir()
        self.assertEqual(self.selection()['entries'], [])

    def test_unknown_association_alongside_closed_source_is_retained(self):
        path = self.associated()
        with path.open('ab') as output:
            output.write(str(self.runtimes / 'unowned/source/run.go').encode())
        self.assertEqual(self.selection()['entries'], [])

    def test_reference_still_protects_recent_closed_source_entry(self):
        path = self.associated()
        self.assertEqual(self.selection(str(path))['entries'], [])

    def test_future_timestamp_is_never_eligible(self):
        path = self.associated(); os.utime(path, (self.now + 3600, self.now + 3600))
        self.assertEqual(self.selection()['entries'], [])

    def test_malformed_cleanup_record_refuses_plan(self):
        self.associated(); self.audit.write_text('{')
        with self.assertRaises(cache.UnsafeMaintenance): self.selection()

    def test_wrong_ownership_refuses_plan(self):
        self.associated(); self.marker.write_text(json.dumps({'path': '/elsewhere'}))
        with self.assertRaises(cache.UnsafeMaintenance): self.selection()

    def test_linked_or_symlink_cleanup_record_refuses_plan(self):
        self.associated(); os.link(self.audit, self.runtime / 'linked-audit')
        with self.assertRaises(cache.UnsafeMaintenance): self.selection()
        (self.runtime / 'linked-audit').unlink()
        self.audit.rename(self.runtime / 'original-audit')
        self.audit.symlink_to(self.runtime / 'original-audit')
        with self.assertRaises(cache.UnsafeMaintenance): self.selection()

    def test_cleanup_record_changed_after_plan_refuses_all_deletion(self):
        path = self.associated(); selected = self.selection()
        with self.audit.open('a') as output: output.write('{}\n')
        with self.assertRaises(cache.UnsafeMaintenance): self.apply(selected)
        self.assertTrue(path.exists())

    def test_source_became_active_after_plan_refuses_apply(self):
        path = self.associated(); selected = self.selection()
        with patch.object(cache, 'inventory', return_value=(str(self.runtime), [])):
            with self.assertRaises(cache.UnsafeMaintenance):
                cache.apply(self.root, selected, lambda: '', now=self.now)
        self.assertTrue(path.exists())

    def test_source_replaced_after_plan_refuses_apply(self):
        path = self.associated(); selected = self.selection()
        self.source.rename(self.runtime / 'old-source'); self.source.mkdir()
        with self.assertRaises(cache.UnsafeMaintenance): self.apply(selected)
        self.assertTrue(path.exists())

    def test_new_reference_before_apply_preserves_recent_entry(self):
        path = self.associated(); selected = self.selection()
        with self.assertRaises(cache.UnsafeMaintenance): self.apply(selected, str(path))
        self.assertTrue(path.exists())

    def test_multiple_closed_sources_are_verified(self):
        path = self.associated(); other = self.runtimes / 'other/source'
        other.mkdir(parents=True)
        (other.parent / '.webui-owned-runtime.json').write_text(json.dumps({
            'version': 1, 'path': str(other.parent), 'policy': 'completed-disposable-data'}))
        (other.parent / 'storage-audit.jsonl').write_text(json.dumps({
            'event': 'reclaimed', 'path': str(other / 'web/node_modules'),
            'allocatedBytes': 1, 'finishedAt': '2026-10-18T00:00:00+00:00'}) + '\n')
        with path.open('ab') as output: output.write(str(other / 'run.go').encode())
        selected = self.selection()
        self.assertEqual(set(selected['entries'][0]['completedSources']),
                         {str(self.source), str(other)})
        self.apply(selected)
        self.assertFalse(path.exists())
        self.assertTrue(other.is_dir())


    def test_symlink_source_refuses_plan(self):
        self.associated(); self.source.rename(self.runtime / 'original-source')
        self.source.symlink_to(self.runtime / 'original-source')
        with self.assertRaises(cache.UnsafeMaintenance): self.selection()

    def test_invalid_completion_timestamp_or_budget_refuses(self):
        self.associated()
        for change in [{'finishedAt': 'not-a-time'}, {'finishedAt': '2099-01-01T00:00:00+00:00'},
                       {'finishedAt': '2026-10-18T00:00:00'}, {'allocatedBytes': -1},
                       {'allocatedBytes': True}]:
            row = {'event': 'reclaimed', 'path': str(self.source / 'web/node_modules'),
                   'allocatedBytes': 1, 'finishedAt': '2026-10-18T00:00:00+00:00'}
            row.update(change); self.audit.write_text(json.dumps(row) + '\n')
            with self.subTest(change=change), self.assertRaises(cache.UnsafeMaintenance):
                self.selection()

    def test_completed_mode_does_not_weaken_minimum_cold_age(self):
        self.associated()
        with patch.object(cache, 'inventory', return_value=('', [])):
            with self.assertRaises(cache.UnsafeMaintenance):
                cache.plan(self.root, '', age_hours=1, completed_runtime_root=self.runtimes)

    def test_source_activity_in_second_observation_refuses_apply(self):
        path = self.associated(); selected = self.selection()
        observations = [('', []), ('', []), (str(self.runtime), [])]
        with patch.object(cache, 'inventory', side_effect=observations):
            with self.assertRaises(cache.UnsafeMaintenance):
                cache.apply(self.root, selected, lambda: '', now=self.now)
        self.assertTrue(path.exists())

    def test_partial_source_prefix_does_not_authorize_recent_entry(self):
        path = self.entry()
        path.write_bytes(str(self.runtime / 'source-backup/run.go').encode())
        self.assertEqual(self.selection()['entries'], [])

    def test_cli_explicit_completed_mode_uses_same_preservation_policy(self):
        original_plan = cache.plan
        def bounded(root, refs, **kwargs):
            return original_plan(root, refs, now=self.now, free_bytes=0,
                                 target_bytes=1 << 30, keep_bytes=0, **kwargs)
        for delete in [False, True]:
            path = self.associated(100 + delete)
            report = self.root.parent / ('completed-report-' + str(delete) + '.json')
            argv = ['prune_go_cache.py', '--completed-runtime-cache', '--report', str(report)]
            if delete: argv.append('--delete')
            with patch.object(cache, 'ROOT', self.root), \
                 patch.object(cache, 'COMPLETED_ROOT', self.runtimes), \
                 patch.object(cache, 'plan', side_effect=bounded), \
                 patch.object(cache, 'inventory', return_value=('', [])), \
                 patch.object(cache, 'evidence_text', return_value=''), \
                 patch.object(cache.time, 'time', return_value=self.now), \
                 patch.object(sys, 'argv', argv), contextlib.redirect_stdout(io.StringIO()):
                cache.main()
            result = json.loads(report.read_text())
            self.assertEqual(result['applied'], delete)
            self.assertEqual(result['plan']['completedRuntimeRoot'], str(self.runtimes))
            self.assertEqual(path.exists(), not delete)
            self.assertEqual(report.stat().st_mode & 0o777, 0o600)
            if path.exists(): path.unlink()


if __name__ == '__main__':
    unittest.main()
