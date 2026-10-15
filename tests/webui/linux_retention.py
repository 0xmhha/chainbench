"""Compare scoped Linux proof archives with pre-archive filesystem observations."""
from decimal import Decimal
import hashlib
from pathlib import PurePosixPath
import re
import tarfile


DATABASE_NAMES = ("chaindata", "lightchaindata", "etcd", "triecache")


def selected_roots(record, nodes):
    """Keep shared keys and this composition, rather than every historical run."""
    composition = record["compositionId"]
    if not re.fullmatch(r"[a-zA-Z0-9_-]+", composition):
        raise ValueError("unsafe composition identifier")
    roots = {
        "/data/chainbench/keys",
        "/data/chainbench/runtime/" + composition,
        "/data/chainbench/logs/" + composition,
    }
    for node in nodes:
        for key, base in (("dataDir", "node"), ("configPath", "runtime"), ("logPath", "logs")):
            value = node[key]
            prefix = "/data/chainbench/" + base + "/" + composition + "/"
            if not value.startswith(prefix) or any(part in {".", "..", ""} for part in value.split("/")[1:]):
                raise ValueError("path outside current composition")
        roots.add(node["dataDir"])
    return sorted(roots)


def require_match(expected, actual):
    before = {row["path"]: row for row in expected}
    archived = {row["path"]: row for row in actual}
    if len(before) != len(expected) or len(archived) != len(actual) or before != archived:
        raise ValueError("archive differs from pre-archive filesystem snapshot")


def canonical_relative(path):
    """An archive member must remain a canonical descendant of the data root."""
    if path.startswith("./"):
        path = path[2:]
    if path.endswith("/"):
        path = path[:-1]
    if path in {"", ".", ".."} or path.startswith("/") or str(PurePosixPath(path)) != path or ".." in PurePosixPath(path).parts:
        raise ValueError("noncanonical archive path")
    if any(part in DATABASE_NAMES for part in PurePosixPath(path).parts):
        raise ValueError("disposable database unexpectedly archived")
    return path


def mtime_ns(value):
    nanoseconds = Decimal(value) * 10**9
    if nanoseconds != nanoseconds.to_integral_value():
        raise ValueError("timestamp loses nanosecond precision")
    return int(nanoseconds)


def decode_find_metadata(raw, hashes):
    """Decode GNU find's NUL-delimited path/type/mode/uid/gid/mtime/size/link."""
    fields = raw.split(b"\0")
    if fields[-1] != b"" or (len(fields) - 1) % 8:
        raise ValueError("incomplete filesystem metadata inventory")
    result = []
    for offset in range(0, len(fields) - 1, 8):
        path, kind, mode, uid, gid, mtime, size, link = [part.decode() for part in fields[offset:offset + 8]]
        relative = canonical_relative(path)
        if kind not in {"f", "d", "l"}:
            raise ValueError("unsupported filesystem entry; preserve data")
        row = {
            "path": relative, "type": kind, "mode": int(mode, 8),
            "uid": int(uid), "gid": int(gid), "mtimeNS": mtime_ns(mtime),
            "size": int(size) if kind == "f" else 0, "link": link,
        }
        if kind == "f":
            checksum = hashes[relative]
            if not re.fullmatch(r"[a-f0-9]{64}", checksum):
                raise ValueError("invalid file checksum")
            row["sha256"] = checksum
        result.append(row)
    return result


def decode_archive(path):
    result = []
    with tarfile.open(path, "r:gz") as archive:
        for member in archive:
            if member.isfile():
                kind = "f"
            elif member.isdir():
                kind = "d"
            elif member.issym():
                kind = "l"
            else:
                raise ValueError("unsupported archive entry; preserve data")
            row = {
                "path": canonical_relative(member.name), "type": kind,
                "mode": member.mode, "uid": member.uid, "gid": member.gid,
                "mtimeNS": mtime_ns(member.pax_headers.get("mtime", str(member.mtime))),
                "size": member.size if kind == "f" else 0,
                "link": member.linkname if kind == "l" else "",
            }
            if kind == "f":
                digest = hashlib.sha256()
                with archive.extractfile(member) as payload:
                    for block in iter(lambda: payload.read(1024 * 1024), b""):
                        digest.update(block)
                row["sha256"] = digest.hexdigest()
            result.append(row)
    return result
