# Source inventory

`all_code_graph.py` parses owned repository sources, including Go test files and
all build tags, generated SPA files, the Go fixture with a `.go.txt` extension,
JSON DSL/contracts and build configuration. Each file records its source digest
and owning area. Declarations, imports, calls and DSL operations become graph
nodes; every edge has an endpoint. External/dynamically resolved imports remain
explicit dependency nodes. This is a structural syntax graph, not a type-resolved
call graph or evidence of runtime coverage.

The default inventory uses Git's tracked and unignored untracked files. Gitignored
build outputs, caches and dependencies are outside that owned source set.
Dependency directories, documentation, test keys/fuzz data and generated proof
metadata are explicitly classified as exclusions. Unknown files, symlinks and
parser failures remain in `missing` and make the command fail. The `--all-files`
option is for synthetic fixture directories.

Go uses `go/parser`; Python uses `ast`; JavaScript/Svelte/CSS use the project's
Acorn/Svelte/PostCSS installations. Build/HTML/config syntax uses pinned
Tree-sitter grammars. Bash uses mvdan's syntax AST (through shfmt's `-to-json`).
The shell arithmetic supported by mvdan avoids a Tree-sitter Bash limitation
encountered in the real repository. The exact native-crash cause observed with
the initial dependency versions was not established; the pinned runtime and
retained parser objects have passed the full scan.

Make recipes, workflow runs, Docker shell commands, HTML script/style and known
interpreter heredocs are parsed separately. Go os/exec and Python subprocess
calls, and shell commands with known interpreter `-c`/`-e` payloads are also
parsed when their lexical bindings resolve to constant source. Python import
aliases and local scopes are respected; shell standard input is treated as code
only when the interpreter reads it as a program. Data passed to a script file
is not parsed as interpreter source. An explicit `-` selects stdin source before
later dynamic program arguments; option-looking operands after it are data.
Unknown options before the marker remain unresolved. Python subprocess `input=` supports constant
UTF-8 bytes and constant strings encoded with the default/explicit UTF-8 codec.
Direct interpreter stdin, Docker `exec -i` stdin and literal `xargs -0 -r sh -c`
programs are included. Docker exec options are deliberately limited to known
interactive/tty/privileged flags and user/workdir/env values; unsupported options
and xargs substitutions (`-I`) remain explicit dynamic inputs. Without Docker's
interactive flag, subprocess input is not forwarded as the child program.
No container, shell, Python payload or xargs command is executed by this scan. Make variables, workflow expressions,
genesis placeholders and heredoc expansions remain symbolic; their transformations
and source positions/digests are recorded. Source code, make recipes and workflow
commands are never executed. Each extracted program has a language, embedding
line and source hash; its nodes retain a `sourceContext` with that identity.
Their node line numbers refer to the extracted source, not the outer file.
Recognized interpreter invocations with unresolved payloads have explicit
`dynamic-program-input` nodes. This is a conservative lexical subset, not
interprocedural evaluation: arbitrary command wrappers, runtime mutation,
unknown interpreters, option grammars and dynamically assembled programs are
not fully resolved. Zero missing files does not mean zero dynamic programs.

Prepare dependencies outside the repository (macOS example):

```sh
python3 -m pip install --no-cache-dir --target /private/tmp/chainbench-ast-tools -r scripts/inventory/ast-requirements.txt
GOBIN=/private/tmp/chainbench-ast-tools GOMODCACHE=/private/tmp/chainbench-ast-go-modcache GOCACHE=/private/tmp/chainbench-go-cache go install mvdan.cc/sh/v3/cmd/shfmt@v3.12.0
npm ci --prefix web
```

Use the project Go cache and disable Python bytecode files:

```sh
export GOCACHE=/private/tmp/chainbench-go-cache
export PYTHONPATH=/private/tmp/chainbench-ast-tools
export PYTHONDONTWRITEBYTECODE=1
export CHAINBENCH_AST_SHFMT=/private/tmp/chainbench-ast-tools/shfmt
python3 -B scripts/inventory/test_all_code_graph.py
python3 -B scripts/inventory/all_code_graph.py > /private/tmp/chainbench-all-code-graph.json
```

Use `CHAINBENCH_AST_PROGRESS=1` for per-file diagnostics on stderr. Store generated
graphs outside the source inventory. Reproduce the graph with unchanged sources;
compare the file sets and digests, parser provenance, missing list and graph hash.

Parser references: [mvdan AST JSON output](https://github.com/mvdan/sh/blob/master/cmd/shfmt/main.go),
[Tree-sitter Python API](https://github.com/xberg-io/tree-sitter-language-pack/blob/main/docs-site/src/content/docs/reference/api-python.md).

`make tc-review` checks the original 87 WEMIX4 criterion IDs against
`docs/dev/wemix4-tc-sufficiency-review.json`. It verifies criterion/source/evidence
digests, exact JSON assertion anchors and real Go AST declarations, including
methods and gated tests. Missing IDs, hidden conditions, cyclic merge references,
stale sources and scoped proofs labelled as full acceptance fail the check.
The guard also runs through `make check` and CI. When a referenced source changes,
review its actual inputs/assertions and remaining conditions before updating the
snapshot; changing a digest alone is not a semantic reassessment.
`confirmed_scope` means only the explicitly recorded proof scope. This inventory
does not execute chains or award Linux/mainnet, legacy or final Web acceptance.

## Other tools in this directory

| Tool | What it does |
|---|---|
| `source-ast/` | Go program that parses Go sources, including tests and all build tags. `all_code_graph.py` and `tc_sufficiency.py` run it with `go run ./scripts/inventory/source-ast`. |
| `code-graph/` | Go program that extracts the chainbench package import graph and the exported symbols each import uses: `go run ./scripts/inventory/code-graph [module-root] > graph.json`. Its output backs `docs/dev/architecture/code-graph.md`. |
| `surface-graph/` | Go program that prints what each surface (CLI, MCP, DSL) registers and which registrations reach past the app layer: `go run ./scripts/inventory/surface-graph .`. |
| `chain-flag-graph/` | Go program that extracts the command and flag graph of a go-ethereum-derived chain repository from its `cmd/` tree: `go run ./scripts/inventory/chain-flag-graph <repo-root> <binary-dir-name>`, where the second argument names the binary's directory under `cmd/`. |
| `list-adapter-functions.sh`, `scan-binary-hardcoding.sh` | Scanners for the removed Bash implementation. They read `lib/adapters/*.sh` and `lib/cmd_*.sh`, which no longer exist in this repository, so both exit with an error today. |
