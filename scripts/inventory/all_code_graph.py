"""Parse every repository source into a structural graph; errors fail closed.

Uses Go's parser, Python ast, Acorn/Svelte/PostCSS, and pinned Tree-sitter
languages for build/HTML/config syntax, and mvdan syntax for shell programs. This is a syntax graph, not a
claim of type-resolved calls, runtime coverage, or cross-language execution.
"""
import argparse
import ast
from collections import Counter
import hashlib
from importlib.metadata import version
import json
import os
from pathlib import Path
import re
import subprocess
import sys

from inline_python import extract as python_programs, inline_program, interpreter

REPO = Path(__file__).resolve().parents[2]
LANGUAGES = {'.go': 'go', '.py': 'python', '.sh': 'bash', '.js': 'javascript',
             '.mjs': 'javascript', '.svelte': 'svelte', '.html': 'html', '.css': 'css',
             '.json': 'json', '.yaml': 'yaml', '.yml': 'yaml', '.toml': 'toml',
             '.mod': 'gomod', '.sum': 'gosum'}
DATA = {'.md', '.txt', '.mmd', '.tsv', '.jsonl', '.sample', '.golden'}
DECLARATIONS = {'function_definition': 'function', 'function_declaration': 'function',
                'class_definition': 'type', 'target': 'target', 'rule': 'target'}
CALLS = {'command', 'call_expression'}
PARSERS = {}


def files_at(root, all_files):
    if all_files:
        return sorted(p.relative_to(root).as_posix() for p in root.rglob('*') if p.is_file() or p.is_symlink())
    result = subprocess.run(['git', 'ls-files', '-z', '--cached', '--others', '--exclude-standard'], cwd=root, capture_output=True, check=True)
    return sorted(set(p for p in result.stdout.decode().split('\0') if p))


def classify(path, source):
    p = Path(path)
    if any(part in {'node_modules', '.git', '__pycache__'} for part in p.parts):
        return None, 'external dependency'
    if path.startswith('docs/dev/') and p.name.endswith('-evidence.json'):
        return None, 'generated proof metadata, not executable source'
    if path.startswith('presets/keys/') or '/testdata/fuzz/' in path:
        return None, 'test data, not executable source'
    if p.name == 'Makefile':
        return 'make', None
    if p.name == 'Dockerfile' or p.name.startswith('Dockerfile.'):
        return 'dockerfile', None
    if path.endswith('.go.txt'):
        return 'go', None
    if p.suffix in LANGUAGES:
        return LANGUAGES[p.suffix], None
    if source.startswith('#!'):
        head = source.splitlines()[0]
        if 'python' in head:
            return 'python', None
        if any(word in head for word in ('bash', '/sh', 'env sh')):
            return 'bash', None
    if p.suffix in DATA or p.name in {'LICENSE', 'CODEOWNERS', '.gitignore', '.golangci-version', '.precommit-allow'}:
        return None, 'documentation or non-executable data'
    return None, 'unsupported source or unclassified file'


def area(path):
    if path.startswith('internal/core/'):
        return 'core'
    for name in ['consensus', 'chains', 'chainsetup', 'testengine', 'app', 'mcp', 'dashboard', 'dsl', 'resource', 'preset', 'testhelper', 'webcli', 'feature', 'accounts']:
        if path.startswith('internal/' + name + '/'):
            return name
    return path.split('/', 1)[0]


def python_nodes(source):
    tree = ast.parse(source)
    nodes = []
    def name(n):
        if isinstance(n, ast.Name):
            return n.id
        if isinstance(n, ast.Attribute):
            return name(n.value) + '.' + n.attr
        return 'anonymous'
    for n in ast.walk(tree):
        kind = None
        label = ''
        if isinstance(n, (ast.FunctionDef, ast.AsyncFunctionDef, ast.ClassDef)):
            kind, label = ('type' if isinstance(n, ast.ClassDef) else 'function'), n.name
        elif isinstance(n, ast.Call):
            kind, label = 'call', name(n.func)
        elif isinstance(n, ast.Import):
            for alias in n.names:
                nodes.append({'kind': 'import', 'name': alias.name, 'line': n.lineno})
        elif isinstance(n, ast.ImportFrom):
            kind, label = 'import', '.' * n.level + (n.module or '')
        if kind:
            nodes.append({'kind': kind, 'name': label, 'line': n.lineno})
    embedded, dynamic = python_programs(tree)
    return {'astNodes': sum(1 for _ in ast.walk(tree)), 'nodes': nodes + dynamic, 'embedded': embedded}


def shell_literal(parts, quoted=False):
    chunks = []
    for part in parts:
        kind = part.get('Type')
        if kind == 'Lit':
            value = part.get('Value', '')
            if not quoted and re.search(r'(?<!\\)[*?\[~]', value):
                return None
            pattern = r'\\([\\"$`\n])' if quoted else r'\\(.)'
            chunks.append(re.sub(pattern, lambda m: '' if m[1] == '\n' else m[1], value, flags=re.S))
        elif kind == 'SglQuoted' and not part.get('Dollar'):
            chunks.append(part.get('Value', ''))
        elif kind == 'DblQuoted' and not part.get('Dollar'):
            value = shell_literal(part.get('Parts', []), quoted=True)
            if value is None:
                return None
            chunks.append(value)
        else:
            return None
    return ''.join(chunks)


def stdin_program(arguments):
    if not arguments:
        return None
    language = interpreter(arguments[0])
    if language is None:
        return None
    for flag in arguments[1:]:
        if not isinstance(flag, str):
            return None
        if flag == '-' or (language == 'bash' and flag == '-s'):
            return language
        if flag in {'-c', '-e', '--eval', '-m', '--help', '--version', '-V'}:
            return None
        if not flag.startswith('-') or flag == '--':
            return None
        if language == 'bash' and not flag.startswith('--') and 'c' in flag:
            return None
    return language


def shell_nodes(source):
    executable = os.environ.get('CHAINBENCH_AST_SHFMT', '/private/tmp/chainbench-ast-tools/shfmt')
    result = subprocess.run([executable, '-ln', 'bash', '-to-json'], input=source, capture_output=True, text=True, check=False)
    if result.returncode:
        raise ValueError(result.stderr.strip())
    tree = json.loads(result.stdout)
    nodes = []
    embedded = []
    count = 0
    raw = source.encode()
    def visit(value):
        nonlocal count
        if isinstance(value, dict):
            if 'Pos' in value:
                count += 1
            command = value.get('Cmd', {})
            args = command.get('Args', []) if command.get('Type') == 'CallExpr' else []
            arguments = [shell_literal(arg.get('Parts', [])) for arg in args]
            language, body = inline_program(arguments)
            if language:
                if body is None:
                    nodes.append({'kind': 'dynamic-program-input', 'name': language + ' interpreter payload', 'line': value['Pos']['Line']})
                else:
                    embedded.append({'language': language, 'source': body, 'line': value['Pos']['Line']})
            language = stdin_program(arguments)
            if language:
                for redirect in value.get('Redirs', []):
                    body = redirect.get('Hdoc')
                    if body:
                        chunks, substitutions = [], []
                        for part in body.get('Parts', []):
                            if part.get('Type') == 'Lit':
                                chunks.append(part.get('Value', ''))
                            else:
                                expression = raw[part['Pos']['Offset']:part['End']['Offset']].decode()
                                symbolic = 'CB_SHELL_' + re.sub(r'[^A-Za-z0-9_]', '_', expression)
                                chunks.append(symbolic)
                                substitutions.append({'expression': expression, 'symbol': symbolic, 'line': part['Pos']['Line']})
                        embedded.append({'language': language, 'source': ''.join(chunks), 'line': body['Pos']['Line'],
                                         'substitutions': substitutions})
            if value.get('Type') == 'FuncDecl':
                nodes.append({'kind': 'function', 'name': value['Name']['Value'], 'line': value['Pos']['Line']})
            if value.get('Type') == 'CallExpr' and value.get('Args'):
                word = value['Args'][0]
                label = raw[word['Pos']['Offset']:word['End']['Offset']].decode()
                nodes.append({'kind': 'call', 'name': label, 'line': value['Pos']['Line']})
            for key, child in value.items():
                if key not in {'Pos', 'End', 'Position'}:
                    visit(child)
        elif isinstance(value, list):
            for child in value:
                visit(child)
    visit(tree)
    return {'astNodes': count, 'nodes': nodes, 'embedded': embedded}


def make_nodes(source):
    if '.RECIPEPREFIX' in source:
        raise ValueError('custom Make recipe prefixes require explicit parser support')
    lines = source.splitlines(keepends=True)
    skeleton, recipes, current = [], [], []
    first = 0
    for number, line in enumerate(lines, 1):
        if line.startswith('\t'):
            if not current:
                first = number
            current.append(line[1:].lstrip('@+-'))
            skeleton.append('\ttrue\n')
        else:
            if current:
                recipes.append((first, ''.join(current)))
                current = []
            skeleton.append(line)
    if current:
        recipes.append((first, ''.join(current)))
    tree = syntax_nodes('make', ''.join(skeleton))
    transformations = []
    for line, original in recipes:
        # Make expands its own variables and $$ before the shell sees a recipe.
        # Keep symbolic variable references explicit, never execute make/eval.
        normalized = original.replace('$$', '\x00')
        references = []
        pattern = re.compile(r'\$\(([^()]*)\)|\$\{([^{}]*)\}')
        while pattern.search(normalized):
            def replace(match):
                references.append(match.group(1) or match.group(2))
                return 'CB_MAKE_SYMBOL'
            normalized = pattern.sub(replace, normalized)
        normalized = normalized.replace('\x00', '$')
        parsed = shell_nodes(normalized)
        for n in parsed['nodes']:
            n['line'] += line - 1
        tree['nodes'].extend(parsed['nodes'])
        tree['embedded'].extend(parsed.get('embedded', []))
        tree['astNodes'] += parsed['astNodes']
        transformations.append({'kind': 'make-recipe-extraction', 'line': line, 'symbolicVariables': references,
                                'rawSHA256': hashlib.sha256(original.encode()).hexdigest()})
    tree['transformations'] = transformations
    return tree


def syntax_nodes(language, source):
    from tree_sitter_language_pack import get_parser
    raw = source.encode()
    if language not in PARSERS:
        PARSERS[language] = get_parser(language)
    tree = PARSERS[language].parse(raw)
    if tree.root_node.has_error:
        pending = [tree.root_node]
        while pending:
            n = pending.pop()
            if n.type == 'ERROR' or n.is_missing:
                raise ValueError(f'{language} syntax error at {n.start_point}')
            pending.extend(reversed(n.children))
        raise ValueError(language + ' syntax error')
    nodes, embedded, count = [], [], 0
    pending = [tree.root_node]
    while pending:
        n = pending.pop()
        count += 1
        pending.extend(reversed(n.named_children))
        kind = DECLARATIONS.get(n.type) or ('call' if n.type in CALLS else None)
        if kind:
            label = n.child_by_field_name('name') or n.child_by_field_name('function')
            nodes.append({'kind': kind, 'name': raw[label.start_byte:label.end_byte].decode() if label else n.type, 'line': n.start_point.row + 1})
        if language == 'dockerfile' and n.type == 'shell_command':
            embedded.append({'language': 'bash', 'source': raw[n.start_byte:n.end_byte].decode(), 'line': n.start_point.row + 1})
        if language == 'html' and n.type in {'script_element', 'style_element'}:
            for child in n.named_children:
                if child.type == 'raw_text':
                    embedded.append({'language': 'javascript' if n.type == 'script_element' else 'css', 'source': raw[child.start_byte:child.end_byte].decode(), 'line': child.start_point.row + 1})
    return {'astNodes': count, 'nodes': nodes, 'embedded': embedded}


def workflow_nodes(source):
    import yaml
    result = syntax_nodes('yaml', source)
    document = yaml.safe_load(source) or {}
    for job in document.get('jobs', {}).values():
        for step in job.get('steps', []):
            if 'run' not in step:
                continue
            raw = step['run']
            if not isinstance(raw, str):
                raise ValueError('workflow run must be a string')
            declared_shell = step.get('shell', job.get('defaults', {}).get('run', {}).get('shell', 'bash'))
            shell = declared_shell.split()[0]
            language = 'python' if shell.startswith('python') else 'bash' if shell in {'bash', 'sh'} else None
            if language is None:
                raise ValueError('unsupported workflow shell: ' + declared_shell)
            expressions = re.findall(r'\$\{\{.*?\}\}', raw, re.S)
            normalized = re.sub(r'\$\{\{.*?\}\}', 'CB_GITHUB_EXPRESSION', raw, flags=re.S)
            result['embedded'].append({'language': language, 'source': normalized, 'line': 1})
            result.setdefault('transformations', []).append({'kind': 'workflow-run-extraction', 'expressions': expressions,
                                                            'rawSHA256': hashlib.sha256(raw.encode()).hexdigest()})
    return result


def json_nodes(source, template=False):
    parameters = []
    if template:
        chunks = []
        quoted = escaped = False
        index = 0
        while index < len(source):
            char = source[index]
            if not quoted:
                match = re.match(r'__[A-Z][A-Z0-9_]*__', source[index:])
                if match:
                    parameters.append({'name': match.group(), 'offset': index})
                    chunks.append(json.dumps(match.group()))
                    index += len(match.group())
                    continue
            chunks.append(char)
            if char == '"' and not escaped:
                quoted = not quoted
            escaped = char == '\\' and not escaped if quoted else False
            index += 1
        normalized = ''.join(chunks)
    else:
        normalized = source
    tree = json.loads(normalized)
    is_dsl = isinstance(tree, dict) and (tree.get('kind') in {'case', 'spec'} or isinstance(tree.get('steps'), list))
    nodes = []
    count = 0
    def visit(value):
        nonlocal count
        count += 1
        if isinstance(value, dict):
            for key in ('do', 'expect', 'read'):
                if is_dsl and isinstance(value.get(key), str):
                    nodes.append({'kind': 'dsl-operation', 'name': value[key], 'line': 0})
            for v in value.values():
                visit(v)
        elif isinstance(value, list):
            for v in value:
                visit(v)
    visit(tree)
    return {'astNodes': count, 'nodes': nodes + [{'kind': 'template-parameter', 'name': p['name'], 'line': 0} for p in parameters],
            'transformations': [{'kind': 'json-template-parameters', 'parameters': parameters}] if parameters else []}


def build(root, all_files=False):
    graph = {'files': [], 'nodes': [], 'edges': [], 'excluded': [], 'missing': [],
             'scope': 'all owned source including tests/build tags and generated SPA; external dependencies and non-executable data classified separately',
             'limitations': 'Syntax-derived declarations/imports/calls and statically resolved interpreter payloads; dynamic-program-input nodes mark unresolved recognized payloads. No type resolution, runtime coverage, interprocedural evaluation or complete dynamic call graph.'}
    inputs, parsed = [], {}
    for path in files_at(root, all_files):
        p = root / path
        if p.is_symlink():
            graph['missing'].append({'path': path, 'reason': 'symlink'})
            continue
        _, reason = classify(path, '')
        if reason and not reason.startswith('unsupported'):
            graph['excluded'].append({'path': path, 'reason': reason})
            continue
        try:
            source = p.read_text()
        except (OSError, UnicodeError) as error:
            graph['missing'].append({'path': path, 'reason': str(error)})
            continue
        language, reason = classify(path, source)
        if reason:
            category = 'missing' if reason.startswith('unsupported') else 'excluded'
            graph[category].append({'path': path, 'reason': reason})
            continue
        inputs.append({'path': path, 'language': language, 'source': source})
    go = [i for i in inputs if i['language'] == 'go']
    frontend = [i for i in inputs if i['language'] in {'javascript', 'svelte', 'css'}]
    for command, batch in [(['go', 'run', './scripts/inventory/source-ast'], go),
                           (['node', str(REPO / 'scripts/inventory/frontend_ast.mjs')], frontend)]:
        if batch:
            result = subprocess.run(command, input=json.dumps(batch), cwd=REPO, capture_output=True, text=True, check=False)
            if result.returncode:
                parsed.update({i['path']: {'error': 'parser backend failed: ' + result.stderr.strip()} for i in batch})
            else:
                parsed.update({r['path']: r for r in json.loads(result.stdout)})
    for i in inputs:
        path, source, language = i['path'], i['source'], i['language']
        if os.environ.get('CHAINBENCH_AST_PROGRESS') == '1':
            print('parsing ' + language + ' ' + path, file=sys.stderr, flush=True)
        try:
            r = parsed.get(path)
            if r is None:
                r = (python_nodes(source) if language == 'python' else
                     shell_nodes(source) if language == 'bash' else
                     make_nodes(source) if language == 'make' else
                     workflow_nodes(source) if path.startswith('.github/workflows/') and language == 'yaml' else
                     json_nodes(source, path.startswith('internal/chains/') and path.endswith('/genesis.json')) if language == 'json' else
                     syntax_nodes(language, source))
            if r.get('error'):
                raise ValueError(r['error'])
            # Embedded HTML in Go templates and script/style in HTML are parsed
            # too. Placeholder normalization is never implicit or silent.
            embedded = list(r.get('embedded', []))
            for n in r['nodes']:
                if n['kind'] == 'embedded-html':
                    embedded.append({'language': 'html', 'source': n['name'], 'line': n['line']})
            embedded_results = []
            for e in embedded:
                if e['language'] == 'html':
                    er = syntax_nodes('html', e['source'])
                    embedded.extend(er.get('embedded', []))
                elif e['language'] in {'bash', 'python'}:
                    er = shell_nodes(e['source']) if e['language'] == 'bash' else python_nodes(e['source'])
                else:
                    run = subprocess.run(['node', str(REPO / 'scripts/inventory/frontend_ast.mjs')], input=json.dumps([{'path': path, **e}]), capture_output=True, text=True, check=True)
                    er = json.loads(run.stdout)[0]
                    if er.get('error'):
                        raise ValueError('embedded ' + e['language'] + ': ' + er['error'])
                embedded_results.append(er)
                if e['language'] != 'html':
                    embedded.extend(er.get('embedded', []))
            all_nodes = [n for n in r['nodes'] if n['kind'] != 'embedded-html']
            for e, er in zip(embedded, embedded_results):
                context = {'language': e['language'], 'embeddingLine': e['line'], 'sha256': hashlib.sha256(e['source'].encode()).hexdigest()}
                all_nodes.extend({**n, 'sourceContext': context} for n in er['nodes'])
            entry = {'path': path, 'language': language, 'area': area(path), 'sha256': hashlib.sha256(source.encode()).hexdigest(),
                     'astNodes': r['astNodes'], 'embeddedASTs': len(embedded_results), 'generated': path.startswith('internal/dashboard/spa/'), 'test': path.endswith('_test.go') or '/tests/' in '/' + path, 'transformations': r.get('transformations', []) + [{'kind': 'embedded-shell-expansions', 'line': e['line'], 'substitutions': e['substitutions']} for e in embedded if e.get('substitutions')]}
            entry['embeddedPrograms'] = [{'language': e['language'], 'line': e['line'], 'sha256': hashlib.sha256(e['source'].encode()).hexdigest()} for e in embedded]
            graph['files'].append(entry)
            graph['nodes'].append({'id': path, 'kind': 'file', 'name': path, 'area': entry['area']})
            for index, n in enumerate(all_nodes):
                ident = path + '#' + str(index)
                graph['nodes'].append({'id': ident, 'file': path, **n})
                graph['edges'].append({'from': path, 'to': ident, 'kind': 'contains'})
                if n['kind'] == 'import':
                    target = n['name']
                    if target.startswith('.') and language in {'javascript', 'svelte'}:
                        target = os.path.normpath(str(Path(path).parent / target))
                    graph['edges'].append({'from': path, 'to': target, 'kind': 'imports'})
        except (ValueError, SyntaxError, ImportError, KeyError) as error:
            graph['missing'].append({'path': path, 'reason': str(error)})
    identifiers = {n['id'] for n in graph['nodes']}
    for f in graph['files']:
        if f['language'] == 'go':
            package = 'package:' + str(Path(f['path']).parent)
            if package not in identifiers:
                graph['nodes'].append({'id': package, 'kind': 'package', 'name': str(Path(f['path']).parent)})
                identifiers.add(package)
            graph['edges'].append({'from': package, 'to': f['path'], 'kind': 'contains'})
    for e in graph['edges']:
        if e['kind'] != 'imports':
            continue
        target = e['to']
        if target.startswith('github.com/0xmhha/chainbench/'):
            target = 'package:' + target.removeprefix('github.com/0xmhha/chainbench/')
        elif target not in identifiers:
            # External and dynamically resolved imports are explicit terminal
            # nodes; the graph does not invent a source binding for them.
            target = 'dependency:' + target
        e['to'] = target
        if target not in identifiers:
            graph['nodes'].append({'id': target, 'kind': 'dependency', 'name': target})
            identifiers.add(target)
    graph['counts'] = {'parsedFiles': len(graph['files']), 'languages': dict(Counter(f['language'] for f in graph['files'])), 'nodes': len(graph['nodes']), 'edges': len(graph['edges']), 'excluded': len(graph['excluded']), 'missing': len(graph['missing'])}
    graph['parsers'] = {'go': subprocess.check_output(['go', 'version'], text=True).strip(), 'python': sys.version.split()[0],
                        'tree-sitter-language-pack': version('tree-sitter-language-pack'), 'tree-sitter': version('tree-sitter'),
                        'PyYAML': version('PyYAML'), 'shfmt': subprocess.check_output([os.environ.get('CHAINBENCH_AST_SHFMT', '/private/tmp/chainbench-ast-tools/shfmt'), '-version'], text=True).strip(),
                        'frontend': json.loads(subprocess.check_output(['node', str(REPO / 'scripts/inventory/frontend_ast.mjs'), '--versions'], text=True))}
    return graph


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--root', type=Path, default=REPO)
    parser.add_argument('--all-files', action='store_true', help='include all files in synthetic fixture roots; normal repository mode uses Git owned/untracked source')
    args = parser.parse_args()
    graph = build(args.root.resolve(), args.all_files)
    json.dump(graph, sys.stdout, indent=2)
    print()
    return int(bool(graph['missing']))


if __name__ == '__main__':
    sys.exit(main())
