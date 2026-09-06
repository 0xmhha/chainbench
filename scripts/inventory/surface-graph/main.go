// Command surface-graph prints what each surface (CLI, MCP, DSL) registers and
// which of those registrations reach past the app layer, so the U track's
// progress is read off the code rather than estimated.
//
// The walk itself lives in internal/arch, next to the ratchet test that holds
// the number down, so the tool and the test cannot disagree about what they are
// counting.
//
// Run it as:
//
//	go run ./scripts/inventory/surface-graph .
//
// The baseline it produced on 2026-09-05, which the U track counted down from:
// 157 registrations (CLI 58, MCP 54, DSL 45 = 18 actions and 27 assertions), of
// which 109 reached past app. As of U6 the two surfaces are at zero; the DSL's
// 45 are the language's vocabulary at L3, which is below app rather than past
// it.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/0xmhha/chainbench/internal/arch"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	entries := arch.Entries(root)

	fmt.Printf("%-5s %-30s %-9s %s\n", "SURF", "FEATURE", "VIA", "PACKAGES")
	counts := map[string]int{}
	past := map[string]int{}
	for _, e := range entries {
		fmt.Printf("%-5s %-30s %-9s %s\n", e.Surface, e.Name, e.Via(), strings.Join(e.Pkgs, " "))
		counts[e.Surface]++
		if e.ReachesPastApp() {
			past[e.Surface]++
		}
	}
	total := 0
	for _, n := range past {
		total += n
	}
	fmt.Printf("\nregistrations: %d", len(entries))
	for _, s := range []string{"CLI", "MCP", "DSL", "DSLa"} {
		fmt.Printf("  %s %d (past app %d)", s, counts[s], past[s])
	}
	// The DSL's actions and assertions are the language's implementation at L3,
	// below app, so they are inventoried rather than counted as debt. See
	// internal/arch/surface_test.go for why.
	surfaces := total - past["DSL"] - past["DSLa"]
	fmt.Printf("\nsurface registrations reaching past app: %d (DSL vocabulary at L3: %d)\n",
		surfaces, past["DSL"]+past["DSLa"])
}
