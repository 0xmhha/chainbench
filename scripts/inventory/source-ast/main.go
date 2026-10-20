// Command source-ast parses Go sources, including tests and all build tags.
package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
)

type input struct {
	Path   string `json:"path"`
	Source string `json:"source"`
}
type node struct {
	Kind  string `json:"kind"`
	Name  string `json:"name"`
	Line  int    `json:"line"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}
type result struct {
	Path     string            `json:"path"`
	Count    int               `json:"astNodes"`
	Nodes    []node            `json:"nodes"`
	Error    string            `json:"error,omitempty"`
	Embedded []embeddedProgram `json:"embedded,omitempty"`
}

func callable(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.StarExpr:
		return callable(v.X)
	case *ast.SelectorExpr:
		return callable(v.X) + "." + v.Sel.Name
	case *ast.IndexExpr:
		return callable(v.X)
	case *ast.IndexListExpr:
		return callable(v.X)
	case *ast.ParenExpr:
		return callable(v.X)
	default:
		return "anonymous"
	}
}

func parse(in input) result {
	r := result{Path: in.Path, Nodes: []node{}}
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, in.Path, in.Source, 0)
	if err != nil {
		r.Error = err.Error()
		return r
	}
	add := func(kind, name string, n ast.Node) {
		r.Nodes = append(r.Nodes, node{kind, name, fs.Position(n.Pos()).Line, fs.Position(n.Pos()).Offset, fs.Position(n.End()).Offset})
	}
	programs := newInlinePrograms(f)
	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			return true
		}
		r.Count++
		switch v := n.(type) {
		case *ast.FuncDecl:
			kind := "function"
			name := v.Name.Name
			if v.Recv != nil {
				kind = "method"
				name = callable(v.Recv.List[0].Type) + "." + name
			}
			add(kind, name, v)
		case *ast.TypeSpec:
			add("type", v.Name.Name, v)
		case *ast.ValueSpec:
			for _, name := range v.Names {
				add("binding", name.Name, v)
			}
		case *ast.CallExpr:
			add("call", callable(v.Fun), v)
			if language, body, known := programs.command(v); language != "" {
				if known {
					r.Embedded = append(r.Embedded, embeddedProgram{language, body, fs.Position(v.Pos()).Line})
				} else {
					add("dynamic-program-input", language+" interpreter payload", v)
				}
			}
		case *ast.ImportSpec:
			name, e := strconv.Unquote(v.Path.Value)
			if e == nil {
				add("import", name, v)
			}
		case *ast.BasicLit:
			// HTML templates stored in raw Go literals contain executable script/style.
			if v.Kind == token.STRING {
				text, e := strconv.Unquote(v.Value)
				if e == nil && ((strings.Contains(text, "<script") && strings.Contains(text, "</script>")) || (strings.Contains(text, "<style") && strings.Contains(text, "</style>"))) {
					add("embedded-html", text, v)
				}
			}
		}
		return true
	})
	return r
}

func main() {
	var inputs []input
	if err := json.NewDecoder(os.Stdin).Decode(&inputs); err != nil {
		panic(err)
	}
	results := make([]result, 0, len(inputs))
	for _, in := range inputs {
		results = append(results, parse(in))
	}
	if err := json.NewEncoder(os.Stdout).Encode(results); err != nil {
		panic(err)
	}
}
