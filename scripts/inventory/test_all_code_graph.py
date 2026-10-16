"""Observable inventory contracts: every source, test symbols, and refusals."""
import json
from pathlib import Path
import subprocess
import tempfile
import unittest

REPO = Path(__file__).resolve().parents[2]
TOOL = REPO / 'scripts/inventory/all_code_graph.py'


class AllCodeGraphTest(unittest.TestCase):
    def run_graph(self, root):
        # Exercise the existing Go-only tool as the RED baseline. Once the
        # complete inventory exists, exercise its public CLI with the same input.
        command = ['python3', '-B', str(TOOL), '--root', str(root), '--all-files']
        if not TOOL.exists():
            command = ['go', 'run', './scripts/inventory/code-graph', '-symbols', str(root)]
        return subprocess.run(command, cwd=REPO, capture_output=True, text=True, check=False)

    def fixture(self, root, path, content):
        target = root / path
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(content)

    def test_every_language_and_build_tagged_test_has_graph_nodes(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            sources = {
                'internal/app/demo.go': 'package app\nfunc Hello() {}\n',
                'internal/app/demo_test.go': '//go:build e2e\npackage app\nfunc TestHidden() { Hello() }\n',
                'scripts/check.py': 'def verify():\n    print("ok")\n',
                'scripts/check.sh': '#!/bin/bash\nverify() { echo ok; }\nverify\n',
                'web/src/App.svelte': '<script>import View from "./View.svelte"; let count=0;</script><button onclick={()=>count++}>{count}</button>',
                'web/src/View.svelte': '<p>Ready</p>',
                'web/src/entry.js': 'import View from "./View.svelte"; export function show() { console.log(View); }',
                'web/index.html': '<html><body><script>console.log("ok")</script></body></html>',
                'web/main.css': 'button { color: red; }',
                'Makefile': 'all:\n\techo ok\n',
                'env/Dockerfile': 'FROM alpine\nRUN echo ok\n',
                'tests/demo.json': '{"schemaVersion":"2","kind":"case","id":"sample","steps":[{"do":"waitBlock","target":2}]}',
                'tests/fixture.go.txt': 'package main\nfunc main() {}\n',
            }
            for path, content in sources.items():
                self.fixture(root, path, content)
            result = self.run_graph(root)
            self.assertEqual(result.returncode, 0, result.stderr)
            graph = json.loads(result.stdout)
            self.assertEqual({f['path'] for f in graph.get('files', [])}, set(sources))
            self.assertEqual(graph['missing'], [])
            identifiers = {n['id'] for n in graph['nodes']}
            self.assertTrue(all(e['from'] in identifiers and e['to'] in identifiers for e in graph['edges']))
            self.assertTrue(any(n.get('name') == 'TestHidden' for n in graph['nodes']))
            self.assertTrue(any(e['kind'] == 'imports' and e['to'] == 'web/src/View.svelte' for e in graph['edges']))
            self.assertTrue(any(n.get('name') == 'waitBlock' and n['kind'] == 'dsl-operation' for n in graph['nodes']))

    def test_invalid_syntax_and_unknown_code_cannot_be_reported_complete(self):
        for path, source in [('bad.go', 'package main\nfunc broken( {'), ('bad.js', 'function broken( {'), ('bad.css', 'a { color: red;'), ('bad.py', 'def broken(:\n'), ('bad.sh', 'if then\n'), ('bad.svelte', '<p>{#if true}'), ('new.rs', 'fn main() {}')]:
            with self.subTest(path=path), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                self.fixture(root, path, source)
                result = self.run_graph(root)
                self.assertNotEqual(result.returncode, 0)
                graph = json.loads(result.stdout)
                self.assertIn(path, [failure['path'] for failure in graph['missing']])

    def test_embedded_workflow_and_build_commands_are_not_opaque(self):
        cases = [('Makefile', 'all:\n\tif then\n'),
                 ('.github/workflows/check.yml', 'jobs:\n  build:\n    steps:\n      - run: if then\n'),
                 ('Dockerfile', 'FROM alpine\nRUN if then\n'),
                 ('inline.sh', "python3 - <<'PY'\ndef broken(:\nPY\n")]
        for path, source in cases:
            with self.subTest(path=path), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                self.fixture(root, path, source)
                result = self.run_graph(root)
                self.assertNotEqual(result.returncode, 0)
                graph = json.loads(result.stdout)
                self.assertIn(path, [f['path'] for f in graph['missing']])

    def test_dependencies_are_explicitly_excluded_and_symlinks_are_refused(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            self.fixture(root, 'node_modules/vendor.js', 'not JavaScript')
            (root / 'node_modules/vendor.bin').write_bytes(b'\xff')
            self.fixture(root, 'README.md', '# Documentation')
            self.fixture(root, 'source.py', 'print("safe")\n')
            (root / 'linked.py').symlink_to(root / 'source.py')
            result = self.run_graph(root)
            self.assertNotEqual(result.returncode, 0)
            graph = json.loads(result.stdout)
            self.assertTrue(any(f['path'] == 'node_modules/vendor.js' and f['reason'] == 'external dependency' for f in graph['excluded']))
            self.assertTrue(any(f['path'] == 'node_modules/vendor.bin' and f['reason'] == 'external dependency' for f in graph['excluded']))
            self.assertTrue(any(f['path'] == 'linked.py' and f['reason'] == 'symlink' for f in graph['missing']))

    def test_executed_inline_programs_are_parsed_in_their_own_language(self):
        cases = [
            ('inline.go', 'package main\nimport "os/exec"\nfunc f() { exec.Command("python3", "-B", "-c", `def broken(:`) }\n'),
            ('context.go', 'package main\nimport "os/exec"\nconst script = `if then`\nfunc f() { exec.CommandContext(nil, "sh", "-c", script) }\n'),
            ('inline.py', 'import subprocess\nsubprocess.run(["node", "-e", "function broken( {"])\n'),
            ('inline.sh', "#!/bin/bash\npython3 -c 'def broken(:'\n"),
        ]
        for path, source in cases:
            with self.subTest(path=path), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                self.fixture(root, path, source)
                result = self.run_graph(root)
                self.assertNotEqual(result.returncode, 0, 'executed source was opaque: ' + path)
                graph = json.loads(result.stdout)
                self.assertIn(path, [failure['path'] for failure in graph['missing']])

    def test_inline_constant_programs_expose_inner_calls(self):
        cases = [
            ('inline.go', 'package main\nimport "os/exec"\nconst script = `import json; json.dumps({})`\nfunc f() { exec.CommandContext(nil, "python3", "-B", "-c", script) }\n', 'json.dumps'),
            ('inline.py', 'import subprocess,sys\nscript = "import json; json.dumps({})"\nsubprocess.run([sys.executable,"-c",script])\n', 'json.dumps'),
            ('inline.sh', "#!/bin/bash\nnode -e 'JSON.stringify({})'\n", 'JSON.stringify'),
        ]
        for path, source, inner in cases:
            with self.subTest(path=path), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                self.fixture(root, path, source)
                result = self.run_graph(root)
                self.assertEqual(result.returncode, 0, result.stderr)
                graph = json.loads(result.stdout)
                self.assertGreater(graph['files'][0]['embeddedASTs'], 0)
                self.assertTrue(any(n.get('kind') == 'call' and n.get('name') == inner for n in graph['nodes']))

    def test_dynamic_payloads_are_marked_without_parsing_stale_bindings(self):
        sources = {
            'dynamic.go': 'package main\nimport "os/exec"\nfunc f(script string) { exec.Command("python3", "-c", script) }\n',
            'reassigned.go': 'package main\nimport "os/exec"\nfunc f() { script := `def broken(:`; script = input(); exec.Command("python3", "-c", script) }\n',
            'dynamic.py': 'import subprocess\ndef f(script):\n    subprocess.run(["python3", "-c", script])\n',
            'reassigned.py': 'import subprocess\nscript = "def broken(:"\nscript = input()\nsubprocess.run(["python3", "-c", script])\n',
            'dynamic.sh': '#!/bin/bash\npython3 -c "$script"\n',
        }
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for path, source in sources.items():
                self.fixture(root, path, source)
            result = self.run_graph(root)
            self.assertEqual(result.returncode, 0, result.stderr)
            graph = json.loads(result.stdout)
            dynamic = [n for n in graph['nodes'] if n['kind'] == 'dynamic-program-input']
            self.assertEqual({n['file'] for n in dynamic}, set(sources))
            self.assertEqual(len(dynamic), len(sources))
            self.assertTrue(all(f['embeddedASTs'] == 0 for f in graph['files']))

    def test_shadowed_imports_and_script_arguments_are_not_child_programs(self):
        sources = {
            'shadow.py': 'import subprocess\ndef f(subprocess):\n    subprocess.run(["python3", "-c", "def broken(:"])\n',
            'lambda.py': 'import subprocess\nf = lambda subprocess: subprocess.run(["python3", "-c", "def broken(:"])\n',
            'comprehension.py': 'import subprocess\n[subprocess.run(["python3", "-c", "def broken(:"]) for subprocess in []]\n',
            'shell-list.py': 'import subprocess\nsubprocess.run(["python3", "-c", "def broken(:"], shell=True)\n',
            'script.py': 'import subprocess\nsubprocess.run(["python3", "script.py", "-c", "def broken(:"])\n',
            'script.go': 'package main\nimport "os/exec"\nfunc f() { exec.Command("python3", "script.py", "-c", `def broken(:`) }\n',
            'script.sh': "python3 script.py -c 'def broken(:'\n",
            'empty.sh': "python3 script.py '' \"\"\n",
        }
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for path, source in sources.items():
                self.fixture(root, path, source)
            result = self.run_graph(root)
            self.assertEqual(result.returncode, 0, result.stderr + result.stdout[-2000:])
            self.assertFalse(any(n.get('name') == 'broken' for n in json.loads(result.stdout)['nodes']))

    def test_scope_and_shell_quoting_preserve_actual_constant_program(self):
        sources = {
            'scope.py': 'import subprocess\n[subprocess.run(["python3", "-c", "def broken(:"]) for subprocess in []]\nsubprocess.run(["python3", "-c", "import json; json.dumps({})"])\n',
            'alias.py': 'from subprocess import run as launch\nlaunch(args=["python3", "-c", "import json; json.dumps({})"])\n',
            'quoted.sh': 'python3 -c "import json; json.dumps({\\\"ok\\\": 1})"\n',
        }
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for path, source in sources.items():
                self.fixture(root, path, source)
            result = self.run_graph(root)
            self.assertEqual(result.returncode, 0, result.stderr)
            graph = json.loads(result.stdout)
            inner = [n for n in graph['nodes'] if n.get('name') == 'json.dumps']
            self.assertEqual({n['file'] for n in inner}, set(sources))
            self.assertTrue(all(n.get('sourceContext', {}).get('language') == 'python' for n in inner))

    def test_heredoc_data_is_not_misclassified_as_interpreter_source(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            self.fixture(root, 'data.sh', "python3 script.py <<'DATA'\ndef broken(:\nDATA\npython3 -c 'print(1)' <<'DATA'\ndef broken(:\nDATA\npython3 -B - arg <<'PY'\nimport json; json.dumps({})\nPY\n")
            result = self.run_graph(root)
            self.assertEqual(result.returncode, 0, result.stderr)
            graph = json.loads(result.stdout)
            self.assertEqual(graph['files'][0]['embeddedASTs'], 2)
            self.assertTrue(any(n.get('name') == 'json.dumps' for n in graph['nodes']))


    def test_explicit_command_takes_precedence_over_shell_stdin_mode(self):
        source = "sh -s -c 'printf ok' <<'DATA'\nif then\nDATA\n"
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory); self.fixture(root, 'data.sh', source)
            result = self.run_graph(root)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(json.loads(result.stdout)['files'][0]['embeddedASTs'], 1)

    def test_python_docker_stdin_and_nested_xargs_are_parsed(self):
        sources = {
            'direct.py': 'import subprocess\nsubprocess.run(["sh", "-s", "--", "arg"], input=b"printf direct_ok")\n',
            'docker.py': 'import subprocess\nSCRIPT = "printf docker_ok"\ndef launch(container, arguments):\n    subprocess.run(["docker", "exec", "-i", "-u", "root", container, "/bin/sh", "-s", "--", *arguments], input=SCRIPT.encode())\n',
            'nested.py': 'import subprocess\nSCRIPT = "xargs -0 -r /bin/sh -c \\"printf nested_ok\\" -- /data"\nsubprocess.run(["sh", "-s"], input=SCRIPT.encode("utf-8"))\n',
        }
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for path, source in sources.items(): self.fixture(root, path, source)
            result = self.run_graph(root)
            self.assertEqual(result.returncode, 0, result.stderr)
            graph = json.loads(result.stdout)
            self.assertTrue(all(f['embeddedASTs'] >= 1 for f in graph['files']))
            self.assertGreaterEqual(next(f['embeddedASTs'] for f in graph['files'] if f['path'] == 'nested.py'), 2)
            self.assertEqual({n['file'] for n in graph['nodes'] if n.get('kind') == 'call' and n.get('name') == 'printf'}, set(sources))

    def test_malformed_stdin_and_nested_shell_cannot_pass_inventory(self):
        sources = {
            'stdin.py': 'import subprocess\nsubprocess.run(["python3", "-"], input=b"def broken(:")\n',
            'docker.py': 'import subprocess\nsubprocess.run(["docker", "exec", "--interactive", "owned", "sh", "-s"], input="if then".encode())\n',
            'nested.sh': 'xargs -0 -r /bin/sh -c "if then" -- /data\n',
        }
        for path, source in sources.items():
            with self.subTest(path=path), tempfile.TemporaryDirectory() as directory:
                root = Path(directory); self.fixture(root, path, source)
                result = self.run_graph(root)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn(path, [r['path'] for r in json.loads(result.stdout)['missing']])

    def test_stdin_data_and_noninteractive_docker_are_not_programs(self):
        source = 'import subprocess\nsubprocess.run(["docker", "exec", "owned", "sh", "-s"], input=b"if then")\nsubprocess.run(["docker", "exec", "-i", "owned", "python3", "script.py"], input=b"def broken(:")\nsubprocess.run(["sh", "-c", "printf ok"], input=b"if then")\n'
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory); self.fixture(root, 'data.py', source)
            result = self.run_graph(root)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(json.loads(result.stdout)['files'][0]['embeddedASTs'], 1)

    def test_dynamic_stdin_and_unsupported_wrapper_options_remain_explicit(self):
        sources = {
            'dynamic.py': 'import subprocess\ndef f(script):\n    subprocess.run(["docker", "exec", "-i", "owned", "sh", "-s"], input=script.encode())\n',
            'rebound.py': 'import subprocess\nscript = "if then"\nscript = input()\nsubprocess.run(["sh", "-s"], input=script.encode())\n',
            'encoding.py': 'import subprocess\nsubprocess.run(["sh", "-s"], input="printf ok".encode(input()))\n',
            'replace.sh': 'xargs -I{} sh -c "printf {}"\n',
            'option.py': 'import subprocess\ndef launch(option):\n    subprocess.run(["docker", "exec", option, "owned", "sh", "-s"], input=b"if then")\n',
        }
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for path, source in sources.items(): self.fixture(root, path, source)
            result = self.run_graph(root)
            self.assertEqual(result.returncode, 0, result.stderr)
            graph = json.loads(result.stdout)
            self.assertEqual({n['file'] for n in graph['nodes'] if n['kind'] == 'dynamic-program-input'}, set(sources))
            self.assertTrue(all(f['embeddedASTs'] == 0 for f in graph['files']))


if __name__ == '__main__':
    unittest.main()
