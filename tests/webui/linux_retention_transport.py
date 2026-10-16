"""Preserve an isolated, stopped Linux fixture before reclaiming its databases.

Remote metadata travels in one package. Container ownership, two fresh process
and file-descriptor inventories, filesystem/archive equality and the exact
disposable namespace remain mandatory. The caller supplies the pinned binary
hashes and the source-bound marker created with its owned local runtime.
"""

import hashlib, io, json, os, posixpath, re, subprocess, tarfile

from pathlib import Path

import linux_retention as contract

MEMBERS = {'before.meta', 'before.hash', 'after.meta', 'after.hash', 'archive.tar.gz', 'roots.txt', 'missing.txt', 'candidates.raw', 'binary.hash', 'marker.json', 'node-roots.txt'}

CAPTURE = r'''set -eu
umask 077
cd /data/chainbench
tmp=$(mktemp -d /tmp/cb-retain-XXXXXXXX)
trap 'rm -rf -- "$tmp"' EXIT HUP INT TERM
root=$1; n=$2; shift 2
: > "$tmp/roots.txt"; : > "$tmp/missing.txt"
i=0
while [ "$i" -lt "$n" ]; do
 p=$1; shift
 [ ! -L "$p" ] || exit 11
 if [ -e "$p" ]; then
  [ "$(readlink -f "$p")" = "$p" ] || exit 12
  printf '%s\n' "${p#/data/chainbench/}" >> "$tmp/roots.txt"
 else printf '%s\n' "${p#/data/chainbench/}" >> "$tmp/missing.txt"
 fi
 i=$((i+1))
done
n=$1; shift; i=0; : > "$tmp/node-roots.txt"
while [ "$i" -lt "$n" ]; do
 p=$1; shift
 [ ! -L "$p" ] || exit 13
 if [ -e "$p" ]; then
  [ "$(readlink -f "$p")" = "$p" ] || exit 14
  printf '%s\n' "$p" >> "$tmp/node-roots.txt"
 fi
 i=$((i+1))
done
sha256sum -- "$@" > "$tmp/binary.hash"
cat "$root/.chainbench-owned-runtime.json" > "$tmp/marker.json"
set --
while IFS= read -r p; do set -- "$@" "$p"; done < "$tmp/roots.txt"
[ "$#" -gt 0 ] || exit 15
find "$@" \( -name chaindata -o -name lightchaindata -o -name etcd -o -name triecache \) -prune -o -printf '%p\0%y\0%m\0%U\0%G\0%T@\0%s\0%l\0' > "$tmp/before.meta"
find "$@" \( -name chaindata -o -name lightchaindata -o -name etcd -o -name triecache \) -prune -o -type f -exec sha256sum -z -- {} + > "$tmp/before.hash"
tar --create --gzip --format=pax --file="$tmp/archive.tar.gz" --directory=/data/chainbench '--exclude=*/chaindata' '--exclude=*/lightchaindata' '--exclude=*/etcd' '--exclude=*/triecache' "$@"
find "$@" \( -name chaindata -o -name lightchaindata -o -name etcd -o -name triecache \) -prune -o -printf '%p\0%y\0%m\0%U\0%G\0%T@\0%s\0%l\0' > "$tmp/after.meta"
find "$@" \( -name chaindata -o -name lightchaindata -o -name etcd -o -name triecache \) -prune -o -type f -exec sha256sum -z -- {} + > "$tmp/after.hash"
set --
while IFS= read -r p; do set -- "$@" "$p"; done < "$tmp/node-roots.txt"
: > "$tmp/candidates.raw"
if [ "$#" -gt 0 ]; then
 find "$@" \( -name chaindata -o -name lightchaindata -o -name etcd -o -name triecache \) -prune ! -type d -print0 > "$tmp/non-directories"
 [ ! -s "$tmp/non-directories" ] || exit 27
 find "$@" \( -name chaindata -o -name lightchaindata -o -name etcd -o -name triecache \) -prune -print0 > "$tmp/candidates.unsorted"
 sort -z "$tmp/candidates.unsorted" > "$tmp/candidates.raw"
fi
tar --create --file=- --directory="$tmp" before.meta before.hash after.meta after.hash archive.tar.gz roots.txt missing.txt candidates.raw binary.hash marker.json node-roots.txt
'''

DELETE = r'''set -eu
umask 077
root=$1; expected=$2; shift 2
[ ! -L "$root" ] && [ "$(readlink -f "$root")" = "$root" ] || exit 20
for p do [ ! -L "$p" ] && [ "$(readlink -f "$p")" = "$p" ] || exit 21; done
tmp=$(mktemp -d /tmp/cb-delete-XXXXXXXX)
trap 'rm -rf -- "$tmp"' EXIT HUP INT TERM
: > "$tmp/candidates"
if [ "$#" -gt 0 ]; then find "$@" \( -name chaindata -o -name lightchaindata -o -name etcd -o -name triecache \) -prune -print0 > "$tmp/unsorted"; sort -z "$tmp/unsorted" > "$tmp/candidates"; fi
actual=$(sha256sum "$tmp/candidates"); actual=${actual%% *}
[ "$actual" = "$expected" ] || exit 22
xargs -0 -r /bin/sh -c '
 root=$1; shift
 for p do
  case "$p" in "$root"/node/*) ;; *) exit 23 ;; esac
  case "${p##*/}" in chaindata|lightchaindata|etcd|triecache) ;; *) exit 24 ;; esac
  [ -d "$p" ] && [ ! -L "$p" ] && [ "$(readlink -f "$p")" = "$p" ] || exit 25
 done
 for p do rm -rf -- "$p"; done
' -- "$root" < "$tmp/candidates"
if [ "$#" -gt 0 ]; then find "$@" \( -name chaindata -o -name lightchaindata -o -name etcd -o -name triecache \) -prune -print0 > "$tmp/remaining"; [ ! -s "$tmp/remaining" ] || exit 26; fi
cat "$tmp/candidates"
printf 'end\0'
'''

def eligible_database(path, node_roots):
    for root in node_roots:
        if not path.startswith(root + '/'):
            continue
        parts = path[len(root) + 1:].split('/')
        return parts == ['etcd'] or (len(parts) == 2 and parts[0] in {'geth', 'gwemix', 'gwbft', 'gstable'} and (parts[1] in contract.DATABASE_NAMES))
    return False

def _private_write(path, data):
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 384)
    with os.fdopen(fd, 'wb') as f:
        os.fchmod(f.fileno(), 0o600)
        f.write(data)
        f.flush()
        os.fsync(f.fileno())

def unpack(payload):
    result = {}
    with tarfile.open(fileobj=io.BytesIO(payload), mode='r:') as package:
        for member in package:
            if not member.isfile() or member.name not in MEMBERS or member.name in result:
                raise ValueError('unexpected metadata package member')
            result[member.name] = package.extractfile(member).read()
    if result.keys() != MEMBERS:
        raise ValueError('incomplete metadata package')
    return result

def hashes(raw):
    if raw and (not raw.endswith(b'\x00')):
        raise ValueError('incomplete checksum observation')
    result = {}
    for entry in raw.split(b'\x00'):
        if not entry:
            continue
        if entry[64:66] != b'  ':
            raise ValueError('malformed checksum')
        path = entry[66:].decode()
        value = entry[:64].decode()
        if path in result:
            raise ValueError('duplicate checksum')
        result[path] = value
    return result

def parse_bundle(payload, roots, nodes, expected_binaries, marker):
    parts = unpack(payload)
    selected = [p.removeprefix('/data/chainbench/') for p in roots]
    present = parts['roots.txt'].decode().splitlines()
    missing = parts['missing.txt'].decode().splitlines()
    if len(present) != len(set(present)) or len(missing) != len(set(missing)) or set(present) & set(missing) or (set(present + missing) != set(selected)):
        raise ValueError('selected root set changed')
    for p in present + missing:
        contract.canonical_relative(p)
    if json.loads(parts['marker.json']) != marker:
        raise ValueError('ownership marker changed')
    binary_hashes = {}
    for line in parts['binary.hash'].decode().splitlines():
        (sha, sep, path) = line.partition('  ')
        if not sep or path in binary_hashes:
            raise ValueError('invalid binary checksum list')
        binary_hashes[path] = sha
    if binary_hashes != expected_binaries:
        raise ValueError('baseline binary changed')
    before = contract.decode_find_metadata(parts['before.meta'], hashes(parts['before.hash']))
    after = contract.decode_find_metadata(parts['after.meta'], hashes(parts['after.hash']))
    contract.require_match(before, after)
    for (rows, label) in [(before, 'before.hash'), (after, 'after.hash')]:
        if set(hashes(parts[label])) != {r['path'] for r in rows if r['type'] == 'f'}:
            raise ValueError('checksum scope differs from metadata')
    archived = []
    with tarfile.open(fileobj=io.BytesIO(parts['archive.tar.gz']), mode='r:gz') as archive:
        for member in archive:
            kind = 'f' if member.isfile() else 'd' if member.isdir() else 'l' if member.issym() else None
            if kind is None:
                raise ValueError('unsupported archive entry')
            row = {'path': contract.canonical_relative(member.name), 'type': kind, 'mode': member.mode, 'uid': member.uid, 'gid': member.gid, 'mtimeNS': contract.mtime_ns(member.pax_headers.get('mtime', str(member.mtime))), 'size': member.size if kind == 'f' else 0, 'link': member.linkname if kind == 'l' else ''}
            if kind == 'f':
                digest = hashlib.sha256()
                with archive.extractfile(member) as payload_file:
                    for block in iter(lambda : payload_file.read(1048576), b''):
                        digest.update(block)
                row['sha256'] = digest.hexdigest()
            archived.append(row)
    contract.require_match(before, archived)
    marker_path = marker['remoteDataRoot'].removeprefix('/data/chainbench/') + '/.chainbench-owned-runtime.json'
    row = next((r for r in before if r['path'] == marker_path), None)
    if not row or row['type'] != 'f' or row['sha256'] != hashlib.sha256(parts['marker.json']).hexdigest():
        raise ValueError('archive marker differs from ownership input')
    for row in before:
        if not any((row['path'] == p or row['path'].startswith(p + '/') for p in present)):
            raise ValueError('filesystem row outside selected scope')
    node_roots = parts['node-roots.txt'].decode().splitlines()
    if len(node_roots) != len(set(node_roots)) or not set(node_roots) <= set(nodes):
        raise ValueError('node root set changed')
    candidates = [v.decode() for v in parts['candidates.raw'].split(b'\x00') if v]
    if parts['candidates.raw'] and (not parts['candidates.raw'].endswith(b'\x00')):
        raise ValueError('incomplete candidate list')
    if candidates != sorted(set(candidates)):
        raise ValueError('duplicate or unordered candidates')
    for p in candidates:
        if posixpath.normpath(p) != p or p.rsplit('/', 1)[-1] not in contract.DATABASE_NAMES or (not eligible_database(p, node_roots)):
            raise ValueError('candidate outside disposable node database namespace')
    return (parts, before, after, present, missing, node_roots, candidates)

def _capture(container, roots, nodes, binaries, data_root):
    args = [data_root, str(len(roots)), *roots, str(len(nodes)), *nodes, *binaries]
    r = subprocess.run(['docker', 'exec', '-i', '-u', 'root', container, '/bin/sh', '-s', '--', *args], input=CAPTURE.encode(), capture_output=True)
    if r.returncode:
        raise RuntimeError('batched capture exit ' + str(r.returncode) + '; retain DB: ' + r.stderr.decode(errors='replace'))
    return (r.stdout, r.stderr)

def _delete(container, data_root, node_roots, candidates):
    raw = b''.join((p.encode() + b'\x00' for p in candidates))
    expected = hashlib.sha256(raw).hexdigest()
    r = subprocess.run(['docker', 'exec', '-i', '-u', 'root', container, '/bin/sh', '-s', '--', data_root, expected, *node_roots], input=DELETE.encode(), capture_output=True)
    if r.returncode or r.stdout != raw + b'end\x00':
        raise RuntimeError('DB deletion or post-check exit ' + str(r.returncode) + '; halt: ' + r.stderr.decode(errors='replace'))
    return r.stdout

def _inspect_owned(container, fleet_runtime, server):
    match = re.fullmatch('server([1-9][0-9]*)', server)
    if match is None:
        raise ValueError('invalid owned server identifier')
    format_ = '{"id":{{json .Id}},"running":{{json .State.Running}},"labels":{{json .Config.Labels}}}'
    result = subprocess.run(['docker', 'inspect', '--format', format_, container], capture_output=True, check=True)
    info = json.loads(result.stdout)
    labels = info.get('labels') or {}
    if info.get('running') is not True or labels.get('chainbench.runtime') != fleet_runtime or labels.get('chainbench.serverIndex') != match[1] or (not re.fullmatch('[a-f0-9]{64}', info.get('id', ''))):
        raise ValueError('container identity or ownership changed')
    return info['id']

def _validate_request(data_root, record, expected_binaries, marker, fleet_runtime, reports):
    from runtime_storage import validate_runtime
    fields = {'runtime', 'sourceDigest', 'workspace', 'remoteDataRoot', 'fleetRuntime'}
    if set(marker) != fields or any((not isinstance(v, str) or any((c in v for c in '\x00\r\n')) for v in marker.values())):
        raise ValueError('invalid proof ownership marker')
    runtime = validate_runtime(marker['runtime'])
    workspace = Path(marker['workspace'])
    reports = Path(reports)
    if runtime != runtime.resolve() or not re.fullmatch('[A-Za-z0-9_-]+', runtime.name) or workspace != workspace.resolve() or (workspace.parent != runtime / 'work') or (not re.fullmatch('c[1-9][0-9]*', workspace.name)):
        raise ValueError('workspace is not inside the owned runtime')
    if data_root != '/data/chainbench/' + runtime.name + '/' + workspace.name or marker['remoteDataRoot'] != data_root or marker['fleetRuntime'] != fleet_runtime or (not re.fullmatch('[a-f0-9]{64}', marker['sourceDigest'])):
        raise ValueError('marker does not bind the requested fixture')
    if Path(record['keysDir']) != runtime / 'private' / workspace.name / 'keys' or Path(record['keysDir']) != Path(record['keysDir']).resolve():
        raise ValueError('record names shared or different private keys')
    if reports != reports.resolve() or not reports.is_relative_to(runtime / 'reports'):
        raise ValueError('retention output outside owned reports')
    if not expected_binaries or any((not re.fullmatch('/data/chainbench/bin/[A-Za-z0-9][A-Za-z0-9_.-]*', p) or not re.fullmatch('[a-f0-9]{64}', h) for (p, h) in expected_binaries.items())):
        raise ValueError('invalid baseline binary inventory')

def reclaim_stopped(*, container, data_root, record, nodes, expected_binaries, marker, proc, fleet_runtime, reports, server):
    all_nodes = record.get('nodes', [])
    if not isinstance(all_nodes, list) or any(not isinstance(node, dict) for node in all_nodes):
        raise ValueError('invalid recorded node inventory')
    if not isinstance(nodes, list) or not nodes or nodes != [node for node in all_nodes if node.get('server') == server]:
        raise ValueError('selection must contain exactly this server’s recorded nodes')
    indices = [node.get('index') for node in all_nodes]
    if any(not isinstance(index, int) or isinstance(index, bool) or index < 1 for index in indices) or len(indices) != len(set(indices)):
        raise ValueError('invalid or repeated node index')
    _validate_request(data_root, record, expected_binaries, marker, fleet_runtime, reports)
    container_id = _inspect_owned(container, fleet_runtime, server)
    reports = Path(reports)
    reports.mkdir(mode=448, parents=True, exist_ok=True)

    def save(label, value):
        _private_write(reports / (server + '-' + label + '.json'), (json.dumps(value, indent=2) + '\n').encode())
    selected = [*contract.selected_roots(record, nodes, data_root=data_root), data_root + '/.chainbench-owned-runtime.json']
    node_dirs = [n['dataDir'] for n in nodes]
    first = proc.probe(container_id, fleet_runtime)
    proc.require_vacant(first, [data_root])
    save('before-proc', first)
    (payload, stderr) = _capture(container_id, selected, node_dirs, list(expected_binaries), data_root)
    (parts, before, after, present, missing, node_roots, candidates) = parse_bundle(payload, selected, node_dirs, expected_binaries, marker)
    archive = reports / (server + '-metadata.tar.gz')
    _private_write(archive, parts['archive.tar.gz'])
    contract.require_match(before, contract.decode_archive(archive))
    _private_write(reports / (server + '-archive-stderr.txt'), stderr)
    save('filesystem-before', before)
    save('filesystem-after', after)
    save('transport', {'packageSHA256': hashlib.sha256(payload).hexdigest(), 'binaryHashes': expected_binaries, 'marker': marker, 'selectedRoots': present, 'missingRoots': missing, 'candidates': candidates, 'nodeRoots': node_roots})
    second = proc.probe(container_id, fleet_runtime)
    proc.require_vacant(second, [data_root])
    save('after-proc', second)
    if _inspect_owned(container, fleet_runtime, server) != container_id:
        raise ValueError('container changed before deletion')
    deleted = _delete(container_id, data_root, node_roots, candidates)
    _private_write(reports / (server + '-deletion-result.raw'), deleted)
    return {'server': server, 'container': container, 'selectedRoots': present, 'missingRoots': missing, 'archive': str(archive), 'archiveSHA256': hashlib.sha256(parts['archive.tar.gz']).hexdigest(), 'preArchiveFilesystem': str(reports / (server + '-filesystem-before.json')), 'metadataEntries': len(before), 'filesystemBeforeArchiveAfterMatched': True, 'freshCompleteProcessInventories': 2, 'reclaimedPaths': candidates, 'remainingDatabasePaths': [], 'batchedTransport': True, 'containerID': container_id, 'remoteCaptureCalls': 1, 'remoteDeleteCalls': 1}
