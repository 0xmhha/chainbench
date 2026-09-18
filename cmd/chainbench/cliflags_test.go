package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// flagLiteral matches a long flag as a spec writes it, with or without an
// attached value: --bp, --dry-run=false. Shorthands are not checked — a single
// letter is too easy to collide with an ordinary string.
var flagLiteral = regexp.MustCompile(`^--[a-z][a-z0-9.-]*(=.*)?$`)

// TestEveryWrittenFlagIsOneTheCommandHas is the check X10 did not have.
//
// `chain up --validators` was renamed to `--bp`, and the e2e harness kept
// calling the old name. Nothing saw it: the harness is behind `//go:build e2e`
// so `go test ./...` does not even compile it, and `go vet -tags e2e` compiles
// it but has no idea that a string is a flag name. It took a real binary and a
// real network to find, which is the most expensive place to find it.
//
// Source is the right level, because the defect is textual: a name written in
// one place and declared in another. Parsing ignores build tags, so a file the
// ordinary test run never compiles is read here like any other.
//
// The unit of judgement is the COMMAND, not the repository. A sequence of
// string literals whose first words spell a command path is taken to be an
// invocation of that command, and every long flag in it has to be one that
// command accepts (its own, or one it inherits). A sequence that does not start
// with a command path is not this CLI's — the node argv this repository builds
// is full of --datadir and --http.port — and is left alone.
//
// What it does not catch: a flag assembled from pieces, one appended to a slice
// whose command words are somewhere else, and a flag in a shell script or a
// document. It is a net, not a proof. It would have caught X10.
func TestEveryWrittenFlagIsOneTheCommandHas(t *testing.T) {
	root := newRootCmd()
	flags := flagsByPath(root)
	paths := commandPaths(flags)
	groups := groupChildren(root)

	repo := filepath.Join("..", "..")
	fset := token.NewFileSet()
	var offenders []string

	err := filepath.WalkDir(repo, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "vendor", "bin", "build":
				return fs.SkipDir
			case "docs":
				// docs/research holds frozen copies of earlier trees, kept so a
				// past analysis can be re-read. They are supposed to disagree
				// with today's CLI — that is what makes them a record — and
				// reporting them would train the reader to ignore this test.
				// (They do carry the X10 defect, which is one way to see that
				// this check finds it.)
				return fs.SkipDir
			}
			return nil
		}
		rel0, _ := filepath.Rel(repo, path)
		if strings.HasSuffix(path, ".sh") {
			offenders = append(offenders, shellComplaints(path, rel0, paths, flags, groups)...)
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		f, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			// A file this build cannot parse is not this test's finding.
			return nil //nolint:nilerr // parse failures belong to the compiler
		}
		rel, _ := filepath.Rel(repo, path)
		ast.Inspect(f, func(n ast.Node) bool {
			var args []ast.Expr
			switch v := n.(type) {
			case *ast.CompositeLit:
				args = v.Elts
			case *ast.CallExpr:
				args = v.Args
			default:
				return true
			}
			// A long flag is the evidence that this sequence is an argv at all.
			// Without it, every list that happens to start with a word this CLI
			// also uses as a command — []string{"validator", "endpoint"} is a
			// role list, not an invocation — would be judged as one.
			written := litFlags(args)
			if len(written) == 0 {
				return true
			}
			words := leadingWords(args)
			cmd, ok := longestPath(words, paths)
			if !ok {
				return true
			}
			line := fset.Position(n.Pos()).Line
			offenders = append(offenders, complaints(rel, line, cmd, words, written, flags, groups)...)
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(offenders)
	for _, o := range offenders {
		t.Errorf("%s", o)
	}
}

// flagsByPath is every command's flag set, keyed by its path below the root
// ("chain up"), with each command's inherited persistent flags folded in. The
// root itself is the empty path.
func flagsByPath(root *cobra.Command) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	var walk func(c *cobra.Command, path []string)
	walk = func(c *cobra.Command, path []string) {
		own := map[string]bool{}
		c.Flags().VisitAll(func(f *pflag.Flag) { own["--"+f.Name] = true })
		c.InheritedFlags().VisitAll(func(f *pflag.Flag) { own["--"+f.Name] = true })
		// Cobra's own, present on everything.
		own["--help"] = true
		out[strings.Join(path, " ")] = own
		for _, sub := range c.Commands() {
			walk(sub, append(append([]string{}, path...), sub.Name()))
		}
	}
	walk(root, nil)
	return out
}

// commandPaths is the set of command paths, for matching a literal run against.
func commandPaths(flags map[string]map[string]bool) map[string]bool {
	out := make(map[string]bool, len(flags))
	for p := range flags {
		if p != "" {
			out[p] = true
		}
	}
	return out
}

// leadingWords is the run of plain string literals a sequence opens with — the
// command words, before any flag or any value that is a variable.
func leadingWords(args []ast.Expr) []string {
	var words []string
	for _, a := range args {
		s, ok := stringLit(a)
		if !ok || strings.HasPrefix(s, "-") {
			break
		}
		words = append(words, s)
	}
	return words
}

// litFlags is every long flag written as a literal anywhere in a sequence.
func litFlags(args []ast.Expr) []string {
	var out []string
	for _, a := range args {
		if s, ok := stringLit(a); ok && flagLiteral.MatchString(s) {
			out = append(out, s)
		}
	}
	return out
}

// longestPath reads the command path a word run opens with, taking the longest
// match so `node stop` is not read as `node`. It reports false when the run does
// not begin with one, which is how another program's argv is left alone.
func longestPath(words []string, paths map[string]bool) (string, bool) {
	for n := len(words); n > 0; n-- {
		if p := strings.Join(words[:n], " "); paths[p] {
			return p, true
		}
	}
	return "", false
}

// complaints is what one invocation gets wrong: a subcommand a group does not
// have, and flags the command does not accept.
//
// The subcommand half matters as much as the flags. A group like `chain` does
// nothing on its own, so the word after it has to be one of its children;
// `chain down` when the command is `chain stop` fails at run time with the same
// silence a retired flag does.
func complaints(file string, line int, cmd string, words, flags []string, byPath map[string]map[string]bool, groups map[string]map[string]bool) []string {
	var out []string
	if children, isGroup := groups[cmd]; isGroup {
		next := ""
		if n := len(strings.Fields(cmd)); n < len(words) {
			next = words[n]
		}
		if next == "" {
			out = append(out, fmt.Sprintf("%s:%d: `chainbench %s` needs a subcommand", file, line, cmd))
		} else if !children[next] {
			out = append(out, fmt.Sprintf("%s:%d: `chainbench %s` has no subcommand %q", file, line, cmd, next))
		}
		return out
	}
	have := byPath[cmd]
	for _, f := range flags {
		name := f
		if i := strings.IndexByte(name, '='); i >= 0 {
			name = name[:i]
		}
		if !have[name] {
			out = append(out, fmt.Sprintf("%s:%d: `chainbench %s` is given %s, which it does not have", file, line, cmd, name))
		}
	}
	return out
}

// groupChildren is, for every command that does nothing on its own, the names
// of the subcommands it has.
func groupChildren(root *cobra.Command) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	var walk func(c *cobra.Command, path []string)
	walk = func(c *cobra.Command, path []string) {
		if !c.Runnable() && len(c.Commands()) > 0 && len(path) > 0 {
			kids := map[string]bool{}
			for _, sub := range c.Commands() {
				kids[sub.Name()] = true
				for _, a := range sub.Aliases {
					kids[a] = true
				}
			}
			out[strings.Join(path, " ")] = kids
		}
		for _, sub := range c.Commands() {
			walk(sub, append(append([]string{}, path...), sub.Name()))
		}
	}
	walk(root, nil)
	return out
}

// stringLit unquotes a string literal expression.
func stringLit(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return s, true
}

// cliInvocation matches how a shell script names this program: by itself, by a
// path, or through `go run ./cmd/chainbench`.
var cliInvocation = regexp.MustCompile(`(?:^|[\s(|&;])(?:[\w./$@{}-]*/)?chainbench\b`)

// shellComplaints applies the same rule to a shell script.
//
// Scripts drive this CLI too, and nothing compiles them, so they are the one
// place where a retired name can sit for as long as nobody runs that script.
// Measured 2026-09-18: `chain down` and `--data-dir`, neither of which exists.
//
// A line is joined across backslash continuations first, comment lines are
// skipped, and the words after the program name are read until the command ends
// (a pipe, a semicolon, a redirect). Only a line that carries a long flag is
// judged, for the same reason the Go half needs one.
func shellComplaints(path, rel string, paths map[string]bool, byPath map[string]map[string]bool, groups map[string]map[string]bool) []string {
	b, err := os.ReadFile(path) //nolint:gosec // a file this walk just listed
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.ReplaceAll(string(b), "\\\n", " "), "\n")
	var out []string
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		loc := cliInvocation.FindStringIndex(line)
		if loc == nil {
			continue
		}
		rest := line[loc[1]:]
		if cut := strings.IndexAny(rest, "|;&>"); cut >= 0 {
			rest = rest[:cut]
		}
		var words, flags []string
		leading := true
		for _, w := range strings.Fields(rest) {
			switch {
			case flagLiteral.MatchString(w):
				flags = append(flags, w)
				leading = false
			case strings.HasPrefix(w, "-"):
				leading = false
			case leading && shellWord.MatchString(w):
				words = append(words, w)
			default:
				leading = false
			}
		}
		if len(flags) == 0 {
			continue
		}
		cmd, ok := longestPath(words, paths)
		if !ok {
			continue
		}
		out = append(out, complaints(rel, i+1, cmd, words, flags, byPath, groups)...)
	}
	return out
}

// shellWord matches a bare word — not a variable, not a path, not a quoted
// value — which is the only shape a command name takes.
var shellWord = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
