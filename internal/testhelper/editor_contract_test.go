package testhelper

import (
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestEditorContractCoversRegistrationsAndConsumedFields(t *testing.T) {
	entries, schemas, err := EditorVocabulary()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("registry has no editor entries")
	}
	fields := map[string]bool{"expected": true, "method": true}
	seen := map[string]bool{}
	for _, entry := range entries {
		key := entry.Kind + "-" + entry.Name
		if seen[key] {
			t.Fatalf("duplicate %s", key)
		}
		seen[key] = true
		schema, ok := schemas[key]
		if !ok {
			t.Fatalf("missing %s", key)
		}
		for name := range schema["properties"].(map[string]any) {
			fields[name] = true
		}
	}
	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		if len(path) > 8 && path[len(path)-8:] == "_test.go" {
			continue
		}
		tree, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(tree, func(n ast.Node) bool {
			index, ok := n.(*ast.IndexExpr)
			if !ok {
				return true
			}
			literal, ok := index.Index.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			name := ""
			switch x := index.X.(type) {
			case *ast.Ident:
				name = x.Name
			case *ast.SelectorExpr:
				name = x.Sel.Name
			}
			if name != "Args" && name != "Spec" && name != "spec" && name != "args" && name != "in" && name != "opts" {
				return true
			}
			field, _ := strconv.Unquote(literal.Value)
			if !fields[field] {
				t.Errorf("%s: consumed argument %q not in editor contract", path, field)
			}
			return true
		})
	}
}

// This checks each implementation separately: a field in rpcCall's contract
// cannot accidentally hide a missing partition argument.
func TestEditorContractDirectBuiltinArguments(t *testing.T) {
	roots := map[string]string{}
	for name, action := range map[string]any{
		actionSendTx: sendTxAction{}, actionWaitBlock: waitBlockAction{}, actionWaitFor: waitForAction{}, interp.ActionRead: readAction{}, actionNewAccount: newAccountAction{},
		actionSendRawTampered: sendRawTamperedAction{}, actionSendSetCode: sendSetCodeAction{}, actionSignAuth: signAuthorizationAction{}, actionLoad: loadAction{},
		actionStopNode: stopNodeAction{}, actionStartNode: startNodeAction{}, actionRestartNode: restartNodeAction{}, actionResetNode: resetNodeAction{}, actionSwapNode: swapNodeAction{},
		actionPartition: partitionAction{}, actionHealPartition: healPartitionAction{}, actionReadNodeLog: readNodeLogAction{}, dsl.ActionCrossFork: crossForkAction{},
		actionFaucet: faucetAction{}, actionDeployContract: deployContractAction{}, actionRegisterContract: registerContractAction{}, actionWSOpen: wsOpenAction{},
	} {
		roots[reflect.TypeOf(action).Name()+".Do"] = "action-" + name
	}
	for _, builtin := range builtinAssertions() {
		name := runtime.FuncForPC(reflect.ValueOf(builtin.read).Pointer()).Name()
		parts := strings.Split(name, ".")
		roots[parts[len(parts)-1]] = "reader-" + builtin.name
	}
	roots["readTxMined"] = "reader-" + assertTxMined
	_, schemas, err := EditorVocabulary()
	if err != nil {
		t.Fatal(err)
	}
	paths, _ := filepath.Glob("*.go")
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		tree, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range tree.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok {
				continue
			}
			name := function.Name.Name
			if function.Recv != nil {
				receiver := function.Recv.List[0].Type
				if pointer, ok := receiver.(*ast.StarExpr); ok {
					receiver = pointer.X
				}
				if id, ok := receiver.(*ast.Ident); ok {
					name = id.Name + "." + name
				}
			}
			key, ok := roots[name]
			if !ok {
				continue
			}
			properties := schemas[key]["properties"].(map[string]any)
			ast.Inspect(function.Body, func(n ast.Node) bool {
				index, ok := n.(*ast.IndexExpr)
				if !ok {
					return true
				}
				literal, ok := index.Index.(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					return true
				}
				var parameter bool
				switch x := index.X.(type) {
				case *ast.Ident:
					parameter = x.Name == "spec" || x.Name == "args"
				case *ast.SelectorExpr:
					parameter = x.Sel.Name == "Args" || x.Sel.Name == "Spec"
				}
				if parameter {
					field, _ := strconv.Unquote(literal.Value)
					if _, exists := properties[field]; !exists {
						t.Errorf("%s %s consumes %s but %s does not expose it", path, name, field, key)
					}
				}
				return true
			})
		}
	}
}
