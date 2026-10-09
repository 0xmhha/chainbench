package testhelper

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// The argument contract is declared by hand in builtinArguments. This file
// derives the same thing from the implementations, so a builtin that starts or
// stops reading an argument breaks the build of the contract instead of
// leaving the editor and the engine offering an argument with no effect.
//
// The derivation follows the argument map through the package: from a
// builtin's Do/Check (or reader function) into every package function that is
// handed the map, a copy of it, or the action/assertion context. A key counts
// as read when it indexes such a map as a string constant, or when a constant
// reaches a helper parameter that indexes it (durationArg(args, "timeout")).
// Functions that return a map[string]any are transforms: what they read only
// decides what they copy, so their reads are not counted, but their result is
// the argument map again.

type extractFunc struct {
	decl    *ast.FuncDecl
	params  []string
	types   []string
	returns bool // copies its argument map (a transform)
	targets bool // turns a context's selected nodes into a slice
}

type extractPackage struct {
	funcs     map[string]*extractFunc
	constants map[string]string
	lists     map[string][]string
}

type extractResult struct {
	keys      map[string]bool
	keyParams map[int]bool
	usesOn    bool
	firstOnly bool // reads only the first of the selected nodes
	output    bool
	problems  []string
}

func loadExtractPackage(t *testing.T) *extractPackage {
	t.Helper()
	p := &extractPackage{funcs: map[string]*extractFunc{}, constants: map[string]string{}, lists: map[string][]string{}}
	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				p.funcs[funcKey(d)] = newExtractFunc(d)
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					value, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for i, name := range value.Names {
						if i >= len(value.Values) {
							continue
						}
						if s, ok := stringLiteral(value.Values[i]); ok {
							p.constants[name.Name] = s
						}
						if list, ok := value.Values[i].(*ast.CompositeLit); ok {
							p.lists[name.Name] = p.literalList(list)
						}
					}
				}
			}
		}
	}
	return p
}

func funcKey(d *ast.FuncDecl) string {
	if d.Recv == nil {
		return d.Name.Name
	}
	receiver := d.Recv.List[0].Type
	if star, ok := receiver.(*ast.StarExpr); ok {
		receiver = star.X
	}
	if id, ok := receiver.(*ast.Ident); ok {
		return id.Name + "." + d.Name.Name
	}
	return d.Name.Name
}

func newExtractFunc(d *ast.FuncDecl) *extractFunc {
	f := &extractFunc{decl: d}
	for _, field := range d.Type.Params.List {
		typ := exprString(field.Type)
		if len(field.Names) == 0 {
			f.params, f.types = append(f.params, "_"), append(f.types, typ)
		}
		for _, name := range field.Names {
			f.params, f.types = append(f.params, name.Name), append(f.types, typ)
		}
	}
	returnsMap, returnsSlice := false, false
	if d.Type.Results != nil {
		for _, field := range d.Type.Results.List {
			typ := exprString(field.Type)
			returnsMap = returnsMap || typ == "map[string]any"
			returnsSlice = returnsSlice || strings.HasPrefix(typ, "[]")
		}
	}
	if returnsSlice && d.Body != nil {
		ast.Inspect(d.Body, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok && sel.Sel.Name == "On" {
				if id, ok := sel.X.(*ast.Ident); ok {
					for i, name := range f.params {
						if name == id.Name && isContextType(f.types[i]) {
							f.targets = true
						}
					}
				}
			}
			return true
		})
	}
	// A transform copies its argument map: it ranges over a map parameter or
	// returns one. A function that merely returns some other map (a receipt, a
	// filter) is not one, and its reads count.
	mapParams := map[string]bool{}
	for i, name := range f.params {
		if f.types[i] == "map[string]any" {
			mapParams[name] = true
		}
	}
	if returnsMap && d.Body != nil {
		ast.Inspect(d.Body, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.RangeStmt:
				if id, ok := x.X.(*ast.Ident); ok && mapParams[id.Name] {
					f.returns = true
				}
			case *ast.ReturnStmt:
				for _, result := range x.Results {
					if id, ok := result.(*ast.Ident); ok && mapParams[id.Name] {
						f.returns = true
					}
				}
			}
			return true
		})
	}
	return f
}

func exprString(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.StarExpr:
		return "*" + exprString(x.X)
	case *ast.SelectorExpr:
		return exprString(x.X) + "." + x.Sel.Name
	case *ast.MapType:
		return "map[" + exprString(x.Key) + "]" + exprString(x.Value)
	case *ast.InterfaceType:
		return "any"
	case *ast.ArrayType:
		return "[]" + exprString(x.Elt)
	}
	return "?"
}

func stringLiteral(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(lit.Value)
	return s, err == nil
}

func (p *extractPackage) literalList(list *ast.CompositeLit) []string {
	var out []string
	for _, element := range list.Elts {
		if s, ok := p.stringValue(element); ok {
			out = append(out, s)
		}
	}
	return out
}

func (p *extractPackage) stringValue(e ast.Expr) (string, bool) {
	if s, ok := stringLiteral(e); ok {
		return s, true
	}
	if id, ok := e.(*ast.Ident); ok {
		s, ok := p.constants[id.Name]
		return s, ok
	}
	return "", false
}

func isContextType(typ string) bool {
	return typ == "*interp.ActionCtx" || typ == "*interp.AssertCtx"
}

// extractCall is one static call made with the argument map, a context, or a
// string constant in some position.
type extractCall struct {
	callee    string
	tainted   map[int]bool
	nodes     map[int]bool // positions carrying the selected nodes
	constants map[int]string
	keyParams map[int]int // callee position -> caller key parameter
}

// walk reads one function given which of its parameters carry the argument
// map, and reports what it reads and which calls carry the map onward.
func (p *extractPackage) walk(f *extractFunc, tainted, nodeParams map[int]bool) (extractResult, []extractCall) {
	r := extractResult{keys: map[string]bool{}, keyParams: map[int]bool{}}
	maps, contexts, keyParams := map[string]bool{}, map[string]bool{}, map[string]int{}
	keyVars, nodes := map[string][]string{}, map[string]bool{}
	for i, name := range f.params {
		switch {
		case nodeParams[i]:
			nodes[name] = true
		case tainted[i]:
			maps[name] = true
		case isContextType(f.types[i]):
			contexts[name] = true
		case f.types[i] == "string":
			keyParams[name] = i
		}
	}
	var isMap func(ast.Expr) bool
	isMap = func(e ast.Expr) bool {
		switch x := e.(type) {
		case *ast.ParenExpr:
			return isMap(x.X)
		case *ast.Ident:
			return maps[x.Name]
		case *ast.SelectorExpr:
			id, ok := x.X.(*ast.Ident)
			return ok && contexts[id.Name] && (x.Sel.Name == "Args" || x.Sel.Name == "Spec")
		case *ast.CallExpr:
			id, ok := x.Fun.(*ast.Ident)
			if !ok {
				return false
			}
			callee, ok := p.funcs[id.Name]
			if !ok || !callee.returns {
				return false
			}
			for _, arg := range x.Args {
				if isMap(arg) {
					return true
				}
			}
		}
		return false
	}
	// isNodes is a slice of the statement's selected nodes: ac.On, an alias,
	// or what a targets helper builds from it.
	isNodes := func(e ast.Expr) bool {
		switch x := e.(type) {
		case *ast.Ident:
			return nodes[x.Name]
		case *ast.SelectorExpr:
			id, ok := x.X.(*ast.Ident)
			return ok && contexts[id.Name] && x.Sel.Name == "On"
		case *ast.CallExpr:
			id, ok := x.Fun.(*ast.Ident)
			if !ok || p.funcs[id.Name] == nil || !p.funcs[id.Name].targets {
				return false
			}
			for _, arg := range x.Args {
				if a, ok := arg.(*ast.Ident); ok && contexts[a.Name] {
					return true
				}
			}
		}
		return false
	}
	// Aliases and copies propagate in source order; two passes settle the
	// assignments that precede a closure using them.
	for pass := 0; pass < 2; pass++ {
		ast.Inspect(f.decl.Body, func(n ast.Node) bool {
			switch s := n.(type) {
			case *ast.AssignStmt:
				for i, rhs := range s.Rhs {
					// v, err := transform(args) taints its first result.
					target := i
					if len(s.Rhs) == 1 {
						target = 0
					} else if len(s.Lhs) != len(s.Rhs) {
						continue
					}
					if id, ok := s.Lhs[target].(*ast.Ident); ok {
						if isMap(rhs) {
							maps[id.Name] = true
						}
						if isNodes(rhs) {
							nodes[id.Name] = true
						}
					}
				}
			case *ast.RangeStmt:
				value, ok := s.Value.(*ast.Ident)
				if !ok {
					return true
				}
				switch over := s.X.(type) {
				case *ast.CompositeLit:
					keyVars[value.Name] = p.literalList(over)
				case *ast.Ident:
					if list, ok := p.lists[over.Name]; ok {
						keyVars[value.Name] = list
					}
				}
			}
			return true
		})
	}
	var calls []extractCall
	// Indexing the first node is a report of the common case when the same
	// function also visits every node, and a truncation when it does not.
	first, visitsAll := false, false
	ast.Inspect(f.decl.Body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.IndexExpr:
			if lit, ok := x.Index.(*ast.BasicLit); ok && lit.Value == "0" && isNodes(x.X) {
				first = true
			}
			if !isMap(x.X) {
				return true
			}
			if s, ok := p.stringValue(x.Index); ok {
				r.keys[s] = true
				return true
			}
			if id, ok := x.Index.(*ast.Ident); ok {
				if i, ok := keyParams[id.Name]; ok {
					r.keyParams[i] = true
					return true
				}
				if list, ok := keyVars[id.Name]; ok {
					for _, key := range list {
						r.keys[key] = true
					}
					return true
				}
			}
			r.problems = append(r.problems, "unresolved index "+exprString(x.Index))
		case *ast.RangeStmt:
			if isNodes(x.X) {
				visitsAll = true
			}
			if isMap(x.X) && !f.returns {
				r.problems = append(r.problems, "ranges over every argument")
			}
		case *ast.SelectorExpr:
			if id, ok := x.X.(*ast.Ident); ok && contexts[id.Name] && x.Sel.Name == "On" {
				r.usesOn = true
			}
		case *ast.AssignStmt:
			for _, lhs := range x.Lhs {
				sel, ok := lhs.(*ast.SelectorExpr)
				if !ok {
					continue
				}
				if id, ok := sel.X.(*ast.Ident); ok && contexts[id.Name] && (sel.Sel.Name == "Value" || sel.Sel.Name == "Hash") {
					r.output = true
				}
			}
		case *ast.CallExpr:
			id, ok := x.Fun.(*ast.Ident)
			if !ok {
				return true
			}
			callee, ok := p.funcs[id.Name]
			if !ok {
				return true
			}
			call := extractCall{callee: id.Name, tainted: map[int]bool{}, nodes: map[int]bool{}, constants: map[int]string{}, keyParams: map[int]int{}}
			carries := false
			for i, arg := range x.Args {
				if i >= len(callee.params) {
					break
				}
				if isMap(arg) {
					call.tainted[i], carries = true, true
				}
				if isNodes(arg) {
					call.nodes[i], carries = true, true
				}
				if a, ok := arg.(*ast.Ident); ok && contexts[a.Name] {
					carries = true
				}
				if s, ok := p.stringValue(arg); ok {
					call.constants[i] = s
				}
				if a, ok := arg.(*ast.Ident); ok {
					if j, ok := keyParams[a.Name]; ok {
						call.keyParams[i] = j
					}
				}
			}
			if carries {
				calls = append(calls, call)
			}
		}
		return true
	})
	r.firstOnly = first && !visitsAll
	return r, calls
}

// consumed returns what a root reads through every function it hands the map to.
func (p *extractPackage) consumed(t *testing.T, root string, tainted map[int]bool) extractResult {
	t.Helper()
	if _, ok := p.funcs[root]; !ok {
		t.Fatalf("no function %s", root)
	}
	taints := map[string]map[int]bool{root: tainted}
	nodeTaints := map[string]map[int]bool{}
	results := map[string]extractResult{}
	for changed := true; changed; {
		changed = false
		names := make([]string, 0, len(taints))
		for name := range taints {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			f := p.funcs[name]
			r, calls := p.walk(f, taints[name], nodeTaints[name])
			for _, call := range calls {
				known, ok := taints[call.callee]
				if !ok {
					known = map[int]bool{}
					taints[call.callee], changed = known, true
				}
				for i := range call.tainted {
					if !known[i] {
						known[i], changed = true, true
					}
				}
				if nodeTaints[call.callee] == nil {
					nodeTaints[call.callee] = map[int]bool{}
				}
				for i := range call.nodes {
					if !nodeTaints[call.callee][i] {
						nodeTaints[call.callee][i], changed = true, true
					}
				}
				callee := results[call.callee]
				for i := range callee.keyParams {
					if s, ok := call.constants[i]; ok {
						r.keys[s] = true
					}
					if j, ok := call.keyParams[i]; ok {
						r.keyParams[j] = true
					}
				}
				for key := range callee.keys {
					r.keys[key] = true
				}
				r.usesOn = r.usesOn || callee.usesOn
				r.firstOnly = r.firstOnly || callee.firstOnly
				r.output = r.output || callee.output
				r.problems = append(r.problems, callee.problems...)
			}
			if f.returns {
				// A transform's reads only choose what it copies.
				r.keys = map[string]bool{}
			}
			previous, seen := results[name]
			if !seen || len(previous.keys) != len(r.keys) || len(previous.keyParams) != len(r.keyParams) || previous.usesOn != r.usesOn || previous.firstOnly != r.firstOnly || previous.output != r.output {
				changed = true
			}
			results[name] = r
		}
	}
	return results[root]
}

func readerFunc(read any) string {
	name := runtime.FuncForPC(reflect.ValueOf(read).Pointer()).Name()
	return name[strings.LastIndex(name, ".")+1:]
}

// derivedArguments maps "kind:name" to the runtime argument keys the
// implementation reads, with the interpreter's own consumption folded in:
// "save" binds only what an action reports, and onEach fans out only a
// statement that selects a node.
func derivedArguments(t *testing.T) (map[string][]string, map[string][]string) {
	t.Helper()
	p := loadExtractPackage(t)
	out, problems := map[string][]string{}, map[string][]string{}
	add := func(key string, r extractResult, kind string) {
		set := map[string]bool{}
		for k := range r.keys {
			set[k] = true
		}
		if kind == "action" && r.output {
			set["save"] = true
		}
		if r.usesOn {
			set["on"] = true
			// onEach is read only by an implementation that visits every
			// selected node; one that takes the first would drop the rest.
			set["onEach"] = !r.firstOnly
		}
		if kind == "action" && set["on"] {
			set["onEach"] = true
		}
		list := make([]string, 0, len(set))
		for k, ok := range set {
			if ok {
				list = append(list, k)
			}
		}
		sort.Strings(list)
		out[key] = list
		if len(r.problems) > 0 {
			problems[key] = r.problems
		}
	}
	reg := &captureRegistry{Registry: interp.NewRegistry(), impls: map[string]any{}, readers: map[string]interp.Reader{}}
	Register(reg)
	for _, entry := range reg.entries {
		key := entry.Kind + ":" + entry.Name
		switch entry.Kind {
		case "action", "assertion":
			impl := reg.impls[key]
			typ := reflect.TypeOf(impl)
			method := "Do"
			if entry.Kind == "assertion" {
				method = "Check"
			}
			r := p.consumed(t, typ.Name()+"."+method, map[int]bool{})
			if a, ok := impl.(rpcAssertion); ok {
				merge(&r, p.consumed(t, readerFunc(a.read), map[int]bool{3: true}))
			}
			add(key, r, entry.Kind)
		case "reader":
			add(key, p.consumed(t, readerFunc(reg.readers[entry.Name]), map[int]bool{3: true}), "reader")
		}
	}
	return out, problems
}

// captureRegistry keeps the registered implementations so each can be traced.
type captureRegistry struct {
	interp.Registry
	entries []VocabularyEntry
	impls   map[string]any
	readers map[string]interp.Reader
}

func (r *captureRegistry) RegisterAction(name string, a interp.Action) {
	r.entries = append(r.entries, VocabularyEntry{Kind: "action", Name: name})
	r.impls["action:"+name] = a
}
func (r *captureRegistry) RegisterAssertion(name string, a interp.Assertion) {
	r.entries = append(r.entries, VocabularyEntry{Kind: "assertion", Name: name})
	r.impls["assertion:"+name] = a
}
func (r *captureRegistry) RegisterReader(name string, rd interp.Reader) {
	r.entries = append(r.entries, VocabularyEntry{Kind: "reader", Name: name})
	r.readers[name] = rd
}

func merge(into *extractResult, from extractResult) {
	for k := range from.keys {
		into.keys[k] = true
	}
	into.usesOn = into.usesOn || from.usesOn
	into.firstOnly = into.firstOnly || from.firstOnly
	into.output = into.output || from.output
	into.problems = append(into.problems, from.problems...)
}

func TestBuiltinArgumentsMatchImplementations(t *testing.T) {
	derived, problems := derivedArguments(t)
	for key, issues := range problems {
		t.Errorf("%s: argument reads the derivation cannot follow: %v", key, issues)
	}
	for key, fields := range derived {
		if declared, ok := builtinArguments[key]; !ok {
			t.Errorf("%s is registered but has no declared arguments; it reads %q", key, strings.Join(fields, " "))
		} else if declared != strings.Join(fields, " ") {
			t.Errorf("%s declares %q but its implementation reads %q", key, declared, strings.Join(fields, " "))
		}
	}
	for key := range builtinArguments {
		if _, ok := derived[key]; !ok {
			t.Errorf("%s is declared but not registered", key)
		}
	}
}

// The derivation must notice a builtin that stops visiting every node, or
// the onEach it then drops would still be offered.
func TestDerivedArgumentsSeeFirstNodeOnly(t *testing.T) {
	derived, _ := derivedArguments(t)
	for key, want := range map[string]bool{"assertion:blockStalled": false, "assertion:blockAdvance": true, "assertion:balanceAt": true, "assertion:wsSubscribe": false} {
		got := strings.Contains(" "+strings.Join(derived[key], " ")+" ", " onEach ")
		if got != want {
			t.Errorf("%s onEach = %v, want %v", key, got, want)
		}
	}
}
