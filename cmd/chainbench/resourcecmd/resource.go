// Package resourcecmd mounts the resource command group: the servers, port
// slots, and capacity a network may be composed from, asked as questions.
//
// The group is queries only. Nothing here composes, launches, or writes; the
// commands read the server set and answer. Composing is `net`'s job, and a
// composed network's own placement is `net show` — the split follows the
// module boundary: this group speaks for the resource module (what is
// available), while `net` speaks for the composition (what was made of it).
//
// File placement follows the keyringcmd rule: one file for the group, one per
// verb, tests mirroring the group.
package resourcecmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/app"
)

// New builds the resource command group.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resource",
		Short: "The servers, port slots, and capacity a network may be composed from",
		Long: "Queries over the resource a network draws on. `pool` reports the addresses\n" +
			"and port slots the server set offers and how many are taken; `plan` computes\n" +
			"the placement a network shape would get, with nothing written anywhere.\n" +
			"Nothing in this group changes anything — composing is `net`'s job.",
	}
	cmd.AddCommand(newPoolCmd(), newPlanCmd())
	return cmd
}

// deps is the Deps every resource verb runs with: operational side notes print
// as they happen.
func deps(cmd *cobra.Command) app.Deps {
	errOut := cmd.ErrOrStderr()
	return app.Deps{Logf: func(format string, args ...any) {
		fmt.Fprintf(errOut, format+"\n", args...)
	}}
}
