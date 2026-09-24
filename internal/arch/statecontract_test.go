package arch

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestStateContractsMatchTheCode holds each state's declared contract to the
// code under it (design-v3 state-machine-06 §5).
//
// The machine checks the declarations at run time, but only on the paths a run
// takes. This reads every state's Process and the helpers it calls for the
// message types it switches on, and every Enter/Exit/Process and the helpers
// they call for the messages they send, and compares both with Contract().
// A drop like the one that stalled a rebuild — a message handled only by a
// sibling of its sender — is caught by Audit from the declarations; this is
// what keeps the declarations honest.
//
// It is syntactic: a message is the type name of a composite literal handed to
// SendSelf, a handled message is a type in a type switch or assertion. Commands
// (What constants named Cmd…) come from outside the machine and are not held
// to Emits.
func TestStateContractsMatchTheCode(t *testing.T) {
	for _, pkg := range []string{"internal/chainsetup", "internal/testengine"} {
		t.Run(filepath.Base(pkg), func(t *testing.T) {
			problems := auditContracts(t, filepath.Join("../..", pkg))
			for _, p := range problems {
				t.Error(p)
			}
		})
	}
}

type pkgFacts struct {
	whatOf   map[string]string          // message type -> What constant
	methods  map[string]map[string]bool // type -> method names
	ifaces   map[string][]string        // interface -> its method names
	switched map[string][]string        // "Type.Method" -> types in its switches/assertions
	sends    map[string][]string        // func key -> composite-literal types passed to SendSelf
	calls    map[string][]string        // func key -> names it calls
	declared map[string]declaredContract
}

type declaredContract struct {
	accepts, emits []string
	dynamic        bool // Emits computed at run time (e.g. l.report.What())
}

func auditContracts(t *testing.T, dir string) []string {
	t.Helper()
	fset := token.NewFileSet()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var files []*ast.File
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, file)
	}
	f := &pkgFacts{
		whatOf: map[string]string{}, methods: map[string]map[string]bool{},
		ifaces: map[string][]string{}, switched: map[string][]string{},
		sends: map[string][]string{}, calls: map[string][]string{},
		declared: map[string]declaredContract{},
	}
	for _, file := range files {
		f.scan(file)
	}
	if len(f.declared) == 0 {
		t.Fatalf("%s declares no contracts — the scan is broken", dir)
	}
	// What constant -> message type, to compare in one vocabulary.
	typeOf := map[string]string{}
	for typ, w := range f.whatOf {
		typeOf[w] = typ
	}
	var problems []string
	states := make([]string, 0, len(f.declared))
	for s := range f.declared {
		states = append(states, s)
	}
	sort.Strings(states)
	for _, st := range states {
		d := f.declared[st]
		handled := f.handledBy(st)
		sent := f.sentBy(st)
		acc := set(mapTypes(d.accepts, typeOf))
		em := set(mapTypes(d.emits, typeOf))
		for _, h := range sorted(handled) {
			if !acc[h] {
				problems = append(problems, st+" handles "+h+" but its contract does not accept it")
			}
		}
		for a := range acc {
			if !handled[a] {
				problems = append(problems, st+" declares it accepts "+a+" but its Process never handles it")
			}
		}
		if d.dynamic {
			continue
		}
		for _, s := range sorted(sent) {
			if strings.HasPrefix(f.whatOf[s], "Cmd") {
				continue
			}
			if !em[s] {
				problems = append(problems, st+" sends "+s+" but its contract does not emit it")
			}
		}
		for e := range em {
			if !sent[e] {
				problems = append(problems, st+" declares it emits "+e+" but never sends it")
			}
		}
	}
	sort.Strings(problems)
	return problems
}

func (f *pkgFacts) scan(file *ast.File) {
	for _, d := range file.Decls {
		switch d := d.(type) {
		case *ast.GenDecl:
			if d.Tok != token.TYPE {
				continue
			}
			for _, sp := range d.Specs {
				ts := sp.(*ast.TypeSpec)
				if it, ok := ts.Type.(*ast.InterfaceType); ok {
					for _, m := range it.Methods.List {
						for _, n := range m.Names {
							f.ifaces[ts.Name.Name] = append(f.ifaces[ts.Name.Name], n.Name)
						}
					}
				}
			}
		case *ast.FuncDecl:
			f.scanFunc(d)
		}
	}
}

func (f *pkgFacts) scanFunc(d *ast.FuncDecl) {
	recv := receiverType(d)
	key := d.Name.Name
	if recv != "" {
		key = recv + "." + d.Name.Name
		if f.methods[recv] == nil {
			f.methods[recv] = map[string]bool{}
		}
		f.methods[recv][d.Name.Name] = true
	}
	if d.Body == nil {
		return
	}
	if recv != "" && d.Name.Name == "What" && len(d.Body.List) == 1 {
		if r, ok := d.Body.List[0].(*ast.ReturnStmt); ok && len(r.Results) == 1 {
			if id, ok := r.Results[0].(*ast.Ident); ok {
				f.whatOf[recv] = id.Name
			}
		}
	}
	if recv != "" && d.Name.Name == "Contract" {
		f.declared[recv] = readContract(d)
		return
	}
	ast.Inspect(d.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			name := calleeName(n.Fun)
			if name == "SendSelf" {
				for _, a := range n.Args {
					if t := litType(a); t != "" {
						f.sends[key] = append(f.sends[key], t)
					}
				}
			} else if name != "" {
				f.calls[key] = append(f.calls[key], name)
			}
		case *ast.TypeSwitchStmt:
			for _, c := range n.Body.List {
				for _, e := range c.(*ast.CaseClause).List {
					if t := typeName(e); t != "" {
						f.switched[key] = append(f.switched[key], t)
					}
				}
			}
		case *ast.TypeAssertExpr:
			if n.Type != nil {
				if t := typeName(n.Type); t != "" {
					f.switched[key] = append(f.switched[key], t)
				}
			}
		}
		return true
	})
}

// handledBy is the message types st's Process handles, directly or through a
// method of the same type it calls, with interfaces expanded.
func (f *pkgFacts) handledBy(st string) map[string]bool {
	out := map[string]bool{}
	add := func(key string) {
		for _, t := range f.switched[key] {
			if need, ok := f.ifaces[t]; ok {
				for typ, have := range f.methods {
					all := len(need) > 0
					for _, n := range need {
						if !have[n] {
							all = false
						}
					}
					if all && f.whatOf[typ] != "" {
						out[typ] = true
					}
				}
				continue
			}
			if f.whatOf[t] != "" {
				out[t] = true
			}
		}
	}
	add(st + ".Process")
	for _, c := range f.calls[st+".Process"] {
		if f.methods[st][c] {
			add(st + "." + c)
		}
	}
	return out
}

// sentBy is the message types st's Enter, Exit and Process send, directly or
// through the helpers they call. Helpers are followed by name within the
// package, one level of methods of the same type and of the package's own
// helpers (fail, operated), which is where every state sends from.
func (f *pkgFacts) sentBy(st string) map[string]bool {
	out := map[string]bool{}
	seen := map[string]bool{}
	var walk func(key string, depth int)
	walk = func(key string, depth int) {
		if seen[key] || depth > 3 {
			return
		}
		seen[key] = true
		for _, t := range f.sends[key] {
			out[t] = true
		}
		for _, c := range f.calls[key] {
			for k := range f.sends {
				if k == c || strings.HasSuffix(k, "."+c) && helperOwner(k, st) {
					walk(k, depth+1)
				}
			}
			for k := range f.calls {
				if k == c || strings.HasSuffix(k, "."+c) && helperOwner(k, st) {
					walk(k, depth+1)
				}
			}
		}
	}
	for _, m := range []string{"Enter", "Exit", "Process"} {
		walk(st+"."+m, 0)
	}
	return out
}

// helperOwner reports whether key is a method of the state itself or of the
// machine's owner (Manager, runner), which is where shared send helpers live.
func helperOwner(key, st string) bool {
	owner := key[:strings.Index(key, ".")]
	return owner == st || owner == "Manager" || owner == "runner"
}

func readContract(d *ast.FuncDecl) declaredContract {
	var c declaredContract
	ast.Inspect(d.Body, func(n ast.Node) bool {
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}
		k, _ := kv.Key.(*ast.Ident)
		if k == nil {
			return true
		}
		var names []string
		if cl, ok := kv.Value.(*ast.CompositeLit); ok {
			for _, e := range cl.Elts {
				if id, ok := e.(*ast.Ident); ok {
					names = append(names, id.Name)
				} else {
					c.dynamic = c.dynamic || k.Name == "Emits"
				}
			}
		}
		switch k.Name {
		case "Accepts":
			c.accepts = names
		case "Emits":
			c.emits = names
		}
		return true
	})
	return c
}

func receiverType(d *ast.FuncDecl) string {
	if d.Recv == nil || len(d.Recv.List) == 0 {
		return ""
	}
	return typeName(d.Recv.List[0].Type)
}

func calleeName(e ast.Expr) string {
	switch f := e.(type) {
	case *ast.SelectorExpr:
		return f.Sel.Name
	case *ast.Ident:
		return f.Name
	}
	return ""
}

func litType(e ast.Expr) string {
	if u, ok := e.(*ast.UnaryExpr); ok {
		e = u.X
	}
	if cl, ok := e.(*ast.CompositeLit); ok {
		return typeName(cl.Type)
	}
	return ""
}

func typeName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return typeName(t.X)
	case *ast.SelectorExpr:
		return t.Sel.Name
	}
	return ""
}

func mapTypes(whats []string, typeOf map[string]string) []string {
	out := make([]string, 0, len(whats))
	for _, w := range whats {
		if t, ok := typeOf[w]; ok {
			out = append(out, t)
		} else {
			out = append(out, w)
		}
	}
	return out
}

func set(xs []string) map[string]bool {
	m := map[string]bool{}
	for _, x := range xs {
		m[x] = true
	}
	return m
}

func sorted(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
