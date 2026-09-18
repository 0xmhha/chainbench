package testhelper

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

// addressShapedKeys are the argument names that hold an account or a contract.
// A spec may write a name in any of them, so whatever reads one has to resolve
// it before use.
//
// "expected" and "params" joined the list when the comparison side was opened
// to labels. They are not address-ONLY — both carry numbers, hex and block tags
// too, and an element that does not resolve is left alone — but that is a rule
// about what resolving does, not about who has to call it. Both were being read
// straight off the spec by four assertions, one of which resolved the spec
// first and then read the unresolved original anyway.
var addressShapedKeys = map[string]bool{
	"to": true, "address": true, "from": true, "deployer": true, "funder": true,
	"expected": true, "params": true,
}

// resolvers are the functions that turn a written name into an address. funder
// counts: it resolves "from" through ResolveAccount on the caller's behalf.
var resolvers = map[string]bool{
	"ResolveAccount": true, "ResolveAddress": true,
	"resolveAddressArgs": true, "resolveNames": true, "funder": true,
}

// alwaysResolved are the resolvers themselves: they read the keys in order to
// resolve them.
var alwaysResolved = map[string]bool{
	"resolveAddressArgs": true, "resolveNames": true,
	"ResolveAccount": true, "ResolveAddress": true,
}

// TestEveryAddressArgumentIsResolved.
//
// A spec may write "govMinter" wherever it may write an address, and the code
// that reads the argument decides whether that works. Step 3 changed the path
// readers take and left the actions calling ResolveAccount directly, so
// "to": "govMinter" resolved in an assertion and died in sendTx with "unknown
// account". No offline check saw it: the address record exercises one path and
// the actions use another.
//
// This reads the source instead. A function that takes an address-shaped
// argument out of a spec must also call a resolver, or be listed as one whose
// arguments arrive resolved.
func TestEveryAddressArgumentIsResolved(t *testing.T) {
	fset := token.NewFileSet()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]*ast.File{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, perr := parser.ParseFile(fset, filepath.Join(".", name), nil, 0)
		if perr != nil {
			t.Fatalf("parse %s: %v", name, perr)
		}
		files[name] = f
	}
	// The exemption is derived, not listed: every function wired in as a reader
	// has its spec resolved by read.go before dispatch. Deriving it means a new
	// reader is covered without editing this test, and a new ACTION is not.
	readers := readerFuncs(files)

	var offenders []string
	for name, f := range files {
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || alwaysResolved[fn.Name.Name] || readers[fn.Name.Name] {
				continue
			}
			missing := scanBody(fn.Body)
			if len(missing) == 0 {
				continue
			}
			offenders = append(offenders, name+":"+funcLabel(fn)+" reads "+strings.Join(missing, ", ")+" without resolving it")
		}
	}
	for _, o := range offenders {
		t.Errorf("%s", o)
	}
}

// funcLabel names a method by its receiver, so the failure says which action
// rather than "Do".
func funcLabel(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fn.Name.Name
	}
	t := fn.Recv.List[0].Type
	if star, ok := t.(*ast.StarExpr); ok {
		t = star.X
	}
	if id, ok := t.(*ast.Ident); ok {
		return id.Name + "." + fn.Name.Name
	}
	return fn.Name.Name
}

// readerFuncs finds the functions wired in as readers: the "read:" field of the
// assertion table and whatever RegisterReader is handed.
func readerFuncs(files map[string]*ast.File) map[string]bool {
	out := map[string]bool{}
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			switch v := n.(type) {
			case *ast.KeyValueExpr:
				if k, ok := v.Key.(*ast.Ident); ok && k.Name == "read" {
					if id, ok := v.Value.(*ast.Ident); ok {
						out[id.Name] = true
					}
				}
			case *ast.CallExpr:
				sel, ok := v.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "RegisterReader" {
					return true
				}
				for _, a := range v.Args {
					ast.Inspect(a, func(m ast.Node) bool {
						if id, ok := m.(*ast.Ident); ok && strings.HasPrefix(id.Name, "read") {
							out[id.Name] = true
						}
						return true
					})
				}
			}
			return true
		})
	}
	return out
}

// scanBody reports the address-shaped keys a function takes out of a map
// WITHOUT handing the value to a resolver.
//
// It is per key, not per function, because per function is not enough: sendTx
// resolves "from" and forgot "to", and a check that asked only "does this
// function resolve anything" called that clean. It follows the variable the
// read is assigned to and asks whether that variable reaches a resolver.
func scanBody(body *ast.BlockStmt) []string {
	fromKey := map[string]string{} // variable -> key it was read from
	resolved := map[string]bool{}  // variable handed to a resolver

	note := func(lhs []ast.Expr, rhs []ast.Expr) {
		if len(rhs) != 1 {
			return
		}
		key := addressKeyOf(rhs[0])
		if key == "" || len(lhs) == 0 {
			return
		}
		if id, ok := lhs[0].(*ast.Ident); ok && id.Name != "_" {
			fromKey[id.Name] = key
		}
	}
	ast.Inspect(body, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.AssignStmt:
			note(v.Lhs, v.Rhs)
		case *ast.CallExpr:
			id, ok := v.Fun.(*ast.Ident)
			if !ok || !resolvers[id.Name] {
				return true
			}
			for _, a := range v.Args {
				if ai, ok := a.(*ast.Ident); ok {
					resolved[ai.Name] = true
				}
			}
			// resolveAddressArgs and funder take the whole map, so everything
			// read out of it afterwards is already resolved.
			if id.Name == "resolveAddressArgs" || id.Name == "funder" {
				resolved["*"] = true
			}
		}
		return true
	})

	if resolved["*"] {
		return nil
	}
	var missing []string
	seen := map[string]bool{}
	for v, key := range fromKey {
		if resolved[v] || seen[key] {
			continue
		}
		seen[key] = true
		missing = append(missing, key)
	}
	sort.Strings(missing)
	return missing
}

// addressKeyOf returns the address-shaped key an expression reads, or "".
// It sees both m["to"] and m["to"].(string).
func addressKeyOf(e ast.Expr) string {
	if ta, ok := e.(*ast.TypeAssertExpr); ok {
		e = ta.X
	}
	ix, ok := e.(*ast.IndexExpr)
	if !ok {
		return ""
	}
	lit, ok := ix.Index.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}
	if k := strings.Trim(lit.Value, `"`); addressShapedKeys[k] {
		return k
	}
	return ""
}
