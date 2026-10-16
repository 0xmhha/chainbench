package main

import (
	"go/ast"
	"go/token"
	"path"
	"strconv"
	"strings"
)

type embeddedProgram struct {
	Language string `json:"language"`
	Source   string `json:"source"`
	Line     int    `json:"line"`
}

type inlinePrograms struct {
	execAliases map[string]bool
	writes      map[*ast.Ident]int
}

func newInlinePrograms(f *ast.File) inlinePrograms {
	p := inlinePrograms{map[string]bool{}, map[*ast.Ident]int{}}
	for _, spec := range f.Imports {
		if value, _ := strconv.Unquote(spec.Path.Value); value == "os/exec" {
			alias := "exec"
			if spec.Name != nil {
				alias = spec.Name.Name
			}
			p.execAliases[alias] = true
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.ValueSpec:
			for _, name := range v.Names {
				if name.Obj != nil && name.Obj.Kind == ast.Var {
					p.writes[definition(name)]++
				}
			}
		case *ast.AssignStmt:
			for _, target := range v.Lhs {
				if name, ok := target.(*ast.Ident); ok && name.Obj != nil {
					p.writes[definition(name)]++
				}
			}
		}
		return true
	})
	return p
}

// definition identifies a source declaration, not a type-resolved object.
// Only bare string bindings and argument lists are supported by this extractor.
func definition(id *ast.Ident) *ast.Ident {
	if id.Obj == nil {
		return nil
	}
	switch declaration := id.Obj.Decl.(type) {
	case *ast.ValueSpec:
		for _, name := range declaration.Names {
			if name.Name == id.Name {
				return name
			}
		}
	case *ast.AssignStmt:
		for _, target := range declaration.Lhs {
			if name, ok := target.(*ast.Ident); ok && name.Name == id.Name {
				return name
			}
		}
	}
	return nil
}

// initializer resolves lexical bindings with one definition. Reassigned values,
// function arguments and arbitrary computations remain explicit dynamic inputs.
func (p inlinePrograms) initializer(id *ast.Ident) ast.Expr {
	obj := id.Obj
	if obj == nil || p.writes[definition(id)] > 1 || (obj.Kind == ast.Con && p.writes[definition(id)] > 0) {
		return nil
	}
	switch d := obj.Decl.(type) {
	case *ast.ValueSpec:
		for i, name := range d.Names {
			if name.Name == id.Name && i < len(d.Values) {
				return d.Values[i]
			}
		}
	case *ast.AssignStmt:
		for i, name := range d.Lhs {
			if n, ok := name.(*ast.Ident); ok && n.Name == id.Name && i < len(d.Rhs) {
				return d.Rhs[i]
			}
		}
	}
	return nil
}

func (p inlinePrograms) text(expr ast.Expr, seen map[*ast.Ident]bool) (string, bool) {
	switch v := expr.(type) {
	case *ast.BasicLit:
		if v.Kind == token.STRING {
			value, err := strconv.Unquote(v.Value)
			return value, err == nil
		}
	case *ast.ParenExpr:
		return p.text(v.X, seen)
	case *ast.BinaryExpr:
		if v.Op == token.ADD {
			left, a := p.text(v.X, seen)
			right, b := p.text(v.Y, seen)
			return left + right, a && b
		}
	case *ast.Ident:
		if v.Obj != nil && !seen[definition(v)] {
			seen[definition(v)] = true
			defer delete(seen, definition(v))
			return p.text(p.initializer(v), seen)
		}
	}
	return "", false
}

func (p inlinePrograms) arguments(expr ast.Expr) ([]ast.Expr, bool) {
	if id, ok := expr.(*ast.Ident); ok {
		expr = p.initializer(id)
	}
	if list, ok := expr.(*ast.CompositeLit); ok {
		return list.Elts, true
	}
	return nil, false
}

func (p inlinePrograms) command(call *ast.CallExpr) (language, source string, known bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || (sel.Sel.Name != "Command" && sel.Sel.Name != "CommandContext") {
		return "", "", false
	}
	owner, ok := sel.X.(*ast.Ident)
	if !ok || !p.execAliases[owner.Name] || (owner.Obj != nil && owner.Obj.Kind != ast.Pkg) {
		return "", "", false
	}
	offset := 0
	if sel.Sel.Name == "CommandContext" {
		offset = 1
	}
	if len(call.Args) <= offset {
		return "", "", false
	}
	args := append([]ast.Expr(nil), call.Args[offset:]...)
	program, ok := p.text(args[0], map[*ast.Ident]bool{})
	if !ok {
		return "", "", false
	}
	switch path.Base(program) {
	case "python", "python3":
		language = "python"
	case "bash", "sh":
		language = "bash"
	case "node", "nodejs":
		language = "javascript"
	default:
		return "", "", false
	}
	if call.Ellipsis.IsValid() {
		last, ok := p.arguments(args[len(args)-1])
		if !ok {
			return language, "", false
		}
		args = append(args[:len(args)-1], last...)
	}
	for i := 1; i < len(args); i++ {
		flag, ok := p.text(args[i], map[*ast.Ident]bool{})
		if !ok {
			return language, "", false
		}
		if flag == "--" || !strings.HasPrefix(flag, "-") {
			return "", "", false
		}
		eval := (language == "javascript" && (flag == "-e" || flag == "--eval")) || (language == "python" && flag == "-c") || (language == "bash" && !strings.HasPrefix(flag, "--") && strings.Contains(flag, "c"))
		if eval {
			if i+1 < len(args) {
				source, known = p.text(args[i+1], map[*ast.Ident]bool{})
			}
			return language, source, known
		}
	}
	return "", "", false // a script file is inventoried as its own source
}
