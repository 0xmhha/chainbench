// Package wemix implements the chainbench capabilities specific to the wemix
// chain (its poa/etcd bootstrap), separate from the common set. Importing it
// for side effects loads wemix.jsonl and registers its handlers.
package wemix

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/0xmhha/chainbench/pkg/consensus/poa"
	"github.com/0xmhha/chainbench/pkg/core/capability"
)

//go:embed caps.jsonl
var catalog []byte

func init() {
	if err := capability.LoadCatalog(catalog); err != nil {
		panic(err)
	}
	capability.RegisterHandler("v1", "wemix", "bootstrap.plan", bootstrapPlan)
}

func bootstrapPlan(_ context.Context, _ map[string]any) (string, error) {
	var b strings.Builder
	for i, s := range poa.BootstrapPlan() {
		scope := ""
		if s.OnBootNode {
			scope = " (boot node)"
		}
		fmt.Fprintf(&b, "%d. %s%s — %s\n", i+1, s.Name, scope, s.Detail)
	}
	return b.String(), nil
}
