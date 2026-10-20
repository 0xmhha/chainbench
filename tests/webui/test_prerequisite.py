"""Prerequisite checks cannot mutate frozen source or open live gates."""
import importlib
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest import mock
import run_web01


class ExistingRunnerChecks(unittest.TestCase):
    def capture(self):
        original = Path.cwd()
        with tempfile.TemporaryDirectory() as raw:
            root = Path(raw).resolve()
            spa = root / 'internal/dashboard/spa/index.html'
            spa.parent.mkdir(parents=True)
            spa.write_text('bundle')
            os.utime(spa, ns=(1000000000, 1000000000))
            initial = spa.stat().st_mtime_ns
            (root / 'web').mkdir()
            (root / 'web/package.json').write_text('{}')
            runtime = root / 'runtime'
            runtime.mkdir()
            calls = []
            def command(args, **kwargs):
                calls.append((args, kwargs))
                if args == 'npm --prefix web run build':
                    target = Path(kwargs.get('cwd') or Path.cwd()) / 'internal/dashboard/spa/index.html'
                    target.parent.mkdir(parents=True, exist_ok=True)
                    target.write_text('bundle')
                failed = isinstance(args, list) and args[0] == 'bash'
                return subprocess.CompletedProcess(args, 1 if failed else 0, '', 'fixture deliberately stops before browser')
            try:
                os.chdir(root)
                with mock.patch.dict(os.environ, {'WEBUI_RUNTIME_ROOT': str(root / 'scratch'), 'GSTABLE_BIN': 'live', 'GWBFT_BIN': 'live', 'GWEMIX_BIN': 'live', 'CHAINBENCH_DOCKER_SERVERS': 'live', 'CHAINBENCH_W4_PROOF': '1', 'PRESERVE_SETTING': 'kept'}):
                    with mock.patch.object(sys, 'argv', ['run_web01', 'chainbench-out/check']), mock.patch.object(run_web01, 'create_runtime', return_value=runtime), mock.patch.object(run_web01.subprocess, 'run', side_effect=command):
                        run_web01.main()
                return initial, spa.stat().st_mtime_ns, calls, root
            finally:
                os.chdir(original)

    def test_web01_build_does_not_touch_embedded_checkout(self):
        before, after, calls, root = self.capture()
        self.assertEqual(before, after, 'build touched frozen SPA even when bytes matched')
        npm = next(kw for cmd, kw in calls if cmd == 'npm --prefix web run build')
        self.assertNotEqual(Path(npm.get('cwd') or root), root)

    def test_web01_unit_checks_do_not_inherit_native_gates(self):
        _, _, calls, root = self.capture()
        checks = [(cmd, kw) for cmd, kw in calls if isinstance(cmd, str)]
        self.assertEqual(len(checks), 2)
        for _, kw in checks:
            self.assertIn('env', kw)
            self.assertEqual(kw['env']['PRESERVE_SETTING'], 'kept')
            for name in ['GSTABLE_BIN', 'GWBFT_BIN', 'GWEMIX_BIN', 'CHAINBENCH_DOCKER_SERVERS', 'CHAINBENCH_W4_PROOF']:
                self.assertNotIn(name, kw['env'])
        go = next(kw for cmd, kw in checks if cmd.startswith('go test'))
        self.assertEqual(Path(go['cwd']), root)


class SharedPrerequisiteTests(unittest.TestCase):
    def fixture(self, root):
        (root / 'web').mkdir()
        (root / 'web/package.json').write_text('{}')
        spa = root / 'internal/dashboard/spa/index.html'
        spa.parent.mkdir(parents=True)
        spa.write_text('reviewed bundle')
        return spa

    def test_mismatched_build_fails_without_repairing_checkout(self):
        import prerequisite
        with tempfile.TemporaryDirectory() as raw:
            root = Path(raw).resolve()
            spa = self.fixture(root)
            before = (spa.read_bytes(), spa.stat().st_mtime_ns)
            def command(args, **kwargs):
                target = Path(kwargs['cwd']) / 'internal/dashboard/spa/index.html'
                target.parent.mkdir(parents=True)
                target.write_text('different bundle')
                return subprocess.CompletedProcess(args, 0, 'build output', '')
            scratch = root / 'scratch'
            with mock.patch.dict(os.environ, {'WEBUI_RUNTIME_ROOT': str(scratch)}), mock.patch.object(prerequisite.subprocess, 'run', side_effect=command):
                result = prerequisite.run_check(prerequisite.SPA_BUILD, cwd=root, shell=True, text=True, capture_output=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('differs from frozen', result.stderr)
            self.assertEqual(before, (spa.read_bytes(), spa.stat().st_mtime_ns))
            self.assertEqual(list(scratch.iterdir()), [])

    def test_unit_command_keeps_checkout_and_caller_environment(self):
        import prerequisite
        with tempfile.TemporaryDirectory() as raw:
            root = Path(raw).resolve()
            environment = {name: 'native' for name in prerequisite.LIVE_GATES}
            environment['PRESERVE_SETTING'] = 'kept'
            initial = dict(environment)
            with mock.patch.object(prerequisite.subprocess, 'run', return_value=subprocess.CompletedProcess('go', 0)) as run:
                prerequisite.run_check('go test ./internal/resource', cwd=root, env=environment)
            self.assertEqual(run.call_args.kwargs['cwd'], root)
            self.assertEqual(run.call_args.kwargs['env'], {'PRESERVE_SETTING': 'kept'})
            self.assertEqual(environment, initial)

    def test_failed_build_status_and_logs_survive_and_staging_is_removed(self):
        import prerequisite
        with tempfile.TemporaryDirectory() as raw:
            root = Path(raw).resolve()
            self.fixture(root)
            scratch = root / 'scratch'
            failed = subprocess.CompletedProcess(prerequisite.SPA_BUILD, 7, 'compiler output', 'compiler failure')
            with mock.patch.dict(os.environ, {'WEBUI_RUNTIME_ROOT': str(scratch)}), mock.patch.object(prerequisite.subprocess, 'run', return_value=failed):
                result = prerequisite.run_check(prerequisite.SPA_BUILD, cwd=root, text=True, capture_output=True)
            self.assertEqual((result.returncode, result.stdout, result.stderr), (7, 'compiler output', 'compiler failure'))
            self.assertEqual(list(scratch.iterdir()), [])

    def test_unreviewed_missing_embedded_assets_cannot_pass(self):
        import prerequisite
        with tempfile.TemporaryDirectory() as raw:
            root = Path(raw).resolve()
            (root / 'web').mkdir()
            with mock.patch.dict(os.environ, {'WEBUI_RUNTIME_ROOT': str(root / 'scratch')}), mock.patch.object(prerequisite.subprocess, 'run', return_value=subprocess.CompletedProcess('npm', 0, '', '')):
                result = prerequisite.run_check(prerequisite.SPA_BUILD, cwd=root, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
            self.assertEqual(result.returncode, 1)
            self.assertIn('differs from frozen', result.stdout)

    def test_symlinked_embedded_assets_refuse_before_build(self):
        import prerequisite
        with tempfile.TemporaryDirectory() as raw:
            root = Path(raw).resolve()
            spa = self.fixture(root)
            outside = root / 'outside'
            outside.write_text('unrelated')
            spa.unlink()
            spa.symlink_to(outside)
            with mock.patch.object(prerequisite.subprocess, 'run') as run:
                with self.assertRaisesRegex(ValueError, 'symlinks'):
                    prerequisite.run_check(prerequisite.SPA_BUILD, cwd=root)
                run.assert_not_called()
            self.assertEqual(outside.read_text(), 'unrelated')


if __name__ == '__main__':
    unittest.main()
