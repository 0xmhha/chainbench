"""Conservative lexical extraction of statically known subprocess programs.

This never executes source. Reassigned bindings, parameters and expressions
outside the supported constant subset are recorded as dynamic inputs.
"""
import ast
from pathlib import Path
import re


def interpreter(program):
    name = Path(program).name if isinstance(program, str) else ''
    if re.fullmatch(r'python(?:3(?:\.\d+)?)?', name):
        return 'python'
    if name in {'bash', 'sh'}:
        return 'bash'
    if name in {'node', 'nodejs'}:
        return 'javascript'
    return None


def command_target(arguments):
    """Unwrap only understood Docker exec and xargs option grammars.

    Return argv, whether standard input reaches it, and an unresolved language.
    Unknown options remain dynamic rather than being mistaken for operands.
    """
    if not isinstance(arguments, (list, tuple)) or not arguments:
        return [], False, None
    name = Path(arguments[0]).name if isinstance(arguments[0], str) else ''
    if name not in {'docker', 'xargs'}:
        return arguments, True, None
    fallback = next((interpreter(a) for a in arguments[1:] if interpreter(a)), None)
    i, forwards = 1, False
    if name == 'docker':
        if len(arguments) < 2 or arguments[1] != 'exec':
            return [], False, None
        i = 2
        values = {'-u', '--user', '-w', '--workdir', '-e', '--env', '--env-file', '--detach-keys'}
        while i < len(arguments):
            flag = arguments[i]
            if flag is None or (isinstance(flag, str) and not flag.startswith('-')):
                # A dynamic container is safe to skip only when the command
                # boundary itself is known. Otherwise this may be an option.
                if flag is None and (i + 1 >= len(arguments) or not interpreter(arguments[i + 1])):
                    return [], False, fallback
                return arguments[i + 1:], forwards, None
            if flag == '--':
                return arguments[i + 2:], forwards, None
            if flag in {'-i', '--interactive', '-it', '-ti'}:
                forwards = True
            elif flag in {'-t', '--tty', '--privileged'}:
                pass
            elif flag in values:
                i += 1
                if i >= len(arguments):
                    return [], False, fallback
            elif isinstance(flag, str) and '=' in flag and flag.split('=', 1)[0] in values:
                pass
            else:
                return [], False, fallback
            i += 1
    else:
        while i < len(arguments):
            flag = arguments[i]
            if isinstance(flag, str) and not flag.startswith('-'):
                return arguments[i:], False, None
            if flag == '--':
                return arguments[i + 1:], False, None
            if flag not in {'-0', '--null', '-r', '--no-run-if-empty'}:
                # In particular -I can substitute runtime text into the program.
                return [], False, fallback
            i += 1
    return [], False, fallback


def stdin_program(arguments):
    # An explicit -c/-e program takes precedence even if -s appeared first.
    if inline_program(arguments)[0] is not None:
        return None
    arguments, forwards, unresolved = command_target(arguments)
    if unresolved or not forwards or not arguments:
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


def source_text(value):
    if isinstance(value, bytes):
        try:
            return value.decode('utf-8')
        except UnicodeDecodeError:
            return None
    return value if isinstance(value, str) else None


def inline_program(arguments):
    """Return language/body, or None body when a known payload is dynamic."""
    arguments, _, unresolved = command_target(arguments)
    if unresolved:
        return unresolved, None
    if not arguments:
        return None, None
    language = interpreter(arguments[0])
    if language is None:
        return None, None
    for i, flag in enumerate(arguments[1:], 1):
        if not isinstance(flag, str):
            return language, None  # unknown flags may select an inline program
        if flag == '--' or not flag.startswith('-'):
            return None, None  # remaining operands belong to a script file
        evaluate = ((language == 'python' and flag == '-c') or
                    (language == 'javascript' and flag in {'-e', '--eval'}) or
                    (language == 'bash' and not flag.startswith('--') and 'c' in flag))
        if evaluate:
            body = arguments[i + 1] if i + 1 < len(arguments) else None
            return language, body if isinstance(body, str) else None
    return None, None


class Scope:
    def __init__(self, parent=None, kind='module'):
        self.parent, self.kind, self.bindings = parent, kind, {}

    def bind(self, name, value):
        self.bindings.setdefault(name, []).append(value)

    def lookup(self, name):
        scope = self
        while scope:
            if name in scope.bindings:
                values = scope.bindings[name]
                return (scope, values[0]) if len(values) == 1 else (scope, None)
            scope = scope.parent
        return None, None


class Programs(ast.NodeVisitor):
    def __init__(self):
        self.scope = Scope()
        self.calls = []

    def visit_Name(self, node):
        if isinstance(node.ctx, (ast.Store, ast.Del)):
            self.scope.bind(node.id, None)

    def visit_Import(self, node):
        for alias in node.names:
            self.scope.bind(alias.asname or alias.name.split('.')[0], ('import', alias.name))

    def visit_ImportFrom(self, node):
        for alias in node.names:
            self.scope.bind(alias.asname or alias.name, ('import', (node.module or '') + '.' + alias.name) if not node.level else None)

    def visit_Assign(self, node):
        for target in node.targets:
            if isinstance(target, ast.Name):
                self.scope.bind(target.id, node.value)
            else:
                self.visit(target)
        self.visit(node.value)

    def visit_AnnAssign(self, node):
        if isinstance(node.target, ast.Name):
            self.scope.bind(node.target.id, node.value)
        else:
            self.visit(node.target)
        if node.value:
            self.visit(node.value)

    def visit_Global(self, node):
        # Cross-scope mutation is deliberately unresolved, including its owner.
        for name in node.names:
            scope = self.scope
            while scope:
                scope.bind(name, None)
                scope = scope.parent

    visit_Nonlocal = visit_Global

    def function(self, node):
        self.scope.bind(node.name, None)
        for value in node.decorator_list + node.args.defaults + [v for v in node.args.kw_defaults if v]:
            self.visit(value)
        parent = self.scope
        enclosing = parent.parent if parent.kind == 'class' else parent
        self.scope = Scope(enclosing, 'function')
        for argument in ast.walk(node.args):
            if isinstance(argument, ast.arg):
                self.scope.bind(argument.arg, None)
        for statement in node.body:
            self.visit(statement)
        self.scope = parent

    visit_FunctionDef = function
    visit_AsyncFunctionDef = function

    def visit_Lambda(self, node):
        for value in node.args.defaults + [v for v in node.args.kw_defaults if v]:
            self.visit(value)
        parent = self.scope
        self.scope = Scope(parent.parent if parent.kind == 'class' else parent, 'function')
        for argument in ast.walk(node.args):
            if isinstance(argument, ast.arg):
                self.scope.bind(argument.arg, None)
        self.visit(node.body)
        self.scope = parent

    def comprehension(self, node):
        parent = self.scope
        self.visit(node.generators[0].iter)
        self.scope = Scope(parent.parent if parent.kind == 'class' else parent, 'comprehension')
        for index, generator in enumerate(node.generators):
            if index:
                self.visit(generator.iter)
            self.visit(generator.target)
            for condition in generator.ifs:
                self.visit(condition)
        if isinstance(node, ast.DictComp):
            self.visit(node.key)
            self.visit(node.value)
        else:
            self.visit(node.elt)
        self.scope = parent

    visit_ListComp = comprehension
    visit_SetComp = comprehension
    visit_DictComp = comprehension
    visit_GeneratorExp = comprehension

    def visit_ExceptHandler(self, node):
        if node.name:
            self.scope.bind(node.name, None)
        self.generic_visit(node)

    def visit_MatchAs(self, node):
        if node.name:
            self.scope.bind(node.name, None)
        self.generic_visit(node)

    visit_MatchStar = visit_MatchAs

    def visit_MatchMapping(self, node):
        if node.rest:
            self.scope.bind(node.rest, None)
        self.generic_visit(node)

    def visit_ClassDef(self, node):
        self.scope.bind(node.name, None)
        for value in node.bases + node.decorator_list:
            self.visit(value)
        parent = self.scope
        self.scope = Scope(parent, 'class')
        for statement in node.body:
            self.visit(statement)
        self.scope = parent

    def visit_Call(self, node):
        self.calls.append((node, self.scope))
        self.generic_visit(node)

    def resolve(self, node, scope, seen=()):
        if isinstance(node, ast.Constant):
            return node.value if isinstance(node.value, (str, bytes)) else None
        if isinstance(node, ast.Name):
            owner, binding = scope.lookup(node.id)
            key = (owner, node.id)
            if owner is None or key in seen:
                return None
            if isinstance(binding, tuple):
                return binding
            return self.resolve(binding, owner, seen + (key,))
        if isinstance(node, ast.Attribute):
            owner = self.resolve(node.value, scope, seen)
            if owner == ('import', 'sys') and node.attr == 'executable':
                return 'python3'
            if isinstance(owner, tuple) and owner[0] == 'import':
                return ('import', owner[1] + '.' + node.attr)
        if isinstance(node, ast.Call) and isinstance(node.func, ast.Attribute) and node.func.attr == 'encode':
            value = self.resolve(node.func.value, scope, seen)
            if not isinstance(value, str) or node.keywords or len(node.args) > 1:
                return None
            encoding = self.resolve(node.args[0], scope, seen) if node.args else 'utf-8'
            if isinstance(encoding, str) and encoding.lower().replace('_', '-') in {'utf-8', 'utf8'}:
                return value.encode('utf-8')
            return None
        if isinstance(node, (ast.List, ast.Tuple)):
            return [self.resolve(value, scope, seen) for value in node.elts]
        if isinstance(node, ast.BinOp) and isinstance(node.op, ast.Add):
            left, right = self.resolve(node.left, scope, seen), self.resolve(node.right, scope, seen)
            if isinstance(left, str) and isinstance(right, str):
                return left + right
        return None


def extract(tree):
    programs = Programs()
    programs.visit(tree)
    embedded, dynamic = [], []
    methods = {'subprocess.' + method for method in ('run', 'Popen', 'call', 'check_call', 'check_output')}
    for call, scope in programs.calls:
        function = programs.resolve(call.func, scope)
        if not isinstance(function, tuple) or function[0] != 'import' or function[1] not in methods:
            continue
        argument = call.args[0] if call.args else next((k.value for k in call.keywords if k.arg == 'args'), None)
        arguments = programs.resolve(argument, scope)
        shell = next((k.value for k in call.keywords if k.arg == 'shell'), None)
        if isinstance(shell, ast.Constant) and shell.value is True:
            # POSIX shell=True passes only argv[0] to sh -c; later elements
            # become shell positional parameters, not interpreter flags.
            body = arguments[0] if isinstance(arguments, list) and arguments else arguments
            language, body = 'bash', body if isinstance(body, str) else None
        elif shell is not None and not (isinstance(shell, ast.Constant) and shell.value is False):
            language, body = 'bash', None
        else:
            language, body = inline_program(arguments)
            if language is None:
                language = stdin_program(arguments)
                supplied = next((k.value for k in call.keywords if k.arg == 'input'), None)
                body = source_text(programs.resolve(supplied, scope))
        if language and body is not None:
            embedded.append({'language': language, 'source': body, 'line': call.lineno})
        elif language:
            dynamic.append({'kind': 'dynamic-program-input', 'name': language + ' interpreter payload', 'line': call.lineno})
    return embedded, dynamic
