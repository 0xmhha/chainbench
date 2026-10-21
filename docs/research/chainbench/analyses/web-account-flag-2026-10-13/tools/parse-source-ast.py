"""Inventory all tracked source files and create a syntax-derived file graph.

Use the codemine environment for Go/Python grammars and --grammar-dir for
the two supplemental grammars. Svelte/HTML/CSS use the complete compiler AST
written by parse-web-ast.cjs. Source literals are deliberately not dumped.
"""

from __future__ import annotations

import argparse
import hashlib
import importlib
import json
import shutil
import subprocess
import sys
from collections import Counter
from pathlib import Path
from typing import NotRequired, TypedDict, cast

from tree_sitter import Language, Node, Parser

GRAMMARS: dict[str, str] = {
    ".go": "tree_sitter_go",
    ".py": "tree_sitter_python",
    ".js": "tree_sitter_javascript",
    ".mjs": "tree_sitter_javascript",
    ".sh": "tree_sitter_bash",
}
DECLARATIONS: frozenset[str] = frozenset(
    {
        "function_declaration",
        "method_declaration",
        "type_spec",
        "function_definition",
        "class_definition",
        "class_declaration",
        "method_definition",
        "variable_declarator",
    }
)
CALLS: frozenset[str] = frozenset(
    {"call_expression", "call", "new_expression", "command"}
)


class Reference(TypedDict):
    """Represent an identifier or import and its source location."""

    target: str
    line: int


class Symbol(TypedDict):
    """Identify a declared symbol."""

    name: str
    kind: str
    line: int


class ParseError(TypedDict, total=False):
    """Record a parser failure without source literals."""

    kind: str
    code: str
    line: int | None


class AstRecord(TypedDict):
    """Contain the common syntax inventory for every parser."""

    astNodes: int
    imports: list[Reference]
    symbols: list[Symbol]
    calls: list[Reference]
    errors: list[ParseError]


class FileRecord(AstRecord):
    """Attach syntax data to a content-addressed tracked file."""

    file: str
    language: str
    sha256: str
    lines: int
    components: NotRequired[list[Reference]]
    guards: NotRequired[list[dict[str, str | int]]]


class WebGraph(TypedDict):
    """Represent the compiler-generated JSON at the input boundary."""

    compiler: str
    files: list[FileRecord]


def load_web_graph(path: Path) -> WebGraph:
    """Validate the compiler JSON boundary before using typed records."""
    decoded: object = json.loads(path.read_text())
    if not isinstance(decoded, dict) or not isinstance(decoded.get("compiler"), str):
        raise ValueError("Invalid compiler metadata")
    records = decoded.get("files")
    if not isinstance(records, list):
        raise ValueError("Invalid compiler file inventory")
    for record in records:
        if not isinstance(record, dict):
            raise ValueError("Invalid compiler file record")
        for key in ("file", "language"):
            if not isinstance(record.get(key), str):
                raise ValueError(f"Invalid compiler field: {key}")
        for key in ("imports", "calls", "symbols", "errors"):
            if not isinstance(record.get(key), list):
                raise ValueError(f"Invalid compiler field: {key}")
        for key in ("lines", "astNodes"):
            if not isinstance(record.get(key), int):
                raise ValueError(f"Invalid compiler field: {key}")
        for reference in [*record["imports"], *record["calls"]]:
            if not isinstance(reference, dict):
                raise ValueError("Invalid compiler reference")
            if not isinstance(reference.get("target"), str):
                raise ValueError("Invalid compiler reference target")
            if not isinstance(reference.get("line"), int):
                raise ValueError("Invalid compiler reference location")
    # The compiler omits sha256; main adds it before publishing FileRecords.
    return cast(WebGraph, decoded)


def node_text(node: Node, source: bytes) -> str:
    """Decode one syntax node without retaining the rest of the source."""
    return source[node.start_byte : node.end_byte].decode("utf-8")


def parse_source(path: Path, parser: Parser) -> AstRecord:
    """Extract declarations, imports and syntactic calls from one complete AST."""
    source = path.read_bytes()
    root = parser.parse(source).root_node
    stack = [root]
    nodes = 0
    imports: list[Reference] = []
    symbols: list[Symbol] = []
    calls: list[Reference] = []
    errors: list[ParseError] = []
    while stack:
        node = stack.pop()
        nodes += 1
        line = node.start_point[0] + 1
        if node.is_error or node.is_missing:
            errors.append({"kind": node.type, "line": line})
        if node.type in DECLARATIONS:
            name = node.child_by_field_name("name")
            if name is not None:
                symbols.append(
                    {"name": node_text(name, source), "kind": node.type, "line": line}
                )
        if node.type in CALLS:
            callee = (
                node.child_by_field_name("function")
                or node.child_by_field_name("name")
                or node.child_by_field_name("constructor")
            )
            if callee is not None and callee.type not in {
                "string",
                "interpreted_string_literal",
                "template_string",
            }:
                target = node_text(callee, source)
                # Dynamic call expressions are not resolved as declarations.
                if len(target) < 160 and all(
                    c.isalnum() or c in "_$.*?[]()" for c in target
                ):
                    calls.append({"target": target, "line": line})
        if node.type in {"import_spec", "import_statement", "import_from_statement"}:
            if path.suffix == ".go":
                import_node = node.child_by_field_name("path")
                if import_node is not None:
                    imports.append(
                        {
                            "target": node_text(import_node, source).strip('"'),
                            "line": line,
                        }
                    )
            elif path.suffix in {".js", ".mjs"}:
                import_node = node.child_by_field_name("source")
                if import_node is not None:
                    imports.append(
                        {
                            "target": node_text(import_node, source).strip("\"'"),
                            "line": line,
                        }
                    )
            else:
                targets = [
                    child
                    for child in node.named_children
                    if child.type in {"dotted_name", "aliased_import"}
                ]
                module = node.child_by_field_name("module_name")
                if module is not None:
                    targets = [module]
                for import_node in targets:
                    import_name = import_node.child_by_field_name("name") or import_node
                    imports.append(
                        {"target": node_text(import_name, source), "line": line}
                    )
        stack.extend(reversed(node.children))
    return {
        "astNodes": nodes,
        "imports": imports,
        "symbols": symbols,
        "calls": calls,
        "errors": errors,
    }


def main() -> int:
    """Create the complete tracked-source inventory and unambiguous import edges."""
    arguments = argparse.ArgumentParser(description=__doc__)
    arguments.add_argument("--repo", type=Path, required=True)
    arguments.add_argument("--grammar-dir", type=Path, required=True)
    arguments.add_argument("--web-ast", type=Path, required=True)
    arguments.add_argument("--out", type=Path, required=True)
    args = arguments.parse_args()
    sys.path.insert(0, str(args.grammar_dir))
    parsers: dict[str, Parser] = {}
    for suffix, module in GRAMMARS.items():
        grammar = importlib.import_module(module)
        parsers[suffix] = Parser(Language(grammar.language()))
    repo = args.repo.resolve()
    git_binary = shutil.which("git")
    if git_binary is None:
        raise FileNotFoundError("git is required to inventory tracked files")
    tracked = (
        subprocess.run(  # noqa: S603 -- resolved git path with fixed read-only arguments
            [git_binary, "ls-files", "-z"],
            cwd=repo,
            check=True,
            capture_output=True,
            timeout=30,
        )
        .stdout.decode()
        .split("\0")
    )
    revision = subprocess.run(  # noqa: S603 -- resolved git path with fixed read-only arguments
        [git_binary, "rev-parse", "HEAD"],
        cwd=repo,
        check=True,
        capture_output=True,
        text=True,
        timeout=30,
    ).stdout.strip()
    web = load_web_graph(args.web_ast)
    web_records = {record["file"]: record for record in web["files"]}
    files: list[FileRecord] = []
    edges: list[dict[str, object]] = []
    coverage: Counter[str] = Counter()
    tracked_paths = set(tracked)
    for relative in sorted(tracked_paths):
        suffix = Path(relative).suffix
        if suffix not in parsers and relative not in web_records:
            continue
        path = repo / relative
        coverage[suffix] += 1
        content = path.read_bytes()
        if relative in web_records:
            record = web_records[relative].copy()
        else:
            syntax = parse_source(path, parsers[suffix])
            record = FileRecord(
                astNodes=syntax["astNodes"],
                imports=syntax["imports"],
                symbols=syntax["symbols"],
                calls=syntax["calls"],
                errors=syntax["errors"],
                file=relative,
                language=suffix.removeprefix("."),
                sha256="",
                lines=len(content.splitlines()),
            )
        record["sha256"] = hashlib.sha256(content).hexdigest()
        files.append(record)
        for imported in record["imports"]:
            target = imported["target"]
            candidates: list[str] = []
            if suffix == ".go" and target.startswith("github.com/0xmhha/chainbench/"):
                module = target.removeprefix("github.com/0xmhha/chainbench/")
                edges.append(
                    {
                        "from": relative,
                        "to": module,
                        "kind": "go-package-import",
                        "line": imported["line"],
                    }
                )
            elif target.startswith("."):
                resolved = (path.parent / target).resolve().relative_to(repo).as_posix()
                candidates = [
                    resolved,
                    *(resolved + ext for ext in (".js", ".mjs", ".svelte", ".py")),
                ]
            elif suffix == ".py":
                resolved = target.replace(".", "/") + ".py"
                candidates = [
                    resolved,
                    (path.parent / resolved).relative_to(repo).as_posix(),
                ]
            match = next(
                (candidate for candidate in candidates if candidate in tracked_paths),
                None,
            )
            if match:
                edges.append(
                    {
                        "from": relative,
                        "to": match,
                        "kind": "file-import",
                        "line": imported["line"],
                    }
                )
    graph = {
        "repo": ".",
        "commit": revision,
        "svelteCompiler": web["compiler"],
        "coverage": dict(sorted(coverage.items())),
        "fileCount": len(files),
        "parseErrorFiles": [record["file"] for record in files if record["errors"]],
        "files": files,
        "edges": edges,
    }
    args.out.write_text(json.dumps(graph, ensure_ascii=False, indent=1) + "\n")
    sys.stdout.write(
        json.dumps(
            {key: graph[key] for key in ("coverage", "fileCount", "parseErrorFiles")}
        )
        + "\n"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
