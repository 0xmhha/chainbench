// Command flaggraph extracts a command/flag graph from a go-ethereum-derived
// chain repository by parsing its cmd/ tree with go/ast (no type checking, no
// build required).
//
// It emits JSON: {binary, flags[], groups{}, commands[]}.
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Flag is one declared cli flag variable.
type Flag struct {
	Var      string `json:"var"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Category string `json:"category,omitempty"`
	Usage    string `json:"usage,omitempty"`
	Default  string `json:"default,omitempty"`
	File     string `json:"file"`
}

// Command is one declared cli.Command variable.
type Command struct {
	Var         string   `json:"var"`
	Name        string   `json:"name"`
	Usage       string   `json:"usage,omitempty"`
	Flags       []string `json:"flags,omitempty"`
	Subcommands []string `json:"subcommands,omitempty"`
	File        string   `json:"file"`
}

// Graph is the extracted per-repository result.
type Graph struct {
	Repo     string              `json:"repo"`
	Binary   string              `json:"binary"`
	Flags    []Flag              `json:"flags"`
	Groups   map[string][]string `json:"groups"`
	Commands []Command           `json:"commands"`
	AppCmds  []string            `json:"appCommands"`
	AppFlags []string            `json:"appFlagGroups"`
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: flaggraph <repo-root> <binary-dir-name>")
		os.Exit(2)
	}
	root, bin := os.Args[1], os.Args[2]
	g := Graph{Repo: filepath.Base(root), Binary: bin, Groups: map[string][]string{}}

	// cmd/utils holds the shared flag declarations, cmd/<bin> the binary's
	// command tree and flag groups, and internal/debug the logging/pprof group
	// that app.Flags merges in but that lives outside cmd/.
	dirs := []string{
		filepath.Join(root, "cmd", "utils"),
		filepath.Join(root, "cmd", bin),
		filepath.Join(root, "internal", "debug"),
	}
	for _, d := range dirs {
		if err := scanDir(d, &g); err != nil {
			fmt.Fprintf(os.Stderr, "scan %s: %v\n", d, err)
		}
	}
	sort.Slice(g.Flags, func(i, j int) bool { return g.Flags[i].Name < g.Flags[j].Name })
	sort.Slice(g.Commands, func(i, j int) bool { return g.Commands[i].Name < g.Commands[j].Name })

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(g); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func scanDir(dir string, g *Graph) error {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return err
	}
	for _, pkg := range pkgs {
		for path, f := range pkg.Files {
			base := filepath.Base(path)
			ast.Inspect(f, func(n ast.Node) bool {
				switch v := n.(type) {
				case *ast.ValueSpec:
					collectValueSpec(v, base, g)
				case *ast.AssignStmt:
					collectAssign(v, base, g)
				}
				return true
			})
		}
	}
	return nil
}

// collectValueSpec handles `var XFlag = &cli.StringFlag{...}` and
// `var nodeFlags = []cli.Flag{...}` and `var initCommand = &cli.Command{...}`.
func collectValueSpec(vs *ast.ValueSpec, file string, g *Graph) {
	for i, name := range vs.Names {
		if i >= len(vs.Values) {
			return
		}
		classify(name.Name, vs.Values[i], file, g)
	}
}

// collectAssign handles `nodeFlags = []cli.Flag{...}` inside init().
func collectAssign(as *ast.AssignStmt, file string, g *Graph) {
	for i, lhs := range as.Lhs {
		if i >= len(as.Rhs) {
			continue
		}
		// Both `nodeFlags = ...` (Ident) and `app.Commands = ...` (SelectorExpr)
		// are group/command declarations we want.
		name := exprString(lhs)
		if name == "" {
			continue
		}
		classify(name, as.Rhs[i], file, g)
	}
}

func classify(varName string, val ast.Expr, file string, g *Graph) {
	switch e := unparen(val).(type) {
	case *ast.UnaryExpr: // &cli.XFlag{...} / &cli.Command{...}
		cl, ok := e.X.(*ast.CompositeLit)
		if !ok {
			return
		}
		classifyComposite(varName, cl, file, g)
	case *ast.CompositeLit:
		classifyComposite(varName, e, file, g)
	case *ast.CallExpr: // flags.Merge(nodeFlags, rpcFlags, ...)
		if sel, ok := e.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Merge" {
			var members []string
			for _, a := range e.Args {
				members = append(members, idents(a)...)
			}
			if len(members) > 0 {
				g.Groups[varName] = members
			}
		}
	}
}

func classifyComposite(varName string, cl *ast.CompositeLit, file string, g *Graph) {
	typeName := exprString(cl.Type)
	switch {
	case strings.HasPrefix(typeName, "[]cli.Flag"):
		g.Groups[varName] = idents(cl)
	case strings.HasPrefix(typeName, "[]*cli.Command"):
		if varName == "app.Commands" || varName == "Commands" {
			g.AppCmds = idents(cl)
		} else {
			g.Groups[varName] = idents(cl)
		}
	case strings.HasSuffix(typeName, "Flag") && strings.Contains(typeName, "."):
		f := Flag{Var: varName, Type: typeName, File: file}
		for _, el := range cl.Elts {
			kv, ok := el.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key := exprString(kv.Key)
			switch key {
			case "Name":
				f.Name = litString(kv.Value)
			case "Usage":
				f.Usage = litString(kv.Value)
			case "Category":
				f.Category = exprString(kv.Value)
			case "Value":
				f.Default = exprString(kv.Value)
			}
		}
		if f.Name != "" {
			g.Flags = append(g.Flags, f)
		}
	case typeName == "cli.Command":
		c := Command{Var: varName, File: file}
		for _, el := range cl.Elts {
			kv, ok := el.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			switch exprString(kv.Key) {
			case "Name":
				c.Name = litString(kv.Value)
			case "Usage":
				c.Usage = litString(kv.Value)
			case "Flags":
				c.Flags = idents(kv.Value)
			case "Subcommands":
				c.Subcommands = idents(kv.Value)
			}
		}
		if c.Name != "" {
			g.Commands = append(g.Commands, c)
		}
	case strings.HasPrefix(typeName, "[]cli.Flag"):
		g.Groups[varName] = idents(cl)
	case strings.HasPrefix(typeName, "[]*cli.Command"):
		if varName == "app.Commands" || varName == "Commands" {
			g.AppCmds = idents(cl)
		} else {
			g.Groups[varName] = idents(cl)
		}
	}
}

// idents flattens the identifiers referenced inside a composite literal or
// nested call (flags.Merge(...)), which is how flag groups reference members.
func idents(e ast.Expr) []string {
	var out []string
	ast.Inspect(e, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.Ident:
			out = append(out, v.Name)
		case *ast.SelectorExpr:
			out = append(out, exprString(v))
			return false
		}
		return true
	})
	return out
}

func unparen(e ast.Expr) ast.Expr {
	if p, ok := e.(*ast.ParenExpr); ok {
		return unparen(p.X)
	}
	return e
}

func litString(e ast.Expr) string {
	if bl, ok := e.(*ast.BasicLit); ok && bl.Kind == token.STRING {
		return strings.Trim(bl.Value, "`\"")
	}
	return exprString(e)
}

func exprString(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return exprString(v.X) + "." + v.Sel.Name
	case *ast.BasicLit:
		return strings.Trim(v.Value, "`\"")
	case *ast.CallExpr:
		return exprString(v.Fun) + "(...)"
	case *ast.ArrayType:
		return "[]" + exprString(v.Elt)
	case *ast.StarExpr:
		return "*" + exprString(v.X)
	case *ast.UnaryExpr:
		return exprString(v.X)
	case *ast.CompositeLit:
		return exprString(v.Type) + "{...}"
	case *ast.BinaryExpr:
		return exprString(v.X) + exprString(v.Y)
	}
	return ""
}
