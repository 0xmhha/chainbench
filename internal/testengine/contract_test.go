package testengine

import (
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
)

var updateContract = flag.Bool("update-contract", false, "rewrite the contract table in design-v3 state-machine-06")

const designDoc = "../../docs/dev/architecture/design-v3/state-machine-06-naming-and-contract.md"

// TestRunContractsAreConsistent: every message a run state says it sends is
// accepted or ignored by that state or an ancestor.
func TestRunContractsAreConsistent(t *testing.T) {
	if err := newRunner(chainsetup.Deps{}, RunSuiteIn{}).m.Audit(); err != nil {
		t.Error(err)
	}
}

// TestRunContractTableMatchesTheDesign: the TEST table in the design document
// is the declarations, rendered. Run with -update-contract to rewrite it.
func TestRunContractTableMatchesTheDesign(t *testing.T) {
	table := newRunner(chainsetup.Deps{}, RunSuiteIn{}).m.ContractTable(whatName)
	raw, err := os.ReadFile(designDoc)
	if err != nil {
		t.Fatal(err)
	}
	doc := string(raw)
	begin, end := "<!-- contract:begin test -->\n", "<!-- contract:end test -->"
	i, j := strings.Index(doc, begin), strings.Index(doc, end)
	if i < 0 || j < i {
		t.Fatalf("%s has no test contract section", designDoc)
	}
	if doc[i+len(begin):j] == table {
		return
	}
	if *updateContract {
		if err := os.WriteFile(designDoc, []byte(doc[:i+len(begin)]+table+doc[j:]), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	t.Errorf("the test contract table in %s is not the declarations; rerun with -update-contract.\nwant:\n%s", designDoc, table)
}
