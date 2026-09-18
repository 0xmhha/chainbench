package arch

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// CommentClaim is one statement a comment makes that this package can decide:
// a path it cites, a package it names, or a symbol it opens with.
type CommentClaim struct {
	// Kind names which rule the claim broke.
	Kind string
	// Pos is "<path>:<line>" of the comment.
	Pos string
	// Detail is the offending text, and what the code says instead.
	Detail string
}

// Rule identifiers for [CommentClaim.Kind].
const (
	// KindDeadFileCite is a "<file>.go:<line>" citation whose file is missing
	// or whose line is past the end of it.
	KindDeadFileCite = "dead-file-cite"
	// KindDeadFileName is a bare "<file>.go" no file in the tree matches.
	KindDeadFileName = "dead-file-name"
	// KindDeadPkgPath is an internal/, cmd/ or pkg/ path that does not exist.
	KindDeadPkgPath = "dead-pkg-path"
	// KindDeadDocPath is a docs/*.md path that does not exist.
	KindDeadDocPath = "dead-doc-path"
	// KindWrongPackageName is a package comment naming a different package.
	KindWrongPackageName = "wrong-package-name"
	// KindNamesOtherSymbol is a doc comment that opens with an identifier
	// other than the one it documents: the symbol was renamed and its comment
	// was not.
	KindNamesOtherSymbol = "doc-names-other-symbol"
)

// skipDirs are directories whose contents are not this module's own source:
// vendored code, research snapshots of other revisions, and the git store.
var skipDirs = map[string]bool{
	".git": true, "vendor": true, "research": true,
	"codemine": true, "archive": true, "node_modules": true, "testdata": true,
}

var (
	reFileLine = regexp.MustCompile(`\b((?:[\w./-]+/)?[\w.-]+\.go):(\d+)\b`)
	reFileOnly = regexp.MustCompile(`\b([\w-]+\.go)\b`)
	rePkgPath  = regexp.MustCompile(`\b(internal/[\w./-]+|cmd/[\w./-]+|pkg/[\w./-]+)\b`)
	reDocPath  = regexp.MustCompile(`\b(docs/[\w./-]+\.md)\b`)
	reDocLink  = regexp.MustCompile(`\[([A-Z][\w]*(?:\.[A-Z][\w]*)?)\]`)
	// reCompound matches an identifier no reader mistakes for an English word:
	// it carries a second capital or an underscore (TestKeySet_Foo, SwapNode).
	reCompound = regexp.MustCompile(`^[A-Z][a-z0-9]*([A-Z_][A-Za-z0-9_]*)+$`)
	// externalRepo marks a citation belonging to another repository. A comment
	// that names the repo is not claiming the file is ours.
	externalRepo = regexp.MustCompile(`(?i)(go-wemix|go-stablenet|go-wbft|go-ethereum|geth|upstream|analysed repo|analyzed repo)`)
	// historical marks a citation the comment presents as past. Such a comment
	// does not claim the path exists now.
	historical = regexp.MustCompile(`(?i)(lived in|used to|previously|retired|ported from|absorbed|replaced|was in|formerly)`)
)

// nearBefore is how much text before a citation is searched for a marker.
const nearBefore = 120

// CommentClaims reports every comment claim under root that the code
// contradicts. It parses source only; it never builds.
//
// It decides four kinds of claim and no more: a cited path, a cited line, the
// package a package comment names, and the identifier a doc comment opens
// with. A comment's prose is not checked, because a parser cannot decide it —
// that audit is a person's job.
func CommentClaims(root string) ([]CommentClaim, error) {
	fset := token.NewFileSet()
	files := map[string]int{}
	base := map[string][]string{}
	decls := map[string]bool{}
	dirs := map[string]bool{}

	type parsedFile struct {
		rel string
		ast *ast.File
	}
	var parsed []parsedFile

	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel := filepath.ToSlash(relTo(root, p))
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			dirs[rel] = true
			return nil
		}
		if !strings.HasSuffix(p, ".go") {
			return nil
		}
		files[rel] = countLines(p)
		base[filepath.Base(p)] = append(base[filepath.Base(p)], rel)
		f, perr := parser.ParseFile(fset, p, nil, parser.ParseComments)
		if perr != nil {
			return nil // a file that does not parse is the compiler's report, not ours
		}
		parsed = append(parsed, parsedFile{rel, f})
		collectDecls(f, decls)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("arch: walk %s: %w", root, err)
	}

	var out []CommentClaim
	for _, p := range parsed {
		out = append(out, citationClaims(fset, p.rel, p.ast, root, files, base, dirs, decls)...)
		out = append(out, namingClaims(fset, p.rel, p.ast, decls)...)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Pos < out[j].Pos
	})
	return out, nil
}

// collectDecls records every identifier the file declares at top level.
func collectDecls(f *ast.File, into map[string]bool) {
	for _, d := range f.Decls {
		switch x := d.(type) {
		case *ast.FuncDecl:
			into[x.Name.Name] = true
		case *ast.GenDecl:
			for _, s := range x.Specs {
				switch y := s.(type) {
				case *ast.TypeSpec:
					into[y.Name.Name] = true
				case *ast.ValueSpec:
					for _, n := range y.Names {
						into[n.Name] = true
					}
				}
			}
		}
	}
}

// citationClaims checks the paths and doc links a file's comments cite.
func citationClaims(fset *token.FileSet, rel string, f *ast.File, root string,
	files map[string]int, base map[string][]string, dirs map[string]bool, decls map[string]bool) []CommentClaim {
	var out []CommentClaim
	for _, cg := range f.Comments {
		text := cg.Text()
		pos := fmt.Sprintf("%s:%d", rel, fset.Position(cg.Pos()).Line)

		for _, m := range reFileLine.FindAllStringSubmatch(text, -1) {
			if unclaimed(text, m[0]) {
				continue
			}
			line, _ := strconv.Atoi(m[2])
			target := resolveFile(m[1], files, base)
			switch {
			case target == "":
				out = append(out, CommentClaim{KindDeadFileCite, pos, m[0]})
			case line > files[target]:
				out = append(out, CommentClaim{KindDeadFileCite, pos,
					fmt.Sprintf("%s (file has %d lines)", m[0], files[target])})
			}
		}
		for _, m := range reFileOnly.FindAllStringSubmatch(text, -1) {
			if m[1] == "_test.go" || strings.Contains(text, m[1]+":") || unclaimed(text, m[1]) {
				continue
			}
			if _, ok := base[m[1]]; !ok {
				out = append(out, CommentClaim{KindDeadFileName, pos, m[1]})
			}
		}
		for _, m := range rePkgPath.FindAllStringSubmatch(text, -1) {
			c := strings.TrimSuffix(m[1], "/")
			if unclaimed(text, m[1]) || dirs[c] || pathExists(root, c) {
				continue
			}
			// A path may name a symbol ("internal/app.ListSpecs"); the package
			// part is what has to exist.
			if i := strings.LastIndex(c, "."); i > 0 {
				if pkg := c[:i]; dirs[pkg] || pathExists(root, pkg) {
					continue
				}
			}
			out = append(out, CommentClaim{KindDeadPkgPath, pos, c})
		}
		for _, m := range reDocPath.FindAllStringSubmatch(text, -1) {
			if unclaimed(text, m[1]) || pathExists(root, m[1]) {
				continue
			}
			out = append(out, CommentClaim{KindDeadDocPath, pos, m[1]})
		}
		for _, m := range reDocLink.FindAllStringSubmatch(text, -1) {
			name := m[1]
			if i := strings.Index(name, "."); i >= 0 {
				name = name[i+1:]
			}
			if !decls[name] {
				out = append(out, CommentClaim{"dead-doc-link", pos, m[0]})
			}
		}
	}
	return out
}

// namingClaims checks that a package comment names its own package and that a
// doc comment does not open with somebody else's identifier.
func namingClaims(fset *token.FileSet, rel string, f *ast.File, decls map[string]bool) []CommentClaim {
	var out []CommentClaim
	if f.Doc != nil {
		first := strings.TrimSpace(f.Doc.Text())
		if strings.HasPrefix(first, "Package ") {
			if said := strings.Fields(first)[1]; said != f.Name.Name {
				out = append(out, CommentClaim{KindWrongPackageName,
					fmt.Sprintf("%s:%d", rel, fset.Position(f.Doc.Pos()).Line),
					fmt.Sprintf("says %q, is %q", said, f.Name.Name)})
			}
		}
	}
	for _, d := range f.Decls {
		name, doc := documented(d)
		if name == "" || doc == nil || !ast.IsExported(name) {
			continue
		}
		first := strings.TrimSpace(doc.Text())
		opener := first
		for _, article := range []string{"A ", "An ", "The "} {
			opener = strings.TrimPrefix(opener, article)
		}
		if strings.HasPrefix(first, name) || strings.HasPrefix(opener, name) {
			continue
		}
		words := strings.FieldsFunc(opener, func(r rune) bool {
			return r == ' ' || r == ':' || r == ',' || r == '(' || r == '.' || r == '\n'
		})
		if len(words) == 0 || words[0] == name || !namesCode(words[0], decls) {
			continue // prose describing the case, which is a style choice
		}
		head := first
		if i := strings.IndexAny(head, ".\n"); i > 0 {
			head = head[:i]
		}
		out = append(out, CommentClaim{KindNamesOtherSymbol,
			fmt.Sprintf("%s:%d", rel, fset.Position(doc.Pos()).Line),
			fmt.Sprintf("%s is documented as %q", name, head)})
	}
	return out
}

// documented returns the name and doc comment of a declaration that carries
// exactly one, and empty otherwise.
func documented(d ast.Decl) (string, *ast.CommentGroup) {
	switch x := d.(type) {
	case *ast.FuncDecl:
		return x.Name.Name, x.Doc
	case *ast.GenDecl:
		if len(x.Specs) == 1 {
			if ts, ok := x.Specs[0].(*ast.TypeSpec); ok {
				return ts.Name.Name, x.Doc
			}
		}
	}
	return "", nil
}

// namesCode reports whether a doc comment's first word names code rather than
// opening a sentence. A compound identifier is unmistakable even after the
// symbol was renamed away; a single capitalised word counts only when
// something still declares it, since "Argument" opens a sentence too.
func namesCode(word string, decls map[string]bool) bool {
	if reCompound.MatchString(word) {
		return true
	}
	if word == "" || word[0] < 'A' || word[0] > 'Z' {
		return false
	}
	return decls[word]
}

// unclaimed reports whether the text just before cite disowns it: it names
// another repository, or presents the path as past.
func unclaimed(text, cite string) bool {
	i := strings.Index(text, cite)
	if i < 0 {
		return false
	}
	start := i - nearBefore
	if start < 0 {
		start = 0
	}
	window := text[start:i]
	return externalRepo.MatchString(window) || historical.MatchString(window)
}

// resolveFile finds the tree path a citation refers to, or empty.
func resolveFile(cited string, files map[string]int, base map[string][]string) string {
	if _, ok := files[cited]; ok {
		return cited
	}
	for rel := range files {
		if strings.HasSuffix(rel, "/"+cited) {
			return rel
		}
	}
	if list, ok := base[filepath.Base(cited)]; ok && len(list) == 1 {
		return list[0]
	}
	return ""
}

func relTo(root, p string) string { r, _ := filepath.Rel(root, p); return r }

func pathExists(root, rel string) bool {
	_, err := os.Stat(filepath.Join(root, rel))
	return err == nil
}

// countLines returns the number of lines in a file, or 0 when it cannot be read.
func countLines(p string) int {
	b, err := os.ReadFile(p)
	if err != nil {
		return 0
	}
	return strings.Count(string(b), "\n") + 1
}
