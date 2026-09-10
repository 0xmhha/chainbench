package surface

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/app"
)

// Deps builds the app dependencies a CLI invocation supplies.
//
// Every command group needs the same three things — where to write side notes,
// how to read the environment, and what the operator typed — and each of the
// twelve groups used to build them itself. Three variants had drifted apart:
// seven set only Logf, four added Env, and one added Command, so which facts a
// use case received depended on which command tree reached it. A workspace lock
// taken through `tx` recorded "(command not recorded)" while the same lock taken
// through `chain` named the invocation.
//
// The union is the right answer for all of them and costs nothing to give: Env
// is os.Getenv, which is exactly what app falls back to when it is nil, and
// Command is provenance the lock should always have.
func Deps(cmd *cobra.Command) app.Deps {
	errOut := cmd.ErrOrStderr()
	return app.Deps{
		Env:     os.Getenv,
		Command: commandLine(cmd),
		Logf: func(format string, args ...any) {
			fmt.Fprintf(errOut, format+"\n", args...)
		},
	}
}

// commandLine renders this invocation the way the operator typed it: the command
// path, so a workspace lock names the run that holds it rather than the binary.
func commandLine(cmd *cobra.Command) string {
	parts := []string{"chainbench"}
	for c := cmd; c != nil && c.Name() != "chainbench"; c = c.Parent() {
		parts = append(parts[:1], append([]string{c.Name()}, parts[1:]...)...)
	}
	return strings.Join(parts, " ")
}
