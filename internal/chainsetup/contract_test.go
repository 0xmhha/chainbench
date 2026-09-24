package chainsetup

import (
	"flag"
	"os"
	"strings"
	"testing"
)

var updateContract = flag.Bool("update-contract", false, "rewrite the contract table in design-v3 state-machine-06")

// designDoc is the document that keeps both machines' contract tables.
const designDoc = "../../docs/dev/architecture/design-v3/state-machine-06-naming-and-contract.md"

// TestContractsAreConsistent: every message a state says it sends is accepted
// or ignored by that state or an ancestor — the check whose absence let
// stoppedToRebuild and nodesRestarted be dropped.
func TestContractsAreConsistent(t *testing.T) {
	if err := newTestManager(t).m.Audit(); err != nil {
		t.Error(err)
	}
}

// TestContractTableMatchesTheDesign: the table in the design document is the
// declarations, rendered. Run with -update-contract to rewrite it.
func TestContractTableMatchesTheDesign(t *testing.T) {
	checkContractTable(t, designDoc, "chain", newTestManager(t).m.ContractTable(WhatName), *updateContract)
}

// checkContractTable compares, or with update rewrites, the table between the
// named markers.
func checkContractTable(t *testing.T, path, section, table string, update bool) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc := string(raw)
	begin, end := "<!-- contract:begin "+section+" -->\n", "<!-- contract:end "+section+" -->"
	i, j := strings.Index(doc, begin), strings.Index(doc, end)
	if i < 0 || j < i {
		t.Fatalf("%s has no %q … %q section", path, begin, end)
	}
	have := doc[i+len(begin) : j]
	if have == table {
		return
	}
	if update {
		if err := os.WriteFile(path, []byte(doc[:i+len(begin)]+table+doc[j:]), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	t.Errorf("the %s contract table in %s is not the declarations; rerun with -update-contract.\nwant:\n%s", section, path, table)
}
