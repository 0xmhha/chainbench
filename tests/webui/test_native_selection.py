"""Native fixture builders must consume the configured, verified selection."""
import glob as glob_module
import hashlib
import json
import os
from pathlib import Path
import platform
import tempfile
import unittest
from unittest import mock
from fixtures import prepare_web04

CHAINS = ['stablenet', 'wbft', 'wemix']
MARKERS = {'stablenet': 'Anzeon', 'wbft': 'Croissant', 'wemix': 'Wemix'}


class NativeSelectionTests(unittest.TestCase):
    def exercise(self, change=None, source_markers=None, broken_version=False, missing_commit=False, vcs_revision=None, modified=False):
        with tempfile.TemporaryDirectory() as raw:
            root = Path(raw).resolve()
            runtime = root / 'runtime'
            output = root / 'output'
            runtime.mkdir()
            output.mkdir()
            entries = []
            legacy = {}
            for index, chain in enumerate(CHAINS, 1):
                for category in ['selected', 'legacy']:
                    repo = root / category / ('go-' + chain)
                    binary = repo / 'build/bin' / ('gstable' if chain == 'stablenet' else 'gwemix')
                    binary.parent.mkdir(parents=True)
                    binary.write_text(category + '-' + chain)
                    if category == 'legacy':
                        legacy[chain] = str(binary)
                    else:
                        entries.append({'chain': chain, 'repository': str(repo), 'binary': str(binary),
                                        'embeddedCommit': str(index) * 40, 'sha256': hashlib.sha256(binary.read_bytes()).hexdigest(),
                                        'dialect': chain})
            manifest = {'binaries': entries}
            if change:
                change(manifest)
            selected = root / 'selection.json'
            selected.write_text(json.dumps(manifest))
            def glob(pattern):
                for chain in CHAINS:
                    if pattern == str(root / 'selected' / ('go-' + chain) / 'build/bin' / ('gstable' if chain == 'stablenet' else 'gwemix')):
                        return [pattern]
                    if '/go-' + chain + '/build/bin/' in pattern:
                        return [legacy[chain]]
                return []
            def observe(args, **kwargs):
                chain = next(c for c in CHAINS if c in ' '.join(map(str, args)))
                if args[0] == 'go':
                    return '\tbuild\tvcs=git\n\tbuild\tvcs.revision=' + (vcs_revision or str(CHAINS.index(chain) + 1) * 40) + '\n\tbuild\tvcs.modified=' + ('true' if modified else 'false') + '\n'
                if args[0] == 'git':
                    return '// ' + (source_markers or MARKERS)[chain]
                if args[-1] == 'version':
                    if broken_version:
                        return 'unrecognized native version'
                    return ('' if missing_commit else 'Git Commit: ' + str(CHAINS.index(chain) + 1) * 40 + '\n') + 'Operating System: ' + platform.system().lower() + '\nArchitecture: ' + {'x86_64': 'amd64', 'aarch64': 'arm64'}.get(platform.machine(), platform.machine()) + '\n'
                return '--fixture value  fixture flag\n'
            with mock.patch.dict(os.environ, {'WEBUI_SELECTED_BINARIES': str(selected)}), mock.patch.object(glob_module, 'glob', side_effect=glob), mock.patch.object(prepare_web04.subprocess, 'check_output', side_effect=observe), mock.patch.object(prepare_web04.subprocess, 'check_call'):
                result = prepare_web04.prepare(runtime, output)
            return result, {c: (runtime / (c + '-node')).read_text() for c in CHAINS}, json.loads((runtime / 'assets.json').read_text())

    def test_explicit_manifest_overrides_legacy_binary_locations(self):
        result, contents, assets = self.exercise()
        self.assertEqual(contents, {c: 'selected-' + c for c in CHAINS})
        self.assertEqual([x['chain'] for x in result], CHAINS)
        self.assertEqual(len(assets), 5)
        self.assertEqual(assets[-2]['id'], 'wrong-wbft')
        self.assertEqual(assets[-1]['id'], 'stale-wbft')

    def test_stale_checksum_refuses_instead_of_falling_back(self):
        with self.assertRaises(RuntimeError):
            self.exercise(lambda m: m['binaries'][0].update(sha256='0' * 64))

    def test_missing_family_refuses_instead_of_using_legacy_binary(self):
        with self.assertRaises(RuntimeError):
            self.exercise(lambda m: m['binaries'].pop())

    def test_duplicate_family_refuses(self):
        with self.assertRaises(RuntimeError):
            self.exercise(lambda m: m['binaries'].append(dict(m['binaries'][0])))

    def test_wrong_embedded_source_commit_refuses(self):
        with self.assertRaises(RuntimeError):
            self.exercise(lambda m: m['binaries'][0].update(embeddedCommit='0' * 40))


    def test_missing_legacy_stamp_uses_verified_go_vcs(self):
        result, contents, _ = self.exercise(missing_commit=True)
        self.assertEqual(contents, {c: 'selected-' + c for c in CHAINS})
        self.assertEqual(len(result), 3)

    def test_missing_stamp_cannot_hide_wrong_go_revision(self):
        with self.assertRaises(RuntimeError):
            self.exercise(missing_commit=True, vcs_revision='0' * 40)

    def test_missing_stamp_cannot_use_modified_source_metadata(self):
        with self.assertRaises(RuntimeError):
            self.exercise(missing_commit=True, modified=True)

    def test_all_profiles_are_verified_before_materialization(self):
        with mock.patch.object(prepare_web04, 'clone_file') as copy:
            with self.assertRaises(RuntimeError):
                self.exercise(lambda m: m['binaries'][-1].update(sha256='0' * 64))
            copy.assert_not_called()

    def test_missing_selected_file_refuses(self):
        with self.assertRaises(RuntimeError):
            self.exercise(lambda m: m['binaries'][0].update(binary='/missing/selected-native'))

    def test_unrecognized_source_family_refuses(self):
        with self.assertRaises(RuntimeError):
            self.exercise(source_markers={**MARKERS, 'wbft': 'Wemix'})

    def test_malformed_version_refuses(self):
        with self.assertRaises(RuntimeError):
            self.exercise(broken_version=True)

    def test_non_commit_metadata_refuses(self):
        with self.assertRaises(RuntimeError):
            self.exercise(lambda m: m['binaries'][0].update(embeddedCommit='not-a-commit'))

    def test_unknown_family_refuses(self):
        with self.assertRaises(RuntimeError):
            self.exercise(lambda m: m['binaries'][0].update(chain='other'))

    def test_missing_or_malformed_manifest_refuses(self):
        from fixtures.native_selection import selected_binaries
        with tempfile.TemporaryDirectory() as raw:
            missing = Path(raw) / 'selection.json'
            with mock.patch.dict(os.environ, {'WEBUI_SELECTED_BINARIES': str(missing)}):
                with self.assertRaises(RuntimeError):
                    selected_binaries()
                missing.write_text('{')
                with self.assertRaises(RuntimeError):
                    selected_binaries()


if __name__ == '__main__':
    unittest.main()
