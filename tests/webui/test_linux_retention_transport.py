import hashlib, io, json, re, tarfile, unittest
from pathlib import Path
import linux_retention_transport as batch

def package(files):
    out = io.BytesIO()
    with tarfile.open(fileobj=out, mode='w') as tar:
        for (name, data) in files:
            i = tarfile.TarInfo(name)
            i.size = len(data)
            tar.addfile(i, io.BytesIO(data))
    return out.getvalue()

class BatchContract(unittest.TestCase):

    def setUp(self):
        self.root = '/data/chainbench/l1i-test/c1'
        self.node = self.root + '/node/abc/node1'
        self.marker = {'runtime': '/private/tmp/cbui/l1i-test', 'sourceDigest': 'a' * 64, 'workspace': '/private/tmp/cbui/l1i-test/work/c1', 'remoteDataRoot': self.root, 'fleetRuntime': '/private/tmp/cbui/lf-owned'}
        marker = json.dumps(self.marker).encode()
        mp = 'l1i-test/c1/.chainbench-owned-runtime.json'
        config = 'l1i-test/c1/runtime/abc/configs/node1.toml'
        self.roots = [self.root + '/runtime/abc', self.root + '/.chainbench-owned-runtime.json']
        self.binaries = {'/data/chainbench/bin/gstable': '1' * 64}
        files = [(config, b'configuration'), (mp, marker)]
        metadata = b''
        hashes = b''
        archive = io.BytesIO()
        with tarfile.open(fileobj=archive, mode='w:gz', format=tarfile.PAX_FORMAT) as tar:
            for (path, data) in files:
                metadata += b'\x00'.join((v.encode() for v in [path, 'f', '600', '2001', '2001', '1700000000.123456789', str(len(data)), ''])) + b'\x00'
                hashes += hashlib.sha256(data).hexdigest().encode() + b'  ' + path.encode() + b'\x00'
                info = tarfile.TarInfo(path)
                info.size = len(data)
                info.mode = 384
                info.uid = info.gid = 2001
                info.mtime = 1700000000
                info.pax_headers = {'mtime': '1700000000.123456789'}
                tar.addfile(info, io.BytesIO(data))
        self.parts = {'before.meta': metadata, 'before.hash': hashes, 'after.meta': metadata, 'after.hash': hashes, 'archive.tar.gz': archive.getvalue(), 'roots.txt': ('\n'.join((v.removeprefix('/data/chainbench/') for v in self.roots)) + '\n').encode(), 'missing.txt': b'', 'candidates.raw': (self.node + '/gstable/chaindata\x00').encode(), 'binary.hash': ('1' * 64 + '  /data/chainbench/bin/gstable\n').encode(), 'marker.json': marker, 'node-roots.txt': (self.node + '\n').encode()}

    def parse(self, parts=None, extra=()):
        return batch.parse_bundle(package(list((parts or self.parts).items()) + list(extra)), self.roots, [self.node], self.binaries, self.marker)

    def bad(self, name, value):
        parts = dict(self.parts)
        parts[name] = value
        with self.assertRaises((ValueError, KeyError, tarfile.TarError)):
            self.parse(parts)

    def test_complete_before_archive_after_comparison(self):
        self.assertEqual(self.parse()[-1], [self.node + '/gstable/chaindata'])

    def test_missing_package_member(self):
        p = dict(self.parts)
        p.pop('after.meta')
        with self.assertRaises(ValueError):
            self.parse(p)

    def test_duplicate_member(self):
        with self.assertRaises(ValueError):
            self.parse(extra=[('marker.json', self.parts['marker.json'])])

    def test_unknown_member(self):
        with self.assertRaises(ValueError):
            self.parse(extra=[('../unowned', b'data')])

    def test_binary_hash_change(self):
        self.bad('binary.hash', b'2' * 64 + b'  /data/chainbench/bin/gstable\n')

    def test_wrong_marker(self):
        self.bad('marker.json', json.dumps({**self.marker, 'remoteDataRoot': self.root + '0'}).encode())

    def test_after_hash_change(self):
        self.bad('after.hash', self.parts['after.hash'].replace(self.parts['after.hash'][:64], b'2' * 64))

    def test_archive_changed_despite_equal_snapshots(self):
        stream = io.BytesIO()
        with tarfile.open(fileobj=stream, mode='w:gz') as tar:
            info = tarfile.TarInfo('l1i-test/c1/runtime/abc/configs/node1.toml')
            info.size = 3
            tar.addfile(info, io.BytesIO(b'bad'))
        self.bad('archive.tar.gz', stream.getvalue())

    def test_selected_scope_change(self):
        self.bad('roots.txt', b'keys\n')

    def test_candidate_from_other_case(self):
        self.bad('candidates.raw', b'/data/chainbench/l1i-test/c10/node/abc/node1/gstable/chaindata\x00')

    def test_candidate_link_alias(self):
        self.bad('candidates.raw', (self.node + '/../node2/gstable/chaindata\x00').encode())

    def test_non_database_candidate(self):
        self.bad('candidates.raw', (self.node + '/keystore\x00').encode())

    def test_incomplete_candidate_list(self):
        self.bad('candidates.raw', self.parts['candidates.raw'][:-1])

    def test_unknown_node_root(self):
        self.bad('node-roots.txt', b'/data/chainbench/node/unowned\n')

    def test_exported_snapshot_is_never_a_disposable_database(self):
        self.bad('candidates.raw', (self.node + '/snapshots/chaindata\x00').encode())

    def test_unknown_namespace_is_never_a_disposable_database(self):
        self.bad('candidates.raw', (self.node + '/evidence/etcd\x00').encode())

    def test_retention_contracts_run_in_check_and_ci(self):
        repo = Path(__file__).resolve().parents[2]
        workflow = (repo / '.github/workflows/ci.yml').read_text()
        makefile = (repo / 'Makefile').read_text()
        self.assertRegex(workflow, '(?m)^\\s*run:\\s*make\\s+sweep-test\\s*$')
        self.assertIn('python3 -B tests/webui/test_linux_retention_transport.py', makefile)

class OwnedFixtureContract(unittest.TestCase):

    def setUp(self):
        import tempfile
        from unittest.mock import patch
        self.temporary = tempfile.TemporaryDirectory(prefix='cb-owner-contract-')
        self.addCleanup(self.temporary.cleanup)
        self.base = Path(self.temporary.name).resolve()
        self.runtime = self.base / 'l1i-owned'
        self.runtime.mkdir()
        (self.runtime / '.webui-owned-runtime.json').write_text(json.dumps({'version': 1, 'path': str(self.runtime), 'policy': 'completed-disposable-data'}))
        (self.runtime / 'reports').mkdir()
        (self.runtime / 'work/c1').mkdir(parents=True)
        (self.runtime / 'private/c1/keys').mkdir(parents=True)
        patched = patch('runtime_storage.runtime_root', return_value=self.base)
        patched.start()
        self.addCleanup(patched.stop)
        self.fleet = '/private/tmp/cbui/lf-owned'
        self.root = '/data/chainbench/l1i-owned/c1'
        self.marker = {'runtime': str(self.runtime), 'workspace': str(self.runtime / 'work/c1'), 'sourceDigest': 'a' * 64, 'remoteDataRoot': self.root, 'fleetRuntime': self.fleet}
        self.record = {'keysDir': str(self.runtime / 'private/c1/keys'), 'compositionId': 'abc'}
        self.nodes = [{'index': 1, 'server': 'server1', 'dataDir': self.root + '/node/abc/node1', 'configPath': self.root + '/runtime/abc/configs/node1.toml', 'logPath': self.root + '/logs/abc/node1.log'}]
        self.record['nodes'] = self.nodes
        self.binary = {'/data/chainbench/bin/gstable': 'b' * 64}
        self.reports = self.runtime / 'reports/retention-c1'

    def validate(self, **changes):
        args = {'data_root': self.root, 'record': self.record, 'expected_binaries': self.binary, 'marker': self.marker, 'fleet_runtime': self.fleet, 'reports': self.reports}
        args.update(changes)
        return batch._validate_request(**args)

    def test_owned_fixture_is_accepted(self):
        self.validate()

    def test_private_key_alias_to_shared_input_is_refused(self):
        keys = self.runtime / 'private/c1/keys'
        keys.rmdir()
        shared = self.base / 'shared-keys'
        shared.mkdir()
        keys.symlink_to(shared, target_is_directory=True)
        with self.assertRaises(ValueError):
            self.validate()

    def test_other_run_and_shared_keys_are_refused(self):
        for key in [str(self.runtime / 'private/c2/keys'), 'presets/keys']:
            with self.subTest(key=key), self.assertRaises(ValueError):
                self.validate(record={**self.record, 'keysDir': key})

    def test_marker_or_fleet_change_is_refused(self):
        for (field, value) in [('sourceDigest', 'not-a-digest'), ('remoteDataRoot', self.root + '0'), ('fleetRuntime', '/unrelated'), ('workspace', str(self.runtime / 'work/c2'))]:
            with self.subTest(field=field), self.assertRaises(ValueError):
                self.validate(marker={**self.marker, field: value})

    def test_reports_outside_runtime_are_refused(self):
        with self.assertRaises(ValueError):
            self.validate(reports=self.base / 'other-output')

    def test_non_fixture_binary_is_refused(self):
        for path in ['/etc/shadow', '/data/chainbench/bin/../gstable', '--help']:
            with self.subTest(path=path), self.assertRaises(ValueError):
                self.validate(expected_binaries={path: 'b' * 64})

    def test_unowned_container_never_reaches_process_probe(self):
        from unittest.mock import Mock, patch
        probe = Mock()
        info = {'id': 'c' * 64, 'running': True, 'labels': {'chainbench.runtime': '/unrelated', 'chainbench.serverIndex': '1'}}
        with patch.object(batch.subprocess, 'run', return_value=Mock(stdout=json.dumps(info).encode())):
            with self.assertRaises(ValueError):
                batch.reclaim_stopped(container='candidate', data_root=self.root, record=self.record, nodes=self.nodes, expected_binaries=self.binary, marker=self.marker, proc=probe, fleet_runtime=self.fleet, reports=self.reports, server='server1')
        probe.probe.assert_not_called()
        self.assertFalse(self.reports.exists())

    def test_incomplete_or_wrong_container_identity_is_refused(self):
        from unittest.mock import Mock, patch
        valid = {'id': 'c' * 64, 'running': True, 'labels': {'chainbench.runtime': self.fleet, 'chainbench.serverIndex': '1'}}
        invalid = [{**valid, 'running': False}, {**valid, 'id': ''}, {**valid, 'labels': {'chainbench.runtime': self.fleet, 'chainbench.serverIndex': '2'}}, {**valid, 'labels': None}]
        for info in invalid:
            with self.subTest(info=info), patch.object(batch.subprocess, 'run', return_value=Mock(stdout=json.dumps(info).encode())):
                with self.assertRaises(ValueError):
                    batch._inspect_owned('candidate', self.fleet, 'server1')
    def test_empty_node_selection_refuses_before_any_probe(self):
        self.refuse_node_selection([])

    def test_unrecorded_node_selection_refuses_before_any_probe(self):
        self.refuse_node_selection([{**self.nodes[0], 'index': 2}])

    def refuse_node_selection(self, nodes):
        from unittest.mock import Mock, patch
        probe = Mock()
        probe.probe.side_effect = AssertionError('must refuse an incomplete node group before observing it')
        with patch.object(batch, '_inspect_owned', return_value='c' * 64):
            with self.assertRaises(ValueError):
                batch.reclaim_stopped(container='candidate', data_root=self.root, record=self.record, nodes=nodes, expected_binaries=self.binary, marker=self.marker, proc=probe, fleet_runtime=self.fleet, reports=self.reports, server='server1')
        probe.probe.assert_not_called()

if __name__ == '__main__':
    unittest.main()
