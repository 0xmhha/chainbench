package mcp_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/feature"
)

func TestMCPOperationsConsumeTypedRegistry(t *testing.T) {
	operations := map[string]bool{}
	files, err := filepath.Glob("../app/*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range file.Decls {
			fn, ok := declaration.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !fn.Name.IsExported() || fn.Type.Params == nil || len(fn.Type.Params.List) != 3 {
				continue
			}
			deps, ok := fn.Type.Params.List[1].Type.(*ast.Ident)
			if ok && deps.Name == "Deps" {
				operations[fn.Name.Name] = true
			}
		}
	}
	if len(operations) < 30 {
		t.Fatal("operation AST scan found too few inputs")
	}
	files, err = filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		set := token.NewFileSet()
		file, err := parser.ParseFile(set, path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			alias, ok := selector.X.(*ast.Ident)
			if ok && alias.Name == "app" && operations[selector.Sel.Name] {
				t.Errorf("%s still bypasses typed registry operation %s", set.Position(call.Pos()), selector.Sel.Name)
			}
			return true
		})
	}
}

func TestMCPCommonFieldsUseRegistrySchemas(t *testing.T) {
	expected := map[string]map[string]any{}
	for property, source := range map[string][2]string{"rpc": {"node.status", "rpc"}, "workspaceDir": {"chain.stop", "workspace-dir"}, "chain": {"chain.describe", "chain"}} {
		entry, ok := feature.Lookup(source[0])
		if !ok {
			t.Fatal("missing schema source " + source[0])
		}
		schema, err := feature.Schema(entry.Input(), entry.ReadOnly)
		if err != nil {
			t.Fatal(err)
		}
		field, ok := schema["properties"].(map[string]any)[source[1]].(map[string]any)
		if !ok {
			t.Fatalf("schema source %s lacks %s", source[0], source[1])
		}
		expected[property] = field
	}
	response := call(t, newServer(), map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list"})
	checked := 0
	for _, value := range response["result"].(map[string]any)["tools"].([]any) {
		tool := value.(map[string]any)
		properties, _ := tool["inputSchema"].(map[string]any)["properties"].(map[string]any)
		for name, wanted := range expected {
			property, ok := properties[name].(map[string]any)
			if !ok || property["type"] != "string" {
				continue
			}
			checked++
			if !reflect.DeepEqual(property["type"], wanted["type"]) {
				t.Errorf("%s.%s type differs from the registry", tool["name"], name)
			}
			description, _ := property["description"].(string)
			if !strings.HasPrefix(description, wanted["description"].(string)) {
				t.Errorf("%s.%s description does not derive from the registry: %q", tool["name"], name, description)
			}
		}
	}
	if checked < 30 {
		t.Fatalf("only %d common fields were compared", checked)
	}
}
