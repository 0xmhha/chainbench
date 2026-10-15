"""A retained archive must agree with an independent pre-archive inventory."""
import io
from pathlib import Path
import tarfile
import tempfile
import unittest

import linux_retention as retention


class RetentionContract(unittest.TestCase):
    def row(self):
        return {"path": "keys/password", "type": "f", "mode": 0o600, "uid": 1000,
                "gid": 1000, "mtimeNS": 1234567890123456789, "size": 5,
                "sha256": "a" * 64, "link": ""}

    def test_selection_is_current_composition_only(self):
        nodes = [{"dataDir": "/data/chainbench/node/abc123/node1",
                  "configPath": "/data/chainbench/runtime/abc123/configs/node1.toml",
                  "logPath": "/data/chainbench/logs/abc123/node1.log"}]
        self.assertEqual(retention.selected_roots({"compositionId": "abc123"}, nodes), [
            "/data/chainbench/keys", "/data/chainbench/logs/abc123",
            "/data/chainbench/node/abc123/node1", "/data/chainbench/runtime/abc123"])

    def test_rejects_unowned_or_ambiguous_selection(self):
        for composition, path in [("../old", "/data/chainbench/node/old/node1"),
                                  ("abc123", "/etc/passwd"),
                                  ("abc123", "/data/chainbench/node/other/node1")]:
            with self.subTest(composition=composition, path=path):
                with self.assertRaises(ValueError):
                    retention.selected_roots({"compositionId": composition}, [{
                        "dataDir": path, "configPath": "/data/chainbench/runtime/abc123/configs/node1.toml",
                        "logPath": "/data/chainbench/logs/abc123/node1.log"}])

    def test_matching_archive(self):
        retention.require_match([self.row()], [self.row()])

    def test_archive_must_match_pre_archive_filesystem(self):
        row = self.row()
        for field, value in [("path", "keys/other"), ("type", "l"), ("mode", 0o644),
                             ("uid", 0), ("gid", 0), ("mtimeNS", 1234567890123456788),
                             ("size", 6), ("sha256", "b" * 64), ("link", "outside")]:
            with self.subTest(field=field):
                with self.assertRaises(ValueError):
                    retention.require_match([row], [{**row, field: value}])

    def test_missing_extra_and_duplicate_files_refused(self):
        row = self.row()
        for observed in [[], [row, {**row, "path": "keys/q"}], [row, row]]:
            with self.subTest(observed=observed):
                with self.assertRaises(ValueError):
                    retention.require_match([row], observed)

    def test_archive_and_find_decoders_keep_exact_metadata(self):
        import hashlib
        data = b"proof"
        with tempfile.TemporaryDirectory() as root:
            path = Path(root) / "metadata.tar.gz"
            with tarfile.open(path, "w:gz", format=tarfile.PAX_FORMAT) as archive:
                for name, kind in [("keys/password", "f"), ("keys", "d"), ("keys/current", "l")]:
                    entry = tarfile.TarInfo(name)
                    entry.type = {"f": tarfile.REGTYPE, "d": tarfile.DIRTYPE, "l": tarfile.SYMTYPE}[kind]
                    entry.mode = 0o600
                    entry.uid = entry.gid = 1000
                    entry.pax_headers = {"mtime": "1234567890.123456789"}
                    entry.size = len(data) if kind == "f" else 0
                    entry.linkname = "password" if kind == "l" else ""
                    archive.addfile(entry, io.BytesIO(data) if kind == "f" else None)
            raw = b""
            for name, kind, size, link in [("keys/password", "f", "5", ""), ("keys", "d", "4096", ""), ("keys/current", "l", "8", "password")]:
                raw += ("\0".join([name, kind, "600", "1000", "1000", "1234567890.1234567890", size, link]) + "\0").encode()
            retention.require_match(retention.decode_find_metadata(raw, {"keys/password": hashlib.sha256(data).hexdigest()}), retention.decode_archive(path))

    def test_unsafe_members_and_incomplete_observations_refused(self):
        for path in ["", ".", "../keys", "keys/../p", "/etc/passwd", "keys//p", "node/chaindata/LOCK"]:
            with self.subTest(path=path):
                with self.assertRaises(ValueError):
                    retention.canonical_relative(path)
        with self.assertRaises(ValueError):
            retention.decode_find_metadata(b"keys/p\0f\0", {})
        with self.assertRaises(ValueError):
            retention.mtime_ns("1.0000000001")


if __name__ == "__main__":
    unittest.main()
