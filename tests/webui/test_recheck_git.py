"""Independent acceptance needs a private Git view without changing inputs."""
import contextlib
import hashlib
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest import mock
import recheck
from runtime_contract import source_digest


class IndependentGitTests(unittest.TestCase):
    def exercise(self, mutate_index=False, split_index=False):
        before_cwd = Path.cwd()
        real_run = subprocess.run
        with tempfile.TemporaryDirectory() as raw:
            root = Path(raw).resolve()
            workspace = root / 'workspace'
            runtime = root / 'runtime'
            workspace.mkdir()
            runtime.mkdir()
            def git(*args):
                result = real_run(['git', *args], cwd=workspace, text=True, capture_output=True)
                self.assertEqual(result.returncode, 0, result.stderr)
                return result.stdout.strip()
            (workspace / 'tracked.md').write_text('original\n')
            git('init', '-q', '-b', 'codex/independent-fixture')
            git('add', 'tracked.md')
            git('-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.invalid', '-c', 'commit.gpgSign=false', '-c', 'core.hooksPath=/dev/null', 'commit', '-q', '-m', 'test: seed independent source fixture')
            if split_index:
                git('update-index', '--split-index')
            (workspace / 'tracked.md').write_text('working changes\n')
            (workspace / 'user-note.md').write_text('untracked user input\n')
            index = workspace / '.git/index'
            index_before = (index.read_bytes(), index.stat().st_mtime_ns)
            def git_files():
                return {str(p.relative_to(workspace)): (hashlib.sha256(p.read_bytes()).hexdigest(), p.stat().st_mode, p.stat().st_uid, p.stat().st_gid, p.stat().st_mtime_ns) for p in (workspace / '.git').rglob('*') if p.is_file() and not p.is_symlink()}
            git_before = git_files()
            original = source_digest(workspace)
            output = Path('chainbench-out/acceptance')
            published = workspace / output / 'WEB-01'
            published.mkdir(parents=True)
            evidence = {'sourceDigest': original, 'invocationId': 'published'}
            (published / 'evidence.json').write_text(json.dumps(evidence))
            (published / 'result.json').write_text(json.dumps({'invocationId': 'published'}))
            observed = []
            def command(args, **kwargs):
                if args[:2] == [sys.executable, 'tests/webui/capture.py']:
                    staged = Path(kwargs['cwd'])
                    tracked = real_run(['git', 'ls-files'], cwd=staged, text=True, capture_output=True)
                    self.assertEqual(tracked.returncode, 0, 'independent tracked-tree checks cannot execute: ' + tracked.stderr)
                    self.assertEqual(tracked.stdout.splitlines(), ['tracked.md'])
                    self.assertEqual((staged / 'tracked.md').read_text(), 'working changes\n')
                    self.assertEqual((staged / 'user-note.md').read_text(), 'untracked user input\n')
                    self.assertNotEqual((staged / '.git/index').stat().st_ino, index.stat().st_ino)
                    if mutate_index:
                        edited = real_run(['git', 'add', 'user-note.md'], cwd=staged, text=True, capture_output=True)
                        self.assertEqual(edited.returncode, 0, edited.stderr)
                    fresh = staged / output / 'WEB-01'
                    fresh.mkdir(parents=True)
                    (fresh / 'result.json').write_text(json.dumps({'invocationId': 'fresh'}))
                    observed.append(staged)
                    return subprocess.CompletedProcess(args, 0)
                return real_run(args, **kwargs)
            try:
                os.chdir(workspace)
                with mock.patch.object(recheck, 'verify_published'), mock.patch.object(recheck, 'create_runtime', return_value=runtime), mock.patch.object(recheck, 'reclaim_after_shutdown'), mock.patch.object(recheck.subprocess, 'run', side_effect=command), contextlib.redirect_stdout(io.StringIO()):
                    recheck.check('WEB-01', str(output))
            finally:
                os.chdir(before_cwd)
            self.assertEqual(len(observed), 1)
            self.assertEqual(source_digest(workspace), original)
            self.assertEqual(index_before, (index.read_bytes(), index.stat().st_mtime_ns))
            self.assertEqual(git_before, git_files())
            self.assertEqual(json.loads((published / 'evidence.json').read_text()), evidence)
            self.assertEqual((workspace / 'user-note.md').read_text(), 'untracked user input\n')

    def test_independent_capture_can_read_the_original_tracked_set(self):
        self.exercise()

    def test_private_index_changes_cannot_touch_original_git_or_source(self):
        self.exercise(mutate_index=True)


    def test_split_index_is_copied_with_its_private_shared_base(self):
        self.exercise(mutate_index=True, split_index=True)

    def test_existing_git_view_refuses_before_git_commands(self):
        from git_snapshot import stage_git_metadata
        with tempfile.TemporaryDirectory() as raw:
            root = Path(raw)
            original, staged = root / 'original', root / 'staged'
            original.mkdir()
            staged.mkdir()
            (staged / '.git').mkdir()
            with mock.patch('git_snapshot.subprocess.run') as run:
                with self.assertRaisesRegex(ValueError, 'existing Git'):
                    stage_git_metadata(original, staged)
                run.assert_not_called()

    def test_source_mismatch_refuses_before_git_commands(self):
        from git_snapshot import stage_git_metadata
        with tempfile.TemporaryDirectory() as raw:
            root = Path(raw)
            original, staged = root / 'original', root / 'staged'
            original.mkdir()
            staged.mkdir()
            (original / 'source.py').write_text('x=1')
            (staged / 'source.py').write_text('x=2')
            with mock.patch('git_snapshot.subprocess.run') as run:
                with self.assertRaisesRegex(ValueError, 'same frozen source'):
                    stage_git_metadata(original, staged)
                run.assert_not_called()

    def test_inherited_git_locations_cannot_redirect_private_checks(self):
        from git_snapshot import GIT_LOCATIONS, git_environment
        values = {name: 'original-git' for name in GIT_LOCATIONS}
        values['PRESERVE_SETTING'] = 'kept'
        before = dict(values)
        self.assertEqual(git_environment(values), {'PRESERVE_SETTING': 'kept'})
        self.assertEqual(before, values)


if __name__ == '__main__':
    unittest.main()
