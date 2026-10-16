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


if __name__ == '__main__':
    unittest.main()
