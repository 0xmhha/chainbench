#!/usr/bin/env python3
"""Select the Go files that `go list -deps -json` says a command build uses,
restricted to the repository itself, and mirror them into a directory the AST
extractor can parse. Writes <out>/build/<project>-selected.json with the list."""
import json, os, shutil, sys, hashlib

def decode_stream(path):
    dec = json.JSONDecoder()
    with open(path) as f:
        buf = f.read()
    i, n = 0, len(buf)
    while i < n:
        while i < n and buf[i].isspace():
            i += 1
        if i >= n:
            break
        obj, j = dec.raw_decode(buf, i)
        yield obj
        i = j

def main(project, repo, golist, mirror, out_json):
    repo = os.path.realpath(repo)
    selected, external, errors, extra = [], [], [], []
    pkgs = 0
    for pkg in decode_stream(golist):
        pkgs += 1
        if pkg.get('Error') or pkg.get('DepsErrors'):
            errors.append({'ImportPath': pkg.get('ImportPath'), 'Error': pkg.get('Error'), 'DepsErrors': pkg.get('DepsErrors')})
        d = os.path.realpath(pkg.get('Dir', ''))
        if not d.startswith(repo + os.sep) and d != repo:
            external.append({'ImportPath': pkg.get('ImportPath'), 'Module': (pkg.get('Module') or {}).get('Path'), 'Standard': pkg.get('Standard', False)})
            continue
        rel = os.path.relpath(d, repo)
        for key in ('GoFiles', 'CgoFiles'):
            for f in pkg.get(key, []):
                selected.append({'package': pkg.get('ImportPath'), 'path': os.path.join(rel, f), 'kind': key})
        for key in ('CFiles', 'SFiles', 'HFiles', 'EmbedFiles'):
            for f in pkg.get(key, []):
                extra.append({'package': pkg.get('ImportPath'), 'path': os.path.join(rel, f), 'kind': key})
    if os.path.exists(mirror):
        shutil.rmtree(mirror)
    for s in selected:
        src = os.path.join(repo, s['path']); dst = os.path.join(mirror, s['path'])
        os.makedirs(os.path.dirname(dst), exist_ok=True)
        shutil.copyfile(src, dst)
        with open(src, 'rb') as fh:
            s['sha256'] = hashlib.sha256(fh.read()).hexdigest()
    in_repo_pkgs = sorted({s['package'] for s in selected})
    json.dump({'project': project, 'repo': repo, 'golist': os.path.basename(golist), 'packages_total': pkgs,
               'packages_in_repo': len(in_repo_pkgs), 'go_files': len(selected), 'errors': errors,
               'selected': selected, 'non_go_inputs': extra,
               'external_packages': len([e for e in external if not e['Standard']]),
               'stdlib_packages': len([e for e in external if e['Standard']]),
               'external': sorted({e['Module'] for e in external if e['Module']})},
              open(out_json, 'w'), indent=1, ensure_ascii=False)
    print(f"{project}: pkgs={pkgs} in_repo={len(in_repo_pkgs)} go_files={len(selected)} errors={len(errors)} non_go={len(extra)}")

if __name__ == '__main__':
    main(*sys.argv[1:6])
